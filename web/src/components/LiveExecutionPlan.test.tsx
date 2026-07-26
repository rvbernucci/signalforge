import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ExecutionPlan } from "../types";
import { LiveExecutionPlan } from "./LiveExecutionPlan";

const phases: ExecutionPlan["phases"] = [
  {
    phase_id: "interpretation", order: 1, safe_label: "Interpretation",
    safe_objective: "Resolve the governed request.", mandatory: true, status: "passed",
    step_ids: ["interpret-request"], safe_summary: "The request passed."
  },
  {
    phase_id: "planning", order: 2, safe_label: "Planning",
    safe_objective: "Freeze the bounded plan.", mandatory: true, status: "passed",
    step_ids: [], safe_summary: "The plan passed."
  },
  {
    phase_id: "context", order: 3, safe_label: "Context",
    safe_objective: "Compile governed specialist context.", mandatory: true, status: "running",
    step_ids: ["context-01"], safe_summary: "Context activity is running."
  },
  {
    phase_id: "tools", order: 4, safe_label: "Tools",
    safe_objective: "Release validated deterministic receipts.", mandatory: false, status: "passed",
    step_ids: [], safe_summary: "One tool activity passed."
  },
  {
    phase_id: "review", order: 5, safe_label: "Review",
    safe_objective: "Independently challenge the evidence.", mandatory: true, status: "pending",
    step_ids: [], safe_summary: "Review is waiting."
  },
  {
    phase_id: "synthesis", order: 6, safe_label: "Synthesis",
    safe_objective: "Compose one bounded answer.", mandatory: true, status: "pending",
    step_ids: ["synthesis-01"], safe_summary: "Synthesis is waiting."
  },
  {
    phase_id: "memory", order: 7, safe_label: "Memory",
    safe_objective: "Apply explicit local retention.", mandatory: false, status: "pending",
    step_ids: [], safe_summary: "Memory preference is waiting."
  },
  {
    phase_id: "release", order: 8, safe_label: "Release",
    safe_objective: "Release only after mandatory gates pass.", mandatory: true, status: "pending",
    step_ids: [], safe_summary: "Release is waiting."
  }
];

const plan: ExecutionPlan = {
  schema_version: "signalforge/execution-plan/v1",
  run_id: "run-test",
  request_id: "request-test",
  plan_id: "plan-test",
  status: "running",
  created_at: "2026-07-25T12:00:00Z",
  started_at: "2026-07-25T12:00:00Z",
  total_steps: 3,
  terminal_steps: 1,
  progress_ratio: 1 / 3,
  max_parallel_specialists: 4,
  current_wave: 1,
  phases,
  last_sequence: 3,
  projection_sha256: "a".repeat(64),
  steps: [
    {
      step_id: "interpret-request",
      parent_phase_id: "interpretation",
      phase: "interpretation",
      kind: "control",
      safe_label: "Interpret request",
      safe_objective: "Identify the governed research intent and scope.",
      role_id: "request-interpreter/v1",
      mandatory: true,
      status: "passed",
      checklist: [{
        check_id: "request-contract",
        label: "Validate request contract",
        status: "passed",
        authority: "contract",
        required: true,
        safe_detail: "Request envelope accepted."
      }],
      attempt: 1,
      max_attempts: 1,
      safe_summary: "The governed request contract passed."
    },
    {
      step_id: "context-01",
      parent_phase_id: "context",
      phase: "context",
      kind: "context",
      safe_label: "Accounting Reporting",
      safe_objective: "Check accounting evidence.",
      role_id: "accounting-reporting/v1",
      wave: 1,
      depends_on: ["interpret-request"],
      mandatory: true,
      status: "running",
      route_reason_code: "intent_requires_specialist",
      checklist: [
        {
          check_id: "evidence-authority",
          label: "Authorize evidence context",
          status: "running",
          authority: "retrieval",
          required: true,
          reference_ids: ["evidence-1"]
        },
        {
          check_id: "tool-receipt",
          label: "Free cash flow receipt validated",
          status: "passed",
          authority: "engine",
          required: false,
          reference_ids: ["receipt-1"]
        }
      ],
      attempt: 1,
      max_attempts: 2,
      reference_ids: ["packet-1"],
      safe_summary: "The specialist is compiling governed context."
    },
    {
      step_id: "synthesis-01",
      parent_phase_id: "synthesis",
      phase: "synthesis",
      kind: "synthesis",
      safe_label: "Final synthesis",
      safe_objective: "Release one evidence-grounded answer.",
      role_id: "final-research-analyst/v1",
      depends_on: ["context-01"],
      mandatory: true,
      status: "pending",
      checklist: [],
      attempt: 0,
      max_attempts: 2,
      safe_summary: "Waiting for its dependencies."
    }
  ]
};

