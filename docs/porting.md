# Porting Guide

> 한국어: [porting.ko.md](porting.ko.md)

How to implement this compiler in another language. The definition of done
is mechanical: **implement [`spec/SPEC.md`](../spec/SPEC.md) and pass the
shared conformance suite** — [`spec/cases.yaml`](../spec/cases.yaml) run
against [`spec/fixtures/`](../spec/fixtures). The suite is the drift guard
(it exercises the contract through representative vectors, not every
conceivable input), so the spec is what you implement and the suite is how
drift gets caught.

## What you are building

A library (plus optionally a CLI) that:

1. reads an SDUI root directory (`_tokens/`, `_components/`,
   `screens/*/screen.yaml` + `template/<v>/_root.yaml`);
2. resolves the build keys (`.ref` / `.vars` / `.defaults` / `.content` /
   `.token`) into a plain JSON tree per screen per template version;
3. selects assets by app-version threshold (`resolve(id, appVersion)`);
4. serializes compact JSON **byte-identically** to the reference and
   derives sha256[:16] etags from it.

The normative behavior — the rules, their edge cases, and the required
error substrings — is [`spec/SPEC.md`](../spec/SPEC.md). Read it end to end first;
it is two pages. This guide covers only what the spec can't: workflow and
the places ports actually go wrong.

## Reference implementations

- [`js-cplr/`](../js-cplr) — TypeScript, the **reference**: when the spec
  is ambiguous, the TS behavior + the goldens decide, and the spec gets
  fixed to match.
- [`py-cplr/`](../py-cplr), [`go-cplr/`](../go-cplr) — worked examples of
  exactly this porting exercise, in a dynamic and a static language. Copy
  their structure: the module split (yaml / composer / include-resolver /
  token-resolver / versioning / screen-manifest / registry / validator /
  cli) maps one-to-one across all three.

## Workflow

1. **Write the conformance harness first.** One test file that reads
   `../spec/cases.yaml` and drives the three case kinds:
   - `golden`: compile `fixtures/<name>/input`, compare each
     `screens/<id>/<v>.json` **byte-for-byte** against `expected/`, compare
     etags against `expected/manifest.json`, and rebuild the CLI-style
     manifest object to byte-compare its pretty serialization against
     `expected/manifest.json` (this locks discovery order too);
   - `versioning`: build the fixture, assert `resolve(screen, app_version)`
     picks the expected template for each query, and that each
     `invalid_app_versions` entry throws;
   - `errors`: each fixture under `fixtures/errors/<name>/input` must fail
     with a message **containing** the listed substring.

   See `js-cplr/test/conformance.test.ts`,
   `py-cplr/tests/test_conformance.py`, or `go-cplr/conformance_test.go` —
   they are all under ~200 lines. The suite assumes the sibling checkout
   layout (`../spec` from the port directory).
2. **Implement until green.** A productive order: YAML loading → token
   resolution → include/component expansion → screen manifest + versioning
   → registry + etag. The error cases usually fall out of doing each layer
   honestly.
3. **Don't regenerate goldens.** The `expected/` files are the contract. If
   your output differs, your port is wrong — or you've found a reference
   bug, which is a spec discussion
   ([CONTRIBUTING](../CONTRIBUTING.md#changing-the-contract)), not a
   fixture edit.

## Where ports actually fail

Every one of these bit the Python or Go port. Budget your time here, not in
the tree walking.

**JSON serialization.** The contract is `JSON.stringify` semantics —
compact (no whitespace), UTF-8, and **key order preserved from the YAML
source** through every transformation:

- Your language's default map is probably **sorted or unordered** — Python
  needed nothing special (`dict` preserves insertion), Go needed a
  dedicated ordered-value model and a hand-written JSON writer because
  `map[string]any` + `encoding/json` sorts keys.
- Numbers serialize like JavaScript: integral floats print **without**
  a trailing `.0` (`1.0` → `1`), non-integral floats in shortest-roundtrip
  form. Check your formatter against the goldens' literals.
- No trailing newline, no escaping beyond what JSON requires.

The etag is sha256 over exactly these bytes, truncated to 16 hex chars —
get the bytes right and the etag is free; get them subtly wrong and every
golden fails at once, which is the suite working as intended.

**YAML dialect.** The contract is YAML 1.2 **core schema as js-yaml v4
implements it** — most YAML libraries default to YAML 1.1:

- `on` / `off` / `yes` / `no` must stay **strings**, not booleans (PyYAML's
  default loader gets this wrong; the Python port tunes the resolver).
- `1.0.0` stays a string; no timestamp types, no sexagesimal numbers.
- Non-finite floats and non-plain objects are rejected with
  `YAML is not JSON-compatible: <path>` — not passed through, not silently
  coerced.

**Semver.** Strict `major.minor.patch` by regex — `1.0`, `01.0.0`, and
`v1.0.0` are all invalid. Compare numerically, not lexically. Mind the
fallbacks: no version / empty / below-all → **oldest** threshold.

**Error messages.** Ports must throw/return messages **containing** the
substrings in `cases.yaml#errors` — wording around them is free, the
substring is not. Mechanism is idiomatic: exceptions in TS/Python, `error`
returns in Go.

**Expansion contexts.** Component args are expanded in the **caller's**
context before the target joins the active path chain — implement it that
way or the composition fixture's "component inside another component's arg"
case will report a false circular reference. Off-by-one in this chain is
the classic composer bug.

## Finishing

- Add a CLI if the ecosystem expects one: `sdui-compile <root> --out <dir>
  [--pretty]`, writing `screens/<id>/<v>.json` + `manifest.json`
  (manifest always pretty, 2-space indent).
- Mirror the public surface: `ScreenRegistry.build` / `resolve` / `get` /
  `ids` / `versionsOf` / `moduleOf` (idiomatic casing welcome).
- Add the port to the root README table and the test matrix in
  [CONTRIBUTING](../CONTRIBUTING.md), with its one-line test command.
