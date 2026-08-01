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
	var decision AccountingProfessionalDecisionRecord
	readAccountingFixture(t, filepath.Join(root, "technology20-accounting-professional-decision.json"), &decision)
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
	if err := ValidateAccountingProfessionalDecision(decision, registry); err != nil {
		t.Fatal(err)
	}
	if decision.DecisionSHA256 != runtimeAccountingProfessionalDecision.DecisionSHA256 {
		t.Fatal("frozen professional decision does not match runtime authority")
	}
	if manifest.Companies != 20 || manifest.Inputs != 160 ||
		len(exceptions.Exceptions) != 18 || len(review.Items) != 9 {
		t.Fatalf(
			"unexpected frozen population: companies=%d inputs=%d exceptions=%d review=%d",
			manifest.Companies, manifest.Inputs, len(exceptions.Exceptions), len(review.Items),
		)
	}
	wantDispositions := map[AccountingDisposition]int{
		AccountingCanonical:     144,
		AccountingReviewedAlias: 5,
		AccountingContextOnly:   3,
		AccountingUnavailable:   8,
	}
	for disposition, want := range wantDispositions {
		if got := manifest.DispositionCount[disposition]; got != want {
			t.Fatalf("manifest disposition %s = %d, want %d", disposition, got, want)
		}
	}
	proposedDispositions := map[AccountingDisposition]int{}
	for _, item := range review.Items {
		proposedDispositions[item.ProposedDisposition]++
		if item.Decision != "pending" || item.NamedReviewer != "" ||
			item.ReviewerQualification != "" || item.DecisionTimestamp != "" {
			t.Fatalf("machine-generated review item self-approved: %+v", item)
		}
		if !strings.HasPrefix(item.SourceCitation, "https://www.sec.gov/Archives/") ||
			item.SourceLocator == "" || item.BoundedSourceLanguage == "" {
			t.Fatalf("review item lacks bounded primary-source evidence: %+v", item)
		}
	}
	if proposedDispositions[AccountingReviewedAlias] != 6 ||
		proposedDispositions[AccountingContextOnly] != 3 {
		t.Fatalf(
			"unexpected review proposals: aliases=%d context_only=%d",
			proposedDispositions[AccountingReviewedAlias],
			proposedDispositions[AccountingContextOnly],
		)
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

func TestFrozenAccountingAuthorityIncludesHumanReviewArtifact(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "accounting-authority")
	review, err := os.ReadFile(filepath.Join(root, "technology20-accounting-professional-review.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(review, []byte("| **CONDITIONALLY ACCEPTED** |")) != 9 ||
		!bytes.Contains(review, []byte(runtimeAccountingAuthorityRegistry.RegistrySHA256)) {
		t.Fatal("human-readable review packet does not match the hash-bound named decision")
	}
	if !bytes.Contains(
		review,
		[]byte("| Company | Input | Concept | Perimeter | Technical outcome | Evidence | Named decision |"),
	) || bytes.Contains(review, []byte("Bounded source language for")) {
		t.Fatal("human-readable review packet does not preserve one review decision per table row")
	}
	if bytes.Count(
		review,
		[]byte("Conditionally support exact issuer-specific alias; reject broader use"),
	) != 6 || bytes.Count(
		review,
		[]byte("Support contextual arithmetic only; reject canonical classification and comparative or ranking use"),
	) != 3 {
		t.Fatal("human-readable review packet does not preserve the technical decision population")
	}
	for _, required := range [][]byte{
		[]byte("Registry content SHA-256 (`registry_sha256`; self-field excluded)"),
		[]byte("Technical research outcome: `CONDITIONALLY_SUPPORTED_AT_EXACT_SCOPE`"),
		[]byte("Named professional decision: `CONDITIONALLY_ACCEPTED`"),
		[]byte("Machine decision encoding: `HASH_BOUND_CONDITIONALLY_ACCEPTED`"),
		[]byte("Runtime activation: `ACTIVE_AT_EXACT_SCOPE_FAIL_CLOSED`"),
		[]byte("Rafael Bernucci"),
		[]byte("2026-07-29T14:00:31Z"),
		[]byte("A valid canonical fact wins for the same issuer and period"),
		[]byte("never canonical capex"),
		[]byte("context-only outputs are mechanically excluded"),
	} {
		if !bytes.Contains(review, required) {
			t.Fatalf("human-readable review packet lacks required boundary %q", required)
		}
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
