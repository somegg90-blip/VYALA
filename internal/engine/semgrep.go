package engine

import (
	"bufio"
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
	"vyala/internal/rules"
	"vyala/internal/severity"
)

type semgrepMetadata struct {
	Algorithm            string `json:"algorithm"`
	Category             string `json:"category"`
	SuggestedReplacement string `json:"suggested_replacement"`
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

// readIgnorePatterns reads a .vyalaignore file from the repoRoot and returns a slice of patterns.
func readIgnorePatterns(repoRoot string) []string {
	ignoreFile := filepath.Join(repoRoot, ".vyalaignore")
	file, err := os.Open(ignoreFile)
	if err != nil {
		return nil
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns
}

// scanTimeout bounds how long a single semgrep invocation may run, so a
// pathologically large repo (or a runaway pattern match) can't hang CI indefinitely.
const scanTimeout = 5 * time.Minute

func Scan(repoRoot string, targets []string) (findings.CBOM, error) {
	ruleDir, err := rules.ExtractToTemp()
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("extracting rules: %w", err)
	}

	args := []string{
		"--config", ruleDir,
		"--json",
		"--no-git-ignore",
		"--exclude", "internal/rules",
		"--exclude", "node_modules",
		"--exclude", "vendor",
		"--exclude", "dist",
		"--exclude", "build",
		"--exclude", ".git",
	}

	ignorePatterns := readIgnorePatterns(repoRoot)
	for _, pattern := range ignorePatterns {
		args = append(args, "--exclude", pattern)
	}

	if len(targets) > 0 {
		args = append(args, targets...)
	} else {
		args = append(args, repoRoot)
	}

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "semgrep", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return findings.CBOM{}, fmt.Errorf("semgrep scan timed out after %s (repo may be too large, or hit a pathological pattern match)", scanTimeout)
	}
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
