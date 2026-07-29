package productscope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/rvbernucci/signalforge/internal/data"
)

const (
	AccountingAuthorityPacketSchemaV1   = "signalforge/technology20-accounting-authority-packet/v1"
	AccountingAuthorityManifestSchemaV1 = "signalforge/technology20-accounting-authority-manifest/v1"
	AccountingExceptionsSchemaV1        = "signalforge/technology20-accounting-exceptions/v1"
	AccountingReviewPacketSchemaV1      = "signalforge/technology20-accounting-professional-review/v1"
)

type AccountingAuthoritySourceHashes struct {
	CatalogSHA256 string `json:"catalog_sha256"`
	MetricsSHA256 string `json:"normalized_metrics_sha256"`
	FactsSHA256   string `json:"reported_facts_sha256"`
	FilingsSHA256 string `json:"filings_sha256"`
}

type AccountingSourceAuthority struct {
	FactID            string            `json:"fact_id"`
	FilingID          string            `json:"filing_id"`
	AccessionNumber   string            `json:"accession_number"`
	FormType          string            `json:"form_type"`
	FilingDate        time.Time         `json:"filing_date"`
	AvailableAt       time.Time         `json:"available_at"`
	RetrievedAt       time.Time         `json:"retrieved_at"`
	PeriodStart       *time.Time        `json:"period_start,omitempty"`
	PeriodEnd         time.Time         `json:"period_end"`
	PeriodType        string            `json:"period_type"`
	Unit              string            `json:"unit"`
	Currency          string            `json:"currency"`
	Scale             int32             `json:"scale"`
	Dimensions        map[string]string `json:"dimensions,omitempty"`
	TaxonomyNamespace string            `json:"taxonomy_namespace"`
	TaxonomyConcept   string            `json:"taxonomy_concept"`
	SourceLabel       string            `json:"source_label,omitempty"`
	SourceLocator     string            `json:"source_locator"`
	SourceURI         string            `json:"source_uri"`
	SourceSHA256      string            `json:"source_sha256"`
	AmendmentChain    []string          `json:"amendment_chain"`
	RestatementState  string            `json:"restatement_state"`
	SupersessionState string            `json:"supersession_state"`
}

type AccountingConceptCoverage struct {
	Mapping            AccountingMappingAuthority  `json:"mapping"`
	ObservedRecords    int                         `json:"observed_records"`
	SourceForms        []string                    `json:"source_forms"`
	Units              []string                    `json:"units"`
	Currencies         []string                    `json:"currencies"`
	DimensionStates    []string                    `json:"dimension_states"`
	ActiveAnnualSource []AccountingSourceAuthority `json:"active_annual_sources"`
	ExcludedRecords    map[string]int              `json:"excluded_records,omitempty"`
}

type AccountingInputAuthority struct {
	CanonicalInput           string                      `json:"canonical_input"`
	ExpectedPeriodType       string                      `json:"expected_period_type"`
	SignPolicy               string                      `json:"sign_policy"`
	Disposition              AccountingDisposition       `json:"disposition"`
	AuthorizedOperations     []string                    `json:"authorized_operations"`
	Concepts                 []AccountingConceptCoverage `json:"concepts"`
	ActiveSourceCount        int                         `json:"active_source_count"`
	ProfessionalReviewNeeded bool                        `json:"professional_review_needed"`
	ReasonCodes              []string                    `json:"reason_codes"`
}

type AccountingAuthorityException struct {
	ExceptionID              string                `json:"exception_id"`
	CompanyID                string                `json:"company_id"`
	CanonicalInput           string                `json:"canonical_input"`
	TaxonomyConcept          string                `json:"taxonomy_concept,omitempty"`
	Disposition              AccountingDisposition `json:"disposition"`
	ReasonCode               string                `json:"reason_code"`
	ObservedRecords          int                   `json:"observed_records"`
	ProfessionalReviewNeeded bool                  `json:"professional_review_needed"`
	RequiredAction           string                `json:"required_action"`
	SourceCitation           string                `json:"source_citation,omitempty"`
	SourceLocator            string                `json:"source_locator,omitempty"`
}

type CompanyAccountingAuthorityPacket struct {
	SchemaVersion   string                          `json:"schema_version"`
	UniverseID      string                          `json:"universe_id"`
	RegistryVersion string                          `json:"registry_version"`
	RegistrySHA256  string                          `json:"registry_sha256"`
	CompanyID       string                          `json:"company_id"`
	DisplayName     string                          `json:"display_name"`
	PrimaryTicker   string                          `json:"primary_ticker"`
	AsOf            time.Time                       `json:"as_of"`
	AuthorityState  string                          `json:"authority_state"`
	Inputs          []AccountingInputAuthority      `json:"inputs"`
	Exceptions      []AccountingAuthorityException  `json:"exceptions"`
	SourceHashes    AccountingAuthoritySourceHashes `json:"source_hashes"`
	ClaimBoundary   string                          `json:"claim_boundary"`
	PacketSHA256    string                          `json:"packet_sha256"`
}

type AccountingAuthorityPacketReference struct {
	CompanyID     string `json:"company_id"`
	PrimaryTicker string `json:"primary_ticker"`
	Path          string `json:"path"`
	PacketSHA256  string `json:"packet_sha256"`
}

