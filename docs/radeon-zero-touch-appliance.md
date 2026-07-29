# Zero-Touch Radeon Appliance

This guide is the canonical deployment path for a fresh AMD Radeon Cloud workspace. It requires
no model, binary, configuration, dataset, or generated artifact to be copied from a Mac. Backend
selection is automatic: Docker Compose is preferred when healthy; otherwise the current AMD
OneClick image uses a pinned native ROCm path.

## What The Appliance Starts

| Profile | Core path | External access after startup |
|---|---|---|
| `fixture` | Immutable image or verified native build and public fixture | None |
| `radeon-local` | Verified app, Gemma cache, and AMD ROCm llama.cpp | None |
| `championship` | Local path plus bounded organizer Radeon API specialists | Radeon API from the application service only |
| `observability` | Optional Alloy, Prometheus, Loki, Tempo, and Grafana | No application egress is added |

The application does not download a model at startup. `model-init` is a separate, bounded phase
that writes only to persistent storage. Local inference and the application remain blocked until
the complete model passes size and SHA-256 verification and the runtime serves the expected model
identity.

## Fresh Workspace: Two Commands

Use the AMD OneClick base with persistent `/workspace` storage, SSH enabled, and this public
repository on `main`. From the cloned repository:

```bash
make radeon-bootstrap ACCEPT_GEMMA_LICENSE=yes
make radeon-up
```

The setup command:

- selects `compose` only when Docker Engine and Compose are healthy, otherwise selects `native`;
- verifies the native OneClick tools (`git`, `curl`, Python, Node/npm, CMake, Ninja, and `hipcc`)
  but installs no host package;
- detects `/dev/kfd`, the DRM render node, effective device group IDs, ROCm, GPU architecture,
  VRAM, RAM, CPU, disk, and persistent storage;
- asks for a Hugging Face read token through a hidden terminal prompt only when the pinned model is
  absent;
- stores credentials only under the ignored `.secrets/` directory;
- records derived, non-secret environment values under persistent storage; and
- writes a machine-readable preflight report without credential values.

