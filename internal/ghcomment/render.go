package ghcomment

import (
	"fmt"
	"sort"
	"strings"

	"vyala/internal/findings"
)

const Marker = "<!-- vyala:pr-comment:v1 -->"

func RenderComment(cbom findings.CBOM, repoSlug, headSHA, severityThreshold string) string {
	var b strings.Builder
	fmt.Fprintln(&b, Marker)
	fmt.Fprintln(&b, "## 🔐 VYALA — Post-Quantum Cryptography Scan")
	fmt.Fprintln(&b)

	if len(cbom.Findings) == 0 {
		fmt.Fprintln(&b, "No quantum-vulnerable cryptography detected in the changed lines of this PR.")
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, footer(cbom))
		return b.String()
	}

	shown := map[string][]findings.Finding{}
	belowThresholdCount := 0
	for _, f := range cbom.Findings {
		if findings.MeetsThreshold(f.Severity, severityThreshold) {
			shown[f.Severity] = append(shown[f.Severity], f)
		} else {
			belowThresholdCount++
		}
	}

	sevs := make([]string, 0, len(shown))
	for s := range shown {
		sevs = append(sevs, s)
	}
	sort.Slice(sevs, func(i, j int) bool { return findings.SeverityRank[sevs[i]] < findings.SeverityRank[sevs[j]] })

	if len(sevs) == 0 {
		fmt.Fprintf(&b, "No findings at or above the **%s** severity threshold in the changed lines of this PR.\n", severityThreshold)
	}

	for _, sev := range sevs {
		fs := shown[sev]
		fmt.Fprintf(&b, "### %s (%d)\n\n", severityBadge(sev), len(fs))
		fmt.Fprintln(&b, "| File | Algorithm | Suggested replacement |")
		fmt.Fprintln(&b, "|---|---|---|")
		for _, f := range fs {
			link := fmt.Sprintf("https://github.com/%s/blob/%s/%s#L%d", repoSlug, headSHA, f.File, f.Line)
			fmt.Fprintf(&b, "| [`%s:%d`](%s) | %s | %s |\n",
				f.File, f.Line, link, f.Algorithm, oneLine(f.SuggestedReplacement))
		}
		fmt.Fprintln(&b)
	}

	if belowThresholdCount > 0 {
		fmt.Fprintf(&b, "_%d additional lower-severity finding(s) not shown (below the %s threshold)._\n\n",
			belowThresholdCount, severityThreshold)
	}

	fmt.Fprintln(&b, footer(cbom))
	return b.String()
}

func footer(cbom findings.CBOM) string {
	return fmt.Sprintf("<sub>Scanned only the lines changed in this PR — not the whole file or repo. Full CBOM schema v%s. Static analysis only; no AI-generated content in this comment.</sub>", cbom.Version)
}

func severityBadge(sev string) string {
	switch sev {
	case "high":
		return "🔴 High"
	case "medium":
		return "🟡 Medium"
	default:
		return "⚪ Low"
	}
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 140 {
		s = s[:137] + "..."
	}
	return s
}