# AMD AI DevMaster Hackathon Track 2 Compliance

Review date: 27 July 2026  
Submission: SignalForge by SignalForge Labs  
Official pull request: [AMD-DEV-CONTEST/Radeon-hackathon-2026-07#16](https://github.com/AMD-DEV-CONTEST/Radeon-hackathon-2026-07/pull/16)

This document maps the published Track 2 rules to SignalForge evidence. It is a navigation aid, not
a scoring claim or a substitute for AMD's eligibility and judging decisions.

## Official Source Set

| Authority | Source | Use |
|---|---|---|
| Track rules and submission format | [Official hackathon README](https://github.com/AMD-DEV-CONTEST/Radeon-hackathon-2026-07/blob/main/README.md) | Track 2 task, platform, capabilities, 120-point criteria, deliverables, PR format, and English-language requirement |
| Radeon Cloud operation | [Official Radeon Cloud User Guide](https://github.com/AMD-DEV-CONTEST/Radeon-hackathon-2026-07/blob/main/Radeon-Cloud-User%20Guide/README.md) | Workspace, storage, SSH, shared APIs, dedicated vLLM APIs, and deployment behavior |
| Registration and legal terms | [AMD AI DevMaster Hackathon event page](https://luma.com/amd-4dhi) and its linked rules | Registration, eligibility, conduct, rights, prizes, privacy, and legal terms |
| Organizer clarification | AMD official Discord, Track 2 channel, 23-24 July 2026; recorded in the private operating brief | Final score is 120; both public AMD GPU Cloud endpoints and dedicated vLLM APIs are acceptable for the 20-point API path |

When sources differ, the event's linked legal terms govern eligibility and legal conditions, while
the current official repository README governs the technical submission format.

## Submission Format

| Official requirement | SignalForge disposition | Evidence |
|---|---|---|
| Fork the official repository and open a pull request | Complete | [Official PR #16](https://github.com/AMD-DEV-CONTEST/Radeon-hackathon-2026-07/pull/16) |
| PR title: `Track x, Team name, your application name` | Complete | `Track 2, SignalForge Labs, SignalForge` |
| Submission and project materials in English | Complete | README, source, PDF, deck, video, and public evidence are in English |
| Complete source repository | Complete | [Canonical source](https://github.com/rvbernucci/signalforge) and immutable forward `v1.1.1` tag |
| README with environment configuration, startup guide, and dependencies | Complete | [README](../README.md), especially Quick Start, Development, and Radeon reproduction |
| Project specification | Complete | [Six-page PDF](https://github.com/rvbernucci/signalforge/releases/download/sprint36-championship-v1/SignalForge-Project-Specification.pdf) |
| Architecture diagram | Complete | [Architecture](architecture.svg) |
| Demo video, recommended 3-5 minutes | Complete | [4 minute 45 second Radeon demo (284.970 seconds)](https://github.com/rvbernucci/signalforge/releases/download/sprint36-championship-v1/SignalForge-Radeon-Demo.mp4) |
| Actual operation from UI or CLI to final result on Radeon | Complete | Demo shows a real accepted hybrid Radeon journey, governed workflow, evidence, deterministic receipts, review, observability, and safe failure behavior |
| PPT or poster | Complete | [Six-slide deck](https://github.com/rvbernucci/signalforge/releases/download/sprint36-championship-v1/SignalForge-Judge-Deck.pptx) |

## Track 2 Technical Requirements

| Requirement | SignalForge implementation | Primary proof |
|---|---|---|
| Run on AMD Radeon Cloud and ROCm | Local Gemma inference on Radeon `gfx1100` and ROCm 7.2.1 | [Current local journey](../evidence/sprint36-radeon-local-journey.json) and [exact-release Radeon journey](../evidence/sprint36-exact-release-radeon-journey.json) |
| Core inference local on AMD Radeon | Local interpreter, specialists, critics, and final synthesis through loopback ROCm `llama.cpp` | Demo, [current local journey](../evidence/sprint36-radeon-local-journey.json), and [safe replay](../evidence/golden-safe-decision-replay.json) |
| No complete dependence on a closed-source agent platform | Typed Go orchestration and authority plane; React workspace; local open model runtime | [Architecture](architecture.svg) |
| Tool invocation and workflow orchestration | Closed role-authorized tool registry, deterministic receipts, bounded state machine | Project specification pages 3-4 |
| Operational stability and response performance | Typed failure behavior, bounded retries, startup checks, adversarial matrix, telemetry, exact-source bounded soak, post-soak sentinels, ROCm profiling, and network-isolated local inference | [Hardening matrix](../evidence/hardening-matrix.json), [resilience record](../evidence/sprint36-radeon-resilience.json), [exact-release attestation](../evidence/sprint36-release-attestation.json), and [vNext runtime resilience](../evidence/vnext-runtime-resilience.json) |
| Local knowledge retrieval | Point-in-time regulatory and official investor-relations evidence with citations | Project specification page 4 |
| Multi-step task planning | Typed interpreter, orchestrator, specialist waves, critics, and answer compiler | Demo `0:42-1:48` |
| Local multi-turn memory | Governed follow-ups and opt-in local case retention | Demo `2:52-3:12` |
| Permission and privacy controls | Read-only model authority, explicit writes, inspect/export/delete, private traces, secret rejection | Demo `0:22-0:42` and `2:52-3:12`; project specification pages 5-6 |

SignalForge implements all five listed capability families; the rule requires at least two.

## 120-Point Evidence Matrix

| Criterion | Points | Status | Evidence |
|---|---:|---|---|
| Clear task positioning and creative application scenario | 20 | Demonstrated | Investor research decision workspace; demo opening; specification pages 1-2 |
| Task decomposition, tools, RAG, and memory | 20 | Demonstrated | All five capability families; demo live journey; architecture and deterministic receipts |
| Smooth multi-turn interaction | 20 | Demonstrated | Governed follow-up, opt-in memory, inspect/export/delete, progressive execution plan |
| Core inference on AMD Radeon | 20 | Measured | Local Gemma route on `gfx1100`/ROCm 7.2.1 with [current local evidence](../evidence/sprint36-radeon-local-journey.json) and [exact-release Radeon evidence](../evidence/sprint36-exact-release-radeon-journey.json) |
| Targeted inference-speed optimization | 20 | Measured | A [three-profile tournament](../evidence/radeon-baseline.json) selected Gemma after it passed 40/40 deterministic contracts; the four-worker profile then passed 44/44 frozen checks in 157.47 seconds, 29.17% faster than the passing three-worker control; the [eight-journey-per-mode tournament](../evidence/sprint33-latency-tournament.json) measured `2.7777x` aggregate local speedup versus the two-worker baseline. Decode and journey results are bounded deployment-profile measurements, not single-stream dense-model microbenchmarks |
| Optional Radeon Cloud Model API path | 20 | Demonstrated | Complete accepted hybrid journey with `radeon-vllm`, local review/synthesis, and local fallback; [current evidence](../evidence/sprint36-radeon-hybrid-journey.json) and [resilience record](../evidence/sprint36-radeon-resilience.json) |

These statuses mean that the required evidence exists and is reviewable. They do not pre-award
points or replace the judges' qualitative assessment.

## Radeon API Boundary

The hybrid path sends only bounded qualitative context-specialist packets to the
organizer-provided OpenAI-compatible Radeon API. Go retains the plan, tools, evidence authority,
numerical authority, contracts, and final envelope. Independent review and final synthesis remain
on the local Radeon model, and a rejected remote packet is replayed locally.

Evidence:

- [Hybrid architecture and configuration](hybrid-vllm-specialists.md)
- [Current accepted hybrid journey](../evidence/sprint36-radeon-hybrid-journey.json)
- [Current accepted demo journey](../evidence/sprint36-radeon-demo-journey.json)
- [Recomputable public latency tournament](../evidence/sprint33-latency-tournament.json)
- [Correlated Workspace capture](assets/sprint36-live-hybrid-success.png)
- [Correlated Mission Control capture](assets/sprint36-live-hybrid-mission-control.png)
- [Current API-loss and model-loss record](../evidence/sprint36-radeon-resilience.json)

## Artifact Integrity

The forward championship release is `v1.1.1` and the public image tag is
`ghcr.io/rvbernucci/signalforge:v1.1.1`. Its immutable source is
`bc9c64746589e79766b2b18226ebb9d1d87d2585`, and its image index digest is
`sha256:cbac58cf3e62df0404e9ef1cfc7db6aec49e491e4beb5e1f214d6d562fad814b`.
Historical `v1.1.0` remains byte-identical and available.
The [release attestation](../evidence/sprint36-release-attestation.json) binds the source, image,
SBOM, provenance, vulnerability scan, anonymous pull, exact-image fixture, and exact-image hybrid
Radeon journey without relabeling the earlier video rehearsal.
<!-- evidence-claim:sprint36-exact-release -->

The image is public `linux/amd64`, runs as `10001:10001`, contains no credentials or model weights,
and includes a verified `/app/licenses` bundle. SBOM, provenance, Trivy HIGH/CRITICAL, anonymous
pull, and clean fixture execution are release-blocking gates.

## Rights, Conduct, and Responsible Scope

- Original SignalForge code and documentation are Apache-2.0 licensed.
- Third-party software, fonts, models, services, and data remain under their own terms.
- Model weights, restricted source bodies, private corpora, credentials, and raw private inference
  material are excluded from the application image and public artifacts.
- Reference-only sources are cited or linked rather than redistributed when redistribution rights
  are not established.
- The public content does not contain prohibited, discriminatory, defamatory, illegal, or
  inappropriate material.
- SignalForge is research software, not an audit, legal opinion, fiduciary service, personalized
  investment recommendation, trading system, or guarantee.

Participant eligibility remains subject to AMD verification, including Luma approval, AMD
Developer Program registration, legal identity, age, location, Discord and GitHub accounts, and
the event's complete terms.

## Known Limitations

- The deepest recorded and human-reviewed journey is bounded to Microsoft and NVIDIA.
- The broader Technology 20 universe is governed by explicit activation and abstention states.
- External answer accuracy has not been scored against independent human ground truth.
- The video is assembled from current, privacy-safe captures of a real accepted hybrid Radeon
  journey; it does not expose prompts, responses, source bodies, credentials, or private reasoning.
- Current Sprint 36 evidence is bound to the forward source candidate and separately identifies the
  rehearsed OCI image. The exact `v1.1.1` image has a separate public digest, supply-chain
  attestation, anonymous pull, clean fixture execution, and Radeon readback; no earlier native run
  is relabeled as exact-image execution.
- The vNext `ce4f2ca` soak completed 30/32 journeys; two repeated synthesis contracts failed
  closed. Median complete-journey latency was `32.023 s`, above the internal `30 s` target.
- Supported launch profiling and network-disabled local inference completed, but literal OCI
  recreation of the vNext public image was blocked by the Radeon OneClick host's mount policy.
- The vNext browser rehearsal was technical and agent-operated; independent investor, judge,
  keyboard, reduced-motion, factual-usefulness, and final-release acceptance remain open.
