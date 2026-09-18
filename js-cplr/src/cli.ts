#!/usr/bin/env node
import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { ScreenRegistry } from './registry.js';

interface CliOptions {
  rootDir: string;
  outDir: string;
  pretty: boolean;
}

/**
 * Compiles every screen under an SDUI root into composed JSON files.
 *
 * Output layout:
 * ```
 * <outDir>/
 *   manifest.json                    # per screen: versions map + etags
 *   screens/<id>/<templateVersion>.json
 * ```
 * A server (any language) can serve this output statically — the manifest
 * carries everything needed for app-version selection and etag revalidation.
 */
function main(argv: string[]): number {
  const options = parseArgs(argv);
  if (options === undefined) {
    process.stderr.write(
      'Usage: sdui-compile <rootDir> --out <outDir> [--pretty]\n',
    );
    return 1;
  }

  const registry = ScreenRegistry.build(options.rootDir);
  const indent = options.pretty ? 2 : undefined;
  const manifest: Record<string, unknown> = {};

  mkdirSync(options.outDir, { recursive: true });
  for (const id of registry.ids()) {
    const module = registry.moduleOf(id);
    if (module === undefined) continue;

    const screenDir = join(options.outDir, 'screens', id);
    mkdirSync(screenDir, { recursive: true });

    const etags: Record<string, string> = {};
    const written = new Set<string>();
    for (const assets of Object.values(module.versions)) {
      if (written.has(assets.template)) continue;
      written.add(assets.template);

      const composed = compiledTemplate(registry, id, assets.template);
      writeFileSync(
        join(screenDir, `${assets.template}.json`),
        JSON.stringify(composed.template, null, indent),
      );
      etags[assets.template] = composed.etag;
    }

    manifest[id] = {
      versions: module.versions,
      params: module.params,
      etags,
    };
  }

  writeFileSync(
    join(options.outDir, 'manifest.json'),
    JSON.stringify(manifest, null, 2),
  );
  process.stdout.write(
    `Compiled ${registry.ids().length} screen(s) to ${options.outDir}\n`,
  );
  return 0;
}

function compiledTemplate(
  registry: ScreenRegistry,
  id: string,
  templateVersion: string,
) {
  const module = registry.moduleOf(id);
  if (module === undefined) throw new Error(`Unknown screen: ${id}`);

  // Resolve via the app-version threshold that maps to this template version.
  const threshold = Object.entries(module.versions).find(
    ([, assets]) => assets.template === templateVersion,
  );
  if (threshold === undefined) {
    throw new Error(`Screen <${id}> has no threshold for ${templateVersion}`);
  }
  const composed = registry.resolve(id, threshold[0]);
  if (composed === undefined) {
    throw new Error(`Screen <${id}> failed to resolve ${templateVersion}`);
  }
  return composed;
}

function parseArgs(argv: string[]): CliOptions | undefined {
  const positional: string[] = [];
  let outDir: string | undefined;
  let pretty = false;

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--out' || arg === '-o') {
      outDir = argv[i + 1];
      i += 1;
    } else if (arg === '--pretty') {
      pretty = true;
    } else {
      positional.push(arg);
    }
  }

  if (positional.length !== 1 || outDir === undefined) return undefined;
  return { rootDir: positional[0], outDir, pretty };
}

process.exit(main(process.argv.slice(2)));
