package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rvbernucci/signalforge/internal/productscope"
)

func main() {
	catalogPath := flag.String(
		"catalog",
		"fixtures/productscope/technology20-catalog.json",
		"Technology 20 catalog",
	)
	reportsDir := flag.String(
		"reports-dir",
		"",
		"Directory containing the twenty ticker-named financial activation reports",
	)
	outputPath := flag.String("output", "", "Pair population JSON output")
	generatedAtRaw := flag.String("generated-at", "", "Generation time in RFC3339")
	flag.Parse()
	if *reportsDir == "" || *outputPath == "" || *generatedAtRaw == "" {
		fatal(fmt.Errorf("--reports-dir, --output, and --generated-at are required"))
	}
	generatedAt, err := time.Parse(time.RFC3339, *generatedAtRaw)
	if err != nil {
		fatal(err)
	}
	var catalog productscope.PublicCatalog
	readJSON(*catalogPath, &catalog)
	reports := map[string]productscope.CompanyFinancialActivation{}
	for _, company := range catalog.Companies {
		var report productscope.CompanyFinancialActivation
		readJSON(filepath.Join(*reportsDir, company.PrimaryTicker+".json"), &report)
		reports[company.CompanyID] = report
	}
	population, err := productscope.BuildTechnology20PairPopulation(
		catalog,
		reports,
		generatedAt,
	)
	if err != nil {
		fatal(err)
	}
	payload, err := json.MarshalIndent(population, "", "  ")
	if err != nil {
		fatal(err)
	}
	payload = append(payload, '\n')
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		fatal(err)
	}
	temporary := *outputPath + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o644); err != nil {
		fatal(err)
	}
	if err := os.Rename(temporary, *outputPath); err != nil {
		fatal(err)
	}
	fmt.Printf(
		"pair population: %d reports, %d pairs, sha256=%s\n",
		len(population.CompanyReports),
		len(population.Pairs),
		population.PopulationSHA256,
	)
}

func readJSON(path string, target any) {
	payload, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
