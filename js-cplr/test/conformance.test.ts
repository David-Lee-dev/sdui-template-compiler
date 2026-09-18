import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { load as loadYaml } from 'js-yaml';
import { describe, expect, it } from 'vitest';
import { ScreenRegistry } from '../src/registry.js';

const SPEC_DIR = join(import.meta.dirname, '..', '..', 'spec');
const FIXTURES_DIR = join(SPEC_DIR, 'fixtures');

interface VersioningQuery {
  app_version: string | null;
  template: string;
}

interface Cases {
  golden: string[];
  versioning: {
    fixture: string;
    screen: string;
    queries: VersioningQuery[];
    invalid_app_versions: string[];
  };
  errors: Record<string, string>;
}

const cases = loadYaml(
  readFileSync(join(SPEC_DIR, 'cases.yaml'), 'utf8'),
) as Cases;

describe('conformance', () => {
  describe('golden', () => {
    for (const fixture of cases.golden) {
      it(`composes ${fixture} byte-identically`, () => {
        const input = join(FIXTURES_DIR, fixture, 'input');
        const expectedDir = join(FIXTURES_DIR, fixture, 'expected');
        const registry = ScreenRegistry.build(input);
        const manifest = JSON.parse(
          readFileSync(join(expectedDir, 'manifest.json'), 'utf8'),
        ) as Record<string, { etags: Record<string, string> }>;

        for (const id of Object.keys(manifest)) {
          const screenDir = join(expectedDir, 'screens', id);
          for (const file of readdirSync(screenDir)) {
            const templateVersion = file.replace(/\.json$/, '');
            const composed = resolveTemplateVersion(
              registry,
              id,
              templateVersion,
            );
            expect(JSON.stringify(composed.template)).toBe(
              readFileSync(join(screenDir, file), 'utf8'),
            );
            expect(composed.etag).toBe(manifest[id].etags[templateVersion]);
          }
        }
      });
    }
  });

  describe('versioning', () => {
    const spec = cases.versioning;
    const input = join(FIXTURES_DIR, spec.fixture, 'input');

    for (const query of spec.queries) {
      it(`selects ${query.template} for app version ${query.app_version ?? '(none)'}`, () => {
        const registry = ScreenRegistry.build(input);
        const composed = registry.resolve(
          spec.screen,
          query.app_version ?? undefined,
        );
        const viaThreshold = resolveTemplateVersion(
          registry,
          spec.screen,
          query.template,
        );
        expect(composed?.etag).toBe(viaThreshold.etag);
      });
    }

    for (const invalid of spec.invalid_app_versions) {
      it(`rejects malformed app version ${invalid}`, () => {
        const registry = ScreenRegistry.build(input);
        expect(() => registry.resolve(spec.screen, invalid)).toThrow(
          'Invalid semantic version',
        );
      });
    }
  });

  describe('errors', () => {
    for (const [fixture, message] of Object.entries(cases.errors)) {
      it(`rejects ${fixture} mentioning "${message}"`, () => {
        const input = join(FIXTURES_DIR, 'errors', fixture, 'input');
        expect(() => ScreenRegistry.build(input)).toThrow(message);
      });
    }
  });
});

function resolveTemplateVersion(
  registry: ScreenRegistry,
  id: string,
  templateVersion: string,
) {
  const module = registry.moduleOf(id);
  if (module === undefined) throw new Error(`Unknown screen: ${id}`);
  const threshold = Object.entries(module.versions).find(
    ([, assets]) => assets.template === templateVersion,
  );
  if (threshold === undefined) {
    throw new Error(`No threshold maps to template ${templateVersion}`);
  }
  const composed = registry.resolve(id, threshold[0]);
  if (composed === undefined) throw new Error(`Failed to resolve ${id}`);
  return composed;
}
