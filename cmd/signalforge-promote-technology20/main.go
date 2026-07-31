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
	catalogPath := flag.String("catalog", "fixtures/productscope/technology20-catalog.json", "unpromoted public catalog")
	peersPath := flag.String("peers", "fixtures/productscope/technology20-peer-evaluation.json", "unpromoted peer authority")
	standaloneDevelopmentPath := flag.String("standalone-development-summary", "", "development standalone aggregate")
	standaloneSealedPath := flag.String("standalone-sealed-summary", "", "sealed standalone aggregate")
	peerDevelopmentPath := flag.String("peer-development-summary", "", "development peer aggregate")
	peerSealedPath := flag.String("peer-sealed-summary", "", "sealed peer aggregate")
	decisionPath := flag.String("decision", "", "private exact-candidate human decision")
	generatedAt := flag.String("generated-at", "", "promotion timestamp in RFC3339")
	outputCatalog := flag.String("output-catalog", "", "promoted catalog output")
	outputPeers := flag.String("output-peers", "", "promoted peer authority output")
	outputManifest := flag.String("output-manifest", "", "promotion manifest output")
	flag.Parse()

	timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(*generatedAt))
	if err != nil {
		fatal(fmt.Errorf("parse --generated-at: %w", err))
	}
	requireDistinctOutputs(*outputCatalog, *outputPeers, *outputManifest)

	var catalog productscope.PublicCatalog
	var peers productscope.PeerEvaluationSuite
	var standaloneDevelopment productscope.Technology20EvaluationSummary
	var standaloneSealed productscope.Technology20EvaluationSummary
	var peerDevelopment productscope.Technology20EvaluationSummary
	var peerSealed productscope.Technology20EvaluationSummary
	var decision productscope.ExactReleaseDecision
	readJSON(*catalogPath, &catalog)
	readJSON(*peersPath, &peers)
	readJSON(requiredPath(*standaloneDevelopmentPath), &standaloneDevelopment)
	readJSON(requiredPath(*standaloneSealedPath), &standaloneSealed)
	readJSON(requiredPath(*peerDevelopmentPath), &peerDevelopment)
	readJSON(requiredPath(*peerSealedPath), &peerSealed)
	readJSON(requiredPath(*decisionPath), &decision)

	evidence := map[string]string{
		productscope.StandaloneDevelopmentEvidence: fileSHA256(*standaloneDevelopmentPath),
		productscope.StandaloneSealedEvidence:      fileSHA256(*standaloneSealedPath),
		productscope.PeerDevelopmentEvidence:       fileSHA256(*peerDevelopmentPath),
		productscope.PeerSealedEvidence:            fileSHA256(*peerSealedPath),
	}
	promotedCatalog, promotedPeers, manifest, err := productscope.PromoteTechnology20(
		productscope.PromotionInput{
			Catalog: catalog, Peers: peers,
			StandaloneDevelopment: standaloneDevelopment,
			StandaloneSealed:      standaloneSealed,
			PeerDevelopment:       peerDevelopment,
			PeerSealed:            peerSealed,
			EvidenceSHA256:        evidence,
			Decision:              decision,
			GeneratedAt:           timestamp.UTC(),
		},
	)
	if err != nil {
		fatal(err)
	}
	catalogPayload := encodeJSON(promotedCatalog)
	peersPayload := encodeJSON(promotedPeers)
	manifest, err = productscope.PopulatePromotionManifestOutputHashes(
		manifest,
		bytesSHA256(catalogPayload),
		bytesSHA256(peersPayload),
	)
	if err != nil {
		fatal(err)
	}
	manifestPayload := encodeJSON(manifest)
	writeAtomic(*outputCatalog, catalogPayload, 0o644)
	writeAtomic(*outputPeers, peersPayload, 0o644)
	writeAtomic(*outputManifest, manifestPayload, 0o644)
	fmt.Printf(
		"Technology 20 promotion: companies=%d peer_lanes=%d manifest=%s\n",
		countPromoted(manifest.Companies),
		countPromoted(manifest.PeerLanes),
		manifest.ManifestSHA256,
	)
}

func countPromoted(values []productscope.PromotionOutcome) int {
	count := 0
	for _, item := range values {
		if item.Promoted {
			count++
		}
	}
	return count
}

func requireDistinctOutputs(values ...string) {
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			fatal(fmt.Errorf("all three promotion output paths are required"))
		}
		absolute, err := filepath.Abs(value)
		if err != nil {
			fatal(err)
		}
		if seen[absolute] {
			fatal(fmt.Errorf("promotion output paths must be distinct"))
		}
		seen[absolute] = true
	}
}

func requiredPath(value string) string {
	if strings.TrimSpace(value) == "" {
		fatal(fmt.Errorf("decision and all four evaluation summary paths are required"))
	}
	return value
}

func readJSON(path string, value any) {
	payload, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	if err := json.Unmarshal(payload, value); err != nil {
		fatal(fmt.Errorf("decode %s: %w", path, err))
	}
}

func encodeJSON(value any) []byte {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fatal(err)
	}
	return append(payload, '\n')
}

func fileSHA256(path string) string {
	payload, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	return bytesSHA256(payload)
}

func bytesSHA256(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func writeAtomic(path string, payload []byte, mode os.FileMode) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		fatal(err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, payload, mode); err != nil {
		fatal(err)
	}
	if err := os.Rename(temporary, path); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
