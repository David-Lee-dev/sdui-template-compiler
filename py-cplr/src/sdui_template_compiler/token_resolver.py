"""Design-token group resolution from the `_tokens` directory."""

import os
from typing import Any, Callable

from .yaml_loader import Yaml

JsonObject = dict[str, Any]
TokenResolverFn = Callable[[str], JsonObject]


class TokenResolver:
    """Factory for cached disk-backed token-group resolvers."""

    @staticmethod
    def create(root_dir: str) -> TokenResolverFn:
        """Creates a cached disk-backed resolver for design-token groups.

        @param root_dir - SDUI root containing the `_tokens` directory
        @returns Resolver that loads each token group map at most once
        @throws When a group file is missing or does not contain a map
        """
        cache: dict[str, JsonObject] = {}

        def resolve_group(group: str) -> JsonObject:
            cached = cache.get(group)
            if cached is not None:
                return cached

            file_path = os.path.abspath(os.path.join(root_dir, "_tokens", f"{group}.yaml"))
            if not os.path.exists(file_path):
                raise ValueError(f"Unknown token group {group}: {file_path}")

            tokens = Yaml.load(file_path)
            TokenResolver._assert_token_map(tokens, group, file_path)
            cache[group] = tokens
            return tokens

        return resolve_group

    @staticmethod
    def _assert_token_map(value: Any, group: str, file_path: str) -> None:
        if isinstance(value, dict):
            return
        raise ValueError(f"Token group {group} must contain a map: {file_path}")
