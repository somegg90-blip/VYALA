package engine

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"

	"vyala/internal/findings"
)

// knownHybridPQCGroups maps TLS SupportedGroup/CurveID values (per the IANA TLS
// registry) to their names, covering both the finalized standard and the earlier
// drafts still seen in production deployments (e.g. Cloudflare, Chrome).
//
// Raw numeric IDs are used rather than only Go's named stdlib constants, since
// not every Go version exports every group as a constant, and IDs are stable
// across versions regardless.
var knownHybridPQCGroups = map[uint16]string{
	0x11EC: "X25519MLKEM768",        // finalized standard hybrid group
	0xFE30: "X25519Kyber512Draft00", // early Cloudflare/Chrome draft
	0xFE31: "X25519Kyber768Draft00", // early Cloudflare/Chrome draft
}

func ScanTLSProbes(endpoints []string) ([]findings.Finding, error) {
	var results []findings.Finding
	for _, raw := range endpoints {
		endpoint := strings.TrimSpace(raw)
		if endpoint == "" {
			continue // guards against trailing/duplicate commas in the flag value
		}
		if !strings.Contains(endpoint, ":") {
			endpoint = endpoint + ":443"
		}
		epFindings, err := probeEndpoint(endpoint)
		if err != nil {
			fmt.Printf("Warning: could not probe %s: %v\n", endpoint, err)
			continue
		}
		results = append(results, epFindings...)
	}
	return results, nil
}

func probeEndpoint(endpoint string) ([]findings.Finding, error) {
	conn, err := tls.Dial("tcp", endpoint, &tls.Config{
		InsecureSkipVerify: true, // we're inspecting the handshake, not validating trust
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	state := conn.ConnectionState()
	var results []findings.Finding
	cipherName := tls.CipherSuiteName(state.CipherSuite)

	// ---------- CHECK 1: Key exchange — the HNDL-relevant signal ----------
	switch {
	case state.Version < tls.VersionTLS13:
		// No version of TLS before 1.3 offers any hybrid/PQC key exchange option —
		// every negotiated suite here is unambiguously classical.
		results = append(results, findings.Finding{
			ID:                   findings.GenerateFindingID("tls-kex", endpoint, 1),
			Type:                 "tls",
			File:                 endpoint,
			Algorithm:            fmt.Sprintf("Classical key exchange, pre-TLS1.3 (%s)", cipherName),
			Severity:             "high",
			Category:             "vulnerable_tls_kex",
			ExposureEstimate:     "HNDL: traffic captured today can be decrypted once quantum computers break classical key exchange",
			SuggestedReplacement: "Upgrade to TLS 1.3 with a hybrid ML-KEM key exchange (FIPS 203), e.g. X25519MLKEM768.",
			RuleID:               "tls-kex-probe",
		})

	default:
		// TLS 1.3: check the actual negotiated group.
		groupID := uint16(state.CurveID)
		if _, isHybrid := knownHybridPQCGroups[groupID]; !isHybrid {
			results = append(results, findings.Finding{
				ID:                   findings.GenerateFindingID("tls-kex", endpoint, 1),
				Type:                 "tls",
				File:                 endpoint,
				Algorithm:            fmt.Sprintf("Classical key exchange group (TLS 1.3, group ID 0x%04X)", groupID),
				Severity:             "high",
				Category:             "vulnerable_tls_kex",
				ExposureEstimate:     "HNDL: traffic captured today can be decrypted once quantum computers break classical key exchange",
				SuggestedReplacement: "Enable a hybrid ML-KEM key exchange (FIPS 203), e.g. X25519MLKEM768, on this endpoint.",
				RuleID:               "tls-kex-probe",
			})
		}
		// If it IS a known hybrid group: correctly no finding. This is the fix —
		// previously this branch was entirely skipped to avoid guessing.
	}

	// ---------- CHECK 2: Certificate signing — real, but lower urgency ----------
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		if cert.PublicKeyAlgorithm == x509.RSA || cert.PublicKeyAlgorithm == x509.ECDSA {
			results = append(results, findings.Finding{
				ID:        findings.GenerateFindingID("tls-cert", endpoint, 2),
				Type:      "tls",
				File:      endpoint,
				Algorithm: cert.PublicKeyAlgorithm.String() + " (TLS certificate signature)",
				Severity:  "low",
				Category:  "vulnerable_tls_certificate",
				ExposureEstimate: "Future identity-forgery risk only — NOT a harvest-now-decrypt-later risk. " +
					"As of 2026, no public CA yet issues PQC certificates, so this reflects an industry-wide " +
					"gap rather than a migration failure specific to this server.",
				SuggestedReplacement: "Plan to migrate to ML-DSA (FIPS 204) certificates once a trusted CA offers them.",
				RuleID:                "tls-cert-probe",
			})
		}
	}

	return results, nil
}