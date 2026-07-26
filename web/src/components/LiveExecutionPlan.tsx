import { useEffect, useState, type CSSProperties } from "react";
import type {
  ExecutionChecklistItem,
  ExecutionPhase,
  ExecutionPlan,
  ExecutionStatus,
  ExecutionStep
} from "../types";
import { CheckIcon, ChipIcon, ShieldIcon, SparkIcon } from "./Icons";

type Props = {
  plan: ExecutionPlan | null;
  traceID?: string;
  running: boolean;
  connection?: "idle" | "live" | "recovering" | "unavailable";
  onProof: (refs: string[]) => void;
  onCalculations: (refs: string[]) => void;
  onLineage: () => void;
};

const terminalStatuses = new Set<ExecutionStatus>([
  "passed", "failed", "degraded", "skipped", "cancelled", "withheld", "unavailable"
]);

export function LiveExecutionPlan({
  plan,
  traceID,
  running,
  connection = "live",
  onProof,
  onCalculations,
  onLineage
}: Props) {
  if (!plan) return null;
  const percentage = Math.round(plan.progress_ratio * 100);
  const active = plan.steps.find((step) => step.status === "running" || step.status === "repairing");
  const headline = active?.safe_label ?? (terminalStatuses.has(plan.status) ? statusLabel(plan.status) : "Preparing the governed plan");
  const phases = [...plan.phases].sort((left, right) => left.order - right.order);

  return (
    <section className={`execution-plan plan-${plan.status}`} aria-labelledby="execution-plan-title">
      <header className="execution-plan-header" aria-live="polite">
        <div
          className="execution-progress-dial"
          style={{ "--plan-progress": `${percentage * 3.6}deg` } as CSSProperties}
          role="progressbar"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={percentage}
          aria-valuetext={`${plan.terminal_steps} of ${plan.total_steps} planned steps closed`}
          aria-label={`${percentage}% ${plan.status === "passed" ? "complete" : "closed"}`}
        >
          <span>{percentage}</span><small>%</small>
        </div>
        <div className="execution-plan-heading">
          <span className="eyebrow">Live agent execution</span>
          <h2 id="execution-plan-title">{headline}</h2>
          <p>Expand any stage to inspect its bounded objective, dependencies, release checks, and safe artifact references.</p>
        </div>
        <div className="execution-plan-stats">
          <span><strong>{plan.terminal_steps}</strong> / {plan.total_steps} closed</span>
          <span><strong>{plan.max_parallel_specialists || 0}</strong> parallel specialists</span>
          <span className={`execution-status status-${plan.status}`}>{statusLabel(plan.status)}</span>
          {(connection === "recovering" || connection === "unavailable") && (
            <span className="execution-status status-unavailable" role="status">Reconnecting to signed plan</span>
          )}
        </div>
      </header>

      <ol className="execution-phase-rail" aria-label="Execution phases">
        {phases.map((phase) => {
          const complete = terminalStatuses.has(phase.status);
          const current = phase.status === "running" || phase.status === "repairing";
          return (
            <li
              key={phase.phase_id}
              className={current ? "is-current" : complete ? "is-complete" : ""}
              aria-current={current ? "step" : undefined}
            >
              {phase.safe_label}
            </li>
          );
        })}
      </ol>

      <div className="execution-phase-groups">
        {phases.map((phase) => (
          <ExecutionPhaseGroup
            key={phase.phase_id}
            phase={phase}
            steps={plan.steps.filter((step) => step.parent_phase_id === phase.phase_id)}
            allSteps={plan.steps}
            running={running}
            onProof={onProof}
            onCalculations={onCalculations}
            onLineage={onLineage}
          />
        ))}
      </div>

      <footer className="execution-plan-footer">
        <span><ShieldIcon /> This view exposes governed state and evidence references, never private chain-of-thought.</span>
        <span className="execution-plan-identities" aria-label="Workspace run identity">
          <code title={plan.run_id}>run {shortID(plan.run_id)}</code>
          {traceID && <code title={traceID}>trace {shortID(traceID)}</code>}
          <code title={plan.projection_sha256}>projection {shortID(plan.projection_sha256)}</code>
        </span>
      </footer>
    </section>
  );
}

