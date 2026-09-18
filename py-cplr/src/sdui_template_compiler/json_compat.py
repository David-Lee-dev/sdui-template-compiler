"""Compact JSON serialization matching JS JSON.stringify output."""

import json
from typing import Any

JsonValue = Any


def stringify(value: JsonValue, indent: int | None = None) -> str:
    """Serializes a JSON value exactly like ECMAScript JSON.stringify.

    Integral floats never appear here — the YAML gate normalizes them to int
    at load time — so json.dumps output matches JS number formatting.

    @param value - JSON-compatible value
    @param indent - Optional indentation width (JSON.stringify third argument)
    @returns Serialized JSON string
    """
    if indent is None:
        return json.dumps(value, separators=(",", ":"), ensure_ascii=False)
    return json.dumps(value, indent=indent, ensure_ascii=False)
