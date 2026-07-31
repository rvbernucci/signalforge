package productscope

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const (
	promotedSourceCommit = "4498e60c16821586f830d196269f39702f38ca99"
	promotionDecisionSHA = "cc4bade454b230a5854ba2d278403749fd8a0f85758c4bd8896492f0cede7181"
)

func TestPublicCatalogPreservesPromotedActivationBoundary(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "productscope", "technology20-catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog PublicCatalog
	if err := json.Unmarshal(payload, &catalog); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePublicCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Companies) != 20 || len(catalog.PeerLanes) != 5 {
		t.Fatalf("catalog population = %d companies / %d lanes", len(catalog.Companies), len(catalog.PeerLanes))
	}
	for _, company := range catalog.Companies {
		if !company.ResearchEnabled || len(company.PromotionEvidenceSHA256) != 4 {
			t.Fatalf("promoted company %q lacks exact evidence: %+v", company.CompanyID, company)
		}
	}
	for _, lane := range catalog.PeerLanes {
		if !lane.Enabled || len(lane.PromotionEvidenceSHA256) != 4 {
			t.Fatalf("promoted lane %q lacks exact evidence: %+v", lane.LaneID, lane)
		}
	}
}

func TestTechnology20PromotionManifestBindsExactPublicArtifacts(t *testing.T) {
	root := filepath.Join("..", "..")
	var manifest Technology20PromotionManifest
	readFixtureJSON(t, filepath.Join(root, "evidence", "technology20-promotion-manifest.json"), &manifest)
	if err := ValidateTechnology20PromotionManifest(manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SourceCommit != promotedSourceCommit ||
		manifest.HumanDecisionSHA256 != promotionDecisionSHA {
		t.Fatalf("promotion identity = %s / %s", manifest.SourceCommit, manifest.HumanDecisionSHA256)
	}

	catalogPath := filepath.Join(root, "fixtures", "productscope", "technology20-catalog.json")
	peersPath := filepath.Join(root, "fixtures", "productscope", "technology20-peer-evaluation.json")
	if got := fixtureSHA256(t, catalogPath); got != manifest.PromotedCatalogSHA256 {
		t.Fatalf("catalog SHA-256 = %s, want %s", got, manifest.PromotedCatalogSHA256)
	}
	if got := fixtureSHA256(t, peersPath); got != manifest.PromotedPeersSHA256 {
		t.Fatalf("peer SHA-256 = %s, want %s", got, manifest.PromotedPeersSHA256)
	}

	var catalog PublicCatalog
	var peers PeerEvaluationSuite
	readFixtureJSON(t, catalogPath, &catalog)
	readFixtureJSON(t, peersPath, &peers)
	if err := ValidateReleaseAlignment(catalog, peers); err != nil {
		t.Fatal(err)
	}
	if catalog.PromotionDecisionSHA256 != manifest.HumanDecisionSHA256 ||
		peers.PromotionDecisionSHA256 != manifest.HumanDecisionSHA256 {
		t.Fatal("public authority artifacts are not bound to the accepted decision")
	}
}

func readFixtureJSON(t *testing.T, path string, target any) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatal(err)
	}
}

func fixtureSHA256(t *testing.T, path string) string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(payload))
}
