package remediation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vyala/internal/findings"
)

const triageJSON = `{"complexity": "drop_in", "risk_notes": ["verify peers support hybrid KEM"], "priority": "high"}`
const planJSON = `{
  "summary": "Swap RSA-OAEP encryption to ML-KEM hybrid encapsulation.",
  "steps": [
    {"title": "Add ML-KEM dependency", "detail": "Use crypto/mlkem (Go 1.24+)."},
    {"title": "Replace Encrypt call", "detail": "Encapsulate with hybrid X25519MLKEM768.", "file": "main.go", "code_before": "rsa.EncryptOAEP(...)", "code_after": "mlkem768.Encapsulate(...)"}
  ],
  "risks": ["Receivers must support ML-KEM before rollout"],
  "references": ["FIPS 203", "RFC 10024"]
}`

type mockModel struct{ fail bool }

func (m mockModel) Complete(_ context.Context, req CompletionRequest) (string, error) {
	if m.fail {
		return "", context.DeadlineExceeded
	}
	if strings.Contains(req.System, "VYALA-Triage") {
		return triageJSON, nil
	}
	return planJSON, nil
}

func fixture(t *testing.T) (string, findings.Finding) {
	t.Helper()
	root := t.TempDir()
	code := "package main\n\nimport \"crypto/rsa\"\n\nfunc Enc(pub interface{}, msg []byte) ([]byte, error) {\n\treturn nil, nil // line 6 placeholder\n}\n"
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}
	f := findings.Finding{
		ID: "vy-test00000001", Type: "code", File: "main.go", Line: 6,
		Algorithm: "RSA", Severity: "medium", Category: "rsa_encryption",
		SuggestedReplacement: "Use ML-KEM hybrid.", RuleID: "pqc-rsa-encryption-go",
	}
	return root, f
}

func TestPipelineEndToEnd(t *testing.T) {
	root, f := fixture(t)
	pipe := &Pipeline{Model: mockModel{}}

	rec, err := pipe.Run(context.Background(), root, f)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	if rec.SchemaVersion != RecordSchemaVersion || rec.Outcome != "planned" {
		t.Errorf("record header wrong: %+v", rec)
	}
	if rec.Context == nil || rec.Context.Language != "go" || rec.Context.StartLine != 1 {
		t.Errorf("context bundle wrong: %+v", rec.Context)
	}
	if !strings.Contains(rec.Context.Snippet, "6: \treturn nil, nil") {
		t.Errorf("snippet missing numbered target line:\n%s", rec.Context.Snippet)
	}
	if len(rec.Context.Imports) != 1 || !strings.Contains(rec.Context.Imports[0], "crypto/rsa") {
		t.Errorf("imports not extracted: %v", rec.Context.Imports)
	}
	if rec.Triage == nil || rec.Triage.Complexity != "drop_in" {
		t.Errorf("triage missing/wrong: %+v", rec.Triage)
	}
	if rec.Plan == nil || len(rec.Plan.Steps) != 2 || rec.Plan.Steps[1].CodeAfter == "" {
		t.Errorf("plan missing/wrong: %+v", rec.Plan)
	}
	if rec.Model != "mock" && rec.Model != "unknown" {
		// non-Ollama model -> display name unknown
		t.Logf("model name = %q", rec.Model)
	}
}

func TestTelemetryRoundTripAndExport(t *testing.T) {
	root, f := fixture(t)
	pipe := &Pipeline{Model: mockModel{}}
	rec, err := pipe.Run(context.Background(), root, f)
	if err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(root, ".vyala", "remediations.jsonl")
	if err := NewTelemetryLog(logPath).Append(rec); err != nil {
		t.Fatalf("append: %v", err)
	}
	// append an incomplete record that must be excluded from export
	bad := &RemediationRecord{SchemaVersion: RecordSchemaVersion, Timestamp: "x", Outcome: "rejected",
		Finding: findings.Finding{ID: "vy-bad000000001"}}
	if err := NewTelemetryLog(logPath).Append(bad); err != nil {
		t.Fatal(err)
	}

	recs, err := NewTelemetryLog(logPath).ReadRecords()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("records = %d, want 2", len(recs))
	}
	if recs[0].Finding.ID != f.ID || recs[0].Plan == nil {
		t.Errorf("round-trip lost plan data: %+v", recs[0])
	}

	out := filepath.Join(root, "pairs.jsonl")
	n, err := ExportTrainingPairs(logPath, out)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if n != 1 {
		t.Fatalf("exported %d pairs, want 1 (incomplete records excluded)", n)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), `"instruction"`) || !strings.Contains(string(data), `"output"`) {
		t.Errorf("training pair JSONL malformed:\n%s", data)
	}
	if !strings.Contains(string(data), "FIPS-grounded") {
		t.Error("instruction prompt missing")
	}
}

func TestPipelineCapturesModelErrorInRecord(t *testing.T) {
	root, f := fixture(t)
	pipe := &Pipeline{Model: mockModel{fail: true}}
	rec, err := pipe.Run(context.Background(), root, f)
	if err == nil {
		t.Fatal("expected error from failing model")
	}
	if rec.Error == "" {
		t.Error("error not captured in telemetry record — training data would lose this failure mode")
	}
}

func TestExtractJSONStringFences(t *testing.T) {
	in := "Sure! Here is the plan:\n```json\n{\"a\": 1}\n```\nHope that helps."
	if got := ExtractJSONString(in); got != `{"a": 1}` {
		t.Errorf("ExtractJSONString = %q", got)
	}
}
