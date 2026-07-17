package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"vyala/internal/findings"
	"vyala/internal/rules"
	"vyala/internal/severity"
)

type semgrepMetadata struct {
	Algorithm             string `json:"algorithm"`
	Category              string `json:"category"`
	SuggestedReplacement  string `json:"suggested_replacement"`
}

type semgrepExtra struct {
	Message  string          `json:"message"`
	Metadata semgrepMetadata `json:"metadata"`
}

type semgrepPosition struct {
	Line int `json:"line"`
}

type semgrepResult struct {
	CheckID string          `json:"check_id"`
	Path    string          `json:"path"`
	Start   semgrepPosition `json:"start"`
	Extra   semgrepExtra    `json:"extra"`
}

type semgrepOutput struct {
	Results []semgrepResult `json:"results"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func Scan(repoRoot string, targets []string) (findings.CBOM, error) {
	ruleDir, err := rules.ExtractToTemp()
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("extracting rules: %w", err)
	}

	args := []string{"--config", ruleDir, "--json", "--no-git-ignore"}
	if len(targets) > 0 {
		args = append(args, targets...)
	} else {
		args = append(args, repoRoot)
	}

	cmd := exec.Command("semgrep", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr != nil {
		if _, ok := runErr.(*exec.ExitError); !ok {
			return findings.CBOM{}, fmt.Errorf("running semgrep: %w (stderr: %s)", runErr, stderr.String())
		}
	}

	var parsed semgrepOutput
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		return findings.CBOM{}, fmt.Errorf("parsing semgrep output: %w (stderr: %s)", err, stderr.String())
	}

	cbom := findings.CBOM{
		Version:   findings.SchemaVersion,
		Generated: time.Now().UTC(),
		Findings:  []findings.Finding{},
	}
	for _, r := range parsed.Results {
		relPath := findings.NormalizeRelPath(repoRoot, r.Path)
		sev := severity.Classify(relPath)

		if r.Extra.Metadata.Category == "key_loading" {
			sev = "low"
		}

		stableRuleID := normalizeCheckID(r.CheckID)
		cbom.Findings = append(cbom.Findings, findings.Finding{
			ID:                   findings.GenerateFindingID(stableRuleID, relPath, r.Start.Line),
			File:                 relPath,
			Line:                 r.Start.Line,
			Algorithm:            r.Extra.Metadata.Algorithm,
			Severity:             sev,
			Category:             r.Extra.Metadata.Category,
			ExposureEstimate:     severity.ExposureEstimate(relPath, sev),
			SuggestedReplacement: r.Extra.Metadata.SuggestedReplacement,
			RuleID:               stableRuleID,
		})
	}

	return cbom, nil
}

func normalizeCheckID(checkID string) string {
	parts := strings.Split(checkID, ".")
	return parts[len(parts)-1]
}