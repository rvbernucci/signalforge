# SignalForge Contracts

SignalForge uses versioned JSON Schemas and matching Go validators to keep model output,
deterministic computation, evidence, orchestration, privacy, and product release as separate
authority boundaries. Unknown fields fail closed unless a contract explicitly permits them.

## Agent And Execution

- `execution-plan.schema.json` is the privacy-safe lifecycle shown in the Workspace.
- `execution-presentation.schema.json` derives bounded Summary, Details, and Proof views.
- `orchestration-trace.schema.json` records typed runtime transitions without prompt or answer
  bodies.
- `context-packet.schema.json` carries evidence-grounded specialist findings.
- `research-trace.schema.json` and `failure-receipt.schema.json` preserve observable success,
  degradation, and refusal.

Plans and traces describe observed execution. They cannot authorize evidence, calculations, or
answer release.

## Numerical Authority

- `engine-request.schema.json` carries role-authorized deterministic work.
- `calculation-receipt.schema.json` binds formula, inputs, outputs, validation, provenance, and
  replay identity.
- `metric-registry.schema.json` governs canonical financial definitions and applicability.
- `financial-intelligence-packet.schema.json` separates backend numerical authority from the
  value-free model view used under Numerical Silence.

Authoritative decimal values are JSON strings. A model may interpret receipt-backed results but
cannot create or modify them.

## Technology 20 Product Authority

- `company-activation.schema.json` and `company-research-profile.schema.json` define company
  availability.
- `peer-lane.schema.json` bounds allowed peer questions and metrics.
- `metric-comparability-request.schema.json` and
  `metric-comparability-receipt.schema.json` require period, unit, concept, perimeter, and
  accounting compatibility.
- `comparison-bundle.schema.json` releases only receipt-approved comparisons.
- `typed-abstention.schema.json` preserves useful refusals.
- `technology20-financial-summary.schema.json` separates authoritative and contextual outputs.
- `technology20-peer-evaluation.schema.json` distinguishes releasable, caveated, context-only,
  and withheld metrics.
- `accounting-professional-decision.schema.json` binds issuer-specific exceptions to a named,
  exact-scope decision.

The runtime cannot broaden a company, peer lane, taxonomy mapping, or accounting perimeter beyond
these records.

## Evidence And Retrieval

- `evidence-manifest.schema.json` records code, model, dataset, environment, and artifact identity.
- `evidence-chunk.schema.json` preserves point-in-time regulatory and investor-relations lineage.
- `retrieval-vector-fixture.schema.json` keeps reproducible vectors distinct from source
  authority.
- `investor-relations-source-map.schema.json` and the `ir-*` schemas govern discovery,
  collection, transformation, rights quarantine, and lineage.

Raw issuer files remain outside Git. Structured financial facts and calculation receipts stay
directly resolvable rather than being reduced to embeddings.

## Memory And Privacy

- `research-case.schema.json` governs opt-in retained cases.
- `execution-plan.schema.json` and `orchestration-trace.schema.json` intentionally exclude
  credentials, source bodies, prompts, answers, hidden reasoning, and private memory contents.
- Public evidence contracts retain only bounded metrics, hashes, reason codes, and explicit
  authority labels.

Operational replay is bounded. Retained user cases are independent artifacts with inspect,
export, and delete controls.

## Validation Rule

Go validation under `internal/contracts`, `internal/productscope`, `internal/comparability`, and
the deterministic engine is authoritative. Schemas make those boundaries portable to fixtures,
CI, and independent inspection. Evidence and confidence never substitute for one another:
material facts require authorized sources, material calculations require successful receipts,
and inferences require explicit assumptions.
