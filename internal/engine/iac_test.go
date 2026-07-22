package engine

import (
    "path/filepath"
    "testing"
)

func TestScanIaC(t *testing.T) {
    // Resolve the absolute path to our IaC test fixtures
    repoRoot, err := filepath.Abs("../../testdata/vulnerable-iac")
    if err != nil {
        t.Fatalf("Failed to get abs path: %v", err)
    }

    // Run the scan on the test fixtures
    cbom, err := Scan(repoRoot, nil)
    if err != nil {
        t.Fatalf("Scan failed: %v", err)
    }

    // We expect exactly 2 findings (1 Terraform, 1 Kubernetes)
    if len(cbom.Findings) != 2 {
        t.Fatalf("Finding count mismatch: expected 2, got %d", len(cbom.Findings))
    }

    // Verify we caught both specific rules
    foundTerraform := false
    foundK8s := false

    for _, f := range cbom.Findings {
        if f.RuleID == "terraform-aws-insecure-tls" {
            foundTerraform = true
        }
        if f.RuleID == "k8s-insecure-cipher-suites" {
            foundK8s = true
        }
    }

    if !foundTerraform {
        t.Error("Missing Terraform finding (terraform-aws-insecure-tls)")
    }
    if !foundK8s {
        t.Error("Missing Kubernetes finding (k8s-insecure-cipher-suites)")
    }
}