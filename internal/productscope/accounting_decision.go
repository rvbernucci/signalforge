package productscope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	AccountingProfessionalDecisionSchemaV1  = "signalforge/accounting-professional-decision/v1"
	AccountingDecisionConditionallyAccepted = "conditionally_accepted"
	AccountingDecisionRecordLocator         = "docs/accounting-authority/technology20-accounting-professional-review.md"
)

type AccountingProfessionalDecisionRecord struct {
	SchemaVersion         string    `json:"schema_version"`
	UniverseID            string    `json:"universe_id"`
	RegistryVersion       string    `json:"registry_version"`
	RegistrySHA256        string    `json:"registry_sha256"`
	ReviewerName          string    `json:"reviewer_name"`
	ReviewerQualification string    `json:"reviewer_qualification"`
	Disposition           string    `json:"disposition"`
	ApprovedMappingKeys   []string  `json:"approved_mapping_keys"`
	Conditions            []string  `json:"conditions"`
	RejectedScope         string    `json:"rejected_scope"`
	DecidedAt             time.Time `json:"decided_at"`
	RecordLocator         string    `json:"record_locator"`
	DecisionSHA256        string    `json:"decision_sha256"`
}

var approvedAccountingMappingKeys = []string{
	"sec-cik:0000051143|revenue|us-gaap|Revenues|consolidated_periodic_filing",
	"sec-cik:0000796343|revenue|us-gaap|Revenues|consolidated_periodic_filing",
	"sec-cik:0000804328|capital_expenditure|us-gaap|PaymentsToAcquireProductiveAssets|company_reported_cash_capital_expenditures",
	"sec-cik:0000804328|revenue|us-gaap|Revenues|consolidated_periodic_filing",
	"sec-cik:0001018724|capital_expenditure|us-gaap|PaymentsToAcquireProductiveAssets|company_reported_property_and_equipment_cash_purchases",
	"sec-cik:0001045810|capital_expenditure|us-gaap|PaymentsToAcquireProductiveAssets|company_reported_property_equipment_and_intangible_assets",
	"sec-cik:0001045810|revenue|us-gaap|Revenues|consolidated_periodic_filing",
	"sec-cik:0001596532|capital_expenditure|us-gaap|PaymentsToAcquireProductiveAssets|company_reported_property_equipment_and_intangible_assets",
	"sec-cik:0001652044|revenue|us-gaap|Revenues|consolidated_periodic_filing",
}

var runtimeAccountingProfessionalDecision = mustDefaultAccountingProfessionalDecision()

func DefaultAccountingProfessionalDecision() (AccountingProfessionalDecisionRecord, error) {
	decidedAt, err := time.Parse(time.RFC3339, "2026-07-29T14:00:31Z")
	if err != nil {
		return AccountingProfessionalDecisionRecord{}, err
	}
	record := AccountingProfessionalDecisionRecord{
		SchemaVersion:         AccountingProfessionalDecisionSchemaV1,
		UniverseID:            UniverseID,
		RegistryVersion:       AccountingAuthorityRegistryVersion,
		RegistrySHA256:        "1c40b44538eee8c64e066bbf224aae51ff45a05094c1c36a867d58b779973dd4",
		ReviewerName:          "Rafael Bernucci",
		ReviewerQualification: "Project owner and Accounting graduate from the University of Sao Paulo; not acting as an independent auditor.",
		Disposition:           AccountingDecisionConditionallyAccepted,
		ApprovedMappingKeys:   append([]string(nil), approvedAccountingMappingKeys...),
		Conditions: []string{
			"canonical_mapping_precedes_reviewed_alias_for_the_same_issuer_period_and_value",
			"context_only_outputs_never_enter_winner_score_rank_or_relative_conclusion",
			"exact_issuer_mapping_period_unit_dimension_currency_sign_and_active_filing_chain_required",
			"product_labels_and_accounting_perimeters_must_remain_visible",
			"runtime_activation_requires_complete_deterministic_and_pair_level_tests",
		},
		RejectedScope: "Every use beyond the exact issuer-specific mappings, authorized operations, product labels, accounting perimeters, invalidation conditions, and context-only restrictions.",
		DecidedAt:     decidedAt,
		RecordLocator: AccountingDecisionRecordLocator,
	}
	sort.Strings(record.ApprovedMappingKeys)
	sort.Strings(record.Conditions)
	record.DecisionSHA256, err = accountingProfessionalDecisionHash(record)
	if err != nil {
		return AccountingProfessionalDecisionRecord{}, err
	}
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		return AccountingProfessionalDecisionRecord{}, err
	}
	if err := ValidateAccountingProfessionalDecision(record, registry); err != nil {
		return AccountingProfessionalDecisionRecord{}, err
	}
	return record, nil
}

