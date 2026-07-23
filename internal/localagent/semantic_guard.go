package localagent

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rvbernucci/signalforge/internal/contracts"
	"github.com/rvbernucci/signalforge/internal/roles"
)

const (
	semanticUnsupportedCausality = "unsupported_causality"
	semanticScenarioAsFact       = "scenario_as_fact"
	semanticMissingAssumption    = "scenario_missing_assumption"
	semanticAccountingAssertion  = "accounting_assertion_as_fact"
	semanticAccountingPolicy     = "accounting_policy_without_source"
	semanticTransmissionMissing  = "economics_transmission_missing"
)

var (
	assertiveCausalPattern  = regexp.MustCompile(`(?i)\b(?:caused|causes|resulted\s+(?:from|in)|because\s+of|due\s+to|led\s+to|drives|drove)\b`)
	conditionalPattern      = regexp.MustCompile(`(?i)\b(?:if|may|might|could|would|conditional|scenario|subject\s+to|assuming|under)\b`)
	scenarioPattern         = regexp.MustCompile(`(?i)\b(?:forecast|scenario|sensitivity|assum(?:e|ed|ing|ption)|would|could|may|might|project(?:ed|ion)?)\b`)
	managementPattern       = regexp.MustCompile(`(?i)\b(?:management|the\s+company|we)\s+(?:believes?|expects?|estimates?|asserts?|anticipates?|views?)\b`)
	accountingPolicyPattern = regexp.MustCompile(`(?i)\b(?:accounting\s+policy|accounting\s+estimate|non-gaap|adjusted\s+(?:earnings|ebitda|metric))\b`)
	transmissionPattern     = regexp.MustCompile(`(?i)\b(?:through|via|channel|transmi(?:t|ts|tted|ssion)|affect(?:s|ed|ing)?|increase(?:s|d)?|reduce(?:s|d)?|raise(?:s|d)?|lower(?:s|ed|ing)?|compress(?:es|ed|ing)?|expand(?:s|ed|ing)?|weaken(?:s|ed|ing)?|strengthen(?:s|ed|ing)?|in\s+turn|thereby|which\s+(?:may|could|would))\b`)
)

type semanticViolation struct {
	Code    string
	ClaimID string
	Detail  string
}

func (violation semanticViolation) Error() string {
	return fmt.Sprintf("semantic guard %s for claim %q: %s", violation.Code, violation.ClaimID, violation.Detail)
}

// validateSpecialistSemantics rejects narrow, objective boundary violations. It never rewrites a
// claim or creates a financial conclusion; subjective quality remains the reviewers' responsibility.
func validateSpecialistSemantics(packet contracts.ContextPacket) error {
	for _, finding := range append(append([]contracts.Finding(nil), packet.Findings...), packet.Counterevidence...) {
		statement := strings.TrimSpace(finding.Statement)
		if (packet.SpecialistRole == roles.MarketBehavior || packet.SpecialistRole == roles.EconomicsTransmission) &&
			assertiveCausalPattern.MatchString(statement) && !conditionalPattern.MatchString(statement) {
			return semanticViolation{
				Code: semanticUnsupportedCausality, ClaimID: finding.ClaimID,
				Detail: "assertive causal language requires an explicit conditional boundary",
			}
		}
		if finding.ClaimType == contracts.ClaimFact && scenarioPattern.MatchString(statement) {
			return semanticViolation{
				Code: semanticScenarioAsFact, ClaimID: finding.ClaimID,
				Detail: "scenario or forecast language cannot be released as an observed fact",
			}
		}
		if packet.SpecialistRole == roles.EconomicsTransmission && scenarioPattern.MatchString(statement) &&
			(finding.ClaimType == contracts.ClaimInference || finding.ClaimType == contracts.ClaimHypothesis) &&
			len(finding.AssumptionRefs) == 0 {
			return semanticViolation{
				Code: semanticMissingAssumption, ClaimID: finding.ClaimID,
				Detail: "economic scenario language requires an authorized assumption reference",
			}
		}
		if packet.SpecialistRole == roles.AccountingReporting && finding.ClaimType == contracts.ClaimFact {
			if managementPattern.MatchString(statement) {
				return semanticViolation{
					Code: semanticAccountingAssertion, ClaimID: finding.ClaimID,
					Detail: "management assertion cannot be released as a reported accounting fact",
				}
			}
			if accountingPolicyPattern.MatchString(statement) && finding.Origin != contracts.FindingOriginSourceExtraction {
				return semanticViolation{
					Code: semanticAccountingPolicy, ClaimID: finding.ClaimID,
					Detail: "accounting policy or estimate requires explicit source-extraction provenance",
				}
			}
		}
		if packet.SpecialistRole == roles.EconomicsTransmission &&
			(finding.ClaimType == contracts.ClaimInference || finding.ClaimType == contracts.ClaimHypothesis) &&
			!transmissionPattern.MatchString(statement) {
			return semanticViolation{
				Code: semanticTransmissionMissing, ClaimID: finding.ClaimID,
				Detail: "economic conclusion requires an explicit variable-to-company transmission mechanism",
			}
		}
	}
	return nil
}
