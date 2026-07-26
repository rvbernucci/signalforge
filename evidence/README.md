# Evidence Control

SignalForge treats evaluation evidence as a versioned build artifact. The score ledger tracks what
must be demonstrated; measured runs will be stored under hash-addressed directories with an
`EvidenceManifest` describing the exact code, environment, model, dataset, and artifacts used.

Planning language is not proof. A criterion becomes `verified` only when the referenced artifact
exists and its hash can be reproduced.

## Commands

Generate a manifest around selected artifacts:

`go run ./cmd/signalforge-evidence --repo . --output /tmp/signalforge-manifest.json \
  --artifact evidence/architecture-eval.json`

Measured Radeon runs additionally pass `--runtime`, `--gpu`, repeatable
`--model`, and repeatable `--dataset` JSON identity files. These record model
revision, artifact and tokenizer hashes, quantization, serving runtime, ROCm,
GPU identity, and dataset manifests without placing model weights in Git.

Reject a stale manifest:

`go run ./cmd/signalforge-evidence --repo . \
  --check /tmp/signalforge-manifest.json`

Validate registered public claims:

`go run ./cmd/signalforge-release-check --root . \
  --claims evidence/public-claims.json`

After a human has confirmed that an existing claim remains semantically supported by its declared
evidence, refresh only those already-declared evidence hashes and immediately re-run the gate:

```bash
go run ./cmd/signalforge-release-check --root . \
  --claims evidence/public-claims.json --refresh-evidence
go run ./cmd/signalforge-release-check --root . \
  --claims evidence/public-claims.json
```

Refresh never adds evidence paths or changes claim text, status, or public scope. Both refresh and
validation reject absolute paths, repository traversal, missing files, and symlinks that resolve
outside the repository. A passing hash gate proves byte identity, not that changed evidence still
supports the claim; semantic review remains mandatory before refresh.

Validate the public, privacy-safe replay of the current Radeon golden run:

`go run ./cmd/signalforge-validate-replay \
  --input evidence/golden-safe-decision-replay.json`

The replay is a deliberately lossy projection of the private atomic trace. It retains route reason
codes, content hashes, claim dispositions, aggregate latency, token counts, and the hash-pinned
Radeon/ROCm/model identity. It excludes prompts, responses, source excerpts, free-form failure
messages, and private reasoning. The verifier rejects unknown fields, unsafe identifiers, broken
claim references, incomplete runtime attestation, and privacy flags that are not explicitly true.

The final release checklist is intentionally fail-closed. It remains pending
until later Sprints produce the Radeon trace, golden demo, licenses, public
artifacts, secret scan, and SG-05 freeze.

## Retrieval Evidence

The frozen retrieval population combines 25 point-in-time chunks from Microsoft and NVIDIA
regulatory filings and official investor-relations material with 17 investor questions. The
lexical baseline, two hash-pinned Granite embedding baselines, reciprocal-rank fusion variants,
and a local-mode Qdrant comparison are under `evidence/retrieval`.

`configs/retrieval/retrieval-policy-v1.json` records the measured decision: BM25 with bounded
financial-concept expansion is the MVP primary path. Semantic retrieval remains an offline
baseline until it improves complete-evidence rate, while Qdrant and a reranker remain deferred.
Raw issuer files are not committed; `fixtures/investor-relations/document-manifest.json` preserves
their official URLs, SHA-256 identities, authority tiers, temporal metadata, and rights class.

## Golden Radeon Replay

`golden-safe-decision-replay.json` records the successful `golden-run-20260722-v57`
run on the selected local Gemma 4 QAT Q4_0 runtime. The run completed locally on Radeon `gfx1100`
and ROCm 7.2.1, dispositioned all 42 supplied claims, released 31 authority-backed claims approved
by both independent reviewer roles, and retained complete evidence coverage. It contains nine route
decisions and no private prompt or answer body. Assumption-backed scenario claims remain explicitly
distinguishable from evidence-, receipt-, and numerical-authorized claims.

The corresponding frozen semantic evaluation passed 44/44 predeclared checks. Official exchange
closing-price inputs captured before the analysis as-of boundary enabled two validated
price-implied peer-multiple receipts. This proves contract conformance for the frozen journey, not
perfect factual accuracy against an external human ground truth.

## Radeon Workload Optimization

`radeon-optimization.json` is the consolidated Sprint 11 decision over a benchmark contract frozen
before tuning. The accepted runtime uses Gemma 4 26B A4B QAT Q4_0, ROCm `llama.cpp`, flash
attention `auto`, continuous batching, four server slots, four product context workers, and a
unified F16 KV cache with shared 32,768-token request capacity.

