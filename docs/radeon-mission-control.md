# Radeon Mission Control

Radeon Mission Control is SignalForge's optional, privacy-safe operational surface. It correlates
the investor workspace, agent orchestration, evidence retrieval, deterministic financial engines,
local or provided-model routes, and Radeon telemetry without exporting prompt bodies, answers,
private memory, raw financial values, credentials, or hidden reasoning.

## Surfaces

| Surface | Address | Purpose |
| --- | --- | --- |
| SignalForge Workspace | `http://127.0.0.1:8080` | Investor journey, specialist progress, answer, proof, and Intelligence Inspector |
| Grafana | `http://127.0.0.1:3000` | Four provisioned Mission Control dashboards |
| Prometheus | `http://127.0.0.1:9090` | Bounded aggregate metrics |
| Alloy status | `http://127.0.0.1:12345` | Collector health and pipeline status |

Loki and Tempo are intentionally not published to the host. Grafana reaches them on the private
Compose network.

## CPU Fixture

Docker is required. No GPU, model, model API, or network access is required by the application
journey.

```bash
make mission-control-up
```

The command creates a random local Grafana password in
`.secrets/grafana-admin-password`. Read it locally when signing in:

```bash
cat .secrets/grafana-admin-password
```

Run a case in the Workspace, then open **Intelligence path / Inspect orchestration**. Mission
Control collects the same safe execution events. Stop the stack with:

```bash
make mission-control-down
```

The same Makefile exposes the complete operating surface:

```bash
make radeon-local-up   # Local Gemma plus Mission Control
make championship-up  # Radeon API specialists, local Gemma, and Mission Control
make stack-status
make stack-logs
make evidence-export  # Public metadata only; protected bodies are excluded
make stack-down
```

`stack-down` stops containers and networks but deliberately preserves named data volumes and the
externally mounted model. Destructive cleanup is never part of a normal stop. If an operator later
chooses to remove disposable observability data, they must explicitly select the relevant named
volume after inspecting it; SignalForge never deletes the model path or retained research cases on
their behalf.

## Radeon Exporter

Run the exporter on the Radeon host, outside the observability containers, so it can use the
host's supported `amd-smi` or `rocm-smi` binary and device permissions:

```bash
make radeon-exporter
```

It binds to `127.0.0.1:9400`, tries `amd-smi` first, falls back to `rocm-smi`, exports only a
bounded GPU index plus utilization, VRAM, temperature, power, and collector availability, and
returns `radeon_gpu_exporter_up 0` rather than blocking SignalForge when telemetry is unavailable.

The ROCm profile passes `/dev/kfd` and `/dev/dri` without privileged mode. Because Radeon Cloud
images may expose numeric groups without names, inspect the device ownership on the allocated host
and override the defaults when needed:

```bash
stat -c '%g %n' /dev/kfd /dev/dri/renderD* 2>/dev/null
export SIGNALFORGE_VIDEO_GID=44
export SIGNALFORGE_RENDER_GID=109
```

Compose adds those numeric groups to the inference process; it does not depend on `video` or
`render` names existing inside the image.

## Traces

OpenTelemetry is opt-in:

```bash
export SIGNALFORGE_OTEL_ENABLED=true
export SIGNALFORGE_OTEL_INSECURE=true
export SIGNALFORGE_OTEL_ALLOW_PRIVATE_NETWORK=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://alloy:4318
```

The private-network permission only accepts a single-label Compose service such as `alloy`.
External collectors require HTTPS. Collector loss does not prevent journey completion; the batch
exporter times out independently.

## Protected Model I/O

Protected diagnostic capture is off by default. Enabling it requires a token mounted from a file,
never a browser-persisted or command-line token:

```bash
umask 077
printf '%s' "$(openssl rand -base64 30)" > .secrets/audit-operator-token
```

Add `--audit-capture --audit-token-file /run/secrets/audit_operator_token` to the application and
mount the file read-only. The Intelligence Inspector keeps the entered token only in React
component memory. Captured bodies are sanitized, bounded to 16 MiB per run by default, expire
after 15 minutes, and can be purged immediately. Metadata hashes and lineage remain after body
expiry so the execution is still auditable.

## Data Boundaries

Prometheus labels are limited to role class, provider class, route, operation class, capture state,
and typed status. Run IDs, trace IDs, companies, tickers, users, models, URLs, prompts, answers, and
errors never become metric labels.

Loki receives JSON lifecycle events containing safe IDs, hashes, bounded states, durations, and
token counts. Tempo receives the same safe identity spine as span attributes. Exact sanitized
model input/output is written only to the opt-in local audit vault and is never exported to
Prometheus, Loki, or Tempo.

## Dashboards

1. **Executive Journey** shows completion, end-to-end latency, context packets, receipts, and the
   live lifecycle timeline.
2. **Radeon Performance** shows GPU utilization, VRAM, temperature, power, and observed model
   latency.
3. **Agent Orchestration** shows governed routes, token flow, and the Tempo waterfall.
4. **Trust, Evidence, and Privacy** shows releases, verified calculations, retained audit bytes,
   minimized journeys, and trust decisions.

Every dashboard is immutable and provisioned from
`deploy/observability/grafana/dashboards/`.

## Validation

```bash
python3 scripts/validate_observability.py
python3 -m unittest scripts.tests.test_radeon_exporter
scripts/prepare_container_secrets.sh
docker compose --profile fixture --profile observability config --quiet
scripts/verify_container_fixture.sh
```

The last command builds `linux/amd64`, starts the image with `--network none`, a read-only root
filesystem, no capabilities, and a bounded writable volume, then executes health, run, lineage,
metrics, and secret checks. It is also a required GitHub Actions gate.

The release workflow additionally publishes by digest, attaches SBOM and build provenance, runs a
critical/high vulnerability scan, pulls the exact public digest without build cache, and preserves
the manifest as release evidence.

## Radeon-Only Gates

Configuration is complete, but the following claims require a live AMD Radeon workspace and are
not established by CPU fixture mode:

- model hash verification and real ROCm offload;
- cold and warm model startup;
- TTFT, tokens per second, VRAM, temperature, power, and queue behavior;
- exact-image local-only and hybrid journey quality;
- API-loss fallback and the vLLM bonus path;
- `rocprofv3` profiling and a two-to-four-hour soak; and
- synchronized screenshots and trace exports from the exact release image.
