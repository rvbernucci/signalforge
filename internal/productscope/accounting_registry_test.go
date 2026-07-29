package productscope

import "testing"

func TestAccountingAuthorityRegistryCoversEveryTechnology20Issuer(t *testing.T) {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Entries) != 20*8+3 {
		t.Fatalf("registry entries = %d, want 163", len(registry.Entries))
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
}

func TestAccountingAuthorityRegistryIsIssuerSpecificAndFailsClosed(t *testing.T) {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, companyID, input, concept string
		want                            AccountingDisposition
		ranking                         bool
	}{
		{
			name: "Alphabet reviewed revenue alias", companyID: "sec-cik:0001652044",
			input: "revenue", concept: "Revenues", want: AccountingReviewedAlias,
		},
		{
			name: "NVIDIA reviewed revenue alias", companyID: "sec-cik:0001045810",
			input: "revenue", concept: "Revenues", want: AccountingReviewedAlias,
		},
		{
			name: "Microsoft does not inherit Alphabet alias", companyID: "sec-cik:0000789019",
			input: "revenue", concept: "Revenues", want: AccountingRejected,
		},
		{
			name: "NVIDIA productive assets are context only", companyID: "sec-cik:0001045810",
			input: "capital_expenditure", concept: "PaymentsToAcquireProductiveAssets",
			want: AccountingContextOnly,
		},
		{
			name: "AMD does not inherit NVIDIA context mapping", companyID: "sec-cik:0000002488",
			input: "capital_expenditure", concept: "PaymentsToAcquireProductiveAssets",
			want: AccountingRejected,
		},
		{
			name: "semantic lookalike is rejected", companyID: "sec-cik:0001652044",
			input: "revenue", concept: "SalesRevenueNet", want: AccountingRejected,
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
	alias := ResolveAccountingMapping(
		registry, "sec-cik:0001652044", "revenue", "us-gaap", "Revenues",
	)
	if alias.ProfessionalReviewStatus != AccountingReviewPending ||
		accountingMappingNumericallyAuthoritative(alias) {
		t.Fatalf("pending alias became numeric authority: %+v", alias)
	}
	context := ResolveAccountingMapping(
		registry, "sec-cik:0001045810", "capital_expenditure",
		"us-gaap", "PaymentsToAcquireProductiveAssets",
	)
	if context.ProfessionalReviewStatus != AccountingReviewPending ||
		accountingMappingContextDisplayAuthorized(context) {
		t.Fatalf("pending context mapping became product authority: %+v", context)
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
