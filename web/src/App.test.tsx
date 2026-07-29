import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import fixtureData from "../../fixtures/workspace/golden-case.json";
import financialData from "../../fixtures/productscope/technology20-financial-summary.json";
import peerData from "../../fixtures/productscope/technology20-peer-evaluation.json";
import { App } from "./App";
import { displayCaseTitle } from "./format";
import type { FinancialSummary, IntelligenceRecord, PeerEvaluationSuite, ProductCatalog, Projection, WorkspaceConfig } from "./types";

const fixture = fixtureData as unknown as Projection;
const financialFixture = financialData as unknown as FinancialSummary;
const financials: FinancialSummary = {
  ...financialFixture,
  companies: financialFixture.companies.map((company, index) => index === 0 ? {
    ...company,
    company_id: "company-0",
    primary_ticker: "MSFT",
    display_name: "Microsoft"
  } : company)
};
const contextualSample = financialFixture.companies.find(
  (company) => (company.contextual_results?.length ?? 0) > 0
)!.contextual_results![0];
const financialsWithStandaloneContext: FinancialSummary = {
  ...financials,
  companies: financials.companies.map((company, index) => index === 0 ? {
    ...company,
    contextual_results: [contextualSample]
  } : company)
};
const peerFixture = peerData as unknown as PeerEvaluationSuite;
const peers: PeerEvaluationSuite = {
  ...peerFixture,
  lanes: peerFixture.lanes.map((lane, index) => index === 0 ? {
    ...lane,
    lane_id: "lane-0",
    company_ids: ["company-0", "company-1"]
  } : lane)
};
const peersWithContext: PeerEvaluationSuite = {
  ...peers,
  lanes: peers.lanes.map((lane, index) => index === 0 ? {
    ...lane,
    context_only_metric_ids: ["financial.free_cash_flow"],
    receipts: [
      {
        disposition: "context_only",
        operands: [
          {
            company_id: "company-0",
            security_id: "ticker:MSFT",
            canonical_metric_id: "financial.free_cash_flow",
            taxonomy_concept: "PaymentsToAcquirePropertyPlantAndEquipment",
            value: "74100000000",
            unit: "currency",
            currency: "USD",
            accounting_perimeter: "consolidated_periodic_filing",
            output_class: "authoritative",
            product_label: "simple FCF",
            pair_ranking_eligible: true
          },
          {
            company_id: "company-1",
            security_id: "ticker:T1",
            canonical_metric_id: "financial.free_cash_flow",
            taxonomy_concept: "PaymentsToAcquireProductiveAssets",
            value: "96676000000",
            unit: "currency",
            currency: "USD",
            accounting_perimeter: "company_reported_property_equipment_and_intangible_assets",
            output_class: "context_only",
            product_label: "residual cash proxy",
            pair_ranking_eligible: false
          }
        ],
        reason_codes: ["reviewed_taxonomy_mapping", "same_accounting_perimeter"],
        receipt_sha256: "c".repeat(64)
      }
    ]
  } : lane)
};
const config: WorkspaceConfig = {
  mode: "fixture",
  local_only: true,
  endpoint_scope: "loopback_only",
  model: "signalforge-gemma4-26b-q4",
  scenario_defaults: { rates: "higher_for_longer", ai_spending: "slower" },
  follow_ups_live: false,
  retention_available: true,
  retention_default: false,
  intelligence_audit: true,
  protected_capture: false
};

