package productscope

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrozenTechnology20AccountingAuthorityValidatesEndToEnd(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "productscope", "accounting-authority")
	var registry AccountingAuthorityRegistry
	readAccountingFixture(t, filepath.Join(root, "technology20-accounting-perimeter-registry.json"), &registry)
	var manifest Technology20AccountingAuthority
	readAccountingFixture(t, filepath.Join(root, "technology20-accounting-authority.json"), &manifest)
	var exceptions Technology20AccountingExceptions
	readAccountingFixture(t, filepath.Join(root, "technology20-accounting-exceptions.json"), &exceptions)
	var review AccountingProfessionalReviewPacket
	readAccountingFixture(t, filepath.Join(root, "technology20-accounting-professional-review.json"), &review)
	build := Technology20AccountingBuild{
		Registry: registry, Manifest: manifest, Exceptions: exceptions, Review: review,
	}
	for _, reference := range manifest.Packets {
		var packet CompanyAccountingAuthorityPacket
		readAccountingFixture(t, filepath.Join(root, reference.Path), &packet)
		build.Packets = append(build.Packets, packet)
	}
	if err := ValidateTechnology20AccountingBuild(build); err != nil {
		t.Fatal(err)
	}
	if manifest.Companies != 20 || manifest.Inputs != 160 ||
		len(exceptions.Exceptions) != 18 || len(review.Items) != 9 {
		t.Fatalf(
			"unexpected frozen population: companies=%d inputs=%d exceptions=%d review=%d",
			manifest.Companies, manifest.Inputs, len(exceptions.Exceptions), len(review.Items),
		)
	}
	for _, item := range review.Items {
		if item.Decision != "pending" || item.NamedReviewer != "" ||
			item.ReviewerQualification != "" || item.DecisionTimestamp != "" {
			t.Fatalf("machine-generated review item self-approved: %+v", item)
		}
	}
}

func TestFrozenAccountingAuthorityContainsMetadataNotCalculationValues(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "productscope", "accounting-authority")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(payload, []byte(`"value"`)) {
			t.Fatalf("%s contains a calculation value field", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFrozenAccountingAuthorityIncludesHumanReviewAndCoverageArtifacts(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "accounting-authority")
	coverage, err := os.ReadFile(filepath.Join(root, "technology20-concept-coverage.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(coverage)), "\n")
	if len(lines) != 194 {
		t.Fatalf("coverage rows including header = %d, want 194", len(lines))
	}
	if lines[0] != "company_id\tticker\tcanonical_input\ttaxonomy_namespace\ttaxonomy_concept\tdisposition\taccounting_perimeter\tobserved_records\tactive_annual_sources\tsource_forms\treason_code\tprofessional_review_status" {
		t.Fatal("coverage header does not match the frozen review contract")
	}
	review, err := os.ReadFile(filepath.Join(root, "technology20-accounting-professional-review.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(review, []byte("| **PENDING** |")) != 9 ||
		!bytes.Contains(review, []byte(runtimeAccountingAuthorityRegistry.RegistrySHA256)) {
		t.Fatal("human-readable review packet does not match the frozen registry and pending population")
	}
}

func readAccountingFixture(t *testing.T, path string, destination any) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, destination); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
