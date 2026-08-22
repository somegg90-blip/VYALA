package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"vyala/internal/findings"
)

// ---------------------------------------------------------------------------
// Triage agent
// ---------------------------------------------------------------------------

type TriageResult struct {
	Complexity string   `json:"complexity"` // drop_in | config_change | architectural
	RiskNotes  []string `json:"risk_notes"`
	Priority   string   `json:"priority"` // high | medium | low
}

const triageSchema = `{
  "complexity": "drop_in" | "config_change" | "architectural",
  "risk_notes": ["short strings"],
  "priority": "high" | "medium" | "low"
}`

func runTriage(ctx context.Context, m Model, f findings.Finding, c *ContextBundle) (*TriageResult, error) {
	system := `You are VYALA-Triage, a post-quantum cryptography migration triage expert.
Classify how hard the remediation of one finding is:
- drop_in: a library call swap or config line change
- config_change: infrastructure/TLS policy edit
- architectural: protocol-level change affecting peers/interop
Respond ONLY with JSON matching exactly this schema (no prose):
` + triageSchema

	user := fmt.Sprintf("Finding:\n%s\n\nCode context (%s lines %d-%d):\n%s",
		mustJSON(f), c.File, c.StartLine, c.EndLine, c.Snippet)

	raw, err := m.Complete(ctx, CompletionRequest{System: system, User: user, Temperature: 0.1})
	if err != nil {
		return nil, fmt.Errorf("triage: %w", err)
	}
	var t TriageResult
	if err := json.Unmarshal([]byte(ExtractJSONString(raw)), &t); err != nil {
		return nil, fmt.Errorf("triage: parsing model JSON: %w", err)
	}
	switch t.Complexity {
	case "drop_in", "config_change", "architectural":
	default:
		t.Complexity = "architectural"
	}
	switch t.Priority {
	case "high", "medium", "low":
	default:
		t.Priority = "medium"
	}
	return &t, nil
}

// ---------------------------------------------------------------------------
// Planner agent
// ---------------------------------------------------------------------------

type PlanStep struct {
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	File       string `json:"file,omitempty"`
	CodeBefore string `json:"code_before,omitempty"`
	CodeAfter  string `json:"code_after,omitempty"`
}

type PlanResult struct {
	Summary    string     `json:"summary"`
	Steps      []PlanStep `json:"steps"`
	Risks      []string   `json:"risks"`
	References []string   `json:"references"`
}

const planSchema = `{
  "summary": "one paragraph",
  "steps": [{"title": "...", "detail": "...", "file": "optional path", "code_before": "optional", "code_after": "optional"}],
  "risks": ["interoperability/perf/security risks"],
  "references": ["FIPS 203", "RFC 10024", ...]
}`

func runPlanner(ctx context.Context, m Model, f findings.Finding, c *ContextBundle, t *TriageResult, kb string) (*PlanResult, error) {
	system := `You are VYALA-Planner, a post-quantum cryptography migration planner.
Produce a concrete remediation plan for ONE quantum-vulnerability finding.
Ground every recommendation in the AUTHORITATIVE GUIDANCE below; never invent
library names or APIs that are not listed there. Prefer hybrid (classical+PQC)
constructions. Steps must be actionable by the developer who owns this code.
AUTHORITATIVE GUIDANCE:
` + kb + `
Respond ONLY with JSON matching exactly this schema (no prose):
` + planSchema

	user := fmt.Sprintf("Finding:\n%s\n\nTriage assessment:\n%s\n\nCode context (%s lines %d-%d):\n%s",
		mustJSON(f), mustJSON(t), c.File, c.StartLine, c.EndLine, c.Snippet)

	raw, err := m.Complete(ctx, CompletionRequest{System: system, User: user, Temperature: 0.2})
	if err != nil {
		return nil, fmt.Errorf("planner: %w", err)
	}
	var p PlanResult
	if err := json.Unmarshal([]byte(ExtractJSONString(raw)), &p); err != nil {
		return nil, fmt.Errorf("planner: parsing model JSON: %w", err)
	}
	if p.Summary == "" || len(p.Steps) == 0 {
		return nil, fmt.Errorf("planner returned empty plan")
	}
	return &p, nil
}

// ExtractJSONString wraps ExtractJSON for direct use on raw model output.
func ExtractJSONString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start == -1 || end == -1 || end <= start {
		return s
	}
	return s[start : end+1]
}

func mustJSON(v any) string {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
