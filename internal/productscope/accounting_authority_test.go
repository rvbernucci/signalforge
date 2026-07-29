package productscope

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rvbernucci/signalforge/internal/data"
)

func TestTechnology20AccountingAuthorityIsCompleteAndOrderInvariant(t *testing.T) {
	catalog, metrics, facts, filings, asOf, hashes := canonicalAccountingAuthorityFixture(t)
	first, err := BuildTechnology20AccountingAuthority(catalog, metrics, facts, filings, asOf, hashes)
	if err != nil {
		t.Fatal(err)
	}
	if first.Manifest.Companies != 20 || first.Manifest.Inputs != 160 ||
		len(first.Packets) != 20 || len(first.Exceptions.Exceptions) != 0 ||
		len(first.Review.Items) != 0 ||
		first.Manifest.DispositionCount[AccountingCanonical] != 160 {
		t.Fatalf("unexpected complete build: %+v", first.Manifest)
	}
	for companyID := range metrics {
		slices.Reverse(metrics[companyID])
	}
	slices.Reverse(filings)
	second, err := BuildTechnology20AccountingAuthority(catalog, metrics, facts, filings, asOf, hashes)
	if err != nil {
		t.Fatal(err)
	}
	if second.Manifest.ManifestSHA256 != first.Manifest.ManifestSHA256 ||
		second.Exceptions.ReportSHA256 != first.Exceptions.ReportSHA256 ||
		second.Review.PacketSHA256 != first.Review.PacketSHA256 {
		t.Fatal("accounting authority changed under input-order permutation")
	}
	for index := range first.Packets {
		if first.Packets[index].PacketSHA256 != second.Packets[index].PacketSHA256 {
			t.Fatalf("packet %d changed under input-order permutation", index)
		}
	}
}

