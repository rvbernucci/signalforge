import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ExecutionPlan, SafeEvent } from "../types";
import { CompactRunStatus } from "./CompactRunStatus";

const plan: ExecutionPlan = {
  schema_version: "signalforge/execution-plan/v1",
  run_id: "run-compact",
  request_id: "request-compact",
  status: "running",
  created_at: "2026-07-29T12:00:00Z",
  total_steps: 5,
  terminal_steps: 2,
  progress_ratio: 0.4,
  max_parallel_specialists: 4,
  current_wave: 1,
  phases: [
    phase("interpretation", 1, "passed"),
    phase("planning", 2, "passed"),
    phase("context", 3, "running"),
    phase("tools", 4, "pending"),
    phase("review", 5, "pending"),
    phase("synthesis", 6, "pending"),
    phase("memory", 7, "pending"),
    phase("release", 8, "pending")
  ],
  steps: [{
    step_id: "context-accounting",
    parent_phase_id: "context",
    phase: "context",
    kind: "context",
    safe_label: "Accounting specialist",
    safe_objective: "Compile governed evidence.",
    mandatory: true,
    status: "running",
    wave: 1,
    checklist: [],
    attempt: 1,
    max_attempts: 2,
    safe_summary: "Accounting evidence is being compiled."
  }],
  last_sequence: 3,
  projection_sha256: "a".repeat(64)
};

describe("CompactRunStatus", () => {
  it("translates the signed phase into bounded user language", () => {
    const onAudit = vi.fn();
    render(
      <CompactRunStatus
        plan={plan}
        events={[]}
        running
        connection="live"
        onAudit={onAudit}
      />
    );

    expect(screen.getByRole("status")).toHaveTextContent("Gathering governed evidence");
    expect(screen.getByRole("progressbar", { name: "Research progress" })).toHaveAttribute("aria-valuenow", "40");
    expect(screen.getByText("4 specialists in parallel")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "How SignalForge reached this answer" }));
    expect(onAudit).toHaveBeenCalledOnce();
  });

  it("collapses completed progress and preserves the audit action", () => {
    render(
      <CompactRunStatus
        plan={{ ...plan, status: "passed", terminal_steps: 5, progress_ratio: 1 }}
        events={[]}
        running={false}
        connection="live"
        onAudit={vi.fn()}
      />
    );

    expect(screen.getByRole("status")).toHaveTextContent("Research answer ready");
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "How SignalForge reached this answer" })).toBeVisible();
  });

  it("states that audit recovery cannot remove the accepted answer", () => {
    render(
      <CompactRunStatus
        plan={plan}
        events={[]}
        running
        connection="unavailable"
        onAudit={vi.fn()}
      />
    );

    expect(screen.getByRole("status")).toHaveTextContent("Execution detail is reconnecting");
    expect(screen.getByRole("status")).toHaveTextContent("accepted answer remains available");
  });

  it("uses only bounded event types when a signed phase plan is unavailable", () => {
    const events: SafeEvent[] = [{
      sequence: 1,
      run_id: "run-event",
      step_id: "context-accounting",
      type: "context",
      status: "started",
      at: "2026-07-29T12:00:00Z",
      attributes: { raw_prompt: "must-not-render", credential: "must-not-render" }
    }];
    render(
      <CompactRunStatus
        plan={null}
        events={events}
        running
        connection="live"
        onAudit={vi.fn()}
      />
    );

    expect(screen.getByRole("status")).toHaveTextContent("Gathering governed evidence");
    expect(screen.getByText("1 specialist in parallel")).toBeInTheDocument();
    expect(screen.queryByText("must-not-render")).not.toBeInTheDocument();
  });
});

function phase(phaseID: string, order: number, status: ExecutionPlan["status"]): ExecutionPlan["phases"][number] {
  return {
    phase_id: phaseID,
    order,
    safe_label: phaseID,
    safe_objective: `Governed ${phaseID}.`,
    mandatory: phaseID !== "memory",
    status,
    step_ids: phaseID === "context" ? ["context-accounting"] : [],
    safe_summary: `${phaseID} is ${status}.`
  };
}
