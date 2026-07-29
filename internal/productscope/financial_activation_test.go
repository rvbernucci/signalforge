package productscope

import (
	"testing"
	"time"

	"github.com/rvbernucci/signalforge/internal/data"
)

func TestCompanyFinancialActivationReleasesAlignedReceiptsAndAbstainsOtherwise(t *testing.T) {
	asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	companyID := "sec-cik:0000789019"
	annual := func(metric, value string, year int) data.NormalizedMetric {
		start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)
		return data.NormalizedMetric{
			MetricID: "metric-" + metric + value, CompanyID: companyID, CanonicalMetric: metric,
			PeriodStart: start, PeriodEnd: end, PeriodType: "duration", Value: value,
			Unit: "USD", Currency: "USD", SourceFactIDs: []string{"fact-" + metric + value},
			TransformationID: "normalize/v1", NormalizationPolicy: "point-in-time/v1",
			ComparabilityStatus: "standardized", SourceAvailableAt: end.Add(60 * 24 * time.Hour),
			ComputedAt: asOf,
		}
	}
	instant := func(metric, value string) data.NormalizedMetric {
		end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		item := annual(metric, value, 2025)
		item.PeriodStart, item.PeriodEnd, item.PeriodType = end, end, "instant"
		return item
	}
	metrics := []data.NormalizedMetric{
		annual("revenue", "100", 2024), annual("revenue", "120", 2025),
		annual("operating_income", "24", 2025), annual("operating_cash_flow", "30", 2025),
		annual("capital_expenditure", "10", 2025), annual("net_income", "20", 2025),
		instant("total_assets", "100"), instant("total_liabilities", "60"), instant("stockholders_equity", "40"),
	}
	facts := map[string]data.ReportedFact{}
	concepts := map[string]string{
		"revenue":             "RevenueFromContractWithCustomerExcludingAssessedTax",
		"operating_income":    "OperatingIncomeLoss",
		"operating_cash_flow": "NetCashProvidedByUsedInOperatingActivities",
		"capital_expenditure": "PaymentsToAcquirePropertyPlantAndEquipment",
		"net_income":          "NetIncomeLoss",
		"total_assets":        "Assets",
		"total_liabilities":   "Liabilities",
		"stockholders_equity": "StockholdersEquity",
	}
	for _, metric := range metrics {
		fact := data.ReportedFact{
			FactID: metric.SourceFactIDs[0], CompanyID: companyID, Value: metric.Value,
			Taxonomy: "us-gaap", Concept: concepts[metric.CanonicalMetric],
			Unit: metric.Unit, FormType: "10-K", AvailableAt: metric.SourceAvailableAt,
			RetrievedAt: metric.ComputedAt,
		}
		if metric.PeriodType == "instant" {
			value := metric.PeriodEnd
			fact.InstantDate = &value
		} else {
			start, end := metric.PeriodStart, metric.PeriodEnd
			fact.StartDate, fact.EndDate = &start, &end
		}
		facts[fact.FactID] = fact
	}
	report, err := BuildCompanyFinancialActivation(companyID, metrics, facts, asOf, "test-commit")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Receipts) != len(companyOperationSpecs) {
		t.Fatalf("receipts = %d, want %d; abstentions=%+v", len(report.Receipts), len(companyOperationSpecs), report.Abstentions)
	}
	if len(report.Abstentions) != len(unavailableCompanyOperations) {
		t.Fatalf("abstentions = %d, want %d", len(report.Abstentions), len(unavailableCompanyOperations))
	}
}

