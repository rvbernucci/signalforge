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

	"github.com/rvbernucci/signalforge/internal/contracts"
)

const (
	Technology20EvaluationSummarySchemaV2 = "signalforge/technology20-evaluation-summary/v2"
	Technology20PromotionManifestSchemaV1 = "signalforge/technology20-promotion-manifest/v1"
	ExactReleaseDecisionSchemaV1          = "signalforge/exact-release-decision/v1"

	StandaloneDevelopmentEvidence = "standalone_development"
	StandaloneSealedEvidence      = "standalone_sealed"
	PeerDevelopmentEvidence       = "peer_development"
	PeerSealedEvidence            = "peer_sealed"
)

var requiredPromotionArtifacts = []string{
	StandaloneDevelopmentEvidence,
	StandaloneSealedEvidence,
	PeerDevelopmentEvidence,
	PeerSealedEvidence,
}

type EvaluationIdentity struct {
	SchemaVersion         string            `json:"schema_version"`
	UniverseID            string            `json:"universe_id"`
	Split                 string            `json:"split"`
	SuiteSHA256           string            `json:"suite_sha256"`
	SourceCommit          string            `json:"source_commit"`
	ModelID               string            `json:"model_id"`
	SpecialistProvider    string            `json:"specialist_provider,omitempty"`
	SpecialistModel       string            `json:"specialist_model,omitempty"`
	LoopbackCoreInference bool              `json:"loopback_core_inference"`
	ShardEvaluationSHA256 map[string]string `json:"shard_evaluation_sha256"`
}

type EvaluationGroupSummary struct {
	Cases            int                `json:"cases"`
	GatePassRates    map[string]float64 `json:"gate_pass_rates"`
	RuntimePassRate  float64            `json:"runtime_pass_rate"`
	ContractPassRate float64            `json:"contract_pass_rate"`
}

type PacketAuthorityIntegrity struct {
	PacketsObserved int            `json:"packets_observed"`
	PacketsPassed   int            `json:"packets_passed"`
	PacketsFailed   int            `json:"packets_failed"`
	PassRate        float64        `json:"pass_rate"`
	MissingRefs     map[string]int `json:"missing_reference_counts"`
	Failures        map[string]int `json:"packet_failures_by_company"`
}

type Technology20EvaluationSummary struct {
	SchemaVersion            string                            `json:"schema_version"`
	EvaluationKind           string                            `json:"evaluation_kind"`
	EvaluationIdentity       *EvaluationIdentity               `json:"evaluation_identity"`
	ExpectedCases            int                               `json:"expected_cases"`
	CompletedCases           int                               `json:"completed_cases"`
	PopulationComplete       bool                              `json:"population_complete"`
	GateCounts               map[string]int                    `json:"gate_counts"`
	RuntimePassRate          float64                           `json:"runtime_pass_rate"`
	ContractPassRate         float64                           `json:"contract_pass_rate"`
	FailureCodes             map[string]int                    `json:"failure_codes"`
	FailedGateCounts         map[string]int                    `json:"failed_gate_counts"`
	PacketAuthorityIntegrity PacketAuthorityIntegrity          `json:"packet_authority_integrity"`
	ByCompany                map[string]EvaluationGroupSummary `json:"by_company"`
	ByLane                   map[string]EvaluationGroupSummary `json:"by_lane"`
	InputCaseSHA256          map[string]string                 `json:"input_case_sha256"`
	ReleaseDisposition       string                            `json:"release_disposition"`
	ClaimBoundary            string                            `json:"claim_boundary"`
	SummarySHA256            string                            `json:"summary_sha256"`
}

type ExactReleaseDecision struct {
	SchemaVersion  string            `json:"schema_version"`
	UniverseID     string            `json:"universe_id"`
	SourceCommit   string            `json:"source_commit"`
	ReviewerName   string            `json:"reviewer_name"`
	ReviewerRole   string            `json:"reviewer_role"`
	Disposition    string            `json:"disposition"`
	Conditions     []string          `json:"conditions"`
	EvidenceSHA256 map[string]string `json:"evidence_sha256"`
	DecidedAt      time.Time         `json:"decided_at"`
	RecordLocator  string            `json:"record_locator"`
	DecisionSHA256 string            `json:"decision_sha256"`
}

