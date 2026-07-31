package productscope

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rvbernucci/signalforge/internal/comparability"
	"github.com/rvbernucci/signalforge/internal/contracts"
)

const PeerEvaluationSuiteSchemaV1 = "signalforge/technology20-peer-evaluation/v1"

type PeerEvaluationSuite struct {
	SchemaVersion           string                 `json:"schema_version"`
	UniverseID              string                 `json:"universe_id"`
	AsOf                    time.Time              `json:"as_of"`
	PolicyVersion           string                 `json:"policy_version"`
	PromotionDecisionSHA256 string                 `json:"promotion_decision_sha256,omitempty"`
	Lanes                   []PeerEvaluationResult `json:"lanes"`
	ClaimBoundary           string                 `json:"claim_boundary"`
}

type PeerEvaluationResult struct {
	LaneID                  string                                 `json:"lane_id"`
	CompanyIDs              []string                               `json:"company_ids"`
	Receipts                []contracts.MetricComparabilityReceipt `json:"receipts"`
	Abstentions             []contracts.TypedAbstention            `json:"abstentions"`
	Releasable              []string                               `json:"releasable_metric_ids"`
	ContextOnly             []string                               `json:"context_only_metric_ids,omitempty"`
	Withheld                []string                               `json:"withheld_metric_ids"`
	Promoted                bool                                   `json:"promoted"`
	ReasonCodes             []string                               `json:"reason_codes"`
	EnvelopeSHA256          string                                 `json:"envelope_sha256,omitempty"`
	PromotionEvidenceSHA256 []string                               `json:"promotion_evidence_sha256,omitempty"`
}

func BuildPeerEvaluationSuite(
	catalog PublicCatalog,
	reports map[string]CompanyFinancialActivation,
	generatedAt time.Time,
) (PeerEvaluationSuite, error) {
	if err := ValidatePublicCatalog(catalog); err != nil {
		return PeerEvaluationSuite{}, err
	}
	if generatedAt.Before(catalog.AsOf) {
		return PeerEvaluationSuite{}, errors.New("peer evaluation cannot precede catalog authority")
	}
	reportByCompany := map[string]map[string]contracts.CalculationReceipt{}
	contextByCompany := map[string]map[string]contracts.CalculationReceipt{}
	for companyID, report := range reports {
		if err := ValidateCompanyFinancialActivation(report); err != nil {
			return PeerEvaluationSuite{}, err
		}
		reportByCompany[companyID] = map[string]contracts.CalculationReceipt{}
		contextByCompany[companyID] = map[string]contracts.CalculationReceipt{}
		for _, receipt := range report.Receipts {
			reportByCompany[companyID][receipt.OperationID] = receipt
		}
		for _, receipt := range report.ContextualReceipts {
			contextByCompany[companyID][receipt.OperationID] = receipt
		}
	}
	companyByID := map[string]PublicCompany{}
	for _, company := range catalog.Companies {
		companyByID[company.CompanyID] = company
	}
	suite := PeerEvaluationSuite{
		SchemaVersion: PeerEvaluationSuiteSchemaV1, UniverseID: UniverseID,
		AsOf: catalog.AsOf, PolicyVersion: comparability.AnnualPolicyVersionV1,
		ClaimBoundary: "Candidate peer lanes are evaluated metric by metric. A releasable receipt authorizes only that metric and caveats; this report does not promote any pair without separate journeys and professional review.",
	}
	for _, lane := range catalog.PeerLanes {
		result := PeerEvaluationResult{
			LaneID: lane.LaneID, CompanyIDs: append([]string(nil), lane.CompanyIDs...),
			Receipts: []contracts.MetricComparabilityReceipt{}, Abstentions: []contracts.TypedAbstention{},
			Releasable: []string{}, ContextOnly: []string{}, Withheld: []string{},
			ReasonCodes: []string{"peer_journeys_and_professional_review_pending"},
		}
		leftReport, leftOK := reports[lane.CompanyIDs[0]]
		rightReport, rightOK := reports[lane.CompanyIDs[1]]
		if !leftOK || !rightOK {
			return PeerEvaluationSuite{}, errors.New("peer evaluation is missing company authority")
		}
		for _, metricID := range lane.AllowedMetricIDs {
			left, leftFound := reportByCompany[lane.CompanyIDs[0]][metricID]
			right, rightFound := reportByCompany[lane.CompanyIDs[1]][metricID]
			leftContext, rightContext := false, false
			if !leftFound {
				left, leftFound = contextByCompany[lane.CompanyIDs[0]][metricID]
				leftContext = leftFound
			}
			if !rightFound {
				right, rightFound = contextByCompany[lane.CompanyIDs[1]][metricID]
				rightContext = rightFound
			}
			if !leftFound || !rightFound {
				result.Abstentions = append(result.Abstentions, comparisonAbstention(
					lane, metricID, generatedAt, "comparison_operand_unavailable",
				))
				result.Withheld = append(result.Withheld, metricID)
				continue
			}
			leftOperand, err := financialOperand(
				companyByID[lane.CompanyIDs[0]],
				leftReport,
				left,
				leftContext,
			)
			if err != nil {
				return PeerEvaluationSuite{}, err
			}
			rightOperand, err := financialOperand(
				companyByID[lane.CompanyIDs[1]],
				rightReport,
				right,
				rightContext,
			)
			if err != nil {
				return PeerEvaluationSuite{}, err
			}
			request, err := contracts.PopulateMetricComparabilityRequestHash(contracts.MetricComparabilityRequest{
				SchemaVersion: contracts.ComparabilityRequestSchemaV2,
				RequestID:     "comparison-" + lane.LaneID + "-" + sanitizeMetricID(metricID),
				RunID:         "peer-evaluation-" + lane.LaneID, LaneID: lane.LaneID,
				AsOf: catalog.AsOf, ReviewerPolicyVersion: comparability.AnnualPolicyVersionV1,
				Operands: []contracts.MetricComparisonOperand{
					leftOperand,
					rightOperand,
				},
			})
			if err != nil {
				return PeerEvaluationSuite{}, err
			}
			receipt, err := comparability.Evaluate(request, generatedAt, comparability.AnnualDurationPolicy())
			if err != nil {
				return PeerEvaluationSuite{}, err
			}
			result.Receipts = append(result.Receipts, receipt)
			if comparability.IsReleasable(receipt.Disposition) {
				result.Releasable = append(result.Releasable, metricID)
			} else if receipt.Disposition == contracts.ComparisonContextOnly {
				result.ContextOnly = append(result.ContextOnly, metricID)
			} else {
				result.Withheld = append(result.Withheld, metricID)
			}
		}
		sort.Strings(result.Releasable)
		sort.Strings(result.ContextOnly)
		sort.Strings(result.Withheld)
		suite.Lanes = append(suite.Lanes, result)
	}
	sort.Slice(suite.Lanes, func(i, j int) bool { return suite.Lanes[i].LaneID < suite.Lanes[j].LaneID })
	return suite, ValidatePeerEvaluationSuite(suite)
}

