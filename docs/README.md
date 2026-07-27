# Submission Documentation

This directory contains reviewable sources for the SignalForge judge package.

## Artifacts

- `architecture.svg`: primary scalable architecture diagram.
- `architecture.png`: deterministic raster used by the project PDF.
- `project-specification.md`: human-reviewable six-page project specification source, updated
  through Sprint 34.
- `demo-script.md`: verified 4 minute 12.9 second final cut sheet mapped to Track 2 evidence.
- `demo-voiceover.txt`: narration source used by the final local video artifact.
- `financial-intelligence.md`: current financial-intelligence architecture, numerical authority,
  CPU verification, and remaining Radeon evidence gates.
- `hybrid-vllm-specialists.md`: optional organizer-provided vLLM specialist path, runtime
  configuration, secret handling, fallback, and proof requirements.
- `../output/pdf/SignalForge-Project-Specification.pdf`: final six-page project documentation.
- `../output/presentation/SignalForge-Judge-Deck.pptx`: final six-slide supplemental deck.
- `../output/video/SignalForge-Radeon-Demo.mp4`: final local H.264/AAC demo video.
- `../evidence/runs/sprint13/live-demo-capture.json`: safe capture, runtime, QA, and hash record.

## Rebuild The PDF

Install the isolated documentation dependency, then build:

```bash
python3 -m pip install -r requirements-docs.txt
python3 scripts/build_project_spec.py
```

The production application does not import ReportLab. Documentation dependencies remain separate
from the runtime and verification dependency path.

The committed PDF and deck document the current Sprint 34 capability boundary while retaining
clearly labeled historical golden-run and optimization evidence. The PDF was visually inspected
after rendering all six pages with Poppler. It contains
no private prompts, credentials, raw model responses, source bodies, or chain-of-thought.

The PowerPoint deck was rendered through the presentation QA toolchain, visually inspected slide
by slide, passed template-fidelity validation, and passed the canvas-overflow gate. The Radeon demo
remains the 252.9-second Sprint 13 capture: it contains a real local run, source proof, a
deterministic receipt, a governed follow-up, memory controls, optimization evidence, and hardening
evidence. Sprint 34 adds separate hash-bound desktop/mobile Workspace captures and accepted local
and hybrid journey manifests; it does not misrepresent the earlier video as a new recording.
