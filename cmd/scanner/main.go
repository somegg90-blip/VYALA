package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vyala/internal/cyclonedx"
	"vyala/internal/diffscan"
	"vyala/internal/engine"
	"vyala/internal/findings"
	"vyala/internal/ghcomment"
	"vyala/internal/remediation"
	"vyala/internal/uploader"
)

var relevantExtensions = map[string]bool{
	".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true,
	".go": true, ".java": true,
	".c": true, ".cc": true, ".cpp": true, ".cxx": true,
	".h": true, ".hpp": true, ".hh": true,
	".cs": true, ".rs": true,
	// Infrastructure-as-Code surfaces
	".tf": true, ".yaml": true, ".yml": true,
}

var relevantFilenames = map[string]bool{
	"package.json":     true,
	"requirements.txt": true,
	"pyproject.toml":   true,
	"go.mod":           true,
}

func main() {
	path := flag.String("path", ".", "repo path to scan")
	jsonOut := flag.String("json", "", "write CBOM JSON to this path (optional)")
	cdxOut := flag.String("cyclonedx", "", "write CycloneDX 1.6 CBOM to this path (interoperable with SBOM/GRC tooling)")
	diffBase := flag.String("diff-base", "", "if set, scan only files changed vs this git ref/SHA")
	severityThreshold := flag.String("severity-threshold", "medium", "minimum severity to show in detail in PR comment")
	failOn := flag.String("fail-on", "", "exit non-zero if any finding at or above this severity")
	postPRComment := flag.Bool("post-pr-comment", false, "post/update a PR comment with results")
	commentFromFile := flag.String("comment-from-file", "", "post PR comment using the CBOM JSON at this path (no scan)")

	prNumber := flag.Int("pr-number", 0, "PR number (for comment posting, overrides GITHUB_EVENT_PATH)")
	headSHA := flag.String("head-sha", "", "Head commit SHA (for comment posting, overrides GITHUB_EVENT_PATH)")

	probeEndpoints := flag.String("probe-endpoints", "", "comma-separated list of host:port endpoints to probe for TLS PQC readiness")

	uploadURL := flag.String("upload-url", "", "API endpoint to upload CBOM")
	apiKey := flag.String("api-key", "", "API key for backend upload authentication")

	// ---- Experimental: LOCAL-ONLY AI remediation. Not part of CI/PR flows. ----
	planFrom := flag.String("plan-from", "", "[experimental] generate a local AI remediation plan from this CBOM JSON")
	findingID := flag.String("finding", "", "[experimental] finding ID to remediate (requires -plan-from)")
	planOut := flag.String("plan-out", "", "[experimental] write the plan JSON to this path")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "[experimental] local Ollama server URL")
	remModel := flag.String("remediation-model", "qwen2.5-coder:7b-instruct-q4_K_M", "[experimental] local model tag")
	remLog := flag.String("remediation-log", ".vyala/remediations.jsonl", "[experimental] telemetry JSONL path")
	exportPairs := flag.String("export-training-pairs", "", "[experimental] export fine-tuning pairs to this JSONL (uses -remediation-log)")

	flag.Parse()

	// FIX: Standardized to VYALA_API_KEY to match the GitHub Action workflow
	if *apiKey == "" {
		*apiKey = os.Getenv("VYALA_API_KEY")
	}

	if *commentFromFile != "" {
		if err := postCommentFromFile(*commentFromFile, *severityThreshold, *prNumber, *headSHA); err != nil {
			fatal("posting comment from file: %v", err)
		}
		return
	}

	// ---- Experimental: local-only AI remediation (never runs in CI) ----
	if *exportPairs != "" {
		n, err := remediation.ExportTrainingPairs(*remLog, *exportPairs)
		if err != nil {
			fatal("exporting training pairs: %v", err)
		}
		fmt.Printf("Exported %d fine-tuning pair(s) to %s\n", n, *exportPairs)
		return
	}
	if *planFrom != "" {
		repoRoot, perr := filepath.Abs(*path)
		if perr != nil {
			fatal("resolving path: %v", perr)
		}
		if err := runRemediationPlan(remediationPlanOpts{
			cbolPath:  *planFrom,
			findingID: *findingID,
			repoRoot:  repoRoot,
			planOut:   *planOut,
			ollamaURL: *ollamaURL,
			model:     *remModel,
			logPath:   *remLog,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "vyala: warning: %v\n", err)
			os.Exit(1)
		}
		return
	}

	repoRoot, err := filepath.Abs(*path)
	if err != nil {
		fatal("resolving path: %v", err)
	}

	var cbom findings.CBOM

	if *diffBase != "" {
		cbom, err = diffScan(repoRoot, *diffBase)
	} else {
		cbom, err = fullScan(repoRoot)
	}
	if err != nil {
		fatal("scan failed: %v", err)
	}

	if *probeEndpoints != "" {
		endpoints := strings.Split(*probeEndpoints, ",")
		for i, e := range endpoints {
			endpoints[i] = strings.TrimSpace(e)
		}
		tlsFindings, err := engine.ScanTLSProbes(endpoints)
		if err != nil {
			fatal("tls probe failed: %v", err)
		}
		cbom.Findings = append(cbom.Findings, tlsFindings...)
	}

	findings.WriteTerminalReport(os.Stdout, cbom)

	if *jsonOut != "" {
		if err := findings.WriteJSON(cbom, *jsonOut); err != nil {
			fatal("writing JSON: %v", err)
		}
	}

	if *cdxOut != "" {
		projectName := filepath.Base(repoRoot)
		cdx := cyclonedx.Convert(cbom, projectName)
		if err := cyclonedx.Write(cdx, *cdxOut); err != nil {
			fatal("writing CycloneDX CBOM: %v", err)
		}
		fmt.Printf("CycloneDX %s CBOM written to %s (%d components)\n", cyclonedx.SpecVersion, *cdxOut, len(cdx.Components))
	}

	if *uploadURL != "" {
		if err := uploader.UploadCBOM(cbom, *uploadURL, *apiKey); err != nil {
			fmt.Fprintf(os.Stderr, "vyala: warning: failed to upload CBOM: %v\n", err)
		}
	}

	if *postPRComment {
		if err := postComment(cbom, *severityThreshold, *prNumber, *headSHA); err != nil {
			fatal("posting PR comment: %v", err)
		}
	}

	if *failOn != "" {
		for _, f := range cbom.Findings {
			if findings.MeetsThreshold(f.Severity, *failOn) {
				fmt.Fprintf(os.Stderr, "\nvyala: failing check -- at least one finding at or above severity %q\n", *failOn)
				os.Exit(1)
			}
		}
	}
}

