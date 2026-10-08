package httpx

import (
	"errors"
	"strings"
	"testing"
)

func TestMetaOAuthFailureReasonDoesNotExposeGraphDetails(t *testing.T) {
	detail := "meta graph error 190: secret diagnostic detail"
	reason := metaOAuthFailureReason(errors.New("exchange long-lived token: " + detail))
	if strings.Contains(reason, "secret diagnostic detail") || strings.Contains(reason, "190") {
		t.Fatalf("reason exposed internal Graph details: %q", reason)
	}
	if !strings.Contains(reason, "could not renew") {
		t.Fatalf("reason = %q, want renewal-stage guidance", reason)
	}
}

func TestMetaOAuthFailureReasonIdentifiesPersistenceFailure(t *testing.T) {
	reason := metaOAuthFailureReason(errors.New("save encrypted Meta authorization: database offline"))
	if !strings.Contains(reason, "could not save") {
		t.Fatalf("reason = %q, want persistence guidance", reason)
	}
}
