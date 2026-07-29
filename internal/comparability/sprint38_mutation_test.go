package comparability

import (
	"testing"
	"time"

	"github.com/rvbernucci/signalforge/internal/contracts"
)

func TestSprint38ComparabilityMutationMatrixNeverExpandsRelease(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	tests := map[string]func(*contracts.MetricComparabilityRequest){
		"future source availability": func(item *contracts.MetricComparabilityRequest) {
			item.Operands[1].AvailableAt = now.Add(time.Hour)
			item.Operands[1].RetrievedAt = now.Add(2 * time.Hour)
		},
		"currency mismatch": func(item *contracts.MetricComparabilityRequest) {
			item.Operands[1].Currency = "EUR"
		},
		"fiscal period mismatch": func(item *contracts.MetricComparabilityRequest) {
			item.Operands[1].FiscalEnd = item.Operands[1].FiscalEnd.AddDate(0, 0, -1)
		},
		"context-only output": func(item *contracts.MetricComparabilityRequest) {
			item.Operands[1].OutputClass = "context_only"
			item.Operands[1].PairRankingEligible = false
			item.Operands[1].AccountingInputs[0].PairRankingEligible = false
		},
		"accounting perimeter signature mismatch": func(item *contracts.MetricComparabilityRequest) {
			item.Operands[1].AccountingInputs[0].AccountingPerimeter = "unreviewed"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			item := v2Request(t, now)
			mutate(&item)
			item, err := contracts.PopulateMetricComparabilityRequestHash(item)
			if err != nil {
				t.Fatal(err)
			}
			receipt, evaluateErr := Evaluate(item, now.Add(time.Minute), DefaultPolicy())
			if evaluateErr == nil && IsReleasable(receipt.Disposition) {
				t.Fatalf("comparability mutation expanded the release boundary: %+v", receipt)
			}
		})
	}
}
