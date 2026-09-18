"""Screen manifest discovery and validation (`screens/<dir>/screen.yaml`)."""

import os
import re
from dataclasses import dataclass
from typing import Mapping, Sequence

from .yaml_loader import Yaml

_SEMANTIC_VERSION = re.compile(r"^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$")
_MANIFEST_FILE = "screen.yaml"


@dataclass(frozen=True)
class ScreenVersionAssets:
    template: str


@dataclass(frozen=True)
class ScreenModule:
    """A screen declaration loaded from `screens/<dir>/screen.yaml`.

    The manifest is a plain YAML file — not code — so every compiler port
    (TS, Python, Go) reads the exact same declaration.
    """

    id: str
    dir: str
    versions: Mapping[str, ScreenVersionAssets]
    params: Sequence[str]


class ScreenManifest:
    """Loads and validates screen manifest declarations."""

    @staticmethod
    def discover(root_dir: str) -> list[ScreenModule]:
        """Discovers every screen manifest under `<rootDir>/screens`.

        @param root_dir - SDUI root containing the `screens` directory
        @returns Validated screen modules in directory order
        @throws When a manifest is invalid or two screens declare the same id
        """
        screens_dir = os.path.join(root_dir, "screens")
        if not os.path.exists(screens_dir):
            raise ValueError(f"Missing screens directory: {screens_dir}")

        modules: list[ScreenModule] = []
        seen: set[str] = set()
        for entry in os.scandir(screens_dir):
            if not entry.is_dir():
                continue
            manifest_path = os.path.join(screens_dir, entry.name, _MANIFEST_FILE)
            if not os.path.exists(manifest_path):
                continue

            module = ScreenManifest.load(manifest_path, os.path.join(screens_dir, entry.name))
            if module.id in seen:
                raise ValueError(f"Duplicate screen id: {module.id}")
            seen.add(module.id)
            modules.append(module)
        return modules

    @staticmethod
    def load(manifest_path: str, dir: str) -> ScreenModule:
        """Loads and validates a single screen manifest file.

        @param manifest_path - Path to the `screen.yaml` file
        @param dir - Screen directory owning the manifest
        @returns The validated screen module
        @throws When the id, version map, or a version threshold is invalid
        """
        raw = Yaml.load(manifest_path)
        if not isinstance(raw, dict):
            raise ValueError(f"Screen manifest must be a map: {manifest_path}")

        id = raw.get("id")
        if not isinstance(id, str) or len(id.strip()) == 0:
            raise ValueError(f"Screen manifest must declare an id: {manifest_path}")

        versions_raw = raw.get("versions")
        if not isinstance(versions_raw, dict):
            raise ValueError(f"Screen <{id}> must declare a versions map")
        if len(versions_raw) == 0:
            raise ValueError(f"Screen <{id}> must declare at least one version")

        versions: dict[str, ScreenVersionAssets] = {}
        for version, assets in versions_raw.items():
            if _SEMANTIC_VERSION.match(version) is None:
                raise ValueError(f"Invalid semantic version: {version}")
            if not isinstance(assets, dict):
                raise ValueError(f"Screen <{id}> version {version} must be a map")
            template = assets.get("template")
            if not isinstance(template, str) or len(template) == 0:
                raise ValueError(f"Screen <{id}> version {version} must declare a template")
            versions[version] = ScreenVersionAssets(template=template)

        params_raw = raw.get("params")
        if params_raw is None:
            params_raw = []
        if not isinstance(params_raw, list) or any(
            not isinstance(item, str) for item in params_raw
        ):
            raise ValueError(f"Screen <{id}> params must be a list of names")

        return ScreenModule(id=id, dir=dir, versions=versions, params=tuple(params_raw))