const catalog: ProductCatalog = {
  schema_version: "signalforge/technology20-public-catalog/v1",
  universe_id: "us-technology-20-v2",
  as_of: fixture.as_of,
  company_policy_version: "technology20-company-policy/v1",
  activation_policy_version: "technology20-activation/v1",
  peer_lane_policy_version: "technology20-peer-lanes/v1",
  source_registry_sha256: "a".repeat(64),
  claim_boundary: "Catalog presence does not authorize research or comparison.",
  companies: Array.from({ length: 20 }, (_, index) => ({
    company_id: `company-${index}`,
    display_name: index === 0 ? "Microsoft" : `Company ${index}`,
    primary_ticker: index === 0 ? "MSFT" : `T${index}`,
    tickers: [index === 0 ? "MSFT" : `T${index}`],
    research_cluster: "platforms_and_cloud",
    peer_group: "hyperscale_platforms",
    research_role: "bounded research",
    activation_state: "data_ready",
    research_enabled: false,
    reason_codes: ["standalone_journey_not_yet_promoted"],
    metric_state_count: { covered: 20 },
    profile_sha256: `${index}`.padEnd(64, "a")
  })),
  peer_lanes: Array.from({ length: 5 }, (_, index) => ({
    lane_id: `lane-${index}`,
    company_ids: [`company-${index}`, `company-${index + 1}`],
    comparison_type: "candidate",
    decision_question: "A bounded comparison.",
    allowed_question_ids: ["business_quality"],
    allowed_metric_ids: ["financial.revenue_growth"],
    enabled: false,
    reason_codes: ["peer_lane_not_yet_promoted"],
    lane_sha256: `${index}`.padEnd(64, "b")
  }))
};

const intelligence: IntelligenceRecord = {
  schema_version: "signalforge/intelligence-lineage/v1",
  run_id: fixture.run_id,
  request_id: fixture.request_id,
  trace_id: "80c42f9bdbda413b9f86413db94ed20a",
  status: "completed",
  capture: { enabled: false, available: false, status: "disabled", stored_bytes: 0, maximum_bytes: 16777216 },
  started_at: fixture.events[0]?.at ?? fixture.as_of,
  completed_at: fixture.events.at(-1)?.at ?? fixture.as_of,
  timeline: [],
  model_calls: [],
  retrievals: [{
    retrieval_id: "retrieval-accounting",
    step_id: "context-accounting",
    role_id: "accounting-context/v1",
    method: "public_fixture",
    context_packet_id: "packet-accounting",
    evidence_ids: ["evidence-msft-revenue"],
    estimated_tokens: 320,
    status: "completed",
    completed_at: fixture.events.at(-1)?.at ?? fixture.as_of
  }],
  engine_calls: [{
    engine_call_id: "engine-dcf",
    step_id: "engine-valuation",
    requested_by: "valuation-context/v1",
    engine_id: "valuation",
    engine_version: "1.0.0",
    operation_id: "valuation.dcf",
    formula_version: "v1",
    receipt_id: "receipt-dcf",
    receipt_sha256: "d".repeat(64),
    input_refs: ["variable-fcf"],
    output_refs: ["variable-enterprise-value"],
    invariants_total: 2,
    invariants_passed: 2,
    status: "success",
    generated_at: fixture.events.at(-1)?.at ?? fixture.as_of
  }],
  reviews: [],
  release: {
    answer_id: "answer-golden",
    primary_intent: fixture.intent,
    section_types: fixture.sections.map((section) => section.section_type),
    claim_refs: [],
    evidence_refs: ["evidence-msft-revenue"],
    receipt_refs: ["receipt-dcf"],
    status: "released"
  }
};

