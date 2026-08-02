# Zero-Touch Radeon Appliance

## Goal

A fresh AMD Radeon Cloud workspace should start SignalForge from this repository without copying
artifacts from a developer machine, installing host packages, or downloading model weights at
application startup.

The appliance separates:

- immutable application source and image;
- separately licensed model hydration;
- persistent runtime state;
- runtime-only secrets; and
- optional observability.

## Requirements

- AMD Radeon Cloud workspace with persistent `/workspace` storage;
- `gfx1100` GPU devices available at `/dev/kfd` and `/dev/dri`;
- Docker Compose or the supported native ROCm toolchain;
- explicit Gemma license acceptance;
- Hugging Face read token only when the verified model is not already cached; and
- organizer Radeon API key only for the optional hybrid profile.

## Fresh Workspace

```bash
git clone https://github.com/rvbernucci/signalforge.git
cd signalforge

make radeon-bootstrap \
  BACKEND=auto \
  ACCEPT_GEMMA_LICENSE=yes

make radeon-up \
  BACKEND=auto
```

Open the loopback endpoint shown by:

```bash
make radeon-status \
  PROFILE=radeon-local \
  BACKEND=auto
```

## Backend Selection

`BACKEND=auto`:

1. validates the selected appliance manifest;
2. uses digest-pinned Docker Compose services when Docker is healthy;
3. otherwise selects the pinned native ROCm path already present in the AMD image; and
4. fails closed if neither path can preserve the declared source, model, runtime, or storage
   identity.

Bootstrap does not run `apt`, `brew`, or another host package manager.

## Persistent Layout

Default root: `/workspace/signalforge-runtime`

```text
data/       released case database, safe audit records, traces, logs
models/     verified separately licensed model artifact
state/      bootstrap, manifest, runtime, and readiness receipts
source/     exact source materialization for native execution
toolchain/  pinned native build outputs
```

Application memory is off by default. Persistent infrastructure storage does not by itself enable
case retention.

## Model Hydration

The model manifest pins:

- upstream repository and revision;
- exact filename;
- byte size;
- SHA-256;
- license boundary; and
- local model ID.

Hydration requires explicit acceptance, downloads to a temporary path, verifies size and hash, and
publishes atomically into persistent storage. Later starts reuse the verified cache.

The application image contains no model weight and performs no startup download.

## Profiles

| Profile | Behavior |
|---|---|
| `fixture` | No GPU, model, API key, or network after image pull |
| `radeon-local` | Local Gemma inference on ROCm only |
| `championship` | Local authority plus bounded Radeon API specialists |
| `observability` | Optional Grafana, Prometheus, Loki, Tempo, and Alloy |

## Secrets

Secrets are runtime files under `.secrets/`, mode `0600`, mounted read-only into the required
service. They are excluded from Git, images, logs, telemetry, evidence, and generated environment
files.

## Network Boundary

First-run network is limited to declared registries, source transport, and model hydration hosts.
The model data plane remains on the internal network; host-facing application and observability
ports bind only to loopback through the operator plane. Championship mode additionally permits the
organizer Radeon API host from the application process through a distinct specialist-egress
network.

## Safe Operations

```bash
make radeon-logs PROFILE=radeon-local BACKEND=auto
make radeon-down BACKEND=auto
make radeon-clean CONFIRM=clean
make radeon-reset CONFIRM=reset
```

`clean` removes disposable application state while retaining the verified model cache. `reset`
removes the complete runtime root and requires an explicit confirmation token.

## Verification

```bash
python3 scripts/radeon_validate_appliance.py
python3 scripts/validate_observability.py
scripts/verify_container_fixture.sh
```

The immutable release workflow adds SBOM, provenance, vulnerability scan, public pull, exact-image
fixture execution, and manifest evidence.
