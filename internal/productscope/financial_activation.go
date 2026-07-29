package productscope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rvbernucci/signalforge/internal/capability"
	"github.com/rvbernucci/signalforge/internal/contracts"
	"github.com/rvbernucci/signalforge/internal/data"
	"github.com/rvbernucci/signalforge/internal/engine"
	"github.com/rvbernucci/signalforge/internal/roles"
)

const CompanyFinancialActivationSchemaV2 = "signalforge/technology20-financial-activation/v2"
const FinancialMetricFreshnessPolicy = "period-end-age-760d/v1"
const FinancialConsolidationPolicy = "dimensionless-consolidated-facts-only/v1"
const CapexPerimeterPolicy = "registry-bound-per-input-accounting-perimeter/v2"
const ReviewedRevenueAliasPolicy = "company-specific-us-gaap-revenues/v1"
const ProductiveAssetsContextPerimeter = "company_reported_property_equipment_and_intangible_assets"

const (
	AccountingOutputAuthoritative = "authoritative"
	AccountingOutputContextOnly   = "context_only"
)

type ReceiptInputAccountingAuthority struct {
	InputID                  string                `json:"input_id"`
	CanonicalInput           string                `json:"canonical_input"`
	MetricID                 string                `json:"metric_id"`
	SourceFactIDs            []string              `json:"source_fact_ids"`
	MappingKey               string                `json:"mapping_key"`
	TaxonomyConcept          string                `json:"taxonomy_concept"`
	AccountingPerimeter      string                `json:"accounting_perimeter"`
	Disposition              AccountingDisposition `json:"disposition"`
	ProductLabel             string                `json:"product_label"`
	NumericallyAuthoritative bool                  `json:"numerically_authoritative"`
	ContextOnly              bool                  `json:"context_only"`
	PairRankingEligible      bool                  `json:"pair_ranking_eligible"`
}

type ReceiptAccountingAuthority struct {
	ReceiptID                    string                            `json:"receipt_id"`
	OperationID                  string                            `json:"operation_id"`
	OutputClass                  string                            `json:"output_class"`
	ProductLabel                 string                            `json:"product_label"`
	AccountingPerimeterSignature string                            `json:"accounting_perimeter_signature"`
	Inputs                       []ReceiptInputAccountingAuthority `json:"inputs"`
	PairRankingEligible          bool                              `json:"pair_ranking_eligible"`
}

type CompanyFinancialActivation struct {
	SchemaVersion             string                                `json:"schema_version"`
	UniverseID                string                                `json:"universe_id"`
	CompanyID                 string                                `json:"company_id"`
	AsOf                      time.Time                             `json:"as_of"`
	CodeCommit                string                                `json:"code_commit"`
	AccountingRegistryVersion string                                `json:"accounting_registry_version"`
	AccountingRegistrySHA256  string                                `json:"accounting_registry_sha256"`
	AccountingDecisionSHA256  string                                `json:"accounting_decision_sha256"`
	FreshnessPolicy           string                                `json:"freshness_policy"`
	ConsolidationPolicy       string                                `json:"consolidation_policy"`
	CapexPerimeterPolicy      string                                `json:"capex_perimeter_policy"`
	Receipts                  []contracts.CalculationReceipt        `json:"receipts"`
	Abstentions               []contracts.TypedAbstention           `json:"abstentions"`
	Excluded                  map[string]int                        `json:"excluded_records"`
	SourceConcepts            map[string][]string                   `json:"source_concepts"`
	SourceForms               map[string][]string                   `json:"source_forms"`
	ContextualReceipts        []contracts.CalculationReceipt        `json:"contextual_receipts,omitempty"`
	ContextualConcepts        map[string][]string                   `json:"contextual_source_concepts,omitempty"`
	ContextualPerimeters      map[string]string                     `json:"contextual_accounting_perimeters,omitempty"`
	ReceiptAuthorities        map[string]ReceiptAccountingAuthority `json:"receipt_accounting_authorities"`
	ReportSHA256              string                                `json:"report_sha256"`
}

type authorizedMetric struct {
	Metric  data.NormalizedMetric
	Fact    data.ReportedFact
	Mapping AccountingMappingAuthority
}

type operationSpec struct {
	OperationID string
	Inputs      map[string]string
	PeriodType  string
	Annual      bool
	Role        string
}

