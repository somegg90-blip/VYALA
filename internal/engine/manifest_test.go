package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vyala/internal/findings"
)

func TestScanManifests(t *testing.T) {
	// Resolve the absolute path to our test fixtures
	repoRoot, err := filepath.Abs("../../testdata/vulnerable-deps")
	if err != nil {
		t.Fatalf("Failed to get abs path: %v", err)
	}

	expectedPath, err := filepath.Abs("../../testdata/vulnerable-deps/expected.json")
	if err != nil {
		t.Fatalf("Failed to get expected json path: %v", err)
	}

	// 1. Load expected findings from the JSON fixture
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read expected json: %v", err)
	}

	var expectedCBOM findings.CBOM
	if err := json.Unmarshal(data, &expectedCBOM); err != nil {
		t.Fatalf("Failed to unmarshal expected json: %v", err)
	}

	// 2. Run the manifest scanner on the test fixtures
	actualFindings, err := ScanManifests(repoRoot, nil)
	if err != nil {
		t.Fatalf("ScanManifests failed: %v", err)
	}

	// 3. Compare findings
	if len(actualFindings) != len(expectedCBOM.Findings) {
		t.Fatalf("Finding count mismatch: expected %d, got %d", len(expectedCBOM.Findings), len(actualFindings))
	}

	// We check that every expected finding exists in the actual findings
	// (Order doesn't matter because map iteration in Go is random)
	for _, expected := range expectedCBOM.Findings {
		found := false
		for _, actual := range actualFindings {
			if actual.ID == expected.ID {
				if actual.Algorithm != expected.Algorithm {
					t.Errorf("Algorithm mismatch for %s: expected %s, got %s", expected.ID, expected.Algorithm, actual.Algorithm)
				}
				if actual.SuggestedReplacement != expected.SuggestedReplacement {
					t.Errorf("Replacement mismatch for %s: expected %s, got %s", expected.ID, expected.SuggestedReplacement, actual.SuggestedReplacement)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected finding ID %s not found in actual scan results", expected.ID)
		}
	}
}
