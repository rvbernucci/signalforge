#!/usr/bin/env python3
"""Compare two IR document manifests without treating absence as deletion proof."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--previous", type=Path, required=True)
    parser.add_argument("--current", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    return parser.parse_args()


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def compare(previous: list[dict[str, Any]], current: list[dict[str, Any]]) -> list[dict[str, Any]]:
    previous_by_uri = {item["canonical_uri"]: item for item in previous}
    current_by_uri = {item["canonical_uri"]: item for item in current}
    records: list[dict[str, Any]] = []
    for uri in sorted(previous_by_uri.keys() | current_by_uri.keys()):
        old = previous_by_uri.get(uri)
        new = current_by_uri.get(uri)
        if old and new:
            disposition = "unchanged" if old["content_sha256"] == new["content_sha256"] else "content_changed"
        elif new:
            disposition = "newly_observed"
        else:
            disposition = "not_observed_in_current_run"
        record = {"canonical_uri": uri, "disposition": disposition}
        if old:
            record.update(
                {
                    "previous_document_id": old["document_id"],
                    "previous_content_sha256": old["content_sha256"],
                    "previous_retrieved_at": old["retrieved_at"],
                }
            )
        if new:
            record.update(
                {
                    "current_document_id": new["document_id"],
                    "current_content_sha256": new["content_sha256"],
                    "current_retrieved_at": new["retrieved_at"],
                }
            )
        records.append(record)
    return records


def main() -> int:
    args = parse_args()
    records = compare(read_jsonl(args.previous), read_jsonl(args.current))
    counts: dict[str, int] = {}
    for record in records:
        counts[record["disposition"]] = counts.get(record["disposition"], 0) + 1
    report = {
        "schema_version": "signalforge/ir-temporal-lineage/v1",
        "counts": dict(sorted(counts.items())),
        "records": records,
        "claim_boundary": "Not observed in one bounded run is a collection gap, not proof that an issuer deleted a document.",
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    print(json.dumps(report["counts"], sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
