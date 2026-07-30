#!/usr/bin/env bash

signalforge_load_generated_env() {
  local path="$1"
  local line key value seen

  [[ -f "$path" && ! -L "$path" ]] || {
    echo "Generated environment is missing, non-regular, or symbolic: $path" >&2
    return 2
  }
  python3 -c '
import os
import stat
import sys

details = os.stat(sys.argv[1], follow_symlinks=False)
if details.st_uid != os.geteuid() or stat.S_IMODE(details.st_mode) & 0o077:
    raise SystemExit("generated environment must be owner-only and owned by the current user")
' "$path" || return 2

  seen="|"
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "${line:0:1}" == "#" ]] && continue
    [[ "$line" == *"="* ]] || {
      echo "Generated environment contains an invalid line." >&2
      return 2
    }
    key="${line%%=*}"
    value="${line#*=}"
    case "$key" in
      SIGNALFORGE_ACCEPT_GEMMA_LICENSE|\
      SIGNALFORGE_APPLIANCE_MANIFEST|\
      SIGNALFORGE_APPLIANCE_MANIFEST_SHA256|\
      SIGNALFORGE_APPLICATION_ARTIFACT_IDENTITY|\
      SIGNALFORGE_APP_IMAGE|\
      SIGNALFORGE_EXECUTION_BACKEND|\
      SIGNALFORGE_LLAMA_ROCM_IMAGE|\
      SIGNALFORGE_MODEL_ARTIFACT_IDENTITY|\
      SIGNALFORGE_MODEL_SOURCE|\
      SIGNALFORGE_PERSIST_ROOT|\
      SIGNALFORGE_RENDER_GID|\
      SIGNALFORGE_RUNTIME_IDENTITY|\
      SIGNALFORGE_VIDEO_GID)
        ;;
      *)
        echo "Generated environment contains an unapproved key: $key" >&2
        return 2
        ;;
    esac
    case "$seen" in
      *"|$key|"*)
        echo "Generated environment repeats a key: $key" >&2
        return 2
        ;;
    esac
    [[ "$value" != *$'\r'* ]] || {
      echo "Generated environment contains a carriage return." >&2
      return 2
    }
    seen="${seen}${key}|"
    export "$key=$value"
  done < "$path"
}