describe("LiveExecutionPlan", () => {
  it("shows live progress and expands each governed stage", () => {
    const onProof = vi.fn();
    const onCalculations = vi.fn();
    const onLineage = vi.fn();
    render(
      <LiveExecutionPlan
        plan={plan}
        traceID="80c42f9bdbda413b9f86413db94ed20a"
        running
        onProof={onProof}
        onCalculations={onCalculations}
        onLineage={onLineage}
      />
    );

    expect(screen.getByRole("heading", { name: "Accounting Reporting" })).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "33% closed" })).toHaveAttribute("aria-valuetext", "1 of 3 planned steps closed");
    expect(screen.getByText((_, element) => element?.textContent === "1 / 3 closed")).toBeInTheDocument();
    expect(screen.getByText("Check accounting evidence.")).toBeVisible();
    expect(screen.getByText("Engine · supporting")).toBeVisible();
    const phaseRail = screen.getByRole("list", { name: "Execution phases" });
    expect(phaseRail.querySelectorAll("li")).toHaveLength(8);
    expect(Array.from(phaseRail.querySelectorAll("li"), (item) => item.textContent)).toEqual([
      "Interpretation", "Planning", "Context", "Tools", "Review", "Synthesis", "Memory", "Release"
    ]);
    expect(phaseRail.querySelector('[aria-current="step"]')).toHaveTextContent("Context");
    expect(screen.getByLabelText("Workspace run identity")).toHaveTextContent("run run-test");
    expect(screen.getByLabelText("Workspace run identity")).toHaveTextContent("trace 80c42f9bdbda");

    fireEvent.click(screen.getByRole("button", { name: "Open evidence" }));
    expect(onProof).toHaveBeenCalledWith(["evidence-1"]);
    fireEvent.click(screen.getByRole("button", { name: "Open calculations" }));
    expect(onCalculations).toHaveBeenCalledWith(["receipt-1"]);
    fireEvent.click(screen.getAllByRole("button", { name: "Inspect lineage" })[1]);
    expect(onLineage).toHaveBeenCalledOnce();
  });

  it("states the privacy boundary explicitly", () => {
    render(
      <LiveExecutionPlan
        plan={plan}
        running
        onProof={vi.fn()}
        onCalculations={vi.fn()}
        onLineage={vi.fn()}
      />
    );
    expect(screen.getByText(/never private chain-of-thought/i)).toBeInTheDocument();
    expect(screen.queryByText(/raw prompt/i)).not.toBeInTheDocument();
  });

  it("keeps the signed plan visible while the event stream reconnects", () => {
    render(
      <LiveExecutionPlan
        plan={plan}
        running
        connection="recovering"
        onProof={vi.fn()}
        onCalculations={vi.fn()}
        onLineage={vi.fn()}
      />
    );
    expect(screen.getByRole("status")).toHaveTextContent("Reconnecting to signed plan");
    expect(screen.getByRole("heading", { name: "Accounting Reporting" })).toBeInTheDocument();
  });

  it("lets phases and their steps expand independently", () => {
    render(
      <LiveExecutionPlan
        plan={plan}
        running
        onProof={vi.fn()}
        onCalculations={vi.fn()}
        onLineage={vi.fn()}
      />
    );
    const phase = screen.getByLabelText("Interpretation phase");
    const step = screen.getByLabelText("Interpret request step");
    expect(phase).not.toHaveAttribute("open");

    fireEvent.click(phase.querySelector("summary")!);
    expect(phase).toHaveAttribute("open");
    expect(step).not.toHaveAttribute("open");

    fireEvent.click(step.querySelector("summary")!);
    expect(step).toHaveAttribute("open");
    expect(screen.getByText("Identify the governed research intent and scope.")).toBeVisible();
  });

  it("expands and collapses phases and steps from the keyboard", () => {
    render(
      <LiveExecutionPlan
        plan={plan}
        running
        onProof={vi.fn()}
        onCalculations={vi.fn()}
        onLineage={vi.fn()}
      />
    );
    const phase = screen.getByLabelText("Interpretation phase");
    const phaseSummary = phase.querySelector("summary")!;
    const step = screen.getByLabelText("Interpret request step");
    const stepSummary = step.querySelector("summary")!;

    fireEvent.keyDown(phaseSummary, { key: "Enter" });
    expect(phase).toHaveAttribute("open");
    fireEvent.keyDown(stepSummary, { key: " " });
    expect(step).toHaveAttribute("open");

    fireEvent.keyDown(stepSummary, { key: "Enter" });
    fireEvent.keyDown(phaseSummary, { key: " " });
    expect(step).not.toHaveAttribute("open");
    expect(phase).not.toHaveAttribute("open");
  });

  it("collapses successful phases and keeps bounded outcomes visible after completion", () => {
    const complete: ExecutionPlan = {
      ...plan,
      status: "degraded",
      terminal_steps: 3,
      progress_ratio: 1,
      phases: plan.phases.map((phase) => ({
        ...phase,
        status: phase.phase_id === "context" ? "degraded" : "passed"
      })),
      steps: plan.steps.map((step) => ({
        ...step,
        status: step.step_id === "context-01" ? "degraded" : "passed"
      }))
    };
    render(
      <LiveExecutionPlan
        plan={complete}
        running={false}
        onProof={vi.fn()}
        onCalculations={vi.fn()}
        onLineage={vi.fn()}
      />
    );

    expect(screen.getByLabelText("Interpretation phase")).not.toHaveAttribute("open");
    expect(screen.getByLabelText("Context phase")).toHaveAttribute("open");
    expect(screen.getByLabelText("Accounting Reporting step")).toHaveAttribute("open");
  });

  it("renders routes, fallback, repair, failure, and withheld states as safe operational facts", () => {
    const exceptional: ExecutionPlan = {
      ...plan,
      status: "failed",
      total_steps: 5,
      terminal_steps: 4,
      progress_ratio: 0.8,
      phases: plan.phases.map((phase) => {
        const status = phase.phase_id === "context" ? "failed"
          : phase.phase_id === "review" ? "repairing"
            : phase.phase_id === "synthesis" ? "withheld"
              : phase.phase_id === "release" ? "failed"
                : phase.status;
        return { ...phase, status };
      }),
      steps: [
        plan.steps[0],
        {
          ...plan.steps[1],
          safe_label: "Radeon specialist",
          status: "degraded",
          route: "radeon_api_fallback",
          degradation_code: "primary_route_unavailable",
          safe_summary: "The authorized fallback returned a bounded subset."
        },
        {
          ...plan.steps[1],
          step_id: "context-02",
          safe_label: "Local specialist",
          status: "failed",
          route: "local_rocm",
          failure_code: "model_unavailable",
          safe_summary: "The local specialist stopped safely."
        },
        {
          ...plan.steps[1],
          step_id: "review-01",
          parent_phase_id: "review",
          phase: "review",
          kind: "review",
          safe_label: "Evidence review",
          role_id: "evidence-critic/v1",
          status: "repairing",
          route: "local_rocm",
          safe_summary: "The reviewer requested one bounded repair."
        },
        {
          ...plan.steps[2],
          status: "withheld",
          failure_code: "evidence_rejected",
          safe_summary: "The final answer was withheld."
        }
      ]
    };
    render(
      <LiveExecutionPlan
        plan={exceptional}
        running={false}
        onProof={vi.fn()}
        onCalculations={vi.fn()}
        onLineage={vi.fn()}
      />
    );

    expect(screen.getAllByText("Radeon Api Fallback").length).toBeGreaterThan(0);
    expect(screen.getByText("Primary Route Unavailable")).toBeInTheDocument();
    expect(screen.getAllByText("Local Rocm").length).toBeGreaterThan(0);
    expect(screen.getByText("Model Unavailable")).toBeInTheDocument();
    expect(screen.getAllByText("Repairing").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Withheld").length).toBeGreaterThan(0);
    expect(screen.getByText("Evidence Rejected")).toBeInTheDocument();
  });
});
