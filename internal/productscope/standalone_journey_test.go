package productscope

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStandaloneJourneyPopulationIsBalancedAndAuthorityBound(t *testing.T) {
	catalog := loadPublicCatalogFixture(t)
	financials := loadPublicFinancialSummaryFixture(t)
	development, sealed, err := BuildStandaloneJourneySuites(catalog, financials)
	if err != nil {
		t.Fatal(err)
	}
	if len(development.Cases) != 80 || len(sealed.Cases) != 40 {
		t.Fatalf("population = %d development / %d sealed", len(development.Cases), len(sealed.Cases))
	}
	for _, item := range append(development.Cases, sealed.Cases...) {
		if item.ExpectedReceipts == nil || item.ExpectedAbstentions == nil {
			t.Fatalf("authority outcomes must serialize as arrays: %+v", item)
		}
		if item.QuestionID == "valuation-macro" && item.ExpectedDisposition != "typed_abstention" {
			t.Fatalf("valuation was released without price/macro authority: %+v", item)
		}
	}
}

func loadPublicCatalogFixture(t *testing.T) PublicCatalog {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "productscope", "technology20-catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result PublicCatalog
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func loadPublicFinancialSummaryFixture(t *testing.T) PublicFinancialSummary {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "productscope", "technology20-financial-summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result PublicFinancialSummary
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
