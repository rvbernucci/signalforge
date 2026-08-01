#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

if ! python3 -c 'import jsonschema' >/dev/null 2>&1; then
  echo "Missing verification dependency. Run: python3 -m pip install -r requirements-verify.txt" >&2
  exit 1
fi

# Build the exact frontend consumed by the Go workspace before exercising the application.
(cd web && npm ci --no-audit --no-fund && npm run test && npm run build)

# Current implementation, contracts, fixtures, and failure behavior.
go test -race -count=1 ./...
go vet ./...
python3 scripts/reference_finance.py
python3 -m unittest discover -s scripts/tests -p 'test_*.py'
python3 -m compileall -q scripts deploy/observability/radeon-exporter

while IFS= read -r -d '' shell_file; do
  bash -n "$shell_file"
done < <(find scripts -maxdepth 1 -type f -name '*.sh' -print0)

while IFS= read -r -d '' json_file; do
  jq empty "$json_file"
done < <(find contracts evidence fixtures configs deploy -type f -name '*.json' -print0)

# Security, privacy, deployment, and adversarial release gates.
python3 scripts/run_hardening_matrix.py --check
jq -e '
  .schema_version == "signalforge/hardening-report/v1" and
  .status == "passed" and
  .cases == 26 and
  .release_blockers == 0 and
  (.gates | length) == 11 and
  all(.gates[]; .status == "passed" and (.source_hashes | length) > 0)
' evidence/hardening-matrix.json >/dev/null
python3 scripts/radeon_validate_appliance.py >/dev/null
python3 scripts/radeon_validate_appliance.py \
  --manifest deploy/radeon/appliance-manifest.vnext.json >/dev/null
python3 scripts/validate_observability.py
python3 scripts/audit_restricted_egress.py >/dev/null

# Active Radeon decision records. Historical raw runs remain outside the public repository.
jq -e '
  .schema_version == "signalforge/radeon-baseline/v1" and
  (.candidates | length) == 3 and
  ([.candidates[] | select(
    .profile_id == "gemma4-26b-a4b-qat-q4-llama-rocm" and
    .contract_checks_passed == 40 and
    .contract_checks_total == 40 and
    .decode_tokens_per_second_p50 == 86.4601
  )] | length) == 1
' evidence/radeon-baseline.json >/dev/null
jq -e '
  .schema_version == "signalforge/radeon-optimization-decision/v1" and
  .selected_configuration.flash_attention == "auto" and
  .selected_configuration.kv_cache == "unified_f16" and
  .selected_configuration.context_capacity_tokens == 32768 and
  .selected_configuration.server_slots == 4 and
  .selected_configuration.product_context_concurrency == 4 and
  .selected_configuration.continuous_batching == true and
  .accepted_improvements.selected_vs_three_context_workers_end_to_end_improvement_percent >= 29
' evidence/radeon-optimization.json >/dev/null

# Current product boundary and deterministic fixture experience.
jq -e '
  .schema_version == "signalforge/research-workspace/v1" and
  .status == "completed" and
  (.companies | length) == 2 and
  (.sections | length) == 8 and
  (.evidence | length) == 12 and
  (.calculations | length) == 18 and
  .execution.local_only == true and
  .execution.endpoint_scope == "loopback_only" and
  .metrics.evidence_coverage == 1
' fixtures/workspace/golden-case.json >/dev/null

go run ./cmd/signalforge-eval-workspace \
  --output "$tmp_dir/workspace-evaluation.json"
jq -e '
  .schema_version == "signalforge/workspace-evaluation/v1" and
  .mode == "fixture" and .local_only == true and
  .frontend.index_status == 200 and
  .frontend.content_security_ready == true and
  .frontend.initial_case_ms < 1000 and
  .journey.start_status == 202 and
  .journey.time_to_first_progress_ms < 1000 and
  .journey.time_to_completed_case_ms < 2000 and
  .journey.streamed_events >= 21 and
  .journey.sections == 8 and
  .journey.evidence_items == 12 and
  .journey.calculation_receipts == 18 and
  .journey.private_fields_excluded == true
' "$tmp_dir/workspace-evaluation.json" >/dev/null
jq -e --slurpfile fresh "$tmp_dir/workspace-evaluation.json" '
  .schema_version == "signalforge/workspace-evaluation/v1" and
  .mode == "fixture" and .local_only == true and
  .journey.streamed_events == $fresh[0].journey.streamed_events and
  .journey.sections == $fresh[0].journey.sections and
  .journey.evidence_items == $fresh[0].journey.evidence_items and
  .journey.calculation_receipts == $fresh[0].journey.calculation_receipts and
  .journey.private_fields_excluded == $fresh[0].journey.private_fields_excluded
' evidence/workspace-evaluation.json >/dev/null

go run ./cmd/signalforge-calculate \
  --request fixtures/engine/margin-request.json \
  --output "$tmp_dir/calculation-result.json" \
  --receipt-store "$tmp_dir/receipts" \
  --code-commit verification-tree
