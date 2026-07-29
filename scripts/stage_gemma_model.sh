#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
manifest="${SIGNALFORGE_MODEL_MANIFEST:-$repo_root/deploy/radeon/model-manifest.json}"
destination="${1:-$repo_root/.signalforge/radeon/models}"
token_file="${SIGNALFORGE_HF_TOKEN_FILE:-$repo_root/.secrets/hf-token}"
source="${SIGNALFORGE_MODEL_SOURCE:-huggingface}"
state="${SIGNALFORGE_MODEL_STATE:-$repo_root/.signalforge/radeon/state/model-init.json}"
license_args=()
existing_args=()

if [[ "${SIGNALFORGE_ACCEPT_GEMMA_LICENSE:-}" == "yes" ]]; then
  license_args+=(--license-accepted)
fi
if [[ -n "${SIGNALFORGE_EXISTING_MODEL_FILE:-}" ]]; then
  existing_args+=(--existing-file "$SIGNALFORGE_EXISTING_MODEL_FILE")
fi

exec python3 "$repo_root/scripts/radeon_model_cache.py" \
  --manifest "$manifest" \
  --cache-dir "$destination" \
  --source "$source" \
  --token-file "$token_file" \
  --retries "${SIGNALFORGE_MODEL_DOWNLOAD_RETRIES:-5}" \
  --timeout-seconds "${SIGNALFORGE_MODEL_DOWNLOAD_TIMEOUT_SECONDS:-60}" \
  --state "$state" \
  "${license_args[@]}" \
  "${existing_args[@]}"
