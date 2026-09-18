import { existsSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import type { JsonValue } from './types.js';
import { Yaml } from './yaml.js';

export interface ScreenVersionAssets {
  template: string;
}

/**
 * A screen declaration loaded from `screens/<dir>/screen.yaml`.
 *
 * The manifest is a plain YAML file — not code — so every compiler port
 * (TS, Python, Go) reads the exact same declaration:
 *
 * ```yaml
 * id: home
 * versions:
 *   "1.0.0": { template: "1.0.0" }
 * params: [tab]
 * ```
 */
export interface ScreenModule {
  id: string;
  /** Absolute path to the screen directory containing its template assets. */
  dir: string;
  versions: Record<string, ScreenVersionAssets>;
  /**
   * Route query-parameter keys the template may bind as root data.
   *
   * The serving route forwards its query parameters into the engine's root
   * binding scope, but that contract is otherwise implicit — nothing ties a
   * `${id}` binding in a template back to the URL that must supply it.
   * Declaring the keys here makes the contract explicit and lets tooling seed
   * the right root data instead of guessing. Defaults to an empty array.
   */
  params: readonly string[];
}

const SEMANTIC_VERSION = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;
const MANIFEST_FILE = 'screen.yaml';

export class ScreenManifest {
  /**
   * Discovers every screen manifest under `<rootDir>/screens`.
   *
   * @param rootDir - SDUI root containing the `screens` directory
   * @returns Validated screen modules in directory order
   * @throws When a manifest is invalid or two screens declare the same id
   */
  static discover(rootDir: string): ScreenModule[] {
    const screensDir = join(rootDir, 'screens');
    if (!existsSync(screensDir)) {
      throw new Error(`Missing screens directory: ${screensDir}`);
    }

    const modules: ScreenModule[] = [];
    const seen = new Set<string>();
    for (const entry of readdirSync(screensDir, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const manifestPath = join(screensDir, entry.name, MANIFEST_FILE);
      if (!existsSync(manifestPath)) continue;

      const module = ScreenManifest.load(
        manifestPath,
        join(screensDir, entry.name),
      );
      if (seen.has(module.id)) {
        throw new Error(`Duplicate screen id: ${module.id}`);
      }
      seen.add(module.id);
      modules.push(module);
    }
    return modules;
  }

  /**
   * Loads and validates a single screen manifest file.
   *
   * @param manifestPath - Path to the `screen.yaml` file
   * @param dir - Screen directory owning the manifest
   * @returns The validated screen module
   * @throws When the id, version map, or a version threshold is invalid
   */
  static load(manifestPath: string, dir: string): ScreenModule {
    const raw = Yaml.load(manifestPath);
    if (!ScreenManifest.isObject(raw)) {
      throw new Error(`Screen manifest must be a map: ${manifestPath}`);
    }

    const id = raw['id'];
    if (typeof id !== 'string' || id.trim().length === 0) {
      throw new Error(`Screen manifest must declare an id: ${manifestPath}`);
    }

    const versionsRaw = raw['versions'];
    if (!ScreenManifest.isObject(versionsRaw)) {
      throw new Error(`Screen <${id}> must declare a versions map`);
    }
    const versionEntries = Object.entries(versionsRaw);
    if (versionEntries.length === 0) {
      throw new Error(`Screen <${id}> must declare at least one version`);
    }

    const versions: Record<string, ScreenVersionAssets> = {};
    for (const [version, assets] of versionEntries) {
      if (!SEMANTIC_VERSION.test(version)) {
        throw new Error(`Invalid semantic version: ${version}`);
      }
      if (!ScreenManifest.isObject(assets)) {
        throw new Error(`Screen <${id}> version ${version} must be a map`);
      }
      const template = assets['template'];
      if (typeof template !== 'string' || template.length === 0) {
        throw new Error(
          `Screen <${id}> version ${version} must declare a template`,
        );
      }
      versions[version] = { template };
    }

    const paramsRaw = raw['params'] ?? [];
    if (
      !Array.isArray(paramsRaw) ||
      paramsRaw.some((item) => typeof item !== 'string')
    ) {
      throw new Error(`Screen <${id}> params must be a list of names`);
    }

    return {
      id,
      dir,
      versions,
      params: Object.freeze([...(paramsRaw as string[])]),
    };
  }

  private static isObject(
    value: JsonValue,
  ): value is Record<string, JsonValue> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
  }
}
