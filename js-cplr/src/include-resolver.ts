import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import type { IncludeResolver as IncludeResolverFn } from './types.js';
import { Yaml } from './yaml.js';

export class IncludeResolver {
  /**
   * Creates a disk-backed resolver for component and fragment references.
   *
   * @param rootDir - SDUI root used as the base for reference paths
   * @returns Resolver that returns an absolute path and parsed YAML content
   * @throws When a referenced file is missing or is not JSON-compatible
   */
  static create(rootDir: string): IncludeResolverFn {
    return (refPath, referrerDir) => {
      const filePath = refPath.startsWith('/')
        ? resolve(rootDir, '_components', `${refPath.slice(1)}.yaml`)
        : resolve(referrerDir, `${refPath}.yaml`);
      if (!existsSync(filePath)) {
        throw new Error(`Unresolved .ref ${refPath}: ${filePath}`);
      }

      return { filePath, content: Yaml.load(filePath) };
    };
  }
}
