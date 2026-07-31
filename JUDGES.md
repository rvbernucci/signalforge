# SignalForge Judge Guide

SignalForge is a private, local-first financial research desk for independent investors. It turns
public company evidence into an inspectable process with specialist agents, deterministic
financial tools, citations, local memory, independent review, and explicit limitations.

This is the shortest path through the Track 2 submission. The complete requirement-to-proof map is
in [Track 2 Compliance](docs/track2-compliance.md).

## Two-Minute Review Path

1. Watch the [4 minute 45 second Radeon demo (284.970 seconds)](https://github.com/rvbernucci/signalforge/releases/download/sprint36-championship-v1/SignalForge-Radeon-Demo.mp4).
2. Open the [six-slide judge deck](https://github.com/rvbernucci/signalforge/releases/download/sprint36-championship-v1/SignalForge-Judge-Deck.pptx).
3. Read the [six-page project specification](https://github.com/rvbernucci/signalforge/releases/download/sprint36-championship-v1/SignalForge-Project-Specification.pdf).
4. Inspect the [architecture](docs/architecture.svg), [current Radeon journey](evidence/sprint36-radeon-demo-journey.json),
   [bounded latency tournament](evidence/sprint33-latency-tournament.json), and
   [exact-release Radeon journey](evidence/sprint36-exact-release-radeon-journey.json).
5. Reproduce the credential-free fixture or pull the exact public image identified in the
   championship release section.

The frozen release architecture diagram labels the immutable 28-operation Tier 0 core. The current
runtime adds the 52-operation Financial Intelligence Registry for 80 role-authorized deterministic
operations in total.

## Questions To Try

- How would higher-for-longer rates affect Microsoft and NVIDIA?
- Which evidence weakens NVIDIA's investment thesis, and what should be monitored next?
- Is a Microsoft/Alphabet metric comparison authorized, caveated, or unavailable?

SignalForge narrows or abstains when evidence, time boundaries, issuer activation, or peer
comparison authority are insufficient. That behavior is part of the product contract.

## What Judges Can See

- **A concrete application:** evidence-grounded company research for a serious independent
  investor.
- **Multi-step planning:** an expandable execution plan shows interpretation, specialist waves,
  tools, independent critics, repair, synthesis, and release.
- **Local retrieval and tools:** point-in-time SEC and official investor-relations evidence,
  resolvable citations, and 80 role-authorized deterministic financial operations.
- **Local memory and privacy:** governed follow-ups plus opt-in inspect, export, and delete controls.
- **Local AMD inference:** Gemma 4 26B A4B Instruct QAT Q4_0 on Radeon `gfx1100`, ROCm 7.2.1, and
  ROCm `llama.cpp`.
- **Optional Radeon API path:** bounded context-specialist calls use the organizer-provided API
  while local Radeon critics, final synthesis, deterministic authority, and fallback remain under
  SignalForge control.
- **Correlated observability:** the Workspace, Proof Drawer, engines, and Mission Control share one
  privacy-safe `run_id` and `trace_id`.

## 120-Point Score Map

The judges retain final scoring authority. Each row below points to the fastest reviewable proof.

| Criterion | Judge-facing proof |
|---|---|
| Task positioning and application value - 20 | Demo `0:00-0:22`; specification pages 1-2; deck slide 2 |
| Decomposition, tools, RAG, and memory - 20 | Demo `0:22-3:12`; specification pages 3-4; deck slides 3-4 |
| Smooth multi-turn experience - 20 | Demo `2:52-3:12`; governed follow-up and local memory controls |
| Core inference on AMD Radeon - 20 | Demo `0:22-2:22`; [current local journey](evidence/sprint36-radeon-local-journey.json); [exact-release Radeon journey](evidence/sprint36-exact-release-radeon-journey.json) |
| Targeted ROCm optimization - 20 | Demo `3:12-3:38`; deck slide 5; [three-profile model selection](evidence/radeon-baseline.json); [optimization evidence](evidence/radeon-optimization.json); [three-mode tournament](evidence/sprint33-latency-tournament.json) |
| Optional Radeon Cloud Model API bonus - 20 | Demo `1:12-2:22` and `3:38-4:02`; [current hybrid journey](evidence/sprint36-radeon-hybrid-journey.json), [current demo journey](evidence/sprint36-radeon-demo-journey.json), and [failure recovery](evidence/sprint36-radeon-resilience.json) |

The frozen Radeon model tournament compared three hash-pinned deployment profiles. Gemma 4 26B
A4B QAT Q4_0 on `llama.cpp` passed `40/40` deterministic contract checks at `86.46` median decode
tokens/s; Qwen3 8B BF16 on vLLM passed `40/40` at `26.39`; Granite 4.1 8B BF16 on vLLM passed
`35/40` at `24.99`. Google's
[upstream model card](https://huggingface.co/google/gemma-4-26B-A4B-it-qat-q4_0-gguf/blob/d1c082be9cf3c8a514acf63b8761f4b41935842e/README.md)
reports `25.2B` total and `3.8B` active parameters for the selected Gemma MoE. Because model,
runtime, and precision differed, this establishes a bounded application deployment decision, not
a universal model ranking or dense-model microbenchmark.

The Sprint 33 development tournament used eight public, non-sealed journeys per mode. Local
four-worker execution passed `8/8` contracts with `2.7777x` aggregate speedup versus the recorded
two-worker baseline. Hybrid four-worker execution also passed `8/8`, accepted 20 Radeon API calls,
and recovered one failed remote call locally. These are bounded workload results, not universal
performance or factual-accuracy claims.

The current Sprint 36 Radeon run released a complete hybrid answer after 52 public-safe timeline
events, six context packets, 18 deterministic engine calls, five review events, and both local
ROCm and Radeon API inference. The public record excludes prompts, responses, source bodies,
credentials, memory, and private reasoning.
<!-- evidence-claim:sprint36-championship-journey -->

The exact `v1.1.1` image was then pulled anonymously, read back on Radeon Cloud, and used for both
the fixture and a complete hybrid journey. The exact-image record keeps this verification separate
from the pre-freeze video rehearsal and documents that two remote failures recovered before answer
release.
<!-- evidence-claim:sprint36-exact-release -->

## Authority Boundary

Language models interpret evidence, propose qualitative claims, challenge support, and synthesize
a bounded semantic draft. Go owns identity, scope, evidence authorization, calculations, tool
permissions, lineage, numerical relations, contract validation, and publication.

Fluent model output is never the financial system of record. Authoritative values and calculations
enter the final answer only through deterministic, hash-verifiable receipts.

## Championship Release

The forward championship release is `v1.1.1`, frozen at source
`bc9c64746589e79766b2b18226ebb9d1d87d2585` and public image index
`sha256:cbac58cf3e62df0404e9ef1cfc7db6aec49e491e4beb5e1f214d6d562fad814b`.
Historical `v1.1.0` remains available and byte-identical as the rollback.

| Item | Required release property |
|---|---|
| Version | `v1.1.1` |
| Source | `bc9c64746589e79766b2b18226ebb9d1d87d2585` |
| Public image | `ghcr.io/rvbernucci/signalforge:v1.1.1` |
| Image index | `sha256:cbac58cf3e62df0404e9ef1cfc7db6aec49e491e4beb5e1f214d6d562fad814b` |
| Platform | `linux/amd64` |
| Runtime user | `10001:10001` |
| Runtime notices | `/app/licenses` |
| Artifact release | [`sprint36-championship-v1`](https://github.com/rvbernucci/signalforge/releases/tag/sprint36-championship-v1) |
| Artifact hashes | [Judge package](evidence/judge-package.json) |
| Release proof | [Sprint 36 attestation](evidence/sprint36-release-attestation.json) |

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

Open `http://127.0.0.1:8080`. The selected Radeon model and runtime reproduction is documented in
[README: Reproduce The Selected Radeon Runtime](README.md#reproduce-the-selected-radeon-runtime).

## Honest Scope

- The deepest recorded product journey compares Microsoft and NVIDIA.
- The current `main` branch promotes twenty US technology company authorities and five bounded
  peer lanes from 180/180 contract- and runtime-passing development and sealed journeys. This
  Technology 20 candidate is not yet relabelled as the frozen championship release, and
  unavailable, context-only, or non-comparable directions remain fail-closed.
- The exact vNext source `ce4f2ca` admits one complete local research journey at a time while
  retaining up to four specialist calls inside that journey. Two representative journeys submitted
  together both completed in a controlled Radeon replay. Their elapsed times were `98.116 s` and
  `152.149 s` including queue time, so this is bounded resilience evidence, not a 30-second latency
  claim.
- The 80-operation total is the composed deterministic registry, not a claim that the current
  natural-language planner automatically selects every registered operation.
- In the frozen frontend, aggregate header/sidebar labels can say `Local inference` or `local calls`
  during a hybrid journey. The expandable execution plan and Radeon Mission Control are the
  route-specific authorities and distinguish Radeon API from local ROCm calls.
- Frozen semantic checks prove bounded contract conformance, not universal factual accuracy.
- SignalForge does not predict prices, execute trades, provide personalized investment
  recommendations, or replace professional judgment.
