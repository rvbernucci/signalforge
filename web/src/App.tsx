import { startTransition, useEffect, useState } from "react";
import { getCatalog, getConfig, getFinancials, getGoldenCase, getPeerEvaluations, getReadiness } from "./api";
import { CaseNotes, InsightPanel } from "./components/InsightPanel";
import { MobileHeader, Navigation } from "./components/Navigation";
import { ProofDrawer } from "./components/ProofDrawer";
import { IntelligenceDrawer } from "./components/IntelligenceDrawer";
import { AuditWorkspace } from "./components/AuditWorkspace";
import { CompactRunStatus } from "./components/CompactRunStatus";
import { ScenarioBar } from "./components/ScenarioBar";
import { ResearchScope } from "./components/ResearchScope";
import { ArrowIcon, ChipIcon, ShieldIcon, SparkIcon } from "./components/Icons";
import { MemoryControls } from "./components/CaseLibrary";
import { useResearchRun } from "./hooks/useResearchRun";
import { displayCaseTitle } from "./format";
import type { FinancialSummary, PeerEvaluationSuite, ProductCatalog, Projection, ScenarioControl, WorkspaceConfig, WorkspaceReadiness } from "./types";

const fallbackScenario: ScenarioControl = { rates: "higher_for_longer", ai_spending: "slower" };