type PromotionOutcome struct {
	SubjectID  string   `json:"subject_id"`
	Promoted   bool     `json:"promoted"`
	Evidence   []string `json:"evidence_sha256,omitempty"`
	ReasonCode string   `json:"reason_code,omitempty"`
}

type Technology20PromotionManifest struct {
	SchemaVersion            string             `json:"schema_version"`
	UniverseID               string             `json:"universe_id"`
	SourceCommit             string             `json:"source_commit"`
	GeneratedAt              time.Time          `json:"generated_at"`
	EvidenceSHA256           map[string]string  `json:"evidence_sha256"`
	AccountingDecisionSHA256 string             `json:"accounting_decision_sha256"`
	HumanDecisionSHA256      string             `json:"human_decision_sha256"`
	Companies                []PromotionOutcome `json:"companies"`
	PeerLanes                []PromotionOutcome `json:"peer_lanes"`
	PromotedCatalogSHA256    string             `json:"promoted_catalog_sha256,omitempty"`
	PromotedPeersSHA256      string             `json:"promoted_peers_sha256,omitempty"`
	ClaimBoundary            string             `json:"claim_boundary"`
	ManifestSHA256           string             `json:"manifest_sha256"`
}

type PromotionInput struct {
	Catalog               PublicCatalog
	Peers                 PeerEvaluationSuite
	StandaloneDevelopment Technology20EvaluationSummary
	StandaloneSealed      Technology20EvaluationSummary
	PeerDevelopment       Technology20EvaluationSummary
	PeerSealed            Technology20EvaluationSummary
	EvidenceSHA256        map[string]string
	Decision              ExactReleaseDecision
	GeneratedAt           time.Time
}

func PopulateExactReleaseDecisionHash(record ExactReleaseDecision) (ExactReleaseDecision, error) {
	record.DecisionSHA256 = ""
	digest, err := promotionStructHash(record)
	if err != nil {
		return ExactReleaseDecision{}, err
	}
	record.DecisionSHA256 = digest
	return record, ValidateExactReleaseDecision(record)
}

func ValidateExactReleaseDecision(record ExactReleaseDecision) error {
	if record.SchemaVersion != ExactReleaseDecisionSchemaV1 ||
		record.UniverseID != UniverseID ||
		!validCommit(record.SourceCommit) ||
		strings.TrimSpace(record.ReviewerName) == "" ||
		strings.TrimSpace(record.ReviewerRole) == "" ||
		record.Disposition != "accepted" ||
		len(record.Conditions) == 0 ||
		record.DecidedAt.IsZero() ||
		strings.TrimSpace(record.RecordLocator) == "" ||
		!validPromotionHash(record.DecisionSHA256) {
		return errors.New("exact release decision envelope is invalid")
	}
	if err := validatePromotionArtifacts(record.EvidenceSHA256); err != nil {
		return err
	}
	expected, err := promotionHashWithout(record, func(value *ExactReleaseDecision) {
		value.DecisionSHA256 = ""
	})
	if err != nil {
		return err
	}
	if expected != record.DecisionSHA256 {
		return errors.New("exact release decision hash mismatch")
	}
	return nil
}

