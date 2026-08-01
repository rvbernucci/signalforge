import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { listCases } from "../api";
import type { CaseSummary } from "../types";
import { MemoryControls } from "./CaseLibrary";

vi.mock("../api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api")>();
  return {
    ...actual,
    listCases: vi.fn()
  };
});

const savedCase: CaseSummary = {
  case_id: "case-saved",
  run_id: "run-saved",
  title: "Microsoft / NVIDIA research case",
  as_of: "2026-07-21T16:00:00Z",
  intent: "company_comparison",
  saved_at: "2026-08-01T20:42:45Z",
  evidence_items: 12,
  calculation_receipts: 18,
  projection_sha256: "a".repeat(64)
};

describe("case library refresh", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("refreshes an open library when retention becomes durable", async () => {
    vi.mocked(listCases)
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([savedCase]);

    const view = render(
      <MemoryControls
        available
        retain
        retention={{ requested: true, status: "pending" }}
        open
        onRetain={vi.fn()}
        onOpen={vi.fn()}
        onClose={vi.fn()}
        onLoad={vi.fn()}
      />
    );

    expect(await screen.findByText("No cases have been saved on this device.")).toBeInTheDocument();

    view.rerender(
      <MemoryControls
        available
        retain
        retention={{ requested: true, status: "saved", case_id: savedCase.case_id }}
        open
        onRetain={vi.fn()}
        onOpen={vi.fn()}
        onClose={vi.fn()}
        onLoad={vi.fn()}
      />
    );

    expect(await screen.findByRole("heading", { name: "Microsoft / NVIDIA research case" })).toBeInTheDocument();
    await waitFor(() => expect(listCases).toHaveBeenCalledTimes(2));
  });

  it("does not report an empty library while its index is still loading", async () => {
    let resolveList: (items: CaseSummary[]) => void = () => undefined;
    vi.mocked(listCases).mockReturnValueOnce(new Promise((resolve) => {
      resolveList = resolve;
    }));

    render(
      <MemoryControls
        available
        retain={false}
        open
        onRetain={vi.fn()}
        onOpen={vi.fn()}
        onClose={vi.fn()}
        onLoad={vi.fn()}
      />
    );

    expect(await screen.findByRole("status")).toHaveTextContent("Loading saved cases...");
    expect(screen.queryByText("No cases have been saved on this device.")).not.toBeInTheDocument();

    resolveList([]);
    expect(await screen.findByText("No cases have been saved on this device.")).toBeInTheDocument();
  });
});
