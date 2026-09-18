"""YAML loading with js-yaml v4 (YAML 1.2 core schema) semantics."""

import math
import re
from typing import Any

import yaml as pyyaml

JsonValue = Any


_MAX_SAFE_INTEGER = 2**53 - 1  # JS Number.MAX_SAFE_INTEGER


class _CoreSchemaLoader(pyyaml.SafeLoader):
    """SafeLoader with YAML 1.2 core-schema implicit resolvers.

    PyYAML defaults to YAML 1.1, where `on`/`off`/`yes`/`no` are booleans and
    `1_000` is an int. The TS reference uses js-yaml v4 (YAML 1.2 core), so
    those plain scalars must stay strings; only true/false (and their
    Title/UPPER forms) are booleans.
    """


# Drop every default implicit resolver for bool/int/float/null, keep the rest
# (timestamp etc. would produce non-JSON values; js-yaml core has no timestamp,
# so drop it too — the JSON gate would reject it anyway, but with the wrong
# error shape for merge-key style scalars).
_DROPPED_TAGS = {
    "tag:yaml.org,2002:bool",
    "tag:yaml.org,2002:int",
    "tag:yaml.org,2002:float",
    "tag:yaml.org,2002:null",
    "tag:yaml.org,2002:timestamp",
    "tag:yaml.org,2002:value",
}
_CoreSchemaLoader.yaml_implicit_resolvers = {
    first: [(tag, regexp) for tag, regexp in resolvers if tag not in _DROPPED_TAGS]
    for first, resolvers in pyyaml.SafeLoader.yaml_implicit_resolvers.items()
}

_CoreSchemaLoader.add_implicit_resolver(
    "tag:yaml.org,2002:null",
    re.compile(r"^(?:~|null|Null|NULL|)$"),
    ["~", "n", "N", ""],
)
_CoreSchemaLoader.add_implicit_resolver(
    "tag:yaml.org,2002:bool",
    re.compile(r"^(?:true|True|TRUE|false|False|FALSE)$"),
    list("tTfF"),
)
_CoreSchemaLoader.add_implicit_resolver(
    "tag:yaml.org,2002:int",
    re.compile(r"^(?:[-+]?[0-9]+|0o[0-7]+|0x[0-9a-fA-F]+)$"),
    list("-+0123456789"),
)
_CoreSchemaLoader.add_implicit_resolver(
    "tag:yaml.org,2002:float",
    re.compile(
        r"^(?:[-+]?(?:\.[0-9]+|[0-9]+(?:\.[0-9]*)?)(?:[eE][-+]?[0-9]+)?"
        r"|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$"
    ),
    list("-+0123456789."),
)


def _construct_int(loader: pyyaml.SafeLoader, node: pyyaml.ScalarNode) -> int:
    return int(node.value, 0) if node.value.startswith(("0x", "0o")) else int(node.value)


_CoreSchemaLoader.add_constructor("tag:yaml.org,2002:int", _construct_int)


class Yaml:
    """YAML file loader producing JSON-compatible values."""

    @staticmethod
    def load(path: str) -> JsonValue:
        """Loads a YAML file as a JSON-compatible value.

        @param path - YAML file path
        @returns Parsed JSON-compatible value
        @throws When YAML contains a value that JSON cannot represent
        """
        with open(path, "r", encoding="utf-8") as handle:
            parsed = pyyaml.load(handle, Loader=_CoreSchemaLoader)
        return Yaml._assert_json_value(parsed, path)

    @staticmethod
    def _assert_json_value(value: Any, path: str) -> JsonValue:
        if value is None or isinstance(value, (bool, str)):
            return value
        if isinstance(value, int):
            # Beyond 2^53 JS holds the value as a lossy double (`1e21` prints
            # as "1e+21", not full digits) — mirror it to keep bytes identical.
            return value if abs(value) <= _MAX_SAFE_INTEGER else float(value)
        if isinstance(value, float):
            if not math.isfinite(value):
                raise ValueError(f"YAML is not JSON-compatible: {path}")
            # JS numbers have no int/float split: an integral float must
            # serialize without a trailing `.0` to match JSON.stringify.
            # Only exact conversions qualify — above 2^53 stay float so the
            # serializer applies JS exponent notation.
            if value.is_integer() and abs(value) <= _MAX_SAFE_INTEGER:
                return int(value)
            return value
        if isinstance(value, list):
            return [Yaml._assert_json_value(item, path) for item in value]
        if isinstance(value, dict):
            result = {}
            for key, item in value.items():
                if not isinstance(key, str):
                    raise ValueError(f"YAML is not JSON-compatible: {path}")
                result[key] = Yaml._assert_json_value(item, path)
            return result

        raise ValueError(f"YAML is not JSON-compatible: {path}")
