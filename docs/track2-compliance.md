# AMD AI DevMaster Hackathon Track 2 Compliance

Submission: SignalForge by SignalForge Labs  
Official pull request:
[AMD-DEV-CONTEST/Radeon-hackathon-2026-07#16](https://github.com/AMD-DEV-CONTEST/Radeon-hackathon-2026-07/pull/16)

This document maps the current official Track 2 requirements to reviewable SignalForge evidence.
It is a navigation aid, not a scoring claim.

## Official Sources

| Authority | Source |
|---|---|
| Track rules and submission format | [Official hackathon README](https://github.com/AMD-DEV-CONTEST/Radeon-hackathon-2026-07/blob/main/README.md) |
| Radeon Cloud operation | [Radeon Cloud User Guide](https://github.com/AMD-DEV-CONTEST/Radeon-hackathon-2026-07/blob/main/Radeon-Cloud-User%20Guide/README.md) |
| Registration and legal terms | [AMD AI DevMaster event](https://luma.com/amd-4dhi) and linked terms |

Organizer clarification in the official Track 2 Discord channel confirmed a 120-point total and
that both public AMD GPU Cloud endpoints and dedicated vLLM APIs are acceptable for the optional
20-point API path.

## Submission Requirements

| Requirement | Current disposition |
|---|---|
| Fork official repository and open PR | Complete: PR #16 |
| PR title format | Complete: `Track 2, SignalForge Labs, SignalForge` |
| English source and materials | Complete |
| Complete source and README | Complete |
| Project specification | Current Markdown source; final PDF URL and hash remain a media gate |
| Architecture diagram | [`architecture.svg`](architecture.svg) |
| 3-5 minute demo video | Final video URL and hash remain a media gate |
| PPT or poster | Final deck URL and hash remain a media gate |

Moving media binaries are not retained on `main`; the final judge package binds their release URLs
and SHA-256 values after freeze.

## Technical Requirements

| Requirement | SignalForge proof |
|---|---|
| AMD Radeon Cloud and ROCm | Gemma 4 26B Q4_0 on `gfx1100`, ROCm 7.2.1; [`championship-radeon-runtime.json`](../evidence/championship-radeon-runtime.json) |
| Core inference local | Local interpretation, critics, final synthesis, and release authority |
| Not dependent entirely on closed-source agent platform | Typed Go control plane, React workspace, local open-weight model |
| Tool invocation | Closed role-authorized deterministic registry |
| Workflow orchestration | Typed plan, specialist waves, critics, repair, synthesis, release |
| Operational stability and response performance | 180/180 population pass, 5h28 soak, failure matrix |
| Local knowledge retrieval | Point-in-time authority, citation lineage, hybrid retrieval |
| Multi-step planning | Expandable governed execution plan |
| Local multi-turn memory | Opt-in SQLite retention with inspect/export/delete |
| Permission and privacy | Read-only model authority, explicit writes, secret files, protected telemetry |

SignalForge implements all five listed capability families; the rule requires at least two.

## 120-Point Matrix

| Criterion | Points | Evidence status | Primary evidence |
|---|---:|---|---|
| Task positioning and application value | 20 | Implemented | Investor research workspace and [specification](project-specification.md) |
| Decomposition, tools, RAG, and memory | 20 | Implemented | [Architecture](architecture.svg), plan, receipts, case controls |
| Smooth multi-turn experience | 20 | Implemented | Governed follow-up and optional local memory |
| Core inference on AMD Radeon | 20 | Measured | [Current Radeon runtime](../evidence/championship-radeon-runtime.json) |
| Targeted Radeon/ROCm optimization | 20 | Measured | [Model selection](../evidence/radeon-baseline.json), [runtime profile](../evidence/radeon-optimization.json), current soak |
| Optional Radeon Cloud Model API | 20 | Measured | Selective hybrid route and local fallback in [current runtime evidence](../evidence/championship-radeon-runtime.json) |

## Rights And Responsible Scope

- SignalForge original source and documentation are Apache-2.0 licensed.
- Third-party software, fonts, models, services, and data retain their own terms.
- Reference-only source bodies are linked rather than republished.
- Restricted accounting publications, private corpora, model weights, credentials, prompts,
  answers, raw provider payloads, private traces, and hidden reasoning are excluded.
- SignalForge is not an audit, legal opinion, investment recommendation, fiduciary service,
  trading system, or guarantee.

## Artifact Integrity

The release workflow built the current `linux/amd64` image, attached SBOM and provenance, found
zero HIGH/CRITICAL vulnerabilities, verified a clean public pull, and ran the exact image fixture.
The immutable image and source identity are recorded in
[`release-identity.json`](../evidence/release-identity.json) and the Radeon appliance manifest.
Final media URLs and hashes remain a separate gate in
[`release-checklist.json`](../evidence/release-checklist.json).

## Known Limitations

- Authority is bounded by company, metric, period, unit, definition, and accounting perimeter.
- External answer accuracy has not been scored against independent human ground truth.
- Model-assisted semantic review is not professional or final authority.
- Whole-journey concurrency is bounded by the local 26B model.
- Human investor and judge-readability acceptance remain external gates.

<!-- evidence-claim:current-product -->
<!-- evidence-claim:current-evaluation -->
<!-- evidence-claim:current-radeon-runtime -->
<!-- evidence-claim:accounting-authority -->
<!-- evidence-claim:release-identity -->
<!-- evidence-claim:privacy-and-rights -->
