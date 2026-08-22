// Package cyclonedx converts VYALA CBOM findings into CycloneDX 1.6
// Cryptographic Bills of Materials, interoperable with any spec-compliant
// GRC/SBOM tooling (Trivy, Dependency-Track, IBM CBOMkit, etc.).
package cyclonedx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"vyala/internal/findings"
)

const (
	SpecVersion = "1.6"
	ToolName    = "vyala"
)

type Property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Occurrence follows the CycloneDX 1.6 evidence schema: `location` is a flat
// path string with `line` as a sibling field.
type Occurrence struct {
	BomRef            string `json:"bom-ref,omitempty"`
	Location          string `json:"location"`
	Line              int    `json:"line,omitempty"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

type Evidence struct {
	Occurrences []Occurrence `json:"occurrences,omitempty"`
}

type AlgorithmProperties struct {
	Primitive       string   `json:"primitive,omitempty"`
	CryptoFunctions []string `json:"cryptoFunctions,omitempty"`
}

type ProtocolProperties struct {
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

type RelatedCryptoMaterialProperties struct {
	Type string `json:"type,omitempty"`
}

type CryptoProperties struct {
	AssetType                          string                            `json:"assetType,omitempty"`
	AlgorithmProperties                *AlgorithmProperties              `json:"algorithmProperties,omitempty"`
	ProtocolProperties                 *ProtocolProperties               `json:"protocolProperties,omitempty"`
	RelatedCryptoMaterialProperties    *RelatedCryptoMaterialProperties  `json:"relatedCryptoMaterialProperties,omitempty"`
}

type ToolComponent struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type toolsBlock struct {
	Components []ToolComponent `json:"components"`
}

type Metadata struct {
	Timestamp string      `json:"timestamp"`
	Tools     toolsBlock  `json:"tools"`
	Component *Component  `json:"component,omitempty"`
}

type Component struct {
	Type             string            `json:"type"`
	BomRef           string            `json:"bom-ref,omitempty"`
	Group            string            `json:"group,omitempty"`
	Name             string            `json:"name"`
	Description      string            `json:"description,omitempty"`
	Scope            string            `json:"scope,omitempty"`
	CryptoProperties *CryptoProperties `json:"cryptoProperties,omitempty"`
	Evidence         *Evidence         `json:"evidence,omitempty"`
	Properties       []Property        `json:"properties,omitempty"`
}

type BOM struct {
	BOMFormat    string      `json:"bomFormat"`
	SpecVersion  string      `json:"specVersion"`
	SerialNumber string      `json:"serialNumber,omitempty"`
	Version      int         `json:"version"`
	Metadata     Metadata    `json:"metadata"`
	Components   []Component `json:"components,omitempty"`
}

// Convert maps every VYALA finding to exactly one CycloneDX component so that
// finding IDs remain traceable across both output formats.
//
// Mapping:
//   - code/IaC findings        -> cryptographic-asset (algorithm | protocol)
//   - TLS certificate findings -> cryptographic-asset (certificate)
//   - dependency findings      -> library component (the vulnerable package itself)
func Convert(cbom findings.CBOM, projectName string) *BOM {
	bom := &BOM{
		BOMFormat:    "CycloneDX",
		SpecVersion:  SpecVersion,
		SerialNumber: uuid.New().URN(),
		Version:      1,
		Metadata: Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tools: toolsBlock{
				Components: []ToolComponent{{Type: "application", Name: ToolName}},
			},
			Component: &Component{
				Type:        "application",
				Name:        projectName,
				Description: "Scanned by VYALA for quantum-vulnerable cryptography.",
				Properties: []Property{
					{Name: "vyala:cbom-schema-version", Value: cbom.Version},
				},
			},
		},
		Components: make([]Component, 0, len(cbom.Findings)),
	}

	for _, f := range cbom.Findings {
		if f.Type == "dependency" {
			bom.Components = append(bom.Components, dependencyComponent(f))
			continue
		}
		bom.Components = append(bom.Components, cryptoAssetComponent(f))
	}

	return bom
}

func cryptoAssetComponent(f findings.Finding) Component {
	c := Component{
		Type:   "cryptographic-asset",
		BomRef: f.ID,
		Name:   displayName(f),
		Evidence: &Evidence{
			Occurrences: []Occurrence{{
				BomRef:            f.ID,
				Location:          f.File,
				Line:              f.Line,
				AdditionalContext: oneLine(f.SuggestedReplacement),
			}},
		},
		Properties: vyalaProperties(f),
	}

	switch {
	case strings.Contains(f.Category, "tls_certificate"):
		c.CryptoProperties = &CryptoProperties{AssetType: "certificate"}
	case strings.Contains(f.Category, "tls_kex") || f.Type == "tls":
		c.CryptoProperties = &CryptoProperties{
			AssetType:          "protocol",
			ProtocolProperties: &ProtocolProperties{Type: "tls"},
		}
	case f.Type == "iac":
		c.CryptoProperties = &CryptoProperties{
			AssetType:          "protocol",
			ProtocolProperties: &ProtocolProperties{Type: "tls"},
		}
	default:
		primitive, fns := classifyAlgorithm(f.Category)
		c.CryptoProperties = &CryptoProperties{
			AssetType: "algorithm",
			AlgorithmProperties: &AlgorithmProperties{
				Primitive:       primitive,
				CryptoFunctions: fns,
			},
		}
	}

	return c
}

// classifyAlgorithm maps VYALA rule categories onto the CycloneDX 1.6
// algorithmProperties.primitive enum and cryptoFunctions enum.
func classifyAlgorithm(category string) (string, []string) {
	switch category {
	case "rsa_keygen":
		return "pke", []string{"keygen"}
	case "rsa_signing", "ecdsa_signing", "dsa_signing", "jwt_vulnerable_algorithm":
		return "signature", []string{"sign"}
	case "rsa_encryption":
		return "pke", []string{"encrypt"}
	case "ecc_usage", "dh_keyexchange":
		return "key-agree", []string{"keyderive"}
	case "vulnerable_dependency":
		return "other", nil
	case "unresolved_algorithm_call", "key_loading":
		return "unknown", []string{"unknown"}
	default:
		return "unknown", nil
	}
}

// dependencyComponent emits the vulnerable third-party package as a regular
// library component, which is what CycloneDX consumers expect for packages.
func dependencyComponent(f findings.Finding) Component {
	group, name := parseDepIdentity(f.RuleID)
	return Component{
		Type:        "library",
		BomRef:      f.ID,
		Group:       group,
		Name:        name,
		Scope:       "required",
		Description: "Dependency known to provide quantum-vulnerable cryptography.",
		Properties:  vyalaProperties(f),
	}
}

// parseDepIdentity splits RuleIDs like "dep-gomod-github.com/youmark/pkcs8"
// into a Go-style group ("youmark") and package name ("pkcs8").
func parseDepIdentity(ruleID string) (group, name string) {
	rest := strings.TrimPrefix(ruleID, "dep-")
	parts := strings.SplitN(rest, "-", 2)
	if len(parts) != 2 {
		return "", rest
	}
	eco, pkg := parts[0], parts[1]
	switch eco {
	case "gomod":
		segs := strings.Split(pkg, "/")
		if len(segs) >= 2 {
			return segs[1], segs[len(segs)-1]
		}
		return "", pkg
	default:
		return "", pkg
	}
}

func vyalaProperties(f findings.Finding) []Property {
	props := []Property{
		{Name: "vyala:finding-type", Value: f.Type},
		{Name: "vyala:category", Value: f.Category},
		{Name: "vyala:severity", Value: f.Severity},
		{Name: "vyala:rule-id", Value: f.RuleID},
	}
	if f.ExposureEstimate != "" {
		props = append(props, Property{Name: "vyala:hndl-exposure", Value: oneLine(f.ExposureEstimate)})
	}
	if f.SuggestedReplacement != "" {
		props = append(props, Property{Name: "vyala:suggested-replacement", Value: oneLine(f.SuggestedReplacement)})
	}
	return props
}

func displayName(f findings.Finding) string {
	name := f.Algorithm
	if idx := strings.Index(name, " ("); idx > 0 {
		name = name[:idx]
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = f.Category
	}
	return name
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
	if len(s) > 200 {
		s = s[:197] + "..."
	}
	return s
}

func Write(bom *BOM, path string) error {
	b, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
			return fmt.Errorf("creating output directory: %w", mkErr)
		}
	}
	return os.WriteFile(path, b, 0644)
}