export function App() {
  const [fixture, setFixture] = useState<Projection | null>(null);
  const [config, setConfig] = useState<WorkspaceConfig | null>(null);
  const [readiness, setReadiness] = useState<WorkspaceReadiness | null>(null);
  const [catalog, setCatalog] = useState<ProductCatalog | null>(null);
  const [financials, setFinancials] = useState<FinancialSummary | null>(null);
  const [peerEvaluations, setPeerEvaluations] = useState<PeerEvaluationSuite | null>(null);
  const [bootError, setBootError] = useState(false);
  const [question, setQuestion] = useState("");
  const [scenario, setScenario] = useState<ScenarioControl>(fallbackScenario);
  const [activeSection, setActiveSection] = useState("");
  const [navOpen, setNavOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerTab, setDrawerTab] = useState<"evidence" | "calculations">("evidence");
  const [drawerRefs, setDrawerRefs] = useState<string[]>([]);
  const [followUp, setFollowUp] = useState("");
  const [retain, setRetain] = useState(false);
  const [libraryOpen, setLibraryOpen] = useState(false);
  const [intelligenceOpen, setIntelligenceOpen] = useState(false);
  const initialAuditRoute = readAuditRoute();
  const [auditOpen, setAuditOpen] = useState(initialAuditRoute.open);
  const [judgeMode, setJudgeMode] = useState(initialAuditRoute.judge);
  const research = useResearchRun(fixture);

  useEffect(() => {
    let active = true;
    Promise.all([getGoldenCase(), getConfig(), getReadiness(), getCatalog(), getFinancials(), getPeerEvaluations()]).then(([nextFixture, nextConfig, nextReadiness, nextCatalog, nextFinancials, nextPeers]) => {
      if (!active) return;
      startTransition(() => {
        setFixture(nextFixture);
        setConfig(nextConfig);
        setReadiness(nextReadiness);
        setCatalog(nextCatalog);
        setFinancials(nextFinancials);
        setPeerEvaluations(nextPeers);
        setQuestion(nextFixture.question);
        setScenario(nextConfig.scenario_defaults);
        setActiveSection(nextFixture.sections[0]?.section_type ?? "");
      });
    }).catch(() => setBootError(true));
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (research.projection && !research.projection.sections.some((section) => section.section_type === activeSection)) {
      setActiveSection(research.projection.sections[0]?.section_type ?? "");
    }
  }, [research.projection, activeSection]);

  useEffect(() => {
    function closeOverlay(event: KeyboardEvent) {
      if (event.key !== "Escape") return;
      if (drawerOpen) setDrawerOpen(false);
      else if (intelligenceOpen) setIntelligenceOpen(false);
      else if (libraryOpen) setLibraryOpen(false);
      else if (auditOpen) closeAudit();
      else if (navOpen) setNavOpen(false);
    }
    window.addEventListener("keydown", closeOverlay);
    return () => window.removeEventListener("keydown", closeOverlay);
  }, [auditOpen, drawerOpen, intelligenceOpen, libraryOpen, navOpen]);

  useEffect(() => {
    function restoreAuditRoute() {
      const route = readAuditRoute();
      setAuditOpen(route.open);
      setJudgeMode(route.judge);
    }
    window.addEventListener("popstate", restoreAuditRoute);
    return () => window.removeEventListener("popstate", restoreAuditRoute);
  }, []);

  if (bootError) return <BootFailure />;
  const projection = research.projection ?? fixture;
  if (!projection || !config || !readiness || !catalog || !financials || !peerEvaluations) return <BootScreen />;
  const executionPlan = research.executionPlan ?? projection.execution_plan ?? null;
  const progressEvents = research.events.length > 0 ? research.events : projection.events;
  const traceID = research.run?.run_id === projection.run_id ? research.run.trace_id : undefined;

  function openProof(tab: "evidence" | "calculations", refs: string[] = []) {
    setDrawerTab(tab);
    setDrawerRefs(refs ?? []);
    setDrawerOpen(true);
  }

  function submitFollowUp(questionText: string) {
    const clean = questionText.trim();
    if (!clean || !config?.follow_ups_live) return;
    setFollowUp("");
    void research.askFollowUp(clean, retain);
  }

  function openAudit(judge = false) {
    setAuditOpen(true);
    setJudgeMode(judge);
    writeAuditRoute(true, judge);
  }

  function closeAudit() {
    setAuditOpen(false);
    setJudgeMode(false);
    writeAuditRoute(false, false);
  }

  function changeJudgeMode(enabled: boolean) {
    setJudgeMode(enabled);
    writeAuditRoute(true, enabled);
  }

  return (
    <div className="app-shell">
      <MobileHeader onOpen={() => setNavOpen(true)} />
      <Navigation projection={projection} activeSection={activeSection} open={navOpen} onOpen={() => setNavOpen(true)} onClose={() => setNavOpen(false)} onSection={setActiveSection} />
      <main className="research-main">
        <header className="case-topbar">
          <div className="case-title">
            <span className="status-pulse" />
            <div><span>Research case · {projection.intent.replaceAll("_", " ")}</span><strong>{displayCaseTitle(projection.title)}</strong></div>
          </div>
          <div className="trust-badge"><ShieldIcon /><span><strong>Governed research</strong>Evidence and limitations stay visible</span></div>
        </header>

        <div className="research-canvas">
          <ResearchScope
            catalog={catalog}
            financials={financials}
            peers={peerEvaluations}
            scenario={scenario}
            live={config.mode === "live"}
            onQuestion={setQuestion}
          />
          <ScenarioBar question={question} scenario={scenario} running={research.running} onQuestion={setQuestion} onScenario={setScenario} onRun={() => void research.start(question, scenario, retain)} />
          <MemoryControls
            available={config.retention_available}
            retain={retain}
            retention={research.run?.retention}
            open={libraryOpen}
            onRetain={setRetain}
            onOpen={() => setLibraryOpen(true)}
            onClose={() => setLibraryOpen(false)}
            onLoad={(stored) => {
              research.loadProjection(stored);
              setQuestion(stored.question);
              setActiveSection(stored.sections[0]?.section_type ?? "");
            }}
          />
          {research.error && <div className="degraded-banner" role="alert"><ShieldIcon /><span><strong>Fail-safe state</strong>{research.error}</span></div>}
          <CompactRunStatus
            plan={executionPlan}
            events={progressEvents}
            running={research.running}
            connection={research.executionConnection}
            onAudit={() => openAudit(false)}
          />
          <InsightPanel projection={projection} activeSection={activeSection} onSection={setActiveSection} onProof={openProof} />
          <CaseNotes projection={projection} />
          <FollowUpPanel projection={projection} enabled={config.follow_ups_live} value={followUp} running={research.running} onValue={setFollowUp} onSubmit={submitFollowUp} />
        </div>
        <footer className="site-footer"><span>SignalForge · Private investor intelligence</span><span><ShieldIcon /> AI research can be inaccurate. Verify evidence and deterministic receipts before acting.</span></footer>
      </main>
      <AuditWorkspace
        projection={projection}
        readiness={readiness}
        plan={executionPlan}
        events={progressEvents}
        traceID={traceID}
        running={research.running}
        connection={research.executionConnection}
        open={auditOpen}
        judgeMode={judgeMode}
        intelligenceAvailable={config.intelligence_audit}
        onClose={closeAudit}
        onJudgeMode={changeJudgeMode}
        onProof={(refs) => openProof("evidence", refs)}
        onCalculations={(refs) => openProof("calculations", refs)}
        onMissionControl={() => setIntelligenceOpen(true)}
      />
      <ProofDrawer projection={projection} open={drawerOpen} tab={drawerTab} refs={drawerRefs} onTab={setDrawerTab} onClose={() => setDrawerOpen(false)} />
      <IntelligenceDrawer
        runID={projection.run_id}
        traceID={traceID}
        open={intelligenceOpen}
        protectedCapture={config.protected_capture}
        onClose={() => setIntelligenceOpen(false)}
      />
    </div>
  );
}

