#!/usr/bin/env python3
"""Collect a bounded, auditable IR corpus into a private content-addressed store."""

from __future__ import annotations

import argparse
import datetime as dt
import email.utils
import hashlib
from html.parser import HTMLParser
import http.client
import json
import os
from pathlib import Path
import re
import socket
import time
from typing import Any
import urllib.error
import urllib.parse
import urllib.request
import urllib.robotparser

try:
    import ir_discover
except ModuleNotFoundError:  # Imported by the repository test runner.
    from scripts import ir_discover


COLLECTOR_VERSION = "ir-collector/1.2.0"
DOCUMENT_SCHEMA = "signalforge/investor-relations-document/v2"
DOCUMENT_SUFFIXES = {".pdf", ".txt", ".htm", ".html", ".aspx"}
INDEX_PATTERN = re.compile(
    r"financial|earning|quarter|annual|result|event|presentation|governance|"
    r"committee|board|shareholder|history|about|overview|strategy|risk|guidance|outlook|"
    r"product|segment|business|conference|prepared.remarks|transcript|capital.allocation|"
    r"dividend|repurchase|buyback",
    re.IGNORECASE,
)
EXCLUDED_PATTERN = re.compile(
    r"privacy|cookie|career|jobs?|support|store|shop|diversity|notification|email.alert|"
    r"press.contact|sitemap|accessibility|terms|legal|sec.filing|annual.report|10[- ]?[kq]",
    re.IGNORECASE,
)
YEAR_PATTERN = re.compile(r"\b(20[0-9]{2})\b")


class LinkParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.links: list[tuple[str, str]] = []
        self.title_parts: list[str] = []
        self._in_title = False
        self._anchor_href = ""
        self._anchor_text: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        values = dict(attrs)
        if tag.lower() == "title":
            self._in_title = True
        if tag.lower() == "a" and values.get("href"):
            self._anchor_href = values["href"] or ""
            self._anchor_text = [values.get("title") or values.get("aria-label") or ""]

    def handle_data(self, data: str) -> None:
        if self._in_title and data.strip():
            self.title_parts.append(data.strip())
        if self._anchor_href and data.strip():
            self._anchor_text.append(data.strip())

    def handle_endtag(self, tag: str) -> None:
        if tag.lower() == "title":
            self._in_title = False
        if tag.lower() == "a" and self._anchor_href:
            self.links.append((self._anchor_href, " ".join(self._anchor_text)))
            self._anchor_href = ""
            self._anchor_text = []


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--registry", type=Path, required=True)
    parser.add_argument("--discovery-report", type=Path, required=True)
    parser.add_argument("--raw-store", type=Path, required=True)
    parser.add_argument("--report-dir", type=Path, required=True)
    parser.add_argument("--tickers", default="")
    parser.add_argument("--max-index-pages", type=int, default=12)
    parser.add_argument("--max-documents", type=int, default=80)
    parser.add_argument("--max-depth", type=int, default=2)
    parser.add_argument("--timeout", type=float, default=45.0)
    parser.add_argument("--evaluation-only-pending-rights", action="store_true")
    parser.add_argument("--resume", action="store_true")
    return parser.parse_args()


def canonical_uri(raw: str) -> str:
    parsed = urllib.parse.urlsplit(raw)
    query = urllib.parse.parse_qsl(parsed.query, keep_blank_values=True)
    filtered = [(key, value) for key, value in query if not key.lower().startswith("utm_")]
    return urllib.parse.urlunsplit(
        (parsed.scheme.lower(), parsed.netloc.lower(), parsed.path or "/", urllib.parse.urlencode(filtered), "")
    )


