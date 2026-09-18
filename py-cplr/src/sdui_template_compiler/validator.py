"""Structural validation guards shared by the composer."""

from typing import Any, Mapping, Sequence

JsonValue = Any


class Validator:
    """Composition-time validation of component interfaces and reference chains."""

    @staticmethod
    def assert_args_declared(vars: Sequence[str], args: Mapping[str, JsonValue]) -> None:
        """Rejects arguments absent from a component variable declaration.

        @param vars - Names declared by the component
        @param args - Arguments supplied by the reference node
        @throws When one or more argument names are unknown
        """
        unknown = [name for name in args if name not in vars]
        if unknown:
            raise ValueError(f"Component reference has unknown args: {', '.join(unknown)}")

    @staticmethod
    def assert_hole_declared(vars: Sequence[str], name: str) -> None:
        """Rejects a variable slot absent from its component declaration.

        @param vars - Names declared by the component
        @param name - Variable slot name found in the component body
        @throws When the slot name is undeclared
        """
        if name not in vars:
            raise ValueError(f"Component body has undeclared slot .{name}")

    @staticmethod
    def assert_no_circular_path(file_path: str, visiting: Sequence[str]) -> None:
        """Rejects a resolved file path re-entered on the active expansion chain.

        @param file_path - Absolute component or fragment path about to be expanded
        @param visiting - Ordered absolute paths currently being expanded
        @throws When the path already exists in the active chain
        """
        if file_path in visiting:
            raise ValueError(f"Circular reference: {' -> '.join([*visiting, file_path])}")
