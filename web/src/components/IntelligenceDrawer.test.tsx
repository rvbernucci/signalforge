import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { IntelligenceRecord } from "../types";
import { IntelligenceDrawer } from "./IntelligenceDrawer";

const record: IntelligenceRecord = {
  schema_version: "signalforge/intelligence-lineage/v1",
  run_id: "run-accepted",
  request_id: "request-accepted",
  trace_id: "80c42f9bdbda413b9f86413db94ed20a",
  status: "completed",
  capture: {
    enabled: false,
    available: false,
    status: "disabled",
    stored_bytes: 0,
    maximum_bytes: 0
  },
  started_at: "2026-07-25T00:00:00Z",
  completed_at: "2026-07-25T00:00:01Z",
  model_calls: [],
  retrievals: [],
  engine_calls: [],
  reviews: []
};

describe("IntelligenceDrawer identity boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("renders lineage only when Workspace and Mission Control identities match", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => record
    } as Response)));
    render(
      <IntelligenceDrawer
        runID={record.run_id}
        traceID={record.trace_id}
        open
        protectedCapture={false}
        onClose={vi.fn()}
      />
    );
    expect(await screen.findByText(/80c42f9bdbda/)).toBeInTheDocument();
    expect(screen.getByText("run-accepted")).toBeInTheDocument();
  });

  it("fails closed when Mission Control returns a different trace", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({ ...record, trace_id: "ffffffffffffffffffffffffffffffff" })
    } as Response)));
    render(
      <IntelligenceDrawer
        runID={record.run_id}
        traceID={record.trace_id}
        open
        protectedCapture={false}
        onClose={vi.fn()}
      />
    );
    expect(await screen.findByText("Lineage unavailable")).toBeInTheDocument();
    expect(screen.queryByText("ffffffffffff")).not.toBeInTheDocument();
  });
});