The experiment first exposed that explicit four-slot serving without unified KV reduced each slot
to 8,192 tokens and rejected every long-context case. This run is retained as an invalid
experiment and is not used for speed comparison. Unified KV restored 80/80 contract success. In
the full golden journey, four context workers completed 44/44 semantic checks in 157.47 seconds;
three workers also passed 44/44 but required 222.31 seconds. Forced flash attention, a 1,024-token
micro-batch, Q8 KV, and two-worker product concurrency were rejected by the full-journey,
efficiency, or quality gates recorded in the report.

The safe Sprint 11 artifacts under `runs/sprint11` include synthetic benchmark reports, summaries,
telemetry, manifests, native metric deltas, privacy-safe golden replays, and frozen semantic
evaluations. Private prompts, responses, source excerpts, free-form failures, and full private
golden reports remain excluded from this repository.

Rebuild and verify the public decision and chart with:

```bash
python3 scripts/build_radeon_optimization_report.py --check
python3 scripts/render_radeon_optimization.py --output /tmp/radeon-optimization.svg
cmp evidence/radeon-optimization.svg /tmp/radeon-optimization.svg
```

## Adversarial Hardening

`hardening-matrix.json` is the deterministic Sprint 12 result for the frozen 26-case matrix in
`configs/hardening/sprint12-matrix-v1.json`. It records 22 critical and four high-severity cases,
11 passing executable gates, source hashes, and zero current release blockers. Every case names
its threat, owner, expected behavior, mitigation, and residual risk.

The matrix adds direct tests for retrieved prompt injection, impossible or stale market data,
direct investment instructions, guaranteed outcomes, isolated startup, and bounded demo read
load. It also binds existing temporal, deterministic-engine, citation, provider-chaos, tool,
memory, privacy, and follow-up gates into one release decision.

Reproduce the report and the clean fixture startup with:

```bash
python3 scripts/run_hardening_matrix.py --check
scripts/verify_clean_startup.sh
```

The report does not claim universal semantic entailment, immunity to novel prompt injection,
vendor-data correctness, container-pull behavior, or concurrent 26B generation. Those limitations
remain explicit in the source matrix.

## Judge Package

`judge-package.json` binds the six-page project specification, six-slide supplemental deck,
architecture diagram, final cut sheet, narration, safe live-run export, capture manifest, and
4 minute 12.9 second H.264/AAC demo to their current hashes. The PDF was rendered page by page and
visually inspected. The deck was rendered through an external office renderer, inspected slide by
slide, and passed the canvas-overflow gate.

`runs/sprint13/live-demo-capture.json` records the real Radeon run, governed follow-up, memory
control, runtime identity, playback disclosures, video properties, audio measurements, and the
12-timestamp visual review. The primary run completed locally in 161.51 seconds with ten local
model calls, six context packets, 38/38 supported claims, eight required sections, and complete
evidence coverage. Those values are bound to `live-demo-safe-export.json`; they do not replace the
separately frozen Sprint 11 optimization result.

The package status is `public_artifacts_verified`. The recording passed technical and visual
review, and the video, PDF, deck, architecture, cut sheet, narration, capture manifest, safe
export, release page, and repository were downloaded or opened without authentication. Every
downloaded artifact matched its registered local SHA-256.

## Chaos Evidence

`TestUnifiedFakeProviderChaosSuite` is the deterministic failure-injection gate for the current
agent boundary. It exercises malformed JSON, a single bounded larger retry for incomplete JSON,
provider timeout, invented evidence, contradictory review, and one failed specialist within a
multi-specialist run. The partial-failure case must remain observable in the trace and may release
only claims from surviving packets that completed review.

`go test ./internal/localagent -run TestUnifiedFakeProviderChaosSuite -count=1 -v`

## Follow-Up Contract Evidence

`TestRuntimeExecutesThreeGovernedFollowUpsWithScopeAndEvidenceLineage` executes three chained
follow-ups through the typed runtime. Every child request receives a new run and request identity,
links to its immediate parent, retains the original point-in-time and company/comparison scope, and
passes prior evidence and receipt IDs to specialists as retrieval lineage. Those IDs do not confer
authority: the new packet must still load authorized material, survive claim validation, and pass
independent review.

`go test ./internal/orchestrator \
  -run TestRuntimeExecutesThreeGovernedFollowUpsWithScopeAndEvidenceLineage -count=1 -v`

## Golden Journey Scorecard

`golden-journey-scorecard.json` is a machine-checked projection of the v57 replay plus the semantic,
chaos, and follow-up gates. It reports local runtime identity, claim disposition and authority coverage,
evidence coverage, latency, throughput, resilience, and continuity. The verifier recomputes its
source replay hash and cross-checks every replay-derived value.

The scorecard deliberately does not translate complete evidence coverage into answer accuracy.
External semantic accuracy remains `not_scored_against_external_ground_truth`. The frozen 44-check
rubric, official point-in-time price fixture, and validated multiple receipts close the bounded
Sprint 08 Microsoft/NVIDIA vertical without claiming broader market coverage.

## Research Workspace Evidence

