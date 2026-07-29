package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rvbernucci/signalforge/internal/data"
	"github.com/rvbernucci/signalforge/internal/productscope"
)

func main() {
	metricsPath := flag.String("metrics", "", "Frozen normalized_metrics.jsonl path")
	factsPath := flag.String("facts", "", "Frozen reported_facts.jsonl path")
	filingsPath := flag.String("filings", "", "Frozen filings.jsonl path")
	catalogPath := flag.String(
		"catalog", "fixtures/productscope/technology20-catalog.json",
		"Public-safe Technology 20 catalog",
	)
	outputDir := flag.String("output-dir", "", "Output directory for public-safe accounting authority artifacts")
	reportDir := flag.String(
		"report-dir", "docs/accounting-authority",
		"Documentation-only output directory for human-readable review artifacts",
	)
	asOfRaw := flag.String("as-of", "", "Point-in-time cutoff in RFC3339")
	flag.Parse()
	if *metricsPath == "" || *factsPath == "" || *filingsPath == "" ||
		*catalogPath == "" || *outputDir == "" || *asOfRaw == "" {
		exit(errors.New("--metrics, --facts, --filings, --catalog, --output-dir, and --as-of are required"))
	}
	asOf, err := time.Parse(time.RFC3339, *asOfRaw)
	if err != nil {
		exit(err)
	}
	catalogPayload, err := os.ReadFile(*catalogPath)
	if err != nil {
		exit(err)
	}
	var catalog productscope.PublicCatalog
	if err := json.Unmarshal(catalogPayload, &catalog); err != nil {
		exit(err)
	}
	if err := productscope.ValidatePublicCatalog(catalog); err != nil {
		exit(err)
	}
	metrics, metricsSHA, err := readMetrics(*metricsPath)
	if err != nil {
		exit(err)
	}
	facts, factsSHA, err := readFacts(*factsPath, referencedFactIDs(metrics))
	if err != nil {
		exit(err)
	}
	filings, filingsSHA, err := readFilings(*filingsPath, catalog)
	if err != nil {
		exit(err)
	}
	build, err := productscope.BuildTechnology20AccountingAuthority(
		catalog, metrics, facts, filings, asOf,
		productscope.AccountingAuthoritySourceHashes{
			CatalogSHA256: hashBytes(catalogPayload), MetricsSHA256: metricsSHA,
			FactsSHA256: factsSHA, FilingsSHA256: filingsSHA,
		},
	)
	if err != nil {
		exit(err)
	}
	if err := writeBuild(*outputDir, *reportDir, build); err != nil {
		exit(err)
	}
	fmt.Printf(
		"accounting authority: %d companies, %d inputs, %d exceptions, %d professional decisions pending\n",
		build.Manifest.Companies, build.Manifest.Inputs,
		len(build.Exceptions.Exceptions), len(build.Review.Items),
	)
}