func financialOperand(
	company PublicCompany,
	report CompanyFinancialActivation,
	receipt contracts.CalculationReceipt,
	contextOnly bool,
) (contracts.MetricComparisonOperand, error) {
	authority, exists := report.ReceiptAuthorities[receipt.ReceiptID]
	if !exists {
		return contracts.MetricComparisonOperand{}, errors.New(
			"financial operand lacks per-input accounting authority",
		)
	}
	currentPeriod := receipt.Scope.Periods[len(receipt.Scope.Periods)-1]
	start, end := parseReceiptPeriod(currentPeriod)
	var fiscalStart *time.Time
	if !start.Equal(end) {
		fiscalStart = &start
	}
	output := receipt.Outputs[0].Quantity
	concepts := operationConcepts(report, receipt, contextOnly)
	accountingInputs := make([]contracts.AccountingInputComparisonAuthority, 0, len(authority.Inputs))
	for _, input := range authority.Inputs {
		accountingInputs = append(accountingInputs, contracts.AccountingInputComparisonAuthority{
			InputID: input.InputID, CanonicalInput: input.CanonicalInput,
			MappingKey: input.MappingKey, TaxonomyConcept: input.TaxonomyConcept,
			AccountingPerimeter: input.AccountingPerimeter,
			Disposition:         string(input.Disposition), ProductLabel: input.ProductLabel,
			PairRankingEligible: input.PairRankingEligible,
		})
	}
	operand := contracts.MetricComparisonOperand{
		CompanyID: company.CompanyID, SecurityID: "ticker:" + company.PrimaryTicker,
		SourceObservationIDs: append([]string(nil), receipt.EvidenceRefs...),
		SourceHashes:         sourceHashes(receipt.EvidenceRefs),
		AvailableAt:          receipt.SourceAsOf, RetrievedAt: report.AsOf,
		CanonicalMetricID: receipt.OperationID, MetricVersion: receipt.FormulaVersion,
		TaxonomyConcept:    strings.Join(concepts, "+"),
		ExtensionMappingID: reviewedOperationMappingID(report, receipt, concepts),
		Value:              output.Value, Unit: output.Unit, Currency: output.Currency, Scale: output.Scale,
		SignPolicy: "formula_defined", DimensionalIdentity: "consolidated",
		PeriodType: periodType(start, end), FiscalStart: fiscalStart, FiscalEnd: end,
		FilingDate:          receipt.SourceAsOf,
		AccountingPerimeter: authority.AccountingPerimeterSignature,
		AccountingInputs:    accountingInputs, OutputClass: authority.OutputClass,
		ProductLabel:        authority.ProductLabel,
		PairRankingEligible: authority.PairRankingEligible,
		DefinitionID:        receipt.OperationID + "/" + receipt.FormulaVersion,
		RestatementState:    "active_amendment_chain", SupersessionState: "active",
	}
	if err := validateRegisteredComparisonAuthority(operand); err != nil {
		return contracts.MetricComparisonOperand{}, err
	}
	return operand, nil
}

