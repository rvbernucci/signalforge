import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import fixtureData from "../../../fixtures/workspace/golden-case.json";
import type { Projection, SafeEvent } from "../types";
import { AuditWorkspace } from "./AuditWorkspace";

const projection = fixtureData as unknown as Projection;

describe("AuditWorkspace", () => {
  it("uses the accepted projection and excludes unsafe event attributes", () => {
    const onProof = vi.fn();
    const unsafeEvents: SafeEvent[] = [{
      sequence: 1,
      run_id: projection.run_id,
      step_id: "context-accounting",
      type: "context",
      status: "completed",
      at: projection.as_of,
      attributes: {
        raw_prompt: "private-prompt-body",
        raw_output: "private-model-output",
        api_key: "private-credential"
      }
    }];

    render(
      <AuditWorkspace
        projection={projection}
        plan={null}
        events={unsafeEvents}
        running={false}
        connection="live"
        open
        judgeMode={false}
        intelligenceAvailable
        onClose={vi.fn()}
        onJudgeMode={vi.fn()}
        onProof={onProof}
        onCalculations={vi.fn()}
        onMissionControl={vi.fn()}
      />
    );

    expect(screen.getByRole("region", { name: "Exact accepted-run identity" })).toHaveTextContent(projection.run_id);
    expect(screen.getByRole("region", { name: "Exact accepted-run identity" })).toHaveTextContent(projection.execution.model);
    expect(screen.getByText("Recorded governed journey")).toBeInTheDocument();
    expect(screen.queryByText("private-prompt-body")).not.toBeInTheDocument();
    expect(screen.queryByText("private-model-output")).not.toBeInTheDocument();
    expect(screen.queryByText("private-credential")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Inspect source evidence/i }));
    expect(onProof).toHaveBeenCalledWith([]);
  });

  it("keeps disclosure open across signed event updates and exposes judge orientation explicitly", () => {
    const props = {
      projection,
      plan: null,
      running: true,
      connection: "live" as const,
      open: true,
      judgeMode: true,
      intelligenceAvailable: true,
      onClose: vi.fn(),
      onJudgeMode: vi.fn(),
      onProof: vi.fn(),
      onCalculations: vi.fn(),
      onMissionControl: vi.fn()
    };
    const { rerender } = render(<AuditWorkspace {...props} events={projection.events.slice(0, 2)} />);

    expect(screen.getByRole("region", { name: "Track 2 judge orientation" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Close audit workspace" })).toHaveFocus();
    rerender(<AuditWorkspace {...props} events={projection.events.slice(0, 12)} />);
    expect(screen.getByRole("dialog", { name: "How SignalForge reached this answer" })).toBeVisible();
    expect(screen.getByText("12 lifecycle events")).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Track 2 judge orientation" })).toBeVisible();
  });

  it("makes Mission Control conditional without removing evidence or calculations", () => {
    render(
      <AuditWorkspace
        projection={projection}
        plan={null}
        events={projection.events}
        running={false}
        connection="unavailable"
        open
        judgeMode={false}
        intelligenceAvailable={false}
        onClose={vi.fn()}
        onJudgeMode={vi.fn()}
        onProof={vi.fn()}
        onCalculations={vi.fn()}
        onMissionControl={vi.fn()}
      />
    );

    expect(screen.queryByRole("button", { name: /Open Mission Control/i })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Inspect source evidence/i })).toBeVisible();
    expect(screen.getByRole("button", { name: /Inspect calculations/i })).toBeVisible();
    expect(screen.getByRole("region", { name: "Exact accepted-run identity" })).toHaveTextContent(projection.run_id);
  });
});