func PromoteTechnology20(input PromotionInput) (
	PublicCatalog,
	PeerEvaluationSuite,
	Technology20PromotionManifest,
	error,
) {
	if err := ValidatePublicCatalog(input.Catalog); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := ValidatePeerEvaluationSuite(input.Peers); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := ValidateReleaseAlignment(input.Catalog, input.Peers); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := validatePromotionArtifacts(input.EvidenceSHA256); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := ValidateExactReleaseDecision(input.Decision); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if input.Decision.SourceCommit == "" || input.GeneratedAt.IsZero() ||
		!equalHashMaps(input.Decision.EvidenceSHA256, input.EvidenceSHA256) {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{},
			errors.New("promotion is not bound to the exact human-reviewed evidence")
	}
	if err := validatePromotionSummary(
		input.StandaloneDevelopment, "standalone", StandaloneDevelopmentSplit,
		80, input.Decision.SourceCommit,
	); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := validatePromotionSummary(
		input.StandaloneSealed, "standalone", StandaloneSealedSplit,
		40, input.Decision.SourceCommit,
	); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := validatePromotionSummary(
		input.PeerDevelopment, "peer", "development",
		40, input.Decision.SourceCommit,
	); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := validatePromotionSummary(
		input.PeerSealed, "peer", "sealed_holdout",
		20, input.Decision.SourceCommit,
	); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}

	accounting, err := DefaultAccountingProfessionalDecision()
	if err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	catalog := clonePromotionValue(input.Catalog)
	peers := clonePromotionValue(input.Peers)
	companyEvidence := sortedPromotionHashes(
		input.EvidenceSHA256[StandaloneDevelopmentEvidence],
		input.EvidenceSHA256[StandaloneSealedEvidence],
		accounting.DecisionSHA256,
		input.Decision.DecisionSHA256,
	)
	laneEvidence := sortedPromotionHashes(
		input.EvidenceSHA256[PeerDevelopmentEvidence],
		input.EvidenceSHA256[PeerSealedEvidence],
		accounting.DecisionSHA256,
		input.Decision.DecisionSHA256,
	)
	manifest := Technology20PromotionManifest{
		SchemaVersion:            Technology20PromotionManifestSchemaV1,
		UniverseID:               UniverseID,
		SourceCommit:             input.Decision.SourceCommit,
		GeneratedAt:              input.GeneratedAt.UTC(),
		EvidenceSHA256:           cloneStringMap(input.EvidenceSHA256),
		AccountingDecisionSHA256: accounting.DecisionSHA256,
		HumanDecisionSHA256:      input.Decision.DecisionSHA256,
		ClaimBoundary:            "Promotion authorizes only companies and peer lanes whose exact development and sealed contract populations passed. It does not convert model output into audited fact, investment advice, or unrestricted comparison authority.",
	}
	catalog.PromotionDecisionSHA256 = input.Decision.DecisionSHA256
	peers.PromotionDecisionSHA256 = input.Decision.DecisionSHA256

	promotedCompanies := map[string]bool{}
	for index := range catalog.Companies {
		company := &catalog.Companies[index]
		passed := groupPassed(
			input.StandaloneDevelopment.ByCompany[company.CompanyID], 4,
		) && groupPassed(input.StandaloneSealed.ByCompany[company.CompanyID], 2)
		outcome := PromotionOutcome{SubjectID: company.CompanyID, Promoted: passed}
		if passed {
			company.ActivationState = contracts.ActivationResearchReady
			company.ResearchEnabled = true
			company.ReasonCodes = nil
			company.PromotionEvidenceSHA256 = append([]string(nil), companyEvidence...)
			outcome.Evidence = append([]string(nil), companyEvidence...)
			promotedCompanies[company.CompanyID] = true
		} else {
			company.ResearchEnabled = false
			company.ReasonCodes = []string{"standalone_release_contract_not_satisfied"}
			company.PromotionEvidenceSHA256 = nil
			outcome.ReasonCode = company.ReasonCodes[0]
		}
		manifest.Companies = append(manifest.Companies, outcome)
	}

	peerByID := map[string]*PeerEvaluationResult{}
	for index := range peers.Lanes {
		peerByID[peers.Lanes[index].LaneID] = &peers.Lanes[index]
	}
	for index := range catalog.PeerLanes {
		lane := &catalog.PeerLanes[index]
		evaluation := peerByID[lane.LaneID]
		passed := evaluation != nil &&
			promotedCompanies[lane.CompanyIDs[0]] &&
			promotedCompanies[lane.CompanyIDs[1]] &&
			groupPassed(input.PeerDevelopment.ByLane[lane.LaneID], 8) &&
			groupPassed(input.PeerSealed.ByLane[lane.LaneID], 4)
		outcome := PromotionOutcome{SubjectID: lane.LaneID, Promoted: passed}
		if passed {
			lane.Enabled = true
			lane.ReasonCodes = nil
			lane.PromotionEvidenceSHA256 = append([]string(nil), laneEvidence...)
			evaluation.Promoted = true
			evaluation.ReasonCodes = nil
			evaluation.PromotionEvidenceSHA256 = append([]string(nil), laneEvidence...)
			outcome.Evidence = append([]string(nil), laneEvidence...)
		} else {
			lane.Enabled = false
			lane.ReasonCodes = []string{"peer_release_contract_not_satisfied"}
			lane.PromotionEvidenceSHA256 = nil
			if evaluation != nil {
				evaluation.Promoted = false
				evaluation.ReasonCodes = []string{"peer_release_contract_not_satisfied"}
				evaluation.PromotionEvidenceSHA256 = nil
			}
			outcome.ReasonCode = lane.ReasonCodes[0]
		}
		manifest.PeerLanes = append(manifest.PeerLanes, outcome)
	}
	sort.Slice(manifest.Companies, func(i, j int) bool {
		return manifest.Companies[i].SubjectID < manifest.Companies[j].SubjectID
	})
	sort.Slice(manifest.PeerLanes, func(i, j int) bool {
		return manifest.PeerLanes[i].SubjectID < manifest.PeerLanes[j].SubjectID
	})
	if err := ValidatePublicCatalog(catalog); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := ValidatePeerEvaluationSuite(peers); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	if err := ValidateReleaseAlignment(catalog, peers); err != nil {
		return PublicCatalog{}, PeerEvaluationSuite{}, Technology20PromotionManifest{}, err
	}
	return catalog, peers, manifest, nil
}