type Technology20AccountingAuthority struct {
	SchemaVersion    string                               `json:"schema_version"`
	UniverseID       string                               `json:"universe_id"`
	AsOf             time.Time                            `json:"as_of"`
	RegistryVersion  string                               `json:"registry_version"`
	RegistrySHA256   string                               `json:"registry_sha256"`
	SourceHashes     AccountingAuthoritySourceHashes      `json:"source_hashes"`
	Companies        int                                  `json:"companies"`
	Inputs           int                                  `json:"canonical_inputs"`
	DispositionCount map[AccountingDisposition]int        `json:"disposition_count"`
	Packets          []AccountingAuthorityPacketReference `json:"packets"`
	ClaimBoundary    string                               `json:"claim_boundary"`
	ManifestSHA256   string                               `json:"manifest_sha256"`
}

type Technology20AccountingExceptions struct {
	SchemaVersion  string                         `json:"schema_version"`
	UniverseID     string                         `json:"universe_id"`
	AsOf           time.Time                      `json:"as_of"`
	RegistrySHA256 string                         `json:"registry_sha256"`
	Exceptions     []AccountingAuthorityException `json:"exceptions"`
	ReportSHA256   string                         `json:"report_sha256"`
}

type AccountingProfessionalReviewItem struct {
	ExceptionID           string                `json:"exception_id"`
	CompanyID             string                `json:"company_id"`
	CanonicalInput        string                `json:"canonical_input"`
	TaxonomyConcept       string                `json:"taxonomy_concept"`
	ProposedDisposition   AccountingDisposition `json:"proposed_disposition"`
	AccountingPerimeter   string                `json:"accounting_perimeter"`
	ProductLabel          string                `json:"product_label"`
	ReviewerBasis         string                `json:"reviewer_basis"`
	SourceCitation        string                `json:"source_citation"`
	SourceLocator         string                `json:"source_locator"`
	BoundedSourceLanguage string                `json:"bounded_source_language"`
	Decision              string                `json:"decision"`
	NamedReviewer         string                `json:"named_reviewer"`
	ReviewerQualification string                `json:"reviewer_qualification"`
	DecisionTimestamp     string                `json:"decision_timestamp"`
}

type AccountingProfessionalReviewPacket struct {
	SchemaVersion  string                             `json:"schema_version"`
	UniverseID     string                             `json:"universe_id"`
	AsOf           time.Time                          `json:"as_of"`
	RegistrySHA256 string                             `json:"registry_sha256"`
	Items          []AccountingProfessionalReviewItem `json:"items"`
	ClaimBoundary  string                             `json:"claim_boundary"`
	PacketSHA256   string                             `json:"packet_sha256"`
}

type Technology20AccountingBuild struct {
	Registry   AccountingAuthorityRegistry
	Manifest   Technology20AccountingAuthority
	Packets    []CompanyAccountingAuthorityPacket
	Exceptions Technology20AccountingExceptions
	Review     AccountingProfessionalReviewPacket
}

type filingAuthorityState struct {
	Filing         data.Filing
	Active         bool
	Valid          bool
	AmendmentChain []string
	ReasonCode     string
}

type sourceCandidate struct {
	Metric data.NormalizedMetric
	Fact   data.ReportedFact
	Source AccountingSourceAuthority
}