func TestAccountingSourceCandidateFailsClosedAtEveryMechanicalGate(t *testing.T) {
	asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	available := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	metric := data.NormalizedMetric{
		MetricID: "metric-revenue", CompanyID: "sec-cik:0000789019",
		CanonicalMetric: "revenue", PeriodStart: start, PeriodEnd: end,
		PeriodType: "duration", Value: "100", Unit: "USD", Currency: "USD",
		SourceFactIDs: []string{"fact-revenue"}, TransformationID: "normalize/v1",
		NormalizationPolicy: "point-in-time/v1", ComparabilityStatus: "standardized",
		SourceAvailableAt: available, ComputedAt: asOf,
	}
	filing := testAnnualFiling(metric.CompanyID, 1, 2025, available, asOf)
	fact := data.ReportedFact{
		FactID: "fact-revenue", FilingID: filing.FilingID, CompanyID: metric.CompanyID,
		Taxonomy: "us-gaap", Concept: "RevenueFromContractWithCustomerExcludingAssessedTax",
		Label: "Revenue", Value: metric.Value, Unit: metric.Unit, StartDate: &start, EndDate: &end,
		FiscalYear: 2025, FiscalPeriod: "FY", FormType: "10-K",
		SourceContextID: "context-revenue", SourceLocator: "companyfacts:revenue/1",
		AvailableAt: available, RetrievedAt: asOf,
	}
	definition, _ := accountingInputDefinition("revenue")
	validState := filingAuthorityState{
		Filing: filing, Valid: true, Active: true,
		AmendmentChain: []string{filing.AccessionNumber},
	}
	if _, reason := accountingSourceCandidate(
		metric, fact, map[string]filingAuthorityState{filing.FilingID: validState}, definition, asOf,
	); reason != "" {
		t.Fatalf("valid source rejected: %s", reason)
	}
	tests := []struct {
		name, want string
		mutate     func(*data.NormalizedMetric, *data.ReportedFact, map[string]filingAuthorityState)
	}{
		{
			name: "look ahead", want: "look_ahead",
			mutate: func(metric *data.NormalizedMetric, fact *data.ReportedFact, _ map[string]filingAuthorityState) {
				metric.SourceAvailableAt = asOf.Add(time.Hour)
				metric.ComputedAt = asOf.Add(2 * time.Hour)
				fact.AvailableAt = asOf.Add(time.Hour)
				fact.RetrievedAt = asOf.Add(2 * time.Hour)
			},
		},
		{
			name: "stale", want: "stale_metric_period",
			mutate: func(metric *data.NormalizedMetric, fact *data.ReportedFact, _ map[string]filingAuthorityState) {
				oldStart, oldEnd := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
				metric.PeriodStart, metric.PeriodEnd = oldStart, oldEnd
				fact.StartDate, fact.EndDate = &oldStart, &oldEnd
			},
		},
		{
			name: "unit", want: "identity_unit_or_currency_mismatch",
			mutate: func(metric *data.NormalizedMetric, _ *data.ReportedFact, _ map[string]filingAuthorityState) {
				metric.Currency = "EUR"
			},
		},
		{
			name: "dimension", want: "dimensioned_fact",
			mutate: func(_ *data.NormalizedMetric, fact *data.ReportedFact, _ map[string]filingAuthorityState) {
				fact.Dimensions = map[string]string{"segment": "cloud"}
			},
		},
		{
			name: "scale", want: "nonzero_scale",
			mutate: func(_ *data.NormalizedMetric, fact *data.ReportedFact, _ map[string]filingAuthorityState) {
				fact.Scale = 3
			},
		},
		{
			name: "quarter", want: "non_annual_form",
			mutate: func(_ *data.NormalizedMetric, fact *data.ReportedFact, _ map[string]filingAuthorityState) {
				fact.FormType = "10-Q"
			},
		},
		{
			name: "period", want: "annual_period_alignment_failed",
			mutate: func(_ *data.NormalizedMetric, fact *data.ReportedFact, _ map[string]filingAuthorityState) {
				wrong := end.AddDate(0, 0, -1)
				fact.EndDate = &wrong
			},
		},
		{
			name: "missing filing", want: "filing_authority_missing_or_invalid",
			mutate: func(_ *data.NormalizedMetric, _ *data.ReportedFact, states map[string]filingAuthorityState) {
				delete(states, filing.FilingID)
			},
		},
		{
			name: "superseded filing", want: "filing_superseded",
			mutate: func(_ *data.NormalizedMetric, _ *data.ReportedFact, states map[string]filingAuthorityState) {
				state := states[filing.FilingID]
				state.Active = false
				states[filing.FilingID] = state
			},
		},
		{
			name: "source hash", want: "source_hash_or_origin_invalid",
			mutate: func(_ *data.NormalizedMetric, _ *data.ReportedFact, states map[string]filingAuthorityState) {
				state := states[filing.FilingID]
				state.Filing.ContentSHA256 = "not-a-hash"
				states[filing.FilingID] = state
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidateMetric, candidateFact := metric, fact
			states := map[string]filingAuthorityState{filing.FilingID: validState}
			test.mutate(&candidateMetric, &candidateFact, states)
			if _, reason := accountingSourceCandidate(
				candidateMetric, candidateFact, states, definition, asOf,
			); reason != test.want {
				t.Fatalf("reason = %q, want %q", reason, test.want)
			}
		})
	}
}

func TestAccountingAuthorityEnforcesCapexSignPolicy(t *testing.T) {
	asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	start, end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	available := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	companyID := "sec-cik:0000789019"
	filing := testAnnualFiling(companyID, 1, 2025, available, asOf)
	metric := data.NormalizedMetric{
		MetricID: "metric-capex", CompanyID: companyID, CanonicalMetric: "capital_expenditure",
		PeriodStart: start, PeriodEnd: end, PeriodType: "duration", Value: "-10",
		Unit: "USD", Currency: "USD", SourceFactIDs: []string{"fact-capex"},
		TransformationID: "normalize/v1", NormalizationPolicy: "point-in-time/v1",
		ComparabilityStatus: "standardized", SourceAvailableAt: available, ComputedAt: asOf,
	}
	fact := data.ReportedFact{
		FactID: "fact-capex", FilingID: filing.FilingID, CompanyID: companyID,
		Taxonomy: "us-gaap", Concept: "PaymentsToAcquirePropertyPlantAndEquipment",
		Value: "-10", Unit: "USD", StartDate: &start, EndDate: &end,
		FormType: "10-K", SourceContextID: "context-capex", SourceLocator: "fixture:capex",
		AvailableAt: available, RetrievedAt: asOf,
	}
	definition, _ := accountingInputDefinition("capital_expenditure")
	_, reason := accountingSourceCandidate(
		metric, fact,
		map[string]filingAuthorityState{
			filing.FilingID: {
				Filing: filing, Valid: true, Active: true,
				AmendmentChain: []string{filing.AccessionNumber},
			},
		},
		definition, asOf,
	)
	if reason != "numeric_or_sign_policy_failed" {
		t.Fatalf("reason = %q, want numeric_or_sign_policy_failed", reason)
	}
}

