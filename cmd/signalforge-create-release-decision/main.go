package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rvbernucci/signalforge/internal/productscope"
)

func main() {
	sourceCommit := flag.String("source-commit", "", "exact 40-character candidate commit")
	reviewerName := flag.String("reviewer-name", "", "named exact-candidate reviewer")
	reviewerRole := flag.String("reviewer-role", "", "reviewer's release role")
	recordLocator := flag.String("record-locator", "", "private review record locator")
	decidedAt := flag.String("decided-at", "", "review decision timestamp in RFC3339")
	conditions := flag.String(
		"conditions",
		"release only the hash-bound passing cohort",
		"semicolon-separated release conditions",
	)
	standaloneDevelopment := flag.String("standalone-development-summary", "", "development standalone aggregate")
	standaloneSealed := flag.String("standalone-sealed-summary", "", "sealed standalone aggregate")
	peerDevelopment := flag.String("peer-development-summary", "", "development peer aggregate")
	peerSealed := flag.String("peer-sealed-summary", "", "sealed peer aggregate")
	output := flag.String("output", "", "private exact-candidate decision output")
	accept := flag.Bool(
		"accept-exact-candidate",
		false,
		"explicitly accept the exact commit and evidence hashes",
	)
	flag.Parse()

	if !*accept {
		fatal(fmt.Errorf("--accept-exact-candidate is required; no decision is inferred"))
	}
	timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(*decidedAt))
	if err != nil {
		fatal(fmt.Errorf("parse --decided-at: %w", err))
	}
	evidence := map[string]string{
		productscope.StandaloneDevelopmentEvidence: fileSHA256(requiredPath(*standaloneDevelopment)),
		productscope.StandaloneSealedEvidence:      fileSHA256(requiredPath(*standaloneSealed)),
		productscope.PeerDevelopmentEvidence:       fileSHA256(requiredPath(*peerDevelopment)),
		productscope.PeerSealedEvidence:            fileSHA256(requiredPath(*peerSealed)),
	}
	record, err := productscope.PopulateExactReleaseDecisionHash(
		productscope.ExactReleaseDecision{
			SchemaVersion:  productscope.ExactReleaseDecisionSchemaV1,
			UniverseID:     productscope.UniverseID,
			SourceCommit:   strings.TrimSpace(*sourceCommit),
			ReviewerName:   strings.TrimSpace(*reviewerName),
			ReviewerRole:   strings.TrimSpace(*reviewerRole),
			Disposition:    "accepted",
			Conditions:     splitConditions(*conditions),
			EvidenceSHA256: evidence,
			DecidedAt:      timestamp.UTC(),
			RecordLocator:  strings.TrimSpace(*recordLocator),
		},
	)
	if err != nil {
		fatal(err)
	}
	if strings.TrimSpace(*output) == "" {
		fatal(fmt.Errorf("--output is required"))
	}
	if err := writeJSONAtomic(*output, record); err != nil {
		fatal(err)
	}
	fmt.Printf("exact release decision: %s\n", record.DecisionSHA256)
}

func splitConditions(value string) []string {
	result := []string{}
	for _, item := range strings.Split(value, ";") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func requiredPath(value string) string {
	if strings.TrimSpace(value) == "" {
		fatal(fmt.Errorf("all four evaluation summary paths are required"))
	}
	return value
}

func fileSHA256(path string) string {
	payload, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func writeJSONAtomic(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
