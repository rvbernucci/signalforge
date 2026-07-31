import type { ExecutionPlan, Projection, SafeEvent, WorkspaceReadiness } from "../types";
import { ArrowIcon, CheckIcon, CloseIcon, DocumentIcon, ReceiptIcon, ShieldIcon } from "./Icons";
import { LiveExecutionPlan } from "./LiveExecutionPlan";
import { useDialogFocus } from "../hooks/useDialogFocus";

type Props = {
  projection: Projection;
  readiness?: WorkspaceReadiness;
  plan: ExecutionPlan | null;
  events: SafeEvent[];
  traceID?: string;
  running: boolean;
  connection: "idle" | "live" | "recovering" | "unavailable";
  open: boolean;
  suspended: boolean;
  judgeMode: boolean;
  intelligenceAvailable: boolean;
  onClose: () => void;
  onJudgeMode: (enabled: boolean) => void;
  onProof: (refs: string[]) => void;
  onCalculations: (refs: string[]) => void;
  onMissionControl: () => void;
};

export function AuditWorkspace({
  projection,
  readiness,
  plan,
  events,
  traceID,
  running,
  connection,
  open,
  suspended,
  judgeMode,
  intelligenceAvailable,
  onClose,
  onJudgeMode,
  onProof,
  onCalculations,
  onMissionControl
}: Props) {
  const focus = useDialogFocus(open, onClose);

  return (
    <>
      <button
        className={`drawer-scrim audit-workspace-scrim ${open ? "is-open" : ""}`}
        onClick={onClose}
        aria-hidden="true"
        tabIndex={-1}
      />
      <aside
        ref={focus.panelRef}
        className={`audit-workspace ${open ? "is-open" : ""}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="audit-workspace-title"
        aria-hidden={!open || suspended}
        inert={!open || suspended}
        onKeyDown={focus.onKeyDown}
      >
        <header className="audit-workspace-header">
          <div>
            <span className="eyebrow">{judgeMode ? "Judge audit view" : "Governed answer audit"}</span>
            <h2 id="audit-workspace-title">How SignalForge reached this answer</h2>
            <p>One accepted research projection, progressively disclosed from answer to signed execution detail.</p>
          </div>
          <div className="audit-header-actions">
            <button
              type="button"
              className="judge-mode-toggle"
              aria-pressed={judgeMode}
              onClick={() => onJudgeMode(!judgeMode)}
            >
              {judgeMode ? "Hide judge orientation" : "Judge orientation"}
            </button>
            <button ref={focus.initialFocusRef} className="icon-button" onClick={onClose} aria-label="Close audit workspace">
              <CloseIcon />
            </button>
          </div>
        </header>

        <div className="audit-workspace-body">
          {judgeMode && <JudgeOrientation />}

          <section className="audit-identity" aria-label="Exact accepted-run identity">
            <div><span>Run</span><code>{projection.run_id}</code></div>
            <div><span>Request</span><code>{projection.request_id}</code></div>
            {traceID && <div><span>Trace</span><code>{traceID}</code></div>}
            {plan?.projection_sha256 && <div><span>Projection</span><code>{plan.projection_sha256}</code></div>}
            <div><span>Model</span><code>{projection.execution.model}</code></div>
            <div><span>Runtime</span><code>{projection.execution.runtime_label}</code></div>
            <div><span>As of</span><time dateTime={projection.as_of}>{formatDateTime(projection.as_of)}</time></div>
          </section>

          {readiness && (
            <details className="audit-artifact-identities">
              <summary>Exact source and artifact identities</summary>
              <div>
                <span>Source commit</span><code>{readiness.identities.source}</code>
                <span>Application artifact</span><code>{readiness.identities.application}</code>
                <span>Inference runtime</span><code>{readiness.identities.runtime}</code>
                <span>Model artifact</span><code>{readiness.identities.model}</code>
                <span>Served model</span><code>{readiness.identities.served_model}</code>
                <span>Configuration</span><code>{readiness.identities.configuration_sha256}</code>
                <span>Governed data</span><code>{readiness.identities.data_sha256}</code>
              </div>
            </details>
          )}

          <section className="audit-shortcuts" aria-label="Audit shortcuts">
            <button type="button" onClick={() => onProof([])}>
              <DocumentIcon />
              <span><strong>Inspect source evidence</strong><small>{projection.evidence.length} answer-bound cards</small></span>
              <ArrowIcon />
            </button>
            <button type="button" onClick={() => onCalculations([])}>
              <ReceiptIcon />
              <span><strong>Inspect calculations</strong><small>{projection.calculations.length} deterministic receipts</small></span>
              <ArrowIcon />
            </button>
            {intelligenceAvailable && (
              <button type="button" onClick={onMissionControl}>
                <ShieldIcon />
                <span><strong>Open Mission Control</strong><small>Agents, routes, lineage, latency, and tokens</small></span>
                <ArrowIcon />
              </button>
            )}
          </section>

          {plan ? (
            <LiveExecutionPlan
              plan={plan}
              traceID={traceID}
              running={running}
              connection={connection}
              onProof={onProof}
              onCalculations={onCalculations}
              onLineage={onMissionControl}
            />
          ) : (
            <HistoricalExecutionRecord events={events} />
          )}

          <section className="audit-privacy-boundary">
            <ShieldIcon />
            <div>
              <strong>Privacy-safe disclosure</strong>
              <p>Audit surfaces expose governed metadata, hashes, evidence locators, and deterministic receipts. They never render credentials, private memory, raw model prompts or outputs, or hidden chain-of-thought.</p>
            </div>
          </section>
        </div>
      </aside>
    </>
  );
}

function JudgeOrientation() {
  const capabilities = [
    ["Reasoning and planning", "Interpretation, bounded plan, review, and release gates"],
    ["Tool invocation", "Deterministic financial engines with receipt-level proof"],
    ["Knowledge retrieval", "Answer-used evidence, source lineage, and freshness"],
    ["Local memory and permissions", "Explicit opt-in retention with delete and export controls"],
    ["Local and hybrid inference", "Local Radeon routes and optional Radeon API specialists remain distinct"]
  ];
  return (
    <section className="judge-orientation" aria-label="Track 2 judge orientation">
      <header>
        <div><span className="eyebrow">Two-minute orientation</span><h3>Track 2 capabilities in the product</h3></div>
        <p>These labels map visible behavior to evidence. They do not claim or pre-assign a score.</p>
      </header>
      <div>
        {capabilities.map(([label, evidence]) => (
          <article key={label}><CheckIcon /><span><strong>{label}</strong><small>{evidence}</small></span></article>
        ))}
      </div>
    </section>
  );
}

function HistoricalExecutionRecord({ events }: { events: SafeEvent[] }) {
  const safeEvents = [...events].sort((left, right) => left.sequence - right.sequence);
  return (
    <section className="historical-execution" aria-labelledby="historical-execution-title">
      <header>
        <div>
          <span className="eyebrow">Credential-free fixture audit</span>
          <h3 id="historical-execution-title">Recorded governed journey</h3>
        </div>
        <span>{safeEvents.length} lifecycle events</span>
      </header>
      <p className="historical-boundary">This accepted fixture predates the signed phase-plan projection. Its bounded lifecycle record is shown without reconstructing or inventing missing execution detail.</p>
      <ol>
        {safeEvents.map((event) => (
          <li key={`${event.sequence}-${event.type}-${event.step_id ?? "run"}`}>
            <span>{String(event.sequence).padStart(2, "0")}</span>
            <div>
              <strong>{eventLabel(event.type)}</strong>
              <small>{event.step_id ? humanize(event.step_id) : "Governed runtime"}</small>
            </div>
            <em>{humanize(event.status)}</em>
            <time dateTime={event.at}>{formatTime(event.at)}</time>
          </li>
        ))}
      </ol>
    </section>
  );
}

function eventLabel(type: string) {
  const labels: Record<string, string> = {
    plan: "Understanding and plan accepted",
    context: "Specialist evidence context",
    tool: "Deterministic calculation",
    engine: "Deterministic calculation",
    review: "Independent challenge",
    critique: "Independent challenge",
    synthesis: "Answer synthesis",
    release: "Release decision",
    workspace: "Workspace completion"
  };
  return labels[type] ?? humanize(type);
}

function humanize(value: string) {
  return value.replace(/[._:/-]+/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "UTC"
  }).format(new Date(value));
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    timeZone: "UTC"
  }).format(new Date(value));
}