func BuildTechnology20AccountingAuthority(
	catalog PublicCatalog,
	metrics map[string][]data.NormalizedMetric,
	facts map[string]data.ReportedFact,
	filings []data.Filing,
	asOf time.Time,
	sourceHashes AccountingAuthoritySourceHashes,
) (Technology20AccountingBuild, error) {
	if err := ValidatePublicCatalog(catalog); err != nil {
		return Technology20AccountingBuild{}, err
	}
	if asOf.IsZero() || asOf.Before(catalog.AsOf) || !validAccountingSourceHashes(sourceHashes) {
		return Technology20AccountingBuild{}, errors.New("accounting authority requires a valid point-in-time envelope and source hashes")
	}
	registry, err := DefaultAccountingAuthorityRegistry()
	if err != nil {
		return Technology20AccountingBuild{}, err
	}
	filingStates, filingExceptions := buildFilingAuthorityStates(filings, asOf)
	build := Technology20AccountingBuild{
		Registry: registry,
		Manifest: Technology20AccountingAuthority{
			SchemaVersion: AccountingAuthorityManifestSchemaV1, UniverseID: UniverseID,
			AsOf: asOf.UTC(), RegistryVersion: registry.RegistryVersion,
			RegistrySHA256: registry.RegistrySHA256, SourceHashes: sourceHashes,
			DispositionCount: map[AccountingDisposition]int{},
			ClaimBoundary:    "The manifest proves deterministic source and mapping decisions, not professional approval, investment suitability, or comparability of context-only perimeters.",
		},
		Exceptions: Technology20AccountingExceptions{
			SchemaVersion: AccountingExceptionsSchemaV1, UniverseID: UniverseID,
			AsOf: asOf.UTC(), RegistrySHA256: registry.RegistrySHA256,
		},
		Review: AccountingProfessionalReviewPacket{
			SchemaVersion: AccountingReviewPacketSchemaV1, UniverseID: UniverseID,
			AsOf: asOf.UTC(), RegistrySHA256: registry.RegistrySHA256,
			ClaimBoundary: "Blank decision fields are deliberate. A machine run cannot grant professional accounting approval.",
		},
	}
	for _, company := range catalog.Companies {
		packet, buildErr := buildCompanyAccountingAuthorityPacket(
			company, metrics[company.CompanyID], facts, filingStates, filingExceptions[company.CompanyID],
			asOf, sourceHashes, registry,
		)
		if buildErr != nil {
			return Technology20AccountingBuild{}, fmt.Errorf("%s: %w", company.CompanyID, buildErr)
		}
		build.Packets = append(build.Packets, packet)
		build.Manifest.Packets = append(build.Manifest.Packets, AccountingAuthorityPacketReference{
			CompanyID: company.CompanyID, PrimaryTicker: company.PrimaryTicker,
			Path: "packets/" + company.PrimaryTicker + ".json", PacketSHA256: packet.PacketSHA256,
		})
		for _, input := range packet.Inputs {
			build.Manifest.DispositionCount[input.Disposition]++
			build.Manifest.Inputs++
		}
		build.Exceptions.Exceptions = append(build.Exceptions.Exceptions, packet.Exceptions...)
		for _, exception := range packet.Exceptions {
			if !exception.ProfessionalReviewNeeded {
				continue
			}
			mapping := ResolveAccountingMapping(
				registry, exception.CompanyID, exception.CanonicalInput, "us-gaap", exception.TaxonomyConcept,
			)
			citation, locator := mapping.SourceCitation, mapping.SourceLocator
			if citation == "" {
				citation = exception.SourceCitation
			}
			if locator == "" {
				locator = exception.SourceLocator
			}
			sourceLanguage := mapping.BoundedSourceLanguage
			if sourceLanguage == "" {
				sourceLanguage = exception.TaxonomyConcept
			}
			build.Review.Items = append(build.Review.Items, AccountingProfessionalReviewItem{
				ExceptionID: exception.ExceptionID, CompanyID: exception.CompanyID,
				CanonicalInput: exception.CanonicalInput, TaxonomyConcept: exception.TaxonomyConcept,
				ProposedDisposition: mapping.Disposition, AccountingPerimeter: mapping.AccountingPerimeter,
				ProductLabel: mapping.ProductLabel, ReviewerBasis: mapping.ReviewerBasis,
				SourceCitation: citation, SourceLocator: locator,
				BoundedSourceLanguage: sourceLanguage, Decision: "pending",
			})
		}
	}
	build.Manifest.Companies = len(build.Packets)
	sort.Slice(build.Packets, func(i, j int) bool { return build.Packets[i].CompanyID < build.Packets[j].CompanyID })
	sort.Slice(build.Manifest.Packets, func(i, j int) bool {
		return build.Manifest.Packets[i].CompanyID < build.Manifest.Packets[j].CompanyID
	})
	sortAccountingExceptions(build.Exceptions.Exceptions)
	sort.Slice(build.Review.Items, func(i, j int) bool {
		return build.Review.Items[i].ExceptionID < build.Review.Items[j].ExceptionID
	})
	build.Exceptions.Exceptions = uniqueAccountingExceptions(build.Exceptions.Exceptions)
	build.Review.Items = uniqueProfessionalReviewItems(build.Review.Items)
	build.Exceptions.ReportSHA256, err = hashAccountingExceptions(build.Exceptions)
	if err != nil {
		return Technology20AccountingBuild{}, err
	}
	build.Review.PacketSHA256, err = hashAccountingReviewPacket(build.Review)
	if err != nil {
		return Technology20AccountingBuild{}, err
	}
	build.Manifest.ManifestSHA256, err = hashAccountingManifest(build.Manifest)
	if err != nil {
		return Technology20AccountingBuild{}, err
	}
	if err := ValidateTechnology20AccountingBuild(build); err != nil {
		return Technology20AccountingBuild{}, err
	}
	return build, nil
}

func buildCompanyAccountingAuthorityPacket(
	company PublicCompany,
	metrics []data.NormalizedMetric,
	facts map[string]data.ReportedFact,
	filingStates map[string]filingAuthorityState,
	filingExceptions []AccountingAuthorityException,
	asOf time.Time,
	sourceHashes AccountingAuthoritySourceHashes,
	registry AccountingAuthorityRegistry,
) (CompanyAccountingAuthorityPacket, error) {
	packet := CompanyAccountingAuthorityPacket{
		SchemaVersion: AccountingAuthorityPacketSchemaV1, UniverseID: UniverseID,
		RegistryVersion: registry.RegistryVersion, RegistrySHA256: registry.RegistrySHA256,
		CompanyID: company.CompanyID, DisplayName: company.DisplayName,
		PrimaryTicker: company.PrimaryTicker, AsOf: asOf.UTC(), AuthorityState: "data_ready",
		SourceHashes:  sourceHashes,
		ClaimBoundary: "This metadata-only packet authorizes deterministic source selection. It contains no calculation values and cannot authorize model arithmetic or an unreviewed equivalence.",
		Exceptions:    append([]AccountingAuthorityException(nil), filingExceptions...),
	}
	sort.Slice(metrics, func(i, j int) bool { return normalizedMetricOrder(metrics[i]) < normalizedMetricOrder(metrics[j]) })
	for _, definition := range canonicalAccountingInputs {
		input := buildAccountingInputAuthority(
			company.CompanyID, definition, metrics, facts, filingStates, asOf, registry,
		)
		packet.Inputs = append(packet.Inputs, input)
		packet.Exceptions = append(packet.Exceptions, exceptionsForAccountingInput(company.CompanyID, input)...)
	}
	sort.Slice(packet.Inputs, func(i, j int) bool {
		return packet.Inputs[i].CanonicalInput < packet.Inputs[j].CanonicalInput
	})
	sortAccountingExceptions(packet.Exceptions)
	packet.Exceptions = uniqueAccountingExceptions(packet.Exceptions)
	for _, exception := range packet.Exceptions {
		if exception.ProfessionalReviewNeeded || exception.Disposition == AccountingRejected {
			packet.AuthorityState = "limited"
			break
		}
	}
	hash, err := companyAccountingPacketHash(packet)
	if err != nil {
		return CompanyAccountingAuthorityPacket{}, err
	}
	packet.PacketSHA256 = hash
	return packet, ValidateCompanyAccountingAuthorityPacket(packet)
}