var companyOperationSpecs = []operationSpec{
	{
		OperationID: "accounting.balance_sheet_identity", PeriodType: "instant",
		Inputs: map[string]string{"assets": "total_assets", "liabilities": "total_liabilities", "equity": "stockholders_equity"},
		Role:   roles.AccountingReporting,
	},
	{
		OperationID: "financial.revenue_growth", PeriodType: "duration", Annual: true,
		Inputs: map[string]string{"revenue_current": "revenue", "revenue_prior": "revenue"},
		Role:   roles.FinancialQuality,
	},
	{
		OperationID: "financial.operating_margin", PeriodType: "duration", Annual: true,
		Inputs: map[string]string{"operating_income": "operating_income", "revenue": "revenue"},
		Role:   roles.FinancialQuality,
	},
	{
		OperationID: "financial.free_cash_flow", PeriodType: "duration", Annual: true,
		Inputs: map[string]string{"operating_cash_flow": "operating_cash_flow", "capital_expenditure": "capital_expenditure"},
		Role:   roles.FinancialQuality,
	},
	{
		OperationID: "financial.cash_conversion", PeriodType: "duration", Annual: true,
		Inputs: map[string]string{"operating_cash_flow": "operating_cash_flow", "net_income": "net_income"},
		Role:   roles.FinancialQuality,
	},
	{
		OperationID: "financial.capex_intensity", PeriodType: "duration", Annual: true,
		Inputs: map[string]string{"capital_expenditure": "capital_expenditure", "revenue": "revenue"},
		Role:   roles.FinancialQuality,
	},
	{
		OperationID: "financial.quality_of_earnings", PeriodType: "duration", Annual: true,
		Inputs: map[string]string{"operating_cash_flow": "operating_cash_flow", "net_income": "net_income"},
		Role:   roles.FinancialQuality,
	},
}

var unavailableCompanyOperations = []string{
	"financial.dilution",
	"financial.net_debt",
	"financial.current_ratio",
	"financial.debt_to_equity",
	"financial.earnings_per_share",
	"financial.roic",
	"valuation.fcff_dcf",
	"valuation.peer_multiple",
}