func mustDefaultAccountingProfessionalDecision() AccountingProfessionalDecisionRecord {
	record, err := DefaultAccountingProfessionalDecision()
	if err != nil {
		panic(err)
	}
	return record
}

func ValidateAccountingProfessionalDecision(
	record AccountingProfessionalDecisionRecord,
	registry AccountingAuthorityRegistry,
) error {
	if record.SchemaVersion != AccountingProfessionalDecisionSchemaV1 ||
		record.UniverseID != UniverseID ||
		record.RegistryVersion != registry.RegistryVersion ||
		record.RegistrySHA256 != registry.RegistrySHA256 ||
		record.ReviewerName == "" ||
		record.ReviewerQualification == "" ||
		record.Disposition != AccountingDecisionConditionallyAccepted ||
		record.RejectedScope == "" ||
		record.DecidedAt.IsZero() ||
		record.RecordLocator != AccountingDecisionRecordLocator ||
		record.DecisionSHA256 == "" {
		return errors.New("accounting professional decision envelope is invalid")
	}
	if len(record.ApprovedMappingKeys) != len(approvedAccountingMappingKeys) ||
		len(record.Conditions) == 0 {
		return errors.New("accounting professional decision scope is incomplete")
	}
	expectedKeys := append([]string(nil), approvedAccountingMappingKeys...)
	actualKeys := append([]string(nil), record.ApprovedMappingKeys...)
	sort.Strings(expectedKeys)
	sort.Strings(actualKeys)
	for index := range expectedKeys {
		if actualKeys[index] != expectedKeys[index] {
			return fmt.Errorf("accounting professional decision mapping scope mismatch at %d", index)
		}
	}
	registryMappings := map[string]AccountingMappingAuthority{}
	for _, mapping := range registry.Entries {
		if mapping.Disposition != AccountingCanonical {
			registryMappings[mapping.MappingKey] = mapping
		}
	}
	if len(registryMappings) != len(actualKeys) {
		return errors.New("accounting professional decision does not cover the exact non-canonical registry")
	}
	for _, key := range actualKeys {
		mapping, exists := registryMappings[key]
		if !exists || (mapping.Disposition != AccountingReviewedAlias &&
			mapping.Disposition != AccountingContextOnly) {
			return fmt.Errorf("accounting professional decision references unauthorized mapping %q", key)
		}
	}
	expectedHash, err := accountingProfessionalDecisionHash(record)
	if err != nil {
		return err
	}
	if expectedHash != record.DecisionSHA256 {
		return errors.New("accounting professional decision hash mismatch")
	}
	return nil
}

func accountingDecisionAcceptsMapping(
	record AccountingProfessionalDecisionRecord,
	registry AccountingAuthorityRegistry,
	mapping AccountingMappingAuthority,
) bool {
	if err := ValidateAccountingProfessionalDecision(record, registry); err != nil {
		return false
	}
	index := sort.SearchStrings(record.ApprovedMappingKeys, mapping.MappingKey)
	return index < len(record.ApprovedMappingKeys) &&
		record.ApprovedMappingKeys[index] == mapping.MappingKey
}

func effectiveAccountingMappingNumericallyAuthoritative(
	mapping AccountingMappingAuthority,
	registry AccountingAuthorityRegistry,
	decision AccountingProfessionalDecisionRecord,
) bool {
	if mapping.Disposition == AccountingCanonical {
		return true
	}
	return mapping.Disposition == AccountingReviewedAlias &&
		accountingDecisionAcceptsMapping(decision, registry, mapping)
}

func effectiveAccountingMappingContextDisplayAuthorized(
	mapping AccountingMappingAuthority,
	registry AccountingAuthorityRegistry,
	decision AccountingProfessionalDecisionRecord,
) bool {
	return mapping.Disposition == AccountingContextOnly &&
		!mapping.ComparableRankingEligible &&
		accountingDecisionAcceptsMapping(decision, registry, mapping)
}

func accountingProfessionalDecisionHash(record AccountingProfessionalDecisionRecord) (string, error) {
	record.DecisionSHA256 = ""
	payload, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