func buildAccountingInputAuthority(
	companyID string,
	definition canonicalAccountingInput,
	metrics []data.NormalizedMetric,
	facts map[string]data.ReportedFact,
	filingStates map[string]filingAuthorityState,
	asOf time.Time,
	registry AccountingAuthorityRegistry,
) AccountingInputAuthority {
	input := AccountingInputAuthority{
		CanonicalInput: definition.CanonicalInput, ExpectedPeriodType: definition.PeriodType,
		SignPolicy: definition.SignPolicy, Disposition: AccountingUnavailable,
		AuthorizedOperations: append([]string(nil), definition.AuthorizedOperations...),
	}
	coverage := map[string]*AccountingConceptCoverage{}
	for _, metric := range metrics {
		if metric.CompanyID != companyID || metric.CanonicalMetric != definition.CanonicalInput {
			continue
		}
		for _, factID := range metric.SourceFactIDs {
			fact, exists := facts[factID]
			if !exists {
				key := "missing|missing"
				item := ensureConceptCoverage(coverage, key, ResolveAccountingMapping(
					registry, companyID, definition.CanonicalInput, "unknown", "missing",
				))
				item.ObservedRecords++
				item.ExcludedRecords["source_fact_missing"]++
				continue
			}
			mapping := ResolveAccountingMapping(
				registry, companyID, definition.CanonicalInput, fact.Taxonomy, fact.Concept,
			)
			key := fact.Taxonomy + "|" + fact.Concept
			item := ensureConceptCoverage(coverage, key, mapping)
			item.ObservedRecords++
			item.SourceForms = appendUniqueSorted(item.SourceForms, fact.FormType)
			item.Units = appendUniqueSorted(item.Units, fact.Unit)
			item.Currencies = appendUniqueSorted(item.Currencies, metric.Currency)
			if len(fact.Dimensions) == 0 {
				item.DimensionStates = appendUniqueSorted(item.DimensionStates, "consolidated")
			} else {
				item.DimensionStates = appendUniqueSorted(item.DimensionStates, "dimensioned")
			}
			source, reason := accountingSourceCandidate(metric, fact, filingStates, definition, asOf)
			if reason != "" {
				item.ExcludedRecords[reason]++
				continue
			}
			item.ActiveAnnualSource = append(item.ActiveAnnualSource, source)
		}
	}
	keys := make([]string, 0, len(coverage))
	for key := range coverage {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item := coverage[key]
		var ambiguous int
		item.ActiveAnnualSource, ambiguous = activeAnnualSources(item.ActiveAnnualSource)
		if ambiguous > 0 {
			item.ExcludedRecords["equally_authoritative_duplicate_facts"] += ambiguous
		}
		input.ActiveSourceCount += len(item.ActiveAnnualSource)
		input.Concepts = append(input.Concepts, *item)
	}
	input.Disposition = inputDisposition(input.Concepts)
	if input.Disposition == AccountingUnavailable {
		input.ReasonCodes = []string{"authorized_annual_source_unavailable"}
	}
	for _, concept := range input.Concepts {
		if len(concept.ActiveAnnualSource) == 0 {
			continue
		}
		switch concept.Mapping.Disposition {
		case AccountingReviewedAlias, AccountingContextOnly:
			input.ProfessionalReviewNeeded = true
			input.ReasonCodes = appendUniqueSorted(input.ReasonCodes, concept.Mapping.ReasonCode)
		case AccountingRejected:
			input.ReasonCodes = appendUniqueSorted(input.ReasonCodes, concept.Mapping.ReasonCode)
		}
	}
	sort.Strings(input.AuthorizedOperations)
	sort.Strings(input.ReasonCodes)
	return input
}

