package productscope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	AccountingAuthorityRegistrySchemaV1 = "signalforge/technology20-accounting-perimeter-registry/v1"
	AccountingAuthorityRegistryVersion  = "technology20-accounting-authority/2026-07-29"
	ConsolidatedPeriodicPerimeter       = "consolidated_periodic_filing"
	AccountingReviewNotRequired         = "not_required_for_exact_canonical_mapping"
	AccountingReviewPending             = "pending_named_professional_disposition"
	AccountingReviewAccepted            = "accepted_named_professional_disposition"
)

type AccountingDisposition string

const (
	AccountingCanonical     AccountingDisposition = "canonical"
	AccountingReviewedAlias AccountingDisposition = "reviewed_alias"
	AccountingContextOnly   AccountingDisposition = "context_only"
	AccountingRejected      AccountingDisposition = "rejected"
	AccountingUnavailable   AccountingDisposition = "unavailable"
)

type AccountingMappingAuthority struct {
	MappingKey                string                `json:"mapping_key"`
	CompanyID                 string                `json:"company_id"`
	CanonicalInput            string                `json:"canonical_input"`
	TaxonomyNamespace         string                `json:"taxonomy_namespace"`
	TaxonomyConcept           string                `json:"taxonomy_concept"`
	AccountingPerimeter       string                `json:"accounting_perimeter"`
	Disposition               AccountingDisposition `json:"disposition"`
	ReasonCode                string                `json:"reason_code"`
	ReviewerBasis             string                `json:"reviewer_basis"`
	ProfessionalReviewStatus  string                `json:"professional_review_status"`
	AuthorizedOperations      []string              `json:"authorized_operations"`
	EffectiveFormFamilies     []string              `json:"effective_form_families"`
	InvalidationConditions    []string              `json:"invalidation_conditions"`
	ProductLabel              string                `json:"product_label"`
	SourceCitation            string                `json:"source_citation,omitempty"`
	SourceLocator             string                `json:"source_locator,omitempty"`
	BoundedSourceLanguage     string                `json:"bounded_source_language,omitempty"`
	ComparableRankingEligible bool                  `json:"comparable_ranking_eligible"`
}

type AccountingAuthorityRegistry struct {
	SchemaVersion      string                       `json:"schema_version"`
	UniverseID         string                       `json:"universe_id"`
	RegistryVersion    string                       `json:"registry_version"`
	DefaultDisposition AccountingDisposition        `json:"default_disposition"`
	Entries            []AccountingMappingAuthority `json:"entries"`
	RegistrySHA256     string                       `json:"registry_sha256"`
}

type canonicalAccountingInput struct {
	CanonicalInput       string
	TaxonomyConcept      string
	AccountingPerimeter  string
	PeriodType           string
	SignPolicy           string
	AuthorizedOperations []string
}

var canonicalAccountingInputs = []canonicalAccountingInput{
	{
		CanonicalInput: "capital_expenditure", TaxonomyConcept: "PaymentsToAcquirePropertyPlantAndEquipment",
		AccountingPerimeter: ConsolidatedPeriodicPerimeter, PeriodType: "duration",
		SignPolicy:           "cash_outflow_positive_magnitude",
		AuthorizedOperations: []string{"financial.capex_intensity", "financial.free_cash_flow"},
	},
	{
		CanonicalInput: "net_income", TaxonomyConcept: "NetIncomeLoss",
		AccountingPerimeter: ConsolidatedPeriodicPerimeter, PeriodType: "duration",
		SignPolicy:           "company_reported_sign",
		AuthorizedOperations: []string{"financial.cash_conversion", "financial.quality_of_earnings"},
	},
	{
		CanonicalInput: "operating_cash_flow", TaxonomyConcept: "NetCashProvidedByUsedInOperatingActivities",
		AccountingPerimeter: ConsolidatedPeriodicPerimeter, PeriodType: "duration",
		SignPolicy: "company_reported_sign",
		AuthorizedOperations: []string{
			"financial.cash_conversion", "financial.free_cash_flow", "financial.quality_of_earnings",
		},
	},
	{
		CanonicalInput: "operating_income", TaxonomyConcept: "OperatingIncomeLoss",
		AccountingPerimeter: ConsolidatedPeriodicPerimeter, PeriodType: "duration",
		SignPolicy:           "company_reported_sign",
		AuthorizedOperations: []string{"financial.operating_margin"},
	},
	{
		CanonicalInput: "revenue", TaxonomyConcept: "RevenueFromContractWithCustomerExcludingAssessedTax",
		AccountingPerimeter: ConsolidatedPeriodicPerimeter, PeriodType: "duration",
		SignPolicy: "company_reported_sign",
		AuthorizedOperations: []string{
			"financial.capex_intensity", "financial.operating_margin", "financial.revenue_growth",
		},
	},
	{
		CanonicalInput: "stockholders_equity", TaxonomyConcept: "StockholdersEquity",
		AccountingPerimeter: ConsolidatedPeriodicPerimeter, PeriodType: "instant",
		SignPolicy:           "company_reported_sign",
		AuthorizedOperations: []string{"accounting.balance_sheet_identity"},
	},
	{
		CanonicalInput: "total_assets", TaxonomyConcept: "Assets",
		AccountingPerimeter: ConsolidatedPeriodicPerimeter, PeriodType: "instant",
		SignPolicy:           "company_reported_sign",
		AuthorizedOperations: []string{"accounting.balance_sheet_identity"},
	},
	{
		CanonicalInput: "total_liabilities", TaxonomyConcept: "Liabilities",
		AccountingPerimeter: ConsolidatedPeriodicPerimeter, PeriodType: "instant",
		SignPolicy:           "company_reported_sign",
		AuthorizedOperations: []string{"accounting.balance_sheet_identity"},
	},
}