describe("SignalForge workspace", () => {
  beforeEach(() => {
    window.history.replaceState({}, "", "/");
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      const value = url.endsWith("/api/v1/config") ? config
        : url.endsWith("/api/v1/catalog") ? catalog
        : url.endsWith("/api/v1/financials") ? financials
        : url.endsWith("/api/v1/peer-evaluations") ? peers
        : url.endsWith("/intelligence") ? intelligence
        : url.endsWith("/api/v1/cases") ? { cases: [] }
        : fixture;
      return { ok: true, status: 200, json: async () => value } as Response;
    }));
  });

  it("renders the safe local case and its proof boundary", async () => {
    render(<App />);
    expect(await screen.findByText("Ask a harder question.")).toBeInTheDocument();
    expect(screen.getByText("Research answer ready")).toBeInTheDocument();
    expect(screen.getByText("Numerical authority preserved")).toBeInTheDocument();
    expect(screen.getByText(/AI research can be inaccurate/)).toBeInTheDocument();
    expect(screen.queryByText("Live research execution")).not.toBeInTheDocument();
    expect(screen.queryByText("Supported context claims")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "How SignalForge reached this answer" }));
    fireEvent.click(screen.getByRole("button", { name: /Inspect source evidence/i }));
    expect(await screen.findByText("Inspect the work.")).toBeInTheDocument();
    expect(screen.getAllByText(/Evidence/).length).toBeGreaterThan(0);
  });

  it("moves between analysis chapters without regenerating content", async () => {
    render(<App />);
    await screen.findByText("Ask a harder question.");
    fireEvent.click(screen.getByRole("button", { name: /Transmission Mechanisms/i }));
    await waitFor(() => expect(screen.getByRole("heading", { name: "Transmission Mechanisms" })).toBeInTheDocument());
    expect(fetch).toHaveBeenCalledTimes(5);
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
    fireEvent.click(await screen.findByRole("button", { name: "How SignalForge reached this answer" }));
    const trigger = screen.getByRole("button", { name: /Inspect source evidence/i });
    trigger.focus();
    fireEvent.click(trigger);
    expect(screen.getByRole("button", { name: "Close proof drawer" })).toHaveFocus();
    fireEvent.keyDown(window, { key: "Escape" });
    await waitFor(() => expect(trigger).toHaveFocus());
    expect(screen.getByRole("dialog", { name: "How SignalForge reached this answer" })).toBeVisible();
  });

  it("shows an honest empty state when proof filtering has no matches", async () => {
    render(<App />);
    fireEvent.click(await screen.findByRole("button", { name: "How SignalForge reached this answer" }));
    fireEvent.click(screen.getByRole("button", { name: /Inspect source evidence/i }));
    fireEvent.change(screen.getByPlaceholderText("Filter this proof set"), { target: { value: "not-a-real-proof-id" } });
    expect(await screen.findByText("No proof items match this filter.")).toBeInTheDocument();
  });

  it("renders a plain-language fail-closed boot error", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => { throw new Error("offline"); }));
    render(<App />);
    expect(await screen.findByRole("alert")).toHaveTextContent("The workspace stopped safely.");
    expect(screen.getByText("Start the local SignalForge server, then reload this page.")).toBeInTheDocument();
  });

  it("opens the privacy-safe intelligence lineage without exposing model bodies", async () => {
    render(<App />);
    await screen.findByText("Research answer ready");
    fireEvent.click(screen.getByRole("button", { name: "How SignalForge reached this answer" }));
    fireEvent.click(screen.getByRole("button", { name: /Open Mission Control/i }));
    expect(await screen.findByText("Intelligence lineage.")).toBeInTheDocument();
    expect(screen.getByText(/0 answer-used claims/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "evidence" }));
    expect(await screen.findByText("Accounting Context V1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "engines" }));
    expect(await screen.findByText("Valuation Dcf")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "privacy" }));
    expect(await screen.findByText("Operational proof without model bodies")).toBeInTheDocument();
    expect(screen.getByText("Stored body bytes shown").parentElement).toHaveTextContent("0");
    expect(screen.queryByLabelText("Protected audit token")).not.toBeInTheDocument();
  });

  it("explains why a selected peer lane remains guarded", async () => {
    render(<App />);
    await screen.findByText("Ask a harder question.");
    fireEvent.click(screen.getByRole("button", { name: "Compare companies" }));
    fireEvent.click(screen.getByRole("listitem", { name: "Inspect bounded evidence for Microsoft" }));
    fireEvent.click(screen.getByRole("listitem", { name: "Inspect bounded evidence for Company 1" }));
    expect(await screen.findByText("Comparison remains guarded")).toBeInTheDocument();
    expect(screen.getByText("Peer lane not yet promoted")).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Metric-level comparison authority" })).toBeInTheDocument();
    expect(screen.getAllByText("Comparable with caveat").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Different fiscal periods/).length).toBeGreaterThan(0);
    expect(screen.getAllByText("Withheld").length).toBeGreaterThan(0);
  });

  it("shows deterministic context without ranking accounting perimeters that differ", async () => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      const value = url.endsWith("/api/v1/config") ? config
        : url.endsWith("/api/v1/catalog") ? catalog
        : url.endsWith("/api/v1/financials") ? financials
        : url.endsWith("/api/v1/peer-evaluations") ? peersWithContext
        : url.endsWith("/intelligence") ? intelligence
        : fixture;
      return { ok: true, status: 200, json: async () => value } as Response;
    }));

    render(<App />);
    await screen.findByText("Ask a harder question.");
    fireEvent.click(screen.getByRole("button", { name: "Compare companies" }));
    fireEvent.click(screen.getByRole("listitem", { name: "Inspect bounded evidence for Microsoft" }));
    fireEvent.click(screen.getByRole("listitem", { name: "Inspect bounded evidence for Company 1" }));

    expect(await screen.findByText("residual cash proxy")).toBeInTheDocument();
    expect(screen.getByText("Context only*")).toBeInTheDocument();
    expect(screen.getByText("$74.1B")).toBeInTheDocument();
    expect(screen.getByText("$96.7B")).toBeInTheDocument();
    expect(screen.getByText(/not directly comparable metrics/i)).toBeInTheDocument();
    expect(screen.getByText(/no ranking or relative conclusion is released/i)).toBeInTheDocument();
    expect(screen.getAllByText("Withheld")).toHaveLength(1);
  });

  it("keeps standalone contextual receipts visibly separate from authoritative results", async () => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      const value = url.endsWith("/api/v1/config") ? config
        : url.endsWith("/api/v1/catalog") ? catalog
        : url.endsWith("/api/v1/financials") ? financialsWithStandaloneContext
        : url.endsWith("/api/v1/peer-evaluations") ? peers
        : url.endsWith("/intelligence") ? intelligence
        : fixture;
      return { ok: true, status: 200, json: async () => value } as Response;
    }));

    render(<App />);
    await screen.findByText("Ask a harder question.");
    fireEvent.click(screen.getByRole("listitem", { name: "Inspect bounded evidence for Microsoft" }));

    expect(await screen.findByRole("region", { name: "Context-only financial evidence" })).toBeInTheDocument();
    expect(screen.getByText("Context only · never ranked")).toBeInTheDocument();
    expect(screen.getByText(/cannot produce a winner, score, rank, or relative conclusion/i)).toBeInTheDocument();
  });

  it("shows governed company scope, recency, receipts, and missing evidence", async () => {
    render(<App />);
    await screen.findByText("Ask a harder question.");
    fireEvent.click(screen.getByRole("listitem", { name: "Inspect bounded evidence for Microsoft" }));
    expect(await screen.findByText("Evidence inspection only")).toBeInTheDocument();
    expect(screen.getByText("Evidence recency")).toBeInTheDocument();
    expect(screen.getByText("Latest governed period")).toBeInTheDocument();
    expect(screen.getByText("Price observation date")).toBeInTheDocument();
    expect(screen.getByText("Unavailable · not activated")).toBeInTheDocument();
    expect(screen.getByText("Scenario assumptions")).toBeInTheDocument();
    expect(screen.getByText("Higher for longer · AI spending slower")).toBeInTheDocument();
    expect(screen.getByText(/receipts · .* abstentions/)).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Explicit missing evidence" })).toBeInTheDocument();
  });

  it("keeps an unusually long company name usable in the governed selector", async () => {
    const longName = "International Semiconductor Infrastructure and Advanced Computing Systems Corporation";
    const longCatalog: ProductCatalog = {
      ...catalog,
      companies: catalog.companies.map((company, index) =>
        index === 0 ? { ...company, display_name: longName } : company
      )
    };
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      const value = url.endsWith("/api/v1/config") ? config
        : url.endsWith("/api/v1/catalog") ? longCatalog
        : url.endsWith("/api/v1/financials") ? financials
        : url.endsWith("/api/v1/peer-evaluations") ? peers
        : fixture;
      return { ok: true, status: 200, json: async () => value } as Response;
    }));

    render(<App />);
    fireEvent.click(await screen.findByText("Configure"));
    expect(screen.getByText(longName)).toBeInTheDocument();
    expect(screen.getByRole("listitem", { name: `Inspect bounded evidence for ${longName}` })).toBeEnabled();
  });

  it("keeps the previous case visible when the local model is unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "POST") {
        return {
          ok: false,
          status: 503,
          json: async () => ({ error: { code: "model_unavailable" } })
        } as Response;
      }
      const url = String(input);
      const value = url.endsWith("/api/v1/config") ? config
        : url.endsWith("/api/v1/catalog") ? catalog
        : url.endsWith("/api/v1/financials") ? financials
        : url.endsWith("/api/v1/peer-evaluations") ? peers
        : fixture;
      return { ok: true, status: 200, json: async () => value } as Response;
    }));
    render(<App />);
    fireEvent.click(await screen.findByRole("button", { name: /Forge analysis/i }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The local model is unavailable.");
    expect(screen.getByText(displayCaseTitle(fixture.title))).toBeInTheDocument();
  });

  it("disables orchestration inspection when observability is unavailable", async () => {
    const withoutObservability = { ...config, intelligence_audit: false, protected_capture: false };
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      const value = url.endsWith("/api/v1/config") ? withoutObservability
        : url.endsWith("/api/v1/catalog") ? catalog
        : url.endsWith("/api/v1/financials") ? financials
        : url.endsWith("/api/v1/peer-evaluations") ? peers
        : fixture;
      return { ok: true, status: 200, json: async () => value } as Response;
    }));
    render(<App />);
    fireEvent.click(await screen.findByRole("button", { name: "How SignalForge reached this answer" }));
    expect(screen.queryByRole("button", { name: /Open Mission Control/i })).not.toBeInTheDocument();
    expect(screen.getByText(displayCaseTitle(fixture.title))).toBeInTheDocument();
  });

  it("keeps operational telemetry out of the default research canvas", async () => {
    render(<App />);
    const answer = await screen.findByRole("heading", { name: "Comparison" });
    const canvas = answer.closest(".research-canvas")!;
    expect(canvas).not.toHaveTextContent(fixture.run_id);
    expect(canvas).not.toHaveTextContent("Trace");
    expect(canvas).not.toHaveTextContent("Model calls");
    expect(canvas).not.toHaveTextContent("Observed tokens");
    expect(canvas).not.toHaveTextContent(fixture.execution.runtime_label);
    expect(screen.getByRole("button", { name: "How SignalForge reached this answer" })).toBeVisible();
  });

  it("opens one audit workspace, updates the stable URL, and returns focus on Escape", async () => {
    render(<App />);
    const trigger = await screen.findByRole("button", { name: "How SignalForge reached this answer" });
    trigger.focus();
    fireEvent.click(trigger);

    expect(screen.getByRole("button", { name: "Close audit workspace" })).toHaveFocus();
    expect(screen.getByRole("dialog", { name: "How SignalForge reached this answer" })).toBeVisible();
    expect(window.location.search).toBe("?view=audit");
    expect(screen.getByRole("region", { name: "Exact accepted-run identity" })).toHaveTextContent(fixture.run_id);
    expect(screen.getByText("Recorded governed journey")).toBeInTheDocument();

    fireEvent.keyDown(window, { key: "Escape" });
    await waitFor(() => expect(trigger).toHaveFocus());
    expect(window.location.search).toBe("");
  });

  it("supports a credential-free judge deep link without changing the data path", async () => {
    window.history.replaceState({}, "", "/?view=audit&audience=judge");
    render(<App />);

    expect(await screen.findByRole("region", { name: "Track 2 judge orientation" })).toBeVisible();
    expect(screen.getByText("Track 2 capabilities in the product")).toBeInTheDocument();
    expect(screen.getByText("Tool invocation")).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Exact accepted-run identity" })).toHaveTextContent(fixture.run_id);
    expect(fetch).toHaveBeenCalledTimes(5);
    fireEvent.click(screen.getByRole("button", { name: /Inspect calculations/i }));
    expect(await screen.findByRole("dialog", { name: "Inspect the work." })).toBeVisible();
  });

  it("keeps judge orientation absent from the normal investor flow", async () => {
    render(<App />);
    await screen.findByText("Research answer ready");
    expect(screen.queryByRole("region", { name: "Track 2 judge orientation" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "How SignalForge reached this answer" }));
    expect(screen.queryByRole("region", { name: "Track 2 judge orientation" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Judge orientation" }));
    expect(screen.getByRole("region", { name: "Track 2 judge orientation" })).toBeVisible();
    expect(window.location.search).toBe("?view=audit&audience=judge");
  });
});
