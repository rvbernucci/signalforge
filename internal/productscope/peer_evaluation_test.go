package productscope

import (
	"testing"
	"time"

	"github.com/rvbernucci/signalforge/internal/contracts"
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

func TestReviewedOperationMappingIDIsBoundedToAuthorizedRevenueConcepts(t *testing.T) {
	receipt := contracts.CalculationReceipt{
		OperationID: "financial.operating_margin",
		NormalizedInputs: []contracts.EngineInput{
			{InputID: "operating_income"},
			{InputID: "revenue"},
		},
	}
	reviewedID := ReviewedRevenueAliasPolicy +
		":OperatingIncomeLoss+RevenueFromContractWithCustomerExcludingAssessedTax"
	for _, test := range []struct {
		name      string
		companyID string
		concept   string
		want      string
	}{
		{
			name: "canonical revenue", companyID: "sec-cik:0000789019",
			concept: "RevenueFromContractWithCustomerExcludingAssessedTax",
			want:    reviewedID,
		},
		{
			name: "reviewed Alphabet alias", companyID: "sec-cik:0001652044",
			concept: "Revenues",
		},
		{
			name: "reviewed NVIDIA alias", companyID: "sec-cik:0001045810",
			concept: "Revenues",
		},
		{
			name: "unreviewed issuer alias", companyID: "sec-cik:0000789019",
			concept: "Revenues",
		},
		{
			name: "unreviewed alternate concept", companyID: "sec-cik:0001652044",
			concept: "SalesRevenueNet",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := CompanyFinancialActivation{
				CompanyID: test.companyID,
				SourceConcepts: map[string][]string{
					"operating_income": {"OperatingIncomeLoss"},
					"revenue":          {test.concept},
				},
			}
			concepts := operationConcepts(report, receipt, false)
			if got := reviewedOperationMappingID(report, receipt, concepts); got != test.want {
				t.Fatalf("mapping ID = %q, want %q", got, test.want)
			}
		})
	}

	nonRevenue := contracts.CalculationReceipt{OperationID: "financial.cash_conversion"}
	report := CompanyFinancialActivation{
		CompanyID:      "sec-cik:0001652044",
		SourceConcepts: map[string][]string{"revenue": {"Revenues"}},
	}
	if got := reviewedOperationMappingID(report, nonRevenue, nil); got != "" {
		t.Fatalf("non-revenue operation received mapping ID %q", got)
	}
}

func TestContextOnlyCapexCannotInheritRevenueEquivalence(t *testing.T) {
	receipt := contracts.CalculationReceipt{
		OperationID: "financial.capex_intensity",
		NormalizedInputs: []contracts.EngineInput{
			{InputID: "capital_expenditure"},
			{InputID: "revenue"},
		},
	}
	nvidia := CompanyFinancialActivation{
		CompanyID: "sec-cik:0001045810",
		SourceConcepts: map[string][]string{
			"revenue": {"Revenues"},
		},
		ContextualConcepts: map[string][]string{
			"capital_expenditure": {"PaymentsToAcquireProductiveAssets"},
		},
	}
	amd := CompanyFinancialActivation{
		CompanyID: "sec-cik:0000002488",
		SourceConcepts: map[string][]string{
			"capital_expenditure": {"PaymentsToAcquirePropertyPlantAndEquipment"},
			"revenue":             {"RevenueFromContractWithCustomerExcludingAssessedTax"},
		},
	}
	nvidiaConcepts := operationConcepts(nvidia, receipt, true)
	amdConcepts := operationConcepts(amd, receipt, false)
	nvidiaMapping := reviewedOperationMappingID(nvidia, receipt, nvidiaConcepts)
	amdMapping := reviewedOperationMappingID(amd, receipt, amdConcepts)
	if nvidiaMapping != "" || amdMapping == "" {
		t.Fatalf("pending NVIDIA mapping must remain withheld: NVIDIA=%q AMD=%q", nvidiaMapping, amdMapping)
	}
}

func loadCompanyFinancialReports(t *testing.T, catalog PublicCatalog) map[string]CompanyFinancialActivation {
	t.Helper()
	result := map[string]CompanyFinancialActivation{}
	for _, company := range catalog.Companies {
		report, err := BuildCompanyFinancialActivation(
			company.CompanyID,
			nil,
			nil,
			catalog.AsOf,
			"clean-clone-test",
		)
		if err != nil {
			t.Fatal(err)
		}
		result[company.CompanyID] = report
	}
	return result
}