var runtimeAccountingAuthorityRegistry = mustDefaultAccountingAuthorityRegistry()

func DefaultAccountingAuthorityRegistry() (AccountingAuthorityRegistry, error) {
	registry := AccountingAuthorityRegistry{
		SchemaVersion: AccountingAuthorityRegistrySchemaV1,
		UniverseID:    UniverseID, RegistryVersion: AccountingAuthorityRegistryVersion,
		DefaultDisposition: AccountingRejected,
	}
	for _, company := range Companies() {
		for _, input := range canonicalAccountingInputs {
			registry.Entries = append(registry.Entries, AccountingMappingAuthority{
				CompanyID: company.CompanyID, CanonicalInput: input.CanonicalInput,
				TaxonomyNamespace: "us-gaap", TaxonomyConcept: input.TaxonomyConcept,
				AccountingPerimeter: input.AccountingPerimeter, Disposition: AccountingCanonical,
				ReasonCode:               "canonical_us_gaap_concept",
				ReviewerBasis:            "Exact US-GAAP concept and consolidated, dimensionless periodic-filing policy.",
				ProfessionalReviewStatus: AccountingReviewNotRequired,
				AuthorizedOperations:     append([]string(nil), input.AuthorizedOperations...),
				EffectiveFormFamilies:    []string{"10-K", "10-Q"},
				InvalidationConditions: []string{
					"company_identity_changes", "concept_or_taxonomy_changes", "dimensions_are_present",
					"filing_chain_is_not_active", "period_unit_currency_or_sign_gate_fails",
				},
				ProductLabel: input.CanonicalInput, ComparableRankingEligible: true,
			})
		}
	}
	registry.Entries = append(registry.Entries,
		revenueAliasAuthority("sec-cik:0001045810", "NVIDIA"),
		revenueAliasAuthority("sec-cik:0001652044", "Alphabet"),
		AccountingMappingAuthority{
			CompanyID: "sec-cik:0001045810", CanonicalInput: "capital_expenditure",
			TaxonomyNamespace: "us-gaap", TaxonomyConcept: "PaymentsToAcquireProductiveAssets",
			AccountingPerimeter: ProductiveAssetsContextPerimeter, Disposition: AccountingContextOnly,
			ReasonCode:               "productive_assets_scope_is_broader_than_ppe_capex",
			ReviewerBasis:            "Issuer-specific SEC filing language identifies a broader productive-assets perimeter.",
			ProfessionalReviewStatus: AccountingReviewPending,
			AuthorizedOperations:     []string{"financial.capex_intensity", "financial.free_cash_flow"},
			EffectiveFormFamilies:    []string{"10-K", "10-Q"},
			InvalidationConditions: []string{
				"issuer_changes_line_item_scope", "filing_chain_is_not_active",
				"dimensions_are_present", "period_unit_currency_or_sign_gate_fails",
			},
			ProductLabel:              "reported reinvestment intensity / residual cash proxy",
			SourceCitation:            "https://www.sec.gov/Archives/edgar/data/1045810/000104581026000021/nvda-20260125.htm",
			SourceLocator:             "Consolidated Statements of Cash Flows, investing activities",
			BoundedSourceLanguage:     "Purchases related to property and equipment and intangible assets",
			ComparableRankingEligible: false,
		},
	)
	for index := range registry.Entries {
		entry := &registry.Entries[index]
		entry.MappingKey = accountingMappingKey(
			entry.CompanyID,
			entry.CanonicalInput,
			entry.TaxonomyNamespace,
			entry.TaxonomyConcept,
			entry.AccountingPerimeter,
		)
	}
	sortAccountingMappings(registry.Entries)
	hash, err := accountingRegistryHash(registry)
	if err != nil {
		return AccountingAuthorityRegistry{}, err
	}
	registry.RegistrySHA256 = hash
	if err := ValidateAccountingAuthorityRegistry(registry); err != nil {
		return AccountingAuthorityRegistry{}, err
	}
	return registry, nil
}