jq -e '.receipt.status == "success" and .receipt.outputs[0].quantity.value == "0.25"' \
  "$tmp_dir/calculation-result.json" >/dev/null

# Typed architecture, orchestration, prompts, retrieval, and active product authority.
go run ./cmd/signalforge-eval-architecture > "$tmp_dir/architecture-eval.json"
cmp evidence/architecture-eval.json "$tmp_dir/architecture-eval.json"
go run ./cmd/signalforge-eval-orchestration > "$tmp_dir/orchestration-eval.json"
cmp evidence/orchestration-eval.json "$tmp_dir/orchestration-eval.json"
go run ./cmd/signalforge-export-prompts > "$tmp_dir/role-prompts-v12.json"
cmp configs/prompts/role-prompts-v12.json "$tmp_dir/role-prompts-v12.json"

jq -e '
  .schema_version == "signalforge/role-evaluation-suite/v1" and
  .suite_id == "role-held-out-v2" and
  .prompt_set_version == "signalforge-role-prompts/v12" and
  (.cases | length) == 33
' fixtures/roles/held-out-v12-cases.json >/dev/null
jq -e '
  .schema_version == "signalforge/technology20-public-catalog/v1" and
  (.companies | length) == 20 and
  ([.companies[] | select(.research_enabled == true)] | length) == 20 and
  (.peer_lanes | length) == 5 and
  all(.peer_lanes[]; .enabled == true)
' fixtures/productscope/technology20-catalog.json >/dev/null
jq -e '
  .schema_version == "signalforge/technology20-public-financial-summary/v2" and
  (.companies | length) == 20
' fixtures/productscope/technology20-financial-summary.json >/dev/null
jq -e '
  .schema_version == "signalforge/technology20-peer-evaluation/v1" and
  (.lanes | length) == 5
' fixtures/productscope/technology20-peer-evaluation.json >/dev/null

go run ./cmd/signalforge-eval-retrieval \
  --eval fixtures/retrieval/golden-eval.json \
  --vectors fixtures/retrieval/vectors/granite-embedding-97m-multilingual-r2.json \
  --output "$tmp_dir/retrieval-eval.json"
jq -e '
  (.methods[] | select(.method == "bm25/v1") | .metrics.complete_evidence_rate) == 1 and
  (.methods[] | select(.method == "cosine/v1") | .metrics.recall_at_k) >= 0.84 and
  ([.methods[].metrics.citation_correctness] | min) == 1
' "$tmp_dir/retrieval-eval.json" >/dev/null

# Current, privacy-safe championship aggregates.
jq -e '
  .schema_version == "signalforge/championship-evaluation/v1" and
  .status == "evaluated_candidate" and
  .scope.companies == 20 and
  .scope.total_cases == 180 and
  .runtime_and_contract.passed == 180 and
  .runtime_and_contract.total == 180 and
  .model_assisted_semantic_review.accepted == 180 and
  .model_assisted_semantic_review.false_release_candidates == 0 and
  .final_authority == "not_granted"
' evidence/championship-evaluation.json >/dev/null
jq -e '
  .schema_version == "signalforge/championship-radeon-runtime/v1" and
  .status == "measured_candidate" and
  .platform.gpu_architecture == "gfx1100" and
  .platform.host_rocm_version == "7.2.1" and
  .selected_model_profile.contract_checks_passed == 40 and
  .selected_model_profile.contract_checks_total == 40 and
  .hybrid_route.runtime_passed == 5 and
  .soak.journeys_runtime_and_contract_passed == 180 and
  .soak.vram_growth_percentage_points == 0 and
  .failure_behavior.local_model_loss.answer_released == false and
  .final_authority == "not_granted"
' evidence/championship-radeon-runtime.json >/dev/null
jq -e '
  .schema_version == "signalforge/championship-product-check/v1" and
  .status == "automated_and_agent_operated_acceptance_passed" and
  all(.checks[]; . == true) and
  .bounded_live_scope.adobe_standalone.released == true and
  .bounded_live_scope.nvidia_amd_peer.released == true and
  .bounded_live_scope.overbroad_peer_request.released == false and
  .judge_navigation.under_two_minutes == true and
  .human_acceptance == "pending" and
  .final_authority == "not_granted"
' evidence/championship-product-check.json >/dev/null

# Public-claim identity and repository hygiene are the final source-level gates.
go run ./cmd/signalforge-release-check \
  --root . \
  --claims evidence/public-claims.json
python3 scripts/audit_public_repo.py --check

go run ./cmd/signalforge-evidence \
  --repo . \
  --output "$tmp_dir/manifest.json" \
  --artifact evidence/championship-evaluation.json \
  --artifact evidence/championship-radeon-runtime.json \
  --artifact evidence/championship-product-check.json \
  --artifact evidence/public-claims.json
go run ./cmd/signalforge-evidence \
  --repo . \
  --check "$tmp_dir/manifest.json"

git diff --check --cached
git diff --check
echo "SignalForge verification passed"
