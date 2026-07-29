package main

import (
	"strings"
	"testing"

	"github.com/rvbernucci/signalforge/internal/productscope"
)

func TestNamedProfessionalDecisionIsBoundToExactRegistryHash(t *testing.T) {
	review := productscope.AccountingProfessionalReviewPacket{
		RegistrySHA256: acceptedProfessionalReviewRegistrySHA,
	}
	accepted := professionalReviewMarkdown(review)
	for _, required := range []string{
		"Named professional decision: `CONDITIONALLY_ACCEPTED`",
		"Reviewer: `" + namedProfessionalReviewer + "`",
		"Machine decision encoding: `HASH_BOUND_CONDITIONALLY_ACCEPTED`",
		"Runtime activation: `ACTIVE_AT_EXACT_SCOPE_FAIL_CLOSED`",
	} {
		if !strings.Contains(accepted, required) {
			t.Fatalf("accepted exact-hash review lacks %q", required)
		}
	}

	review.RegistrySHA256 = strings.Repeat("0", 64)
	pending := professionalReviewMarkdown(review)
	if !strings.Contains(pending, "Named professional decision: `PENDING`") {
		t.Fatal("changed registry hash did not return named decision to pending")
	}
	if strings.Contains(pending, "Reviewer: `"+namedProfessionalReviewer+"`") ||
		strings.Contains(pending, "Named professional decision: `CONDITIONALLY_ACCEPTED`") {
		t.Fatal("changed registry hash inherited an approval for different content")
	}
}