func writeBuild(
	root string,
	reportRoot string,
	build productscope.Technology20AccountingBuild,
) error {
	if err := os.MkdirAll(filepath.Join(root, "packets"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(reportRoot, 0o755); err != nil {
		return err
	}
	artifacts := []struct {
		path  string
		value any
	}{
		{filepath.Join(root, "technology20-accounting-perimeter-registry.json"), build.Registry},
		{filepath.Join(root, "technology20-accounting-authority.json"), build.Manifest},
		{filepath.Join(root, "technology20-accounting-exceptions.json"), build.Exceptions},
		{filepath.Join(root, "technology20-accounting-professional-review.json"), build.Review},
	}
	for _, artifact := range artifacts {
		if err := writeJSON(artifact.path, artifact.value); err != nil {
			return err
		}
	}
	for _, packet := range build.Packets {
		if err := writeJSON(filepath.Join(root, "packets", packet.PrimaryTicker+".json"), packet); err != nil {
			return err
		}
	}
	if err := atomicWrite(
		filepath.Join(reportRoot, "technology20-concept-coverage.tsv"),
		[]byte(conceptCoverageTSV(build.Packets)),
	); err != nil {
		return err
	}
	return atomicWrite(
		filepath.Join(reportRoot, "technology20-accounting-professional-review.md"),
		[]byte(professionalReviewMarkdown(build.Review)),
	)
}

func conceptCoverageTSV(packets []productscope.CompanyAccountingAuthorityPacket) string {
	var output bytes.Buffer
	output.WriteString(strings.Join([]string{
		"company_id", "ticker", "canonical_input", "taxonomy_namespace", "taxonomy_concept",
		"disposition", "accounting_perimeter", "observed_records", "active_annual_sources",
		"source_forms", "reason_code", "professional_review_status",
	}, "\t") + "\n")
	for _, packet := range packets {
		for _, input := range packet.Inputs {
			if len(input.Concepts) == 0 {
				output.WriteString(strings.Join([]string{
					packet.CompanyID, packet.PrimaryTicker, input.CanonicalInput, "", "",
					string(productscope.AccountingUnavailable), "unavailable", "0", "0", "",
					"authorized_annual_source_unavailable", "not_submitted_for_review",
				}, "\t") + "\n")
				continue
			}
			for _, concept := range input.Concepts {
				output.WriteString(strings.Join([]string{
					packet.CompanyID,
					packet.PrimaryTicker,
					input.CanonicalInput,
					concept.Mapping.TaxonomyNamespace,
					concept.Mapping.TaxonomyConcept,
					string(concept.Mapping.Disposition),
					concept.Mapping.AccountingPerimeter,
					strconv.Itoa(concept.ObservedRecords),
					strconv.Itoa(len(concept.ActiveAnnualSource)),
					strings.Join(concept.SourceForms, ","),
					concept.Mapping.ReasonCode,
					concept.Mapping.ProfessionalReviewStatus,
				}, "\t") + "\n")
			}
		}
	}
	return output.String()
}

func professionalReviewMarkdown(review productscope.AccountingProfessionalReviewPacket) string {
	var output strings.Builder
	output.WriteString("# Technology 20 Accounting Professional Review\n\n")
	output.WriteString("This public-safe review artifact is documentation only and is not copied into the application\n")
	output.WriteString("runtime.\n\n")
	output.WriteString("Registry SHA-256: `" + review.RegistrySHA256 + "`  \n")
	output.WriteString("As of: `" + review.AsOf.Format(time.RFC3339) + "`\n\n")
	output.WriteString(review.ClaimBoundary + "\n\n")
	output.WriteString("| Company | Input | Concept | Proposed status | Source locator | Decision |\n")
	output.WriteString("|---|---|---|---|---|---|\n")
	for _, item := range review.Items {
		locator := item.SourceLocator
		if item.SourceCitation != "" {
			locator = "[" + strings.ReplaceAll(locator, "|", "/") + "](" + item.SourceCitation + ")"
		}
		output.WriteString("| " + item.CompanyID + " | " + item.CanonicalInput + " | `" +
			item.TaxonomyConcept + "` | `" + string(item.ProposedDisposition) + "` | " +
			locator + " | **PENDING** |\n")
		if item.BoundedSourceLanguage != "" {
			output.WriteString("\nBounded source language for `" + item.ExceptionID + "`: \"" +
				item.BoundedSourceLanguage + "\"\n\n")
		}
	}
	output.WriteString("\nA named reviewer must record qualification, decision, timestamp, and release identity outside runtime inputs.\n")
	return output.String()
}

func readMetrics(path string) (map[string][]data.NormalizedMetric, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	hasher := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(file, hasher))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	result := map[string][]data.NormalizedMetric{}
	for scanner.Scan() {
		var metric data.NormalizedMetric
		if err := json.Unmarshal(scanner.Bytes(), &metric); err != nil {
			return nil, "", err
		}
		result[metric.CompanyID] = append(result[metric.CompanyID], metric)
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	return result, hex.EncodeToString(hasher.Sum(nil)), nil
}

func readFacts(path string, wanted map[string]bool) (map[string]data.ReportedFact, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	hasher := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(file, hasher))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	result := map[string]data.ReportedFact{}
	for scanner.Scan() {
		var fact data.ReportedFact
		if err := json.Unmarshal(scanner.Bytes(), &fact); err != nil {
			return nil, "", err
		}
		if wanted[fact.FactID] {
			result[fact.FactID] = fact
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	return result, hex.EncodeToString(hasher.Sum(nil)), nil
}

func readFilings(path string, catalog productscope.PublicCatalog) ([]data.Filing, string, error) {
	companies := map[string]bool{}
	for _, company := range catalog.Companies {
		companies[company.CompanyID] = true
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	hasher := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(file, hasher))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	result := []data.Filing{}
	for scanner.Scan() {
		var filing data.Filing
		if err := json.Unmarshal(scanner.Bytes(), &filing); err != nil {
			return nil, "", err
		}
		if companies[filing.CompanyID] && (filing.FormType == "10-K" || filing.FormType == "10-K/A") {
			result = append(result, filing)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].FilingID < result[j].FilingID })
	return result, hex.EncodeToString(hasher.Sum(nil)), nil
}

func referencedFactIDs(metrics map[string][]data.NormalizedMetric) map[string]bool {
	result := map[string]bool{}
	for _, companyMetrics := range metrics {
		for _, metric := range companyMetrics {
			for _, factID := range metric.SourceFactIDs {
				result[factID] = true
			}
		}
	}
	return result
}

func writeJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(payload, '\n'))
}

func atomicWrite(path string, payload []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func hashBytes(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
