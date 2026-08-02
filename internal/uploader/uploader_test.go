package uploader

import (
	"testing"

	"vyala/internal/findings"
)

func TestBuildPayloadIncludesSuccessStatus(t *testing.T) {
	cbom := findings.CBOM{}
	payload := buildPayload(cbom, "demo/repo", "abc123", "main", "push", 42, true)

	if payload.Status != "success" {
		t.Fatalf("expected status success, got %q", payload.Status)
	}
	if payload.RepoFullName != "demo/repo" {
		t.Fatalf("expected repo full name to be preserved")
	}
	if payload.IsPrivate != true {
		t.Fatalf("expected private flag to be preserved")
	}
}
