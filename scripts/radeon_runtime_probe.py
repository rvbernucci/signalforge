#!/usr/bin/env python3
"""Wait for a fully loaded local model and publish a bounded readiness receipt."""

from __future__ import annotations

import argparse
import http.server
import json
import os
import socketserver
import sys
import threading
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


def fetch_json(url: str, timeout: float) -> dict[str, Any]:
    request = urllib.request.Request(url, headers={"User-Agent": "SignalForge-Runtime-Probe/1"})
    with urllib.request.urlopen(request, timeout=timeout) as response:
        if response.status != 200:
            raise RuntimeError(f"unexpected HTTP status {response.status}")
        return json.load(response)


def atomic_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    try:
        temporary.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        os.chmod(temporary, 0o444)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def wait_ready(
    base_url: str,
    model_id: str,
    timeout_seconds: float,
    request_timeout_seconds: float,
    poll_seconds: float,
    fetcher=fetch_json,
) -> dict[str, Any]:
    started = time.monotonic()
    deadline = started + timeout_seconds
    attempts = 0
    last_error = "runtime did not respond"
    while time.monotonic() < deadline:
        attempts += 1
        try:
            health = fetcher(f"{base_url.rstrip('/')}/health", request_timeout_seconds)
            models = fetcher(f"{base_url.rstrip('/')}/v1/models", request_timeout_seconds)
            observed_models = {
                str(item.get("id", ""))
                for item in models.get("data", [])
                if isinstance(item, dict)
            }
            if health.get("status") == "ok" and model_id in observed_models:
                return {
                    "schema_version": "signalforge/runtime-ready/v1",
                    "status": "ready",
                    "model_id": model_id,
                    "health_status": "ok",
                    "attempts": attempts,
                    "elapsed_ms": round((time.monotonic() - started) * 1000, 3),
                }
            last_error = "health or served model identity did not match"
        except (OSError, RuntimeError, urllib.error.URLError, json.JSONDecodeError) as error:
            last_error = type(error).__name__
        time.sleep(poll_seconds)
    raise TimeoutError(f"runtime readiness timed out: {last_error}")


def check_once(
    base_url: str,
    model_id: str,
    request_timeout_seconds: float,
    fetcher=fetch_json,
) -> dict[str, Any]:
    health = fetcher(f"{base_url.rstrip('/')}/health", request_timeout_seconds)
    models = fetcher(f"{base_url.rstrip('/')}/v1/models", request_timeout_seconds)
    observed_models = sorted(
        {
            str(item.get("id", ""))
            for item in models.get("data", [])
            if isinstance(item, dict) and item.get("id")
        }
    )
    ready = health.get("status") == "ok" and model_id in observed_models
    return {
        "schema_version": "signalforge/runtime-health/v1",
        "status": "ready" if ready else "unavailable",
        "model_id": model_id,
        "observed_models": observed_models,
        "health_status": str(health.get("status", "unknown")),
    }


def serve(
    *,
    base_url: str,
    model_id: str,
    listen_host: str,
    listen_port: int,
    request_timeout_seconds: float,
    state_path: Path | None,
    fetcher=fetch_json,
) -> None:
    lock = threading.Lock()

    class Handler(http.server.BaseHTTPRequestHandler):
        def log_message(self, _format: str, *_args: object) -> None:
            return

        def do_GET(self) -> None:
            if self.path != "/health":
                self.send_error(404)
                return
            try:
                receipt = check_once(base_url, model_id, request_timeout_seconds, fetcher)
            except (OSError, RuntimeError, urllib.error.URLError, json.JSONDecodeError) as error:
                receipt = {
                    "schema_version": "signalforge/runtime-health/v1",
                    "status": "unavailable",
                    "model_id": model_id,
                    "reason": type(error).__name__,
                }
            if state_path:
                with lock:
                    atomic_json(state_path, receipt)
            encoded = (json.dumps(receipt, sort_keys=True) + "\n").encode()
            self.send_response(200 if receipt["status"] == "ready" else 503)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)

    class Server(socketserver.ThreadingMixIn, http.server.HTTPServer):
        daemon_threads = True
        allow_reuse_address = True

    with Server((listen_host, listen_port), Handler) as server:
        server.serve_forever()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://llama-rocm:8000")
    parser.add_argument("--model-id", required=True)
    parser.add_argument("--timeout-seconds", type=float, default=300)
    parser.add_argument("--request-timeout-seconds", type=float, default=3)
    parser.add_argument("--poll-seconds", type=float, default=2)
    parser.add_argument("--state", type=Path)
    parser.add_argument("--serve", action="store_true")
    parser.add_argument("--listen-host", default="0.0.0.0")
    parser.add_argument("--listen-port", type=int, default=8090)
    args = parser.parse_args()
    if args.timeout_seconds <= 0 or args.request_timeout_seconds <= 0 or args.poll_seconds <= 0:
        print("runtime probe durations must be positive", file=sys.stderr)
        return 2
    if args.serve:
        serve(
            base_url=args.base_url,
            model_id=args.model_id,
            listen_host=args.listen_host,
            listen_port=args.listen_port,
            request_timeout_seconds=args.request_timeout_seconds,
            state_path=args.state,
        )
        return 0
    try:
        receipt = wait_ready(
            args.base_url,
            args.model_id,
            args.timeout_seconds,
            args.request_timeout_seconds,
            args.poll_seconds,
        )
    except TimeoutError as error:
        print(str(error), file=sys.stderr)
        return 1
    if args.state:
        atomic_json(args.state, receipt)
    print(
        f"Local runtime ready: {receipt['model_id']} "
        f"after {receipt['attempts']} probe attempt(s)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
