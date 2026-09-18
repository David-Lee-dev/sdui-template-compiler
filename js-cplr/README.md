# sdui-template-compiler (TypeScript)

> 한국어: [README.ko.md](README.ko.md)

The reference implementation of the [SDUI template compiler](../README.md).
ESM, Node ≥ 20, one runtime dependency (`js-yaml`).

Not yet published to npm — consume it from source as a path dependency.

## Install

```sh
cd js-cplr
pnpm install && pnpm build        # emits dist/
```

From another project:

```jsonc
// package.json
{ "dependencies": { "sdui-template-compiler": "file:../sdui-template-compiler/js-cplr" } }
```

## CLI

```sh
node dist/cli.js <sdui-root> --out <dir> [--pretty]
# or, once installed as a dependency: pnpm exec sdui-compile <root> --out <dir>
```

Writes `<dir>/manifest.json` plus `<dir>/screens/<id>/<templateVersion>.json`.
`--pretty` affects screen JSON readability only — etags always hash the
compact form.

## Library

```ts
import { ScreenRegistry } from 'sdui-template-compiler';

// Compiles every screen under the root eagerly — throws on any template error.
const registry = ScreenRegistry.build('path/to/sdui-root');

// Newest version threshold ≤ the client app version
// (selection rules: ../docs/integration.md).
const composed = registry.resolve('home', '2.3.0');
if (composed !== undefined) {
  composed.template; // client-ready JSON tree
  composed.etag;     // sha256[:16] of the compact serialization
}

registry.get('home', '2.3.0');   // just the template
registry.ids();                  // all screen ids, in discovery order
registry.versionsOf('home');     // app-version thresholds, ascending
registry.moduleOf('home');       // screen.yaml manifest (versions, params)
```

Errors are thrown `Error`s whose messages carry the diagnostic substrings
fixed by the conformance spec — catch at your build/boot boundary.

Lower-level pieces (`Composer`, `IncludeResolver`, `TokenResolver`,
`Versioning`, `Yaml`, …) are exported for tooling that needs to compose
single trees; `ScreenRegistry` is the intended entry point.

## Tests

```sh
pnpm test          # conformance suite against ../spec
pnpm typecheck
```