func postCommentFromFile(path, severityThreshold string, prNumber int, headSHA string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading CBOM file: %w", err)
	}
	var cbom findings.CBOM
	if err := json.Unmarshal(data, &cbom); err != nil {
		return fmt.Errorf("parsing CBOM file: %w", err)
	}
	return postComment(cbom, severityThreshold, prNumber, headSHA)
}

func postComment(cbom findings.CBOM, severityThreshold string, prNumber int, headSHA string) error {
	var ev *ghcomment.PREvent
	var err error

	if prNumber > 0 && headSHA != "" {
		ev = &ghcomment.PREvent{
			PullRequest: struct {
				Number int `json:"number"`
				Base   struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				} `json:"base"`
				Head struct {
					SHA string `json:"sha"`
				} `json:"head"`
			}{
				Number: prNumber,
				Head: struct {
					SHA string `json:"sha"`
				}{SHA: headSHA},
			},
		}
	} else {
		ev, err = ghcomment.LoadEventFromEnv()
		if err != nil {
			return err
		}
	}

	cfg, err := ghcomment.ConfigFromEnv()
	if err != nil {
		return err
	}
	body := ghcomment.RenderComment(cbom, cfg.Repo, ev.PullRequest.Head.SHA, severityThreshold)
	return ghcomment.PostOrUpdate(cfg, ev.PullRequest.Number, body)
}

func fullScan(repoRoot string) (findings.CBOM, error) {
	cbom := findings.CBOM{
		Version:   findings.SchemaVersion,
		Generated: time.Now().UTC(),
		Findings:  []findings.Finding{},
	}

	if semgrepAvailable() {
		codeCBOM, err := engine.Scan(repoRoot, nil)
		if err != nil {
			return findings.CBOM{}, fmt.Errorf("code scan failed: %w", err)
		}
		cbom.Findings = append(cbom.Findings, codeCBOM.Findings...)
	} else {
		fmt.Fprintln(os.Stderr, "vyala: warning: semgrep not found on PATH — skipping code rules scan (dependency & TLS scans still run). Install with `pip install semgrep` for full coverage.")
	}

	depFindings, err := engine.ScanManifests(repoRoot, nil)
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("dependency scan failed: %w", err)
	}

	cbom.Findings = append(cbom.Findings, depFindings...)
	return cbom, nil
}

// semgrepAvailable reports whether the code-scanning engine can run.
func semgrepAvailable() bool {
	_, err := exec.LookPath("semgrep")
	return err == nil
}

