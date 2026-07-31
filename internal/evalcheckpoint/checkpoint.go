package evalcheckpoint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SchemaVersion         = "signalforge/evaluation-checkpoint/v1"
	IdentitySchemaVersion = "signalforge/evaluation-checkpoint-identity/v1"
)

var ErrIdentityMismatch = errors.New("evaluation checkpoint identity mismatch")

// Identity binds a private case checkpoint to every input that can materially
// change the evaluated answer. Secrets and source bodies are deliberately
// excluded; their stable, non-secret authorities are represented by digests.
type Identity struct {
	SchemaVersion            string `json:"schema_version"`
	EvaluationKind           string `json:"evaluation_kind"`
	SuiteSHA256              string `json:"suite_sha256"`
	CatalogSHA256            string `json:"catalog_sha256"`
	PeerAuthoritySHA256      string `json:"peer_authority_sha256"`
	FinancialAuthoritySHA256 string `json:"financial_authority_sha256"`
	RunnerSHA256             string `json:"runner_sha256"`
	SourceCommit             string `json:"source_commit"`
	ModelID                  string `json:"model_id"`
	BaseURL                  string `json:"base_url"`
	SpecialistEnabled        bool   `json:"specialist_enabled"`
	SpecialistProvider       string `json:"specialist_provider,omitempty"`
	SpecialistBaseURL        string `json:"specialist_base_url,omitempty"`
	SpecialistModel          string `json:"specialist_model,omitempty"`
	Timeout                  string `json:"timeout"`
	ContextConcurrency       int    `json:"context_concurrency"`
	JourneyID                string `json:"journey_id"`
	QuestionSHA256           string `json:"question_sha256"`
	RunID                    string `json:"run_id"`
	RequestID                string `json:"request_id"`
}

type Envelope[T any] struct {
	SchemaVersion string   `json:"schema_version"`
	Identity      Identity `json:"identity"`
	Result        T        `json:"result"`
}

func NewEnvelope[T any](identity Identity, result T) Envelope[T] {
	return Envelope[T]{
		SchemaVersion: SchemaVersion,
		Identity:      identity,
		Result:        result,
	}
}

func Decode[T any](payload []byte, expected Identity) (T, error) {
	var zero T
	var checkpoint Envelope[T]
	if err := json.Unmarshal(payload, &checkpoint); err != nil {
		return zero, err
	}
	if checkpoint.SchemaVersion != SchemaVersion ||
		checkpoint.Identity.SchemaVersion != IdentitySchemaVersion ||
		checkpoint.Identity != expected {
		return zero, ErrIdentityMismatch
	}
	return checkpoint.Result, nil
}

func SHA256String(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func FileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// DirectorySHA256 hashes sorted relative paths, file sizes, and file contents.
// Symlinks and non-regular entries fail closed so two different authority trees
// cannot share a checkpoint identity through filesystem indirection.
func DirectorySHA256(root string) (string, error) {
	cleanRoot := filepath.Clean(root)
	var paths []string
	if err := filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == cleanRoot {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("checkpoint authority directory contains symlink %q", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("checkpoint authority directory contains non-regular file %q", path)
		}
		relative, err := filepath.Rel(cleanRoot, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", errors.New("checkpoint authority directory contains no regular files")
	}
	sort.Strings(paths)
	digest := sha256.New()
	for _, relative := range paths {
		path := filepath.Join(cleanRoot, filepath.FromSlash(relative))
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		fileDigest, err := FileSHA256(path)
		if err != nil {
			return "", err
		}
		_, _ = fmt.Fprintf(digest, "%s\x00%d\x00%s\n", relative, info.Size(), fileDigest)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func ValidateIdentity(identity Identity) error {
	required := map[string]string{
		"schema_version":             identity.SchemaVersion,
		"evaluation_kind":            identity.EvaluationKind,
		"suite_sha256":               identity.SuiteSHA256,
		"catalog_sha256":             identity.CatalogSHA256,
		"peer_authority_sha256":      identity.PeerAuthoritySHA256,
		"financial_authority_sha256": identity.FinancialAuthoritySHA256,
		"runner_sha256":              identity.RunnerSHA256,
		"source_commit":              identity.SourceCommit,
		"model_id":                   identity.ModelID,
		"base_url":                   identity.BaseURL,
		"timeout":                    identity.Timeout,
		"journey_id":                 identity.JourneyID,
		"question_sha256":            identity.QuestionSHA256,
		"run_id":                     identity.RunID,
		"request_id":                 identity.RequestID,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("checkpoint identity %s is required", name)
		}
	}
	if identity.SchemaVersion != IdentitySchemaVersion {
		return fmt.Errorf("unsupported checkpoint identity schema %q", identity.SchemaVersion)
	}
	if identity.ContextConcurrency <= 0 {
		return errors.New("checkpoint identity context_concurrency must be positive")
	}
	if identity.SpecialistEnabled &&
		(strings.TrimSpace(identity.SpecialistProvider) == "" ||
			strings.TrimSpace(identity.SpecialistBaseURL) == "" ||
			strings.TrimSpace(identity.SpecialistModel) == "") {
		return errors.New("enabled specialist identity requires provider, base URL, and model")
	}
	return nil
}
