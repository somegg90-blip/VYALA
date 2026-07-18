package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vyala/internal/findings"
)

func readExpected(t *testing.T, path string) findings.CBOM {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read expected file %s: %v", path, err)
	}
	var cbom findings.CBOM
	if err := json.Unmarshal(data, &cbom); err != nil {
		t.Fatalf("cannot unmarshal expected CBOM from %s: %v", path, err)
	}
	return cbom
}

func cbomEqual(a, b findings.CBOM) bool {
	if len(a.Findings) != len(b.Findings) {
		return false
	}
	for i := range a.Findings {
		if a.Findings[i].ID != b.Findings[i].ID {
			return false
		}
		if a.Findings[i].File != b.Findings[i].File {
			return false
		}
		if a.Findings[i].Line != b.Findings[i].Line {
			return false
		}
		if a.Findings[i].Category != b.Findings[i].Category {
			return false
		}
		if a.Findings[i].Severity != b.Findings[i].Severity {
			return false
		}
	}
	return true
}

func TestKnownFindings(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		expectedFile string
	}{
		{
			name:         "rsa-keygen",
			target:       filepath.Join("..", "..", "testdata", "rsa-keygen"),
			expectedFile: filepath.Join("..", "..", "testdata", "rsa-keygen", "expected.json"),
		},
		{
			name:         "ecdsa-signing",
			target:       filepath.Join("..", "..", "testdata", "ecdsa-signing"),
			expectedFile: filepath.Join("..", "..", "testdata", "ecdsa-signing", "expected.json"),
		},
		{
			name:         "clean",
			target:       filepath.Join("..", "..", "testdata", "clean"),
			expectedFile: filepath.Join("..", "..", "testdata", "clean", "expected.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absTarget, err := filepath.Abs(tt.target)
			if err != nil {
				t.Fatalf("cannot resolve target path: %v", err)
			}
			cbom, err := Scan(absTarget, nil)
			if err != nil {
				t.Fatalf("Scan() error: %v", err)
			}
			expected := readExpected(t, tt.expectedFile)

			if !cbomEqual(cbom, expected) {
				t.Errorf("findings mismatch.\nGot:\n%+v\nExpected:\n%+v", cbom.Findings, expected.Findings)
			}
		})
	}
}