function shortID(value: string): string {
  return value.length > 12 ? value.slice(0, 12) : value;
}

function ExecutionPhaseGroup({
  phase,
  steps,
  allSteps,
  running,
  onProof,
  onCalculations,
  onLineage
}: {
  phase: ExecutionPhase;
  steps: ExecutionStep[];
  allSteps: ExecutionStep[];
  running: boolean;
  onProof: (refs: string[]) => void;
  onCalculations: (refs: string[]) => void;
  onLineage: () => void;
}) {
  const status = phase.status;
  const terminal = steps.filter((step) => terminalStatuses.has(step.status)).length;
  const needsAttention = ["failed", "degraded", "withheld", "unavailable", "cancelled"].includes(status);
  const automaticOpen = status === "running" || status === "repairing" || needsAttention;
  const [expanded, setExpanded] = useState(automaticOpen);

  useEffect(() => {
    if (automaticOpen) setExpanded(true);
    if (!running && status === "passed") setExpanded(false);
  }, [automaticOpen, running, status]);

  return (
    <details
      className={`execution-phase-group status-${status}`}
      aria-label={`${phase.safe_label} phase`}
      open={expanded}
      onToggle={(event) => setExpanded(event.currentTarget.open)}
    >
      <summary
        onKeyDown={(event) => {
          if (event.key !== "Enter" && event.key !== " ") return;
          event.preventDefault();
          setExpanded((current) => !current);
        }}
      >
        <span>
          <small>Execution phase</small>
          <strong>{phase.safe_label}</strong>
        </span>
        <em>{terminal} / {steps.length} closed</em>
        <span className={`execution-status status-${status}`}>{statusLabel(status)}</span>
      </summary>
      <div className="execution-phase-objective">
        <span className="eyebrow">Phase objective</span>
        <p>{phase.safe_objective}</p>
        <small>{phase.safe_summary}</small>
      </div>
      {steps.length > 0 ? (
        <ol
          className="execution-step-list"
          aria-label={`${phase.safe_label} execution stages`}
          start={allSteps.indexOf(steps[0]) + 1}
        >
          {steps.map((step) => (
            <ExecutionStepCard
              key={step.step_id}
              step={step}
              index={allSteps.indexOf(step)}
              running={running}
              onProof={onProof}
              onCalculations={onCalculations}
              onLineage={onLineage}
            />
          ))}
        </ol>
      ) : (
        <p className="execution-phase-empty">
          No standalone step was required; embedded activity remains governed by this phase.
        </p>
      )}
    </details>
  );
}

