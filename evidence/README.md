# Current Public Evidence

This directory contains only small, privacy-safe evidence needed to understand and verify the
current SignalForge release. Raw runs, prompts, answers, source bodies, credentials, hidden
reasoning, sealed labels, screenshots, media, and superseded experiment versions are intentionally
excluded.

## Judge-Facing Evidence

- [`championship-evaluation.json`](championship-evaluation.json) aggregates the current 180-case
  runtime, contract, semantic-review, and financial-reliability evaluation.
- [`championship-radeon-runtime.json`](championship-radeon-runtime.json) records the selected
  Radeon model/runtime profile, hybrid boundary, failure behavior, and 5h28 soak result.
- [`championship-product-check.json`](championship-product-check.json) records bounded,
  agent-operated acceptance checks for the Workspace and Mission Control.
- [`judge-package.json`](judge-package.json) hash-binds the current public judge documents and
  aggregate evidence.
- [`public-claims.json`](public-claims.json) binds public claims to exact repository artifacts.
- [`release-identity.json`](release-identity.json) freezes the verified public application image,
  source commit, platform manifest, CI, and supply-chain checks.
- [`release-checklist.json`](release-checklist.json) separates completed technical gates from
  the completed media and project-owner authority gates.
- [`final-release-authority.json`](final-release-authority.json) binds the final media URLs and
  hashes to the project-owner submission decision and its limitations.
- [`exact-image-radeon-readback.json`](exact-image-radeon-readback.json) records the clean OCI
  pull, immutable manifest verification, and bounded execution of the unchanged entrypoint payload
  on the Radeon host.
- [`mission-control-runtime.json`](mission-control-runtime.json) records the sanitized synchronized
  observability run and telemetry-loss behavior against the immutable image.

These artifacts prove only their stated scope. They do not reveal private evaluation cases, grant
independent professional assurance, pre-award points, or replace organizer judging.

## Reproducible Technical Evidence

- [`architecture-eval.json`](architecture-eval.json) and
  [`orchestration-eval.json`](orchestration-eval.json) cover typed planning and orchestration.
- [`radeon-baseline.json`](radeon-baseline.json) records the bounded three-profile model/runtime
  comparison.
- [`radeon-optimization.json`](radeon-optimization.json) records the selected ROCm configuration
  and rejected alternatives.
- [`hardening-matrix.json`](hardening-matrix.json) records the current adversarial gates.
- [`technology20-promotion-manifest.json`](technology20-promotion-manifest.json) binds the active
  20-company and five-peer-lane product authority.
- [`workspace-evaluation.json`](workspace-evaluation.json) covers the deterministic fixture
  experience.

## Verification

Run the complete source-level gate:

```bash
python3 -m pip install --requirement requirements-verify.txt
scripts/verify.sh
```

Validate only public claim hashes:

```bash
go run ./cmd/signalforge-release-check \
  --root . \
  --claims evidence/public-claims.json
```

The public repository is a product and judging surface, not an experiment archive. Intermediate
runs and superseded evidence remain outside this repository.
