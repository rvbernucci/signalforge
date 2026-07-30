#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$repo_root/scripts/radeon_generated_env.sh"
profile="${SIGNALFORGE_PROFILE:-radeon-local}"
if [[ -n "${SIGNALFORGE_PERSIST_ROOT:-}" ]]; then
  persist_root="$SIGNALFORGE_PERSIST_ROOT"
elif [[ -d /workspace ]]; then
  persist_root="/workspace/signalforge-runtime"
else
  persist_root="$repo_root/.signalforge/radeon"
fi
generated_env="$persist_root/state/generated.env"

"$repo_root/scripts/radeon_preflight.sh"
if [[ -f "$generated_env" && ! -L "$generated_env" ]]; then
  signalforge_load_generated_env "$generated_env"
fi
backend="${SIGNALFORGE_EXECUTION_BACKEND:-auto}"

if [[ "$backend" == "compose" ]]; then
  timeout="${SIGNALFORGE_STARTUP_TIMEOUT_SECONDS:-2400}"
  "$repo_root/scripts/radeon_compose.sh" current pull
  "$repo_root/scripts/radeon_compose.sh" current up --detach --no-build --remove-orphans
  exec python3 "$repo_root/scripts/radeon_status.py" \
    --profile "$profile" \
    --backend compose \
    --manifest "$SIGNALFORGE_APPLIANCE_MANIFEST" \
    --manifest-sha256 "$SIGNALFORGE_APPLIANCE_MANIFEST_SHA256" \
    --wait-seconds "$timeout"
fi

if [[ "$backend" != "native" ]]; then
  echo "Unsupported resolved execution backend: $backend" >&2
  exit 2
fi
if [[ "${SIGNALFORGE_OBSERVABILITY:-0}" == "1" ]]; then
  echo "The native backend does not install an observability stack; use Compose or the app's safe local logs." >&2
  exit 2
fi
dirty_args=()
if [[ "${SIGNALFORGE_ALLOW_DIRTY_NATIVE_BUILD:-0}" == "1" ]]; then
  dirty_args+=(--allow-dirty)
fi
python3 "$repo_root/scripts/radeon_native_runtime.py" up \
  --persist-root "$persist_root" \
  --profile "$profile" \
  --secrets-dir "$repo_root/.secrets" \
  --manifest "$SIGNALFORGE_APPLIANCE_MANIFEST" \
  --manifest-sha256 "$SIGNALFORGE_APPLIANCE_MANIFEST_SHA256" \
  "${dirty_args[@]}"
exec python3 "$repo_root/scripts/radeon_status.py" \
  --profile "$profile" \
  --backend native \
  --manifest "$SIGNALFORGE_APPLIANCE_MANIFEST" \
  --manifest-sha256 "$SIGNALFORGE_APPLIANCE_MANIFEST_SHA256"
