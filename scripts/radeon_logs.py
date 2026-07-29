#!/usr/bin/env python3
"""Redact credentials and private prompt/response bodies from operator logs."""

from __future__ import annotations

import json
import re
import sys
from typing import Any


SECRET_KEYS = {
    "api_key",
    "apikey",
    "authorization",
    "bearer",
    "credential",
    "password",
    "secret",
    "token",
}
BODY_KEYS = {
    "answer",
    "completion",
    "content",
    "input",
    "output",
    "prompt",
    "question",
    "response",
}
BEARER = re.compile(r"(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+")
ASSIGNMENT = re.compile(
    r"(?i)\b(api[_-]?key|authorization|password|secret|token)\b(\s*[:=]\s*)([^\s,;]+)"
)
QUERY = re.compile(r"(?i)([?&](?:api[_-]?key|password|secret|token)=)[^&\s]+")
JSON_BODY = re.compile(
    r'(?i)("(?:answer|completion|content|input|output|prompt|question|response)"\s*:\s*)'
    r'("(?:[^"\\]|\\.)*"|\[[^\]]*\]|\{[^\}]*\})'
)


def redact_value(value: Any, key: str | None = None) -> Any:
    normalized = (key or "").lower().replace("-", "_")
    if normalized in SECRET_KEYS:
        return "[REDACTED_SECRET]"
    if normalized in BODY_KEYS:
        return "[REDACTED_BODY]"
    if isinstance(value, dict):
        return {item_key: redact_value(item_value, item_key) for item_key, item_value in value.items()}
    if isinstance(value, list):
        return [redact_value(item) for item in value]
    if isinstance(value, str):
        return redact_text(value)
    return value


def redact_text(value: str) -> str:
    redacted = BEARER.sub("Bearer [REDACTED_SECRET]", value)
    redacted = ASSIGNMENT.sub(
        lambda match: f"{match.group(1)}{match.group(2)}[REDACTED_SECRET]",
        redacted,
    )
    redacted = QUERY.sub(lambda match: f"{match.group(1)}[REDACTED_SECRET]", redacted)
    redacted = JSON_BODY.sub(lambda match: f'{match.group(1)}"[REDACTED_BODY]"', redacted)
    return redacted


def redact_line(line: str) -> str:
    prefix = ""
    candidate = line
    brace = line.find("{")
    if brace > 0:
        prefix, candidate = line[:brace], line[brace:]
    try:
        payload = json.loads(candidate)
    except json.JSONDecodeError:
        return redact_text(line)
    return prefix + json.dumps(redact_value(payload), sort_keys=True, separators=(",", ":"))


def main() -> None:
    for raw in sys.stdin:
        print(redact_line(raw.rstrip("\n")))


if __name__ == "__main__":
    main()