def classify(uri: str, label: str, title: str) -> tuple[str, str, bool]:
    text = " ".join((uri, label, title)).lower()
    if any(value in text for value in ("dividend", "repurchase", "buyback", "capital allocation")):
        return "capital_allocation_update", "C", False
    if "analyst day" in text or "investor day" in text:
        return "investor_presentation", "C", False
    if "safe harbor" in text or "forward-looking" in text:
        return "guidance_and_outlook", "C", False
    if any(value in text for value in ("governance", "committee", "board", "code of conduct", "code of ethics", "charter", "bylaws")):
        return "governance_document", "D", False
    if "annual meeting" in text or "proxy" in text:
        return "annual_meeting_material", "D", False
    if "prepared" in text and "remark" in text:
        return "prepared_remarks", "C", False
    if "transcript" in text:
        return "official_earnings_transcript", "C", False
    if "shareholder" in text and "letter" in text:
        return "shareholder_letter", "C", False
    if "presentation" in text or "conference" in text:
        return "investor_presentation", "C", False
    if "guidance" in text or "outlook" in text:
        return "guidance_and_outlook", "C", False
    if "earning" in text or "quarterly result" in text or "financial result" in text:
        return "earnings_release", "B", False
    if "history" in text or "company" in text or "about" in text:
        return "corporate_profile_and_history", "E", True
    if "product" in text or "segment" in text or "business" in text:
        return "business_products_and_segments", "E", True
    if "strategy" in text or "risk" in text:
        return "official_strategy_or_risk_update", "C", False
    return "official_strategy_or_risk_update", "C", False


def eligible_year(uri: str, label: str, start_year: int) -> bool:
    years = [int(value) for value in YEAR_PATTERN.findall(uri + " " + label)]
    return not years or max(years) >= start_year


def is_material_candidate(uri: str, label: str) -> bool:
    parsed = urllib.parse.urlsplit(uri)
    path = urllib.parse.unquote(parsed.path)
    haystack = path + " " + label
    if EXCLUDED_PATTERN.search(path) and not re.search(r"shareholder.letter", haystack, re.IGNORECASE):
        return False
    if not INDEX_PATTERN.search(path) and EXCLUDED_PATTERN.search(label):
        return False
    return bool(INDEX_PATTERN.search(haystack))


def normalize_links(base_uri: str, parser: LinkParser) -> list[dict[str, str]]:
    result: dict[str, dict[str, str]] = {}
    for href, label in parser.links:
        uri = canonical_uri(urllib.parse.urljoin(base_uri, href))
        parsed = urllib.parse.urlsplit(uri)
        if parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password:
            continue
        result.setdefault(uri, {"uri": uri, "label": " ".join(label.split())})
    return sorted(result.values(), key=lambda item: item["uri"])


class HostBudget:
    def __init__(self, requests_per_second: float) -> None:
        self._interval = 1.0 / requests_per_second
        self._last_request: dict[str, float] = {}

    def wait(self, host: str) -> None:
        delay = self._interval - (time.monotonic() - self._last_request.get(host, 0.0))
        if delay > 0:
            time.sleep(delay)
        self._last_request[host] = time.monotonic()


class RobotsGuard:
    def __init__(self, user_agent: str, timeout: float, budget: HostBudget) -> None:
        self.user_agent = user_agent
        self.timeout = timeout
        self.budget = budget
        self.decisions: dict[tuple[str, str], tuple[bool, dict[str, Any]]] = {}

    def check(self, uri: str, company_id: str, allowed_hosts: list[str]) -> tuple[bool, dict[str, Any] | None]:
        host = urllib.parse.urlsplit(uri).hostname or ""
        cache_key = (company_id, host)
        if cache_key in self.decisions:
            return self.decisions[cache_key][0], None
        robots_uri = f"https://{host}/robots.txt"
        self.budget.wait(host)
        fetched = ir_discover.fetch(robots_uri, self.user_agent, min(self.timeout, 20.0), 512 * 1024)
        observed_at = dt.datetime.now(dt.timezone.utc).isoformat()
        final_uri = fetched.get("final_uri", robots_uri)
        final_host = urllib.parse.urlsplit(final_uri).hostname or ""
        status = fetched.get("status_code")
        allowed = False
        reason = "robots_fetch_failed"
        if not ir_discover.host_allowed(final_host, allowed_hosts):
            reason = "robots_redirect_outside_allowlist"
        elif status == 200 and isinstance(fetched.get("payload"), bytes):
            parser = urllib.robotparser.RobotFileParser()
            parser.set_url(robots_uri)
            parser.parse(fetched["payload"].decode("utf-8", errors="replace").splitlines())
            allowed = parser.can_fetch(self.user_agent, uri)
            reason = "robots_allowed" if allowed else "robots_disallowed"
        elif status in {404, 410}:
            allowed = True
            reason = "robots_missing_standard_allow"
        observation = {
            "observation_id": "obs-" + hashlib.sha256((company_id + robots_uri + observed_at).encode()).hexdigest()[:24],
            "company_id": company_id,
            "requested_uri": robots_uri,
            "observed_at": observed_at,
            "disposition": "accepted" if allowed else "disallowed",
            "attempt": 1,
            "elapsed_ms": fetched.get("elapsed_ms", 0),
            "failure_class": reason,
        }
        for key in ("status_code", "final_uri", "media_type", "content_bytes", "content_sha256"):
            if fetched.get(key) is not None:
                observation[key] = fetched[key]
        self.decisions[cache_key] = (allowed, observation)
        return allowed, observation


