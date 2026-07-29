#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
requested_backend="${SIGNALFORGE_EXECUTION_BACKEND:-auto}"
if [[ -n "${SIGNALFORGE_PERSIST_ROOT:-}" ]]; then
  persist_root="$SIGNALFORGE_PERSIST_ROOT"
elif [[ -d /workspace ]]; then
  persist_root="/workspace/signalforge-runtime"
else
  persist_root="$repo_root/.signalforge/radeon"
fi
generated_env="$persist_root/state/generated.env"
if [[ -f "$generated_env" && ! -L "$generated_env" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$generated_env"
  set +a
fi
if [[ "$requested_backend" != "auto" ]]; then
  SIGNALFORGE_EXECUTION_BACKEND="$requested_backend"
fi
backend="$(
  python3 "$repo_root/scripts/radeon_backend.py" \
    --backend "${SIGNALFORGE_EXECUTION_BACKEND:-auto}" \
    --persist-root "$persist_root"
)"
if [[ "$backend" == "native" ]]; then
  python3 "$repo_root/scripts/radeon_native_runtime.py" logs \
    --persist-root "$persist_root" \
    --tail "${SIGNALFORGE_LOG_TAIL:-200}" \
    | python3 "$repo_root/scripts/radeon_logs.py"
else
  "$repo_root/scripts/radeon_compose.sh" all logs --no-color --tail="${SIGNALFORGE_LOG_TAIL:-200}" "$@" \
    | python3 "$repo_root/scripts/radeon_logs.py"
fi
