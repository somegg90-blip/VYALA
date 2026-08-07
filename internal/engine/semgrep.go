package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vyala/internal/findings"
)

type SemgrepOutput struct {
	Results []SemgrepResult `json:"results"`
}

type SemgrepResult struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Start   struct {
		Line int `json:"line"`
	} `json:"start"`
	Extra struct {
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Metadata struct {
			Category    string `json:"category"`
			Algorithm   string `json:"algorithm"`
			Replacement string `json:"suggested_replacement"`
			Exposure    string `json:"exposure_estimate"`
		} `json:"metadata"`
	} `json:"extra"`
}

// findRulesDir dynamically locates the Semgrep rules directory.
// It returns the ABSOLUTE path to prevent any working-directory ambiguity.
func findRulesDir() string {
	candidates := []string{
		"/etc/vyala/rules",        // 1. Production Docker container
		"internal/rules",         // 2. Running locally from the repo root
		"../rules",               // 3. Running tests from internal/engine/
		"../../internal/rules",   // 4. Running tests from deeper directories
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			absPath, err := filepath.Abs(dir)
			if err == nil {
				return absPath
			}
			return dir
		}
	}
	// Fallback
	absPath, _ := filepath.Abs("internal/rules")
	return absPath
}

func Scan(repoRoot string, targets []string) (findings.CBOM, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	rulesPath := findRulesDir()
	
	args := []string{
		"--json",
		"--timeout", "30", 
		"--max-target-bytes", "1000000", 
		"--config", rulesPath,
	}

	// If scanning the whole repo (no specific targets), exclude the rules directory 
	// to prevent Semgrep from scanning its own YAML files.
	if len(targets) == 0 {
		args = append(args, "--exclude", rulesPath)
	}

	if len(targets) > 0 {
		args = append(args, targets...)
	} else {
		args = append(args, repoRoot)
	}

	cmd := exec.CommandContext(ctx, "semgrep", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	if stdout.Len() == 0 {
		if ctx.Err() == context.DeadlineExceeded {
			return findings.CBOM{}, fmt.Errorf("semgrep timed out after 5 minutes")
		}
		return findings.CBOM{}, fmt.Errorf("semgrep execution failed: %w\nstderr: %s", err, stderr.String())
	}

	var out SemgrepOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return findings.CBOM{}, fmt.Errorf("failed to parse semgrep JSON: %w", err)
	}

	cbom := findings.CBOM{
		Version:   findings.SchemaVersion,
		Generated: time.Now().UTC(),
		Findings:  make([]findings.Finding, 0, len(out.Results)),
	}

	for _, r := range out.Results {
		relPath, _ := filepath.Rel(repoRoot, r.Path)
		if relPath == "" {
			relPath = r.Path
		}

		// Double-check that we skip findings inside the rules directory
		if strings.Contains(relPath, "internal/rules") || strings.Contains(relPath, "internal\\rules") || strings.HasPrefix(r.Path, rulesPath) {
			continue
		}

		// FIX: Robustly strip the directory prefix Semgrep adds to the Rule ID.
		// Semgrep prefixes IDs with the folder structure (e.g., "internal.rules.my-rule").
		// Splitting by "." and taking the last segment safely extracts the actual rule ID.
		parts := strings.Split(r.CheckID, ".")
		ruleID := parts[len(parts)-1]

		severity := strings.ToUpper(r.Extra.Severity)
		if severity == "ERROR" {
			severity = "HIGH"
		} else if severity == "WARNING" {
			severity = "MEDIUM"
		} else {
			severity = "LOW"
		}

		cbom.Findings = append(cbom.Findings, findings.Finding{
			ID:                   findings.GenerateFindingID(ruleID, relPath, r.Start.Line),
			Type:                 "code", 
			File:                 relPath,
			Line:                 r.Start.Line,
			Algorithm:            r.Extra.Metadata.Algorithm,
			Severity:             strings.ToLower(severity),
			Category:             r.Extra.Metadata.Category,
			ExposureEstimate:     r.Extra.Metadata.Exposure,
			SuggestedReplacement: r.Extra.Metadata.Replacement,
			RuleID:               ruleID,
		})
	}

	return cbom, nil
}