func accountingSourceCandidate(
	metric data.NormalizedMetric,
	fact data.ReportedFact,
	filingStates map[string]filingAuthorityState,
	definition canonicalAccountingInput,
	asOf time.Time,
) (AccountingSourceAuthority, string) {
	if err := data.ValidateNormalizedMetric(metric); err != nil {
		return AccountingSourceAuthority{}, "invalid_metric_contract"
	}
	if err := data.ValidateReportedFact(fact); err != nil {
		return AccountingSourceAuthority{}, "invalid_fact_contract"
	}
	if metric.SourceAvailableAt.After(asOf) || fact.AvailableAt.After(asOf) {
		return AccountingSourceAuthority{}, "look_ahead"
	}
	if asOf.Sub(metric.PeriodEnd) > 760*24*time.Hour {
		return AccountingSourceAuthority{}, "stale_metric_period"
	}
	if metric.CompanyID != fact.CompanyID || metric.Value != fact.Value ||
		metric.Unit != fact.Unit || metric.Currency != "USD" || fact.Unit != "USD" {
		return AccountingSourceAuthority{}, "identity_unit_or_currency_mismatch"
	}
	if !validAccountingSign(metric.Value, definition.SignPolicy) {
		return AccountingSourceAuthority{}, "numeric_or_sign_policy_failed"
	}
	if metric.PeriodType != definition.PeriodType {
		return AccountingSourceAuthority{}, "period_type_mismatch"
	}
	if len(fact.Dimensions) != 0 {
		return AccountingSourceAuthority{}, "dimensioned_fact"
	}
	if fact.Scale != 0 {
		return AccountingSourceAuthority{}, "nonzero_scale"
	}
	if fact.FormType != "10-K" && fact.FormType != "10-K/A" {
		return AccountingSourceAuthority{}, "non_annual_form"
	}
	if definition.PeriodType == "duration" {
		if fact.StartDate == nil || fact.EndDate == nil ||
			!fact.StartDate.Equal(metric.PeriodStart) || !fact.EndDate.Equal(metric.PeriodEnd) ||
			!annualDuration(metric.PeriodStart, metric.PeriodEnd) {
			return AccountingSourceAuthority{}, "annual_period_alignment_failed"
		}
	} else if fact.InstantDate == nil || !fact.InstantDate.Equal(metric.PeriodEnd) ||
		!metric.PeriodStart.Equal(metric.PeriodEnd) {
		return AccountingSourceAuthority{}, "instant_period_alignment_failed"
	}
	state, exists := filingStates[fact.FilingID]
	if !exists || !state.Valid {
		return AccountingSourceAuthority{}, "filing_authority_missing_or_invalid"
	}
	if !state.Active {
		return AccountingSourceAuthority{}, "filing_superseded"
	}
	if state.Filing.CompanyID != fact.CompanyID || state.Filing.FormType != fact.FormType ||
		!state.Filing.PublishedAt.Equal(fact.AvailableAt) {
		return AccountingSourceAuthority{}, "filing_fact_chain_mismatch"
	}
	if !validSHA256(state.Filing.ContentSHA256) || !officialSECSource(state.Filing.SourceURI) {
		return AccountingSourceAuthority{}, "source_hash_or_origin_invalid"
	}
	periodEnd := metric.PeriodEnd
	var periodStart *time.Time
	if definition.PeriodType == "duration" {
		value := metric.PeriodStart
		periodStart = &value
	}
	return AccountingSourceAuthority{
		FactID: fact.FactID, FilingID: fact.FilingID,
		AccessionNumber: state.Filing.AccessionNumber, FormType: fact.FormType,
		FilingDate: state.Filing.FiledAt, AvailableAt: fact.AvailableAt, RetrievedAt: fact.RetrievedAt,
		PeriodStart: periodStart, PeriodEnd: periodEnd, PeriodType: definition.PeriodType,
		Unit: fact.Unit, Currency: metric.Currency, Scale: fact.Scale,
		TaxonomyNamespace: fact.Taxonomy, TaxonomyConcept: fact.Concept,
		SourceLabel: fact.Label, SourceLocator: fact.SourceLocator, SourceURI: state.Filing.SourceURI,
		SourceSHA256:     state.Filing.ContentSHA256,
		AmendmentChain:   append([]string(nil), state.AmendmentChain...),
		RestatementState: restatementState(state), SupersessionState: "active",
	}, ""
}