func BuildCompanyFinancialActivation(
	companyID string,
	metrics []data.NormalizedMetric,
	facts map[string]data.ReportedFact,
	asOf time.Time,
	codeCommit string,
) (CompanyFinancialActivation, error) {
	if companyID == "" || asOf.IsZero() || strings.TrimSpace(codeCommit) == "" {
		return CompanyFinancialActivation{}, errors.New("company, as_of, and code commit are required")
	}
	if !technologyCompany(companyID) {
		return CompanyFinancialActivation{}, errors.New("company is outside the Technology 20 authority")
	}
	report := CompanyFinancialActivation{
		SchemaVersion: CompanyFinancialActivationSchemaV2, UniverseID: UniverseID,
		CompanyID: companyID, AsOf: asOf.UTC(), CodeCommit: codeCommit,
		AccountingRegistryVersion: runtimeAccountingAuthorityRegistry.RegistryVersion,
		AccountingRegistrySHA256:  runtimeAccountingAuthorityRegistry.RegistrySHA256,
		AccountingDecisionSHA256:  runtimeAccountingProfessionalDecision.DecisionSHA256,
		FreshnessPolicy:           FinancialMetricFreshnessPolicy,
		ConsolidationPolicy:       FinancialConsolidationPolicy,
		CapexPerimeterPolicy:      CapexPerimeterPolicy, Excluded: map[string]int{},
		SourceConcepts: map[string][]string{}, SourceForms: map[string][]string{},
		ContextualConcepts: map[string][]string{}, ContextualPerimeters: map[string]string{},
		ReceiptAuthorities: map[string]ReceiptAccountingAuthority{},
	}
	eligible := make([]authorizedMetric, 0, len(metrics))
	contextualEligible := make([]authorizedMetric, 0, len(metrics))
	for _, metric := range metrics {
		if metric.CompanyID != companyID {
			continue
		}
		if err := data.ValidateNormalizedMetric(metric); err != nil {
			report.Excluded["invalid_metric_contract"]++
			continue
		}
		if metric.SourceAvailableAt.After(asOf) {
			report.Excluded["look_ahead"]++
			continue
		}
		if asOf.Sub(metric.PeriodEnd) > 760*24*time.Hour {
			report.Excluded["stale_metric_period"]++
			continue
		}
		fact, mapping, authorized := periodicFactAuthorityWithMapping(metric, facts)
		if !authorized {
			report.Excluded["periodic_fact_authority_missing"]++
			continue
		}
		if metric.Unit != "USD" || metric.Currency != "USD" {
			report.Excluded["unit_or_currency_mismatch"]++
			continue
		}
		if !effectiveAccountingMappingNumericallyAuthoritative(
			mapping,
			runtimeAccountingAuthorityRegistry,
			runtimeAccountingProfessionalDecision,
		) {
			if effectiveAccountingMappingContextDisplayAuthorized(
				mapping,
				runtimeAccountingAuthorityRegistry,
				runtimeAccountingProfessionalDecision,
			) {
				report.ContextualConcepts[metric.CanonicalMetric] = appendUniqueSorted(
					report.ContextualConcepts[metric.CanonicalMetric], fact.Concept,
				)
				contextualEligible = append(contextualEligible, authorizedMetric{
					Metric: metric, Fact: fact, Mapping: mapping,
				})
				continue
			}
			report.Excluded["unreviewed_semantic_mapping"]++
			continue
		}
		report.SourceConcepts[metric.CanonicalMetric] = appendUniqueSorted(
			report.SourceConcepts[metric.CanonicalMetric], fact.Concept,
		)
		report.SourceForms[metric.CanonicalMetric] = appendUniqueSorted(
			report.SourceForms[metric.CanonicalMetric], fact.FormType,
		)
		item := authorizedMetric{Metric: metric, Fact: fact, Mapping: mapping}
		eligible = append(eligible, item)
		if metric.CanonicalMetric != "capital_expenditure" {
			contextualEligible = append(contextualEligible, item)
		}
	}
	executor, err := engine.NewWithClock(codeCommit, func() time.Time { return asOf.UTC() })
	if err != nil {
		return CompanyFinancialActivation{}, err
	}
	runID := "run-activation-" + strings.TrimPrefix(companyID, "sec-cik:")
	for _, spec := range companyOperationSpecs {
		inputs, reason := selectOperationInputs(eligible, spec)
		if reason != "" {
			report.Abstentions = append(report.Abstentions, operationAbstention(
				runID, companyID, spec.OperationID, reason, asOf,
			))
			continue
		}
		request, requestErr := operationRequest(runID, companyID, spec, inputs)
		if requestErr != nil {
			return CompanyFinancialActivation{}, requestErr
		}
		result := executor.Execute(request)
		if result.Failure != nil {
			report.Abstentions = append(report.Abstentions, operationAbstention(
				runID, companyID, spec.OperationID, result.Failure.FailureCode, asOf,
			))
			continue
		}
		report.Receipts = append(report.Receipts, *result.Receipt)
		authority, authorityErr := receiptAccountingAuthority(*result.Receipt, spec, inputs, false)
		if authorityErr != nil {
			return CompanyFinancialActivation{}, authorityErr
		}
		report.ReceiptAuthorities[result.Receipt.ReceiptID] = authority
	}
	if len(report.ContextualConcepts["capital_expenditure"]) > 0 {
		for _, spec := range companyOperationSpecs {
			if spec.OperationID != "financial.capex_intensity" &&
				spec.OperationID != "financial.free_cash_flow" {
				continue
			}
			inputs, reason := selectOperationInputs(contextualEligible, spec)
			if reason != "" {
				continue
			}
			request, requestErr := operationRequest(runID+"-context", companyID, spec, inputs)
			if requestErr != nil {
				return CompanyFinancialActivation{}, requestErr
			}
			result := executor.Execute(request)
			if result.Failure != nil {
				continue
			}
			report.ContextualReceipts = append(report.ContextualReceipts, *result.Receipt)
			authority, authorityErr := receiptAccountingAuthority(*result.Receipt, spec, inputs, true)
			if authorityErr != nil {
				return CompanyFinancialActivation{}, authorityErr
			}
			report.ReceiptAuthorities[result.Receipt.ReceiptID] = authority
			report.ContextualPerimeters[spec.OperationID] = contextOnlyPerimeter(authority)
		}
	}
	for _, operationID := range unavailableCompanyOperations {
		report.Abstentions = append(report.Abstentions, operationAbstention(
			runID, companyID, operationID, "required_normalized_inputs_unavailable", asOf,
		))
	}
	sort.Slice(report.Receipts, func(i, j int) bool {
		return report.Receipts[i].OperationID < report.Receipts[j].OperationID
	})
	sort.Slice(report.ContextualReceipts, func(i, j int) bool {
		return report.ContextualReceipts[i].OperationID < report.ContextualReceipts[j].OperationID
	})
	sort.Slice(report.Abstentions, func(i, j int) bool {
		return report.Abstentions[i].MetricIDs[0] < report.Abstentions[j].MetricIDs[0]
	})
	hash, err := companyFinancialActivationHash(report)
	if err != nil {
		return CompanyFinancialActivation{}, err
	}
	report.ReportSHA256 = hash
	return report, ValidateCompanyFinancialActivation(report)
}

