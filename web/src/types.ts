export type ScenarioControl = {
  rates: "higher_for_longer" | "easing";
  ai_spending: "slower" | "resilient";
};

export type WorkspaceConfig = {
  mode: "fixture" | "live";
  local_only: boolean;
  endpoint_scope: string;
  model: string;
  scenario_defaults: ScenarioControl;
  follow_ups_live: boolean;
  retention_available: boolean;
  retention_default: boolean;
  intelligence_audit: boolean;
  protected_capture: boolean;
};

export type RetentionView = {
  requested: boolean;
  status: "not_requested" | "pending" | "saved" | "failed" | "unavailable";
  case_id?: string;
  error_code?: string;
};

export type CaseSummary = {
  case_id: string;
  run_id: string;
  parent_run_id?: string;
  title: string;
  as_of: string;
  intent: string;
  saved_at: string;
  evidence_items: number;
  calculation_receipts: number;
  projection_sha256: string;
};

export type StoredCase = { summary: CaseSummary; case: Projection };

export type Company = { entity_id: string; label: string };

export type Section = {
  section_type: string;
  title: string;
  content: string;
  claim_refs?: string[];
  evidence_refs?: string[];
  receipt_refs?: string[];
  numerical_refs?: string[];
};

export type EvidenceCard = {
  evidence_id: string;
  source_type: string;
  document_section?: string;
  locator: string;
  content_sha256: string;
  as_of: string;
  used_in_sections: string[];
};

export type ReceiptOutput = {
  output_id: string;
  quantity: { value: string; unit: string };
  status: string;
};

export type CalculationCard = {
  receipt_id: string;
  operation_id: string;
  engine_id: string;
  engine_version: string;
  formula_version: string;
  status: string;
  outputs: ReceiptOutput[];
  invariant_results: Array<{ invariant_id: string; passed: boolean }>;
  warnings?: string[];
  evidence_refs?: string[];
  source_as_of: string;
  receipt_sha256: string;
  used_in_sections: string[];
};

export type SafeEvent = {
  sequence: number;
  run_id?: string;
  step_id?: string;
  type: string;
  status: string;
  label?: string;
  at: string;
  attributes?: Record<string, string>;
};

export type Projection = {
  schema_version: string;
  case_id: string;
  run_id: string;
  request_id: string;
  status: string;
  title: string;
  question: string;
  as_of: string;
  intent: string;
  companies: Company[];
  sections: Section[];
  evidence: EvidenceCard[];
  calculations: CalculationCard[];
  assumptions?: string[];
  limitations?: string[];
  next_actions?: string[];
  warnings?: Array<{ kind: string; role_id?: string; text: string }>;
  events: SafeEvent[];
  execution: {
    local_only: boolean;
    endpoint_scope: string;
    model: string;
    runtime_label: string;
  };
  metrics: {
    duration_ms: number;
    model_calls: number;
    context_packets: number;
    critiques: number;
    claims: number;
    supported_claims: number;
    evidence_coverage: number;
    required_sections: number;
    present_required_sections: number;
    max_concurrent_context: number;
  };
  follow_up_suggestions: Array<{ suggestion_id: string; label: string; question: string }>;
};

export type RunView = {
  run_id: string;
  parent_run_id?: string;
  status: "running" | "completed" | "failed" | "cancelled";
  started_at: string;
  completed_at?: string;
  result?: Projection;
  failure?: { code: string; retryable: boolean };
  retention: RetentionView;
};

export type CaptureState = {
  enabled: boolean;
  available: boolean;
  status: "disabled" | "active" | "expired" | "purged" | "capacity_exceeded";
  expires_at?: string;
  stored_bytes: number;
  maximum_bytes: number;
};

export type ModelCallAudit = {
  model_call_id: string;
  step_id: string;
  role_id: string;
  role_class: string;
  provider_id: string;
  model_id: string;
  route: string;
  prompt_template_id: string;
  prompt_instance_id: string;
  system_prompt_sha256: string;
  request_payload_sha256: string;
  response_payload_sha256?: string;
  response_schema_sha256?: string;
  input_tokens: number;
  output_tokens: number;
  max_output_tokens: number;
  started_at: string;
  duration_ms: number;
  ttft_ms: number;
  finish_reason?: string;
  status: string;
  failure_code?: string;
};

export type RetrievalAudit = {
  retrieval_id: string;
  step_id: string;
  role_id: string;
  method: string;
  index_version?: string;
  query_sha256?: string;
  context_packet_id: string;
  evidence_ids: string[];
  evidence_sources?: Array<{
    evidence_id: string;
    source_type: string;
    locator: string;
    document_section?: string;
    content_sha256: string;
    as_of: string;
  }>;
  chunk_ids?: string[];
  document_ids?: string[];
  graph_traversal_ids?: string[];
  dropped_ids?: string[];
  estimated_tokens: number;
  status: string;
  completed_at: string;
};

export type EngineCallAudit = {
  engine_call_id: string;
  step_id: string;
  requested_by: string;
  engine_id: string;
  engine_version: string;
  operation_id: string;
  formula_version: string;
  receipt_id: string;
  receipt_sha256: string;
  input_refs: string[];
  output_refs: string[];
  evidence_refs?: string[];
  invariants_total: number;
  invariants_passed: number;
  status: string;
  generated_at: string;
};

export type ReviewAudit = {
  review_id: string;
  role_id: string;
  decision: string;
  repair_pass: number;
  approved_claims?: string[];
  rejected_claims?: string[];
};

export type ReleaseAudit = {
  answer_id: string;
  primary_intent: string;
  section_types: string[];
  claim_refs: string[];
  evidence_refs: string[];
  receipt_refs: string[];
  status: string;
};

export type IntelligenceRecord = {
  schema_version: string;
  run_id: string;
  request_id: string;
  trace_id: string;
  status: string;
  capture: CaptureState;
  started_at: string;
  completed_at?: string;
  model_calls: ModelCallAudit[];
  retrievals: RetrievalAudit[];
  engine_calls: EngineCallAudit[];
  reviews: ReviewAudit[];
  release?: ReleaseAudit;
};

export type ProtectedModelCall = {
  model_call_id: string;
  prompt_instance_id: string;
  messages: Array<{ role: string; content: string }>;
  response_format?: Record<string, unknown>;
  parameters: { model: string; max_tokens: number; temperature: number; stream: boolean };
  raw_output?: string;
};

export type ProtectedIntelligenceRecord = {
  schema_version: string;
  run_id: string;
  request_id: string;
  question: string;
  created_at: string;
  expires_at: string;
  model_calls: ProtectedModelCall[];
  receipts: Array<{ receipt_id: string; payload: unknown }>;
};
