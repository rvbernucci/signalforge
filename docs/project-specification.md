# SignalForge

## Project Summary

SignalForge is a private, local-first financial research workspace for independent investors. It
turns public company evidence into a governed multi-agent process that can reason, plan, retrieve,
invoke deterministic tools, preserve optional local memory, challenge its own conclusions, and
release only contract-valid research.

The product runs core inference locally on AMD Radeon through ROCm. A bounded organizer-provided
Radeon API route may accelerate selected context specialists, while local critics, final
synthesis, deterministic financial authority, and publication controls remain under SignalForge
control.

SignalForge does not predict prices, execute trades, or provide personalized investment
recommendations.

## User Scenario And Value

The target user is an informed US or European individual investor researching US-listed
companies. The product helps that user:

- understand a company's business, products, history, and dependencies;
- inspect accounting and financial fundamentals with visible source boundaries;
- reason about macroeconomic transmission mechanisms;
- compare companies only when definitions and accounting perimeters permit it;
- explore valuation scenarios and monitoring conditions;
- learn finance through cited real-company evidence; and
- continue, retain, export, or delete local research under explicit control.

The governed scope contains 20 US-listed technology issuers and five peer lanes. Scope promotion
never implies universal comparability.

<!-- pagebreak -->

## Architecture

<!-- architecture -->

The React workspace presents a concise investor view and an optional judge view. A typed Go control
plane owns request parsing, planning, retrieval, tool permissions, agent scheduling, evidence
lineage, comparison authority, validation, memory, and release.

Models perform bounded interpretation and qualitative synthesis. They never become the source of
record for financial values or permissions.

## Agent Flow

1. The interpreter identifies the requested decision, entities, horizon, and constraints.
2. The planner creates a typed execution plan and specialist wave.
3. Retrieval selects point-in-time, source-authorized evidence with lineage.
4. Deterministic tools calculate financial and economic quantities.
5. Specialists produce bounded context packets in parallel.
6. Evidence and risk critics independently challenge support and limitations.
7. The local final analyst synthesizes only authorized material.
8. The Answer Contract Engine validates and deterministically constructs the released projection.

The expandable execution plan exposes these stages without forcing operational telemetry into the
default investor experience.

<!-- pagebreak -->

## Core Capabilities

SignalForge implements all five Track 2 capability families:

- **Local knowledge retrieval:** point-in-time SEC, macroeconomic, market, and official
  investor-relations evidence with resolvable citations.
- **Tool invocation:** a closed role-authorized registry of deterministic financial operations.
- **Multi-step planning:** typed decomposition, parallel specialist waves, independent critics,
  repair, synthesis, and release.
- **Local multi-turn memory:** governed follow-ups and opt-in SQLite case retention.
- **Permission and privacy controls:** read-only model authority, explicit writes, local
  inspect/export/delete, secret files, and protected telemetry.

### Numerical Silence

Financial values are transported as typed variables and deterministic receipts. Engines calculate
metrics, periods, units, signs, scenarios, and comparisons. Models receive qualitative relations
and approved variables but cannot promote unsupported prose into numerical authority.

### Evidence And Comparison Authority

Every material claim resolves to source evidence, a deterministic receipt, an explicit assumption,
or a limitation. Peer comparisons have metric-level dispositions:

- `comparable`;
- `comparable_with_caveat`;
- `not_comparable`; or
- `unavailable`.

An individually valid company metric does not automatically authorize a cross-company conclusion.

<!-- pagebreak -->

## AMD Radeon And ROCm

The selected local runtime uses:

- AMD Radeon `gfx1100` with 47.98 GiB VRAM;
- host ROCm 7.2.1;
- Gemma 4 26B A4B Instruct QAT Q4_0;
- AMD-validated ROCm `llama.cpp`;
- 32,768-token context;
- four continuous-batching specialist slots; and
- unified F16 KV cache with Flash Attention `auto`.

The model is hydrated separately after explicit license acceptance and verified by size and
SHA-256. The public application image contains no weights or credentials.

### Selection And Optimization

The bounded deployment study compared three measured Radeon application profiles:

- Gemma 4 26B A4B Instruct QAT Q4_0 through `llama.cpp`: `40/40` contract checks and
  `86.4601` median decode tokens/s;
