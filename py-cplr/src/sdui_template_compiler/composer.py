"""Template expansion: references, components, tokens, and variable slots."""

import copy
import os
import re
from dataclasses import dataclass, replace
from typing import Any, Mapping, Sequence

from .include_resolver import IncludeResolverFn
from .json_compat import stringify
from .token_resolver import TokenResolverFn
from .validator import Validator

JsonValue = Any
JsonObject = dict[str, Any]

REF_KEY = ".ref"
VARS_KEY = ".vars"
DEFAULTS_KEY = ".defaults"
CONTENT_KEY = ".content"
TOKEN_KEY = ".token"

_MISSING = object()
_GROUP_NAME = re.compile(r"^[A-Za-z0-9_-]+$")

# `@{group.path}` — the inline form of `.token`, usable anywhere a string is.
# `@@{` escapes a literal `@{`. Both are consumed at compile time, so the sigil
# is invisible to the client and composes inside runtime `${...}` expressions.
_TOKEN_SIGIL = re.compile(r"@@\{|@\{([^{}]*)\}")


@dataclass(frozen=True)
class ComponentFile:
    vars: Sequence[str]
    defaults: Mapping[str, JsonValue] | None
    content: JsonValue


@dataclass(frozen=True)
class _ExpansionContext:
    active_paths: tuple[str, ...]
    component_stack: tuple[str, ...]
    include_resolver: IncludeResolverFn
    include_stack: tuple[str, ...]
    referrer_dir: str
    root_dir: str
    token_resolver: TokenResolverFn