func PopulatePromotionManifestOutputHashes(
	manifest Technology20PromotionManifest,
	catalogSHA256 string,
	peersSHA256 string,
) (Technology20PromotionManifest, error) {
	if !validPromotionHash(catalogSHA256) || !validPromotionHash(peersSHA256) {
		return Technology20PromotionManifest{}, errors.New("promotion outputs require exact SHA-256 digests")
	}
	manifest.PromotedCatalogSHA256 = catalogSHA256
	manifest.PromotedPeersSHA256 = peersSHA256
	manifest.ManifestSHA256 = ""
	digest, err := promotionStructHash(manifest)
	if err != nil {
		return Technology20PromotionManifest{}, err
	}
	manifest.ManifestSHA256 = digest
	return manifest, ValidateTechnology20PromotionManifest(manifest)
}

func ValidateTechnology20PromotionManifest(manifest Technology20PromotionManifest) error {
	if manifest.SchemaVersion != Technology20PromotionManifestSchemaV1 ||
		manifest.UniverseID != UniverseID ||
		!validCommit(manifest.SourceCommit) ||
		manifest.GeneratedAt.IsZero() ||
		!validPromotionHash(manifest.AccountingDecisionSHA256) ||
		!validPromotionHash(manifest.HumanDecisionSHA256) ||
		!validPromotionHash(manifest.PromotedCatalogSHA256) ||
		!validPromotionHash(manifest.PromotedPeersSHA256) ||
		!validPromotionHash(manifest.ManifestSHA256) ||
		len(manifest.Companies) != 20 ||
		len(manifest.PeerLanes) != 5 ||
		strings.TrimSpace(manifest.ClaimBoundary) == "" {
		return errors.New("Technology 20 promotion manifest envelope is invalid")
	}
	if err := validatePromotionArtifacts(manifest.EvidenceSHA256); err != nil {
		return err
	}
	for _, group := range [][]PromotionOutcome{manifest.Companies, manifest.PeerLanes} {
		seen := map[string]bool{}
		for _, outcome := range group {
			if outcome.SubjectID == "" || seen[outcome.SubjectID] {
				return errors.New("promotion manifest contains duplicate or empty subject")
			}
			seen[outcome.SubjectID] = true
			if outcome.Promoted {
				if !validPromotionHashes(outcome.Evidence) || outcome.ReasonCode != "" {
					return errors.New("promoted subject lacks exact evidence")
				}
			} else if outcome.ReasonCode == "" || len(outcome.Evidence) != 0 {
				return errors.New("withheld subject lacks a reason code")
			}
		}
	}
	expected, err := promotionHashWithout(manifest, func(value *Technology20PromotionManifest) {
		value.ManifestSHA256 = ""
	})
	if err != nil {
		return err
	}
	if expected != manifest.ManifestSHA256 {
		return errors.New("Technology 20 promotion manifest hash mismatch")
	}
	return nil
}

func ValidateReleaseAlignment(catalog PublicCatalog, peers PeerEvaluationSuite) error {
	if catalog.PromotionDecisionSHA256 != peers.PromotionDecisionSHA256 {
		return errors.New("catalog and peer authority use different promotion decisions")
	}
	peerByID := map[string]PeerEvaluationResult{}
	for _, lane := range peers.Lanes {
		peerByID[lane.LaneID] = lane
	}
	for _, lane := range catalog.PeerLanes {
		evaluation, exists := peerByID[lane.LaneID]
		if !exists || lane.Enabled != evaluation.Promoted {
			return fmt.Errorf("peer lane %q promotion is not aligned", lane.LaneID)
		}
		if lane.Enabled && !sameStrings(lane.PromotionEvidenceSHA256, evaluation.PromotionEvidenceSHA256) {
			return fmt.Errorf("peer lane %q promotion evidence is not aligned", lane.LaneID)
		}
	}
	return nil
}

