# Private Sprint 16B Financial Intelligence

Status: implemented and verified on CPU in the private Sprint 16B branch  
Promotion: not authorized  
Radeon journey evidence: not yet collected

## Purpose

Sprint 16B extends the frozen 28-operation Tier 0 engine with 34 governed financial-intelligence
operations. The extension turns point-in-time reported facts into inspectable cash-generation,
return, quality, capital-allocation, valuation, peer, and association evidence without delegating
arithmetic or numerical authorship to a language model.

The public `v1.0.0` release is unchanged. This document describes only
`exp/sprint16b-financial-intelligence`.

## Runtime Boundary

```text
reported fact + filing availability
        |
        v
typed period and sign normalization
        |
        v
canonical metric registry (62 active definitions)
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
the 34 private additions. `RuntimeRegistry()` composes both without changing the historical Tier 0
identity.

## Added Capability Groups

| Group | Operations | Primary controls |
|---|---:|---|
| Cash generation and returns | 14 | explicit tax assumptions, two-way invested-capital reconciliation, positive denominator policies, aligned periods |
| Financial quality and allocation | 9 | typed margins, accruals, liquidity, leverage, shareholder yield, sources-and-uses reconciliation |
| Valuation | 8 | CAPM, beta leverage conventions, multi-stage DCF, terminal-method separation, reverse expectations, value-level-safe multiples |
| Comparison and economics | 3 | DuPont, robust peer statistics, lagged association explicitly labeled non-causal |

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

## Fail-Closed Behavior

The private runtime rejects unsupported operations, roles, inputs, units, currencies, periods,
future evidence, formula versions, invalid denominators, inapplicable company profiles, unresolved
reconciliations, non-convergent reverse valuations, small peer samples, zero-variance regressions,
and value-level mismatches.

Operating-company formulas explicitly exclude banks, insurers, and REITs where the definition is
structurally misleading. Sector-specific models require separate reviewed definitions.

## Verification

The private audit is reproducible from the Sprint root:

```bash
python3 tools/audit_financial_intelligence.py
```

It verifies the immutable public boundary, exports all 62 canonical definitions, confirms the 34
operation extension, runs the focused Go suite, runs five CPU benchmarks, and checks 12 complex
methods through an independent Python implementation.

Property and metamorphic tests cover beta leverage round trips, DCF monotonicity, capital-allocation
conservation, invalid rates and denominators, TTM look-ahead, common-size period compatibility, and
sign preservation. Fuzz seeds cover financial denominator and period normalization boundaries.

The first benchmark exposed avoidable registry reconstruction in each financial packet. Reusing the
immutable canonical registry reduced packet construction from approximately 582 KB and 5,021
allocations to 6.3 KB and 59 allocations on the development Mac. These are CPU development results,
not Radeon claims.

## Remaining Promotion Gates

- Freeze and execute a separate financial-intelligence holdout without tuning against it.
- Run complete Accounting, Financial Quality, Valuation, Economics, Market Behavior, critic, and
  final-analysis journeys on the selected local Radeon runtime.
- Measure factual correctness, abstention, evidence coverage, latency, VRAM, and complete-task
  success separately.
- Add product-facing comparison, valuation, capital-allocation, missing-data, and not-applicable
  golden journeys.
- Record SG-08 before proposing a reviewed release newer than `v1.0.0`.

Until those gates pass, this work is a verified private CPU implementation, not a public product or
Radeon performance claim.