func TestAccountingFilingAuthoritySupersedesOriginalWithAmendment(t *testing.T) {
	asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	parent := testAnnualFiling("sec-cik:0000789019", 1, 2025, time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC), asOf)
	amendment := testAnnualFiling("sec-cik:0000789019", 2, 2025, time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC), asOf)
	amendment.FormType = "10-K/A"
	amendment.AmendsFilingID = parent.FilingID
	states, exceptions := buildFilingAuthorityStates([]data.Filing{amendment, parent}, asOf)
	if len(exceptions) != 0 {
		t.Fatalf("unexpected amendment exceptions: %+v", exceptions)
	}
	if states[parent.FilingID].Active || !states[amendment.FilingID].Active {
		t.Fatal("amendment chain did not supersede the original filing")
	}
	if got := states[amendment.FilingID].AmendmentChain; len(got) != 2 ||
		got[0] != amendment.AccessionNumber || got[1] != parent.AccessionNumber {
		t.Fatalf("amendment chain = %+v", got)
	}
}

func TestAccountingAuthorityRejectsEquallyAuthoritativeDuplicateFacts(t *testing.T) {
	available := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	values := []AccountingSourceAuthority{
		{FactID: "fact-a", PeriodStart: &start, PeriodEnd: end, PeriodType: "duration", AvailableAt: available},
		{FactID: "fact-b", PeriodStart: &start, PeriodEnd: end, PeriodType: "duration", AvailableAt: available},
	}
	active, ambiguous := activeAnnualSources(values)
	if len(active) != 0 || ambiguous != 2 {
		t.Fatalf("active=%d ambiguous=%d, want 0/2", len(active), ambiguous)
	}
}

func TestAccountingAuthorityCannotValidateWithMissingCompanyPacket(t *testing.T) {
	catalog, metrics, facts, filings, asOf, hashes := canonicalAccountingAuthorityFixture(t)
	build, err := BuildTechnology20AccountingAuthority(catalog, metrics, facts, filings, asOf, hashes)
	if err != nil {
		t.Fatal(err)
	}
	build.Packets = build.Packets[:19]
	if err := ValidateTechnology20AccountingBuild(build); err == nil {
		t.Fatal("incomplete packet population unexpectedly validated")
	}
}

func TestFinancialActivationRejectsPipelineStandardizedUnreviewedConcept(t *testing.T) {
	asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	start, end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	metric := data.NormalizedMetric{
		MetricID: "metric-sales", CompanyID: "sec-cik:0000789019", CanonicalMetric: "revenue",
		PeriodStart: start, PeriodEnd: end, PeriodType: "duration", Value: "100",
		Unit: "USD", Currency: "USD", SourceFactIDs: []string{"fact-sales"},
		TransformationID: "normalize/v1", NormalizationPolicy: "point-in-time/v1",
		ComparabilityStatus: "standardized", SourceAvailableAt: asOf.Add(-time.Hour), ComputedAt: asOf,
	}
	facts := map[string]data.ReportedFact{
		"fact-sales": {
			FactID: "fact-sales", CompanyID: metric.CompanyID, Taxonomy: "us-gaap",
			Concept: "SalesRevenueNet", Value: metric.Value, Unit: metric.Unit,
			StartDate: &start, EndDate: &end, FormType: "10-K",
			AvailableAt: metric.SourceAvailableAt, RetrievedAt: asOf,
		},
	}
	report, err := BuildCompanyFinancialActivation(metric.CompanyID, []data.NormalizedMetric{metric}, facts, asOf, "test-commit")
	if err != nil {
		t.Fatal(err)
	}
	if _, exposed := report.SourceConcepts["revenue"]; exposed ||
		report.Excluded["unreviewed_semantic_mapping"] != 1 {
		t.Fatalf("pipeline label bypassed registry: concepts=%+v excluded=%+v", report.SourceConcepts, report.Excluded)
	}
}

