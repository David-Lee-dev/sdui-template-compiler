# sdui-template-compiler (Python)

Python port of the TypeScript reference compiler in `../ts`. Both ports run the
language-neutral conformance vectors in `../spec/cases.yaml`.

## Setup and tests

```sh
cd compiler/python
python3 -m venv .venv
.venv/bin/pip install -e '.[dev]'
.venv/bin/python -m pytest
```

## CLI

```sh
sdui-compile <rootDir> --out <outDir> [--pretty]
```

Writes `<outDir>/manifest.json` and `<outDir>/screens/<id>/<templateVersion>.json`.
