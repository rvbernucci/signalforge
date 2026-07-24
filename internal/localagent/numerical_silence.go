package localagent

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var numericalLiteralPattern = regexp.MustCompile(`[0-9]+(?:[.,][0-9]+)*%?`)
var numericalWordPattern = regexp.MustCompile(`(?i)\b(?:zero|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety|hundred|thousand|million|billion|trillion|single|double|triple|half|quarter)\b`)
var singleEvidenceArtifactPattern = regexp.MustCompile(`(?i)\b(?:single|one)\s+(?:provided\s+)?(?:(?:held[- ]out|retrieved)\s+)?(?:evidence(?:\s+source)?|source|fixture|document)\b`)

func containsAuthoritativeNumericalLiteral(value string) bool {
	if numericalWordPattern.MatchString(value) {
		return true
	}
	for _, token := range numericalLiteralPattern.FindAllString(value, -1) {
		if allowedCalendarYear(token) {
			continue
		}
		return true
	}
	return false
}

func redactFinancialNumerics(value string) string {
	redacted := numericalLiteralPattern.ReplaceAllStringFunc(value, func(token string) string {
		if allowedCalendarYear(token) {
			return token
		}
		return "[value withheld]"
	})
	return numericalWordPattern.ReplaceAllString(redacted, "[value withheld]")
}

func allowedCalendarYear(token string) bool {
	if len(token) != 4 || strings.ContainsAny(token, ".,%") {
		return false
	}
	year, err := strconv.Atoi(token)
	return err == nil && year >= 1900 && year <= 2200
}

func validateNumericallySilentDraft(body finalBody) error {
	for _, section := range body.Sections {
		if containsAuthoritativeNumericalLiteral(section.Title) || containsAuthoritativeNumericalLiteral(section.Content) {
			return fmt.Errorf("section %q crossed the numerical-silence boundary", section.SectionType)
		}
	}
	for _, group := range [][]string{body.Assumptions, body.Limitations, body.NextActions} {
		for _, value := range group {
			if containsAuthoritativeNumericalLiteral(value) {
				return fmt.Errorf("final semantic metadata crossed the numerical-silence boundary")
			}
		}
	}
	return nil
}

// Internal identifiers already travel through structured reference fields. Repeating them in
// user-facing prose adds no authority and can make their numeric suffixes look like financial
// values to the numerical-silence guard. Only identifiers present in the approved synthesis
// material are replaced; arbitrary numbers and unknown identifiers remain untouched.
func neutralizeInternalReferenceMentions(body *finalBody, material synthesisPromptInput) {
	replacements := map[string]string{}
	for _, claim := range material.Claims {
		replacements[claim.Finding.ClaimID] = "the approved claim"
		for _, evidenceID := range claim.Finding.EvidenceRefs {
			replacements[evidenceID] = "the approved evidence"
		}
		for _, receiptID := range claim.Finding.CalculationRefs {
			replacements[receiptID] = "the validated calculation receipt"
		}
		for _, numericalID := range claim.Finding.NumericalRefs {
			replacements[numericalID] = "the validated numerical reference"
		}
	}
	for _, evidence := range material.Evidence {
		replacements[evidence.EvidenceID] = "the approved evidence"
	}
	for _, receipt := range material.Receipts {
		replacements[receipt.ReceiptID] = "the validated calculation receipt"
	}

	identifiers := make([]string, 0, len(replacements))
	for identifier := range replacements {
		if strings.TrimSpace(identifier) != "" {
			identifiers = append(identifiers, identifier)
		}
	}
	sort.Slice(identifiers, func(i, j int) bool {
		return len(identifiers[i]) > len(identifiers[j])
	})

	replace := func(value string) string {
		for _, identifier := range identifiers {
			pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(identifier) + `\b`)
			value = pattern.ReplaceAllString(value, replacements[identifier])
		}
		if len(material.Evidence) == 1 {
			value = singleEvidenceArtifactPattern.ReplaceAllString(value, "provided evidence set")
		}
		return value
	}
	for index := range body.Sections {
		body.Sections[index].Title = replace(body.Sections[index].Title)
		body.Sections[index].Content = replace(body.Sections[index].Content)
	}
	for _, group := range [][]string{body.Assumptions, body.Limitations, body.NextActions} {
		for index := range group {
			group[index] = replace(group[index])
		}
	}
}
