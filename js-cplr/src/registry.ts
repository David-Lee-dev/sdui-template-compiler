import { createHash } from 'node:crypto';
import { dirname, join } from 'node:path';
import { Composer } from './composer.js';
import { IncludeResolver } from './include-resolver.js';
import type { ScreenModule } from './screen-manifest.js';
import { ScreenManifest } from './screen-manifest.js';
import { TokenResolver } from './token-resolver.js';
import type { JsonValue } from './types.js';
import { Versioning } from './versioning.js';
import { Yaml } from './yaml.js';

/**
 * One composed template and its content validator.
 *
 * `etag` is a content hash of the composed output — not the declared version
 * string. Screens are routinely edited in place inside one version directory
 * (`template/1.0.0/**`), which leaves the version unchanged; validating a
 * cache on the version would answer "unchanged" to a client holding a stale
 * template.
 */
export interface ComposedTemplate {
  template: JsonValue;
  etag: string;
}

interface RegisteredScreen {
  module: ScreenModule;
  templates: Map<string, ComposedTemplate>;
}

/** ETag length. Two revisions of one screen colliding does not happen in practice. */
const ETAG_LENGTH = 16;

export class ScreenRegistry {
  private constructor(
    private readonly screens: Map<string, RegisteredScreen>,
  ) {}

  /**
   * Composes and caches every template asset declared under an SDUI root.
   *
   * Screens are discovered from `screens/<dir>/screen.yaml` manifests;
   * components resolve from `_components/` and tokens from `_tokens/`.
   *
   * @param rootDir - SDUI root containing screens, components, and tokens
   * @param modules - Explicit modules override used by controlled test fixtures
   * @returns A registry serving composed templates by id and app version
   * @throws When ids are duplicated or composition fails
   */
  static build(
    rootDir: string,
    modules?: readonly ScreenModule[],
  ): ScreenRegistry {
    const discovered = modules ?? ScreenManifest.discover(rootDir);
    const includeResolver = IncludeResolver.create(rootDir);
    const tokenResolver = TokenResolver.create(rootDir);
    const screens = new Map<string, RegisteredScreen>();

    for (const module of discovered) {
      if (screens.has(module.id)) {
        throw new Error(`Duplicate screen id: ${module.id}`);
      }

      const templates = new Map<string, ComposedTemplate>();
      const templateVersions = new Set(
        Object.values(module.versions).map((entry) => entry.template),
      );
      for (const templateVersion of templateVersions) {
        const rootPath = join(
          module.dir,
          'template',
          templateVersion,
          '_root.yaml',
        );
        const template = Composer.expand(
          Yaml.load(rootPath),
          includeResolver,
          tokenResolver,
          rootDir,
          dirname(rootPath),
        );
        templates.set(templateVersion, {
          template,
          etag: ScreenRegistry.etagOf(template),
        });
      }

      screens.set(module.id, { module, templates });
    }

    return new ScreenRegistry(screens);
  }

  /**
   * Returns the composed screen selected for an application version.
   *
   * @param id - Authoritative declared screen id
   * @param appVersion - Optional client application version
   * @returns Composed template, or undefined when the id is unknown
   * @throws When the application version is malformed
   */
  get(id: string, appVersion?: string): JsonValue | undefined {
    return this.resolve(id, appVersion)?.template;
  }

  /**
   * Returns the composed screen for an application version with its etag.
   *
   * A serving layer needs both — the etag to answer a conditional GET, the
   * template to send when the client's copy is stale. [get] stays as the
   * template-only view for callers that never revalidate.
   *
   * @param id - Authoritative declared screen id
   * @param appVersion - Optional client application version
   * @returns Composed template and its content etag, or undefined when the id is unknown
   * @throws When the application version is malformed
   */
  resolve(id: string, appVersion?: string): ComposedTemplate | undefined {
    const screen = this.screens.get(id);
    if (screen === undefined) return undefined;

    const assets = Versioning.resolve(screen.module, appVersion);
    return screen.templates.get(assets.template);
  }

  /** Lists every registered screen id. */
  ids(): string[] {
    return [...this.screens.keys()];
  }

  /**
   * Lists app-version thresholds declared for a screen id.
   *
   * @param id - Authoritative declared screen id
   * @returns Version-map threshold keys in ascending semantic order
   */
  versionsOf(id: string): string[] {
    const screen = this.screens.get(id);
    if (screen === undefined) return [];

    return Object.keys(screen.module.versions).sort((left, right) =>
      Versioning.compareSemver(left, right),
    );
  }

  /** Returns the discovered module declaration for a screen id. */
  moduleOf(id: string): ScreenModule | undefined {
    return this.screens.get(id)?.module;
  }

  /**
   * Hashes a composed template into its cache validator.
   *
   * Runs once per template at build time; serving a request then costs one
   * string comparison.
   *
   * @param template - Composed template to fingerprint
   * @returns Hex etag of [ETAG_LENGTH] characters
   */
  private static etagOf(template: JsonValue): string {
    return createHash('sha256')
      .update(JSON.stringify(template))
      .digest('hex')
      .slice(0, ETAG_LENGTH);
  }
}