func buildFilingAuthorityStates(
	filings []data.Filing,
	asOf time.Time,
) (map[string]filingAuthorityState, map[string][]AccountingAuthorityException) {
	states := map[string]filingAuthorityState{}
	exceptions := map[string][]AccountingAuthorityException{}
	for _, filing := range filings {
		if filing.FormType != "10-K" && filing.FormType != "10-K/A" {
			continue
		}
		state := filingAuthorityState{Filing: filing, Valid: true}
		if err := data.ValidateFiling(filing); err != nil {
			state.Valid = false
			state.ReasonCode = "invalid_filing_contract"
		} else if filing.PublishedAt.After(asOf) {
			state.Valid = false
			state.ReasonCode = "filing_look_ahead"
		}
		if prior, exists := states[filing.FilingID]; exists {
			prior.Valid = false
			prior.ReasonCode = "duplicate_filing_identity"
			states[filing.FilingID] = prior
			state.Valid = false
			state.ReasonCode = "duplicate_filing_identity"
		}
		states[filing.FilingID] = state
	}
	for id, state := range states {
		if !state.Valid || state.Filing.FormType != "10-K/A" {
			continue
		}
		parent, exists := states[state.Filing.AmendsFilingID]
		if state.Filing.AmendsFilingID == "" || !exists || !parent.Valid ||
			parent.Filing.CompanyID != state.Filing.CompanyID ||
			!state.Filing.PublishedAt.After(parent.Filing.PublishedAt) {
			state.Valid = false
			state.ReasonCode = "invalid_amendment_chain"
			states[id] = state
		}
	}
	for id, state := range states {
		if !state.Valid {
			continue
		}
		cursor := state
		seen := map[string]bool{}
		for {
			if seen[cursor.Filing.FilingID] {
				state.Valid = false
				state.ReasonCode = "amendment_cycle"
				break
			}
			seen[cursor.Filing.FilingID] = true
			state.AmendmentChain = append(state.AmendmentChain, cursor.Filing.AccessionNumber)
			if cursor.Filing.AmendsFilingID == "" {
				break
			}
			parent, exists := states[cursor.Filing.AmendsFilingID]
			if !exists || !parent.Valid {
				state.Valid = false
				state.ReasonCode = "amendment_ancestor_missing"
				break
			}
			cursor = parent
		}
		if state.Valid {
			for ancestorID := state.Filing.AmendsFilingID; ancestorID != ""; {
				ancestor := states[ancestorID]
				ancestor.Active = false
				states[ancestorID] = ancestor
				ancestorID = ancestor.Filing.AmendsFilingID
			}
		}
		states[id] = state
	}
	activePeriod := map[string][]string{}
	for id, state := range states {
		if !state.Valid {
			continue
		}
		if !filingSuperseded(id, states) {
			state.Active = true
			states[id] = state
			key := state.Filing.CompanyID + "|" + state.Filing.ReportPeriodEnd.Format("2006-01-02")
			activePeriod[key] = append(activePeriod[key], id)
		}
	}
	for _, ids := range activePeriod {
		if len(ids) <= 1 {
			continue
		}
		for _, id := range ids {
			state := states[id]
			state.Active = false
			state.Valid = false
			state.ReasonCode = "ambiguous_active_annual_filing"
			states[id] = state
		}
	}
	for _, state := range states {
		if state.Valid {
			continue
		}
		if state.Filing.ReportPeriodEnd.Before(asOf.AddDate(-4, 0, 0)) {
			continue
		}
		exceptions[state.Filing.CompanyID] = append(
			exceptions[state.Filing.CompanyID],
			newAccountingException(
				state.Filing.CompanyID, "filing_chain", state.Filing.FormType,
				AccountingRejected, state.ReasonCode, 1, false,
				"Repair or exclude the malformed point-in-time filing chain before activation.",
				state.Filing.SourceURI, state.Filing.AccessionNumber,
			),
		)
	}
	return states, exceptions
}

func ensureConceptCoverage(
	values map[string]*AccountingConceptCoverage,
	key string,
	mapping AccountingMappingAuthority,
) *AccountingConceptCoverage {
	if values[key] == nil {
		values[key] = &AccountingConceptCoverage{
			Mapping: mapping, ExcludedRecords: map[string]int{},
		}
	}
	return values[key]
}

func inputDisposition(concepts []AccountingConceptCoverage) AccountingDisposition {
	best := AccountingUnavailable
	for _, concept := range concepts {
		if len(concept.ActiveAnnualSource) == 0 {
			continue
		}
		switch concept.Mapping.Disposition {
		case AccountingCanonical:
			return AccountingCanonical
		case AccountingReviewedAlias:
			if best != AccountingCanonical {
				best = AccountingReviewedAlias
			}
		case AccountingContextOnly:
			if best == AccountingUnavailable || best == AccountingRejected {
				best = AccountingContextOnly
			}
		case AccountingRejected:
			if best == AccountingUnavailable {
				best = AccountingRejected
			}
		}
	}
	return best
}

func exceptionsForAccountingInput(
	companyID string,
	input AccountingInputAuthority,
) []AccountingAuthorityException {
	result := []AccountingAuthorityException{}
	if input.Disposition == AccountingUnavailable {
		result = append(result, newAccountingException(
			companyID, input.CanonicalInput, "", AccountingUnavailable,
			"authorized_annual_source_unavailable", 0, false,
			"Preserve typed unavailability until an authorized annual source becomes available.", "", "",
		))
	}
	for _, concept := range input.Concepts {
		if len(concept.ActiveAnnualSource) == 0 {
			continue
		}
		mapping := concept.Mapping
		switch mapping.Disposition {
		case AccountingReviewedAlias, AccountingContextOnly:
			result = append(result, newAccountingException(
				companyID, input.CanonicalInput, mapping.TaxonomyConcept, mapping.Disposition,
				mapping.ReasonCode, concept.ObservedRecords, true,
				"Obtain and record a named professional disposition before final release.",
				mapping.SourceCitation, mapping.SourceLocator,
			))
		case AccountingRejected:
			professional := input.Disposition == AccountingRejected
			action := "Keep rejected unless an issuer-specific primary-source review creates a versioned registry entry."
			if professional {
				action = "Review the active issuer-specific source and either add a bounded registry entry or preserve rejection."
			}
			citation, locator := mapping.SourceCitation, mapping.SourceLocator
			if len(concept.ActiveAnnualSource) > 0 {
				if citation == "" {
					citation = concept.ActiveAnnualSource[0].SourceURI
				}
				if locator == "" {
					locator = concept.ActiveAnnualSource[0].SourceLocator
				}
			}
			result = append(result, newAccountingException(
				companyID, input.CanonicalInput, mapping.TaxonomyConcept, AccountingRejected,
				mapping.ReasonCode, concept.ObservedRecords, professional,
				action, citation, locator,
			))
		}
	}
	return result
}

