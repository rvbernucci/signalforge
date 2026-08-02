package productscope

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rvbernucci/signalforge/internal/data"
)

func FuzzAccountingSourceBoundary(f *testing.F) {
	f.Add(uint8(0), "", int16(0))
	f.Add(uint8(1), "duplicate", int16(0))
	f.Add(uint8(2), "100", int16(-900))
	f.Add(uint8(3), "100", int16(1))
	f.Add(uint8(4), strings.Repeat("9", 96), int16(0))
	f.Add(uint8(5), "10", int16(0))

	f.Fuzz(func(t *testing.T, mode uint8, raw string, dayDelta int16) {
		if len(raw) > 128 {
			t.Skip()
		}

		asOf := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		available := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
		companyID := "sec-cik:0000789019"
		filing := testAnnualFiling(companyID, 1, 2025, available, asOf)
		metric := data.NormalizedMetric{
			MetricID: "metric-revenue", CompanyID: companyID,
			CanonicalMetric: "revenue", PeriodStart: start, PeriodEnd: end,
			PeriodType: "duration", Value: "100", Unit: "USD", Currency: "USD",
			SourceFactIDs: []string{"fact-revenue"}, TransformationID: "normalize/v1",
			NormalizationPolicy: "point-in-time/v1", ComparabilityStatus: "standardized",
			SourceAvailableAt: available, ComputedAt: asOf,
		}
		fact := data.ReportedFact{
			FactID: "fact-revenue", FilingID: filing.FilingID, CompanyID: companyID,
			Taxonomy: "us-gaap", Concept: "RevenueFromContractWithCustomerExcludingAssessedTax",
			Label: "Revenue", Value: "100", Unit: "USD", StartDate: &start, EndDate: &end,
			FiscalYear: 2025, FiscalPeriod: "FY", FormType: "10-K",
			SourceContextID: "context-revenue", SourceLocator: "companyfacts:revenue/1",
			AvailableAt: available, RetrievedAt: asOf,
		}
		states := map[string]filingAuthorityState{
			filing.FilingID: {
				Filing: filing, Valid: true, Active: true,
				AmendmentChain: []string{filing.AccessionNumber},
			},
		}
		definition, _ := accountingInputDefinition("revenue")
		mustReject := false

		switch mode % 6 {
		case 0:
			fact.FactID = ""
			mustReject = true
		case 1:
			values := []AccountingSourceAuthority{
				{
					FactID: "fact-a", PeriodStart: &start, PeriodEnd: end,
					PeriodType: "duration", AvailableAt: available,
				},
				{
					FactID: "fact-b-" + fmt.Sprint(dayDelta), PeriodStart: &start,
					PeriodEnd: end, PeriodType: "duration", AvailableAt: available,
				},
			}
			active, ambiguous := activeAnnualSources(values)
			if len(active) != 0 || ambiguous != 2 {
				t.Fatalf("equally authoritative duplicates escaped quarantine: active=%d ambiguous=%d", len(active), ambiguous)
			}
			return
		case 2:
			metric.PeriodStart = start.AddDate(-3, 0, 0)
			metric.PeriodEnd = end.AddDate(-3, 0, 0)
			fact.StartDate = &metric.PeriodStart
			fact.EndDate = &metric.PeriodEnd
			mustReject = true
		case 3:
			offset := int(dayDelta)
			if offset < 0 {
				offset = -offset
			}
			offset = 1 + offset%30
			wrongEnd := end.AddDate(0, 0, -offset)
			fact.EndDate = &wrongEnd
			mustReject = true
		case 4:
			value := strings.TrimSpace(raw)
			if value == "" {
				value = strings.Repeat("9", 96)
			}
			metric.Value = value
			fact.Value = value
		case 5:
			metric.CanonicalMetric = "capital_expenditure"
			metric.Value = fmt.Sprintf("-%d", 1+len(raw))
			fact.Concept = "PaymentsToAcquirePropertyPlantAndEquipment"
			fact.Value = metric.Value
			definition, _ = accountingInputDefinition("capital_expenditure")
			mustReject = true
		}

		authority, reason := accountingSourceCandidate(metric, fact, states, definition, asOf)
		if mustReject && reason == "" {
			t.Fatalf("adversarial SEC/accounting mutation was accepted: mode=%d authority=%+v", mode%6, authority)
		}
		if reason == "" &&
			(authority.FactID != fact.FactID ||
				authority.FilingID != fact.FilingID ||
				!authority.PeriodEnd.Equal(metric.PeriodEnd) ||
				authority.Unit != fact.Unit) {
			t.Fatalf("accepted authority did not preserve source identity: %+v", authority)
		}
	})
}
