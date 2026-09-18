"""App-version threshold selection over screen version maps."""

import re
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from .screen_manifest import ScreenModule, ScreenVersionAssets

_SEMVER = re.compile(r"^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$")


class Versioning:
    """Strict major.minor.patch semantic-version resolution."""

    @staticmethod
    def resolve(module: "ScreenModule", app_version: str | None = None) -> "ScreenVersionAssets":
        """Resolves the assets declared by the newest supported app-version threshold.

        @param module - Screen module containing the version map
        @param app_version - Optional client application version
        @returns Template asset version for the threshold
        @throws When the application version is malformed
        """
        versions = sorted(
            (Versioning._parse(version) for version in module.versions),
            key=lambda parsed: parsed[:3],
        )
        if app_version is None or len(app_version) == 0:
            return module.versions[versions[0][3]]

        client = Versioning._parse(app_version)
        compatible = [version for version in versions if version[:3] <= client[:3]]
        threshold = (compatible[-1] if compatible else versions[0])[3]
        return module.versions[threshold]

    @staticmethod
    def compare_semver(left: str, right: str) -> int:
        """Compares two strict major.minor.patch semantic versions.

        @param left - Left semantic version
        @param right - Right semantic version
        @returns A negative, zero, or positive ordering value
        """
        left_parsed = Versioning._parse(left)[:3]
        right_parsed = Versioning._parse(right)[:3]
        return (left_parsed > right_parsed) - (left_parsed < right_parsed)

    @staticmethod
    def _parse(version: str) -> tuple[int, int, int, str]:
        match = _SEMVER.match(version)
        if match is None:
            raise ValueError(f"Invalid semantic version: {version}")
        return (int(match.group(1)), int(match.group(2)), int(match.group(3)), version)
