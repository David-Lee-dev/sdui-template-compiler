# sdui-template-compiler (Go)

> 한국어: [README.ko.md](README.ko.md)

Go port of the [SDUI template compiler](../README.md). Go ≥ 1.22, one
dependency (`gopkg.in/yaml.v3`). Ported from the TypeScript
reference — held byte-identical by the shared conformance suite (the drift guard) in [`../spec`](../spec).

Module path: `github.com/David-Lee-dev/sdui-template-compiler/go-cplr`.
Until tagged module versions are published, consume it via a checkout and a
`replace` directive:

```go.mod
require github.com/David-Lee-dev/sdui-template-compiler/go-cplr v0.0.0
replace github.com/David-Lee-dev/sdui-template-compiler/go-cplr => ../sdui-template-compiler/go-cplr
```

## CLI

```sh
go run ./cmd/sdui-compile <sdui-root> --out <dir> [--pretty]
```

Writes `<dir>/manifest.json` plus `<dir>/screens/<id>/<templateVersion>.json`.
`--pretty` affects screen JSON readability only — etags always hash the
compact form.

## Library

```go
import sdui "github.com/David-Lee-dev/sdui-template-compiler/go-cplr"

// Compiles every screen under the root eagerly — errors on any template problem.
registry, err := sdui.BuildScreenRegistry("path/to/sdui-root")
if err != nil { /* template error: diagnostic substrings fixed by the spec */ }

// Newest version threshold ≤ the client app version
// (selection rules: ../docs/integration.md).
composed, err := registry.Resolve("home", "2.3.0")
if err != nil { /* e.g. malformed app version */ }
if composed == nil { /* unknown screen — map to your 404 */ }
// composed.Template — client-ready JSON tree (ordered Value model)
// composed.Etag     — sha256[:16] of the compact serialization

template, err := registry.Get("home", "2.3.0") // just the template (nil, nil when unknown)
registry.IDs()                                 // all screen ids, in discovery order
registry.VersionsOf("home")                    // app-version thresholds, ascending
registry.ModuleOf("home")                      // screen.yaml manifest (versions, params)
```

Failures are ordinary `error` returns; their messages carry the same
diagnostic substrings as the TS and Python ports.

Two port-specific notes:

- **Ordered values**: composed templates use a dedicated `Value` model
  (not `map[string]any`) because the contract preserves YAML key order —
  `encoding/json` would sort keys.
- **Serialization**: a hand-written JSON writer reproduces
  `JSON.stringify` semantics (compact bytes, JS number formatting) so
  output and etags match the reference byte-for-byte.

## Tests

```sh
go test ./...        # conformance suite against ../spec
```
