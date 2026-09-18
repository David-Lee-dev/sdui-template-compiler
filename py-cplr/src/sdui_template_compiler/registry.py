"""Screen registry: composes and caches every template under an SDUI root."""

import functools
import hashlib
import os
from dataclasses import dataclass
from typing import Any, Mapping, Sequence

from .composer import Composer
from .include_resolver import IncludeResolver
from .json_compat import stringify
from .screen_manifest import ScreenManifest, ScreenModule
from .token_resolver import TokenResolver
from .versioning import Versioning
from .yaml_loader import Yaml

JsonValue = Any

# ETag length. Two revisions of one screen colliding does not happen in practice.
_ETAG_LENGTH = 16


@dataclass(frozen=True)
class ComposedTemplate:
    """One composed template and its content validator.

    `etag` is a content hash of the composed output — not the declared version
    string. Screens are routinely edited in place inside one version directory
    (`template/1.0.0/**`), which leaves the version unchanged; validating a
    cache on the version would answer "unchanged" to a client holding a stale
    template.
    """

    template: JsonValue
    etag: str


@dataclass(frozen=True)
class _RegisteredScreen:
    module: ScreenModule
    templates: Mapping[str, ComposedTemplate]


class ScreenRegistry:
    """Serves composed templates by screen id and app version."""

    def __init__(self, screens: Mapping[str, _RegisteredScreen]) -> None:
        self._screens = screens

    @staticmethod
    def build(
        root_dir: str,
        modules: Sequence[ScreenModule] | None = None,
    ) -> "ScreenRegistry":
        """Composes and caches every template asset declared under an SDUI root.

        Screens are discovered from `screens/<dir>/screen.yaml` manifests;
        components resolve from `_components/` and tokens from `_tokens/`.

        @param root_dir - SDUI root containing screens, components, and tokens
        @param modules - Explicit modules override used by controlled test fixtures
        @returns A registry serving composed templates by id and app version
        @throws When ids are duplicated or composition fails
        """
        discovered = modules if modules is not None else ScreenManifest.discover(root_dir)
        include_resolver = IncludeResolver.create(root_dir)
        token_resolver = TokenResolver.create(root_dir)
        screens: dict[str, _RegisteredScreen] = {}

        for module in discovered:
            if module.id in screens:
                raise ValueError(f"Duplicate screen id: {module.id}")

            templates: dict[str, ComposedTemplate] = {}
            template_versions = {entry.template for entry in module.versions.values()}
            for template_version in template_versions:
                root_path = os.path.join(module.dir, "template", template_version, "_root.yaml")
                template = Composer.expand(
                    Yaml.load(root_path),
                    include_resolver,
                    token_resolver,
                    root_dir,
                    os.path.dirname(root_path),
                )
                templates[template_version] = ComposedTemplate(
                    template=template,
                    etag=ScreenRegistry._etag_of(template),
                )

            screens[module.id] = _RegisteredScreen(module=module, templates=templates)

        return ScreenRegistry(screens)

    def get(self, id: str, app_version: str | None = None) -> JsonValue | None:
        """Returns the composed screen selected for an application version.

        @param id - Authoritative declared screen id
        @param app_version - Optional client application version
        @returns Composed template, or None when the id is unknown
        @throws When the application version is malformed
        """
        composed = self.resolve(id, app_version)
        return None if composed is None else composed.template

    def resolve(self, id: str, app_version: str | None = None) -> ComposedTemplate | None:
        """Returns the composed screen for an application version with its etag.

        A serving layer needs both — the etag to answer a conditional GET, the
        template to send when the client's copy is stale. [get] stays as the
        template-only view for callers that never revalidate.

        @param id - Authoritative declared screen id
        @param app_version - Optional client application version
        @returns Composed template and its content etag, or None when the id is unknown
        @throws When the application version is malformed
        """
        screen = self._screens.get(id)
        if screen is None:
            return None

        assets = Versioning.resolve(screen.module, app_version)
        return screen.templates.get(assets.template)

    def ids(self) -> list[str]:
        """Lists every registered screen id."""
        return list(self._screens.keys())

    def versions_of(self, id: str) -> list[str]:
        """Lists app-version thresholds declared for a screen id.

        @param id - Authoritative declared screen id
        @returns Version-map threshold keys in ascending semantic order
        """
        screen = self._screens.get(id)
        if screen is None:
            return []

        return sorted(
            screen.module.versions.keys(),
            key=functools.cmp_to_key(Versioning.compare_semver),
        )

    def module_of(self, id: str) -> ScreenModule | None:
        """Returns the discovered module declaration for a screen id."""
        screen = self._screens.get(id)
        return None if screen is None else screen.module

    @staticmethod
    def _etag_of(template: JsonValue) -> str:
        """Hashes a composed template into its cache validator.

        Runs once per template at build time; serving a request then costs one
        string comparison.

        @param template - Composed template to fingerprint
        @returns Hex etag of _ETAG_LENGTH characters
        """
        digest = hashlib.sha256(stringify(template).encode("utf-8")).hexdigest()
        return digest[:_ETAG_LENGTH]