- Qwen3 8B BF16 through `vLLM`: `40/40` contract checks and `26.3855` median decode tokens/s; and
- Granite 4.1 8B BF16 through `vLLM`: `35/40` contract checks and `24.9882` median decode
  tokens/s.

Gemma therefore combined complete contract compliance with `3.28x` Qwen's measured decode
throughput in this workload. After model selection, the four-slot Gemma profile was `29.17%`
faster end-to-end than the passing three-worker control.

These are application-profile measurements, not a universal ranking: models, runtimes, and
precision differed.

### Hybrid Radeon API

The optional API route receives bounded qualitative specialist packets. Local inference retains
critics, final synthesis, and release authority. Representative hybrid testing passed `5/5`, but
only two cases were faster than local-only, so the route remains selective.

Loss of the optional API recovered locally. Loss of indispensable local authority failed closed
with no answer release.

<!-- pagebreak -->

## Evaluation And Stability

The frozen evaluation completed:

- `180/180` standalone and peer, development and sealed journeys with runtime and contract pass;
- `180/180` accepted in independent model-assisted evidence-alignment review;
- 18 of those accepted with limitations and zero false-release candidates;
- `10/10` repeated financial-quality journeys;
- a 5 hour 28 minute soak with 1,945 telemetry samples;
- constant 32% allocated VRAM from first to last sample;
- 63 C maximum observed GPU junction temperature;
- a live Adobe standalone journey;
- a governed NVIDIA/AMD peer journey; and
- a fail-closed overbroad peer request.

Model-assisted review is decision support, not independent human ground truth or professional
assurance.

## Memory, Privacy, And Observability

Retention is off by default. When enabled, SignalForge stores only the released safe projection in
local SQLite. Users can inspect, export, and delete retained cases.

Mission Control correlates safe IDs, routes, tools, durations, receipts, failures, and aggregate GPU
telemetry. It excludes prompt bodies, answer bodies, source bodies, credentials, private memory,
chain-of-thought, and hidden reasoning.

Reference-only sources are linked rather than redistributed when rights are not established.
Restricted accounting publications, private authorial corpora, model weights, credentials, and raw
provider payloads are absent from the public repository and application image.

<!-- pagebreak -->

## Reproduction

Credential-free fixture:

```text
npm --prefix web ci
npm --prefix web run build
go run ./cmd/signalforge-workspace --mode fixture --static-dir web/dist
```

Complete repository gate:

```text
python3 -m pip install -r requirements-verify.txt
scripts/verify.sh
```

Fresh Radeon runtime:

```text
make radeon-bootstrap BACKEND=auto ACCEPT_GEMMA_LICENSE=yes
make radeon-up BACKEND=auto
```

## Honest Limitations

- Coverage is bounded to promoted company, metric, period, unit, and peer authorities.
- External answer accuracy has not been scored against independent human ground truth.
- Citation presence does not by itself prove semantic entailment.
- Whole-journey concurrency is bounded by the local 26B model.
- Some OneClick hosts require the native ROCm backend because of container mount policy.
- The application image is frozen, supply-chain verified, and read back on Radeon by immutable
  digest. Release media and the project-owner decision are hash-bound in
  `evidence/final-release-authority.json`.

SignalForge can make mistakes. Important information must be verified, and qualified professionals
should be consulted before financial decisions.

## Evidence

- Current evaluation: `evidence/championship-evaluation.json`
- Current Radeon runtime: `evidence/championship-radeon-runtime.json`
- Current product check: `evidence/championship-product-check.json`
- Model selection: `evidence/radeon-baseline.json`
- Runtime optimization: `evidence/radeon-optimization.json`
- Hardening: `evidence/hardening-matrix.json`
- Application release identity: `evidence/release-identity.json`
- Exact-image Radeon readback: `evidence/exact-image-radeon-readback.json`
- Synchronized Mission Control: `evidence/mission-control-runtime.json`
- Accounting authority: `docs/accounting-authority/technology20-accounting-professional-review.md`
- Final release authority and media hashes: `evidence/final-release-authority.json`

<!-- evidence-claim:current-product -->
<!-- evidence-claim:current-evaluation -->
<!-- evidence-claim:current-radeon-runtime -->
<!-- evidence-claim:accounting-authority -->
<!-- evidence-claim:release-identity -->
<!-- evidence-claim:privacy-and-rights -->
