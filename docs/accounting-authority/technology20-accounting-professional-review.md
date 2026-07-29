# Technology 20 Accounting Professional Review

This public-safe review artifact is documentation only and is not copied into the application
runtime.

Registry content SHA-256 (`registry_sha256`; self-field excluded): `1c40b44538eee8c64e066bbf224aae51ff45a05094c1c36a867d58b779973dd4`  
As of: `2026-07-24T10:32:53Z`

Blank fields in the machine-generated review packet are deliberate: generation cannot self-approve.
The separate professional decision record is reviewer-authored, hash-bound, and fail-closed.

## Review Status

- Technical research outcome: `CONDITIONALLY_SUPPORTED_AT_EXACT_SCOPE`
- Exact-scope population: `6 reviewed aliases`, `3 context-only mappings`
- Named professional decision: `CONDITIONALLY_ACCEPTED`
- Machine decision encoding: `HASH_BOUND_CONDITIONALLY_ACCEPTED`
- Decision record SHA-256: `13cdfadfcacc5d07f15d0ebae761442608d62ca758fb9a8cfa6177df3aefea2c`
- Runtime activation: `ACTIVE_AT_EXACT_SCOPE_FAIL_CLOSED`

The official filing evidence supports each proposed disposition only at its exact issuer,
period, unit, dimension, filing-chain, label, and accounting-perimeter boundary. Any broader
use described below is technically rejected.

| Company | Input | Concept | Perimeter | Technical outcome | Evidence | Named decision |
|---|---|---|---|---|---|---|
| sec-cik:0001018724 | capital_expenditure | `PaymentsToAcquireProductiveAssets` | `company_reported_property_and_equipment_cash_purchases` | Conditionally support exact issuer-specific alias; reject broader use | [Consolidated Statements of Cash Flows, Purchases of property and equipment](https://www.sec.gov/Archives/edgar/data/1018724/000101872426000004/amzn-20251231.htm): Purchases of property and equipment | **CONDITIONALLY ACCEPTED** |
| sec-cik:0000051143 | revenue | `Revenues` | `consolidated_periodic_filing` | Conditionally support exact issuer-specific alias; reject broader use | [2025 Annual Report, Consolidated Income Statement, Total revenue](https://www.sec.gov/Archives/edgar/data/51143/000005114326000010/ibm-20251231_d2.htm): Total revenue | **CONDITIONALLY ACCEPTED** |
| sec-cik:0001045810 | revenue | `Revenues` | `consolidated_periodic_filing` | Conditionally support exact issuer-specific alias; reject broader use | [Consolidated Statements of Income, Revenue](https://www.sec.gov/Archives/edgar/data/1045810/000104581026000021/nvda-20260125.htm): Revenue | **CONDITIONALLY ACCEPTED** |
| sec-cik:0000804328 | revenue | `Revenues` | `consolidated_periodic_filing` | Conditionally support exact issuer-specific alias; reject broader use | [Consolidated Statements of Operations, Total revenues](https://www.sec.gov/Archives/edgar/data/804328/000080432825000085/qcom-20250928.htm): Total revenues | **CONDITIONALLY ACCEPTED** |
| sec-cik:0001652044 | revenue | `Revenues` | `consolidated_periodic_filing` | Conditionally support exact issuer-specific alias; reject broader use | [Consolidated Statements of Income, Revenues](https://www.sec.gov/Archives/edgar/data/1652044/000165204426000018/goog-20251231.htm): Revenues | **CONDITIONALLY ACCEPTED** |
| sec-cik:0001045810 | capital_expenditure | `PaymentsToAcquireProductiveAssets` | `company_reported_property_equipment_and_intangible_assets` | Support contextual arithmetic only; reject canonical classification and comparative or ranking use | [Consolidated Statements of Cash Flows, investing activities](https://www.sec.gov/Archives/edgar/data/1045810/000104581026000021/nvda-20260125.htm): Purchases related to property and equipment and intangible assets | **CONDITIONALLY ACCEPTED** |
| sec-cik:0000796343 | revenue | `Revenues` | `consolidated_periodic_filing` | Conditionally support exact issuer-specific alias; reject broader use | [Consolidated Statements of Income, Total revenue](https://www.sec.gov/Archives/edgar/data/796343/000079634326000003/adbe-20251128.htm): Total revenue | **CONDITIONALLY ACCEPTED** |
| sec-cik:0001596532 | capital_expenditure | `PaymentsToAcquireProductiveAssets` | `company_reported_property_equipment_and_intangible_assets` | Support contextual arithmetic only; reject canonical classification and comparative or ranking use | [Consolidated Statements of Cash Flows, investing activities](https://www.sec.gov/Archives/edgar/data/1596532/000159653226000013/anet-20251231.htm): Purchases of property, equipment and intangible assets | **CONDITIONALLY ACCEPTED** |
| sec-cik:0000804328 | capital_expenditure | `PaymentsToAcquireProductiveAssets` | `company_reported_cash_capital_expenditures` | Support contextual arithmetic only; reject canonical classification and comparative or ranking use | [Consolidated Statements of Cash Flows, Capital expenditures](https://www.sec.gov/Archives/edgar/data/804328/000080432825000085/qcom-20250928.htm): Capital expenditures | **CONDITIONALLY ACCEPTED** |

## Exact-Scope Decision Boundary

- Revenue aliases are conditionally supported only for consolidated, dimensionless facts in the active filing
  chain. A valid canonical fact wins for the same issuer and period; an alias may bridge an issuer's
  documented taxonomy transition across periods.
- Amazon's alias is conditionally supported only as gross cash purchases of property and equipment. A derived
  result must be labeled `simple FCF`; it is not Amazon-reported FCF, net capex, FCFF, or unrestricted
  peer-comparable capex.
- Qualcomm, NVIDIA, and Arista are supported only for explicitly labeled contextual displays. Their arithmetic
  may be called a company-reported reinvestment intensity or residual-cash proxy, never canonical capex,
  `simple FCF`, FCFF, a winner, a ranking, or a direct relative conclusion.
- Every mapping fails closed if its filing chain, issuer language, dimensions, period, unit, currency, sign,
  or accounting perimeter changes.

## Named Reviewer Record

- Reviewer: `Rafael Bernucci`
- Qualification: `Project owner and Accounting graduate from the University of Sao Paulo; not acting as an independent auditor`
- Disposition: `conditionally_accepted`
- UTC timestamp: `2026-07-29T14:00:31Z`
- Scope: AR-37-01 through AR-37-09 at the exact registry content hash above
- Record locator: explicit declaration in the shared Codex task, recorded in the private Sprint 38 review
- Boundary: this is not an independent audit opinion, legal opinion, investment recommendation, or
  professional assurance engagement. Every use beyond the documented boundaries is rejected.

## Runtime Release Gates

1. COMPLETE: the named reviewer recorded qualification, exact registry content hash (`registry_sha256`;
   self-field excluded), item-level decisions, conditions, timestamp, and stable record locator.
2. COMPLETE: runtime selection is registry- and perimeter-aware, preserves canonical precedence by period,
   and carries exact per-input authority and product labels into every receipt.
3. COMPLETE: context-only outputs are mechanically excluded from scores, winners, rankings, and relative conclusions.
4. COMPLETE: company, 190-pair, mutation, Numerical Silence, and independent-reference gates passed on the
   post-decision candidate. Release identity remains separately governed.

The named decision and machine encoding are active only at the exact hash-bound scope above. Any registry,
decision, issuer, concept, period, unit, dimension, perimeter, filing-chain, or label mismatch fails closed.
