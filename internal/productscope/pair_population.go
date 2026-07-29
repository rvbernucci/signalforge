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

	"github.com/rvbernucci/signalforge/internal/comparability"
	"github.com/rvbernucci/signalforge/internal/contracts"
)

const Technology20PairPopulationSchemaV1 = "signalforge/technology20-pair-population/v1"

type PairPopulationReportReference struct {
	CompanyID    string `json:"company_id"`
	ReportSHA256 string `json:"report_sha256"`
}

type Technology20PairPopulation struct {
	SchemaVersion    string                          `json:"schema_version"`
	UniverseID       string                          `json:"universe_id"`
	AsOf             time.Time                       `json:"as_of"`
	GeneratedAt      time.Time                       `json:"generated_at"`
	PolicyVersion    string                          `json:"policy_version"`
	CompanyReports   []PairPopulationReportReference `json:"company_reports"`
	Pairs            []PeerEvaluationResult          `json:"pairs"`
	ClaimBoundary    string                          `json:"claim_boundary"`
	PopulationSHA256 string                          `json:"population_sha256"`
}

func BuildTechnology20PairPopulation(
	catalog PublicCatalog,
	reports map[string]CompanyFinancialActivation,
	generatedAt time.Time,
) (Technology20PairPopulation, error) {
	if err := ValidatePublicCatalog(catalog); err != nil {
		return Technology20PairPopulation{}, err
	}
	if generatedAt.Before(catalog.AsOf) {
		return Technology20PairPopulation{}, errors.New("pair population cannot precede catalog authority")
	}
	companyByID := map[string]PublicCompany{}
	receiptByCompany := map[string]map[string]contracts.CalculationReceipt{}
	contextByCompany := map[string]map[string]contracts.CalculationReceipt{}
	population := Technology20PairPopulation{
		SchemaVersion: Technology20PairPopulationSchemaV1,
		UniverseID:    UniverseID, AsOf: catalog.AsOf, GeneratedAt: generatedAt.UTC(),
		PolicyVersion: comparability.AnnualPolicyVersionV1,
		ClaimBoundary: "This mechanically complete 190-pair fabric evaluates metric-level boundaries only. Pair existence does not establish an economic peer relationship, authorize an agent narrative, or permit a winner, score, rank, or relative conclusion for context-only or withheld metrics.",
	}
	for _, company := range catalog.Companies {
		report, exists := reports[company.CompanyID]
		if !exists {
			return Technology20PairPopulation{}, fmt.Errorf(
				"pair population is missing report for %s",
				company.CompanyID,
			)
		}
		if err := ValidateCompanyFinancialActivation(report); err != nil {
			return Technology20PairPopulation{}, err
		}
		companyByID[company.CompanyID] = company
		receiptByCompany[company.CompanyID] = map[string]contracts.CalculationReceipt{}
		contextByCompany[company.CompanyID] = map[string]contracts.CalculationReceipt{}
		for _, receipt := range report.Receipts {
			receiptByCompany[company.CompanyID][receipt.OperationID] = receipt
		}
		for _, receipt := range report.ContextualReceipts {
			contextByCompany[company.CompanyID][receipt.OperationID] = receipt
		}
		population.CompanyReports = append(
			population.CompanyReports,
			PairPopulationReportReference{
				CompanyID: company.CompanyID, ReportSHA256: report.ReportSHA256,
			},
		)
	}
	companies := append([]PublicCompany(nil), catalog.Companies...)
	sort.Slice(companies, func(i, j int) bool {
		return companies[i].CompanyID < companies[j].CompanyID
	})
	metrics := companyOperationIDs()
	for leftIndex := 0; leftIndex < len(companies); leftIndex++ {
		for rightIndex := leftIndex + 1; rightIndex < len(companies); rightIndex++ {
			leftCompany := companies[leftIndex]
			rightCompany := companies[rightIndex]
			lane := PublicPeerLane{
				LaneID: "pair-" + strings.ToLower(leftCompany.PrimaryTicker) +
					"-" + strings.ToLower(rightCompany.PrimaryTicker),
				CompanyIDs:       []string{leftCompany.CompanyID, rightCompany.CompanyID},
				AllowedMetricIDs: append([]string(nil), metrics...),
			}
			result := PeerEvaluationResult{
				LaneID: lane.LaneID, CompanyIDs: append([]string(nil), lane.CompanyIDs...),
				Receipts:    []contracts.MetricComparabilityReceipt{},
				Abstentions: []contracts.TypedAbstention{},
				Releasable:  []string{}, ContextOnly: []string{}, Withheld: []string{},
				ReasonCodes: []string{
					"pair_topology_is_not_peer_authority",
					"agent_narrative_requires_separate_reviewed_lane",
				},
			}
			for _, metricID := range metrics {
				left, leftFound := receiptByCompany[leftCompany.CompanyID][metricID]
				right, rightFound := receiptByCompany[rightCompany.CompanyID][metricID]
				leftContext, rightContext := false, false
				if !leftFound {
					left, leftFound = contextByCompany[leftCompany.CompanyID][metricID]
					leftContext = leftFound
				}
				if !rightFound {
					right, rightFound = contextByCompany[rightCompany.CompanyID][metricID]
					rightContext = rightFound
				}
				if !leftFound || !rightFound {
					result.Abstentions = append(
						result.Abstentions,
						comparisonAbstention(
							lane,
							metricID,
							generatedAt,
							"comparison_operand_unavailable",
						),
					)
					result.Withheld = append(result.Withheld, metricID)
					continue
				}
				leftOperand, err := financialOperand(
					leftCompany,
					reports[leftCompany.CompanyID],
					left,
					leftContext,
				)
				if err != nil {
					return Technology20PairPopulation{}, err
				}
				rightOperand, err := financialOperand(
					rightCompany,
					reports[rightCompany.CompanyID],
					right,
					rightContext,
				)
				if err != nil {
					return Technology20PairPopulation{}, err
				}
				request, err := contracts.PopulateMetricComparabilityRequestHash(
					contracts.MetricComparabilityRequest{
						SchemaVersion: contracts.ComparabilityRequestSchemaV2,
						RequestID: "comparison-" + lane.LaneID + "-" +
							sanitizeMetricID(metricID),
						RunID:  "pair-population-" + lane.LaneID,
						LaneID: lane.LaneID, AsOf: catalog.AsOf,
						ReviewerPolicyVersion: comparability.AnnualPolicyVersionV1,
						Operands: []contracts.MetricComparisonOperand{
							leftOperand, rightOperand,
						},
					},
				)
				if err != nil {
					return Technology20PairPopulation{}, err
				}
				receipt, err := comparability.Evaluate(
					request,
					generatedAt,
					comparability.AnnualDurationPolicy(),
				)
				if err != nil {
					return Technology20PairPopulation{}, err
				}
				result.Receipts = append(result.Receipts, receipt)
				switch {
				case comparability.IsReleasable(receipt.Disposition):
					result.Releasable = append(result.Releasable, metricID)
				case receipt.Disposition == contracts.ComparisonContextOnly:
					result.ContextOnly = append(result.ContextOnly, metricID)
				default:
					result.Withheld = append(result.Withheld, metricID)
				}
			}
			sort.Strings(result.Releasable)
			sort.Strings(result.ContextOnly)
			sort.Strings(result.Withheld)
			sort.Slice(result.Receipts, func(i, j int) bool {
				return result.Receipts[i].Operands[0].CanonicalMetricID <
					result.Receipts[j].Operands[0].CanonicalMetricID
			})
			sort.Slice(result.Abstentions, func(i, j int) bool {
				return result.Abstentions[i].MetricIDs[0] <
					result.Abstentions[j].MetricIDs[0]
			})
			envelopeHash, err := peerEvaluationResultHash(result)
			if err != nil {
				return Technology20PairPopulation{}, err
			}
			result.EnvelopeSHA256 = envelopeHash
			population.Pairs = append(population.Pairs, result)
		}
	}
	sort.Slice(population.CompanyReports, func(i, j int) bool {
		return population.CompanyReports[i].CompanyID <
			population.CompanyReports[j].CompanyID
	})
	sort.Slice(population.Pairs, func(i, j int) bool {
		return population.Pairs[i].LaneID < population.Pairs[j].LaneID
	})
	hash, err := technology20PairPopulationHash(population)
	if err != nil {
		return Technology20PairPopulation{}, err
	}
	population.PopulationSHA256 = hash
	return population, ValidateTechnology20PairPopulation(population)
}

