# Radeon Mission Control

## Purpose

Mission Control is an optional operator and judge surface for understanding how one research case
moves through planning, retrieval, tools, agents, models, validation, and release. The default
investor workspace remains clean; operational detail opens only on demand.

## Surfaces

- **Execution:** stage state, elapsed time, dependencies, fallback, and release disposition.
- **Inference:** local ROCm versus Radeon API route, model class, duration, and contract outcome.
- **Evidence:** authorized source counts, recency, conflict, and citation coverage.
- **Engines:** deterministic operation IDs, receipt counts, validation, and failure codes.
- **Privacy:** retention state, redaction counters, secret exposure checks, and export/delete
  events.
- **Radeon:** utilization, allocated VRAM, power, and temperature aggregates.

The Workspace, Proof Drawer, logs, metrics, and traces share privacy-safe `run_id`, `request_id`,
and `trace_id` values.

## Start

Fixture plus observability:

```bash
make mission-control-up
```

Live Radeon profile:

```bash
make radeon-observe PROFILE=championship BACKEND=auto
```

Default loopback endpoints:

| Surface | URL |
|---|---|
| Workspace | `http://127.0.0.1:8080` |
| Grafana | `http://127.0.0.1:3000` |
| Prometheus | `http://127.0.0.1:9090` |
| Alloy | `http://127.0.0.1:12345` |

Tempo and Loki remain internal to the stack. Host publications bind to loopback unless the
operator deliberately overrides them.

## Privacy Contract

Mission Control may contain:

- safe IDs;
- role and route classes;
- durations and aggregate token counts;
- status and failure codes;
- evidence and receipt counts; and
- aggregate GPU telemetry.

It must not contain:

- prompt or answer bodies;
- source bodies;
- credentials or authorization headers;
- private case memory;
- chain-of-thought or hidden reasoning; or
- raw provider payloads.

Telemetry is not a secondary research database.

## Validation

```bash
python3 scripts/validate_observability.py
scripts/verify_mission_control_runtime.sh \
  ghcr.io/rvbernucci/signalforge@sha256:<immutable-index-digest> \
  mission-control-evidence
```

The verifier binds the exact application image, runs the synchronized fixture, checks the
privacy-safe event contract, validates the observability services, and emits sanitized evidence.

## Current Measurement

The current Radeon campaign ran for 5 hours 28 minutes with 1,945 samples, constant 32% allocated
VRAM from first to last sample, and a maximum observed junction temperature of 63 C. See
[`championship-radeon-runtime.json`](../evidence/championship-radeon-runtime.json).
