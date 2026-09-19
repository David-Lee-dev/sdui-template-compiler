# Template Authoring Guide

> 한국어: [template-guide.ko.md](template-guide.ko.md)

A tutorial for writing SDUI templates that this compiler builds. It walks
from an empty directory to components, tokens, and versioned screens. The
normative contract — exact rules, exact error messages — is
[`spec/SPEC.md`](../spec/SPEC.md); when this guide and the spec disagree,
the spec wins. Complete, tested examples live in
[`spec/fixtures/`](../spec/fixtures) — every snippet here is drawn from or
compatible with them.

## The mental model

Two vocabularies coexist in a template file, split by their first character:

- **Dot-prefixed keys** (`.ref`, `.vars`, `.defaults`, `.content`,
  `.token`) are **build language**. The compiler resolves them all at
  compile time; none survive into the output. Dot-prefixed *strings*
  (`.slotname`) are build language too, but only inside a component's
  `.content` — anywhere else they are ordinary strings. So is the `@{...}`
  token sigil, which may appear inside any string value.
- **Underscore-prefixed keys and `${...}` strings** (`_type`, `_children`,
  `_scope`, `${item.title}`, …) are **runtime language**. The compiler
  passes them through untouched — they mean whatever the client engine that
  renders the JSON says they mean (for the Flutter engine, see the
  [starter kit](https://github.com/David-Lee-dev/sdui-flutter-starter-kit)).

This guide teaches the build language. Runtime keys appear in the examples
only as realistic payload.

One build-language rule reaches into runtime territory: `screen_id` is
rejected inside template nodes (screen identity belongs to `screen.yaml`,
nowhere else).

## 1. The SDUI root

A compilation unit is one directory tree:

```text
<root>/
  _tokens/<group>.yaml            # design-token maps
  _components/<name>.yaml         # shared components and fragments
  screens/<dir>/
    screen.yaml                   # screen manifest: id, versions, params
    template/<version>/_root.yaml # the screen tree for that template version
```

The smallest valid root is one screen:

```yaml
# screens/hello/screen.yaml
id: hello
versions:
  '1.0.0': { template: '1.0.0' }
```

```yaml
# screens/hello/template/1.0.0/_root.yaml
_type: text
value: hello, world
```

Compile it:

```sh
sdui-compile <root> --out build/
# build/manifest.json
# build/screens/hello/1.0.0.json
```

## 2. `screen.yaml` — the screen manifest

```yaml
id: home              # authoritative screen id (non-empty)
versions:             # app-version threshold -> asset versions for that threshold
  '1.0.0': { template: '1.0.0' }
  '2.3.0': { template: '2.0.0' }
params: [id]          # optional: route/query keys forwarded to the client as root data
```

- `id` is the screen's identity everywhere: output paths
  (`screens/<id>/...`), manifest keys, and the id clients request. Duplicate
  ids across the root fail the build, and neither ids nor template versions
  may be purely numeric (`"10"` — JS objects would reorder such manifest
  keys).
- `versions` maps **app-version thresholds** to **template versions**. A
  client on app version `V` receives the assets of the **newest threshold
  ≤ V**. Clients below every threshold — or sending no version — get the
  oldest. So the table above reads: apps `1.0.0`–`2.2.x` get template
  `1.0.0`; apps `2.3.0` and newer get `2.0.0`.
- Threshold keys — and the client versions matched against them — are
  strict `major.minor.patch` (`1.0` and `01.0.0` are invalid). Template
  versions are opaque non-empty strings used as directory and output names;
  semver-style values are the convention, not a requirement. Quote both —
  unquoted `1.0.0` is fine, but consistency avoids YAML surprises.
- `params` is metadata passed through to the manifest; engines use it to
  forward route query values into the screen's root data. The compiler does
  not interpret it.

Template versions are directories: `template/1.0.0/_root.yaml`,
`template/2.0.0/_root.yaml`. Add a threshold entry pointing at a new
template directory to change a screen for newer apps only — older apps keep
compiling and receiving the old tree.

> **Editing in place vs. bumping the version.** You can edit
> `template/1.0.0/**` without touching `screen.yaml` — the output *content
> hash* (etag) changes even though the version string doesn't, so clients
> revalidating by etag still see the update. Bump the template version when
> the change must not reach older apps.

## 3. Design tokens — `.token` and `@{...}`

Token groups are YAML maps under `_tokens/`:

```yaml
# _tokens/color.yaml
primary: '#5B8CFF'
surface:
  card: '#1A1D24'
```

Reference them with a dotted path — group name first, then keys:

```yaml
style:
  color: { .token: color.primary }
background: { .token: color.surface.card }
```

Rules:

- The path has **at least two segments** (`group.key`, deeper is fine).
- A `.token` node must have **no sibling keys** — it *is* the value.
- The looked-up value replaces the node **verbatim** (strings, numbers,
  maps — whatever the token file holds). It is not re-scanned for build
  keys: token files hold plain values, not `.ref`s or further `.token`s.
- Unknown groups, unknown paths, and traversing into a non-map all fail the
  build with the offending path in the message.

### The inline form — `@{group.path}`

`.token` replaces a whole node, so it cannot reach inside a string. The sigil
`@{...}` does the same lookup anywhere a string value appears:

```yaml
color: '@{color.primary}'            # same as { .token: color.primary }
padding: '@{spacing.lg}'
```

Alone in a string, it **keeps the token's type** — `'@{spacing.lg}'` composes to
the number `16`, not `"16"`. Mixed with other text it **interpolates**, and the
result is a string:

```yaml
label: 'pad:@{spacing.lg}'           # -> "pad:16"
```

Because the compiler consumes the sigil, it can sit inside a runtime `${...}`
expression — which is the one thing `.token` cannot do:

```yaml
text_color: '${is_urgent ? "@{color.badge}" : "@{color.text_secondary}"}'
width: '${@{spacing.lg} * 2}'
```

Think of the two delimiters as two phases: `@{}` is resolved by the compiler and
is gone from the output; `${}` survives into the JSON and is evaluated by the
client at runtime.

Rules:

- Only **string values** are scanned. Map keys and build-key arguments (a `.ref`
  path, a `.token` value) are left alone.
- `@@{` escapes a literal `@{`.
- Interpolating a map or array fails the build
  (`Cannot interpolate non-scalar token`) — there is no sensible text for it.
  Alone in a string it is fine, since the value replaces the node wholesale.
- An unpaired `@{` fails the build (`Unterminated token sigil`) rather than
  shipping a typo to the client as literal text.
- A token whose own **value** contains `@{` is rejected. Otherwise the same
  token would resolve differently depending on whether it landed in a component
  argument subtree (which gets walked twice).
- Both forms are equivalent and may be mixed freely; `.token` is not deprecated.

## 4. Components and fragments — `.ref`

`.ref` includes another file. Two path forms:

- `/name` — absolute: resolves to `<root>/_components/name.yaml`
  (subdirectories work: `/nested/wrap`).
- `name` or `./name`, `../x` — relative to the **referring file's**
  directory. Handy for screen-private fragments next to `_root.yaml`.

A `.ref` node must not declare `_type` — it is replaced entirely by what it
includes.

### Fragments — inline inclusion

A component file **without `.vars`** is a fragment: its content is expanded
in place, and it accepts no arguments (passing any is a build error).

```yaml
# screens/home/template/1.0.0/_state.yaml
items: null
```

```yaml
# _root.yaml
_scope:
  _state: { .ref: _state }     # relative ref to the sibling file
```

Fragments are how big screens stay readable: split state, actions, and
sections into sibling files and `.ref` them from `_root.yaml`.

### Components — declared parameters

A file **with `.vars`** is a parameterized component:

```yaml
# _components/badge.yaml
.vars: [label, color, icon]          # every accepted arg, declared up front
.defaults:
  color: { .token: color.primary }   # used when the caller omits the arg
.content:                            # the tree that replaces the .ref node
  _type: row
  _children:
    - .icon                          # slot: substituted with the `icon` arg
    - _type: text
      value: .label
      style: { color: .color }
```

Call it with args as sibling keys of `.ref`:

```yaml
- { .ref: /badge, label: hello }                     # default color, no icon
- { .ref: /badge, label: alert, color: '#FF0000',
    icon: { _type: icon, name: star } }
```

Rules:

- `.vars` lists **unique, non-empty** names; `.content` is required.
- Every caller arg must be declared in `.vars`, and every `.defaults` key
  must be too — unknown names fail the build.

### Slots — where args land

Inside `.content`, the string `.name` substitutes the arg `name`:

- **Provided** (by the caller or `.defaults`) → the value replaces the slot,
  whatever its type.
- **Omitted** (no arg, no default) → the slot is **dropped**: a map entry
  whose value was the slot disappears; a list element disappears. That is
  why the first `badge` call above renders without an icon — no `null`
  placeholder, the row just has one child fewer.
- **List-valued arg in an array position** → **spliced** flat into the
  surrounding list, not nested:

  ```yaml
  # _components/list.yaml
  .vars: [items]
  .content:
    _type: column
    _children:
      - { _type: text, value: header }
      - .items                       # a list arg splices: header, a, b
  ```

- Slots must be declared: a `.something` string in `.content` that is not in
  `.vars` fails the build (`undeclared slot`). Strings starting with `./` or
  `..` are exempt — they stay ordinary strings (and only act as paths when
  used as a `.ref` value).

### Composition and recursion

Components nest freely — args are themselves trees that may contain `.ref`,
`.token`, or further components. Argument subtrees are expanded in the
**caller's** context, so passing a component to itself as an arg
(`{ .ref: /wrap, child: { .ref: /badge, ... } }` inside another `/badge`
instance's arg) is not a false positive. Genuine self-recursion is detected
and fails: `Circular reference: a -> b -> a`.

When a `.ref` sitting in an array expands to an array, it is spliced one
level flat — a fragment holding a list of rows drops into `_children`
seamlessly.

## 5. The YAML dialect

Templates are YAML 1.2 **core schema** (as implemented by js-yaml v4), and
the output must be JSON-compatible:

- Only `true`/`false` are booleans — `on`, `off`, `yes`, `no` are **strings**.
- `1.0.0` stays a string; no timestamp or sexagesimal parsing.
- Non-finite numbers (`.inf`, `.nan`) and non-plain objects are rejected:
  `YAML is not JSON-compatible: <path>`.
- **Key order is significant**: it is preserved through composition into the
  serialized output byte-for-byte (and therefore into the etag).

## 6. Output

Per screen, per distinct template version:

- `screens/<id>/<templateVersion>.json` — the composed tree, all build keys
  resolved, compact serialization.
- `manifest.json` — per screen: the `versions` map, `params`, and `etags`
  (template version → sha256[:16] of the compact JSON).

`--pretty` makes the screen JSON human-readable; etags are always computed
from the compact form, so pretty-printing never changes them. What to do
with these files — serving, caching, deployment — is
[`docs/integration.md`](integration.md).

## 7. Errors you will meet

The conformance suite fixes these diagnostic **substrings** — every port's
message contains them (surrounding wording, error types, and paths are the
port's own). The authoritative list is the `errors` map in
[`spec/cases.yaml`](../spec/cases.yaml):

| Message contains | You wrote |
|---|---|
| `Circular reference` | a component chain that includes itself (the message names the path, e.g. `a -> b -> a`) |
| `Unknown token group` | a `.token` referencing a group file that doesn't exist |
| `unknown args` | a caller arg not declared in `.vars` |
| `undeclared slot` | a `.name` string in `.content` missing from `.vars` |
| `does not accept args` | args passed to a fragment (no `.vars`) |
| `Unresolved .ref` | a `.ref` path that resolves to no file |
| `must not have sibling keys` | a `.token` node with siblings |
| `Cannot interpolate non-scalar token` | `@{...}` for a map/array mixed into text |
| `Unterminated token sigil` | an unpaired `@{` in a string |
| `Token value must not contain @{` | a token file whose value carries the sigil |
| `Unsupported build key` | a dot-key that isn't in the build language |
| `Screen id must not be purely numeric` | an all-digit `id` in `screen.yaml` |
| `Template version must not be purely numeric` | an all-digit template version |

Beyond these, each port reports further spec-mandated failures in the same
spirit — `Invalid semantic version: <v>` for malformed versions,
`YAML is not JSON-compatible: <path>` for exotic YAML, token-path traversal
errors naming the failing path — but only the substrings above are
conformance-locked today.

## Worked examples

- [`spec/fixtures/basic/`](../spec/fixtures/basic) — a realistic two-screen
  app: tokens, a `card` component, screen-private fragments (`_state`,
  `_action`), `params` forwarding. Note its `_root.yaml` is dense with
  runtime vocabulary (`_scope`, `_loop`, `${...}`) — engine language the
  compiler passes through.
- [`spec/fixtures/composition/`](../spec/fixtures/composition) — the build
  language end to end: defaults, omitted slots, nested components inside
  args, list splicing, relative fragments.
- [`spec/fixtures/versioning/`](../spec/fixtures/versioning) — one screen,
  three template versions, threshold selection (including an
  arbitrary-precision threshold beyond 2^53).
- [`spec/fixtures/numbers/`](../spec/fixtures/numbers) — number
  serialization edge cases (exponent notation, integral floats).
- [`spec/fixtures/errors/`](../spec/fixtures/errors) — one minimal broken
  root per diagnostic.

Each fixture's `expected/` directory is the compiler's exact output — the
fastest way to answer "what JSON does this YAML become?".
