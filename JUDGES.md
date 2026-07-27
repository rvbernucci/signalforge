# SignalForge Judge Guide

SignalForge is a private, local-first financial research desk for independent investors. It turns
public company evidence into an inspectable research process with specialist agents, deterministic
financial tools, citations, local memory, review gates, and explicit limitations.

This guide is the shortest path through the Track 2 submission. The complete requirement-to-proof
matrix is in [Track 2 Compliance](docs/track2-compliance.md).

## Recommended Review Path

1. Watch the [4 minute 12.9 second Radeon demo](https://github.com/rvbernucci/signalforge/releases/download/sprint34-artifacts-v1/SignalForge-Radeon-Demo.mp4).
2. Open the [six-slide judge deck](https://github.com/rvbernucci/signalforge/releases/download/sprint34-artifacts-v1/SignalForge-Judge-Deck.pptx).
3. Read the [six-page project specification](https://github.com/rvbernucci/signalforge/releases/download/sprint34-artifacts-v1/SignalForge-Project-Specification.pdf).
4. Inspect the [architecture](docs/architecture.svg), [Radeon runtime record](evidence/sprint34-radeon-runtime.json),
   [bounded latency tournament](evidence/sprint33-latency-tournament.json), and
   [release attestation](evidence/sprint34-release-attestation.json).
5. Reproduce the deterministic workspace or pull the exact public image.

## Questions To Try

- How would higher-for-longer rates affect Microsoft and NVIDIA?
- Which evidence weakens NVIDIA's investment thesis, and what should be monitored next?
- Is a Microsoft/Alphabet metric comparison authorized, caveated, or unavailable?

SignalForge may narrow or abstain when evidence, time boundaries, issuer activation, or peer
comparison authority are insufficient. That behavior is part of the product contract.

## What The Product Demonstrates

- **A concrete scenario:** evidence-grounded company research for a serious independent investor.
- **Local knowledge retrieval:** point-in-time SEC and official investor-relations evidence with
  resolvable citations and lineage.
- **Tool invocation:** 80 role-authorized deterministic financial operations with immutable
  receipts.
- **Multi-step planning:** a typed Go control plane with bounded specialist fan-out, independent
  review, and one final synthesis.
- **Local multi-turn memory:** governed follow-ups plus opt-in inspect, export, and delete controls.
- **Permission and privacy controls:** loopback-only local inference, read-only model authority,
  secret rejection, private traces, and fail-closed publication.
- **Local AMD inference:** Gemma 4 26B A4B Instruct QAT Q4_0 on Radeon `gfx1100`, ROCm 7.2.1, and
  ROCm `llama.cpp`.
- **Radeon API path:** bounded context specialists use the organizer-provided API while local Radeon
  review, synthesis, deterministic authority, and fallback remain under SignalForge control.

## Track 2 Score Map

The table maps every published criterion to the fastest human-review path. It identifies evidence;
the judges retain final scoring authority.

| Criterion | Judge-facing proof |
|---|---|
| Task positioning and application value - 20 | Demo `0:00-0:26`; project specification pages 1-2; deck slide 2 |
| Decomposition, tools, RAG, and memory - 20 | Demo `0:26-3:05`; project specification pages 3-4; deck slides 3-4 |
| Smooth multi-turn experience - 20 | Demo `3:05-3:47`; governed follow-up and memory controls |
| Core inference on AMD Radeon - 20 | Demo local run; [runtime identity and telemetry](evidence/sprint34-radeon-runtime.json) |
| Targeted ROCm optimization - 20 | Demo `3:47-4:00`; deck slide 5; [optimization evidence](evidence/radeon-optimization.json); [recomputable three-mode tournament](evidence/sprint33-latency-tournament.json) |
| Optional Radeon Cloud Model API bonus - 20 | [Accepted hybrid journey](evidence/dashboard-radeon-hybrid-journey.json), [recomputable tournament](evidence/sprint33-latency-tournament.json), [correlated capture](docs/assets/mission-control-radeon-hybrid-sprint34-viewport.jpg), and [failure recovery](evidence/runs/sprint34/failure-matrix.json) |

The recorded hybrid journey completed all 12 terminal steps across the eight governed phases. It
used both `radeon-vllm` and `local-rocm` under one run and trace identity. API loss exercised the
authorized local fallback; model or evidence loss remained fail-closed.

The separate Sprint 33 development tournament used eight public, non-sealed journeys per mode.
Local four-worker execution passed `8/8` contracts with `2.7777x` aggregate speedup versus the
two-worker baseline. Hybrid four-worker execution also passed `8/8`, accepted 20 Radeon API calls,
and recovered one failed remote call locally. These are bounded workload results, not universal
performance or factual-accuracy claims.

## Authority Boundary

Language models interpret evidence, propose qualitative claims, challenge support, and synthesize
a bounded semantic draft. Go owns identity, scope, evidence authorization, calculations, tool
permissions, lineage, numerical relations, contract validation, and final publication.

This is the core SignalForge safety idea: fluent model output is never the financial system of
record. Authoritative values and calculations enter the final answer only through deterministic,
hash-verifiable receipts.

## Exact Championship Release

| Item | Identity |
|---|---|
| Version | `v1.1.0` |
| Source commit | `032e9c38c4e74a450b38fec8341ed540b6339170` |
| Public image | `ghcr.io/rvbernucci/signalforge:v1.1.0` |
| Image index digest | `sha256:1354ccbbbd6138119111e23657ad69c1665f4189d75b9adcdecd53084870a4af` |
| Platform | `linux/amd64` |
| Runtime user | `10001:10001` |
| Release evidence | [Sprint 34 release attestation](evidence/sprint34-release-attestation.json) |

The application image contains no model weights, credentials, private corpora, or startup
downloads. The pinned Gemma artifact is staged separately on the Radeon host under its upstream
license.

## Fast Reproduction

The deterministic workspace proves the product contract without a GPU, API key, model download,
database setup, or external data call:

```bash
git clone https://github.com/rvbernucci/signalforge.git
cd signalforge
npm --prefix web ci
npm --prefix web run build
go run ./cmd/signalforge-workspace --mode fixture --static-dir web/dist
```

Open `http://127.0.0.1:8080`.

The exact image can be pulled by digest:

```bash
docker pull \
  ghcr.io/rvbernucci/signalforge@sha256:1354ccbbbd6138119111e23657ad69c1665f4189d75b9adcdecd53084870a4af
```

The selected Radeon model and runtime reproduction is documented in
[README: Reproduce The Selected Radeon Runtime](README.md#reproduce-the-selected-radeon-runtime).

## Honest Scope

- The deepest recorded and human-reviewed product journey compares Microsoft and NVIDIA.
- Twenty US technology companies are represented through governed activation states; unavailable or
  unreviewed directions remain fail-closed.
- Frozen semantic checks prove bounded contract conformance, not universal factual accuracy.
- SignalForge does not predict prices, execute trades, provide personalized investment
  recommendations, or replace professional judgment.
- The demo is the recorded local-only Radeon journey. The later hybrid Radeon API journey is
  separately hash-bound in the Sprint 34 evidence above and is not misrepresented as part of the
  earlier recording.