func canonicalAccountingAuthorityFixture(t *testing.T) (
	PublicCatalog,
	map[string][]data.NormalizedMetric,
	map[string]data.ReportedFact,
	[]data.Filing,
	time.Time,
	AccountingAuthoritySourceHashes,
) {
	t.Helper()
	catalog := validPublicCatalogForSummary(t)
	asOf := catalog.AsOf.Add(time.Hour)
	metrics := map[string][]data.NormalizedMetric{}
	facts := map[string]data.ReportedFact{}
	filings := []data.Filing{}
	for companyIndex, company := range catalog.Companies {
		for _, year := range []int{2024, 2025} {
			available := time.Date(year+1, 2, 15, 12, 0, 0, 0, time.UTC)
			filing := testAnnualFiling(company.CompanyID, companyIndex*10+year-2023, year, available, asOf)
			filings = append(filings, filing)
			for inputIndex, definition := range canonicalAccountingInputs {
				start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
				end := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)
				if definition.PeriodType == "instant" {
					start = end
				}
				id := fmt.Sprintf("%s-%s-%d", company.PrimaryTicker, definition.CanonicalInput, year)
				value := fmt.Sprintf("%d", 1000+companyIndex*100+inputIndex*10+year-2024)
				metric := data.NormalizedMetric{
					MetricID: "metric-" + id, CompanyID: company.CompanyID,
					CanonicalMetric: definition.CanonicalInput, PeriodStart: start, PeriodEnd: end,
					PeriodType: definition.PeriodType, Value: value, Unit: "USD", Currency: "USD",
					SourceFactIDs: []string{"fact-" + id}, TransformationID: "normalize/v1",
					NormalizationPolicy: "point-in-time/v1", ComparabilityStatus: "standardized",
					SourceAvailableAt: available, ComputedAt: asOf,
				}
				fact := data.ReportedFact{
					FactID: "fact-" + id, FilingID: filing.FilingID, CompanyID: company.CompanyID,
					Taxonomy: "us-gaap", Concept: definition.TaxonomyConcept,
					Label: definition.TaxonomyConcept, Value: value, Unit: "USD",
					FiscalYear: year, FiscalPeriod: "FY", FormType: "10-K",
					SourceContextID: "context-" + id, SourceLocator: "fixture:" + id,
					AvailableAt: available, RetrievedAt: asOf,
				}
				if definition.PeriodType == "instant" {
					fact.InstantDate = &end
				} else {
					fact.StartDate, fact.EndDate = &start, &end
				}
				metrics[company.CompanyID] = append(metrics[company.CompanyID], metric)
				facts[fact.FactID] = fact
			}
		}
	}
	hashes := AccountingAuthoritySourceHashes{
		CatalogSHA256: strings.Repeat("a", 64), MetricsSHA256: strings.Repeat("b", 64),
		FactsSHA256: strings.Repeat("c", 64), FilingsSHA256: strings.Repeat("d", 64),
	}
	return catalog, metrics, facts, filings, asOf, hashes
}

func testAnnualFiling(
	companyID string,
	sequence, year int,
	available, retrieved time.Time,
) data.Filing {
	cik := strings.TrimPrefix(companyID, "sec-cik:")
	accession := fmt.Sprintf("%s-%02d-%06d", cik, year%100, sequence)
	return data.Filing{
		FilingID: "sec-filing:" + accession, CompanyID: companyID, AccessionNumber: accession,
		FormType: "10-K", ReportPeriodEnd: time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC),
		FiledAt: available.Add(-time.Hour), AcceptedAt: available.Add(-time.Minute),
		PublishedAt: available, Taxonomy: "us-gaap", TaxonomyVersion: "2025",
		PrimaryDocument: "annual.htm", PrimaryDocTitle: "Annual report", IsXBRL: true,
		IsInlineXBRL: true, SourceRecordID: "source-" + accession,
		SourceURI:     "https://data.sec.gov/submissions/CIK" + cik + ".json",
		ContentSHA256: strings.Repeat("e", 64), RetrievedAt: retrieved,
		ExtractorVersion: "fixture/v1",
	}
}