func ValidateTechnology20PairPopulation(population Technology20PairPopulation) error {
	if population.SchemaVersion != Technology20PairPopulationSchemaV1 ||
		population.UniverseID != UniverseID ||
		population.AsOf.IsZero() ||
		population.GeneratedAt.Before(population.AsOf) ||
		population.PolicyVersion != comparability.AnnualPolicyVersionV1 ||
		len(population.CompanyReports) != len(Companies()) ||
		len(population.Pairs) != len(Companies())*(len(Companies())-1)/2 ||
		population.ClaimBoundary == "" ||
		population.PopulationSHA256 == "" {
		return errors.New("Technology 20 pair population envelope is invalid")
	}
	reportCompanies := map[string]bool{}
	for _, report := range population.CompanyReports {
		if !technologyCompany(report.CompanyID) ||
			reportCompanies[report.CompanyID] ||
			!validSHA256(report.ReportSHA256) {
			return errors.New("Technology 20 pair population report reference is invalid")
		}
		reportCompanies[report.CompanyID] = true
	}
	pairs := map[string]bool{}
	metrics := companyOperationIDs()
	for _, pair := range population.Pairs {
		if pair.LaneID == "" || len(pair.CompanyIDs) != 2 ||
			pair.CompanyIDs[0] >= pair.CompanyIDs[1] ||
			!technologyCompany(pair.CompanyIDs[0]) ||
			!technologyCompany(pair.CompanyIDs[1]) ||
			pair.Promoted ||
			len(pair.ReasonCodes) == 0 ||
			len(pair.Receipts)+len(pair.Abstentions) != len(metrics) ||
			!disjointPeerMetricClasses(pair.Releasable, pair.ContextOnly, pair.Withheld) {
			return errors.New("Technology 20 pair envelope is invalid or over-promoted")
		}
		key := pair.CompanyIDs[0] + "|" + pair.CompanyIDs[1]
		if pairs[key] {
			return errors.New("Technology 20 pair population contains a duplicate pair")
		}
		pairs[key] = true
		seenMetrics := map[string]bool{}
		for _, receipt := range pair.Receipts {
			if err := contracts.ValidateMetricComparabilityReceipt(receipt); err != nil {
				return err
			}
			for _, operand := range receipt.Operands {
				if err := validateRegisteredComparisonAuthority(operand); err != nil {
					return err
				}
			}
			metricID := receipt.Operands[0].CanonicalMetricID
			if seenMetrics[metricID] {
				return errors.New("Technology 20 pair envelope duplicates a metric")
			}
			seenMetrics[metricID] = true
			switch {
			case comparability.IsReleasable(receipt.Disposition):
				if !stringSliceContains(pair.Releasable, metricID) {
					return errors.New("releasable receipt is missing from its release class")
				}
			case receipt.Disposition == contracts.ComparisonContextOnly:
				if !stringSliceContains(pair.ContextOnly, metricID) {
					return errors.New("context-only receipt escaped its non-ranking class")
				}
			case receipt.Disposition == contracts.ComparisonNotComparable:
				if !stringSliceContains(pair.Withheld, metricID) {
					return errors.New("not-comparable receipt escaped its withheld class")
				}
			default:
				return errors.New("Technology 20 pair receipt has an unsupported disposition")
			}
		}
		for _, abstention := range pair.Abstentions {
			if err := contracts.ValidateTypedAbstention(abstention); err != nil {
				return err
			}
			metricID := abstention.MetricIDs[0]
			if seenMetrics[metricID] || !stringSliceContains(pair.Withheld, metricID) {
				return errors.New("Technology 20 pair abstention classification is invalid")
			}
			seenMetrics[metricID] = true
		}
		if len(seenMetrics) != len(metrics) {
			return errors.New("Technology 20 pair envelope has incomplete metric coverage")
		}
		expected, err := peerEvaluationResultHash(pair)
		if err != nil {
			return err
		}
		if expected != pair.EnvelopeSHA256 {
			return errors.New("Technology 20 pair envelope hash mismatch")
		}
	}
	expected, err := technology20PairPopulationHash(population)
	if err != nil {
		return err
	}
	if expected != population.PopulationSHA256 {
		return errors.New("Technology 20 pair population hash mismatch")
	}
	return nil
}

func companyOperationIDs() []string {
	result := make([]string, 0, len(companyOperationSpecs))
	for _, spec := range companyOperationSpecs {
		result = append(result, spec.OperationID)
	}
	sort.Strings(result)
	return result
}

func peerEvaluationResultHash(result PeerEvaluationResult) (string, error) {
	result.EnvelopeSHA256 = ""
	payload, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func technology20PairPopulationHash(population Technology20PairPopulation) (string, error) {
	population.PopulationSHA256 = ""
	payload, err := json.Marshal(population)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