func ValidateCompanyFinancialActivation(report CompanyFinancialActivation) error {
	if report.SchemaVersion != CompanyFinancialActivationSchemaV2 || report.UniverseID != UniverseID ||
		report.CompanyID == "" || report.AsOf.IsZero() || report.CodeCommit == "" || report.ReportSHA256 == "" {
		return errors.New("company financial activation envelope is invalid")
	}
	if report.AccountingRegistryVersion != runtimeAccountingAuthorityRegistry.RegistryVersion ||
		report.AccountingRegistrySHA256 != runtimeAccountingAuthorityRegistry.RegistrySHA256 {
		return errors.New("company financial activation accounting registry is missing or mismatched")
	}
	if report.AccountingDecisionSHA256 != runtimeAccountingProfessionalDecision.DecisionSHA256 ||
		ValidateAccountingProfessionalDecision(
			runtimeAccountingProfessionalDecision,
			runtimeAccountingAuthorityRegistry,
		) != nil {
		return errors.New("company financial activation accounting decision is missing or mismatched")
	}
	if report.FreshnessPolicy != FinancialMetricFreshnessPolicy {
		return errors.New("company financial activation freshness policy is invalid")
	}
	if report.ConsolidationPolicy != FinancialConsolidationPolicy ||
		report.CapexPerimeterPolicy != CapexPerimeterPolicy {
		return errors.New("company financial activation perimeter policy is invalid")
	}
	for metricID, concepts := range report.SourceConcepts {
		for _, concept := range concepts {
			mapping := ResolveAccountingMapping(
				runtimeAccountingAuthorityRegistry,
				report.CompanyID,
				metricID,
				"us-gaap",
				concept,
			)
			if !effectiveAccountingMappingNumericallyAuthoritative(
				mapping,
				runtimeAccountingAuthorityRegistry,
				runtimeAccountingProfessionalDecision,
			) {
				return errors.New("company financial activation contains an unauthorized numerical concept")
			}
		}
	}
	for metricID, concepts := range report.ContextualConcepts {
		for _, concept := range concepts {
			mapping := ResolveAccountingMapping(
				runtimeAccountingAuthorityRegistry,
				report.CompanyID,
				metricID,
				"us-gaap",
				concept,
			)
			if !effectiveAccountingMappingContextDisplayAuthorized(
				mapping,
				runtimeAccountingAuthorityRegistry,
				runtimeAccountingProfessionalDecision,
			) {
				return errors.New("company financial activation contains an unauthorized context-only concept")
			}
		}
	}
	for _, forms := range report.SourceForms {
		for _, form := range forms {
			switch form {
			case "10-K", "10-K/A", "10-Q", "10-Q/A":
			default:
				return errors.New("company financial activation contains a non-periodic source form")
			}
		}
	}
	operations := map[string]bool{}
	authorityCoverage := map[string]bool{}
	for _, receipt := range report.Receipts {
		if err := contracts.ValidateCalculationReceipt(receipt); err != nil {
			return err
		}
		if operations[receipt.OperationID] || len(receipt.Scope.CompanyIDs) != 1 ||
			receipt.Scope.CompanyIDs[0] != report.CompanyID {
			return errors.New("company financial activation contains duplicate or cross-company receipt")
		}
		operations[receipt.OperationID] = true
		authority, exists := report.ReceiptAuthorities[receipt.ReceiptID]
		if !exists {
			return errors.New("company financial activation receipt lacks per-input accounting authority")
		}
		if err := validateReceiptAccountingAuthority(receipt, authority, false); err != nil {
			return err
		}
		authorityCoverage[receipt.ReceiptID] = true
	}
	for _, abstention := range report.Abstentions {
		if len(abstention.MetricIDs) != 1 || len(abstention.CompanyIDs) != 1 ||
			abstention.CompanyIDs[0] != report.CompanyID || operations[abstention.MetricIDs[0]] {
			return errors.New("company financial activation contains invalid abstention")
		}
		operations[abstention.MetricIDs[0]] = true
	}
	contextualOperations := map[string]bool{}
	for _, receipt := range report.ContextualReceipts {
		if err := contracts.ValidateCalculationReceipt(receipt); err != nil {
			return err
		}
		if receipt.OperationID != "financial.capex_intensity" &&
			receipt.OperationID != "financial.free_cash_flow" {
			return errors.New("company financial activation contains an unauthorized contextual operation")
		}
		if contextualOperations[receipt.OperationID] || len(receipt.Scope.CompanyIDs) != 1 ||
			receipt.Scope.CompanyIDs[0] != report.CompanyID ||
			!operations[receipt.OperationID] {
			return errors.New("company financial activation contains invalid contextual authority")
		}
		authority, exists := report.ReceiptAuthorities[receipt.ReceiptID]
		if !exists {
			return errors.New("company financial activation contextual receipt lacks per-input accounting authority")
		}
		if err := validateReceiptAccountingAuthority(receipt, authority, true); err != nil {
			return err
		}
		if report.ContextualPerimeters[receipt.OperationID] != contextOnlyPerimeter(authority) ||
			report.ContextualPerimeters[receipt.OperationID] == "" {
			return errors.New("company financial activation contains invalid contextual authority")
		}
		contextualOperations[receipt.OperationID] = true
		authorityCoverage[receipt.ReceiptID] = true
	}
	if len(report.ContextualPerimeters) != len(contextualOperations) {
		return errors.New("company financial activation contextual perimeter coverage is invalid")
	}
	if len(report.ReceiptAuthorities) != len(authorityCoverage) {
		return errors.New("company financial activation contains orphan receipt authority")
	}
	if len(operations) != len(companyOperationSpecs)+len(unavailableCompanyOperations) {
		return errors.New("company financial activation has incomplete operation coverage")
	}
	expected, err := companyFinancialActivationHash(report)
	if err != nil {
		return err
	}
	if expected != report.ReportSHA256 {
		return errors.New("company financial activation hash mismatch")
	}
	return nil
}