func mustDefaultAccountingAuthorityRegistry() AccountingAuthorityRegistry {
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		panic(err)
	}
	return registry
}

func ValidateAccountingAuthorityRegistry(registry AccountingAuthorityRegistry) error {
	if registry.SchemaVersion != AccountingAuthorityRegistrySchemaV1 ||
		registry.UniverseID != UniverseID ||
		registry.RegistryVersion != AccountingAuthorityRegistryVersion ||
		registry.DefaultDisposition != AccountingRejected ||
		registry.RegistrySHA256 == "" {
		return errors.New("accounting authority registry envelope is invalid")
	}
	if len(registry.Entries) != len(Companies())*len(canonicalAccountingInputs)+3 {
		return fmt.Errorf("accounting authority registry has %d entries, want %d",
			len(registry.Entries), len(Companies())*len(canonicalAccountingInputs)+3)
	}
	seen := map[string]bool{}
	canonicalCoverage := map[string]bool{}
	for index, entry := range registry.Entries {
		expectedKey := accountingMappingKey(
			entry.CompanyID,
			entry.CanonicalInput,
			entry.TaxonomyNamespace,
			entry.TaxonomyConcept,
			entry.AccountingPerimeter,
		)
		if entry.MappingKey != expectedKey || !technologyCompany(entry.CompanyID) || entry.CanonicalInput == "" ||
			entry.TaxonomyNamespace != "us-gaap" || entry.TaxonomyConcept == "" ||
			entry.AccountingPerimeter == "" || entry.ReasonCode == "" ||
			entry.ReviewerBasis == "" || entry.ProfessionalReviewStatus == "" ||
			len(entry.EffectiveFormFamilies) == 0 || len(entry.InvalidationConditions) == 0 {
			return fmt.Errorf("accounting authority registry entry %d is incomplete", index)
		}
		if !validAccountingDisposition(entry.Disposition) {
			return fmt.Errorf("accounting authority registry entry %d has invalid disposition", index)
		}
		key := entry.MappingKey
		if seen[key] {
			return fmt.Errorf("accounting authority registry duplicates %q", key)
		}
		seen[key] = true
		if entry.Disposition == AccountingCanonical {
			canonicalCoverage[entry.CompanyID+"|"+entry.CanonicalInput] = true
		}
		if entry.Disposition == AccountingContextOnly && entry.ComparableRankingEligible {
			return fmt.Errorf("context-only mapping %q cannot enter comparable rankings", key)
		}
		if entry.Disposition == AccountingReviewedAlias &&
			entry.ProfessionalReviewStatus != AccountingReviewAccepted &&
			entry.ComparableRankingEligible {
			return fmt.Errorf("pending reviewed alias %q cannot enter comparable rankings", key)
		}
		if (entry.Disposition == AccountingCanonical || entry.Disposition == AccountingReviewedAlias) &&
			len(entry.AuthorizedOperations) == 0 {
			return fmt.Errorf("releasable mapping %q has no authorized operations", key)
		}
		if entry.Disposition != AccountingCanonical &&
			entry.ProfessionalReviewStatus == AccountingReviewNotRequired {
			return fmt.Errorf("non-canonical mapping %q bypasses professional review", key)
		}
	}
	for _, company := range Companies() {
		for _, input := range canonicalAccountingInputs {
			if !canonicalCoverage[company.CompanyID+"|"+input.CanonicalInput] {
				return fmt.Errorf("canonical authority missing for %s/%s", company.CompanyID, input.CanonicalInput)
			}
		}
	}
	expected, err := accountingRegistryHash(registry)
	if err != nil {
		return err
	}
	if expected != registry.RegistrySHA256 {
		return errors.New("accounting authority registry hash mismatch")
	}
	return nil
}