func validateRegisteredComparisonAuthority(
	operand contracts.MetricComparisonOperand,
) error {
	registryByKey := map[string]AccountingMappingAuthority{}
	for _, mapping := range runtimeAccountingAuthorityRegistry.Entries {
		registryByKey[mapping.MappingKey] = mapping
	}
	expectedOutputClass := AccountingOutputAuthoritative
	expectedRanking := true
	mappings := make([]AccountingMappingAuthority, 0, len(operand.AccountingInputs))
	for _, authority := range operand.AccountingInputs {
		mapping, exists := registryByKey[authority.MappingKey]
		if !exists ||
			mapping.CompanyID != operand.CompanyID ||
			mapping.CanonicalInput != authority.CanonicalInput ||
			mapping.TaxonomyConcept != authority.TaxonomyConcept ||
			mapping.AccountingPerimeter != authority.AccountingPerimeter ||
			string(mapping.Disposition) != authority.Disposition ||
			mapping.ProductLabel != authority.ProductLabel ||
			!stringSliceContains(mapping.AuthorizedOperations, operand.CanonicalMetricID) {
			return errors.New("comparison operand does not match the approved accounting registry")
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
		expectedInputRanking := numerical && mapping.ComparableRankingEligible
		if authority.PairRankingEligible != expectedInputRanking ||
			(!numerical && !context) {
			return errors.New("comparison operand has invalid effective accounting permissions")
		}
		if context {
			expectedOutputClass = AccountingOutputContextOnly
		}
		if !expectedInputRanking {
			expectedRanking = false
		}
		mappings = append(mappings, mapping)
	}
	if operand.OutputClass != expectedOutputClass ||
		operand.PairRankingEligible != expectedRanking ||
		operand.ProductLabel != comparisonProductLabel(
			operand.CanonicalMetricID,
			mappings,
			expectedOutputClass == AccountingOutputContextOnly,
		) {
		return errors.New("comparison operand release class or product label is invalid")
	}
	return nil
}

func comparisonProductLabel(
	operationID string,
	mappings []AccountingMappingAuthority,
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
		for _, mapping := range mappings {
			if mapping.CanonicalInput == "capital_expenditure" &&
				mapping.Disposition == AccountingReviewedAlias {
				return "simple FCF using " + mapping.ProductLabel
			}
		}
		return "simple FCF"
	}
	if operationID == "financial.capex_intensity" {
		for _, mapping := range mappings {
			if mapping.CanonicalInput == "capital_expenditure" &&
				mapping.Disposition == AccountingReviewedAlias {
				return mapping.ProductLabel + " intensity"
			}
		}
	}
	return operationID
}

func reviewedOperationMappingID(
	report CompanyFinancialActivation,
	receipt contracts.CalculationReceipt,
	operationSourceConcepts []string,
) string {
	usesRevenue := false
	for _, spec := range companyOperationSpecs {
		if spec.OperationID != receipt.OperationID {
			continue
		}
		for _, canonicalMetric := range spec.Inputs {
			if canonicalMetric == "revenue" {
				usesRevenue = true
				break
			}
		}
	}
	if !usesRevenue {
		return ""
	}
	revenueConcepts := report.SourceConcepts["revenue"]
	if len(revenueConcepts) != 1 {
		return ""
	}
	if revenueConcepts[0] != "RevenueFromContractWithCustomerExcludingAssessedTax" &&
		!reviewedFinancialConceptAlias(report.CompanyID, "revenue", revenueConcepts[0]) {
		return ""
	}
	concepts := append([]string(nil), operationSourceConcepts...)
	for index, concept := range concepts {
		if concept == revenueConcepts[0] {
			concepts[index] = "RevenueFromContractWithCustomerExcludingAssessedTax"
		}
	}
	sort.Strings(concepts)
	return ReviewedRevenueAliasPolicy + ":" + strings.Join(concepts, "+")
}

