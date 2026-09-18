"""Component and fragment reference resolution from disk."""

import os
from dataclasses import dataclass
from typing import Any, Callable

from .yaml_loader import Yaml

JsonValue = Any


@dataclass(frozen=True)
class ResolvedInclude:
    file_path: str
    content: JsonValue


IncludeResolverFn = Callable[[str, str], ResolvedInclude]


class IncludeResolver:
    """Factory for disk-backed `.ref` resolvers."""

    @staticmethod
    def create(root_dir: str) -> IncludeResolverFn:
        """Creates a disk-backed resolver for component and fragment references.

        @param root_dir - SDUI root used as the base for reference paths
        @returns Resolver that returns an absolute path and parsed YAML content
        @throws When a referenced file is missing or is not JSON-compatible
        """

        def resolve_ref(ref_path: str, referrer_dir: str) -> ResolvedInclude:
            if ref_path.startswith("/"):
                file_path = os.path.abspath(
                    os.path.join(root_dir, "_components", f"{ref_path[1:]}.yaml")
                )
            else:
                file_path = os.path.abspath(os.path.join(referrer_dir, f"{ref_path}.yaml"))
            if not os.path.exists(file_path):
                raise ValueError(f"Unresolved .ref {ref_path}: {file_path}")

            return ResolvedInclude(file_path=file_path, content=Yaml.load(file_path))

        return resolve_ref