func selectOperationInputs(metrics []authorizedMetric, spec operationSpec) (map[string]authorizedMetric, string) {
	if spec.OperationID == "financial.revenue_growth" {
		return selectGrowthInputs(metrics)
	}
	required := make(map[string]bool)
	for _, canonical := range spec.Inputs {
		required[canonical] = true
	}
	type period struct {
		start time.Time
		end   time.Time
	}
	sets := map[period]map[string]authorizedMetric{}
	for _, item := range metrics {
		metric := item.Metric
		if !required[metric.CanonicalMetric] || metric.PeriodType != spec.PeriodType ||
			(spec.Annual && !annualDuration(metric.PeriodStart, metric.PeriodEnd)) {
			continue
		}
		key := period{start: metric.PeriodStart, end: metric.PeriodEnd}
		if sets[key] == nil {
			sets[key] = map[string]authorizedMetric{}
		}
		prior, exists := sets[key][metric.CanonicalMetric]
		if !exists || betterAuthorizedMetric(item, prior) {
			sets[key][metric.CanonicalMetric] = item
		}
	}
	var selected map[string]authorizedMetric
	var selectedEnd time.Time
	for key, values := range sets {
		if len(values) != len(required) || (!selectedEnd.IsZero() && !key.end.After(selectedEnd)) {
			continue
		}
		selected, selectedEnd = values, key.end
	}
	if len(selected) != len(required) {
		return nil, "aligned_standardized_inputs_unavailable"
	}
	result := map[string]authorizedMetric{}
	for inputID, canonical := range spec.Inputs {
		result[inputID] = selected[canonical]
	}
	return result, ""
}

func betterAuthorizedMetric(candidate, prior authorizedMetric) bool {
	candidateRank := accountingMappingSelectionRank(candidate.Mapping)
	priorRank := accountingMappingSelectionRank(prior.Mapping)
	if candidateRank != priorRank {
		return candidateRank > priorRank
	}
	if !candidate.Metric.SourceAvailableAt.Equal(prior.Metric.SourceAvailableAt) {
		return candidate.Metric.SourceAvailableAt.After(prior.Metric.SourceAvailableAt)
	}
	return candidate.Fact.FactID > prior.Fact.FactID
}

func accountingMappingSelectionRank(mapping AccountingMappingAuthority) int {
	switch {
	case mapping.Disposition == AccountingCanonical:
		return 3
	case effectiveAccountingMappingNumericallyAuthoritative(
		mapping,
		runtimeAccountingAuthorityRegistry,
		runtimeAccountingProfessionalDecision,
	):
		return 2
	case effectiveAccountingMappingContextDisplayAuthorized(
		mapping,
		runtimeAccountingAuthorityRegistry,
		runtimeAccountingProfessionalDecision,
	):
		return 1
	default:
		return 0
	}
}

func selectGrowthInputs(metrics []authorizedMetric) (map[string]authorizedMetric, string) {
	byPeriod := map[string]authorizedMetric{}
	for _, item := range metrics {
		metric := item.Metric
		if metric.CanonicalMetric != "revenue" || metric.PeriodType != "duration" ||
			!annualDuration(metric.PeriodStart, metric.PeriodEnd) {
			continue
		}
		key := periodLabel(metric)
		prior, exists := byPeriod[key]
		if !exists || betterAuthorizedMetric(item, prior) {
			byPeriod[key] = item
		}
	}
	values := make([]authorizedMetric, 0, len(byPeriod))
	for _, metric := range byPeriod {
		values = append(values, metric)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].Metric.PeriodEnd.After(values[j].Metric.PeriodEnd)
	})
	if len(values) < 2 {
		return nil, "two_annual_standardized_revenue_periods_unavailable"
	}
	return map[string]authorizedMetric{"revenue_current": values[0], "revenue_prior": values[1]}, ""
}