func activeAnnualSources(values []AccountingSourceAuthority) ([]AccountingSourceAuthority, int) {
	sort.Slice(values, func(i, j int) bool {
		if !values[i].PeriodEnd.Equal(values[j].PeriodEnd) {
			return values[i].PeriodEnd.After(values[j].PeriodEnd)
		}
		if !values[i].AvailableAt.Equal(values[j].AvailableAt) {
			return values[i].AvailableAt.After(values[j].AvailableAt)
		}
		return values[i].FactID < values[j].FactID
	})
	byPeriod := map[string][]AccountingSourceAuthority{}
	for _, source := range values {
		key := accountingSourcePeriodKey(source)
		byPeriod[key] = append(byPeriod[key], source)
	}
	keys := make([]string, 0, len(byPeriod))
	for key := range byPeriod {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := byPeriod[keys[i]][0]
		right := byPeriod[keys[j]][0]
		if !left.PeriodEnd.Equal(right.PeriodEnd) {
			return left.PeriodEnd.After(right.PeriodEnd)
		}
		return keys[i] < keys[j]
	})
	result := make([]AccountingSourceAuthority, 0, 3)
	ambiguous := 0
	for _, key := range keys {
		candidates := byPeriod[key]
		sort.Slice(candidates, func(i, j int) bool {
			if !candidates[i].AvailableAt.Equal(candidates[j].AvailableAt) {
				return candidates[i].AvailableAt.After(candidates[j].AvailableAt)
			}
			return candidates[i].FactID < candidates[j].FactID
		})
		if len(candidates) > 1 &&
			candidates[0].AvailableAt.Equal(candidates[1].AvailableAt) &&
			candidates[0].FactID != candidates[1].FactID {
			ambiguous += len(candidates)
			continue
		}
		result = append(result, candidates[0])
		if len(result) == 3 {
			break
		}
	}
	return result, ambiguous
}

func accountingSourcePeriodKey(source AccountingSourceAuthority) string {
	key := source.PeriodType + "|" + source.PeriodEnd.Format("2006-01-02")
	if source.PeriodStart != nil {
		key += "|" + source.PeriodStart.Format("2006-01-02")
	}
	return key
}

func ValidateTechnology20AccountingBuild(build Technology20AccountingBuild) error {
	if err := ValidateAccountingAuthorityRegistry(build.Registry); err != nil {
		return err
	}
	if build.Manifest.SchemaVersion != AccountingAuthorityManifestSchemaV1 ||
		build.Manifest.UniverseID != UniverseID || build.Manifest.AsOf.IsZero() ||
		build.Manifest.RegistrySHA256 != build.Registry.RegistrySHA256 ||
		build.Manifest.Companies != 20 || build.Manifest.Inputs != 20*len(canonicalAccountingInputs) ||
		len(build.Manifest.Packets) != 20 || len(build.Packets) != 20 ||
		!validAccountingSourceHashes(build.Manifest.SourceHashes) {
		return errors.New("technology20 accounting authority manifest is incomplete")
	}
	companies := map[string]bool{}
	packetHashes := map[string]string{}
	for _, packet := range build.Packets {
		if err := ValidateCompanyAccountingAuthorityPacket(packet); err != nil {
			return err
		}
		if companies[packet.CompanyID] {
			return errors.New("technology20 accounting authority duplicates a company packet")
		}
		companies[packet.CompanyID] = true
		packetHashes[packet.CompanyID] = packet.PacketSHA256
	}
	for _, ref := range build.Manifest.Packets {
		if packetHashes[ref.CompanyID] != ref.PacketSHA256 ||
			ref.Path != "packets/"+ref.PrimaryTicker+".json" {
			return errors.New("technology20 accounting packet reference is invalid")
		}
	}
	expectedManifest, err := hashAccountingManifest(build.Manifest)
	if err != nil || expectedManifest != build.Manifest.ManifestSHA256 {
		return errors.New("technology20 accounting authority manifest hash mismatch")
	}
	expectedExceptions, err := hashAccountingExceptions(build.Exceptions)
	if err != nil || expectedExceptions != build.Exceptions.ReportSHA256 {
		return errors.New("technology20 accounting exceptions hash mismatch")
	}
	expectedReview, err := hashAccountingReviewPacket(build.Review)
	if err != nil || expectedReview != build.Review.PacketSHA256 {
		return errors.New("technology20 accounting professional review hash mismatch")
	}
	return nil
}