class Composer:
    """Expands parsed screen templates into client-owned JSON."""

    @staticmethod
    def expand(
        root_body: JsonValue,
        include_resolver: IncludeResolverFn,
        token_resolver: TokenResolverFn,
        root_dir: str,
        referrer_dir: str,
    ) -> JsonValue:
        """Expands a parsed screen body into client-owned JSON.

        @param root_body - Parsed screen template node tree
        @param include_resolver - Injected path resolver and YAML loader
        @param token_resolver - Injected design-token group resolver
        @param root_dir - SDUI root used to normalize resolved include paths
        @param referrer_dir - Directory containing the screen root file
        @returns Fully expanded JSON without build-time keys
        @throws When references, component interfaces, or reserved fields are invalid
        """
        return Composer._expand_node(
            root_body,
            _ExpansionContext(
                active_paths=(),
                component_stack=(),
                include_resolver=include_resolver,
                include_stack=(),
                referrer_dir=referrer_dir,
                root_dir=os.path.abspath(root_dir),
                token_resolver=token_resolver,
            ),
        )

    @staticmethod
    def substitute_vars(
        node: JsonValue,
        vars: Sequence[str],
        args: Mapping[str, JsonValue],
    ) -> JsonValue | None:
        """Substitutes exact component variable values and drops omitted slots.

        @param node - Component body node to transform
        @param vars - Variable names declared by the component
        @param args - Values supplied by the component reference
        @returns Substituted JSON, or None when the root slot is omitted
        @throws When the body contains an undeclared variable slot
        """
        substituted = Composer._substitute_node(node, vars, args)
        return None if substituted is _MISSING else substituted

    @staticmethod
    def splice_slots(
        nodes: Sequence[JsonValue],
        vars: Sequence[str],
        args: Mapping[str, JsonValue],
    ) -> list[JsonValue]:
        """Substitutes array slots while splicing list-valued arguments in place.

        @param nodes - Component body array containing potential variable slots
        @param vars - Variable names declared by the component
        @param args - Values supplied by the component reference
        @returns Substituted array with omitted slots removed
        @throws When the body contains an undeclared variable slot
        """
        result: list[JsonValue] = []

        for node in nodes:
            slot_name = Composer._slot_name(node)
            if slot_name is not None:
                Validator.assert_hole_declared(vars, slot_name)
                if slot_name not in args:
                    continue

                value = copy.deepcopy(args[slot_name])
                if isinstance(value, list):
                    result.extend(value)
                else:
                    result.append(value)
                continue

            substituted = Composer._substitute_node(node, vars, args)
            if substituted is not _MISSING:
                result.append(substituted)

        return result

    @staticmethod
    def _expand_node(node: JsonValue, context: _ExpansionContext) -> JsonValue:
        if isinstance(node, list):
            return Composer._expand_array(node, context)
        if isinstance(node, str):
            return Composer._expand_string(node, context)
        if not isinstance(node, dict):
            return node

        if "screen_id" in node:
            raise ValueError("screen_id is not allowed in template nodes")
        if TOKEN_KEY in node:
            return Composer._expand_token(node, context)
        if REF_KEY in node:
            return Composer._expand_reference(node, context)

        result: JsonObject = {}
        for key, value in node.items():
            if key.startswith("."):
                raise ValueError(f"Unsupported build key {key}")
            result[key] = Composer._expand_node(value, context)
        return result

    @staticmethod
    def _expand_token(node: JsonObject, context: _ExpansionContext) -> JsonValue:
        if len(node) != 1:
            raise ValueError(".token node must not have sibling keys")

        token_path = node[TOKEN_KEY]
        if not isinstance(token_path, str):
            raise ValueError(".token value must be a non-empty dotted string")

        return Composer._resolve_token(token_path, context)

    @staticmethod
    def _resolve_token(token_path: str, context: _ExpansionContext) -> JsonValue:
        """Resolves a dotted token path against its group file.

        Shared by `{ .token: ... }` and the inline `@{...}` sigil so both forms
        validate identically and fail with the same diagnostics.

        @param token_path - Dotted path, `<group>.<key...>`
        @param context - Expansion context carrying the token resolver
        @returns Deep copy of the token value
        @throws When the path is malformed, the group is unknown, or a key is missing
        """
        segments = token_path.split(".")
        if (
            len(segments) < 2
            or any(len(segment) == 0 or segment.strip() != segment for segment in segments)
            or _GROUP_NAME.match(segments[0]) is None
        ):
            raise ValueError(".token value must be a non-empty dotted string")

        group, *keys = segments
        value: JsonValue = context.token_resolver(group)
        traversed_path = group

        for key in keys:
            if not isinstance(value, dict):
                raise ValueError(
                    f"Cannot traverse non-object token path {traversed_path} "
                    f"while resolving {token_path}"
                )
            if key not in value:
                raise ValueError(f"Missing token key {token_path} at {traversed_path}.{key}")

            value = value[key]
            traversed_path = f"{traversed_path}.{key}"

        # A token value that carries the sigil would be rescanned when it lands in
        # a component argument subtree, making resolution depend on where it was
        # used. Reject it at the source instead.
        if isinstance(value, str) and "@{" in value:
            raise ValueError(f"Token value must not contain @{{: {token_path}")

        return copy.deepcopy(value)

    @staticmethod
    def _expand_string(node: str, context: _ExpansionContext) -> JsonValue:
        """Substitutes `@{group.path}` token sigils inside a string value.

        A string that is exactly one sigil resolves to the token's own value and
        keeps its type (`'@{spacing.lg}'` -> `16`). Any other occurrence is
        interpolated as text, so tokens compose inside runtime expressions.

        @param node - Raw string value from the template
        @param context - Expansion context carrying the token resolver
        @returns The token value, the interpolated string, or the string unchanged
        @throws When a sigil is unterminated or interpolates a non-scalar token
        """
        if "@" not in node:
            return node

        whole = _TOKEN_SIGIL.fullmatch(node)
        if whole is not None and whole.group(1) is not None:
            return Composer._resolve_token(whole.group(1), context)

        def substitute(match: re.Match[str]) -> str:
            if match.group(1) is None:
                return "@{"

            value = Composer._resolve_token(match.group(1), context)
            if isinstance(value, (dict, list)):
                raise ValueError(
                    f"Cannot interpolate non-scalar token {match.group(1)} into a string"
                )
            return value if isinstance(value, str) else stringify(value)

        Composer._assert_no_dangling_sigil(node)
        return _TOKEN_SIGIL.sub(substitute, node)

    @staticmethod
    def _assert_no_dangling_sigil(original: str) -> None:
        """Rejects an unterminated `@{` left behind by substitution.

        A typo like `@{color.primary` would otherwise ship to the client as
        literal text; failing the build is the loud alternative.
        """
        if "@{" in _TOKEN_SIGIL.sub("", original):
            raise ValueError(f"Unterminated token sigil @{{ in: {original}")

    @staticmethod
    def _expand_array(nodes: Sequence[JsonValue], context: _ExpansionContext) -> list[JsonValue]:
        result: list[JsonValue] = []

        for node in nodes:
            expanded = Composer._expand_node(node, context)
            if Composer._is_reference(node) and isinstance(expanded, list):
                result.extend(expanded)
            else:
                result.append(expanded)

        return result

    @staticmethod
    def _expand_reference(node: JsonObject, context: _ExpansionContext) -> JsonValue:
        if "_type" in node:
            raise ValueError(".ref node must not declare _type")

        ref_path = node[REF_KEY]
        if not isinstance(ref_path, str):
            raise ValueError(".ref path must be a string")

        resolved_include = context.include_resolver(ref_path, context.referrer_dir)
        file_path = os.path.abspath(os.path.join(context.root_dir, resolved_include.file_path))
        Validator.assert_no_circular_path(file_path, context.active_paths)

        # Component args are authored by the caller, so expand them in the caller's
        # context — before this file is pushed onto active_paths. Otherwise reusing a
        # component inside another instance's arg subtree (e.g. a /card in a /card's
        # child) re-enters the same file while it is still "active" and is misflagged
        # as circular. Genuine self-recursion is still caught: a component whose own
        # .content re-references itself expands that .content with the file on the path.
        args = {
            key: Composer._expand_node(value, context)
            for key, value in node.items()
            if key != REF_KEY
        }
        target_context = replace(
            context,
            active_paths=(*context.active_paths, file_path),
            referrer_dir=os.path.dirname(file_path),
        )

        if Composer._is_component(resolved_include.content):
            return Composer._expand_component(
                resolved_include.content, args, file_path, target_context
            )

        if args:
            raise ValueError(f"Fragment {file_path} does not accept args: {', '.join(args)}")

        return Composer._expand_node(
            resolved_include.content,
            replace(target_context, include_stack=(*context.include_stack, file_path)),
        )

    @staticmethod
    def _expand_component(
        target: JsonObject,
        args: Mapping[str, JsonValue],
        file_path: str,
        context: _ExpansionContext,
    ) -> JsonValue:
        component = Composer._to_component_file(target, file_path)
        Validator.assert_args_declared(component.vars, args)
        effective_args = {**(component.defaults or {}), **args}

        substituted = Composer.substitute_vars(component.content, component.vars, effective_args)
        if substituted is None:
            raise ValueError(f"Component body resolved to an omitted slot: {file_path}")

        return Composer._expand_node(
            substituted,
            replace(context, component_stack=(*context.component_stack, file_path)),
        )

    @staticmethod
    def _substitute_node(
        node: JsonValue,
        vars: Sequence[str],
        args: Mapping[str, JsonValue],
    ) -> JsonValue:
        if isinstance(node, list):
            return Composer.splice_slots(node, vars, args)

        if isinstance(node, dict):
            result: JsonObject = {}
            for key, value in node.items():
                if key.startswith(".") and key != REF_KEY and key != TOKEN_KEY:
                    raise ValueError(f"Unsupported build key {key}")
                substituted = Composer._substitute_node(value, vars, args)
                if substituted is not _MISSING:
                    result[key] = substituted
            return result

        slot_name = Composer._slot_name(node)
        if slot_name is None:
            return node

        Validator.assert_hole_declared(vars, slot_name)
        if slot_name not in args:
            return _MISSING
        return copy.deepcopy(args[slot_name])

    @staticmethod
    def _to_component_file(target: JsonObject, file_path: str) -> ComponentFile:
        if "screen_id" in target:
            raise ValueError("screen_id is not allowed in template nodes")
        unsupported_key = next(
            (
                key
                for key in target
                if key.startswith(".")
                and key not in (VARS_KEY, DEFAULTS_KEY, CONTENT_KEY)
            ),
            None,
        )
        if unsupported_key is not None:
            raise ValueError(f"Unsupported build key {unsupported_key}")

        vars = target[VARS_KEY]
        if not Composer._is_string_list(vars) or any(len(name) == 0 for name in vars):
            raise ValueError(f"Component .vars must be a list of names: {file_path}")
        if len(set(vars)) != len(vars):
            raise ValueError(f"Component .vars contains duplicate names: {file_path}")
        defaults = target[DEFAULTS_KEY] if DEFAULTS_KEY in target else None
        if DEFAULTS_KEY in target and not isinstance(defaults, dict):
            raise ValueError(f"Component .defaults must be a map: {file_path}")
        if defaults is not None:
            for key in defaults:
                if key not in vars:
                    raise ValueError(f"Component .defaults key not in .vars: {key} ({file_path})")
        if CONTENT_KEY not in target:
            raise ValueError(f"Component with .vars must declare .content: {file_path}")

        return ComponentFile(vars=vars, defaults=defaults, content=target[CONTENT_KEY])

    @staticmethod
    def _slot_name(node: JsonValue) -> str | None:
        if not isinstance(node, str) or not node.startswith(".") or len(node) == 1:
            return None
        if node.startswith("./") or node.startswith(".."):
            return None
        return node[1:]

    @staticmethod
    def _is_reference(node: JsonValue) -> bool:
        return isinstance(node, dict) and REF_KEY in node

    @staticmethod
    def _is_component(node: JsonValue) -> bool:
        return isinstance(node, dict) and VARS_KEY in node

    @staticmethod
    def _is_string_list(node: JsonValue) -> bool:
        return isinstance(node, list) and all(isinstance(item, str) for item in node)