function FollowUpPanel({ projection, enabled, value, running, onValue, onSubmit }: { projection: Projection; enabled: boolean; value: string; running: boolean; onValue: (value: string) => void; onSubmit: (value: string) => void }) {
  return (
    <section className="follow-up-panel" aria-labelledby="follow-up-title">
      <div className="follow-up-heading"><SparkIcon /><div><span className="eyebrow">Case-aware follow-up</span><h2 id="follow-up-title">Push the thesis further.</h2></div></div>
      <div className="suggestion-row">
        {projection.follow_up_suggestions.map((suggestion) => <button key={suggestion.suggestion_id} onClick={() => enabled ? onSubmit(suggestion.question) : onValue(suggestion.question)} disabled={running}>{suggestion.label}<ArrowIcon /></button>)}
      </div>
      <form onSubmit={(event) => { event.preventDefault(); onSubmit(value); }}>
        <input value={value} onChange={(event) => onValue(event.target.value)} placeholder="Ask about evidence, assumptions, or thesis risks" maxLength={1200} aria-label="Follow-up question" />
        <button disabled={!enabled || running || value.trim().length === 0} aria-label="Submit follow-up"><ArrowIcon /></button>
      </form>
      {!enabled && <p className="mode-note"><ChipIcon /> Follow-up inference activates in live Radeon mode. Suggestions remain available as a demo of the intended flow.</p>}
    </section>
  );
}

function BootScreen() {
  return <div className="boot-screen"><span className="forge-loader"><i /><i /><i /></span><strong>Preparing the research workspace</strong><small>Loading the privacy-safe case projection</small></div>;
}

function BootFailure() {
  return <div className="boot-screen failure" role="alert"><ShieldIcon /><strong>The workspace stopped safely.</strong><small>Start the local SignalForge server, then reload this page.</small></div>;
}

function readAuditRoute() {
  if (typeof window === "undefined") return { open: false, judge: false };
  const params = new URLSearchParams(window.location.search);
  return {
    open: params.get("view") === "audit",
    judge: params.get("view") === "audit" && params.get("audience") === "judge"
  };
}

function writeAuditRoute(open: boolean, judge: boolean) {
  const url = new URL(window.location.href);
  if (open) {
    url.searchParams.set("view", "audit");
    if (judge) url.searchParams.set("audience", "judge");
    else url.searchParams.delete("audience");
  } else {
    url.searchParams.delete("view");
    url.searchParams.delete("audience");
  }
  window.history.replaceState(window.history.state, "", `${url.pathname}${url.search}${url.hash}`);
}