function ExecutionStepCard({
  step,
  index,
  running,
  onProof,
  onCalculations,
  onLineage
}: {
  step: ExecutionStep;
  index: number;
  running: boolean;
  onProof: (refs: string[]) => void;
  onCalculations: (refs: string[]) => void;
  onLineage: () => void;
}) {
  const automaticOpen = step.status === "running" || step.status === "repairing" ||
    (!running && ["failed", "degraded", "withheld", "unavailable"].includes(step.status));
  const [expanded, setExpanded] = useState(automaticOpen);
  useEffect(() => {
    if (automaticOpen) setExpanded(true);
    if (!running && step.status === "passed") setExpanded(false);
  }, [automaticOpen, running, step.status]);
  const references = unique([
    ...(step.reference_ids ?? []),
    ...step.checklist.flatMap((item) => item.reference_ids ?? [])
  ]);
  const proofReferences = unique(step.checklist
    .filter((item) => item.authority === "retrieval")
    .flatMap((item) => item.reference_ids ?? []));
  const calculationReferences = unique(step.checklist
    .filter((item) => item.authority === "engine")
    .flatMap((item) => item.reference_ids ?? []));

  return (
    <li className="execution-step-list-item">
      <details
        className={`execution-step status-${step.status}`}
        aria-label={`${step.safe_label} step`}
        open={expanded}
        onToggle={(event) => setExpanded(event.currentTarget.open)}
      >
        <summary
          onKeyDown={(event) => {
            if (event.key !== "Enter" && event.key !== " ") return;
            event.preventDefault();
            setExpanded((current) => !current);
          }}
        >
          <span className="execution-step-index">{String(index + 1).padStart(2, "0")}</span>
          <span className="execution-step-icon"><StatusIcon status={step.status} /></span>
          <span className="execution-step-title">
            <small>{humanize(step.phase)}{step.wave ? ` · wave ${step.wave}` : ""}</small>
            <strong>{step.safe_label}</strong>
            <em>{step.safe_summary}</em>
          </span>
          <span className={`execution-status status-${step.status}`}>{statusLabel(step.status)}</span>
        </summary>

        <div className="execution-step-detail">
          <div className="execution-objective">
            <span className="eyebrow">Bounded objective</span>
            <p>{step.safe_objective}</p>
          </div>
          <dl className="execution-metadata">
            <div><dt>Authority</dt><dd>{humanize(step.role_id ?? "runtime")}</dd></div>
            <div><dt>Route</dt><dd className="execution-route-value"><span className="execution-route-badge">{humanize(step.route ?? step.route_reason_code ?? "governed local path")}</span></dd></div>
            <div><dt>Depends on</dt><dd className="execution-dependency-value">{step.depends_on?.length ? step.depends_on.map(humanize).join(", ") : "No unresolved dependency"}</dd></div>
            <div><dt>Attempt</dt><dd>{step.attempt} / {step.max_attempts}</dd></div>
            <div><dt>Duration</dt><dd>{formatDuration(step.duration_ms)}</dd></div>
            <div><dt>Artifacts</dt><dd>{references.length || "None released"}</dd></div>
            {(step.failure_code || step.degradation_code) && (
              <div><dt>Outcome code</dt><dd>{humanize(step.failure_code ?? step.degradation_code ?? "")}</dd></div>
            )}
          </dl>

          <ul className="execution-checklist" aria-label={`${step.safe_label} checklist`}>
            {step.checklist.map((item) => <ChecklistRow key={item.check_id} item={item} />)}
          </ul>

          <div className="execution-step-actions">
            {proofReferences.length > 0 && <button onClick={() => onProof(proofReferences)}>Open evidence</button>}
            {calculationReferences.length > 0 && (
              <button onClick={() => onCalculations(calculationReferences)}>Open calculations</button>
            )}
            <button onClick={onLineage}>Inspect lineage</button>
            {references.length > 0 && <code>{references.join(" · ")}</code>}
          </div>
        </div>
      </details>
    </li>
  );
}

function ChecklistRow({ item }: { item: ExecutionChecklistItem }) {
  return (
    <li className={`execution-check status-${item.status}`}>
      <StatusIcon status={item.status} />
      <span><strong>{item.label}</strong><small>{item.safe_detail ?? `${humanize(item.authority)} check is ${statusLabel(item.status).toLowerCase()}.`}</small></span>
      <em>{humanize(item.authority)} · {item.required ? "required" : "supporting"}</em>
    </li>
  );
}

function StatusIcon({ status }: { status: ExecutionStatus }) {
  if (status === "passed") return <CheckIcon />;
  if (status === "failed" || status === "withheld" || status === "cancelled") return <ShieldIcon />;
  if (status === "running" || status === "repairing") return <SparkIcon />;
  return <ChipIcon />;
}

function statusLabel(status: ExecutionStatus): string {
  const labels: Record<ExecutionStatus, string> = {
    pending: "Pending",
    ready: "Ready",
    running: "In progress",
    passed: "Verified",
    failed: "Stopped safely",
    degraded: "Bounded subset",
    repairing: "Repairing",
    skipped: "Skipped",
    cancelled: "Cancelled",
    withheld: "Withheld",
    unavailable: "Unavailable"
  };
  return labels[status];
}

function humanize(value: string): string {
  return value
    .replace(/\/v\d+$/, "")
    .replaceAll(/[._/-]/g, " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function formatDuration(duration = 0): string {
  if (duration <= 0) return "Not recorded";
  if (duration < 1000) return `${duration} ms`;
  return `${(duration / 1000).toFixed(duration >= 10000 ? 0 : 1)} s`;
}

function unique(values: string[]): string[] {
  return [...new Set(values.filter(Boolean))];
}