def retry_delay(retry_after: str | None, attempt: int) -> float:
    if retry_after:
        try:
            return min(30.0, max(0.0, float(retry_after)))
        except ValueError:
            try:
                parsed = email.utils.parsedate_to_datetime(retry_after)
                now = dt.datetime.now(dt.timezone.utc)
                return min(30.0, max(0.0, (parsed - now).total_seconds()))
            except (TypeError, ValueError, OverflowError):
                pass
    return min(4.0, float(2 ** (attempt - 1)))


def fetch(
    uri: str,
    user_agent: str,
    timeout: float,
    max_bytes: int,
    allowed_media_types: set[str],
    allowed_hosts: list[str],
    source: dict[str, Any],
    budget: HostBudget,
) -> tuple[dict[str, Any], bytes]:
    started = time.monotonic()
    host = urllib.parse.urlsplit(uri).hostname or ""
    for attempt in range(1, 4):
        budget.wait(host)
        request = urllib.request.Request(
            uri,
            headers={"User-Agent": user_agent, "Accept-Encoding": "identity", "Accept": "text/html,application/pdf,text/plain;q=0.8,*/*;q=0.1"},
        )
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                final_uri = canonical_uri(response.geturl())
                if not ir_discover.source_uri_allowed(final_uri, source):
                    return {"disposition": "rejected", "failure_class": "redirect_outside_allowlist", "final_uri": final_uri, "attempt": attempt}, b""
                content_length = response.headers.get("Content-Length")
                if content_length and int(content_length) > max_bytes:
                    return {"disposition": "rejected", "failure_class": "declared_size_limit", "final_uri": final_uri, "attempt": attempt}, b""
                media_type = response.headers.get_content_type().lower()
                if media_type not in allowed_media_types:
                    return {
                        "disposition": "rejected",
                        "failure_class": "unsupported_media_type",
                        "final_uri": final_uri,
                        "media_type": media_type,
                        "attempt": attempt,
                        "elapsed_ms": round((time.monotonic() - started) * 1000),
                    }, b""
                payload = response.read(max_bytes + 1)
                if len(payload) > max_bytes:
                    return {"disposition": "rejected", "failure_class": "stream_size_limit", "final_uri": final_uri, "attempt": attempt}, b""
                result = {
                    "disposition": "accepted",
                    "status_code": response.status,
                    "final_uri": final_uri,
                    "media_type": media_type,
                    "content_bytes": len(payload),
                    "content_sha256": hashlib.sha256(payload).hexdigest(),
                    "attempt": attempt,
                    "elapsed_ms": round((time.monotonic() - started) * 1000),
                }
                for key, header in (("etag", "ETag"), ("last_modified", "Last-Modified")):
                    if response.headers.get(header):
                        result[key] = response.headers[header]
                return result, payload
        except urllib.error.HTTPError as error:
            retry_after = error.headers.get("Retry-After") if error.headers else None
            if error.code in {429, 500, 502, 503, 504} and attempt < 3:
                time.sleep(retry_delay(retry_after, attempt))
                continue
            result = {
                "disposition": "blocked" if error.code in {401, 403} else "failed",
                "status_code": error.code,
                "failure_class": "http_error",
                "attempt": attempt,
                "elapsed_ms": round((time.monotonic() - started) * 1000),
            }
            if retry_after:
                result["retry_after"] = retry_after
            return result, b""
        except (TimeoutError, socket.timeout):
            if attempt < 3:
                time.sleep(retry_delay(None, attempt))
                continue
            return {"disposition": "timeout", "failure_class": "timeout", "attempt": attempt, "elapsed_ms": round((time.monotonic() - started) * 1000)}, b""
        except (urllib.error.URLError, ValueError, OSError, http.client.HTTPException) as error:
            if attempt < 3:
                time.sleep(retry_delay(None, attempt))
                continue
            return {"disposition": "failed", "failure_class": type(error).__name__, "attempt": attempt, "elapsed_ms": round((time.monotonic() - started) * 1000)}, b""
    raise AssertionError("bounded retry loop exhausted without a result")