func operationRequest(
	runID, companyID string,
	spec operationSpec,
	selected map[string]authorizedMetric,
) (contracts.EngineRequest, error) {
	operation, exists := capability.RuntimeRegistry().Get(spec.OperationID)
	if !exists {
		return contracts.EngineRequest{}, fmt.Errorf("operation %q is not registered", spec.OperationID)
	}
	inputIDs := make([]string, 0, len(selected))
	for inputID := range selected {
		inputIDs = append(inputIDs, inputID)
	}
	sort.Strings(inputIDs)
	inputs := make([]contracts.EngineInput, 0, len(inputIDs))
	periods := make([]string, 0, len(inputIDs))
	sourceAsOf := time.Time{}
	for _, inputID := range inputIDs {
		metric := selected[inputID].Metric
		period := periodLabel(metric)
		available := metric.SourceAvailableAt
		inputs = append(inputs, contracts.EngineInput{
			InputID: inputID,
			Quantity: contracts.Quantity{
				Value: metric.Value, Unit: "currency", Currency: metric.Currency,
				Period: period, AsOf: &available,
			},
			Status: "normalized", EvidenceRefs: []string{selected[inputID].Fact.FactID},
		})
		periods = append(periods, period)
		if sourceAsOf.IsZero() || available.After(sourceAsOf) {
			sourceAsOf = available
		}
	}
	sort.Strings(periods)
	periods = uniqueStrings(periods)
	return contracts.EngineRequest{
		SchemaVersion: contracts.SchemaVersionV1,
		RequestID:     "request-" + strings.TrimPrefix(companyID, "sec-cik:") + "-" + strings.ReplaceAll(spec.OperationID, ".", "-"),
		RunID:         runID, StepID: "engine-" + strings.ReplaceAll(spec.OperationID, ".", "-"),
		RequestedBy: spec.Role, EngineID: operation.Engine, OperationID: operation.ID,
		FormulaVersion: operation.FormulaVersion,
		Scope: contracts.Scope{
			CompanyIDs: []string{companyID}, Periods: periods, AsOf: sourceAsOf,
		},
		Inputs: inputs, PrecisionPolicy: operation.NumericalPolicy,
		RequestedOutputs: append([]string(nil), operation.Outputs...),
	}, nil
}

func operationAbstention(runID, companyID, operationID, reason string, at time.Time) contracts.TypedAbstention {
	digest := sha256.Sum256([]byte(runID + "\n" + operationID + "\n" + reason))
	return contracts.TypedAbstention{
		SchemaVersion: contracts.TypedAbstentionSchemaV1,
		AbstentionID:  "abstention-" + hex.EncodeToString(digest[:10]),
		RequestID:     "request-" + strings.TrimPrefix(companyID, "sec-cik:") + "-financial-activation",
		RunID:         runID, Code: reason,
		Message:    "SignalForge withheld " + operationID + " because " + strings.ReplaceAll(reason, "_", " ") + ".",
		CompanyIDs: []string{companyID}, MetricIDs: []string{operationID}, GeneratedAt: at.UTC(),
	}
}

func annualDuration(start, end time.Time) bool {
	days := int(end.Sub(start).Hours() / 24)
	return days >= 300 && days <= 380
}

func periodLabel(metric data.NormalizedMetric) string {
	return metric.PeriodStart.Format("2006-01-02") + "/" + metric.PeriodEnd.Format("2006-01-02")
}

func technologyCompany(companyID string) bool {
	for _, company := range Companies() {
		if company.CompanyID == companyID {
			return true
		}
	}
	return false
}

func periodicFactAuthority(metric data.NormalizedMetric, facts map[string]data.ReportedFact) (data.ReportedFact, bool) {
	fact, _, found := periodicFactAuthorityWithMapping(metric, facts)
	return fact, found
}

func periodicFactAuthorityWithMapping(
	metric data.NormalizedMetric,
	facts map[string]data.ReportedFact,
) (data.ReportedFact, AccountingMappingAuthority, bool) {
	var selected data.ReportedFact
	var selectedMapping AccountingMappingAuthority
	found := false
	for _, factID := range metric.SourceFactIDs {
		fact, exists := facts[factID]
		if !exists || fact.CompanyID != metric.CompanyID || fact.Unit != metric.Unit ||
			fact.Value != metric.Value || fact.AvailableAt.After(metric.SourceAvailableAt) ||
			len(fact.Dimensions) != 0 {
			continue
		}
		switch fact.FormType {
		case "10-K", "10-K/A", "10-Q", "10-Q/A":
		default:
			continue
		}
		periodMatches := metric.PeriodType == "instant" && fact.InstantDate != nil &&
			fact.InstantDate.Equal(metric.PeriodEnd)
		if metric.PeriodType == "duration" && fact.StartDate != nil && fact.EndDate != nil &&
			fact.StartDate.Equal(metric.PeriodStart) && fact.EndDate.Equal(metric.PeriodEnd) {
			periodMatches = true
		}
		if !periodMatches {
			continue
		}
		namespace := fact.Taxonomy
		if namespace == "" {
			namespace = "us-gaap"
		}
		mapping := ResolveAccountingMapping(
			runtimeAccountingAuthorityRegistry,
			metric.CompanyID,
			metric.CanonicalMetric,
			namespace,
			fact.Concept,
		)
		candidate := authorizedMetric{Metric: metric, Fact: fact, Mapping: mapping}
		prior := authorizedMetric{Metric: metric, Fact: selected, Mapping: selectedMapping}
		if !found || betterAuthorizedMetric(candidate, prior) {
			selected, selectedMapping, found = fact, mapping, true
		}
	}
	return selected, selectedMapping, found
}

