# Hybrid Radeon API Specialists

## Boundary

The `championship` profile combines:

- local Gemma inference on AMD Radeon and ROCm for interpretation, critics, final synthesis, and
  release authority; and
- optional organizer-provided OpenAI-compatible Radeon API calls for selected qualitative context
  specialists.

The API path cannot calculate authoritative financial values, mutate local memory, broaden product
scope, approve its own claims, or publish an answer.

## Configuration

Non-secret configuration:

```text
SIGNALFORGE_SPECIALIST_API_ENABLED=true
SIGNALFORGE_SPECIALIST_API_PROVIDER=radeon-vllm
SIGNALFORGE_SPECIALIST_API_BASE_URL=https://radeon.anruicloud.com/api/v1
SIGNALFORGE_SPECIALIST_TEXT_MODEL=DeepSeek-V4-Flash
SIGNALFORGE_SPECIALIST_VISION_MODEL=Qwen3.6-35B-A3B
SIGNALFORGE_SPECIALIST_API_TIMEOUT=90s
```

The credential is read from `.secrets/radeon_api_key`. It is never assigned in `.env`, committed,
embedded in the image, emitted through the process list, or exported to telemetry.

```bash
mkdir -p .secrets
chmod 700 .secrets
printf '%s' "$RADEON_API_KEY" > .secrets/radeon_api_key
chmod 600 .secrets/radeon_api_key
```

The repository audit rejects known credential patterns and forbidden private artifacts.

## Transport Contract

Each remote request contains only:

- the selected role;
- a bounded qualitative task;
- public-safe evidence references and compact authorized context;
- response constraints; and
- safe correlation identifiers.

It excludes credentials, prompts from other roles, private memory, source corpora, raw tool
receipts, answer bodies from other models, and hidden reasoning.

Go validates the response schema before it can enter the local critic stage.

## Fallback

- Remote timeout, malformed output, denied route, or API loss: replay through the authorized local
  specialist.
- Missing indispensable local model: fail closed, even when the remote API is healthy.
- Invalid local critic or final contract: no answer release.

The current representative tournament passed `5/5`; two cases were faster than local-only and
three were slower. The route is therefore selective, not the default for every specialist.

## Run

```bash
make championship-up BACKEND=auto
```

Inspect the runtime without exposing model bodies:

```bash
make radeon-status PROFILE=championship BACKEND=auto
make radeon-logs PROFILE=championship BACKEND=auto
```

Current aggregate evidence:
[`championship-radeon-runtime.json`](../evidence/championship-radeon-runtime.json).
