package productscope

import "testing"

func TestAccountingAuthorityRegistryCoversEveryTechnology20Issuer(t *testing.T) {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Entries) != 20*8+9 {
		t.Fatalf("registry entries = %d, want 169", len(registry.Entries))
	}
	for _, company := range Companies() {
		for _, input := range canonicalAccountingInputs {
			mapping := ResolveAccountingMapping(
				registry, company.CompanyID, input.CanonicalInput, "us-gaap", input.TaxonomyConcept,
			)
			if mapping.Disposition != AccountingCanonical || !mapping.ComparableRankingEligible {
				t.Fatalf("canonical mapping missing for %s/%s", company.CompanyID, input.CanonicalInput)
			}
			if mapping.MappingKey != accountingMappingKey(
				mapping.CompanyID,
				mapping.CanonicalInput,
				mapping.TaxonomyNamespace,
				mapping.TaxonomyConcept,
				mapping.AccountingPerimeter,
			) {
				t.Fatalf("mapping key is not perimeter-bound: %+v", mapping)
			}
		}
	}
	dispositions := map[AccountingDisposition]int{}
	for _, mapping := range registry.Entries {
		dispositions[mapping.Disposition]++
	}
	if dispositions[AccountingCanonical] != 160 ||
		dispositions[AccountingReviewedAlias] != 6 ||
		dispositions[AccountingContextOnly] != 3 {
		t.Fatalf(
			"unexpected registry distribution: canonical=%d aliases=%d context_only=%d",
			dispositions[AccountingCanonical],
			dispositions[AccountingReviewedAlias],
			dispositions[AccountingContextOnly],
		)
	}
}

func TestAccountingAuthorityRegistryIsIssuerSpecificAndFailsClosed(t *testing.T) {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, companyID, input, concept, perimeter string
		want                                       AccountingDisposition
		ranking                                    bool
	}{
		{
			name: "IBM reviewed revenue alias", companyID: "sec-cik:0000051143",
			input: "revenue", concept: "Revenues", want: AccountingReviewedAlias,
			perimeter: ConsolidatedPeriodicPerimeter,
		},
		{
			name: "Adobe reviewed revenue alias", companyID: "sec-cik:0000796343",
			input: "revenue", concept: "Revenues", want: AccountingReviewedAlias,
			perimeter: ConsolidatedPeriodicPerimeter,
		},
		{
			name: "Qualcomm reviewed revenue alias", companyID: "sec-cik:0000804328",
			input: "revenue", concept: "Revenues", want: AccountingReviewedAlias,
			perimeter: ConsolidatedPeriodicPerimeter,
		},
		{
			name: "Alphabet reviewed revenue alias", companyID: "sec-cik:0001652044",
			input: "revenue", concept: "Revenues", want: AccountingReviewedAlias,
			perimeter: ConsolidatedPeriodicPerimeter,
		},
		{
			name: "NVIDIA reviewed revenue alias", companyID: "sec-cik:0001045810",
			input: "revenue", concept: "Revenues", want: AccountingReviewedAlias,
			perimeter: ConsolidatedPeriodicPerimeter,
		},
		{
			name: "Microsoft does not inherit Alphabet alias", companyID: "sec-cik:0000789019",
			input: "revenue", concept: "Revenues", want: AccountingRejected,
			perimeter: "unreviewed",
		},
		{
			name: "Amazon reviewed cash PPE purchases", companyID: "sec-cik:0001018724",
			input: "capital_expenditure", concept: "PaymentsToAcquireProductiveAssets",
			want: AccountingReviewedAlias, perimeter: PropertyEquipmentCashPerimeter,
		},
		{
			name: "Qualcomm reported cash capex is context only", companyID: "sec-cik:0000804328",
			input: "capital_expenditure", concept: "PaymentsToAcquireProductiveAssets",
			want: AccountingContextOnly, perimeter: ReportedCashCapexPerimeter,
		},
		{
			name: "NVIDIA productive assets are context only", companyID: "sec-cik:0001045810",
			input: "capital_expenditure", concept: "PaymentsToAcquireProductiveAssets",
			want: AccountingContextOnly, perimeter: ProductiveAssetsContextPerimeter,
		},
		{
			name: "Arista productive assets are context only", companyID: "sec-cik:0001596532",
			input: "capital_expenditure", concept: "PaymentsToAcquireProductiveAssets",
			want: AccountingContextOnly, perimeter: ProductiveAssetsContextPerimeter,
		},
		{
			name: "AMD does not inherit NVIDIA context mapping", companyID: "sec-cik:0000002488",
			input: "capital_expenditure", concept: "PaymentsToAcquireProductiveAssets",
			want: AccountingRejected, perimeter: "unreviewed",
		},
		{
			name: "semantic lookalike is rejected", companyID: "sec-cik:0001652044",
			input: "revenue", concept: "SalesRevenueNet", want: AccountingRejected,
			perimeter: "unreviewed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping := ResolveAccountingMapping(
				registry, test.companyID, test.input, "us-gaap", test.concept,
			)
			if mapping.Disposition != test.want {
				t.Fatalf("disposition = %s, want %s", mapping.Disposition, test.want)
			}
			if mapping.AccountingPerimeter != test.perimeter {
				t.Fatalf("perimeter = %q, want %q", mapping.AccountingPerimeter, test.perimeter)
			}
			if mapping.ComparableRankingEligible != test.ranking {
				t.Fatalf("ranking eligibility = %t, want %t", mapping.ComparableRankingEligible, test.ranking)
			}
		})
	}
}

func TestPendingAccountingMappingsAreNotRuntimeAuthority(t *testing.T) {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range issuerSpecificAccountingMappings() {
		mapping := ResolveAccountingMapping(
			registry,
			candidate.CompanyID,
			candidate.CanonicalInput,
			candidate.TaxonomyNamespace,
			candidate.TaxonomyConcept,
		)
		if mapping.ProfessionalReviewStatus != AccountingReviewPending {
			t.Fatalf("issuer-specific mapping bypassed review: %+v", mapping)
		}
		if accountingMappingNumericallyAuthoritative(mapping) ||
			accountingMappingContextDisplayAuthorized(mapping) {
			t.Fatalf("pending mapping became runtime authority: %+v", mapping)
		}
		if mapping.SourceCitation == "" || mapping.SourceLocator == "" ||
			mapping.BoundedSourceLanguage == "" {
			t.Fatalf("issuer-specific mapping lacks review evidence: %+v", mapping)
		}
	}
}

func TestAccountingAuthorityRegistryHashRejectsMutation(t *testing.T) {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		t.Fatal(err)
	}
	registry.Entries[0].ProductLabel = "mutated"
	if err := ValidateAccountingAuthorityRegistry(registry); err == nil {
		t.Fatal("mutated registry unexpectedly validated")
	}
}