func TestCompanyFinancialActivationRejectsAliasesAndLookAhead(t *testing.T) {
	asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	metric := data.NormalizedMetric{
		MetricID: "metric-alias", CompanyID: "sec-cik:0000789019", CanonicalMetric: "revenue",
		PeriodStart: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), PeriodType: "duration",
		Value: "100", Unit: "USD", Currency: "USD", SourceFactIDs: []string{"fact-alias"},
		TransformationID: "normalize/v1", NormalizationPolicy: "point-in-time/v1",
		ComparabilityStatus: "concept_alias", SourceAvailableAt: asOf.Add(-time.Hour),
		ComputedAt: asOf,
	}
	start, end := metric.PeriodStart, metric.PeriodEnd
	facts := map[string]data.ReportedFact{
		"fact-alias": {
			FactID: "fact-alias", CompanyID: metric.CompanyID, Taxonomy: "us-gaap",
			Concept: "Revenues", Value: metric.Value, Unit: metric.Unit,
			StartDate: &start, EndDate: &end, FormType: "10-K",
			AvailableAt: metric.SourceAvailableAt, RetrievedAt: asOf,
		},
	}
	report, err := BuildCompanyFinancialActivation(metric.CompanyID, []data.NormalizedMetric{metric}, facts, asOf, "test-commit")
	if err != nil {
		t.Fatal(err)
	}
	if report.Excluded["unreviewed_semantic_mapping"] != 1 || len(report.Receipts) != 0 {
		t.Fatalf("alias was not excluded: %+v", report)
	}
}

func TestCompanyFinancialActivationWithholdsAliasesPendingProfessionalDecision(t *testing.T) {
	asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name      string
		companyID string
		concept   string
		accepted  bool
	}{
		{name: "Alphabet Revenues", companyID: "sec-cik:0001652044", concept: "Revenues", accepted: false},
		{name: "NVIDIA Revenues", companyID: "sec-cik:0001045810", concept: "Revenues", accepted: false},
		{name: "Microsoft Revenues", companyID: "sec-cik:0000789019", concept: "Revenues", accepted: false},
		{name: "Alphabet SalesRevenueNet", companyID: "sec-cik:0001652044", concept: "SalesRevenueNet", accepted: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			factID := "fact-reviewed-alias"
			metric := data.NormalizedMetric{
				MetricID: "metric-reviewed-alias", CompanyID: test.companyID, CanonicalMetric: "revenue",
				PeriodStart: start, PeriodEnd: end, PeriodType: "duration",
				Value: "100", Unit: "USD", Currency: "USD", SourceFactIDs: []string{factID},
				TransformationID: "normalize/v1", NormalizationPolicy: "point-in-time/v1",
				ComparabilityStatus: "concept_alias", SourceAvailableAt: asOf.Add(-time.Hour),
				ComputedAt: asOf,
			}
			facts := map[string]data.ReportedFact{
				factID: {
					FactID: factID, CompanyID: test.companyID, Taxonomy: "us-gaap",
					Concept: test.concept, Value: metric.Value, Unit: metric.Unit,
					StartDate: &start, EndDate: &end, FormType: "10-K",
					AvailableAt: metric.SourceAvailableAt, RetrievedAt: asOf,
				},
			}
			report, err := BuildCompanyFinancialActivation(test.companyID, []data.NormalizedMetric{metric}, facts, asOf, "test-commit")
			if err != nil {
				t.Fatal(err)
			}
			_, exposed := report.SourceConcepts["revenue"]
			if exposed != test.accepted {
				t.Fatalf("revenue authority exposed = %t, want %t; excluded=%+v", exposed, test.accepted, report.Excluded)
			}
		})
	}
}

func TestPeriodicFactAuthorityUsesLatestAuthorizedPeriodicSource(t *testing.T) {
	companyID := "sec-cik:0001652044"
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	older := time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC)
	selectedAt := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	metric := data.NormalizedMetric{
		MetricID: "metric-revenue", CompanyID: companyID, CanonicalMetric: "revenue",
		PeriodStart: start, PeriodEnd: end, PeriodType: "duration",
		Value: "350018000000", Unit: "USD", Currency: "USD",
		SourceFactIDs:    []string{"fact-standard-older", "fact-alias-selected"},
		TransformationID: "normalize/v1", NormalizationPolicy: "point-in-time/v1",
		ComparabilityStatus: "concept_alias", SourceAvailableAt: selectedAt, ComputedAt: selectedAt,
	}
	facts := map[string]data.ReportedFact{
		"fact-standard-older": {
			FactID: "fact-standard-older", CompanyID: companyID,
			Concept: "RevenueFromContractWithCustomerExcludingAssessedTax",
			Value:   metric.Value, Unit: metric.Unit, StartDate: &start, EndDate: &end,
			FormType: "10-K", AvailableAt: older, RetrievedAt: older,
		},
		"fact-alias-selected": {
			FactID: "fact-alias-selected", CompanyID: companyID, Concept: "Revenues",
			Value: metric.Value, Unit: metric.Unit, StartDate: &start, EndDate: &end,
			FormType: "10-K", AvailableAt: selectedAt, RetrievedAt: selectedAt,
		},
	}
	fact, ok := periodicFactAuthority(metric, facts)
	if !ok || fact.FactID != "fact-alias-selected" {
		t.Fatalf("selected authority = %+v, %t", fact, ok)
	}
}

