package remediation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"vyala/internal/findings"
)

// RemediationRecord captures EVERYTHING about one remediation-planning run:
// the input finding, the exact context the model saw, both agent outputs,
// model identity, latency, and lifecycle outcome. Records are appended as
// JSONL so thousands of runs accumulate into an append-only dataset.
type RemediationRecord struct {
	SchemaVersion string           `json:"schema_version"`
	Timestamp     string           `json:"timestamp"`
	Finding       findings.Finding `json:"finding"`
	Context       *ContextBundle   `json:"context,omitempty"`
	Triage        *TriageResult    `json:"triage,omitempty"`
	Plan          *PlanResult      `json:"plan,omitempty"`
	Model         string           `json:"model,omitempty"`
	LatencyMS     int64            `json:"latency_ms,omitempty"`
	Error         string           `json:"error,omitempty"`

	// Outcome tracks the human/verifier lifecycle:
	// planned -> applied -> verified | rejected | superseded
	Outcome string `json:"outcome"`
}

// ToTrainingPair renders the record as a supervised fine-tuning example:
// instruction + input (what the small model saw) -> output (accepted plan).
// Records with errors or rejected outcomes are excluded by the exporter.
func (r *RemediationRecord) ToTrainingPair() (map[string]string, error) {
	if r.Plan == nil || r.Context == nil || r.Triage == nil || r.Error != "" {
		return nil, fmt.Errorf("record %s is not a complete successful run", r.Finding.ID)
	}
	input := map[string]any{
		"finding": r.Finding,
		"context": map[string]any{
			"language":   r.Context.Language,
			"snippet":    r.Context.Snippet,
			"imports":    r.Context.Imports,
			"start_line": r.Context.StartLine,
			"end_line":   r.Context.EndLine,
		},
		"kb_excerpt": KB(r.Finding.Category),
	}
	output := map[string]any{
		"triage":     r.Triage,
		"plan":       r.Plan,
	}
	instruction := "You are a post-quantum cryptography migration planner. Given a quantum-vulnerability finding, " +
		"the surrounding code context, and authoritative NIST migration guidance, produce a triage assessment and " +
		"a concrete, FIPS-grounded remediation plan. Prefer hybrid classical+PQC constructions and cite references."
	inJSON, _ := json.Marshal(input)
	outJSON, _ := json.Marshal(output)
	return map[string]string{
		"instruction": instruction,
		"input":       string(inJSON),
		"output":      string(outJSON),
	}, nil
}

// TelemetryLog is an append-only JSONL file of RemediationRecords.
type TelemetryLog struct {
	Path string
}

func NewTelemetryLog(path string) *TelemetryLog { return &TelemetryLog{Path: path} }

// Append writes one record as a single JSON line, creating directories as
// needed. Appends are crash-safe: a torn final line is ignored on read.
func (t *TelemetryLog) Append(rec *RemediationRecord) error {
	if t.Path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(t.Path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(t.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// ReadRecords loads all complete records from the log, skipping torn lines.
func (t *TelemetryLog) ReadRecords() ([]*RemediationRecord, error) {
	f, err := os.Open(t.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []*RemediationRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec RemediationRecord
		if err := json.Unmarshal(line, &rec); err == nil {
			out = append(out, &rec)
		}
	}
	return out, sc.Err()
}

// ExportTrainingPairs converts all successful records into fine-tuning
// examples and writes them as JSONL.
func ExportTrainingPairs(logPath, outPath string) (int, error) {
	tl := NewTelemetryLog(logPath)
	recs, err := tl.ReadRecords()
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return 0, err
	}
	out, err := os.Create(outPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	n := 0
	for _, rec := range recs {
		pair, err := rec.ToTrainingPair()
		if err != nil {
			continue
		}
		b, err := json.Marshal(pair)
		if err != nil {
			continue
		}
		if _, err := out.Write(append(b, '\n')); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