func operationConcepts(
	report CompanyFinancialActivation,
	receipt contracts.CalculationReceipt,
	contextOnly bool,
) []string {
	concepts := []string{}
	for _, input := range receipt.NormalizedInputs {
		for _, spec := range companyOperationSpecs {
			if spec.OperationID != receipt.OperationID {
				continue
			}
			canonical := spec.Inputs[input.InputID]
			if contextOnly && len(report.ContextualConcepts[canonical]) > 0 {
				concepts = append(concepts, report.ContextualConcepts[canonical]...)
			} else {
				concepts = append(concepts, report.SourceConcepts[canonical]...)
			}
		}
	}
	sort.Strings(concepts)
	return uniqueStrings(concepts)
}

func parseReceiptPeriod(value string) (time.Time, time.Time) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}
	}
	start, _ := time.Parse("2006-01-02", parts[0])
	end, _ := time.Parse("2006-01-02", parts[1])
	return start, end
}

func periodType(start, end time.Time) string {
	if start.Equal(end) {
		return "instant"
	}
	return "duration"
}

func sourceHashes(refs []string) []string {
	hashes := make([]string, 0, len(refs))
	for _, ref := range refs {
		parts := strings.Split(ref, ":")
		hashes = append(hashes, parts[len(parts)-1])
	}
	sort.Strings(hashes)
	return uniqueStrings(hashes)
}

func comparisonAbstention(
	lane PublicPeerLane,
	metricID string,
	generatedAt time.Time,
	code string,
) contracts.TypedAbstention {
	return contracts.TypedAbstention{
		SchemaVersion: contracts.TypedAbstentionSchemaV1,
		AbstentionID:  "abstention-" + lane.LaneID + "-" + sanitizeMetricID(metricID),
		RequestID:     "comparison-" + lane.LaneID + "-" + sanitizeMetricID(metricID),
		RunID:         "peer-evaluation-" + lane.LaneID, Code: code,
		Message:    fmt.Sprintf("SignalForge withheld %s for %s because the required authorized operands are unavailable.", metricID, lane.LaneID),
		CompanyIDs: append([]string(nil), lane.CompanyIDs...), MetricIDs: []string{metricID},
		GeneratedAt: generatedAt,
	}
}

func sanitizeMetricID(value string) string {
	return strings.NewReplacer(".", "-", "_", "-").Replace(value)
}

func ValidatePeerEvaluationSuite(suite PeerEvaluationSuite) error {
	if suite.SchemaVersion != PeerEvaluationSuiteSchemaV1 || suite.UniverseID != UniverseID ||
		suite.AsOf.IsZero() || suite.PolicyVersion != comparability.AnnualPolicyVersionV1 ||
		len(suite.Lanes) != 5 || suite.ClaimBoundary == "" {
		return errors.New("peer evaluation suite envelope is invalid")
	}
	if suite.PromotionDecisionSHA256 != "" &&
		!validPromotionHash(suite.PromotionDecisionSHA256) {
		return errors.New("peer evaluation promotion decision hash is invalid")
	}
	containsPromotion := false
	for _, lane := range suite.Lanes {
		if lane.LaneID == "" || len(lane.CompanyIDs) != 2 ||
			len(lane.Receipts)+len(lane.Abstentions) == 0 {
			return errors.New("peer evaluation lane is invalid or over-promoted")
		}
		if lane.Promoted {
			containsPromotion = true
			if len(lane.ReasonCodes) != 0 ||
				!validPromotionHashes(lane.PromotionEvidenceSHA256) {
				return errors.New("promoted peer evaluation lacks exact evidence")
			}
		} else if len(lane.ReasonCodes) == 0 ||
			len(lane.PromotionEvidenceSHA256) != 0 {
			return errors.New("unpromoted peer evaluation requires reason codes and no promotion evidence")
		}
		for _, receipt := range lane.Receipts {
			if err := contracts.ValidateMetricComparabilityReceipt(receipt); err != nil {
				return err
			}
			for _, operand := range receipt.Operands {
				if err := validateRegisteredComparisonAuthority(operand); err != nil {
					return err
				}
			}
		}
		if !disjointPeerMetricClasses(lane.Releasable, lane.ContextOnly, lane.Withheld) {
			return errors.New("peer evaluation metric classes overlap")
		}
	}
	if containsPromotion && !validPromotionHash(suite.PromotionDecisionSHA256) {
		return errors.New("peer evaluation promotion is not bound to a human decision")
	}
	return nil
}

func disjointPeerMetricClasses(classes ...[]string) bool {
	seen := map[string]bool{}
	for _, values := range classes {
		for _, value := range values {
			if seen[value] {
				return false
			}
			seen[value] = true
		}
	}
	return true
}
