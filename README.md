# SDUI Template Compiler

> 한국어: [README.ko.md](README.ko.md)

A build tool for server-driven UI: it compiles a tree of **YAML screen
templates** — with shared components, design tokens, and app-version
thresholds — into fully composed, client-ready **JSON + an etag manifest**.

Three conforming implementations ship in this repository — **TypeScript**
(reference), **Python**, and **Go** — locked together by a shared,
executable conformance spec: all three produce **byte-identical output**
for the same input.

```text
 authoring (this compiler)                        serving (your server)        rendering (client)
┌──────────────────────────┐   sdui-compile   ┌──────────────────────┐   HTTP   ┌───────────────┐
│ <root>/                  │ ───────────────▶ │ screens/<id>/<v>.json│ ───────▶ │ SDUI engine   │
│   _tokens/*.yaml         │                  │ manifest.json        │          │ (e.g. Flutter)│
│   _components/*.yaml     │                  │  · version map       │          └───────────────┘
│   screens/<id>/          │                  │  · params            │
│     screen.yaml          │                  │  · etags             │
│     template/<v>/_root…  │                  └──────────────────────┘
└──────────────────────────┘
```

## Why

- **Ship UI without app releases.** Screens are data. Edit YAML, recompile,
  deploy JSON — clients pick up the change on next load.
- **Author with structure, serve flat.** Components (`.ref` with declared
  variables and defaults), design tokens (`.token`), and fragments keep
  authoring DRY; the compiler resolves everything at build time, so servers
  and clients only ever see plain JSON.
- **Version templates against app versions.** Each screen maps app-version
  thresholds to template versions (`screen.yaml`); `resolve(id, appVersion)`
  picks the right asset for every client, old or new.
- **No server framework required.** Output is static JSON plus a manifest —
  serve it from any language, any framework, or a CDN. Caching works over
  standard etag revalidation.
- **Pick your language.** The TS, Python, and Go ports implement one
  normative spec ([`spec/SPEC.md`](spec/SPEC.md)) and are held
  byte-identical by a shared golden-fixture suite — the drift guard every
  port must keep green.

## Quick start

Compile the example fixture with the TypeScript CLI (any port works — same
flags, same bytes):

```sh
cd js-cplr
pnpm install && pnpm build
node dist/cli.js ../spec/fixtures/basic/input --out /tmp/sdui-out --pretty
```

You get one JSON file per screen per template version, plus the manifest:

```text
/tmp/sdui-out/
  manifest.json                 # { home: { versions, params, etags }, ... }
  screens/home/1.0.0.json
  screens/detail/1.0.0.json
```

Or resolve screens in-process instead of precompiling:

```ts
import { ScreenRegistry } from 'sdui-template-compiler';

const registry = ScreenRegistry.build('path/to/sdui-root');
const composed = registry.resolve('home', '2.3.0'); // newest threshold ≤ 2.3.0
// composed.template — client-ready JSON tree
// composed.etag     — sha256[:16] content hash for cache revalidation
```