func validatePromotionSummary(
	summary Technology20EvaluationSummary,
	kind string,
	split string,
	expectedCases int,
	sourceCommit string,
) error {
	if summary.SchemaVersion != Technology20EvaluationSummarySchemaV2 ||
		summary.EvaluationKind != kind ||
		summary.EvaluationIdentity == nil ||
		summary.EvaluationIdentity.UniverseID != UniverseID ||
		summary.EvaluationIdentity.Split != split ||
		summary.EvaluationIdentity.SourceCommit != sourceCommit ||
		!summary.EvaluationIdentity.LoopbackCoreInference ||
		!validPromotionHash(summary.EvaluationIdentity.SuiteSHA256) ||
		strings.TrimSpace(summary.EvaluationIdentity.ModelID) == "" ||
		len(summary.EvaluationIdentity.ShardEvaluationSHA256) == 0 ||
		!validPromotionHash(summary.SummarySHA256) ||
		summary.ExpectedCases != expectedCases ||
		summary.CompletedCases != expectedCases ||
		!summary.PopulationComplete ||
		len(summary.InputCaseSHA256) != expectedCases ||
		!validPromotionHashMap(summary.InputCaseSHA256) ||
		summary.RuntimePassRate != 1 ||
		summary.ContractPassRate != 1 ||
		len(summary.FailureCodes) != 0 ||
		len(summary.FailedGateCounts) != 0 ||
		summary.ReleaseDisposition != "evaluation_only_not_promoted" ||
		strings.TrimSpace(summary.ClaimBoundary) == "" {
		return fmt.Errorf("%s/%s evaluation summary is not release-bound", kind, split)
	}
	if !validPromotionHashMap(summary.EvaluationIdentity.ShardEvaluationSHA256) ||
		len(summary.GateCounts) == 0 ||
		summary.PacketAuthorityIntegrity.PacketsFailed != 0 ||
		summary.PacketAuthorityIntegrity.PassRate != 1 ||
		len(summary.PacketAuthorityIntegrity.MissingRefs) != 0 ||
		len(summary.PacketAuthorityIntegrity.Failures) != 0 {
		return fmt.Errorf("%s/%s packet authority integrity failed", kind, split)
	}
	for _, count := range summary.GateCounts {
		if count != expectedCases {
			return fmt.Errorf("%s/%s contains a failed release gate", kind, split)
		}
	}
	return nil
}

func groupPassed(group EvaluationGroupSummary, expectedCases int) bool {
	if group.Cases != expectedCases || group.RuntimePassRate != 1 || group.ContractPassRate != 1 {
		return false
	}
	for _, value := range group.GatePassRates {
		if value != 1 {
			return false
		}
	}
	return len(group.GatePassRates) > 0
}

func validatePromotionArtifacts(values map[string]string) error {
	if len(values) != len(requiredPromotionArtifacts) {
		return errors.New("promotion evidence does not contain the exact required artifacts")
	}
	for _, key := range requiredPromotionArtifacts {
		if !validPromotionHash(values[key]) {
			return fmt.Errorf("promotion evidence %q is missing or invalid", key)
		}
	}
	return nil
}

func validPromotionHashMap(values map[string]string) bool {
	if len(values) == 0 {
		return false
	}
	for key, value := range values {
		if strings.TrimSpace(key) == "" || !validPromotionHash(value) {
			return false
		}
	}
	return true
}

func validPromotionHashes(values []string) bool {
	if len(values) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !validPromotionHash(value) || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func validPromotionHash(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validCommit(value string) bool {
	return len(value) == 40 && validPromotionHex(value)
}

func validPromotionHex(value string) bool {
	if value == "" || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sortedPromotionHashes(values ...string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func promotionStructHash(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func promotionHashWithout[T any](value T, clear func(*T)) (string, error) {
	clear(&value)
	return promotionStructHash(value)
}

func clonePromotionValue[T any](value T) T {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var result T
	if err := json.Unmarshal(payload, &result); err != nil {
		panic(err)
	}
	return result
}

func cloneStringMap(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func equalHashMaps(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