func TestCompanyFinancialActivationWithholdsProductiveAssetsPendingProfessionalDecision(t *testing.T) {
	asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	companyID := "sec-cik:0001045810"
	start := time.Date(2025, 1, 27, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	availableAt := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	metric := func(id, canonical, value, status string) data.NormalizedMetric {
		return data.NormalizedMetric{
			MetricID: id, CompanyID: companyID, CanonicalMetric: canonical,
			PeriodStart: start, PeriodEnd: end, PeriodType: "duration",
			Value: value, Unit: "USD", Currency: "USD", SourceFactIDs: []string{"fact-" + id},
			TransformationID: "normalize/v1", NormalizationPolicy: "point-in-time/v1",
			ComparabilityStatus: status, SourceAvailableAt: availableAt, ComputedAt: asOf,
		}
	}
	metrics := []data.NormalizedMetric{
		metric("revenue", "revenue", "200", "concept_alias"),
		metric("ocf", "operating_cash_flow", "50", "standardized"),
		metric("productive-assets", "capital_expenditure", "10", "concept_alias"),
	}
	concepts := map[string]string{
		"revenue":           "Revenues",
		"ocf":               "NetCashProvidedByUsedInOperatingActivities",
		"productive-assets": "PaymentsToAcquireProductiveAssets",
	}
	facts := map[string]data.ReportedFact{}
	for _, item := range metrics {
		factID := item.SourceFactIDs[0]
		facts[factID] = data.ReportedFact{
			FactID: factID, CompanyID: companyID, Taxonomy: "us-gaap",
			Concept: concepts[item.MetricID], Value: item.Value, Unit: item.Unit,
			StartDate: &start, EndDate: &end, FormType: "10-K",
			AvailableAt: availableAt, RetrievedAt: asOf,
		}
	}
	report, err := BuildCompanyFinancialActivation(companyID, metrics, facts, asOf, "test-commit")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.ContextualReceipts) != 0 || len(report.ContextualConcepts) != 0 {
		t.Fatalf("pending context mapping reached product output: %+v", report)
	}
	if report.Excluded["unreviewed_semantic_mapping"] != 2 {
		t.Fatalf("pending mappings were not typed as excluded: %+v", report.Excluded)
	}
}

func TestCompanyFinancialActivationValidationRejectsUnauthorizedSourceAuthority(t *testing.T) {
	report := validFinancialActivationForValidation(t)
	report.SourceConcepts["capital_expenditure"] = []string{"PropertyPlantAndEquipment"}
	report.ReportSHA256 = financialActivationHashForTest(t, report)
	if err := ValidateCompanyFinancialActivation(report); err == nil {
		t.Fatal("expected unauthorized capex concept rejection")
	}

	report = validFinancialActivationForValidation(t)
	report.SourceForms["revenue"] = []string{"DEF 14A"}
	report.ReportSHA256 = financialActivationHashForTest(t, report)
	if err := ValidateCompanyFinancialActivation(report); err == nil {
		t.Fatal("expected non-periodic form rejection")
	}
}

func validFinancialActivationForValidation(t *testing.T) CompanyFinancialActivation {
	t.Helper()
	report, err := BuildCompanyFinancialActivation(
		"sec-cik:0000789019", nil, nil,
		time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC), "test-commit",
	)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func financialActivationHashForTest(t *testing.T, report CompanyFinancialActivation) string {
	t.Helper()
	hash, err := companyFinancialActivationHash(report)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
