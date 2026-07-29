import { useMemo } from "react";
import { buildExecutionPresentation } from "../executionPresentation";
import type { ExecutionPlan, ExecutionStatus, SafeEvent } from "../types";
import { ArrowIcon, CheckIcon, ChipIcon, ShieldIcon, SparkIcon } from "./Icons";

type Props = {
  plan: ExecutionPlan | null;
  events: SafeEvent[];
  running: boolean;
  connection: "idle" | "live" | "recovering" | "unavailable";
  onAudit: () => void;
};

type CompactState = {
  label: string;
  detail: string;
  progress: number;
  status: ExecutionStatus | "complete";
  specialistCount: number;
};

export function CompactRunStatus({ plan, events, running, connection, onAudit }: Props) {
  const presentation = useMemo(() => plan ? buildExecutionPresentation(plan) : null, [plan]);
  const state = presentation
    ? fromPresentation(presentation, running)
    : fromEvents(events, running);
  const connectionIssue = connection === "recovering" || connection === "unavailable";
  const Icon = state.status === "failed" || state.status === "cancelled"
    ? ShieldIcon
    : state.status === "complete" || state.status === "passed"
      ? CheckIcon
      : state.status === "repairing" || state.status === "degraded"
        ? ShieldIcon
        : running
          ? SparkIcon
          : ChipIcon;

  return (
    <section
      className={`compact-run-status status-${state.status} ${running ? "is-running" : "is-settled"}`}
      aria-label="Research status"
    >
      <div className="compact-run-icon"><Icon /></div>
      <div className="compact-run-copy">
        <span className="eyebrow">{running ? "Research in progress" : "Accepted research"}</span>
        <div className="compact-run-announcement" role="status" aria-live="polite" aria-atomic="true">
          <strong>{connectionIssue ? "Execution detail is reconnecting" : state.label}</strong>
          <small>
            {connectionIssue
              ? "The accepted answer remains available while signed audit detail recovers."
              : state.detail}
          </small>
        </div>
        {running && (
          <div
            className="compact-progress-track"
            role="progressbar"
            aria-label="Research progress"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={state.progress}
          >
            <span style={{ width: `${state.progress}%` }} />
          </div>
        )}
      </div>
      {running && state.specialistCount > 0 && (
        <span className="compact-specialists">
          {state.specialistCount} specialist{state.specialistCount === 1 ? "" : "s"} in parallel
        </span>
      )}
      <button type="button" className="audit-entry" onClick={onAudit}>
        <span>How SignalForge reached this answer</span>
        <ArrowIcon />
      </button>
    </section>
  );
}

function fromPresentation(
  presentation: ReturnType<typeof buildExecutionPresentation>,
  running: boolean
): CompactState {
  const activePhase = presentation.phases.find((phase) =>
    phase.status === "running" || phase.status === "repairing"
  );
  const status = running ? presentation.status : settledStatus(presentation.status);
  const label = running
    ? phaseLabel(activePhase?.id)
    : settledLabel(presentation.status);
  const detail = running
    ? activePhase?.summary || "SignalForge is advancing the signed research plan."
    : settledDetail(presentation.status);

  return {
    label,
    detail,
    progress: presentation.percentage,
    status,
    specialistCount: running ? presentation.maxParallelSpecialists : 0
  };
}

function fromEvents(events: SafeEvent[], running: boolean): CompactState {
  if (!running) {
    return {
      label: "Research answer ready",
      detail: events.length
        ? "The accepted answer, evidence, calculations, and limitations remain available."
        : "The accepted answer and its governed proof remain available.",
      progress: 100,
      status: "complete",
      specialistCount: 0
    };
  }

  const latest = events.at(-1);
  const activeContexts = activeContextCount(events);
  const completed = events.filter((event) =>
    ["completed", "passed", "accepted", "released"].includes(event.status)
  ).length;
  const progress = events.length ? Math.min(92, Math.max(8, Math.round((completed / events.length) * 100))) : 4;

  return {
    label: eventPhaseLabel(latest?.type),
    detail: eventDetail(latest?.status),
    progress,
    status: eventStatus(latest?.status),
    specialistCount: activeContexts
  };
}

function phaseLabel(phase?: string) {
  const labels: Record<string, string> = {
    interpretation: "Understanding your question",
    planning: "Understanding your question",
    context: "Gathering governed evidence",
    tools: "Calculating with deterministic engines",
    review: "Challenging the emerging answer",
    synthesis: "Writing the research answer",
    memory: "Applying your privacy choice",
    release: "Checking the answer for release"
  };
  return labels[phase ?? ""] ?? "Advancing the research";
}

function eventPhaseLabel(type?: string) {
  const labels: Record<string, string> = {
    plan: "Understanding your question",
    interpretation: "Understanding your question",
    context: "Gathering governed evidence",
    retrieval: "Gathering governed evidence",
    tool: "Calculating with deterministic engines",
    engine: "Calculating with deterministic engines",
    review: "Challenging the emerging answer",
    critique: "Challenging the emerging answer",
    synthesis: "Writing the research answer",
    workspace: "Checking the answer for release",
    release: "Checking the answer for release"
  };
  return labels[type ?? ""] ?? "Advancing the research";
}

function eventDetail(status?: string) {
  if (status === "repairing") return "A bounded contract repair is in progress.";
  if (status === "degraded") return "SignalForge is preserving only the evidence-supported subset.";
  if (status === "failed") return "The run stopped safely; the previous accepted answer remains visible.";
  return "Specialists, tools, and reviewers are working within the signed plan.";
}

function eventStatus(status?: string): ExecutionStatus {
  const known = new Set<ExecutionStatus>([
    "pending", "ready", "running", "passed", "failed", "degraded",
    "repairing", "skipped", "cancelled", "withheld", "unavailable"
  ]);
  if (status && known.has(status as ExecutionStatus)) return status as ExecutionStatus;
  return status === "completed" || status === "accepted" || status === "released" ? "passed" : "running";
}

function activeContextCount(events: SafeEvent[]) {
  const active = new Set<string>();
  for (const event of events) {
    if (event.type !== "context" || !event.step_id) continue;
    if (event.status === "started" || event.status === "running") active.add(event.step_id);
    if (["completed", "failed", "cancelled"].includes(event.status)) active.delete(event.step_id);
  }
  return active.size;
}

function settledStatus(status: ExecutionStatus): ExecutionStatus | "complete" {
  return status === "failed" || status === "cancelled" || status === "degraded" || status === "withheld"
    ? status
    : "complete";
}

function settledLabel(status: ExecutionStatus) {
  if (status === "failed" || status === "cancelled") return "Research stopped safely";
  if (status === "degraded" || status === "withheld") return "Bounded answer ready";
  return "Research answer ready";
}

function settledDetail(status: ExecutionStatus) {
  if (status === "failed" || status === "cancelled") {
    return "No partial replacement was released; the previous accepted answer remains available.";
  }
  if (status === "degraded" || status === "withheld") {
    return "Only the evidence-supported subset was released, with limitations kept visible.";
  }
  return "The accepted answer, evidence, calculations, and limitations remain available.";
}
