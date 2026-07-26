# SignalForge Contracts

## Technology 20 activation

Sprint 32 adds fail-closed product-scope contracts:

- `company-activation.schema.json` records evidence-backed company or peer-lane state changes;
- `company-research-profile.schema.json` separates issuer identity, listed securities, metric
  availability, and activation authority;
- `peer-lane.schema.json` bounds the companies, questions, and metrics allowed in a comparison;
- `metric-comparability-request.schema.json` carries point-in-time metric operands;
- `metric-comparability-receipt.schema.json` records deterministic compatibility or refusal;
- `comparison-bundle.schema.json` permits release only for metrics with approved receipts; and
- `typed-abstention.schema.json` preserves a useful, machine-readable refusal;
- `standalone-journey-suite.schema.json` freezes balanced company-level development and holdout
  expectations without embedding answer text; and
- `peer-journey-suite.schema.json` freezes metric-level comparison behavior, including useful
  refusal and the prohibition on pair-level ranking.

The executable validators and content-hash helpers live in `internal/contracts/activation.go`,
`internal/productscope`, and `internal/comparability`. Models and UI components consume these
records but cannot promote their state or alter hash-bound authority references.

The current mainline adds two versioned financial-intelligence contracts:

- `metric-registry.schema.json` governs canonical financial definitions, formula ownership,
  periods, applicability, authority, and failure dispositions.
- `financial-intelligence-packet.schema.json` separates the authoritative backend numerical
  context from the value-free model view used under Numerical Silence.

Retrieval boundaries are also versioned: `evidence-chunk.schema.json` preserves point-in-time
regulatory and investor-relations lineage without pretending issuer material is an SEC filing.
The original `filing-chunk.schema.json` remains as a legacy Sprint 05 artifact, while
`retrieval-vector-fixture.schema.json` separates reproducible derived vectors
from source evidence. Structured financial facts and calculation receipts remain directly
resolvable and are not converted into opaque embedding-only records.

Official investor-relations discovery is bounded by
`investor-relations-source-map.schema.json`. The corresponding Go policy validates issuer,
allowlisted domains, document class, authority tier, timestamps, content identity, rights class,
and supersession before narrative material can enter retrieval. The bounded golden corpus is
portable through `investor-relations-document-manifest.schema.json`; raw issuer files stay outside
Git while their immutable identities remain testable.

The 20-company pipeline uses the newer `ir-source-registry-v2.schema.json`,
`ir-document-v2.schema.json`, and `ir-run-manifest-v2.schema.json` contracts for discovery,
collection, transformation, lineage, rights quarantine, and reproducible run identity.

SignalForge agents, engines, and evidence tooling communicate through versioned JSON contracts.
The canonical Go types and fail-closed validation rules currently live in
`internal/contracts`.

Version `signalforge/v1` establishes four boundaries:

- `ContextPacket`: evidence-grounded specialist findings for one research step;
- `EngineRequest`: an authorized request for a registered deterministic operation;
- `CalculationReceipt`: immutable numerical results, validation, provenance, and replay data;
- `ReceiptSupersession`: append-only linkage from a prior receipt to a corrected replacement;
- `EvidenceManifest`: the code, model, dataset, environment, and artifact identity behind a measured run.

Portable JSON Schemas currently cover:

- `execution-plan.schema.json`;
- `engine-request.schema.json`;
- `calculation-receipt.schema.json`;
- `failure-receipt.schema.json`;
- `evidence-manifest.schema.json`;
- `benchmark-row.schema.json`;
- `research-trace.schema.json`;
- `orchestration-trace.schema.json`;
- `safe-decision-replay.schema.json`;
- `golden-journey-scorecard.schema.json`;
- `golden-semantic-rubric.schema.json`;
- `golden-semantic-evaluation.schema.json`;
- `demo-evidence.schema.json`.

`execution-plan.schema.json` is the user-visible, privacy-safe projection of the accepted typed
plan and its lifecycle. It records bounded labels, role authority, dependencies, waves, route reason
codes, attempts, durations, checklist decisions, and safe artifact references. Prompt bodies,
model responses, credentials, raw source bodies, and hidden reasoning are not members of the
contract. The server owns transition validation, monotonic sequence handling, and the projection
hash; the browser renders this state but cannot authorize execution or release an answer. Validated
projections always contain the ordered parent phases `interpretation`, `planning`, `context`,
`tools`, `review`, `synthesis`, `memory`, and `release`. Steps bind to exactly one parent phase.
Parent phases summarize governed activity but are never counted as execution steps, so an empty or
embedded tools, memory, or release phase cannot inflate progress. The tools phase derives only from
observed deterministic executions, memory derives from explicit retention events, and release
derives from the authoritative run outcome.

Validated
specialist adapters may append real retrieval `started`, `passed`, `degraded`, or `failed` rows
using only retrieval, bundle, method, candidate-count, source-class, and as-of metadata. Providers
that cannot establish candidate counts report that telemetry as unavailable rather than inventing
it. Deterministic engines may append tool `started`, `passed`, or `failed` rows using only execution,
engine, operation, formula, input/output-reference, invariant, warning, and receipt metadata.
Precomputed receipts and capability authorization alone never create a runtime execution row.
Review and synthesis projections may add bounded counts and coverage ratios, but never claim bodies,
rejected content, raw answers, formula values, or model reasoning.

Model events retain a privacy-safe route class, observed attempt, and call kind only. The call kind
is one of `primary`, `retry`, `fallback`, or `bounded_repair`; it is derived from runtime-observed
route transitions, failures, output-budget changes, and non-persisted prompt fingerprints. Memory
events use the closed lifecycle `not_requested`, `requested`, `approved`, `saved`, `unavailable`,
`failed`, and `deleted`. Neither event family carries prompt bodies, response bodies, credentials,
case contents, or chain-of-thought.

Operational event replay is bounded to the latest 256 events per run, and the in-memory completed-run
window is bounded to 64 records. Active runs are never evicted. User-retained cases are independent
SQLite artifacts governed by the explicit memory lifecycle rather than this operational cache.

`orchestration-trace.schema.json` is the runtime state-machine trace. It records only bounded
identifiers, lifecycle events, artifact references, concurrency, and timestamps. Prompt text,
answers, credentials, tokens, and secret-shaped metadata are outside this contract by design.

`safe-decision-replay.schema.json` is the public, privacy-safe projection of a golden run. It keeps
route reason codes, content hashes, claim dispositions and their explicit assumption authority,
aggregate and per-role latency, token
counts, and an optional all-or-none Radeon runtime attestation. It deliberately excludes prompt
bodies, response bodies, chain-of-thought, source locators, and failure messages.

`golden-journey-scorecard.schema.json` separates measured runtime and release-integrity properties
from claims that have not been established. Its quality block reports disposition, authority,
review, evidence coverage, and a hash-bound frozen semantic rubric while explicitly recording that
external answer accuracy has not been scored against an independent ground truth. Passing the
frozen rubric establishes contract conformance, not universal factual accuracy.

All decimal values are JSON strings. Production Go validation remains
authoritative; the schemas make the boundary portable to fixtures, CI, and
independent consumers.

Material facts require primary-evidence references. Material calculations require successful,
replayable receipts. Inferences require both support and explicit assumptions. A confidence score
never replaces evidence.
