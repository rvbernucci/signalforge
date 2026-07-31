package productscope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestPromotionRequiresExactPopulationsAndProducesAlignedRelease(t *testing.T) {
	catalog := loadPublicCatalogFixture(t)
	peers := loadPeerEvaluationFixture(t)
	sourceCommit := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	evidence := promotionEvidenceForTest()
	decision, err := PopulateExactReleaseDecisionHash(ExactReleaseDecision{
		SchemaVersion:  ExactReleaseDecisionSchemaV1,
		UniverseID:     UniverseID,
		SourceCommit:   sourceCommit,
		ReviewerName:   "Release owner",
		ReviewerRole:   "Exact-candidate reviewer",
		Disposition:    "accepted",
		Conditions:     []string{"release only the hash-bound passing cohort"},
		EvidenceSHA256: cloneStringMap(evidence),
		DecidedAt:      time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC),
		RecordLocator:  "private/release-decision.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	standaloneDevelopment := passingSummaryForTest(
		catalog, peers, "standalone", StandaloneDevelopmentSplit, 80, 4, sourceCommit,
	)
	standaloneSealed := passingSummaryForTest(
		catalog, peers, "standalone", StandaloneSealedSplit, 40, 2, sourceCommit,
	)
	peerDevelopment := passingSummaryForTest(
		catalog, peers, "peer", "development", 40, 8, sourceCommit,
	)
	peerSealed := passingSummaryForTest(
		catalog, peers, "peer", "sealed_holdout", 20, 4, sourceCommit,
	)

	promotedCatalog, promotedPeers, manifest, err := PromoteTechnology20(PromotionInput{
		Catalog: catalog, Peers: peers,
		StandaloneDevelopment: standaloneDevelopment,
		StandaloneSealed:      standaloneSealed,
		PeerDevelopment:       peerDevelopment,
		PeerSealed:            peerSealed,
		EvidenceSHA256:        evidence,
		Decision:              decision,
		GeneratedAt:           time.Date(2026, 7, 30, 12, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, company := range promotedCatalog.Companies {
		if !company.ResearchEnabled || len(company.PromotionEvidenceSHA256) != 4 {
			t.Fatalf("company was not promoted with exact evidence: %+v", company)
		}
	}
	for _, lane := range promotedCatalog.PeerLanes {
		if !lane.Enabled || len(lane.PromotionEvidenceSHA256) != 4 {
			t.Fatalf("catalog peer lane was not promoted: %+v", lane)
		}
	}
	for _, lane := range promotedPeers.Lanes {
		if !lane.Promoted || len(lane.PromotionEvidenceSHA256) != 4 {
			t.Fatalf("peer authority was not promoted: %+v", lane)
		}
	}
	if err := ValidateReleaseAlignment(promotedCatalog, promotedPeers); err != nil {
		t.Fatal(err)
	}
	catalogPayload, err := promotionJSON(promotedCatalog)
	if err != nil {
		t.Fatal(err)
	}
	peerPayload, err := promotionJSON(promotedPeers)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err = PopulatePromotionManifestOutputHashes(
		manifest,
		promotionBytesHash(catalogPayload),
		promotionBytesHash(peerPayload),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTechnology20PromotionManifest(manifest); err != nil {
		t.Fatal(err)
	}
}

func TestPromotionWithholdsOnlyFailedCompanyAndDependentPeerLane(t *testing.T) {
	catalog := loadPublicCatalogFixture(t)
	peers := loadPeerEvaluationFixture(t)
	sourceCommit := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	evidence := promotionEvidenceForTest()
	decision, err := PopulateExactReleaseDecisionHash(ExactReleaseDecision{
		SchemaVersion: ExactReleaseDecisionSchemaV1,
		UniverseID:    UniverseID, SourceCommit: sourceCommit,
		ReviewerName: "Release owner", ReviewerRole: "Exact-candidate reviewer",
		Disposition: "accepted", Conditions: []string{"promote passing subjects only"},
		EvidenceSHA256: cloneStringMap(evidence),
		DecidedAt:      time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC),
		RecordLocator:  "private/release-decision.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	development := passingSummaryForTest(
		catalog, peers, "standalone", StandaloneDevelopmentSplit, 80, 4, sourceCommit,
	)
	sealed := passingSummaryForTest(
		catalog, peers, "standalone", StandaloneSealedSplit, 40, 2, sourceCommit,
	)
	failedCompany := catalog.PeerLanes[0].CompanyIDs[0]
	group := sealed.ByCompany[failedCompany]
	group.ContractPassRate = 0.5
	group.GatePassRates["contract_passed"] = 0.5
	sealed.ByCompany[failedCompany] = group
	peerDevelopment := passingSummaryForTest(
		catalog, peers, "peer", "development", 40, 8, sourceCommit,
	)
	peerSealed := passingSummaryForTest(
		catalog, peers, "peer", "sealed_holdout", 20, 4, sourceCommit,
	)

	promotedCatalog, promotedPeers, _, err := PromoteTechnology20(PromotionInput{
		Catalog: catalog, Peers: peers,
		StandaloneDevelopment: development, StandaloneSealed: sealed,
		PeerDevelopment: peerDevelopment, PeerSealed: peerSealed,
		EvidenceSHA256: evidence, Decision: decision,
		GeneratedAt: time.Date(2026, 7, 30, 12, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, company := range promotedCatalog.Companies {
		if company.CompanyID == failedCompany && company.ResearchEnabled {
			t.Fatal("failed company escaped the promotion gate")
		}
	}
	for _, lane := range promotedCatalog.PeerLanes {
		if lane.CompanyIDs[0] == failedCompany || lane.CompanyIDs[1] == failedCompany {
			if lane.Enabled {
				t.Fatal("peer lane depending on a failed company was promoted")
			}
		}
	}
	if err := ValidateReleaseAlignment(promotedCatalog, promotedPeers); err != nil {
		t.Fatal(err)
	}
}

func TestPromotionRejectsMismatchedHumanEvidence(t *testing.T) {
	catalog := loadPublicCatalogFixture(t)
	peers := loadPeerEvaluationFixture(t)
	sourceCommit := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	evidence := promotionEvidenceForTest()
	decisionEvidence := cloneStringMap(evidence)
	decisionEvidence[PeerSealedEvidence] = "f" + evidence[PeerSealedEvidence][1:]
	decision, err := PopulateExactReleaseDecisionHash(ExactReleaseDecision{
		SchemaVersion: ExactReleaseDecisionSchemaV1,
		UniverseID:    UniverseID, SourceCommit: sourceCommit,
		ReviewerName: "Release owner", ReviewerRole: "Exact-candidate reviewer",
		Disposition: "accepted", Conditions: []string{"exact evidence only"},
		EvidenceSHA256: decisionEvidence,
		DecidedAt:      time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC),
		RecordLocator:  "private/release-decision.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = PromoteTechnology20(PromotionInput{
		Catalog: catalog, Peers: peers,
		StandaloneDevelopment: passingSummaryForTest(catalog, peers, "standalone", StandaloneDevelopmentSplit, 80, 4, sourceCommit),
		StandaloneSealed:      passingSummaryForTest(catalog, peers, "standalone", StandaloneSealedSplit, 40, 2, sourceCommit),
		PeerDevelopment:       passingSummaryForTest(catalog, peers, "peer", "development", 40, 8, sourceCommit),
		PeerSealed:            passingSummaryForTest(catalog, peers, "peer", "sealed_holdout", 20, 4, sourceCommit),
		EvidenceSHA256:        evidence, Decision: decision,
		GeneratedAt: time.Date(2026, 7, 30, 12, 30, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("mismatched human evidence was accepted")
	}
}

func passingSummaryForTest(
	catalog PublicCatalog,
	peers PeerEvaluationSuite,
	kind string,
	split string,
	expectedCases int,
	casesPerSubject int,
	sourceCommit string,
) Technology20EvaluationSummary {
	summary := Technology20EvaluationSummary{
		SchemaVersion:  Technology20EvaluationSummarySchemaV2,
		EvaluationKind: kind,
		EvaluationIdentity: &EvaluationIdentity{
			SchemaVersion: "signalforge/technology20-" + kind + "-evaluation/v1",
			UniverseID:    UniverseID, Split: split,
			SuiteSHA256:           "b" + stringsOf("0", 63),
			SourceCommit:          sourceCommit,
			ModelID:               "signalforge-gemma4-26b-q4",
			LoopbackCoreInference: true,
			ShardEvaluationSHA256: map[string]string{"shard/evaluation.json": "c" + stringsOf("0", 63)},
		},
		ExpectedCases: expectedCases, CompletedCases: expectedCases,
		PopulationComplete: true,
		GateCounts:         map[string]int{"runtime_passed": expectedCases, "contract_passed": expectedCases},
		RuntimePassRate:    1, ContractPassRate: 1,
		FailureCodes: map[string]int{}, FailedGateCounts: map[string]int{},
		PacketAuthorityIntegrity: PacketAuthorityIntegrity{
			PacketsObserved: expectedCases, PacketsPassed: expectedCases,
			PacketsFailed: 0, PassRate: 1,
			MissingRefs: map[string]int{}, Failures: map[string]int{},
		},
		ByCompany:          map[string]EvaluationGroupSummary{},
		ByLane:             map[string]EvaluationGroupSummary{},
		InputCaseSHA256:    map[string]string{},
		ReleaseDisposition: "evaluation_only_not_promoted",
		ClaimBoundary:      "contract evidence only",
		SummarySHA256:      "d" + stringsOf("0", 63),
	}
	for index := 0; index < expectedCases; index++ {
		summary.InputCaseSHA256[fmt.Sprintf("cases/%03d.json", index)] =
			fmt.Sprintf("%064x", index+1)
	}
	group := func() EvaluationGroupSummary {
		return EvaluationGroupSummary{
			Cases:           casesPerSubject,
			GatePassRates:   map[string]float64{"runtime_passed": 1, "contract_passed": 1},
			RuntimePassRate: 1, ContractPassRate: 1,
		}
	}
	if kind == "standalone" {
		for _, company := range catalog.Companies {
			summary.ByCompany[company.CompanyID] = group()
		}
	} else {
		for _, lane := range peers.Lanes {
			summary.ByLane[lane.LaneID] = group()
		}
	}
	return summary
}

func promotionEvidenceForTest() map[string]string {
	return map[string]string{
		StandaloneDevelopmentEvidence: "1" + stringsOf("0", 63),
		StandaloneSealedEvidence:      "2" + stringsOf("0", 63),
		PeerDevelopmentEvidence:       "3" + stringsOf("0", 63),
		PeerSealedEvidence:            "4" + stringsOf("0", 63),
	}
}

func stringsOf(value string, count int) string {
	result := ""
	for index := 0; index < count; index++ {
		result += value
	}
	return result
}

func promotionJSON(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

func promotionBytesHash(value []byte) string {
	digest := sha256.Sum256(append(value, '\n'))
	return hex.EncodeToString(digest[:])
}
