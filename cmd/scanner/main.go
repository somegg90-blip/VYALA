package main

import (
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
	path := flag.String("path", ".", "repo path to scan")
	jsonOut := flag.String("json", "", "write CBOM JSON to this path (optional)")
	diffBase := flag.String("diff-base", "", "if set, scan only files changed vs this git ref/SHA (diff-aware mode)")
	severityThreshold := flag.String("severity-threshold", "medium", "minimum severity to show in detail in the PR comment (high|medium|low)")
	failOn := flag.String("fail-on", "", "exit non-zero if any finding at or above this severity is present (high|medium|low); empty = never fail")
	postPRComment := flag.Bool("post-pr-comment", false, "post/update a PR comment with the results (must run inside a GitHub Action pull_request context)")
	flag.Parse()

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
		if err := postComment(cbom, *severityThreshold); err != nil {
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

func diffScan(repoRoot, baseRef string) (findings.CBOM, error) {
	changed, err := diffscan.ChangedFiles(repoRoot, baseRef)
	if err != nil {
		return findings.CBOM{}, fmt.Errorf("computing changed files (is this a shallow checkout missing %q? fetch-depth: 0 is required): %w", baseRef, err)
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

func postComment(cbom findings.CBOM, severityThreshold string) error {
	ev, err := ghcomment.LoadEventFromEnv()
	if err != nil {
		return err
	}
	cfg, err := ghcomment.ConfigFromEnv()
	if err != nil {
		return err
	}
	body := ghcomment.RenderComment(cbom, cfg.Repo, ev.PullRequest.Head.SHA, severityThreshold)
	return ghcomment.PostOrUpdate(cfg, ev.PullRequest.Number, body)
}

func fatal(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "vyala: "+format+"\n", a...)
	os.Exit(1)
}