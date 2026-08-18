package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/mod/modfile"

	"vyala/internal/db"
	"vyala/internal/findings"
)

type npmPackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type pyProjectTOML struct {
	Tool struct {
		Poetry struct {
			Dependencies map[string]string `toml:"dependencies"`
		} `toml:"poetry"`
		Project struct {
			Dependencies []string `toml:"dependencies"`
		} `toml:"project"`
	} `toml:"tool"`
}

func ScanManifests(repoRoot string, targets []string) ([]findings.Finding, error) {
	pqcDB, err := db.ReadDB()
	if err != nil {
		return nil, fmt.Errorf("loading PQC db: %w", err)
	}

	var allFindings []findings.Finding

	filesToScan := targets
	if len(filesToScan) == 0 {
		filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			// Skip ignored directories entirely
			if info.IsDir() {
				dirName := filepath.Base(path)
				if dirName == "testdata" || dirName == "internal" || dirName == "node_modules" || dirName == "vendor" || dirName == ".git" {
					return filepath.SkipDir
				}
			}

			base := filepath.Base(path)
			if base == "package.json" || strings.HasSuffix(base, "requirements.txt") || base == "pyproject.toml" || base == "go.mod" {
				filesToScan = append(filesToScan, path)
			}
			return nil
		})
	}

	for _, file := range filesToScan {
		base := filepath.Base(file)
		relPath := findings.NormalizeRelPath(repoRoot, file)

		if base == "package.json" {
			found := scanNPM(file, relPath, pqcDB["npm"])
			allFindings = append(allFindings, found...)
		} else if strings.HasSuffix(base, "requirements.txt") {
			found := scanPIP(file, relPath, pqcDB["pip"])
			allFindings = append(allFindings, found...)
		} else if base == "pyproject.toml" {
			found := scanPyProject(file, relPath, pqcDB["pip"])
			allFindings = append(allFindings, found...)
		} else if base == "go.mod" {
			found := scanGoMod(file, relPath, pqcDB["gomod"])
			allFindings = append(allFindings, found...)
		}
	}

	return allFindings, nil
}

// scanNPM checks BOTH dependencies and devDependencies — dev-only crypto/testing
// libraries are just as real a finding as production ones, and skipping them
// was a silent under-detection gap.
func scanNPM(absPath, relPath string, dbMap map[string]db.PQCStatus) []findings.Finding {
	var results []findings.Finding
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	var pkg npmPackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	lineNum := 1
	checkDeps := func(deps map[string]string) {
		for depName := range deps {
			if status, ok := dbMap[depName]; ok {
				results = append(results, findings.Finding{
					ID:                   findings.GenerateFindingID("dep-"+depName, relPath, lineNum),
					Type:                 "dependency",
					File:                 relPath,
					Line:                 lineNum,
					Algorithm:            status.Algorithm,
					Severity:             "high",
					Category:             "vulnerable_dependency",
					ExposureEstimate:     "Library-level usage",
					SuggestedReplacement: status.Replacement,
					RuleID:               "dep-npm-" + depName,
				})
			}
		}
	}
	checkDeps(pkg.Dependencies)
	checkDeps(pkg.DevDependencies)

	return results
}

func scanPIP(absPath, relPath string, dbMap map[string]db.PQCStatus) []findings.Finding {
	var results []findings.Finding
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Strip inline comments
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		// Strip extras like [extra]
		if idx := strings.Index(line, "["); idx != -1 {
			line = line[:idx]
		}

		depName := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '>' || r == '<' || r == '!' || r == ' ' || r == ';'
		})
		if len(depName) == 0 {
			continue // guards against a malformed/blank line after stripping
		}

		if status, ok := dbMap[depName[0]]; ok {
			results = append(results, findings.Finding{
				ID:                   findings.GenerateFindingID("dep-"+depName[0], relPath, i+1),
				Type:                 "dependency",
				File:                 relPath,
				Line:                 i + 1,
				Algorithm:            status.Algorithm,
				Severity:             "high",
				Category:             "vulnerable_dependency",
				ExposureEstimate:     "Library-level usage",
				SuggestedReplacement: status.Replacement,
				RuleID:               "dep-pip-" + depName[0],
			})
		}
	}
	return results
}

func scanPyProject(absPath, relPath string, dbMap map[string]db.PQCStatus) []findings.Finding {
	var results []findings.Finding
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	var proj pyProjectTOML
	if _, err := toml.Decode(string(data), &proj); err != nil {
		return nil
	}

	lineNum := 1 // TOML parsing doesn't give us easy line numbers, so we approximate

	// Check Poetry dependencies
	for depName := range proj.Tool.Poetry.Dependencies {
		depName = strings.ToLower(depName)
		if status, ok := dbMap[depName]; ok {
			results = append(results, findings.Finding{
				ID:                   findings.GenerateFindingID("dep-"+depName, relPath, lineNum),
				Type:                 "dependency",
				File:                 relPath,
				Line:                 lineNum,
				Algorithm:            status.Algorithm,
				Severity:             "high",
				Category:             "vulnerable_dependency",
				ExposureEstimate:     "Library-level usage",
				SuggestedReplacement: status.Replacement,
				RuleID:               "dep-pip-" + depName,
			})
		}
	}

	// Check PEP 621 dependencies (array of strings, e.g. "requests[security]>=2.0")
	for _, depStr := range proj.Tool.Project.Dependencies {
		// Strip bracket extras first — previously missing, caused "requests[security]"
		// to fail matching against the DB entry for "requests".
		if idx := strings.Index(depStr, "["); idx != -1 {
			depStr = depStr[:idx]
		}

		parts := strings.FieldsFunc(depStr, func(r rune) bool {
			return r == '=' || r == '>' || r == '<' || r == '!' || r == ' ' || r == ';'
		})
		if len(parts) == 0 {
			continue
		}
		depName := strings.ToLower(parts[0])

		if status, ok := dbMap[depName]; ok {
			results = append(results, findings.Finding{
				ID:                   findings.GenerateFindingID("dep-"+depName, relPath, lineNum),
				Type:                 "dependency",
				File:                 relPath,
				Line:                 lineNum,
				Algorithm:            status.Algorithm,
				Severity:             "high",
				Category:             "vulnerable_dependency",
				ExposureEstimate:     "Library-level usage",
				SuggestedReplacement: status.Replacement,
				RuleID:               "dep-pip-" + depName,
			})
		}
	}

	return results
}

func scanGoMod(absPath, relPath string, dbMap map[string]db.PQCStatus) []findings.Finding {
	var results []findings.Finding
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	// Parse the go.mod file using Go's official modfile library
	modFile, err := modfile.Parse(absPath, data, nil)
	if err != nil {
		return nil
	}

	lineNum := 1 // modfile doesn't give exact line numbers easily, so we approximate
	for _, req := range modFile.Require {
		if req.Indirect {
			continue // Skip indirect dependencies for now to reduce noise
		}

		depName := req.Mod.Path
		if status, ok := dbMap[depName]; ok {
			results = append(results, findings.Finding{
				ID:                   findings.GenerateFindingID("dep-"+depName, relPath, lineNum),
				Type:                 "dependency",
				File:                 relPath,
				Line:                 lineNum,
				Algorithm:            status.Algorithm,
				Severity:             "high",
				Category:             "vulnerable_dependency",
				ExposureEstimate:     "Library-level usage",
				SuggestedReplacement: status.Replacement,
				RuleID:               "dep-gomod-" + depName,
			})
		}
	}

	return results
}
