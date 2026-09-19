# SDUI Template Compiler — Specification

Every compiler port (TS, Python, Go) implements exactly this. The conformance
vectors in `cases.yaml` + `fixtures/` are the executable form of this document —
the drift guard every port must keep green. The suite exercises the contract
through representative vectors; this document is what a port implements.

## Input: an SDUI root directory

```
<root>/
  _tokens/<group>.yaml          # design-token maps, referenced as { .token: <g>.<path> } or @{<g>.<path>}
  _components/<name>.yaml       # shared components/fragments, referenced as { .ref: /<name> }
  screens/<dir>/
    screen.yaml                 # manifest: id, versions, params
    template/<version>/_root.yaml   # screen root (may .ref sibling fragments relatively)
```

### screen.yaml

```yaml
id: home                        # authoritative screen id (non-empty)
versions:                       # app-version threshold -> asset versions
  '1.0.0': { template: '1.0.0' }
  '2.3.0': { template: '2.0.0' }
params: [id]                    # optional: route query keys forwarded as root data
```

Threshold keys — and the client app versions matched against them — are
strict `major.minor.patch` (regex `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`).
Components compare **numerically at arbitrary precision** (no leading zeros,
so digit-string length-then-lexicographic order is exact). Template asset
versions are opaque non-empty strings, used verbatim as the `template/<v>/`
directory and `<v>.json` output names. Screen ids and template versions must
not be **purely numeric** (`^[0-9]+$`) — JS objects reorder integer-index keys,
which would break cross-port manifest key order.

Screens are discovered from `screens/*/screen.yaml` in **Unicode code-point
order of the directory name** (equals UTF-8 byte order) — registry id order
and manifest key order follow it.

## Build keys (compile-time only; none survive into output)

- `{ .ref: <path>, <args...> }` — include. `/x` resolves to `<root>/_components/x.yaml`;
  a relative path resolves against the referring file's directory. A `.ref` node
  must not declare `_type`.
- Component file: `{ .vars: [names], .defaults: {name: value}, .content: <node> }`.
  Args must all be declared in `.vars`; `.defaults` keys must be in `.vars`;
  `.vars` must be unique non-empty names; `.content` is required when `.vars` exists.
- A file without `.vars` is a **fragment**: expanded in place, accepts no args.
- Slot: the string `.name` inside `.content` substitutes the arg `name`.
  Omitted (no arg, no default) slots are dropped; in arrays, a list-valued arg
  is spliced in place. Slots must be declared (`undeclared slot` otherwise).
  Strings starting with `./` or `..` are not slots.
- Component args are expanded in the **caller's** context before the target file
  joins the active path chain (so reusing a component inside another instance's
  arg subtree is not a false circular reference). Genuine self-recursion throws
  `Circular reference: a -> b -> a`.
- `{ .token: group.path.to.value }` — design-token lookup. The node must have no
  sibling keys; the value is a dotted string of at least two segments; traversal
  errors identify the failing path. Token group files must be maps; groups are
  cached per build.
- `@{group.path.to.value}` — the same lookup, inline in any **string value**
  (map keys and build-key arguments are not scanned). A string that is exactly
  one sigil resolves to the token value and **keeps its type**
  (`'@{spacing.lg}'` → `16`); any other occurrence is **interpolated as text**,
  each token rendered with its JSON scalar form (`16`, `1.5`, `true`, `null`),
  producing a string. Because substitution happens at compile time, a sigil may
  sit inside a runtime `${...}` expression — the client never sees either sigil.
  `@@{` escapes a literal `@{`. Interpolating a map or array throws
  `Cannot interpolate non-scalar token`; an unpaired `@{` throws
  `Unterminated token sigil`; a **token value** that itself contains `@{` throws
  `Token value must not contain @{` (it would otherwise be rescanned when the
  value lands in a component-argument subtree, making the result depend on where
  the token was used).
- Any other `.`-prefixed key throws `Unsupported build key .x`.
- `screen_id` is rejected inside template nodes.
- When a `.ref` in an array expands to an array, it is spliced (flattened one level).

## Versioning

`resolve(module, appVersion)` picks the assets of the **newest threshold ≤ the
client version**; no version / empty string → the oldest threshold; below every
threshold → the oldest; malformed versions throw `Invalid semantic version: <v>`.

## Output

Per screen, per distinct template version:
- Composed JSON with all build keys resolved, **key order preserved from the
  YAML source**, serialized compact (`JSON.stringify` semantics: UTF-8, no added
  whitespace, numbers per ECMAScript `Number::toString` — integral values
  without a trailing `.0`, exponent notation only for `|x| >= 1e21` or
  `|x| < 1e-6`, bare exponent digits: `1e-7`, never `1e-07`).
- `etag` = first 16 hex chars of sha256 over that compact serialization.

CLI (`sdui-compile <root> --out <dir> [--pretty]`) writes
`screens/<id>/<templateVersion>.json` plus `manifest.json`
(`{ <id>: { versions, params, etags } }`, pretty-printed with 2-space indent).

## YAML dialect

YAML 1.2 core schema as implemented by js-yaml v4: only `true`/`false` are
booleans (`on`/`off`/`yes`/`no` are strings), `1.0.0` stays a string, non-finite
numbers and non-plain objects are rejected (`YAML is not JSON-compatible: <path>`).
Integers beyond ±(2^53 − 1) are held as lossy doubles, exactly as JS holds
every number — `9007199254740993` composes and serializes as
`9007199254740992`.

## Error messages

Ports must throw messages **containing** the substrings listed in
`cases.yaml#errors` so operators get identical diagnostics in every language.
