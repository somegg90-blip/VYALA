package remediation

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"vyala/internal/findings"
)

const snippetRadius = 30

// ContextBundle is everything the agents see besides the finding itself.
// It is captured verbatim in telemetry so future fine-tuning data includes
// exactly the inputs the model conditioned on.
type ContextBundle struct {
	File      string   `json:"file"`
	Language  string   `json:"language"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Snippet   string   `json:"snippet"`
	Imports   []string `json:"imports,omitempty"`
}

// GatherContext reads the code around a finding. This agent is deterministic
// Go — no LLM — so context is cheap, fast, and reproducible for training data.
func GatherContext(repoRoot string, f findings.Finding) (*ContextBundle, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(f.File))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	start := f.Line - snippetRadius
	if start < 1 {
		start = 1
	}
	end := f.Line + snippetRadius
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for i := start; i <= end; i++ {
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(": ")
		sb.WriteString(lines[i-1])
		sb.WriteByte('\n')
	}

	return &ContextBundle{
		File:      f.File,
		Language:  detectLanguage(f.File),
		StartLine: start,
		EndLine:   end,
		Snippet:   sb.String(),
		Imports:   extractImports(lines),
	}, nil
}

var langByExt = map[string]string{
	".py": "python", ".go": "go", ".js": "javascript", ".jsx": "javascript",
	".ts": "typescript", ".tsx": "typescript", ".java": "java",
	".c": "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".hpp": "cpp",
	".cs": "csharp", ".rs": "rust", ".tf": "terraform", ".yaml": "yaml", ".yml": "yaml",
}

func detectLanguage(file string) string {
	return langByExt[strings.ToLower(filepath.Ext(file))]
}

func extractImports(lines []string) []string {
	var out []string
	prefixes := []string{"import ", "from ", "#include", "using ", "extern crate ", "use "}
	for _, l := range lines {
		t := strings.TrimSpace(l)
		for _, p := range prefixes {
			if strings.HasPrefix(t, p) && len(out) < 40 {
				out = append(out, t)
				break
			}
		}
	}
	return out
}
