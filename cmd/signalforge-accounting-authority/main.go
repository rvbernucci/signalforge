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

const (
	acceptedProfessionalReviewRegistrySHA  = "1c40b44538eee8c64e066bbf224aae51ff45a05094c1c36a867d58b779973dd4"
	namedProfessionalReviewer              = "Rafael Bernucci"
	namedProfessionalReviewerQualification = "Project owner and Accounting graduate from the University of Sao Paulo; not acting as an independent auditor"
	namedProfessionalReviewTimestamp       = "2026-07-29T14:00:31Z"
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
		"accounting authority: %d companies, %d inputs, %d exceptions, %d hash-bound professional decisions\n",
		build.Manifest.Companies, build.Manifest.Inputs,
		len(build.Exceptions.Exceptions), len(build.Review.Items),
	)
}

func writeBuild(
	root string,
	reportRoot string,
	build productscope.Technology20AccountingBuild,
) error {
	decision, err := productscope.DefaultAccountingProfessionalDecision()
	if err != nil {
		return err
	}
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
		{filepath.Join(root, "technology20-accounting-professional-decision.json"), decision},
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
	output.WriteString("Registry content SHA-256 (`registry_sha256`; self-field excluded): `" +
		review.RegistrySHA256 + "`  \n")
	output.WriteString("As of: `" + review.AsOf.Format(time.RFC3339) + "`\n\n")
	output.WriteString("Blank fields in the machine-generated review packet are deliberate: generation cannot self-approve.\n")
	output.WriteString("The separate professional decision record is reviewer-authored, hash-bound, and fail-closed.\n\n")
	aliases, contextual := reviewDispositionCounts(review)
	namedDecisionAccepted := review.RegistrySHA256 == acceptedProfessionalReviewRegistrySHA
	decision, decisionErr := productscope.DefaultAccountingProfessionalDecision()
	machineDecisionActive := decisionErr == nil &&
		decision.RegistrySHA256 == review.RegistrySHA256
	namedDecisionStatus := "`PENDING`"
	tableDecision := "**PENDING**"
	if namedDecisionAccepted {
		namedDecisionStatus = "`CONDITIONALLY_ACCEPTED`"
		tableDecision = "**CONDITIONALLY ACCEPTED**"
	}
	output.WriteString("## Review Status\n\n")
	output.WriteString("- Technical research outcome: `CONDITIONALLY_SUPPORTED_AT_EXACT_SCOPE`\n")
	output.WriteString("- Exact-scope population: `" + strconv.Itoa(aliases) + " reviewed aliases`, `" +
		strconv.Itoa(contextual) + " context-only mappings`\n")
	output.WriteString("- Named professional decision: " + namedDecisionStatus + "\n")
	if machineDecisionActive {
		output.WriteString("- Machine decision encoding: `HASH_BOUND_CONDITIONALLY_ACCEPTED`\n")
		output.WriteString("- Decision record SHA-256: `" + decision.DecisionSHA256 + "`\n")
		output.WriteString("- Runtime activation: `ACTIVE_AT_EXACT_SCOPE_FAIL_CLOSED`\n\n")
	} else {
		output.WriteString("- Machine decision encoding: `PENDING`\n")
		output.WriteString("- Runtime activation: `BLOCKED`\n\n")
	}
	output.WriteString("The official filing evidence supports each proposed disposition only at its exact issuer,\n")
	output.WriteString("period, unit, dimension, filing-chain, label, and accounting-perimeter boundary. Any broader\n")
	output.WriteString("use described below is technically rejected.\n\n")
	output.WriteString("| Company | Input | Concept | Perimeter | Technical outcome | Evidence | Named decision |\n")
	output.WriteString("|---|---|---|---|---|---|---|\n")
	for _, item := range review.Items {
		locator := item.SourceLocator
		if item.SourceCitation != "" {
			locator = "[" + markdownTableCell(locator) + "](" + item.SourceCitation + ")"
		}
		output.WriteString("| " + markdownTableCell(item.CompanyID) + " | " +
			markdownTableCell(item.CanonicalInput) + " | `" +
			item.TaxonomyConcept + "` | `" + markdownTableCell(item.AccountingPerimeter) + "` | " +
			markdownTableCell(technicalReviewOutcome(item)) + " | " + locator + ": " +
			markdownTableCell(item.BoundedSourceLanguage) + " | " + tableDecision + " |\n")
	}
	output.WriteString("\n## Exact-Scope Decision Boundary\n\n")
	output.WriteString("- Revenue aliases are conditionally supported only for consolidated, dimensionless facts in the active filing\n")
	output.WriteString("  chain. A valid canonical fact wins for the same issuer and period; an alias may bridge an issuer's\n")
	output.WriteString("  documented taxonomy transition across periods.\n")
	output.WriteString("- Amazon's alias is conditionally supported only as gross cash purchases of property and equipment. A derived\n")
	output.WriteString("  result must be labeled `simple FCF`; it is not Amazon-reported FCF, net capex, FCFF, or unrestricted\n")
	output.WriteString("  peer-comparable capex.\n")
	output.WriteString("- Qualcomm, NVIDIA, and Arista are supported only for explicitly labeled contextual displays. Their arithmetic\n")
	output.WriteString("  may be called a company-reported reinvestment intensity or residual-cash proxy, never canonical capex,\n")
	output.WriteString("  `simple FCF`, FCFF, a winner, a ranking, or a direct relative conclusion.\n")
	output.WriteString("- Every mapping fails closed if its filing chain, issuer language, dimensions, period, unit, currency, sign,\n")
	output.WriteString("  or accounting perimeter changes.\n\n")
	if namedDecisionAccepted {
		output.WriteString("## Named Reviewer Record\n\n")
		output.WriteString("- Reviewer: `" + namedProfessionalReviewer + "`\n")
		output.WriteString("- Qualification: `" + namedProfessionalReviewerQualification + "`\n")
		output.WriteString("- Disposition: `conditionally_accepted`\n")
		output.WriteString("- UTC timestamp: `" + namedProfessionalReviewTimestamp + "`\n")
		output.WriteString("- Scope: AR-37-01 through AR-37-09 at the exact registry content hash above\n")
		output.WriteString("- Record locator: explicit declaration in the shared Codex task, retained in the private review record\n")
		output.WriteString("- Boundary: this is not an independent audit opinion, legal opinion, investment recommendation, or\n")
		output.WriteString("  professional assurance engagement. Every use beyond the documented boundaries is rejected.\n\n")
	}
	output.WriteString("## Runtime Release Gates\n\n")
	if namedDecisionAccepted {
		output.WriteString("1. COMPLETE: the named reviewer recorded qualification, exact registry content hash (`registry_sha256`;\n")
		output.WriteString("   self-field excluded), item-level decisions, conditions, timestamp, and stable record locator.\n")
	} else {
		output.WriteString("1. A named reviewer must record qualification, exact registry content hash (`registry_sha256`; self-field\n")
		output.WriteString("   excluded), item-level decisions, conditions, timestamp, and stable record locator outside runtime inputs.\n")
	}
	if machineDecisionActive {
		output.WriteString("2. COMPLETE: runtime selection is registry- and perimeter-aware, preserves canonical precedence by period,\n")
		output.WriteString("   and carries exact per-input authority and product labels into every receipt.\n")
		output.WriteString("3. COMPLETE: context-only outputs are mechanically excluded from scores, winners, rankings, and relative conclusions.\n")
		output.WriteString("4. COMPLETE: company, 190-pair, mutation, Numerical Silence, and independent-reference gates passed on the\n")
		output.WriteString("   post-decision candidate. Release identity remains separately governed.\n\n")
		output.WriteString("The named decision and machine encoding are active only at the exact hash-bound scope above. Any registry,\n")
		output.WriteString("decision, issuer, concept, period, unit, dimension, perimeter, filing-chain, or label mismatch fails closed.\n")
	} else {
		output.WriteString("2. Runtime selection must be registry- and perimeter-aware, preserve canonical precedence by period,\n")
		output.WriteString("   and carry the exact product label into every receipt and rendered answer.\n")
		output.WriteString("3. Context-only outputs must be mechanically excluded from scores, winners, rankings, and relative conclusions.\n")
		output.WriteString("4. The complete company, pair, mutation, Numerical Silence, and independent-reference suites must pass after\n")
		output.WriteString("   the exact reviewed registry is activated.\n\n")
		output.WriteString("Until all four gates pass, every named decision and every non-canonical runtime mapping remains pending.\n")
	}
	return output.String()
}

func reviewDispositionCounts(review productscope.AccountingProfessionalReviewPacket) (int, int) {
	aliases := 0
	contextual := 0
	for _, item := range review.Items {
		switch item.ProposedDisposition {
		case productscope.AccountingReviewedAlias:
			aliases++
		case productscope.AccountingContextOnly:
			contextual++
		}
	}
	return aliases, contextual
}

func technicalReviewOutcome(item productscope.AccountingProfessionalReviewItem) string {
	switch item.ProposedDisposition {
	case productscope.AccountingReviewedAlias:
		return "Conditionally support exact issuer-specific alias; reject broader use"
	case productscope.AccountingContextOnly:
		return "Support contextual arithmetic only; reject canonical classification and comparative or ranking use"
	default:
		return "Reject"
	}
}

func markdownTableCell(value string) string {
	value = strings.ReplaceAll(value, "|", "/")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
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
