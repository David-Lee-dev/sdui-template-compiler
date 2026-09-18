"""CLI: compiles every screen under an SDUI root into composed JSON files.

Output layout:
```
<outDir>/
  manifest.json                    # per screen: versions map + etags
  screens/<id>/<templateVersion>.json
```
A server (any language) can serve this output statically — the manifest
carries everything needed for app-version selection and etag revalidation.
"""

import os
import sys
from dataclasses import dataclass

from .json_compat import stringify
from .registry import ComposedTemplate, ScreenRegistry


@dataclass(frozen=True)
class _CliOptions:
    root_dir: str
    out_dir: str
    pretty: bool


def main(argv: list[str]) -> int:
    options = _parse_args(argv)
    if options is None:
        sys.stderr.write("Usage: sdui-compile <rootDir> --out <outDir> [--pretty]\n")
        return 1

    registry = ScreenRegistry.build(options.root_dir)
    indent = 2 if options.pretty else None
    manifest: dict[str, object] = {}

    os.makedirs(options.out_dir, exist_ok=True)
    for id in registry.ids():
        module = registry.module_of(id)
        if module is None:
            continue

        screen_dir = os.path.join(options.out_dir, "screens", id)
        os.makedirs(screen_dir, exist_ok=True)

        etags: dict[str, str] = {}
        written: set[str] = set()
        for assets in module.versions.values():
            if assets.template in written:
                continue
            written.add(assets.template)

            composed = _compiled_template(registry, id, assets.template)
            _write_file(
                os.path.join(screen_dir, f"{assets.template}.json"),
                stringify(composed.template, indent),
            )
            etags[assets.template] = composed.etag

        manifest[id] = {
            "versions": {
                version: {"template": assets.template}
                for version, assets in module.versions.items()
            },
            "params": list(module.params),
            "etags": etags,
        }

    _write_file(os.path.join(options.out_dir, "manifest.json"), stringify(manifest, 2))
    sys.stdout.write(f"Compiled {len(registry.ids())} screen(s) to {options.out_dir}\n")
    return 0


def _compiled_template(
    registry: ScreenRegistry, id: str, template_version: str
) -> ComposedTemplate:
    module = registry.module_of(id)
    if module is None:
        raise ValueError(f"Unknown screen: {id}")

    # Resolve via the app-version threshold that maps to this template version.
    threshold = next(
        (
            version
            for version, assets in module.versions.items()
            if assets.template == template_version
        ),
        None,
    )
    if threshold is None:
        raise ValueError(f"Screen <{id}> has no threshold for {template_version}")
    composed = registry.resolve(id, threshold)
    if composed is None:
        raise ValueError(f"Screen <{id}> failed to resolve {template_version}")
    return composed


def _write_file(path: str, content: str) -> None:
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(content)


def _parse_args(argv: list[str]) -> _CliOptions | None:
    positional: list[str] = []
    out_dir: str | None = None
    pretty = False

    i = 0
    while i < len(argv):
        arg = argv[i]
        if arg in ("--out", "-o"):
            out_dir = argv[i + 1] if i + 1 < len(argv) else None
            i += 1
        elif arg == "--pretty":
            pretty = True
        else:
            positional.append(arg)
        i += 1

    if len(positional) != 1 or out_dir is None:
        return None
    return _CliOptions(root_dir=positional[0], out_dir=out_dir, pretty=pretty)


def run() -> None:
    sys.exit(main(sys.argv[1:]))


if __name__ == "__main__":
    run()
