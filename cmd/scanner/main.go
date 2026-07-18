package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vyala/internal/diffscan"
	"vyala/internal/engine"
	"vyala/internal/findings"
	"vyala/internal/ghcomment"
)

var relevantExtensions = map[string]bool{
	".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true,
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

	flag.Parse()

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
		cbom, err = engine.Scan(repoRoot, nil)
	}
	if err != nil {
		fatal("scan failed: %v", err)
	}

	findings.WriteTerminalReport(os.Stdout, cbom)

	if *jsonOut != "" {
		if err := findings.WriteJSON(cbom, *jsonOut); err != nil {
			fatal("writing JSON: %v", err)
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
// If prNumber > 0 and headSHA is non-empty, those values are used directly
// instead of reading $GITHUB_EVENT_PATH. This is essential for the
// fork‑safe comment workflow, which runs on `workflow_run` and doesn't have
// a natural pull_request event payload.
func postComment(cbom findings.CBOM, severityThreshold string, prNumber int, headSHA string) error {
	var ev *ghcomment.PREvent
	var err error

	if prNumber > 0 && headSHA != "" {
		// Construct a minimal PREvent from the flags – no JSON parsing needed.
		ev = &ghcomment.PREvent{
			PullRequest: struct {
				Number int    `json:"number"`
				Base   struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				} `json:"base"`
				Head struct {
					SHA string `json:"sha"`
				} `json:"head"`
			}{
				Number: prNumber,
				// Base info isn't needed for comment posting, so we leave it zero.
				Head: struct {
					SHA string `json:"sha"`
				}{SHA: headSHA},
			},
		}
	} else {
		// Fall back to the standard environment-based event parsing.
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

// diffScan returns the CBOM for only the changed files between baseRef and HEAD.
func diffScan(repoRoot, baseRef string) (findings.CBOM, error) {
	changed, err := diffscan.ChangedFiles(repoRoot, baseRef)
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("computing changed files: %w", err)
	}

	var targets []string
	var relFiles []string
	for _, f := range changed {
		if relevantExtensions[strings.ToLower(filepath.Ext(f))] {
			targets = append(targets, filepath.Join(repoRoot, f))
			relFiles = append(relFiles, f)
		}
	}

	if len(targets) == 0 {
		return findings.CBOM{Version: findings.SchemaVersion, Findings: []findings.Finding{}}, nil
	}

	cbom, err := engine.Scan(repoRoot, targets)
	if err != nil {
		return findings.CBOM{}, err
	}

	addedByFile := map[string]map[int]bool{}
	for _, f := range relFiles {
		lines, err := diffscan.AddedLines(repoRoot, baseRef, f)
		if err != nil {
			return findings.CBOM{}, fmt.Errorf("computing added lines for %s: %w", f, err)
		}
		addedByFile[f] = lines
	}

	filtered := cbom.Findings[:0]
	for _, finding := range cbom.Findings {
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