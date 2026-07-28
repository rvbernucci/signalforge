# Financial Intelligence Runtime

Status: integrated on `main`, verified on CPU, and exercised in accepted Radeon journeys  
Historical baseline: `v1.0.0` remains immutable  
Championship release: `v1.1.1`

## Purpose

Sprint 16B extends the frozen 28-operation Tier 0 engine with 52 governed financial-intelligence
operations. The extension turns point-in-time reported facts into inspectable cash-generation,
return, quality, capital-allocation, valuation, peer, and association evidence without delegating
arithmetic or numerical authorship to a language model.

The `v1.0.0` tag is unchanged and remains the reproducible historical baseline. The championship
runtime integrates this extension without rewriting the historical Tier 0 identity or evidence.

## Runtime Boundary

```text
reported fact + filing availability
        |
        v
typed period and sign normalization
        |
        v
canonical metric registry (80 active definitions)
        |
        v
role-authorized deterministic Go executor
        |
        v
hash-verifiable calculation receipts
        |
        +--> authoritative numerical context
        |
        +--> value-free model view
        |
        v
deterministic reference renderer
```

`Tier0Registry()` remains the frozen 28-operation surface. `FinancialIntelligenceRegistry()` owns
the 52 post-v1 additions. `RuntimeRegistry()` composes both without changing the historical Tier 0
identity.

All 80 registered operations have an explicit unit contract for every required input. The executor
also enforces currency and governed period compatibility before dispatch. Auxiliary cohort,
historical-band, debt-maturity, scenario, tornado, and diagnostic helpers are deterministic
composition surfaces; they cannot independently publish an authoritative value outside an owning
registered receipt and deterministic presentation path.

## Added Capability Groups

| Group | Operations | Primary controls |
|---|---:|---|
| Financial analysis | 30 | cash generation, returns, liquidity, debt maturity, typed cash conversion, payout, and sources-and-uses reconciliation |
| Valuation | 19 | CAPM, beta conventions, multi-stage DCF, dividend discount, reverse expectations, detailed value bridge, scenarios, tornado inputs, and governed multiples |
| Comparison | 2 | DuPont and robust peer statistics with governed cohorts and historical bands |
| Economics | 1 | lagged association with train/test, missingness, stability, and confidence diagnostics, explicitly labeled non-causal |

The generated canonical catalog is produced with:

```bash
go run ./cmd/signalforge-export-metric-registry
```

Every active definition records formula version, meaning, typed inputs, sign policy, period basis,
applicability, GAAP status, authority, implementation owner, and fail-closed dispositions.

## Period And Sign Authority

`internal/financialperiod` distinguishes reported facts, normalized facts, derived metrics,
estimates, and scenarios. It supports instant, quarter, YTD, annual, and TTM periods. TTM is built
only as prior annual plus current YTD less comparable prior YTD, with cutoff and lineage checks.

Sign normalization never overwrites source truth. A normalized fact retains the raw reported value,
the transformation ID, the selected sign policy, source fact IDs, amendment chain, availability
time, and computation time.

## Numerical Silence

`FinancialIntelligencePacket` contains two deliberately different views:

- `numerical_context` contains authoritative values derived from validated receipts;
- `model_view` contains only closed references, meanings, periods, methods, formula versions,
  evidence, receipt IDs, relations, and warnings.

The model view cannot contain a `value` property under the portable JSON Schema. Final numerical
release uses the existing deterministic presentation compiler. Models cannot alter receipt values
or introduce an authoritative number.

Role-scoped specialist packets preserve this boundary: they carry value-free receipt references
with operation, formula, period, status, assumptions, evidence IDs, and hashes, but never normalized
inputs, intermediate values, or numerical outputs. The Evidence Critic receives an independent
superset of those references rather than copied specialist prose.

## Fail-Closed Behavior

The runtime rejects unsupported operations, roles, inputs, units, currencies, periods,
future evidence, formula versions, invalid denominators, inapplicable company profiles, unresolved
reconciliations, non-convergent reverse valuations, small peer samples, zero-variance regressions,
and value-level mismatches.

Operating-company formulas explicitly exclude banks, insurers, and REITs where the definition is
structurally misleading. Sector-specific models require separate reviewed definitions.

## Verification

The financial-intelligence checks are reproducible from the repository root:

```bash
go test ./internal/capability ./internal/engine ./internal/finance \
  ./internal/financialintelligence ./internal/financialperiod ./internal/metricregistry
python3 scripts/reference_finance.py
go run ./cmd/signalforge-export-metric-registry >/tmp/signalforge-metric-registry.json
```

The complete `scripts/verify.sh` gate additionally validates the immutable evidence baseline,
exports all 80 canonical definitions through tests, confirms the 52-operation extension, runs the
full race-enabled Go suite, and checks complex methods through an independent Python
implementation.

Property and metamorphic tests cover beta leverage round trips, DCF monotonicity, capital-allocation
conservation, invalid rates and denominators, TTM look-ahead, common-size period compatibility, and
sign preservation. Fuzz seeds cover financial denominator and period normalization boundaries.

The first benchmark exposed avoidable registry reconstruction in each financial packet. Reusing the
immutable canonical registry reduced packet construction from approximately 582 KB and 5,021
allocations to 6.3 KB and 59 allocations on the development Mac. These are CPU development results,
not Radeon claims.

## Current Radeon Evidence Boundary

- The selected local Gemma runtime completed the accepted Sprint 36 local journey, including
  deterministic financial-engine receipts, independent review, final synthesis, and governed
  release.
- The accepted Sprint 36 hybrid journey used bounded Radeon API specialists while local ROCm
  retained review and final synthesis; rejected or failed remote packets remained subject to local
  fallback and the same deterministic contracts.
- The exact `v1.1.1` image was pulled anonymously and read back on Radeon Cloud before an
  exact-image hybrid journey.
- The public Sprint 33 tournament measures complete product-journey latency and contract success
  across frozen worker profiles. It does not claim independent factual accuracy or universal model
  throughput.
- The 240-case development set and restricted 80-case future holdout remain bounded evaluation
  assets; they cannot retroactively validate earlier changes or substitute for independent
  professional review.

Primary records:

- `evidence/sprint36-radeon-local-journey.json`
- `evidence/sprint36-radeon-hybrid-journey.json`
- `evidence/sprint36-exact-release-radeon-journey.json`
- `evidence/sprint36-radeon-resilience.json`
- `evidence/sprint33-latency-tournament.json`

The implementation and Radeon execution are demonstrated. External answer accuracy has not been
scored against independent human ground truth, and every performance claim remains bounded to its
recorded workload.
