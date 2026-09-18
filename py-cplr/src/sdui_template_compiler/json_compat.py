"""Compact JSON serialization matching JS JSON.stringify output."""

import json
from decimal import Decimal
from typing import Any

JsonValue = Any


def stringify(value: JsonValue, indent: int | None = None) -> str:
    """Serializes a JSON value exactly like ECMAScript JSON.stringify.

    Compact output feeds the etag hash, so byte fidelity with the reference
    matters: strings escape identically, and floats follow the ECMAScript
    Number-to-string algorithm (`1e-7`, not Python's `1e-07`; `0.00001`,
    not `1e-05`).

    @param value - JSON-compatible value
    @param indent - Optional indentation width (JSON.stringify third argument)
    @returns Serialized JSON string
    """
    parts: list[str] = []
    _write(value, indent, 0, parts)
    return "".join(parts)


def _write(value: JsonValue, indent: int | None, depth: int, out: list[str]) -> None:
    if value is None or isinstance(value, bool):
        out.append("null" if value is None else ("true" if value else "false"))
    elif isinstance(value, str):
        out.append(json.dumps(value, ensure_ascii=False))
    elif isinstance(value, float):
        out.append(_format_float(value))
    elif isinstance(value, int):
        out.append(str(value))
    elif isinstance(value, dict):
        _write_container(
            "{",
            "}",
            [(json.dumps(str(k), ensure_ascii=False), v) for k, v in value.items()],
            indent,
            depth,
            out,
        )
    elif isinstance(value, (list, tuple)):
        _write_container("[", "]", [(None, v) for v in value], indent, depth, out)
    else:
        raise TypeError(f"Not JSON-serializable: {type(value).__name__}")


def _write_container(
    open_ch: str,
    close_ch: str,
    entries: list[tuple[str | None, JsonValue]],
    indent: int | None,
    depth: int,
    out: list[str],
) -> None:
    if not entries:
        out.append(open_ch + close_ch)
        return

    pad = "" if indent is None else "\n" + " " * (indent * (depth + 1))
    end_pad = "" if indent is None else "\n" + " " * (indent * depth)
    colon = ":" if indent is None else ": "

    out.append(open_ch)
    for index, (key, item) in enumerate(entries):
        if index > 0:
            out.append(",")
        out.append(pad)
        if key is not None:
            out.append(key)
            out.append(colon)
        _write(item, indent, depth + 1, out)
    out.append(end_pad)
    out.append(close_ch)


def _format_float(value: float) -> str:
    """Formats a finite float per ECMAScript Number::toString (ECMA-262 §6.1.6.1.20).

    Python's shortest-roundtrip repr provides the significand digits; only the
    surrounding notation differs (exponent thresholds and `e-7` vs `e-07`).
    """
    if value != value or value in (float("inf"), float("-inf")):
        raise ValueError(f"Not JSON-serializable: {value}")
    if value == 0:
        return "0"

    sign = "-" if value < 0 else ""
    tup = Decimal(repr(abs(value))).normalize().as_tuple()
    digits = "".join(map(str, tup.digits))
    k = len(digits)
    n = int(tup.exponent) + k  # decimal point position: 0.digits * 10^n

    if k <= n <= 21:
        body = digits + "0" * (n - k)
    elif 0 < n <= 21:
        body = digits[:n] + "." + digits[n:]
    elif -6 < n <= 0:
        body = "0." + "0" * (-n) + digits
    else:
        exponent = n - 1
        mantissa = digits[0] + ("." + digits[1:] if k > 1 else "")
        exp_sign = "+" if exponent >= 0 else "-"
        body = f"{mantissa}e{exp_sign}{abs(exponent)}"
    return sign + body
