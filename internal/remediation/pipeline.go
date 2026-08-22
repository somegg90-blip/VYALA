package remediation

import (
	"context"
	"time"

	"vyala/internal/findings"
)

const RecordSchemaVersion = "remediation-record/1"

// Pipeline runs Triage -> Context -> Planner for one finding and emits a
// RemediationRecord. Every run is captured for future fine-tuning.
type Pipeline struct {
	Model Model
}

func (p *Pipeline) Run(ctx context.Context, repoRoot string, f findings.Finding) (*RemediationRecord, error) {
	started := time.Now()

	rec := &RemediationRecord{
		SchemaVersion: RecordSchemaVersion,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Finding:       f,
		Outcome:       "planned",
		Model:         modelName(p.Model),
	}

	bundle, err := GatherContext(repoRoot, f)
	if err != nil {
		// Degrade gracefully instead of failing: the CBOM may have been
		// produced elsewhere (e.g. CI) or the file moved since the scan.
		// The planner can still produce a guidance-grounded plan from the
		// finding alone; the degraded run is visible in telemetry.
		bundle = nil
	}
	rec.Context = bundle

	var triage *TriageResult
	if bundle != nil {
		triage, err = runTriage(ctx, p.Model, f, bundle)
		if err != nil {
			rec.Error = err.Error()
			return rec, err
		}
		rec.Triage = triage
	}

	plan, err := runPlanner(ctx, p.Model, f, bundle, triage, KB(f.Category))
	if err != nil {
		rec.Error = err.Error()
		return rec, err
	}
	rec.Plan = plan
	rec.LatencyMS = time.Since(started).Milliseconds()

	return rec, nil
}

// modelName extracts a display name from the model implementation without
// requiring the interface to expose it.
func modelName(m Model) string {
	if oc, ok := m.(*OllamaClient); ok {
		return oc.Model
	}
	return "unknown"
}
