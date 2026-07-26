package productscope

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPeerEvaluationNeverPromotesCandidateLanes(t *testing.T) {
	catalog := loadPublicCatalogFixture(t)
	reports := loadCompanyFinancialReports(t, catalog)
	suite, err := BuildPeerEvaluationSuite(catalog, reports, catalog.AsOf.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	for _, lane := range suite.Lanes {
		if lane.Promoted {
			t.Fatalf("candidate lane %q was over-promoted", lane.LaneID)
		}
		if len(lane.Withheld) == 0 {
			t.Fatalf("lane %q lacks an explicit not-comparable or unavailable demonstration", lane.LaneID)
		}
	}
}

func loadCompanyFinancialReports(t *testing.T, catalog PublicCatalog) map[string]CompanyFinancialActivation {
	t.Helper()
	result := map[string]CompanyFinancialActivation{}
	for _, company := range catalog.Companies {
		payload, err := os.ReadFile(filepath.Join("..", "..", "..", "experiments", "sprint32", "financial-activation", company.PrimaryTicker+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var report CompanyFinancialActivation
		if err := json.Unmarshal(payload, &report); err != nil {
			t.Fatal(err)
		}
		result[company.CompanyID] = report
	}
	return result
}