Before passing `ACCEPT_GEMMA_LICENSE=yes`, review and accept the
[upstream Gemma terms](https://ai.google.dev/gemma/terms) for the Hugging Face account whose
read token is supplied. The model is gated upstream and is not redistributed by SignalForge.

`make radeon-up` then either pulls digest-pinned OCI images or hydrates the pinned native
toolchain. Both paths reuse the same verified model cache, start the local model, verify `/health`
and `/v1/models`, and start SignalForge. It prints:

- the active profile;
- the current startup phase;
- safe service health;
- immutable application, runtime, and model identities; and
- the local workspace URL.

The default workspace is <http://127.0.0.1:8080>.

The persistent root defaults to `/workspace/signalforge-runtime`. When a separate PVC mount is
preferred, export `SIGNALFORGE_PERSIST_ROOT` before both commands. Bootstrap, preflight, startup,
status, shutdown, and reset then resolve the same path; no command silently falls back to another
cache.

## Automatic Execution Backend

`SIGNALFORGE_EXECUTION_BACKEND=auto` is the default.

- `compose` uses the existing digest-pinned images and never performs a local image build.
- `native` downloads Go `1.25.12` for `linux/amd64`, verifies SHA-256
  `234828b7a89e0e303d2556310ee549fbcf253d28de937bac3da13d6294262ac1`, and extracts it
  atomically under persistent storage.
- Native `llama.cpp` reuses `scripts/build_llama_rocm.sh` at
  `305ba519ab61cdff8044922cba2347826a04453f`, built by the OneClick CMake/Ninja/`hipcc`
  toolchain for `gfx1100`.
- Native frontend construction uses `npm ci` and the pinned package lock. Backend construction
  uses the pinned Go toolchain, `go.sum`, `GOTOOLCHAIN=local`, the Radeon-region Go proxy and
  checksum mirror declared in the native manifest, persistent module/build caches, and a clean Git
  commit identity.
- Native processes bind to loopback. PID receipts, logs, build receipts, health state, and
  readiness live under `/workspace/signalforge-runtime/state/native` with private permissions.
- Native startup refuses an application or model port owned by an untracked process, and
  application readiness must report the exact source commit and runtime profile launched by the
  supervisor.
- Championship passes only `.secrets/radeon-model-api-key` to the application through
  `SIGNALFORGE_SPECIALIST_API_KEY_FILE`; the value is not placed in an environment variable or
  command line.

Explicit selection remains available for diagnosis:

```bash
make radeon-up BACKEND=compose
make radeon-up BACKEND=native
```

The optional containerized Grafana stack requires `compose`. Native mode retains safe status and
redacted application/model logs but does not install observability packages on the host.

## Profiles And Operator Commands

```bash
# Local private inference, the default
make radeon-up

# Local authority plus organizer Radeon API specialists
make radeon-bootstrap PROFILE=championship ACCEPT_GEMMA_LICENSE=yes
make radeon-up PROFILE=championship

# Start the optional audit stack beside the selected profile
make radeon-observe PROFILE=radeon-local

# Safe inspection
make radeon-status PROFILE=radeon-local
make radeon-logs PROFILE=radeon-local

# Stop containers and retain application data and the verified model cache
make radeon-down
```

`radeon-logs` redacts credential-shaped values plus prompt, question, answer, response, content,
input, and output bodies. Full private inference bodies are not an operator-log feature.

## Noninteractive Operation

CI or an exact-image verifier may create the ignored files before bootstrap:

```text
.secrets/hf-token
.secrets/radeon-model-api-key
.secrets/grafana-admin-password
```

Each value must be a single line. The secret directory must be mode `0700`; files must not be
group- or world-writable. Then run:

```bash
make radeon-bootstrap \
  PROFILE=radeon-local \
  ACCEPT_GEMMA_LICENSE=yes \
  RADEON_NONINTERACTIVE=1
make radeon-up PROFILE=radeon-local
```

The Radeon API key is required only for `championship`. The Grafana password is required only when
the observability profile is selected. No credential belongs in `container.env.example`, Compose,
Git, an OCI layer, or a command-line argument.

## Immutable Identities

The authoritative identities live in:

- `deploy/radeon/appliance-manifest.json`;
- `deploy/radeon/model-manifest.json`; and
- `deploy/radeon/native-toolchain-manifest.json`; and
- the generated preflight report under `/workspace/signalforge-runtime/state/preflight.json`.

The current zero-touch contract pins:

- SignalForge `v1.1.1` by the public multi-platform index digest whose runtime manifest is
  `linux/amd64`;
- the AMD-published ROCm llama.cpp server by digest;
- Go `1.25.12` and native `llama.cpp` source revision by SHA/revision;
- Python and Alpine init images by digest;
- the exact Gemma repository, revision, filename, byte size, SHA-256, license locator, and served
  model ID.

No moving tag is sufficient for a release. A future application release must update the appliance
manifest, Compose default, environment example, and static appliance audit together.

## Model Hydration Contract

The cache supports:

- `huggingface`: resumable, retry-bounded HTTP download through the Radeon-region Hugging Face
  mirror using a file-mounted read token, while preserving the canonical Hugging Face locator;
- `existing`: import of an independently acquired exact file, still subject to complete size and
  hash verification; and
- `oci`: deliberately disabled until a separate rights decision permits model redistribution.

A partial download lives under `.downloads/` and never becomes the served model. Publication uses
an atomic rename only after the complete hash passes. The atomic ready marker binds the cache to
the manifest version, revision, served ID, byte size, and SHA-256.

Interrupted downloads resume from the observed byte offset. A corrupt complete file loses its
ready marker and is rejected. A valid cache is idempotently reused on warm startup without
requiring external network access.

## Storage And Safe Cleanup

On Radeon Cloud, persistent state defaults to:

```text
/workspace/signalforge-runtime/
├── data/
├── models/
└── state/
```

Ordinary shutdown retains all three directories. Cleanup is intentionally explicit:

```bash
# Remove disposable application/state data but retain the verified model
make radeon-clean CONFIRM=clean-signalforge-state

# Remove the complete runtime root, including the 14.4 GB model cache
make radeon-reset CONFIRM=delete-signalforge-runtime
```

Both commands stop the stack first, reject symbolic links and broad paths, and refuse to run
without the exact confirmation phrase.

## Network Boundary

First-run pulls are declared by backend in `deploy/radeon/appliance-manifest.json`. Compose checks
only its OCI registries. Native checks the AMD-image Git proxy, Google Go distribution, the
Radeon-region Go module/checksum mirrors, and npm. The model destination is checked only when a
local model is required and the verified cache is absent. The preflight performs TLS connectivity
checks without credentials.

After hydration:

- Compose `fixture` and `radeon-local` run on an internal Docker network;
- native `fixture` and `radeon-local` expose only the loopback application port;
- model-init is stopped, so its hydration network is inactive;
- local model and application ports are not externally published beyond the application loopback
  URL; and
- `championship` grants egress only to the application service for the organizer API path.

The model runtime is never published to the host. Observability endpoints bind to `127.0.0.1`.

## Failure Diagnosis

| Phase | Meaning | Safe action |
|---|---|---|
| `model-hydration` | No verified ready marker exists | `make radeon-logs`; confirm license, token, disk, and network |
| `model-load-or-runtime-health` | Runtime has not returned healthy with the exact model ID | Inspect GPU/ROCm preflight and redacted logs |
| `application-startup` | Model is ready but the workspace is not | Inspect application health and redacted logs |
| `compose-unavailable` | Existing Docker/Compose cannot be used | Repair the host image; the appliance will not install a duplicate runtime |
| `native-build` | Go, npm, application, or llama.cpp has not completed | Inspect redacted `native-build.log`; no host package is installed |
| `model-runtime` | Native llama-server is absent or unhealthy | Check the pinned build receipt and `make radeon-logs` |
| `ready` | App, verified model, and mandatory dependencies satisfy the selected profile | Open the workspace URL |

`make radeon-preflight` is idempotent. It refuses unsupported architecture, GPU, device, disk,
secret-permission, Docker, Compose, or first-run network states with actionable messages.

## Verification Boundary

The following checks run without a live GPU:

```bash
python3 scripts/radeon_validate_appliance.py
python3 -m unittest \
  scripts.tests.test_radeon_backend \
  scripts.tests.test_radeon_model_cache \
  scripts.tests.test_radeon_native_toolchain \
  scripts.tests.test_radeon_native_runtime \
  scripts.tests.test_radeon_preflight \
  scripts.tests.test_radeon_bootstrap \
  scripts.tests.test_radeon_runtime_probe \
  scripts.tests.test_radeon_operator
```

A final release still requires a genuinely fresh Radeon workspace to prove the native Go and
llama.cpp builds, full 14.4 GB hydration, model load, cold/warm startup, interrupted real
downloads, corrupt-cache recovery, local-only networking, API loss, process teardown, and
persistent-workspace recreation. Local simulations are not relabelled as that evidence.
