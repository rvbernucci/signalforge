package evalcheckpoint

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckpointDecodeRequiresExactIdentity(t *testing.T) {
	identity := validIdentity()
	payload, err := json.Marshal(NewEnvelope(identity, map[string]string{"status": "passed"}))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Decode[map[string]string](payload, identity)
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "passed" {
		t.Fatalf("decoded result = %v", result)
	}
	changed := identity
	changed.SourceCommit = "different"
	if _, err := Decode[map[string]string](payload, changed); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatalf("identity mismatch error = %v", err)
	}
}

func TestCheckpointDecodeRejectsLegacyBareResult(t *testing.T) {
	payload := []byte(`{"journey_id":"journey-1","status":"passed"}`)
	if _, err := Decode[map[string]string](payload, validIdentity()); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatalf("legacy checkpoint error = %v", err)
	}
}

func TestDirectorySHA256ChangesWithPathOrContent(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "company", "authority.json")
	if err := os.MkdirAll(filepath.Dir(first), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("first"), 0o640); err != nil {
		t.Fatal(err)
	}
	before, err := DirectorySHA256(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("second"), 0o640); err != nil {
		t.Fatal(err)
	}
	afterContent, err := DirectorySHA256(root)
	if err != nil {
		t.Fatal(err)
	}
	if before == afterContent {
		t.Fatal("content change did not change authority digest")
	}
	second := filepath.Join(root, "company", "renamed.json")
	if err := os.Rename(first, second); err != nil {
		t.Fatal(err)
	}
	afterPath, err := DirectorySHA256(root)
	if err != nil {
		t.Fatal(err)
	}
	if afterContent == afterPath {
		t.Fatal("path change did not change authority digest")
	}
}

func TestDirectorySHA256RejectsSymlinkAndEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	if _, err := DirectorySHA256(root); err == nil {
		t.Fatal("empty authority directory was accepted")
	}
	target := filepath.Join(root, "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "alias.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := DirectorySHA256(root); err == nil {
		t.Fatal("symlinked authority was accepted")
	}
}

func TestValidateIdentityRequiresSpecialistAuthorityWhenEnabled(t *testing.T) {
	identity := validIdentity()
	identity.SpecialistEnabled = true
	if err := ValidateIdentity(identity); err == nil {
		t.Fatal("enabled specialist without provider authority was accepted")
	}
	identity.SpecialistProvider = "radeon-vllm"
	identity.SpecialistBaseURL = "https://example.test/v1"
	identity.SpecialistModel = "model"
	if err := ValidateIdentity(identity); err != nil {
		t.Fatal(err)
	}
}

func validIdentity() Identity {
	return Identity{
		SchemaVersion: IdentitySchemaVersion, EvaluationKind: "standalone",
		SuiteSHA256: "suite", CatalogSHA256: "catalog",
		PeerAuthoritySHA256: "peers", FinancialAuthoritySHA256: "financial",
		RunnerSHA256: "runner", SourceCommit: "commit", ModelID: "model",
		BaseURL: "http://127.0.0.1:8000/v1", Timeout: "4m0s",
		ContextConcurrency: 4, JourneyID: "journey-1",
		QuestionSHA256: SHA256String("question"), RunID: "run-1", RequestID: "request-1",
	}
}
