import type { JsonValue } from './types.js';

export class Validator {
  /**
   * Rejects arguments absent from a component variable declaration.
   *
   * @param vars - Names declared by the component
   * @param args - Arguments supplied by the reference node
   * @returns Nothing when every argument is declared
   * @throws When one or more argument names are unknown
   */
  static assertArgsDeclared(
    vars: readonly string[],
    args: Readonly<Record<string, JsonValue>>,
  ): void {
    const unknown = Object.keys(args).filter((name) => !vars.includes(name));
    if (unknown.length > 0) {
      throw new Error(
        `Component reference has unknown args: ${unknown.join(', ')}`,
      );
    }
  }

  /**
   * Rejects a variable slot absent from its component declaration.
   *
   * @param vars - Names declared by the component
   * @param name - Variable slot name found in the component body
   * @returns Nothing when the slot is declared
   * @throws When the slot name is undeclared
   */
  static assertHoleDeclared(vars: readonly string[], name: string): void {
    if (!vars.includes(name)) {
      throw new Error(`Component body has undeclared slot .${name}`);
    }
  }

  /**
   * Rejects a resolved file path re-entered on the active expansion chain.
   *
   * @param filePath - Absolute component or fragment path about to be expanded
   * @param visiting - Ordered absolute paths currently being expanded
   * @returns Nothing when the file can be entered safely
   * @throws When the path already exists in the active chain
   */
  static assertNoCircularPath(
    filePath: string,
    visiting: readonly string[],
  ): void {
    if (visiting.includes(filePath)) {
      throw new Error(
        `Circular reference: ${[...visiting, filePath].join(' -> ')}`,
      );
    }
  }
}
