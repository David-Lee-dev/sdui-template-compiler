import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import type {
  JsonObject,
  JsonValue,
  TokenResolver as TokenResolverFn,
} from './types.js';
import { Yaml } from './yaml.js';

export class TokenResolver {
  /**
   * Creates a cached disk-backed resolver for design-token groups.
   *
   * @param rootDir - SDUI root containing the `_tokens` directory
   * @returns Resolver that loads each token group map at most once
   * @throws When a group file is missing or does not contain a map
   */
  static create(rootDir: string): TokenResolverFn {
    const cache = new Map<string, JsonObject>();

    return (group) => {
      const cached = cache.get(group);
      if (cached !== undefined) return cached;

      const filePath = resolve(rootDir, '_tokens', `${group}.yaml`);
      if (!existsSync(filePath)) {
        throw new Error(`Unknown token group ${group}: ${filePath}`);
      }

      const tokens = Yaml.load(filePath);
      TokenResolver.assertTokenMap(tokens, group, filePath);
      cache.set(group, tokens);
      return tokens;
    };
  }

  private static assertTokenMap(
    value: JsonValue,
    group: string,
    filePath: string,
  ): asserts value is JsonObject {
    if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
      return;
    }

    throw new Error(`Token group ${group} must contain a map: ${filePath}`);
  }
}
