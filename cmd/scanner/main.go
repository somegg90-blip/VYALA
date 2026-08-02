package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vyala/internal/diffscan"
	"vyala/internal/engine"
	"vyala/internal/findings"
	"vyala/internal/ghcomment"
	"vyala/internal/uploader"
)

var relevantExtensions = map[string]bool{
	".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true,
}

// RelevantFilenames ensures we only scan specific manifests, not every random .json file
var relevantFilenames = map[string]bool{
	"package.json":     true,
	"requirements.txt": true,
	"pyproject.toml":   true,
}

func main() {
	// ---------- flags ----------
	path := flag.String("path", ".", "repo path to scan")
	jsonOut := flag.String("json", "", "write CBOM JSON to this path (optional)")
	diffBase := flag.String("diff-base", "", "if set, scan only files changed vs this git ref/SHA")
	severityThreshold := flag.String("severity-threshold", "medium", "minimum severity to show in detail in PR comment")
	failOn := flag.String("fail-on", "", "exit non-zero if any finding at or above this severity")
	postPRComment := flag.Bool("post-pr-comment", false, "post/update a PR comment with results")
	commentFromFile := flag.String("comment-from-file", "", "post PR comment using the CBOM JSON at this path (no scan)")

	// New flags for comment-only mode – override GitHub event parsing
	prNumber := flag.Int("pr-number", 0, "PR number (for comment posting, overrides GITHUB_EVENT_PATH)")
	headSHA := flag.String("head-sha", "", "Head commit SHA (for comment posting, overrides GITHUB_EVENT_PATH)")

	// New flag for Phase 3: Live TLS endpoint probing
	probeEndpoints := flag.String("probe-endpoints", "", "comma-separated list of host:port endpoints to probe for TLS PQC readiness (e.g., 'api.example.com:443')")

	// New flags for Phase 4: Backend Upload
	uploadURL := flag.String("upload-url", "", "API endpoint to upload CBOM (e.g., http://localhost:8080/v1/scans)")
	apiKey := flag.String("api-key", "", "API key for backend upload authentication")

	flag.Parse()

	if *apiKey == "" {
		*apiKey = os.Getenv("CLI_API_KEY")
	}

	// ---------- Comment‑from‑file mode ----------
	if *commentFromFile != "" {
		if err := postCommentFromFile(*commentFromFile, *severityThreshold, *prNumber, *headSHA); err != nil {
			fatal("posting comment from file: %v", err)
		}
		return
	}

	// ---------- Normal scan mode ----------
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

	// ---------- Live TLS Endpoint Probing ----------
	if *probeEndpoints != "" {
		endpoints := strings.Split(*probeEndpoints, ",")
		// Trim whitespace from endpoints
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

	// ---------- Backend Upload (Phase 4) ----------
	if *uploadURL != "" {
		if err := uploader.UploadCBOM(cbom, *uploadURL, *apiKey); err != nil {
			// We don't fatal() here because we don't want to fail the CI build just because the dashboard upload failed
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

// ---------- helper functions ----------

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

// postComment obtains the PR event and config, then posts/updates the comment.
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

// fullScan scans the entire repository for both code and dependency vulnerabilities
func fullScan(repoRoot string) (findings.CBOM, error) {
	// 1. Run standard code/IaC scan
	cbom, err := engine.Scan(repoRoot, nil)
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("code scan failed: %w", err)
	}

	// 2. Run dependency scan
	depFindings, err := engine.ScanManifests(repoRoot, nil)
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("dependency scan failed: %w", err)
	}

	cbom.Findings = append(cbom.Findings, depFindings...)
	return cbom, nil
}

// diffScan returns the CBOM for only the changed files between baseRef and HEAD.
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

	// Start with an empty CBOM, we'll append to it
	cbom := findings.CBOM{
		Version:   findings.SchemaVersion,
		Generated: time.Now().UTC(),
		Findings:  []findings.Finding{},
	}

	// 1. Run Code Scan if targets exist
	if len(codeTargets) > 0 {
		codeCBOM, err := engine.Scan(repoRoot, codeTargets)
		if err != nil {
			return findings.CBOM{}, err
		}
		cbom.Findings = append(cbom.Findings, codeCBOM.Findings...)
	}

	// 2. Run Dependency Scan if targets exist
	if len(manifestTargets) > 0 {
		depFindings, err := engine.ScanManifests(repoRoot, manifestTargets)
		if err != nil {
			return findings.CBOM{}, err
		}
		cbom.Findings = append(cbom.Findings, depFindings...)
	}

	// Filter code findings by added lines only.
	// Dependencies are flagged if the manifest file itself was changed in the diff.
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
		// If it's a dependency finding, include it directly (manifest changed)
		if finding.Type == "dependency" {
			filtered = append(filtered, finding)
			continue
		}

		// Otherwise, check if it's in the added lines of a changed code file
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
