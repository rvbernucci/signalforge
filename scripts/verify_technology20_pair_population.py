#!/usr/bin/env python3
"""Independently verify the complete Technology 20 pair population."""

from __future__ import annotations

import argparse
import hashlib
import json
from collections import Counter
from copy import deepcopy
from pathlib import Path
from typing import Any


SCHEMA_VERSION = "signalforge/technology20-pair-population-reference-check/v1"


def compact_json(value: Any) -> bytes:
    return json.dumps(
        value,
        ensure_ascii=False,
        separators=(",", ":"),
    ).encode("utf-8")


def content_hash(value: Any) -> str:
    return hashlib.sha256(compact_json(value)).hexdigest()


def verify(path: Path) -> dict[str, Any]:
    population = json.loads(path.read_text(encoding="utf-8"))
    failures: list[str] = []
    pairs_seen: set[str] = set()
    disposition_counts: Counter[str] = Counter()
    receipts = 0
    abstentions = 0
    context_only = 0
    released = 0
    expected_operations = {
        "accounting.balance_sheet_identity",
        "financial.revenue_growth",
        "financial.operating_margin",
        "financial.free_cash_flow",
        "financial.cash_conversion",
        "financial.capex_intensity",
        "financial.quality_of_earnings",
    }
    if population.get("schema_version") != "signalforge/technology20-pair-population/v1":
        failures.append("population:schema_version")
    if len(population.get("company_reports", [])) != 20:
        failures.append("population:company_report_count")
    for reference in population.get("company_reports", []):
        if len(reference.get("report_sha256", "")) != 64:
            failures.append(f"{reference.get('company_id')}:report_hash")
    for pair in population.get("pairs", []):
        company_ids = pair.get("company_ids", [])
        pair_key = "|".join(company_ids)
        if (
            len(company_ids) != 2
            or company_ids != sorted(company_ids)
            or pair_key in pairs_seen
            or pair.get("promoted", False)
        ):
            failures.append(f"{pair.get('lane_id')}:pair_identity")
        pairs_seen.add(pair_key)
        envelope = deepcopy(pair)
        expected_envelope_hash = envelope.pop("envelope_sha256", "")
        if content_hash(envelope) != expected_envelope_hash:
            failures.append(f"{pair.get('lane_id')}:envelope_hash")
        releasable = set(pair.get("releasable_metric_ids", []))
        contextual = set(pair.get("context_only_metric_ids", []))
        withheld = set(pair.get("withheld_metric_ids", []))
        if releasable & contextual or releasable & withheld or contextual & withheld:
            failures.append(f"{pair.get('lane_id')}:overlapping_classes")
        seen_metrics: set[str] = set()
        for receipt in pair.get("receipts", []):
            receipts += 1
            disposition = receipt.get("disposition", "")
            disposition_counts[disposition] += 1
            metric_id = receipt.get("operands", [{}])[0].get(
                "canonical_metric_id", ""
            )
            if not metric_id or metric_id in seen_metrics:
                failures.append(f"{pair.get('lane_id')}:{metric_id}:duplicate_metric")
            seen_metrics.add(metric_id)
            receipt_copy = deepcopy(receipt)
            expected_receipt_hash = receipt_copy.get("receipt_sha256", "")
            receipt_copy["receipt_sha256"] = ""
            if content_hash(receipt_copy) != expected_receipt_hash:
                failures.append(
                    f"{pair.get('lane_id')}:{metric_id}:receipt_hash"
                )
            if disposition in {"comparable", "comparable_with_caveat"}:
                released += 1
                if metric_id not in releasable:
                    failures.append(
                        f"{pair.get('lane_id')}:{metric_id}:release_class"
                    )
            elif disposition == "context_only":
                context_only += 1
                if metric_id not in contextual or metric_id in releasable:
                    failures.append(
                        f"{pair.get('lane_id')}:{metric_id}:context_class"
                    )
                if "non_ranking_context_only" not in receipt.get(
                    "required_caveat_ids", []
                ):
                    failures.append(
                        f"{pair.get('lane_id')}:{metric_id}:context_caveat"
                    )
                if all(
                    operand.get("pair_ranking_eligible", False)
                    for operand in receipt.get("operands", [])
                ):
                    failures.append(
                        f"{pair.get('lane_id')}:{metric_id}:ranking_escape"
                    )
            elif disposition == "not_comparable":
                if metric_id not in withheld:
                    failures.append(
                        f"{pair.get('lane_id')}:{metric_id}:withheld_class"
                    )
            else:
                failures.append(
                    f"{pair.get('lane_id')}:{metric_id}:unknown_disposition"
                )
            for operand in receipt.get("operands", []):
                if (
                    not operand.get("source_observation_ids")
                    or not operand.get("source_hashes")
                    or not operand.get("accounting_inputs")
                    or not operand.get("product_label")
                ):
                    failures.append(
                        f"{pair.get('lane_id')}:{metric_id}:operand_lineage"
                    )
        for abstention in pair.get("abstentions", []):
            abstentions += 1
            metric_ids = abstention.get("metric_ids", [])
            metric_id = metric_ids[0] if len(metric_ids) == 1 else ""
            if (
                not metric_id
                or metric_id in seen_metrics
                or metric_id not in withheld
            ):
                failures.append(
                    f"{pair.get('lane_id')}:{metric_id}:abstention_class"
                )
            seen_metrics.add(metric_id)
        if seen_metrics != expected_operations:
            failures.append(f"{pair.get('lane_id')}:operation_coverage")
    if len(pairs_seen) != 190:
        failures.append(f"population:expected_190_found_{len(pairs_seen)}")
    population_copy = deepcopy(population)
    expected_population_hash = population_copy.get("population_sha256", "")
    population_copy["population_sha256"] = ""
    if content_hash(population_copy) != expected_population_hash:
        failures.append("population:hash")
    return {
        "schema_version": SCHEMA_VERSION,
        "source_population_sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
        "embedded_population_sha256": expected_population_hash,
        "pairs": len(pairs_seen),
        "receipts": receipts,
        "abstentions": abstentions,
        "released_metrics": released,
        "context_only_metrics": context_only,
        "dispositions": dict(sorted(disposition_counts.items())),
        "failures": failures,
        "passed": not failures,
        "claim_boundary": (
            "This independent verifier checks pair completeness, hashes, lineage, "
            "classification, and the non-ranking context-only boundary. It does not "
            "authorize an economic peer relationship or investment conclusion."
        ),
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--population", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    report = verify(args.population)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(
        json.dumps(report, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )
    if not report["passed"]:
        raise SystemExit("Technology 20 pair population verification failed")
    print(
        f"pair reference check passed: {report['pairs']} pairs, "
        f"{report['receipts']} receipts, {report['abstentions']} abstentions"
    )


if __name__ == "__main__":
    main()
