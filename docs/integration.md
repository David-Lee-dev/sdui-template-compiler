# Integration Guide

> 한국어: [integration.ko.md](integration.ko.md)

How to run the compiler in a build pipeline or a server, and how to serve
its output to clients. Everything here is language-neutral — the three
ports behave identically ([`spec/SPEC.md`](../spec/SPEC.md)); installation
and idiomatic API usage are in the per-port READMEs
([TS](../js-cplr/README.md) · [Python](../py-cplr/README.md) ·
[Go](../go-cplr/README.md)).

## Two operating modes

**Precompile (CLI)** — run `sdui-compile` at build/deploy time, publish the
output directory, serve it statically:

```sh
sdui-compile <sdui-root> --out <dir> [--pretty]
```

Best when templates change on deploys, not at runtime: the artifact is plain
files, servable by anything (nginx, S3+CDN, an asset route in your app), and
the whole compile is verified before anything ships — a broken template
fails the build, not a user request.

**In-process (library)** — build a `ScreenRegistry` from the SDUI root at
server start and resolve per request:

```text
registry = ScreenRegistry.build(rootDir)     # compiles every screen; throws on any error
composed = registry.resolve(id, appVersion)  # -> { template, etag } or absent
```

Best when the server owns the templates (monolith, template hot-reload dev
loops, or when you want to serve straight from source). Registry
construction compiles **everything eagerly** — a broken template fails at
boot, and per-request resolution is a lookup, not a compile.

Either way, compilation is deterministic: same input bytes, same output
bytes, same etags — on any port.

## The artifact

```text
<out>/
  manifest.json
  screens/<id>/<templateVersion>.json    # one per distinct template version
```

`manifest.json` (always pretty-printed, 2-space indent):

```json
{
  "home": {
    "versions": { "1.0.0": { "template": "1.0.0" },
                  "2.3.0": { "template": "2.0.0" } },
    "params": ["id"],
    "etags":  { "1.0.0": "be22f863ad4e95e0",
                "2.0.0": "0f3c5a9d1e7b2c44" }
  }
}
```

- `versions` — app-version thresholds → asset versions. This is the routing
  table for version selection (below).
- `params` — the screen's declared route/query parameter names. The
  compiler passes them through; clients (e.g. the Flutter engine) use them
  to forward query values into the screen's root data.
- `etags` — template version → sha256[:16] content hash of the **compact**
  serialization. `--pretty` changes the file bytes but never the etag: the
  etag identifies content, not encoding. Consequence for HTTP: a strong
  `ETag` header must vary when bytes vary, so **serve the compact output**
  if you use the manifest etag as a strong validator. If you must serve
  pretty JSON, either send the manifest hash as a weak validator
  (`W/"<hash>"`) or compute your validator from the served bytes.

## Version selection

Given a screen and a client app version, the template to serve is decided
by one rule — implemented as `resolve(id, appVersion)` in every port, and
trivially re-implementable from the manifest if you serve statically:

1. Collect the screen's threshold keys, sorted by semver.
2. Pick the **newest threshold ≤ the client version**.
3. No client version, empty string, or below every threshold → the
   **oldest** threshold.
4. Malformed client version (`abc`, `1.0`, `01.0.0`) →
   `Invalid semantic version: <v>` — decide at your edge whether that maps
   to a 400 or falls back to no-version behavior, and do it before calling
   the compiler.

Unknown screen id → the registry returns nothing (`undefined` in TS,
`None` in Python, `nil` result with a `nil` error in Go): map it to your
404.

## A transport-neutral serving algorithm

The compiler does not mandate a protocol. Any transport that can carry "a
screen id, an optional app version, an optional cached validator" works.
Over HTTP, the natural mapping:

```text
GET /screens/<id>
    x-app-version:  2.3.0            # optional
    If-None-Match:  "be22f863ad4e95e0"   # optional

1. version  = resolve threshold from manifest (rule above)
2. etag     = manifest[id].etags[version.template]
3. if If-None-Match matches etag  ->  304 (no body)
4. else                           ->  200, body = screens/<id>/<template>.json
                                      ETag: "<etag>"
```

Header names are conventions, not contract — `x-app-version` is what the
[starter kit](https://github.com/David-Lee-dev/sdui-flutter-starter-kit)'s
client and example server use, and that repo is the reference for the
consuming side (Flutter `ScreenLoader`, client-side caching, routing). A
GraphQL field or a gRPC method carrying the same three inputs is just as
conformant.

## Cache invalidation

The etag is a **content hash, not the version string** — this is the load-
bearing design decision. Screens are routinely edited in place within one
template version directory; validating caches on the version string would
answer "unchanged" to clients holding stale trees. With content etags:

- Edit `template/1.0.0/**` → recompile → the etag changes → clients
  revalidating with `If-None-Match` get a `200` with fresh content.
- Nothing changed → same bytes, same etag → `304`, no payload.

Consequences for your pipeline:

- Always serve etags from the **current** manifest — never cache the
  manifest longer than the screen files it describes.
- Deploy the output directory **atomically** (symlink flip, versioned
  prefix, container image). Mixing `manifest.json` from one compile with
  screen files from another breaks the etag/content pairing in both
  directions.
- CDNs key by URL, not by response etag — and screen URLs are stable while
  their content changes. So either deploy under content-addressed or
  versioned URL prefixes (then cache forever), or keep TTLs short /
  `must-revalidate` so edits actually reach clients. Give the manifest (or
  whatever endpoint answers version selection) a short TTL either way.

## Deployment checklist

- **Compile in CI.** The compile is the validation pass: circular refs,
  unknown tokens, undeclared slots, malformed versions all fail the build.
  There is no partial success — fix the template, not the pipeline.
- **Treat templates as trusted input.** The compiler reads files and
  resolves references; it does not sandbox authors. Template authorship is
  code authorship — same review, same provenance.
- **Auth, rollout, targeting are yours.** The compiler decides *which
  template version* an app version gets — nothing else. Percentage
  rollouts, user targeting, authentication all live in the serving layer.
- **Data is a separate plane.** Compiled screens are static; dynamic
  content arrives through whatever data mechanism your client engine uses
  (the starter kit pairs screens with a `POST /v3/data` operation
  endpoint). Don't recompile templates to change data.
- **Pin one port per pipeline.** The ports are byte-identical, so mixing
  them is *safe* — but a single-port pipeline keeps toolchain drift out of
  your build cache and your incident timeline.
