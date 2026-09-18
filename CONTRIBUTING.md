# Contributing

> 한국어: [CONTRIBUTING.ko.md](CONTRIBUTING.ko.md)

## Repository layout

```text
js-cplr/    TypeScript port — the reference implementation (pnpm, vitest)
py-cplr/    Python port (≥3.11, pytest)
go-cplr/    Go port (≥1.22)
spec/       The contract: SPEC.md (normative) + cases.yaml + fixtures/ (executable)
docs/       Guides: template authoring, integration, porting
```

The ports are deliberately parallel: each has the same module split
(yaml / composer / include-resolver / token-resolver / versioning /
screen-manifest / registry / validator / cli), and each runs the same
conformance suite from `../spec`. There is no shared code between ports —
the spec is the only coupling.

## Setup and tests

Each port is self-contained; run its suite from its own directory:

```sh
(cd js-cplr && pnpm install && pnpm test)        # + pnpm typecheck
(cd py-cplr && python3 -m venv .venv && .venv/bin/pip install -e '.[dev]' \
            && .venv/bin/python -m pytest)
(cd go-cplr && go test ./...)
```

A change is done when **all three** suites are green. There is no CI yet —
run the matrix locally before pushing.

## The change workflow

What you must touch depends on what kind of change it is.

### Port-local changes (no behavior change)

Refactors, performance, docs, idiomatic cleanups inside one port: normal
PR, that port's suite green. If you touched serialization, run the other
suites too — "no behavior change" is exactly what the goldens verify.

### Changing the contract

Any change to what compilers accept, produce, or reject is a **spec
change**, and it travels as one unit through five stops, in order:

1. **`spec/SPEC.md`** — write the rule. If you can't state it normatively
   in a sentence or two, the design isn't done.
2. **`spec/cases.yaml` + `spec/fixtures/`** — add or update the executable
   form: a golden fixture, a versioning query, or an error case with its
   required message substring. Every new normative rule needs a vector that
   fails without the implementation — the suite only guards what it
   covers, so uncovered rules are unguarded rules.
3. **`js-cplr/`** — implement in the reference.
4. **`py-cplr/`, `go-cplr/`** — port it.
5. **All three suites green.**

A PR that changes behavior in one port without the spec and the other
ports is incomplete by definition — the suite will say so.

### Golden files are reviewed, not regenerated

Never bulk-regenerate `expected/` from a port's output and commit the
result — that blesses whatever the port currently does, bugs included.
When goldens must change:

- derive the expected bytes from the **spec**, by hand or by a reviewed
  one-off script;
- review the golden diff with the same care as code — key order, number
  formatting, and etags are all contract;
- expect etag changes whenever content changes; an etag change without a
  content change (or vice versa) is a red flag.

### Fixture hygiene

- Keep fixtures **minimal**: an error fixture demonstrates one failure;
  `composition` exists to exercise the build language, `basic` to look like
  a real app. Don't grow `basic` to cover a corner case — add a focused
  fixture.
- Fixture comments are documentation and are kept accurate like any doc.
- The suites read `../spec` — fixtures assume the sibling layout, so don't
  move `spec/` without updating all three harnesses.

## Documentation

- SSOT boundaries: normative behavior lives in `spec/SPEC.md`; tutorials in
  `docs/`; language-specific commands in the port READMEs; complete
  examples in `spec/fixtures/`. Everything else links instead of restating.
- If your change alters behavior described in `docs/` or a README, update
  it in the same PR.

## Commit / PR expectations

- One logical change per PR; contract changes carry all five stops.
- Say which suites you ran in the PR description.
- New ports are welcome — follow [`docs/porting.md`](docs/porting.md); the
  bar is the conformance suite, not review taste.