func financialSemanticAuthority(metric data.NormalizedMetric, fact data.ReportedFact) bool {
	mapping := ResolveAccountingMapping(
		runtimeAccountingAuthorityRegistry,
		metric.CompanyID,
		metric.CanonicalMetric,
		fact.Taxonomy,
		fact.Concept,
	)
	return effectiveAccountingMappingNumericallyAuthoritative(
		mapping,
		runtimeAccountingAuthorityRegistry,
		runtimeAccountingProfessionalDecision,
	)
}

func reviewedFinancialConceptAlias(companyID, canonicalMetric, concept string) bool {
	mapping := ResolveAccountingMapping(
		runtimeAccountingAuthorityRegistry, companyID, canonicalMetric, "us-gaap", concept,
	)
	return effectiveAccountingMappingNumericallyAuthoritative(
		mapping,
		runtimeAccountingAuthorityRegistry,
		runtimeAccountingProfessionalDecision,
	) &&
		mapping.Disposition == AccountingReviewedAlias
}

func contextOnlyFinancialAuthority(metric data.NormalizedMetric, fact data.ReportedFact) bool {
	mapping := ResolveAccountingMapping(
		runtimeAccountingAuthorityRegistry,
		metric.CompanyID,
		metric.CanonicalMetric,
		fact.Taxonomy,
		fact.Concept,
	)
	return effectiveAccountingMappingContextDisplayAuthorized(
		mapping,
		runtimeAccountingAuthorityRegistry,
		runtimeAccountingProfessionalDecision,
	)
}

func reviewedContextOnlyConcept(companyID, canonicalMetric, concept string) bool {
	mapping := ResolveAccountingMapping(
		runtimeAccountingAuthorityRegistry, companyID, canonicalMetric, "us-gaap", concept,
	)
	return effectiveAccountingMappingContextDisplayAuthorized(
		mapping,
		runtimeAccountingAuthorityRegistry,
		runtimeAccountingProfessionalDecision,
	)
}

func receiptAccountingAuthority(
	receipt contracts.CalculationReceipt,
	spec operationSpec,
	selected map[string]authorizedMetric,
	contextOnly bool,
) (ReceiptAccountingAuthority, error) {
	inputIDs := make([]string, 0, len(selected))
	for inputID := range selected {
		inputIDs = append(inputIDs, inputID)
	}
	sort.Strings(inputIDs)
	authority := ReceiptAccountingAuthority{
		ReceiptID: receipt.ReceiptID, OperationID: receipt.OperationID,
		OutputClass:         AccountingOutputAuthoritative,
		ProductLabel:        operationProductLabel(receipt.OperationID, selected, contextOnly),
		PairRankingEligible: !contextOnly,
	}
	if contextOnly {
		authority.OutputClass = AccountingOutputContextOnly
		authority.PairRankingEligible = false
	}
	signatureParts := make([]string, 0, len(inputIDs))
	for _, inputID := range inputIDs {
		item := selected[inputID]
		numerical := effectiveAccountingMappingNumericallyAuthoritative(
			item.Mapping,
			runtimeAccountingAuthorityRegistry,
			runtimeAccountingProfessionalDecision,
		)
		context := effectiveAccountingMappingContextDisplayAuthorized(
			item.Mapping,
			runtimeAccountingAuthorityRegistry,
			runtimeAccountingProfessionalDecision,
		)
		if !numerical && !context {
			return ReceiptAccountingAuthority{}, fmt.Errorf(
				"receipt %s contains unauthorized input %s",
				receipt.ReceiptID,
				inputID,
			)
		}
		inputAuthority := ReceiptInputAccountingAuthority{
			InputID: inputID, CanonicalInput: item.Metric.CanonicalMetric,
			MetricID: item.Metric.MetricID, SourceFactIDs: []string{item.Fact.FactID},
			MappingKey: item.Mapping.MappingKey, TaxonomyConcept: item.Fact.Concept,
			AccountingPerimeter: item.Mapping.AccountingPerimeter,
			Disposition:         item.Mapping.Disposition, ProductLabel: item.Mapping.ProductLabel,
			NumericallyAuthoritative: numerical, ContextOnly: context,
			PairRankingEligible: numerical && item.Mapping.ComparableRankingEligible,
		}
		if !inputAuthority.PairRankingEligible {
			authority.PairRankingEligible = false
		}
		authority.Inputs = append(authority.Inputs, inputAuthority)
		signatureParts = append(signatureParts, inputID+"="+item.Mapping.AccountingPerimeter)
	}
	authority.AccountingPerimeterSignature = strings.Join(signatureParts, ";")
	return authority, nil
}