Python and Go expose the same model — see the
[per-port READMEs](#the-ports) for installation and idiomatic examples.

## What a template looks like

```yaml
# screens/home/template/1.0.0/_root.yaml
_type: column
_children:
  - _type: text
    value: "Today's picks"
    style: { color: { .token: color.primary } }   # design-token lookup
  - { .ref: /card, padding: 16,                   # shared component + args
      child: { _type: text, value: hello } }
```

```yaml
# _components/card.yaml — declared variables, defaults, slot substitution
.vars: [padding, child]
.defaults: { padding: 12 }
.content:
  _type: container
  padding: .padding
  _child: .child
```

The compiler owns the **dot-prefixed build keys** (`.ref`, `.vars`,
`.defaults`, `.content`, `.token` — resolved at build time, none survive
into output; dot-prefixed *strings* are slots, but only inside a component's
`.content`). Everything **underscore-prefixed** (`_type`, `_scope`, `${bindings}`, …) is
runtime vocabulary that passes through untouched — it belongs to whatever
client engine renders the JSON. Full authoring guide:
[`docs/template-guide.md`](docs/template-guide.md).

## The ports

| Port | Language | Library | CLI |
|---|---|---|---|
| [`js-cplr/`](js-cplr) | TypeScript (reference) | `sdui-template-compiler` (ESM) | `sdui-compile` |
| [`py-cplr/`](py-cplr) | Python ≥ 3.11 | `sdui_template_compiler` | `sdui-compile` |
| [`go-cplr/`](go-cplr) | Go ≥ 1.22 | `github.com/David-Lee-dev/sdui-template-compiler/go-cplr` | `go run ./cmd/sdui-compile` |

None of the ports is published to a package registry yet, but each installs
straight from this repository like a normal package — no local checkout or
registry server needed:

```sh
# Python — pip installs directly from git (pin with ...git@<tag>#subdirectory=...)
pip install "git+https://github.com/David-Lee-dev/sdui-template-compiler.git#subdirectory=py-cplr"

# Go — module path works with go get as usual
go get github.com/David-Lee-dev/sdui-template-compiler/go-cplr
```

The TypeScript port installs from a local path or a git dependency in
package.json. Each port README shows the details, including editable installs
for development.

## Conformance: what "byte-identical" means

[`spec/SPEC.md`](spec/SPEC.md) is the normative contract;
[`spec/cases.yaml`](spec/cases.yaml) + [`spec/fixtures/`](spec/fixtures) are
its executable form. Every port runs the same suite, which locks:

- composed output bytes — compact JSON, UTF-8, **key order preserved from
  the YAML source**, ECMAScript number serialization (`1e-7`, not `1e-07`);
- the CLI manifest — screen discovery order, pretty serialization, etags
  (sha256 of the compact form, truncated to 16 hex chars);
- app-version threshold selection, including fallbacks and malformed input;
- YAML dialect (YAML 1.2 core as js-yaml v4 interprets it);
- error diagnostics — for the covered failure modes, every port's message
  contains the same substring, so operators see the same diagnostics in
  every language.

The suite is the drift guard, not a proof of totality — it covers the
contract through representative vectors, and grows with every contract
change ([CONTRIBUTING](CONTRIBUTING.md#changing-the-contract)). Making it
green is also the porting workflow: [`docs/porting.md`](docs/porting.md).

## Ecosystem

This repository is the authoring/build side of a three-part stack. The parts
are independent — the compiler has no dependency on the engine or vice versa;
they meet only at the JSON + manifest format.

| Repo | Role |
|---|---|
| **sdui-template-compiler** (this repo) | YAML → composed JSON + manifest |
| [sdui-flutter-engine](https://github.com/David-Lee-dev/sdui-flutter-engine) | Flutter package that renders the compiled JSON reactively |
| [sdui-flutter-starter-kit](https://github.com/David-Lee-dev/sdui-flutter-starter-kit) | Runnable Flutter app + example server consuming both |

## Documentation

| Audience | Read |
|---|---|
| Template authors | [`docs/template-guide.md`](docs/template-guide.md) — tutorial: root layout, components, tokens, slots, versioning |
| Server/build integrators | [`docs/integration.md`](docs/integration.md) — CLI vs library, artifact layout, serving algorithm, cache invalidation |
| Port authors | [`docs/porting.md`](docs/porting.md) — implement the spec in a new language |
| Contributors | [`CONTRIBUTING.md`](CONTRIBUTING.md) — repo layout, test commands, contract-change workflow |
| Everyone (normative) | [`spec/SPEC.md`](spec/SPEC.md) — the compiler contract |

## Verifying

Each port must be green against the same `spec/` vectors:

```sh
(cd js-cplr && pnpm install && pnpm test)
(cd py-cplr && python3 -m venv .venv && .venv/bin/pip install -e '.[dev]' \
            && .venv/bin/python -m pytest)
(cd go-cplr && go test ./...)
```

## License

[MIT](LICENSE)
