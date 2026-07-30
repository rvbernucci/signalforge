#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$repo_root/scripts/radeon_generated_env.sh"
profile="${SIGNALFORGE_PROFILE:-radeon-local}"
requested_backend="${SIGNALFORGE_EXECUTION_BACKEND:-auto}"
if [[ -n "${SIGNALFORGE_PERSIST_ROOT:-}" ]]; then
  persist_root="$SIGNALFORGE_PERSIST_ROOT"
elif [[ -d /workspace ]]; then
  persist_root="/workspace/signalforge-runtime"
else
  persist_root="$repo_root/.signalforge/radeon"
fi
generated_env="$persist_root/state/generated.env"
requested_manifest="${SIGNALFORGE_APPLIANCE_MANIFEST:-}"
requested_manifest_sha256="${SIGNALFORGE_APPLIANCE_MANIFEST_SHA256:-}"

if [[ -f "$generated_env" && ! -L "$generated_env" ]]; then
  signalforge_load_generated_env "$generated_env"
fi
stored_manifest="${SIGNALFORGE_APPLIANCE_MANIFEST:-}"
stored_manifest_sha256="${SIGNALFORGE_APPLIANCE_MANIFEST_SHA256:-}"

args=(
  --profile "$profile"
  --persist-root "$persist_root"
  --secrets-dir "$repo_root/.secrets"
  --report "$persist_root/state/preflight.json"
  --env-output "$generated_env"
  --model-source "${SIGNALFORGE_MODEL_SOURCE:-huggingface}"
  --backend "$requested_backend"
)
if [[ -n "$requested_manifest" ]]; then
  args+=(--manifest "$requested_manifest")
elif [[ -n "$stored_manifest" ]]; then
  args+=(--manifest "$stored_manifest")
fi
if [[ -n "$requested_manifest_sha256" ]]; then
  args+=(--manifest-sha256 "$requested_manifest_sha256")
elif [[ -n "$stored_manifest_sha256" ]]; then
  args+=(--manifest-sha256 "$stored_manifest_sha256")
fi
if [[ "${SIGNALFORGE_ACCEPT_GEMMA_LICENSE:-no}" == "yes" ]]; then
  args+=(--license-accepted)
fi
if [[ ! -f "$persist_root/models/.signalforge-model-ready.json" ]]; then
  args+=(--check-network)
fi

exec python3 "$repo_root/scripts/radeon_preflight.py" "${args[@]}"
