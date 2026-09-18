# sdui-template-compiler (Python)

> 한국어: [README.ko.md](README.ko.md)

Python port of the [SDUI template compiler](../README.md). Python ≥ 3.11,
one runtime dependency (`PyYAML`). Ported from the TypeScript
reference — held byte-identical by the shared conformance suite (the drift guard) in
[`../spec`](../spec).

Not yet published to PyPI — install from source.

## Install

```sh
pip install path/to/sdui-template-compiler/py-cplr
# or editable, for development:
pip install -e 'path/to/sdui-template-compiler/py-cplr[dev]'
```

## CLI

```sh
sdui-compile <sdui-root> --out <dir> [--pretty]
```

Writes `<dir>/manifest.json` plus `<dir>/screens/<id>/<templateVersion>.json`.
`--pretty` affects screen JSON readability only — etags always hash the
compact form.

## Library

```python
from sdui_template_compiler import ScreenRegistry

# Compiles every screen under the root eagerly — raises on any template error.
registry = ScreenRegistry.build("path/to/sdui-root")

# Newest version threshold ≤ the client app version
# (selection rules: ../docs/integration.md).
composed = registry.resolve("home", "2.3.0")
if composed is not None:
    composed.template  # client-ready JSON tree (plain dict/list/scalars)
    composed.etag      # sha256[:16] of the compact serialization

registry.get("home", "2.3.0")   # just the template
registry.ids()                  # all screen ids, in discovery order
registry.versions_of("home")    # app-version thresholds, ascending
registry.module_of("home")      # screen.yaml manifest (versions, params)
```

Errors are raised exceptions whose messages carry the diagnostic substrings
fixed by the conformance spec — catch at your build/boot boundary.

Two port-specific notes:

- **YAML dialect**: the loader is tuned to YAML 1.2 core semantics
  (`on`/`yes` stay strings, `1.0.0` stays a string) — PyYAML's default 1.1
  resolver is deliberately overridden.
- **Serialization**: `json_compat` reproduces `JSON.stringify` number
  formatting so compact output and etags match the reference byte-for-byte.

## Tests

```sh
python3 -m venv .venv
.venv/bin/pip install -e '.[dev]'
.venv/bin/python -m pytest        # conformance suite against ../spec
```
