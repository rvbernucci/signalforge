import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import fixtureData from "../../fixtures/workspace/golden-case.json";
import { App } from "./App";
import type { Projection, WorkspaceConfig } from "./types";

const fixture = fixtureData as unknown as Projection;
const config: WorkspaceConfig = {
  mode: "fixture",
  local_only: true,
  endpoint_scope: "loopback_only",
  model: "signalforge-gemma4-26b-q4",
  scenario_defaults: { rates: "higher_for_longer", ai_spending: "slower" },
  follow_ups_live: false,
  retention_available: true,
  retention_default: false
};

describe("SignalForge workspace", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      const value = url.endsWith("/api/v1/config") ? config
        : url.endsWith("/api/v1/cases") ? { cases: [] }
        : fixture;
      return { ok: true, status: 200, json: async () => value } as Response;
    }));
  });

  it("renders the safe local case and its proof boundary", async () => {
    render(<App />);
    expect(await screen.findByText("Ask a harder question.")).toBeInTheDocument();
    expect(screen.getAllByText("Local inference").length).toBeGreaterThan(0);
    expect(screen.getByText("Numerical authority preserved")).toBeInTheDocument();
    expect(screen.getByText("No remote model calls · No model-authored financial values")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Open proof layer"));
    expect(await screen.findByText("Inspect the work.")).toBeInTheDocument();
    expect(screen.getAllByText(/Evidence/).length).toBeGreaterThan(0);
  });

  it("moves between analysis chapters without regenerating content", async () => {
    render(<App />);
    await screen.findByText("Ask a harder question.");
    fireEvent.click(screen.getByRole("button", { name: /Transmission Mechanisms/i }));
    await waitFor(() => expect(screen.getByRole("heading", { name: "Transmission Mechanisms" })).toBeInTheDocument());
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it("states the fixture follow-up limitation instead of pretending it is live", async () => {
    render(<App />);
    expect(await screen.findByText(/Follow-up inference activates in live Radeon mode/)).toBeInTheDocument();
    expect(screen.getByLabelText("Submit follow-up")).toBeDisabled();
  });

  it("keeps retention opt-in and exposes the empty local case library", async () => {
    render(<App />);
    expect(await screen.findByText("Save this case locally")).toBeInTheDocument();
    expect(screen.getByRole("checkbox")).not.toBeChecked();
    expect(screen.getByText("Ephemeral session · nothing retained")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Open saved cases" }));
    expect(await screen.findByText("Research case library")).toBeInTheDocument();
    expect(await screen.findByText("No cases have been saved on this device.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close case library" })).toHaveFocus();
  });

  it("moves focus into the proof dialog and returns it on Escape", async () => {
    render(<App />);
    const trigger = await screen.findByRole("button", { name: /Open proof layer/i });
    trigger.focus();
    fireEvent.click(trigger);
    expect(screen.getByRole("button", { name: "Close proof drawer" })).toHaveFocus();
    fireEvent.keyDown(window, { key: "Escape" });
    await waitFor(() => expect(trigger).toHaveFocus());
  });

  it("shows an honest empty state when proof filtering has no matches", async () => {
    render(<App />);
    fireEvent.click(await screen.findByRole("button", { name: /Open proof layer/i }));
    fireEvent.change(screen.getByPlaceholderText("Filter this proof set"), { target: { value: "not-a-real-proof-id" } });
    expect(await screen.findByText("No proof items match this filter.")).toBeInTheDocument();
  });

  it("renders a plain-language fail-closed boot error", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => { throw new Error("offline"); }));
    render(<App />);
    expect(await screen.findByRole("alert")).toHaveTextContent("The workspace stopped safely.");
    expect(screen.getByText("Start the local SignalForge server, then reload this page.")).toBeInTheDocument();
  });
});
