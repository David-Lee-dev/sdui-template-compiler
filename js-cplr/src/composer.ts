import { dirname, resolve } from 'node:path';
import type {
  ComponentFile,
  IncludeResolver,
  JsonObject,
  JsonValue,
  TokenResolver,
} from './types.js';
import { Validator } from './validator.js';

const REF_KEY = '.ref';
const VARS_KEY = '.vars';
const DEFAULTS_KEY = '.defaults';
const CONTENT_KEY = '.content';
const TOKEN_KEY = '.token';
const MISSING = Symbol('missing component variable');

const GROUP_NAME = /^[A-Za-z0-9_-]+$/;

// `@{group.path}` — the inline form of `.token`, usable anywhere a string is.
// `@@{` escapes a literal `@{`. Both are consumed at compile time, so the sigil
// is invisible to the client and composes inside runtime `${...}` expressions.
const TOKEN_SIGIL = /@@\{|@\{([^{}]*)\}/g;
const WHOLE_TOKEN_SIGIL = /^(?:@@\{|@\{([^{}]*)\})$/;

interface ExpansionContext {
  activePaths: readonly string[];
  componentStack: readonly string[];
  includeResolver: IncludeResolver;
  includeStack: readonly string[];
  referrerDir: string;
  rootDir: string;
  tokenResolver: TokenResolver;
}

type SubstitutionResult = JsonValue | typeof MISSING;

export class Composer {
  /**
   * Expands a parsed screen body into client-owned JSON.
   *
   * @param rootBody - Parsed screen template node tree
   * @param includeResolver - Injected path resolver and YAML loader
   * @param tokenResolver - Injected design-token group resolver
   * @param rootDir - SDUI root used to normalize resolved include paths
   * @param referrerDir - Directory containing the screen root file
   * @returns Fully expanded JSON without build-time keys
   * @throws When references, component interfaces, or reserved fields are invalid
   */
  static expand(
    rootBody: JsonValue,
    includeResolver: IncludeResolver,
    tokenResolver: TokenResolver,
    rootDir: string,
    referrerDir: string,
  ): JsonValue {
    return Composer.expandNode(rootBody, {
      activePaths: [],
      componentStack: [],
      includeResolver,
      includeStack: [],
      referrerDir,
      rootDir: resolve(rootDir),
      tokenResolver,
    });
  }

  /**
   * Substitutes exact component variable values and drops omitted slots.
   *
   * @param node - Component body node to transform
   * @param vars - Variable names declared by the component
   * @param args - Values supplied by the component reference
   * @returns Substituted JSON, or undefined when the root slot is omitted
   * @throws When the body contains an undeclared variable slot
   */
  static substituteVars(
    node: JsonValue,
    vars: readonly string[],
    args: Readonly<Record<string, JsonValue>>,
  ): JsonValue | undefined {
    const substituted = Composer.substituteNode(node, vars, args);
    return substituted === MISSING ? undefined : substituted;
  }

  /**
   * Substitutes array slots while splicing list-valued arguments in place.
   *
   * @param nodes - Component body array containing potential variable slots
   * @param vars - Variable names declared by the component
   * @param args - Values supplied by the component reference
   * @returns Substituted array with omitted slots removed
   * @throws When the body contains an undeclared variable slot
   */
  static spliceSlots(
    nodes: readonly JsonValue[],
    vars: readonly string[],
    args: Readonly<Record<string, JsonValue>>,
  ): JsonValue[] {
    const result: JsonValue[] = [];

    for (const node of nodes) {
      const slotName = Composer.slotName(node);
      if (slotName !== undefined) {
        Validator.assertHoleDeclared(vars, slotName);
        if (!Object.hasOwn(args, slotName)) continue;

        const value = Composer.clone(args[slotName]);
        if (Array.isArray(value)) result.push(...value);
        else result.push(value);
        continue;
      }

      const substituted = Composer.substituteNode(node, vars, args);
      if (substituted !== MISSING) result.push(substituted);
    }

    return result;
  }

  private static expandNode(
    node: JsonValue,
    context: ExpansionContext,
  ): JsonValue {
    if (Array.isArray(node)) return Composer.expandArray(node, context);
    if (typeof node === 'string') return Composer.expandString(node, context);
    if (!Composer.isObject(node)) return node;

    if (Object.hasOwn(node, 'screen_id')) {
      throw new Error('screen_id is not allowed in template nodes');
    }
    if (Object.hasOwn(node, TOKEN_KEY)) {
      return Composer.expandToken(node, context);
    }
    if (Object.hasOwn(node, REF_KEY)) {
      return Composer.expandReference(node, context);
    }

    const result: JsonObject = {};
    for (const [key, value] of Object.entries(node)) {
      if (key.startsWith('.')) {
        throw new Error(`Unsupported build key ${key}`);
      }
      result[key] = Composer.expandNode(value, context);
    }
    return result;
  }

  private static expandToken(
    node: JsonObject,
    context: ExpansionContext,
  ): JsonValue {
    if (Object.keys(node).length !== 1) {
      throw new Error('.token node must not have sibling keys');
    }

    const tokenPath = node[TOKEN_KEY];
    if (typeof tokenPath !== 'string') {
      throw new Error('.token value must be a non-empty dotted string');
    }

    return Composer.resolveToken(tokenPath, context);
  }

  /**
   * Resolves a dotted token path against its group file.
   *
   * Shared by `{ .token: ... }` and the inline `@{...}` sigil so both forms
   * validate identically and fail with the same diagnostics.
   *
   * @param tokenPath - Dotted path, `<group>.<key...>`
   * @param context - Expansion context carrying the token resolver
   * @returns Deep copy of the token value
   * @throws When the path is malformed, the group is unknown, or a key is missing
   */
  private static resolveToken(
    tokenPath: string,
    context: ExpansionContext,
  ): JsonValue {
    const segments = tokenPath.split('.');
    if (
      segments.length < 2 ||
      segments.some(
        (segment) => segment.length === 0 || segment.trim() !== segment,
      ) ||
      !GROUP_NAME.test(segments[0])
    ) {
      throw new Error('.token value must be a non-empty dotted string');
    }

    const [group, ...keys] = segments;
    let value: JsonValue = context.tokenResolver(group);
    let traversedPath = group;

    for (const key of keys) {
      if (!Composer.isObject(value)) {
        throw new Error(
          `Cannot traverse non-object token path ${traversedPath} while resolving ${tokenPath}`,
        );
      }
      if (!Object.hasOwn(value, key)) {
        throw new Error(
          `Missing token key ${tokenPath} at ${traversedPath}.${key}`,
        );
      }

      value = value[key];
      traversedPath = `${traversedPath}.${key}`;
    }

    // A token value that carries the sigil would be rescanned when it lands in
    // a component argument subtree, making resolution depend on where it was
    // used. Reject it at the source instead.
    if (typeof value === 'string' && value.includes('@{')) {
      throw new Error(`Token value must not contain @{: ${tokenPath}`);
    }

    return Composer.clone(value);
  }

  /**
   * Substitutes `@{group.path}` token sigils inside a string value.
   *
   * A string that is exactly one sigil resolves to the token's own value and
   * keeps its type (`'@{spacing.lg}'` -> `16`). Any other occurrence is
   * interpolated as text, so tokens compose inside runtime expressions.
   *
   * @param node - Raw string value from the template
   * @param context - Expansion context carrying the token resolver
   * @returns The token value, the interpolated string, or the string unchanged
   * @throws When a sigil is unterminated or interpolates a non-scalar token
   */
  private static expandString(
    node: string,
    context: ExpansionContext,
  ): JsonValue {
    if (!node.includes('@')) return node;

    const whole = WHOLE_TOKEN_SIGIL.exec(node);
    if (whole !== null && whole[1] !== undefined) {
      return Composer.resolveToken(whole[1], context);
    }

    Composer.assertNoDanglingSigil(node);
    return node.replace(TOKEN_SIGIL, (_match, path?: string) => {
      if (path === undefined) return '@{';

      const value = Composer.resolveToken(path, context);
      if (Composer.isObject(value) || Array.isArray(value)) {
        throw new Error(
          `Cannot interpolate non-scalar token ${path} into a string`,
        );
      }
      return typeof value === 'string' ? value : JSON.stringify(value);
    });
  }

  /**
   * Rejects an unterminated `@{` left behind by substitution.
   *
   * A typo like `@{color.primary` would otherwise ship to the client as
   * literal text; failing the build is the loud alternative.
   */
  private static assertNoDanglingSigil(original: string): void {
    if (original.replace(TOKEN_SIGIL, '').includes('@{')) {
      throw new Error(`Unterminated token sigil @{ in: ${original}`);
    }
  }

  private static expandArray(
    nodes: readonly JsonValue[],
    context: ExpansionContext,
  ): JsonValue[] {
    const result: JsonValue[] = [];

    for (const node of nodes) {
      const expanded = Composer.expandNode(node, context);
      if (Composer.isReference(node) && Array.isArray(expanded)) {
        result.push(...expanded);
      } else {
        result.push(expanded);
      }
    }

    return result;
  }

  private static expandReference(
    node: JsonObject,
    context: ExpansionContext,
  ): JsonValue {
    if (Object.hasOwn(node, '_type')) {
      throw new Error('.ref node must not declare _type');
    }

    const refPath = node[REF_KEY];
    if (typeof refPath !== 'string') {
      throw new Error('.ref path must be a string');
    }

    const resolvedInclude = context.includeResolver(
      refPath,
      context.referrerDir,
    );
    const filePath = resolve(context.rootDir, resolvedInclude.filePath);
    Validator.assertNoCircularPath(filePath, context.activePaths);

    // Component args are authored by the caller, so expand them in the caller's
    // context — before this file is pushed onto activePaths. Otherwise reusing a
    // component inside another instance's arg subtree (e.g. a /card in a /card's
    // child) re-enters the same file while it is still "active" and is misflagged
    // as circular. Genuine self-recursion is still caught: a component whose own
    // .content re-references itself expands that .content with the file on the path.
    const args = Object.fromEntries(
      Object.entries(node)
        .filter(([key]) => key !== REF_KEY)
        .map(([key, value]) => [key, Composer.expandNode(value, context)]),
    );
    const targetContext = {
      ...context,
      activePaths: [...context.activePaths, filePath],
      referrerDir: dirname(filePath),
    };

    if (Composer.isComponent(resolvedInclude.content)) {
      return Composer.expandComponent(
        resolvedInclude.content,
        args,
        filePath,
        targetContext,
      );
    }

    const argNames = Object.keys(args);
    if (argNames.length > 0) {
      throw new Error(
        `Fragment ${filePath} does not accept args: ${argNames.join(', ')}`,
      );
    }

    return Composer.expandNode(resolvedInclude.content, {
      ...targetContext,
      includeStack: [...context.includeStack, filePath],
    });
  }

  private static expandComponent(
    target: JsonObject,
    args: Readonly<Record<string, JsonValue>>,
    filePath: string,
    context: ExpansionContext,
  ): JsonValue {
    const component = Composer.toComponentFile(target, filePath);
    Validator.assertArgsDeclared(component.vars, args);
    const effectiveArgs = { ...(component.defaults ?? {}), ...args };

    const substituted = Composer.substituteVars(
      component.content,
      component.vars,
      effectiveArgs,
    );
    if (substituted === undefined) {
      throw new Error(
        `Component body resolved to an omitted slot: ${filePath}`,
      );
    }

    return Composer.expandNode(substituted, {
      ...context,
      componentStack: [...context.componentStack, filePath],
    });
  }

  private static substituteNode(
    node: JsonValue,
    vars: readonly string[],
    args: Readonly<Record<string, JsonValue>>,
  ): SubstitutionResult {
    if (Array.isArray(node)) return Composer.spliceSlots(node, vars, args);

    if (Composer.isObject(node)) {
      const result: JsonObject = {};
      for (const [key, value] of Object.entries(node)) {
        if (key.startsWith('.') && key !== REF_KEY && key !== TOKEN_KEY) {
          throw new Error(`Unsupported build key ${key}`);
        }
        const substituted = Composer.substituteNode(value, vars, args);
        if (substituted !== MISSING) result[key] = substituted;
      }
      return result;
    }

    const slotName = Composer.slotName(node);
    if (slotName === undefined) return node;

    Validator.assertHoleDeclared(vars, slotName);
    if (!Object.hasOwn(args, slotName)) return MISSING;
    return Composer.clone(args[slotName]);
  }

  private static toComponentFile(
    target: JsonObject,
    filePath: string,
  ): ComponentFile {
    if (Object.hasOwn(target, 'screen_id')) {
      throw new Error('screen_id is not allowed in template nodes');
    }
    const unsupportedKey = Object.keys(target).find(
      (key) =>
        key.startsWith('.') &&
        key !== VARS_KEY &&
        key !== DEFAULTS_KEY &&
        key !== CONTENT_KEY,
    );
    if (unsupportedKey !== undefined) {
      throw new Error(`Unsupported build key ${unsupportedKey}`);
    }

    const vars = target[VARS_KEY];
    if (
      !Composer.isStringArray(vars) ||
      vars.some((name) => name.length === 0)
    ) {
      throw new Error(`Component .vars must be a list of names: ${filePath}`);
    }
    if (new Set(vars).size !== vars.length) {
      throw new Error(`Component .vars contains duplicate names: ${filePath}`);
    }
    const defaults = target[DEFAULTS_KEY];
    if (defaults !== undefined && !Composer.isObject(defaults)) {
      throw new Error(`Component .defaults must be a map: ${filePath}`);
    }
    if (defaults !== undefined) {
      for (const key of Object.keys(defaults)) {
        if (!vars.includes(key)) {
          throw new Error(
            `Component .defaults key not in .vars: ${key} (${filePath})`,
          );
        }
      }
    }
    if (!Object.hasOwn(target, CONTENT_KEY)) {
      throw new Error(
        `Component with .vars must declare .content: ${filePath}`,
      );
    }

    return { vars, defaults, content: target[CONTENT_KEY] };
  }

  private static slotName(node: JsonValue): string | undefined {
    if (
      typeof node !== 'string' ||
      !node.startsWith('.') ||
      node.length === 1
    ) {
      return undefined;
    }
    if (node.startsWith('./') || node.startsWith('..')) return undefined;
    return node.slice(1);
  }

  private static isReference(node: JsonValue): node is JsonObject {
    return Composer.isObject(node) && Object.hasOwn(node, REF_KEY);
  }

  private static isComponent(node: JsonValue): node is JsonObject {
    return Composer.isObject(node) && Object.hasOwn(node, VARS_KEY);
  }

  private static isStringArray(node: JsonValue): node is string[] {
    return (
      Array.isArray(node) && node.every((item) => typeof item === 'string')
    );
  }

  private static isObject(node: JsonValue): node is JsonObject {
    return typeof node === 'object' && node !== null && !Array.isArray(node);
  }

  private static clone(node: JsonValue): JsonValue {
    if (Array.isArray(node)) return node.map((item) => Composer.clone(item));
    if (!Composer.isObject(node)) return node;

    return Object.fromEntries(
      Object.entries(node).map(([key, value]) => [key, Composer.clone(value)]),
    );
  }
}