func ResolveAccountingMapping(
	registry AccountingAuthorityRegistry,
	companyID, canonicalInput, taxonomyNamespace, taxonomyConcept string,
) AccountingMappingAuthority {
	for _, entry := range registry.Entries {
		if entry.CompanyID == companyID && entry.CanonicalInput == canonicalInput &&
			entry.TaxonomyNamespace == taxonomyNamespace && entry.TaxonomyConcept == taxonomyConcept {
			return entry
		}
	}
	return AccountingMappingAuthority{
		MappingKey: accountingMappingKey(
			companyID, canonicalInput, taxonomyNamespace, taxonomyConcept, "unreviewed",
		),
		CompanyID: companyID, CanonicalInput: canonicalInput,
		TaxonomyNamespace: taxonomyNamespace, TaxonomyConcept: taxonomyConcept,
		AccountingPerimeter: "unreviewed", Disposition: AccountingRejected,
		ReasonCode:               "unreviewed_issuer_specific_concept",
		ReviewerBasis:            "No exact issuer-specific entry exists in the versioned authority registry.",
		ProfessionalReviewStatus: "not_submitted_for_review",
		EffectiveFormFamilies:    []string{"10-K", "10-Q"},
		InvalidationConditions:   []string{"explicit_registry_entry_required"},
		ProductLabel:             canonicalInput, ComparableRankingEligible: false,
	}
}

func accountingInputDefinition(canonicalInput string) (canonicalAccountingInput, bool) {
	for _, definition := range canonicalAccountingInputs {
		if definition.CanonicalInput == canonicalInput {
			return definition, true
		}
	}
	return canonicalAccountingInput{}, false
}

func revenueAliasAuthority(companyID, issuer string) AccountingMappingAuthority {
	return AccountingMappingAuthority{
		CompanyID: companyID, CanonicalInput: "revenue", TaxonomyNamespace: "us-gaap",
		TaxonomyConcept: "Revenues", AccountingPerimeter: ConsolidatedPeriodicPerimeter,
		Disposition: AccountingReviewedAlias, ReasonCode: "issuer_specific_revenue_alias",
		ReviewerBasis:            issuer + " reports consolidated revenue under the US-GAAP Revenues concept in the active filing chain.",
		ProfessionalReviewStatus: AccountingReviewPending,
		AuthorizedOperations: []string{
			"financial.capex_intensity", "financial.operating_margin", "financial.revenue_growth",
		},
		EffectiveFormFamilies: []string{"10-K", "10-Q"},
		InvalidationConditions: []string{
			"issuer_changes_revenue_scope", "filing_chain_is_not_active",
			"dimensions_are_present", "period_unit_currency_or_sign_gate_fails",
		},
		ProductLabel: "revenue", SourceCitation: "https://data.sec.gov/api/xbrl/companyfacts/CIK" +
			strings.TrimPrefix(companyID, "sec-cik:") + ".json",
		SourceLocator:         "SEC Company Facts and active periodic filing chain",
		BoundedSourceLanguage: "Revenues", ComparableRankingEligible: false,
	}
}

func accountingMappingNumericallyAuthoritative(mapping AccountingMappingAuthority) bool {
	if mapping.Disposition == AccountingCanonical {
		return true
	}
	return mapping.Disposition == AccountingReviewedAlias &&
		mapping.ProfessionalReviewStatus == AccountingReviewAccepted &&
		mapping.ComparableRankingEligible
}

func accountingMappingContextDisplayAuthorized(mapping AccountingMappingAuthority) bool {
	return mapping.Disposition == AccountingContextOnly &&
		mapping.ProfessionalReviewStatus == AccountingReviewAccepted &&
		!mapping.ComparableRankingEligible
}

func canonicalInputIDs() []string {
	result := make([]string, 0, len(canonicalAccountingInputs))
	for _, input := range canonicalAccountingInputs {
		result = append(result, input.CanonicalInput)
	}
	sort.Strings(result)
	return result
}

func validAccountingDisposition(value AccountingDisposition) bool {
	switch value {
	case AccountingCanonical, AccountingReviewedAlias, AccountingContextOnly,
		AccountingRejected, AccountingUnavailable:
		return true
	default:
		return false
	}
}

func sortAccountingMappings(values []AccountingMappingAuthority) {
	for index := range values {
		sort.Strings(values[index].AuthorizedOperations)
		sort.Strings(values[index].EffectiveFormFamilies)
		sort.Strings(values[index].InvalidationConditions)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].MappingKey < values[j].MappingKey
	})
}

func accountingMappingKey(
	companyID, canonicalInput, taxonomyNamespace, taxonomyConcept, accountingPerimeter string,
) string {
	return strings.Join(
		[]string{companyID, canonicalInput, taxonomyNamespace, taxonomyConcept, accountingPerimeter},
		"|",
	)
}

func accountingRegistryHash(registry AccountingAuthorityRegistry) (string, error) {
	registry.RegistrySHA256 = ""
	payload, err := json.Marshal(registry)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
