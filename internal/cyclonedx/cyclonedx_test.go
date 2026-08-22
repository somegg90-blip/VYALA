package cyclonedx

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"vyala/internal/findings"
)

func sampleCBOM() findings.CBOM {
	return findings.CBOM{
		Version: "1.0",
		Findings: []findings.Finding{
			{
				ID: "vy-code000000000001", Type: "code", File: "auth.py", Line: 42,
				Algorithm: "RSA", Severity: "medium", Category: "rsa_keygen",
				SuggestedReplacement: "Replace with ML-KEM (FIPS 203).",
				RuleID:               "pqc-rsa-keygen-python",
			},
			{
				ID: "vy-tls000000000001", Type: "tls", File: "example.com:443", Line: 0,
				Algorithm:            "Classical key exchange, pre-TLS1.3 (TLS_RSA_WITH_AES_128_CBC_SHA)",
				Severity:             "high",
				Category:             "vulnerable_tls_kex",
				SuggestedReplacement: "Upgrade to TLS 1.3 with hybrid ML-KEM.",
				RuleID:               "tls-kex-probe",
			},
			{
				ID: "vy-cert00000000001", Type: "tls", File: "example.com:443", Line: 0,
				Algorithm: "ECDSA (TLS certificate signature)", Severity: "low",
				Category: "vulnerable_tls_certificate",
				RuleID:   "tls-cert-probe",
			},
			{
				ID: "vy-depabc00000001", Type: "dependency", File: "package.json", Line: 1,
				Algorithm: "RSA/ECDSA", Severity: "high", Category: "vulnerable_dependency",
				SuggestedReplacement: "Migrate to native crypto.",
				RuleID:               "dep-npm-node-forge",
			},
			{
				ID: "vy-depgomod000001", Type: "dependency", File: "go.mod", Line: 5,
				Algorithm: "RSA/ECDSA", Severity: "high", Category: "vulnerable_dependency",
				RuleID: "dep-gomod-github.com/youmark/pkcs8",
			},
		},
	}
}

func TestConvertStructure(t *testing.T) {
	bom := Convert(sampleCBOM(), "test-repo")

	if bom.BOMFormat != "CycloneDX" {
		t.Errorf("bomFormat = %q, want CycloneDX", bom.BOMFormat)
	}
	if bom.SpecVersion != SpecVersion {
		t.Errorf("specVersion = %q, want %q", bom.SpecVersion, SpecVersion)
	}
	urnRe := regexp.MustCompile(`^urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !urnRe.MatchString(bom.SerialNumber) {
		t.Errorf("serialNumber %q does not match RFC-4122 URN pattern", bom.SerialNumber)
	}
	if bom.Version < 1 {
		t.Errorf("version = %d, must be >= 1", bom.Version)
	}
	if bom.Metadata.Component == nil || bom.Metadata.Component.Name != "test-repo" {
		t.Fatalf("metadata.component missing or wrong name")
	}

	// 5 findings -> exactly 5 components (1:1 traceability)
	if len(bom.Components) != len(sampleCBOM().Findings) {
		t.Fatalf("component count = %d, want %d", len(bom.Components), len(sampleCBOM().Findings))
	}

	byRef := map[string]Component{}
	for _, c := range bom.Components {
		if c.BomRef == "" {
			t.Errorf("component %q missing bom-ref", c.Name)
		}
		byRef[c.BomRef] = c
	}

	// Code finding -> algorithm asset
	code := byRef["vy-code000000000001"]
	if code.Type != "cryptographic-asset" {
		t.Errorf("code finding type = %q", code.Type)
	}
	cp := code.CryptoProperties
	if cp == nil || cp.AssetType != "algorithm" {
		t.Fatalf("code finding assetType = %+v, want algorithm", cp)
	}
	if cp.AlgorithmProperties == nil || cp.AlgorithmProperties.Primitive != "pke" {
		t.Errorf("rsa_keygen primitive = %+v, want pke", cp.AlgorithmProperties)
	}
	if len(cp.AlgorithmProperties.CryptoFunctions) == 0 || cp.AlgorithmProperties.CryptoFunctions[0] != "keygen" {
		t.Errorf("rsa_keygen cryptoFunctions = %v, want [keygen]", cp.AlgorithmProperties.CryptoFunctions)
	}
	if len(code.Evidence.Occurrences) != 1 ||
		code.Evidence.Occurrences[0].Location != "auth.py" ||
		code.Evidence.Occurrences[0].Line != 42 {
		t.Errorf("occurrence = %+v, want auth.py:42", code.Evidence.Occurrences)
	}

	// TLS kex finding -> protocol asset
	kex := byRef["vy-tls000000000001"]
	if kex.CryptoProperties == nil || kex.CryptoProperties.AssetType != "protocol" {
		t.Errorf("tls kex assetType wrong: %+v", kex.CryptoProperties)
	}
	if kex.CryptoProperties.ProtocolProperties == nil || kex.CryptoProperties.ProtocolProperties.Type != "tls" {
		t.Errorf("tls kex protocolProperties wrong")
	}

	// Certificate finding -> certificate asset
	cert := byRef["vy-cert00000000001"]
	if cert.CryptoProperties == nil || cert.CryptoProperties.AssetType != "certificate" {
		t.Errorf("tls cert assetType wrong: %+v", cert.CryptoProperties)
	}

	// Dependency findings -> library components with identity parsed from RuleID
	dep := byRef["vy-depabc00000001"]
	if dep.Type != "library" || dep.Name != "node-forge" {
		t.Errorf("npm dep component = type:%s name:%s, want library/node-forge", dep.Type, dep.Name)
	}
	gomod := byRef["vy-depgomod000001"]
	if gomod.Group != "youmark" || gomod.Name != "pkcs8" {
		t.Errorf("gomod dep group/name = %q/%q, want youmark/pkcs8", gomod.Group, gomod.Name)
	}

	// vyala:* properties present on every component
	for _, c := range bom.Components {
		names := map[string]bool{}
		for _, p := range c.Properties {
			names[p.Name] = true
		}
		for _, required := range []string{"vyala:category", "vyala:severity", "vyala:rule-id"} {
			if !names[required] {
				t.Errorf("component %q missing property %s", c.BomRef, required)
			}
		}
	}
}

func TestWriteProducesValidJSON(t *testing.T) {
	bom := Convert(sampleCBOM(), "test-repo")
	data, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundTrip BOM
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundTrip.BOMFormat != "CycloneDX" || len(roundTrip.Components) != 5 {
		t.Fatalf("round-trip mismatch: %+v", roundTrip)
	}
	if !strings.Contains(string(data), `"cryptoProperties"`) {
		t.Error("output missing cryptoProperties block")
	}
}