func ValidateCompanyAccountingAuthorityPacket(packet CompanyAccountingAuthorityPacket) error {
	if packet.SchemaVersion != AccountingAuthorityPacketSchemaV1 ||
		packet.UniverseID != UniverseID || packet.RegistryVersion != AccountingAuthorityRegistryVersion ||
		packet.RegistrySHA256 == "" || !technologyCompany(packet.CompanyID) ||
		packet.DisplayName == "" || packet.PrimaryTicker == "" || packet.AsOf.IsZero() ||
		(packet.AuthorityState != "data_ready" && packet.AuthorityState != "limited") ||
		len(packet.Inputs) != len(canonicalAccountingInputs) ||
		!validAccountingSourceHashes(packet.SourceHashes) || packet.PacketSHA256 == "" {
		return errors.New("company accounting authority packet envelope is invalid")
	}
	seen := map[string]bool{}
	for _, input := range packet.Inputs {
		if seen[input.CanonicalInput] || !validAccountingDisposition(input.Disposition) ||
			input.ExpectedPeriodType == "" || input.SignPolicy == "" ||
			len(input.AuthorizedOperations) == 0 {
			return errors.New("company accounting authority packet contains an invalid input")
		}
		seen[input.CanonicalInput] = true
		for _, concept := range input.Concepts {
			if concept.Mapping.CompanyID != packet.CompanyID ||
				concept.Mapping.CanonicalInput != input.CanonicalInput ||
				concept.ObservedRecords <= 0 {
				return errors.New("company accounting authority packet contains invalid concept coverage")
			}
			for _, source := range concept.ActiveAnnualSource {
				if source.FactID == "" || source.FilingID == "" || source.AccessionNumber == "" ||
					(source.FormType != "10-K" && source.FormType != "10-K/A") ||
					source.AvailableAt.After(packet.AsOf) || source.Unit != "USD" ||
					source.Currency != "USD" || source.Scale != 0 || len(source.Dimensions) != 0 ||
					source.SupersessionState != "active" || !validSHA256(source.SourceSHA256) ||
					!officialSECSource(source.SourceURI) {
					return errors.New("company accounting authority packet contains invalid active source")
				}
			}
			if concept.Mapping.Disposition == AccountingContextOnly &&
				concept.Mapping.ComparableRankingEligible {
				return errors.New("context-only authority entered comparable ranking")
			}
		}
	}
	expected, err := companyAccountingPacketHash(packet)
	if err != nil {
		return err
	}
	if expected != packet.PacketSHA256 {
		return errors.New("company accounting authority packet hash mismatch")
	}
	return nil
}

func newAccountingException(
	companyID, canonicalInput, concept string,
	disposition AccountingDisposition,
	reason string,
	observed int,
	professional bool,
	action, citation, locator string,
) AccountingAuthorityException {
	seed := strings.Join([]string{companyID, canonicalInput, concept, string(disposition), reason}, "\n")
	digest := sha256.Sum256([]byte(seed))
	return AccountingAuthorityException{
		ExceptionID: "accounting-exception-" + hex.EncodeToString(digest[:12]),
		CompanyID:   companyID, CanonicalInput: canonicalInput, TaxonomyConcept: concept,
		Disposition: disposition, ReasonCode: reason, ObservedRecords: observed,
		ProfessionalReviewNeeded: professional, RequiredAction: action,
		SourceCitation: citation, SourceLocator: locator,
	}
}

func sortAccountingExceptions(values []AccountingAuthorityException) {
	sort.Slice(values, func(i, j int) bool { return values[i].ExceptionID < values[j].ExceptionID })
}

func uniqueAccountingExceptions(values []AccountingAuthorityException) []AccountingAuthorityException {
	result := make([]AccountingAuthorityException, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value.ExceptionID] {
			continue
		}
		seen[value.ExceptionID] = true
		result = append(result, value)
	}
	return result
}

func uniqueProfessionalReviewItems(values []AccountingProfessionalReviewItem) []AccountingProfessionalReviewItem {
	result := make([]AccountingProfessionalReviewItem, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value.ExceptionID] {
			continue
		}
		seen[value.ExceptionID] = true
		result = append(result, value)
	}
	return result
}

func filingSuperseded(filingID string, states map[string]filingAuthorityState) bool {
	for _, candidate := range states {
		if candidate.Valid && candidate.Filing.AmendsFilingID == filingID {
			return true
		}
	}
	return false
}

func restatementState(state filingAuthorityState) string {
	if len(state.AmendmentChain) > 1 {
		return "active_amendment_chain"
	}
	return "not_restated"
}

func normalizedMetricOrder(metric data.NormalizedMetric) string {
	return strings.Join([]string{
		metric.CompanyID, metric.CanonicalMetric, metric.PeriodStart.Format(time.RFC3339),
		metric.PeriodEnd.Format(time.RFC3339), metric.MetricID,
	}, "|")
}

func validAccountingSourceHashes(hashes AccountingAuthoritySourceHashes) bool {
	return validSHA256(hashes.CatalogSHA256) && validSHA256(hashes.MetricsSHA256) &&
		validSHA256(hashes.FactsSHA256) && validSHA256(hashes.FilingsSHA256)
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validAccountingSign(value, policy string) bool {
	number, ok := new(big.Rat).SetString(value)
	if !ok {
		return false
	}
	if policy == "cash_outflow_positive_magnitude" {
		return number.Sign() >= 0
	}
	return policy == "company_reported_sign"
}

func officialSECSource(value string) bool {
	return strings.HasPrefix(value, "https://data.sec.gov/") ||
		strings.HasPrefix(value, "https://www.sec.gov/")
}

func companyAccountingPacketHash(packet CompanyAccountingAuthorityPacket) (string, error) {
	packet.PacketSHA256 = ""
	return hashAccountingValue(packet)
}

func hashAccountingManifest(manifest Technology20AccountingAuthority) (string, error) {
	manifest.ManifestSHA256 = ""
	return hashAccountingValue(manifest)
}

func hashAccountingExceptions(report Technology20AccountingExceptions) (string, error) {
	report.ReportSHA256 = ""
	return hashAccountingValue(report)
}

func hashAccountingReviewPacket(packet AccountingProfessionalReviewPacket) (string, error) {
	packet.PacketSHA256 = ""
	return hashAccountingValue(packet)
}

func hashAccountingValue(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
