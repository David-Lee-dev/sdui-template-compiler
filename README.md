# SDUI Starter — Packages

Reusable packages behind the [flutter-sdui-starter-kit](../08_flutter-sdui-starter-kit)
starter project. Each directory is an independently consumable package; the
compilers are kept byte-identical across languages by the shared conformance
spec in `spec/`.

| Package | Language | Use from another project |
|---|---|---|
| `flutter-engine/` | — | separate repo: [sdui-flutter-engine](https://github.com/David-Lee-dev/sdui-flutter-engine) — checked out here as a sibling dir |
| `js-cplr/` | TypeScript/JS (`sdui-template-compiler`) | `pnpm add <path>` → library import or `sdui-compile` CLI |
| `py-cplr/` | Python (`sdui-template-compiler`) | `pip install <path>` → `sdui_template_compiler` or `sdui-compile` |
| `go-cplr/` | Go (`github.com/sdui-starter/template-compiler`) | `go get`/replace → library or `go run ./cmd/sdui-compile` |
| `spec/` | — | Conformance vectors + golden fixtures (SPEC.md is the compiler contract) |

## flutter-engine (`sdui_engine`)

Every dependency is implemented by the app and injected (the package ships
only a no-op telemetry fallback — see `flutter-engine/README.md`):

```dart
Sdui.initialize(
  screenLoader: ..., apiClient: ...,          // your implementations —
  imageSource: ..., videoSource: ...,         // the package ships none
  appStorage: ..., secureStorage: ...,
);
runApp(MaterialApp.router(routerConfig: Sdui.router()));
```

Layering: `shell/` (Sdui facade, screen page, router) over `defaults/`
(HTTP loader/client, memory stores, network media, haptic driver) over
`contract/` ports over the pure engine core — full-control apps can skip the
facade and drive `Engine.initialize` + `EngineRunner` directly.

## Compilers

All three implement `spec/SPEC.md`: YAML screen trees (components, tokens,
version thresholds) → composed JSON + etag manifest. Same CLI everywhere:

```sh
sdui-compile <sdui-root> --out <dir> [--pretty]
```

Verifying (each must be green against the same `spec/` vectors):

```sh
cd js-cplr && pnpm install && pnpm test
cd py-cplr && python3 -m venv .venv && .venv/bin/pip install -e '.[dev]' && .venv/bin/python -m pytest
cd go-cplr && go test ./...
cd flutter-engine && flutter test
```

Porting to a new language = making the `spec/` suite green (byte-identical
compact JSON, preserved key order, sha256[:16] etags, matching error substrings).