def write_jsonl(path: Path, records: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as handle:
        for record in records:
            handle.write(json.dumps(record, sort_keys=True) + "\n")


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def store_payload(raw_store: Path, digest: str, payload: bytes) -> Path:
    path = raw_store / "sha256" / digest[:2] / digest
    path.parent.mkdir(parents=True, exist_ok=True)
    if not path.exists():
        path.write_bytes(payload)
    return path


def crawl_source(
    source: dict[str, Any],
    user_agent: str,
    args: argparse.Namespace,
    budget: HostBudget,
    robots: RobotsGuard,
) -> tuple[list[dict[str, Any]], list[dict[str, Any]], dict[str, Any]]:
    started_at = dt.datetime.now(dt.timezone.utc)
    start_year = json.loads(args.registry.read_text())["historical_policy"]["complete_window_start_year"]
    queue: list[tuple[str, str, int]] = [(source["discovery_uri"], "official IR root", 0)]
    seen: set[str] = set()
    observations: list[dict[str, Any]] = []
    documents: list[dict[str, Any]] = []
    index_pages = 0
    accepted_hashes: set[str] = set()
    accepted_uris: dict[str, str] = {}
    while queue and index_pages < args.max_index_pages and len(documents) < args.max_documents:
        requested_uri, label, depth = queue.pop(0)
        uri = canonical_uri(requested_uri)
        if uri in seen:
            continue
        seen.add(uri)
        host = urllib.parse.urlsplit(uri).hostname or ""
        if not ir_discover.source_uri_allowed(uri, source):
            continue
        robots_allowed, robots_observation = robots.check(uri, source["company_id"], source["allowed_hosts"])
        if robots_observation is not None:
            observations.append(robots_observation)
        if not robots_allowed:
            observations.append(
                {
                    "observation_id": "obs-" + hashlib.sha256((source["company_id"] + uri + "disallowed").encode()).hexdigest()[:24],
                    "company_id": source["company_id"],
                    "requested_uri": uri,
                    "observed_at": dt.datetime.now(dt.timezone.utc).isoformat(),
                    "disposition": "disallowed",
                    "attempt": 1,
                    "elapsed_ms": 0,
                    "failure_class": "robots_policy",
                }
            )
            continue
        result, payload = fetch(
            uri,
            user_agent,
            args.timeout,
            args.max_response_bytes,
            args.allowed_media_types,
            source["allowed_hosts"],
            source,
            budget,
        )
        observed_at = dt.datetime.now(dt.timezone.utc).isoformat()
        observation = {
            "observation_id": "obs-" + hashlib.sha256((source["company_id"] + uri + observed_at).encode()).hexdigest()[:24],
            "company_id": source["company_id"],
            "requested_uri": uri,
            "observed_at": observed_at,
            "attempt": 1,
            "elapsed_ms": result.get("elapsed_ms", 0),
            **result,
        }
        observations.append(observation)
        if result.get("disposition") != "accepted" or not payload:
            continue
        media_type = result["media_type"]
        digest = result["content_sha256"]
        raw_path = store_payload(args.raw_store, digest, payload)
        observation["raw_store_key"] = str(raw_path.relative_to(args.raw_store))
        title = label or Path(urllib.parse.urlsplit(uri).path).name or source["issuer"]
        is_root = uri.rstrip("/") == canonical_uri(source["discovery_uri"]).rstrip("/")
        is_material = is_material_candidate(uri, label)
        if media_type == "text/html":
            index_pages += 1
            parser = LinkParser()
            parser.feed(payload.decode("utf-8", errors="replace"))
            title = " ".join(parser.title_parts) or title
            if depth < args.max_depth:
                for item in normalize_links(result.get("final_uri", uri), parser):
                    child_host = urllib.parse.urlsplit(item["uri"]).hostname or ""
                    if (
                        ir_discover.source_uri_allowed(item["uri"], source)
                        and is_material_candidate(item["uri"], item["label"])
                        and eligible_year(item["uri"], item["label"], start_year)
                    ):
                        queue.append((item["uri"], item["label"], depth + 1))
        canonical = result.get("final_uri", uri)
        is_material = is_material and is_material_candidate(canonical, title)
        if not (is_root or is_material):
            continue
        document_type, authority_tier, promotional = classify(uri, label, title)
        if canonical in accepted_uris:
            observation["duplicate_document_id"] = accepted_uris[canonical]
            continue
        stable_identity = hashlib.sha256((source["company_id"] + "\n" + canonical).encode()).hexdigest()
        document_id = "ir-" + source["primary_ticker"].lower() + "-" + stable_identity[:20]
        if digest in accepted_hashes:
            duplicate_document_id = next(
                document["document_id"] for document in documents if document["content_sha256"] == digest
            )
            observation["duplicate_document_id"] = duplicate_document_id
            accepted_uris[canonical] = duplicate_document_id
            continue
        accepted_hashes.add(digest)
        accepted_uris[canonical] = document_id
        documents.append(
            {
                "schema_version": DOCUMENT_SCHEMA,
                "document_id": document_id,
                "company_id": source["company_id"],
                "title": title[:500],
                "document_type": document_type,
                "authority_tier": authority_tier,
                "issuer": source["issuer"],
                "source_uri": canonical,
                "canonical_uri": canonical,
                "discovery_uri": source["discovery_uri"],
                "media_type": media_type,
                "language": "en",
                "available_at": observed_at,
                "retrieved_at": observed_at,
                "audited": False,
                "filed_with_sec": False,
                "forward_looking": document_type in {"guidance_and_outlook", "prepared_remarks"},
                "promotional": promotional,
                "content_sha256": digest,
                "rights_class": source["rights_class"],
                "collector_version": COLLECTOR_VERSION,
                "parser_version": "pending",
                "classification_status": "quarantined" if source["rights_class"].endswith("pending_review") else "deterministic",
                "classification_reasons": ["evaluation-only pending rights review"] if source["rights_class"].endswith("pending_review") else ["URL and title rule"],
            }
        )
    return observations, documents, {
        "ticker": source["primary_ticker"],
        "started_at": started_at.isoformat(),
        "finished_at": dt.datetime.now(dt.timezone.utc).isoformat(),
        "visited_uri_count": len(seen),
        "index_page_count": index_pages,
        "document_count": len(documents),
        "queue_remaining": len(queue),
    }


def main() -> int:
    args = parse_args()
    registry_bytes = args.registry.read_bytes()
    registry = json.loads(registry_bytes)
    sources = ir_discover.validate_registry(registry)
    discovery = json.loads(args.discovery_report.read_text())
    registry_sha256 = hashlib.sha256(registry_bytes).hexdigest()
    if discovery.get("registry_sha256") != registry_sha256:
        raise SystemExit("discovery report does not match the source registry")
    dispositions = {item["primary_ticker"]: item["disposition"] for item in discovery["sources"]}
    selected = {value.strip().upper() for value in args.tickers.split(",") if value.strip()}
    if selected:
        sources = [source for source in sources if source["primary_ticker"] in selected]
    user_agent = os.environ.get(registry["collection_policy"]["user_agent_environment_variable"], "").strip()
    if not user_agent:
        raise SystemExit("descriptive User-Agent environment variable is required")
    if any(source["rights_class"].endswith("pending_review") for source in sources) and not args.evaluation_only_pending_rights:
        raise SystemExit("rights review is pending; use --evaluation-only-pending-rights only for private quarantined evidence")
    scope = {
        "registry_sha256": registry_sha256,
        "discovery_report_sha256": hashlib.sha256(args.discovery_report.read_bytes()).hexdigest(),
        "tickers": sorted(source["primary_ticker"] for source in sources),
        "max_index_pages": args.max_index_pages,
        "max_documents": args.max_documents,
        "max_depth": args.max_depth,
        "evaluation_only": args.evaluation_only_pending_rights,
    }
    checkpoint_dir = args.report_dir / "checkpoints"
    scope_path = checkpoint_dir / "scope.json"
    if args.resume:
        if not scope_path.is_file() or json.loads(scope_path.read_text()) != scope:
            raise SystemExit("resume checkpoint scope is missing or differs from this run")
    else:
        checkpoint_dir.mkdir(parents=True, exist_ok=True)
        scope_path.write_text(json.dumps(scope, indent=2, sort_keys=True) + "\n")
    budget = HostBudget(registry["collection_policy"]["default_requests_per_second"])
    robots = RobotsGuard(user_agent, args.timeout, budget)
    args.max_response_bytes = registry["collection_policy"]["max_response_bytes"]
    args.allowed_media_types = set(registry["collection_policy"]["allowed_media_types"])
    observations: list[dict[str, Any]] = []
    documents: list[dict[str, Any]] = []
    companies: list[dict[str, Any]] = []
    for source in sources:
        disposition = dispositions.get(source["primary_ticker"], "missing")
        if disposition not in {"root_verified", "head_verified_needs_body"}:
            companies.append({"ticker": source["primary_ticker"], "collection_disposition": "quarantined", "reason": disposition})
            continue
        ticker = source["primary_ticker"].lower()
        checkpoint_observations = checkpoint_dir / f"{ticker}-observations.jsonl"
        checkpoint_documents = checkpoint_dir / f"{ticker}-documents.jsonl"
        checkpoint_summary = checkpoint_dir / f"{ticker}-summary.json"
        if args.resume and checkpoint_observations.is_file() and checkpoint_documents.is_file() and checkpoint_summary.is_file():
            observations.extend(read_jsonl(checkpoint_observations))
            documents.extend(read_jsonl(checkpoint_documents))
            companies.append(json.loads(checkpoint_summary.read_text()))
            continue
        source_observations, source_documents, summary = crawl_source(source, user_agent, args, budget, robots)
        completed_summary = {**summary, "collection_disposition": "completed"}
        write_jsonl(checkpoint_observations, source_observations)
        write_jsonl(checkpoint_documents, source_documents)
        checkpoint_summary.write_text(json.dumps(completed_summary, indent=2, sort_keys=True) + "\n")
        observations.extend(source_observations)
        documents.extend(source_documents)
        companies.append(completed_summary)
    args.report_dir.mkdir(parents=True, exist_ok=True)
    write_jsonl(args.report_dir / "crawl-observations.jsonl", observations)
    write_jsonl(args.report_dir / "documents.jsonl", documents)
    report = {
        "schema_version": "signalforge/ir-collection-run/v1",
        "collector_version": COLLECTOR_VERSION,
        "registry_sha256": scope["registry_sha256"],
        "discovery_report_sha256": scope["discovery_report_sha256"],
        "evaluation_only": args.evaluation_only_pending_rights,
        "source_count": len(sources),
        "observation_count": len(observations),
        "document_count": len(documents),
        "companies": companies,
        "claim_boundary": "Pending-rights documents are quarantined evaluation artifacts and are not product-ready or redistributable.",
    }
    (args.report_dir / "collection-summary.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    print(json.dumps({key: report[key] for key in ("source_count", "observation_count", "document_count")}, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
