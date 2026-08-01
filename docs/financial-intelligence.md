# Financial Intelligence Authority

## Purpose

SignalForge separates semantic judgment from financial authority. Language models explain,
challenge, and synthesize; deterministic Go engines own calculations, periods, units, signs,
lineage, comparison boundaries, and release receipts.

## Numerical Silence

The runtime follows four rules:

1. A model-generated number is not authoritative.
2. A material value must resolve to an approved input, deterministic output, or explicit
   assumption.
3. A calculation must carry its operation, inputs, units, period, code identity, and output hash.
4. A comparison must pass metric-level accounting and temporal authority before release.

Models can receive normalized direction and mechanically derived relations such as `increasing`,
`decreasing`, `A > B`, or `margin compressed`. The final presentation rehydrates only approved
values from receipts.

## Deterministic Registry

The composed registry contains 80 role-authorized operations across:

- accounting and period normalization;
- margins, returns, leverage, liquidity, and cash conversion;
- free-cash-flow and reinvestment analysis;
- growth, common-size, trend, and decomposition analysis;
- valuation scenarios and sensitivity;
- macroeconomic transformations and transmission inputs;
- market statistics and risk measures; and
- comparison and monitoring boundaries.

The registry total is not a claim that every natural-language request can invoke every operation.
The planner can select only tools allowed for its role and supported by available evidence.

## Accounting Authority

The Technology 20 registry distinguishes:

- canonical US GAAP mappings;
- conditionally accepted issuer-specific aliases;
- context-only company-reported measures;
- rejected mappings; and
- unavailable inputs.

Canonical mappings retain precedence. Conditional aliases are bound to issuer, filing chain,
period, unit, dimension, label, and accounting perimeter. Context-only values cannot create
rankings or direct peer conclusions.

See the
[professional review](accounting-authority/technology20-accounting-professional-review.md).

## Fail-Closed Behavior

The engine refuses release when:

- a period or unit is incompatible;
- a required value is absent or stale;
- an amendment or filing chain is unresolved;
- a comparison perimeter is not authorized;
- a receipt hash or code identity does not match;
- a model invents an unapproved value; or
- an answer contract loses required evidence or limitations.

## Verification

```bash
python3 scripts/reference_finance.py
go test -race ./internal/finance ./internal/financialintelligence ./internal/engine
```

The public hardening and evaluation summaries are:

- `evidence/hardening-matrix.json`;
- `evidence/championship-evaluation.json`; and
- `evidence/championship-product-check.json`.

These artifacts prove bounded engineering behavior, not an audit opinion or universal factual
accuracy.
