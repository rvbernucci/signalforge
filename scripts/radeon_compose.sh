#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker Engine is required but was not found. This script never installs host tooling." >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "The Docker Compose plugin is required but unavailable." >&2
  exit 1
fi

profile="${SIGNALFORGE_PROFILE:-radeon-local}"
scope="${1:-current}"
if [[ "$scope" == "all" ]]; then
  shift
  profile_args=(
    --profile fixture
    --profile radeon-local
    --profile championship
    --profile observability
  )
else
  profile_args=(--profile "$profile")
  if [[ "${SIGNALFORGE_OBSERVABILITY:-0}" == "1" ]]; then
    profile_args+=(--profile observability)
  fi
fi

if [[ -n "${SIGNALFORGE_PERSIST_ROOT:-}" ]]; then
  persist_root="$SIGNALFORGE_PERSIST_ROOT"
elif [[ -d /workspace ]]; then
  persist_root="/workspace/signalforge-runtime"
else
  persist_root="$repo_root/.signalforge/radeon"
fi
generated_env="$persist_root/state/generated.env"
env_args=(--env-file "$repo_root/container.env.example")
if [[ -f "$generated_env" && ! -L "$generated_env" ]]; then
  env_args+=(--env-file "$generated_env")
fi

exec docker compose "${env_args[@]}" "${profile_args[@]}" "$@"
