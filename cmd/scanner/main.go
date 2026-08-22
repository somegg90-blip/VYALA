package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vyala/internal/cyclonedx"
	"vyala/internal/diffscan"
	"vyala/internal/engine"
	"vyala/internal/findings"
	"vyala/internal/ghcomment"
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
	cbom, err := engine.Scan(repoRoot, nil)
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("code scan failed: %w", err)
	}

	depFindings, err := engine.ScanManifests(repoRoot, nil)
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("dependency scan failed: %w", err)
	}

	cbom.Findings = append(cbom.Findings, depFindings...)
	return cbom, nil
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
		codeCBOM, err := engine.Scan(repoRoot, codeTargets)
		if err != nil {
			return findings.CBOM{}, err
		}
		cbom.Findings = append(cbom.Findings, codeCBOM.Findings...)
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