func diffScan(repoRoot, baseRef string) (findings.CBOM, error) {
	changed, err := diffscan.ChangedFiles(repoRoot, baseRef)
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("computing changed files: %w", err)
	}

	var codeTargets []string
	var manifestTargets []string
	var relCodeFiles []string

	for _, f := range changed {
		ext := strings.ToLower(filepath.Ext(f))
		base := filepath.Base(f)

		if relevantExtensions[ext] {
			codeTargets = append(codeTargets, filepath.Join(repoRoot, f))
			relCodeFiles = append(relCodeFiles, f)
		} else if relevantFilenames[base] {
			manifestTargets = append(manifestTargets, filepath.Join(repoRoot, f))
		}
	}

	if len(codeTargets) == 0 && len(manifestTargets) == 0 {
		return findings.CBOM{Version: findings.SchemaVersion, Generated: time.Now().UTC(), Findings: []findings.Finding{}}, nil
	}

	cbom := findings.CBOM{
		Version:   findings.SchemaVersion,
		Generated: time.Now().UTC(),
		Findings:  []findings.Finding{},
	}

	if len(codeTargets) > 0 {
		if !semgrepAvailable() {
			fmt.Fprintln(os.Stderr, "vyala: warning: semgrep not found on PATH — skipping code rules scan in diff mode. Install with `pip install semgrep`.")
		} else {
			codeCBOM, err := engine.Scan(repoRoot, codeTargets)
			if err != nil {
				return findings.CBOM{}, err
			}
			cbom.Findings = append(cbom.Findings, codeCBOM.Findings...)
		}
	}

	if len(manifestTargets) > 0 {
		depFindings, err := engine.ScanManifests(repoRoot, manifestTargets)
		if err != nil {
			return findings.CBOM{}, err
		}
		cbom.Findings = append(cbom.Findings, depFindings...)
	}

	addedByFile := map[string]map[int]bool{}
	for _, f := range relCodeFiles {
		lines, err := diffscan.AddedLines(repoRoot, baseRef, f)
		if err != nil {
			return findings.CBOM{}, fmt.Errorf("computing added lines for %s: %w", f, err)
		}
		addedByFile[f] = lines
	}

	var filtered []findings.Finding
	for _, finding := range cbom.Findings {
		if finding.Type == "dependency" {
			filtered = append(filtered, finding)
			continue
		}

		if addedByFile[finding.File][finding.Line] {
			filtered = append(filtered, finding)
		}
	}

	cbom.Findings = filtered
	return cbom, nil
}

func fatal(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "vyala: "+format+"\n", a...)
	os.Exit(1)
}

type remediationPlanOpts struct {
	cbolPath, findingID, repoRoot, planOut, ollamaURL, model, logPath string
}

// runRemediationPlan is the experimental local-only AI remediation entrypoint.
func runRemediationPlan(o remediationPlanOpts) error {
	if o.findingID == "" {
		return fmt.Errorf("-finding is required with -plan-from")
	}
	data, err := os.ReadFile(o.cbolPath)
	if err != nil {
		return fmt.Errorf("reading CBOM: %w", err)
	}
	var cbom findings.CBOM
	if err := json.Unmarshal(data, &cbom); err != nil {
		return fmt.Errorf("parsing CBOM: %w", err)
	}

	var target *findings.Finding
	for i := range cbom.Findings {
		if cbom.Findings[i].ID == o.findingID {
			target = &cbom.Findings[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("finding %q not found in %s", o.findingID, o.cbolPath)
	}

	fmt.Printf("VYALA AI Remediation (experimental, LOCAL-ONLY)\n")
	fmt.Printf("  model:    %s @ %s\n", o.model, o.ollamaURL)
	fmt.Printf("  finding:  %s (%s)\n", target.ID, target.Category)
	fmt.Printf("  location: %s:%d\n\n", target.File, target.Line)

	client := remediation.NewOllamaClient(o.ollamaURL, o.model)
	pipe := &remediation.Pipeline{Model: client}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	rec, err := pipe.Run(ctx, o.repoRoot, *target)
	logErr := remediation.NewTelemetryLog(o.logPath).Append(rec) // capture even failures for training data
	if logErr != nil {
		fmt.Fprintf(os.Stderr, "vyala: warning: telemetry append failed: %v\n", logErr)
	}
	if err != nil {
		return err
	}

	renderPlan(os.Stdout, rec)

	if o.planOut != "" {
		b, jerr := json.MarshalIndent(rec, "", "  ")
		if jerr != nil {
			return jerr
		}
		if werr := os.WriteFile(o.planOut, b, 0644); werr != nil {
			return fmt.Errorf("writing plan: %w", werr)
		}
		fmt.Printf("\nFull record written to %s\nTelemetry appended to %s\n", o.planOut, o.logPath)
	}
	return nil
}

func renderPlan(w io.Writer, rec *remediation.RemediationRecord) {
	p, t := rec.Plan, rec.Triage
	if p == nil || t == nil {
		return
	}
	fmt.Fprintf(w, "Triage: %s (priority %s)\n", t.Complexity, t.Priority)
	for _, n := range t.RiskNotes {
		fmt.Fprintf(w, "  ! %s\n", n)
	}
	fmt.Fprintf(w, "\n%s\n\nSteps:\n", p.Summary)
	for i, s := range p.Steps {
		fmt.Fprintf(w, "  %d. %s\n", i+1, s.Title)
		fmt.Fprintf(w, "     %s\n", s.Detail)
		if s.File != "" {
			fmt.Fprintf(w, "     file: %s\n", s.File)
		}
		if s.CodeBefore != "" || s.CodeAfter != "" {
			fmt.Fprintf(w, "     - %s\n     + %s\n", s.CodeBefore, s.CodeAfter)
		}
	}
	if len(p.Risks) > 0 {
		fmt.Fprintln(w, "\nRisks:")
		for _, r := range p.Risks {
			fmt.Fprintf(w, "  - %s\n", r)
		}
	}
	if len(p.References) > 0 {
		fmt.Fprintln(w, "\nReferences:")
		for _, r := range p.References {
			fmt.Fprintf(w, "  - %s\n", r)
		}
	}
}