func operationProductLabel(
	operationID string,
	selected map[string]authorizedMetric,
	contextOnly bool,
) string {
	if contextOnly {
		if operationID == "financial.capex_intensity" {
			return "reported reinvestment intensity"
		}
		if operationID == "financial.free_cash_flow" {
			return "residual cash proxy"
		}
	}
	if operationID == "financial.free_cash_flow" {
		for _, item := range selected {
			if item.Metric.CanonicalMetric == "capital_expenditure" &&
				item.Mapping.Disposition == AccountingReviewedAlias {
				return "simple FCF using " + item.Mapping.ProductLabel
			}
		}
		return "simple FCF"
	}
	if operationID == "financial.capex_intensity" {
		for _, item := range selected {
			if item.Metric.CanonicalMetric == "capital_expenditure" &&
				item.Mapping.Disposition == AccountingReviewedAlias {
				return item.Mapping.ProductLabel + " intensity"
			}
		}
	}
	return operationID
}

func contextOnlyPerimeter(authority ReceiptAccountingAuthority) string {
	for _, input := range authority.Inputs {
		if input.ContextOnly {
			return input.AccountingPerimeter
		}
	}
	return ""
}

func validateReceiptAccountingAuthority(
	receipt contracts.CalculationReceipt,
	authority ReceiptAccountingAuthority,
	contextOnly bool,
) error {
	if authority.ReceiptID != receipt.ReceiptID ||
		authority.OperationID != receipt.OperationID ||
		authority.ProductLabel == "" ||
		authority.AccountingPerimeterSignature == "" ||
		len(authority.Inputs) != len(receipt.NormalizedInputs) {
		return errors.New("receipt accounting authority envelope is invalid")
	}
	if contextOnly {
		if authority.OutputClass != AccountingOutputContextOnly ||
			authority.PairRankingEligible {
			return errors.New("context-only receipt authority cannot enter pair ranking")
		}
	} else if authority.OutputClass != AccountingOutputAuthoritative {
		return errors.New("numerical receipt authority has an invalid output class")
	}
	receiptInputs := map[string]contracts.EngineInput{}
	for _, input := range receipt.NormalizedInputs {
		if input.InputID == "" || receiptInputs[input.InputID].InputID != "" {
			return errors.New("receipt contains duplicate or empty input IDs")
		}
		receiptInputs[input.InputID] = input
	}
	signatureParts := make([]string, 0, len(authority.Inputs))
	seen := map[string]bool{}
	hasContext := false
	expectedPairEligibility := !contextOnly
	for _, input := range authority.Inputs {
		if input.InputID == "" || seen[input.InputID] ||
			input.CanonicalInput == "" || input.MetricID == "" ||
			len(input.SourceFactIDs) != 1 || input.MappingKey == "" ||
			input.TaxonomyConcept == "" || input.AccountingPerimeter == "" ||
			input.ProductLabel == "" {
			return errors.New("receipt contains incomplete per-input accounting authority")
		}
		seen[input.InputID] = true
		receiptInput, exists := receiptInputs[input.InputID]
		if !exists || !sameStringSet(receiptInput.EvidenceRefs, input.SourceFactIDs) {
			return errors.New("receipt accounting authority does not match receipt evidence")
		}
		mapping := ResolveAccountingMapping(
			runtimeAccountingAuthorityRegistry,
			receipt.Scope.CompanyIDs[0],
			input.CanonicalInput,
			"us-gaap",
			input.TaxonomyConcept,
		)
		if mapping.MappingKey != input.MappingKey ||
			mapping.AccountingPerimeter != input.AccountingPerimeter ||
			mapping.Disposition != input.Disposition ||
			mapping.ProductLabel != input.ProductLabel ||
			!stringSliceContains(mapping.AuthorizedOperations, receipt.OperationID) {
			return errors.New("receipt accounting authority does not match the approved registry mapping")
		}
		numerical := effectiveAccountingMappingNumericallyAuthoritative(
			mapping,
			runtimeAccountingAuthorityRegistry,
			runtimeAccountingProfessionalDecision,
		)
		context := effectiveAccountingMappingContextDisplayAuthorized(
			mapping,
			runtimeAccountingAuthorityRegistry,
			runtimeAccountingProfessionalDecision,
		)
		pairEligible := numerical && mapping.ComparableRankingEligible
		if input.NumericallyAuthoritative != numerical ||
			input.ContextOnly != context ||
			input.PairRankingEligible != pairEligible ||
			(!numerical && !context) {
			return errors.New("receipt accounting authority has invalid effective permissions")
		}
		if context {
			hasContext = true
		}
		if !pairEligible {
			expectedPairEligibility = false
		}
		signatureParts = append(
			signatureParts,
			input.InputID+"="+input.AccountingPerimeter,
		)
	}
	sort.Strings(signatureParts)
	if authority.AccountingPerimeterSignature != strings.Join(signatureParts, ";") ||
		authority.PairRankingEligible != expectedPairEligibility ||
		contextOnly != hasContext {
		return errors.New("receipt accounting authority perimeter or release class is invalid")
	}
	return nil
}

func stringSliceContains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func appendUniqueSorted(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	values = append(values, value)
	sort.Strings(values)
	return values
}

func companyFinancialActivationHash(report CompanyFinancialActivation) (string, error) {
	report.ReportSHA256 = ""
	payload, err := json.Marshal(report)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
