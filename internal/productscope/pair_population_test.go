package productscope

import (
	"testing"
	"time"

	"github.com/rvbernucci/signalforge/internal/contracts"
)

func TestTechnology20PairPopulationGeneratesAllUnorderedPairs(t *testing.T) {
	catalog := loadPublicCatalogFixture(t)
	reports := loadCompanyFinancialReports(t, catalog)
	population, err := BuildTechnology20PairPopulation(
		catalog,
		reports,
		catalog.AsOf.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(population.Pairs) != 190 {
		t.Fatalf("pairs = %d, want 190", len(population.Pairs))
	}
	seen := map[string]bool{}
	for _, pair := range population.Pairs {
		key := pair.CompanyIDs[0] + "|" + pair.CompanyIDs[1]
		if seen[key] {
			t.Fatalf("duplicate pair %s", key)
		}
		seen[key] = true
		if len(pair.Receipts)+len(pair.Abstentions) != len(companyOperationSpecs) {
			t.Fatalf("pair %s has incomplete metric coverage", pair.LaneID)
		}
	}
}

func TestTechnology20PairPopulationRejectsMutation(t *testing.T) {
	catalog := loadPublicCatalogFixture(t)
	reports := loadCompanyFinancialReports(t, catalog)
	population, err := BuildTechnology20PairPopulation(
		catalog,
		reports,
		catalog.AsOf.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	population.Pairs[0].CompanyIDs[0] = population.Pairs[0].CompanyIDs[1]
	if err := ValidateTechnology20PairPopulation(population); err == nil {
		t.Fatal("mutated pair population passed validation")
	}
}

func TestRegisteredComparisonAuthorityRejectsMutation(t *testing.T) {
	companyID := "sec-cik:0000789019"
	operationID := "financial.operating_margin"
	buildOperand := func() contracts.MetricComparisonOperand {
		result := contracts.MetricComparisonOperand{
			CompanyID: companyID, CanonicalMetricID: operationID,
			OutputClass:  AccountingOutputAuthoritative,
			ProductLabel: operationID, PairRankingEligible: true,
		}
		for index, input := range []struct {
			id        string
			canonical string
			concept   string
		}{
			{id: "operating_income", canonical: "operating_income", concept: "OperatingIncomeLoss"},
			{id: "revenue", canonical: "revenue", concept: "RevenueFromContractWithCustomerExcludingAssessedTax"},
		} {
			mapping := ResolveAccountingMapping(
				runtimeAccountingAuthorityRegistry,
				companyID,
				input.canonical,
				"us-gaap",
				input.concept,
			)
			result.AccountingInputs = append(
				result.AccountingInputs,
				contracts.AccountingInputComparisonAuthority{
					InputID: input.id, CanonicalInput: input.canonical,
					MappingKey: mapping.MappingKey, TaxonomyConcept: mapping.TaxonomyConcept,
					AccountingPerimeter: mapping.AccountingPerimeter,
					Disposition:         string(mapping.Disposition), ProductLabel: mapping.ProductLabel,
					PairRankingEligible: true,
				},
			)
			if index > 0 {
				result.AccountingPerimeter += ";"
			}
			result.AccountingPerimeter += input.id + "=" + mapping.AccountingPerimeter
		}
		return result
	}
	baseline := buildOperand()
	if err := validateRegisteredComparisonAuthority(baseline); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*contracts.MetricComparisonOperand){
		"mapping key": func(operand *contracts.MetricComparisonOperand) {
			operand.AccountingInputs[0].MappingKey = "unregistered-mapping"
		},
		"product label": func(operand *contracts.MetricComparisonOperand) {
			operand.ProductLabel = "generic financial metric"
		},
		"pair eligibility": func(operand *contracts.MetricComparisonOperand) {
			operand.PairRankingEligible = !operand.PairRankingEligible
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := buildOperand()
			mutate(&candidate)
			if err := validateRegisteredComparisonAuthority(candidate); err == nil {
				t.Fatal("accounting-authority mutation passed validation")
			}
		})
	}
}
