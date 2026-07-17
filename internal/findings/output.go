package findings

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
)

func WriteJSON(c CBOM, path string) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func WriteTerminalReport(w io.Writer, c CBOM) {
	fmt.Fprintln(w, "========== PQC Vulnerability Scan Report ==========")
	fmt.Fprintln(w)

	if len(c.Findings) == 0 {
		fmt.Fprintln(w, "No quantum-vulnerable cryptography detected. Nice.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "=====================================================")
		return
	}

	grouped := map[string][]Finding{}
	for _, f := range c.Findings {
		grouped[f.Severity] = append(grouped[f.Severity], f)
	}

	sevs := make([]string, 0, len(grouped))
	for s := range grouped {
		sevs = append(sevs, s)
	}
	sort.Slice(sevs, func(i, j int) bool { return SeverityRank[sevs[i]] < SeverityRank[sevs[j]] })

	for _, sev := range sevs {
		fs := grouped[sev]
		fmt.Fprintf(w, "%s Severity (%d finding(s)):\n", upper(sev), len(fs))
		for _, f := range fs {
			fmt.Fprintf(w, "  - %s:%d | %s | %s\n", f.File, f.Line, f.Algorithm, f.Category)
			fmt.Fprintf(w, "    Suggested: %s\n", f.SuggestedReplacement)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "=====================================================")
}

func upper(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	return string(b)
}