`workspace-evaluation.json` records a dated evaluation of the deterministic fixture experience.
It verifies that the production frontend is served with the local security policy, the safe case
is immediately available, the SSE stream reaches a workspace-level terminal event, and the result
contains all eight chapters, 12 answer-used evidence cards, and 18 successful calculation receipts.

The observed 1.137 ms initial case, 7.222 ms first progress event, and 114.362 ms complete replay
measure the fixture demo path only. They are not model or Radeon latency claims. The complete live
Radeon v57 duration remains reported separately in the golden replay and scorecard.

Regenerate a fresh threshold-checked observation after building the frontend:

```bash
npm --prefix web ci
npm --prefix web run build
go run ./cmd/signalforge-eval-workspace --output /tmp/workspace-evaluation.json
```

## Live Execution Plan CPU Overhead

`dashboard-cpu-evidence.json` closes the observational dashboard's accepted-workload CPU gate
without attributing local-model generation variance to the UI projection. Five deterministic
`linux/amd64` benchmark repetitions measure the same accepted plan and lifecycle workload with the
projection disabled and enabled. Their medians are 10.663 ms and 41.421 ms per operation, leaving
30.758 ms of incremental projection work.

`dashboard-workload-cpu-radeon.json` independently records one accepted local Gemma journey on the
Radeon host. It completed on the first attempt, used ten model calls, and consumed 270.013214
seconds of orchestrator-plus-model CPU. Neither artifact retains prompt, response, source, or
private reasoning bodies. The resulting conservative upper bound is **0.011391362%**, below the
strict one-percent gate.

The raw AB/BA model experiment is not decision evidence: model repair count and generated-token
volume varied between conditions. The accepted method keeps model variance out of the incremental
numerator and binds the benchmark, capture runners, workload binary, and source artifacts by
SHA-256.

```bash
python3 scripts/build_dashboard_cpu_evidence.py \
  --benchmark evidence/dashboard-cpu-benchmark-radeon.txt \
  --workload evidence/dashboard-workload-cpu-radeon.json \
  --output evidence/dashboard-cpu-evidence.json \
  --check
```

## Synchronized Radeon Dashboard Evidence

`dashboard-radeon-local-journey.json` and `dashboard-radeon-hybrid-journey.json` are sanitized
aggregates from accepted working-tree journeys on the Radeon host. The local journey records 12
local ROCm calls. The hybrid journey records 18 calls across `radeon-vllm` and `local-rocm`,
including the bounded fallback path. Both plans reached 12 of 12 terminal steps across the eight
governed phases and released an accepted result.

`dashboard-radeon-synchronized-captures.json` binds those manifests to four 1280×720 Workspace and
Mission Control captures, the tested workspace binary, and the frontend bundle. Its verifier
rejects incomplete plans, missing phase coverage, mismatched image formats, captures below
1280×720, a local run using a remote provider, a hybrid run without both Radeon API and local ROCm,
or any declared retention of prompts, responses, source bodies, or credentials.

This evidence closes the synchronized working-tree Radeon gate only. The manifest deliberately
sets `exact_release_artifact` and `release_claim_permitted` to `false`; exact source/image binding
remains a separate release decision.

```bash
python3 scripts/build_dashboard_radeon_evidence.py \
  --local evidence/dashboard-radeon-local-journey.json \
  --hybrid evidence/dashboard-radeon-hybrid-journey.json \
  --local-plan docs/assets/live-execution-plan-radeon-local-viewport.jpg \
  --local-mission docs/assets/mission-control-radeon-local-viewport.jpg \
  --hybrid-plan docs/assets/live-execution-plan-radeon-hybrid-viewport.jpg \
  --hybrid-mission docs/assets/mission-control-radeon-hybrid-viewport.jpg \
  --binary-sha256 c0f741b2ffdf2daac8fb8d0fa0f79acc1f88fb1b6dff43113934122c3c1d7992 \
  --frontend-sha256 c7cfaaf9802c25154dac2719d94a1059afad888392da6dbde4bd4f3078d7bfa7 \
  --output evidence/dashboard-radeon-synchronized-captures.json \
  --check
```

## Local Memory And Privacy Controls

The case-store tests exercise opt-in save, integrity-checked load, bounded listing, export, cascade
deletion, restrictive filesystem modes, secure deletion, duplicate rejection, and credential-shape
rejection. Workspace integration tests prove that retention is off by default, user-controlled,
and nonfatal: a storage failure leaves the completed research result available while exposing a
separate safe retention error.

The permission-policy tests keep model authority read-only and reserve case mutations for explicit
user actions. The circuit-breaker tests verify three-failure opening, cooldown recovery, and success
reset. These controls complement the existing orchestration trace-privacy test; they do not claim
disk encryption or multi-user authentication.

```bash
go test -race ./internal/casestore ./internal/privacy ./internal/permissions \
  ./internal/resilience ./internal/workspace
```
