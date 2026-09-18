import { readFileSync } from 'node:fs';
import { load as loadYaml } from 'js-yaml';
import type { JsonValue } from './types.js';

export class Yaml {
  /**
   * Loads a YAML file as a JSON-compatible value.
   *
   * @param path - YAML file path
   * @returns Parsed JSON-compatible value
   * @throws When YAML contains a value that JSON cannot represent
   */
  static load(path: string): JsonValue {
    const parsed: unknown = loadYaml(readFileSync(path, 'utf8'));
    return Yaml.assertJsonValue(parsed, path);
  }

  private static assertJsonValue(value: unknown, path: string): JsonValue {
    if (
      value === null ||
      typeof value === 'boolean' ||
      typeof value === 'string'
    ) {
      return value;
    }
    if (typeof value === 'number' && Number.isFinite(value)) return value;
    if (Array.isArray(value)) {
      return value.map((item) => Yaml.assertJsonValue(item, path));
    }
    if (Yaml.isPlainObject(value)) {
      return Object.fromEntries(
        Object.entries(value).map(([key, item]) => [
          key,
          Yaml.assertJsonValue(item, path),
        ]),
      );
    }

    throw new Error(`YAML is not JSON-compatible: ${path}`);
  }

  private static isPlainObject(
    value: unknown,
  ): value is Record<string, unknown> {
    if (typeof value !== 'object' || value === null) return false;
    const prototype = Object.getPrototypeOf(value) as unknown;
    return prototype === Object.prototype || prototype === null;
  }
}
