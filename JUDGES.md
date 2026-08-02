# SignalForge Judge Guide

SignalForge is a private, local-first financial research workspace for independent investors. It
combines multi-agent planning, evidence retrieval, deterministic financial tools, local memory,
independent review, and explicit release contracts on AMD Radeon and ROCm.

This document is the shortest route through the current Track 2 release. It contains no
development chronology.

## Three-Minute Review

1. Watch the [4 min 26 s Radeon demo](https://github.com/rvbernucci/signalforge/releases/download/v1.2.1/SignalForge-Radeon-Demo.mp4).
2. Read the [project specification PDF](https://github.com/rvbernucci/signalforge/releases/download/v1.2.1/SignalForge-Project-Specification.pdf)
   or its [reviewable Markdown source](docs/project-specification.md).
3. Inspect the [judge deck](https://github.com/rvbernucci/signalforge/releases/download/v1.2.1/SignalForge-Judge-Deck.pptx)
   and [architecture](docs/architecture.svg).
4. Review the current [evaluation](evidence/championship-evaluation.json),
   [Radeon runtime](evidence/championship-radeon-runtime.json), and
   [product check](evidence/championship-product-check.json).
5. Run the credential-free fixture:

```bash
npm --prefix web ci
npm --prefix web run build
go run ./cmd/signalforge-workspace --mode fixture --static-dir web/dist
```

Open `http://127.0.0.1:8080/?audience=judge`.

The current application image and supply-chain checks are frozen in
[`release-identity.json`](evidence/release-identity.json). The image is public, immutable,
`linux/amd64`, SBOM- and provenance-attested, vulnerability-scanned, clean-pulled, and
exact-image fixture-tested. A clean OCI pull and bounded execution of the unchanged entrypoint
payload also passed on the Radeon host; see the
[readback receipt](evidence/exact-image-radeon-readback.json). The separate
[release checklist](evidence/release-checklist.json) records the completed gates, and the
[final release authority](evidence/final-release-authority.json) binds the media hashes and
project-owner decision. The synchronized
[Mission Control runtime](evidence/mission-control-runtime.json) passed against this exact image
and preserved answer completion after deliberate telemetry loss.

## What To Inspect

- **Investor experience:** a clean research answer, cited evidence, calculations, caveats, and
  governed follow-ups.
- **Expandable plan:** interpretation, specialist waves, tool calls, critics, repair, synthesis,
  and release.
- **Deterministic authority:** financial values and calculations resolve to typed receipts instead
  of model-generated numbers.
- **Evidence authority:** every released claim has authorized evidence, deterministic receipts,
  assumptions, or an explicit limitation.
- **Local memory:** retention is off by default; inspect, export, and delete require user action.
- **Mission Control:** optional route, tool, latency, GPU, failure, and lineage telemetry is
  correlated without recording model or source bodies.
- **Fail-closed behavior:** unsupported comparisons, invalid contracts, and missing local authority
  do not release partial answers.

## 120-Point Map

The judges retain complete scoring authority. `Implemented` and `Measured` below describe available
evidence; they do not pre-award points.

| Track 2 criterion | Status | Fastest proof |
|---|---|---|
| Clear task positioning and creative scenario - 20 | Implemented | Investor research workspace; [specification](docs/project-specification.md) |
| Decomposition, tools, RAG, and memory - 20 | Implemented | Expandable plan; [architecture](docs/architecture.svg); deterministic receipts |
| Smooth multi-turn interaction - 20 | Implemented | Governed follow-ups; opt-in inspect/export/delete; concise investor view |
| Core inference on AMD Radeon - 20 | Measured | Gemma 4 26B Q4_0 on `gfx1100`/ROCm 7.2.1; [runtime evidence](evidence/championship-radeon-runtime.json) |
| Targeted Radeon/ROCm optimization - 20 | Measured | Three-profile selection, four-slot tuning, 5h28 soak; [baseline](evidence/radeon-baseline.json) and [runtime evidence](evidence/championship-radeon-runtime.json) |
| Optional Radeon Cloud Model API - 20 | Measured | Selective specialist route with local authority and tested fallback; [hybrid evidence](evidence/championship-radeon-runtime.json) |

SignalForge implements all five capability families listed by the rules; the minimum is two.

## Bounded Radeon Model Selection

| Profile | Runtime and precision | Contract checks | Median decode tokens/s |
|---|---|---:|---:|
| Gemma 4 26B A4B Instruct | ROCm `llama.cpp`, QAT Q4_0 | `40/40` | `86.4601` |
| Qwen3 8B | ROCm `vLLM`, BF16 | `40/40` | `26.3855` |
| Granite 4.1 8B | ROCm `vLLM`, BF16 | `35/40` | `24.9882` |

Gemma was selected because it combined full contract compliance with `3.28x` Qwen's measured
decode throughput in this SignalForge workload. The profiles differ in model, runtime, and
precision; this evidence supports a bounded deployment decision, not a universal model ranking.
See the [baseline evidence](evidence/radeon-baseline.json).

## Current Results

| Evidence | Result |
|---|---:|
| Full four-population Radeon journeys | `180/180` runtime and contract pass |
| Model-assisted evidence-alignment review | `180/180` accepted; 18 with limitations |
| False-release candidates | `0` |
| Numerical faithfulness mean | `4.000/4` |
| Citation authority mean | `4.000/4` |
| Factual support mean | `3.989/4` |
| Accounting boundary mean | `3.978/4` |
| Repeated financial-quality journey | `10/10` |
| Representative hybrid journey | `5/5`, retained selectively |
| Radeon soak | `5h28m`, 1,945 samples, zero observed VRAM growth |
| Bounded product checks | Adobe standalone and NVIDIA/AMD peer completed |
| Unsupported peer request | Failed closed, no answer released |

The semantic review used an independent model as decision support. It is not human ground truth,
professional assurance, or judging authority.

## Why The Architecture Matters

The language model does not own the financial system:

- Go interprets the request into a typed plan.
- Retrieval is point-in-time and source-authorized.
- Tools are role-scoped and deterministic.
- Models receive bounded context packets, not the application envelope.
- Independent critics evaluate evidence and risk.
- Final local synthesis remains subject to the Answer Contract Engine.
- The application deterministically constructs transport and presentation structures.

This separation reduces model formatting burden, protects numerical fidelity, and makes every
released result inspectable.

## Radeon Boundary

Core interpretation, review, final synthesis, and release authority run locally on Radeon through
ROCm. The optional organizer-provided API can serve selected context specialists. It never receives
credentials, private memory, raw source corpora, or authority to publish.

When the optional API failed, the tested journey completed through local fallback. When the
indispensable local model was absent, the system failed closed even though remote specialists were
available.

## Reproduce

Full repository gate:

```bash
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install -r requirements-verify.txt
scripts/verify.sh
```

Exact `linux/amd64` fixture image gate:

```bash
scripts/verify_container_fixture.sh
```

Fresh Radeon workspace:

```bash
make radeon-bootstrap \
  BACKEND=auto \
  ACCEPT_GEMMA_LICENSE=yes
make radeon-up \
  BACKEND=auto
```

The [operator guide](docs/radeon-zero-touch-appliance.md) documents model hydration, persistent
storage, secret files, profiles, fallback, observability, and safe cleanup.

## Honest Scope

- The product universe is 20 US-listed technology companies and five promoted peer lanes.
- Authority remains metric-, period-, unit-, definition-, and accounting-perimeter-specific.
- SignalForge does not predict prices, execute trades, or give personalized recommendations.
- Public evidence contains aggregates and hashes only, never prompts, answers, source bodies,
  private traces, credentials, hidden reasoning, or sealed labels.
- Independent professional assurance and final judging remain external to the project.
- The application image identity, Radeon readback, release media, and project-owner authority are
  frozen and hash-bound; this does not pre-award points or replace organizer review.

<!-- evidence-claim:current-product -->
<!-- evidence-claim:current-evaluation -->
<!-- evidence-claim:current-radeon-runtime -->
<!-- evidence-claim:accounting-authority -->
<!-- evidence-claim:release-identity -->
<!-- evidence-claim:privacy-and-rights -->
