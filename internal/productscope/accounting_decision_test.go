package productscope

import "testing"

func TestDefaultAccountingProfessionalDecisionBindsExactRegistryAndMappings(t *testing.T) {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		t.Fatal(err)
	}
	decision, err := DefaultAccountingProfessionalDecision()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAccountingProfessionalDecision(decision, registry); err != nil {
		t.Fatal(err)
	}
	if decision.RegistrySHA256 != registry.RegistrySHA256 ||
		len(decision.ApprovedMappingKeys) != len(issuerSpecificAccountingMappings()) {
		t.Fatalf("decision scope is not exact: %+v", decision)
	}
	for _, mapping := range registry.Entries {
		if mapping.Disposition == AccountingCanonical {
			if !effectiveAccountingMappingNumericallyAuthoritative(mapping, registry, decision) {
				t.Fatalf("canonical mapping was not authoritative: %s", mapping.MappingKey)
			}
			continue
		}
		switch mapping.Disposition {
		case AccountingReviewedAlias:
			if !effectiveAccountingMappingNumericallyAuthoritative(mapping, registry, decision) {
				t.Fatalf("accepted numerical alias was not authoritative: %s", mapping.MappingKey)
			}
		case AccountingContextOnly:
			if !effectiveAccountingMappingContextDisplayAuthorized(mapping, registry, decision) {
				t.Fatalf("accepted context mapping was not display-authorized: %s", mapping.MappingKey)
			}
		}
	}
}

func TestAccountingProfessionalDecisionFailsClosedOnMutation(t *testing.T) {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		t.Fatal(err)
	}
	decision, err := DefaultAccountingProfessionalDecision()
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*AccountingProfessionalDecisionRecord){
		"registry hash": func(value *AccountingProfessionalDecisionRecord) {
			value.RegistrySHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
		"mapping scope": func(value *AccountingProfessionalDecisionRecord) {
			value.ApprovedMappingKeys = value.ApprovedMappingKeys[:len(value.ApprovedMappingKeys)-1]
		},
		"reviewer": func(value *AccountingProfessionalDecisionRecord) {
			value.ReviewerName = ""
		},
		"decision hash": func(value *AccountingProfessionalDecisionRecord) {
			value.DecisionSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := decision
			candidate.ApprovedMappingKeys = append([]string(nil), decision.ApprovedMappingKeys...)
			mutate(&candidate)
			if err := ValidateAccountingProfessionalDecision(candidate, registry); err == nil {
				t.Fatal("mutated decision passed validation")
			}
			for _, mapping := range registry.Entries {
				if mapping.Disposition == AccountingReviewedAlias &&
					effectiveAccountingMappingNumericallyAuthoritative(mapping, registry, candidate) {
					t.Fatal("mutated decision activated numerical authority")
				}
				if mapping.Disposition == AccountingContextOnly &&
					effectiveAccountingMappingContextDisplayAuthorized(mapping, registry, candidate) {
					t.Fatal("mutated decision activated context authority")
				}
			}
		})
	}
}

func TestAcceptedContextMappingsPreserveIssuerSpecificPerimeters(t *testing.T) {
	tests := map[string]string{
		"sec-cik:0000804328": ReportedCashCapexPerimeter,
		"sec-cik:0001045810": ProductiveAssetsContextPerimeter,
		"sec-cik:0001596532": ProductiveAssetsContextPerimeter,
	}
	for companyID, perimeter := range tests {
		mapping := ResolveAccountingMapping(
			runtimeAccountingAuthorityRegistry,
			companyID,
			"capital_expenditure",
			"us-gaap",
			"PaymentsToAcquireProductiveAssets",
		)
		if mapping.AccountingPerimeter != perimeter ||
			!effectiveAccountingMappingContextDisplayAuthorized(
				mapping,
				runtimeAccountingAuthorityRegistry,
				runtimeAccountingProfessionalDecision,
			) ||
			effectiveAccountingMappingNumericallyAuthoritative(
				mapping,
				runtimeAccountingAuthorityRegistry,
				runtimeAccountingProfessionalDecision,
			) {
			t.Fatalf("%s context authority is invalid: %+v", companyID, mapping)
		}
	}
}
