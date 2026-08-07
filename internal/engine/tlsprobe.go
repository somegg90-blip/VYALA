package engine

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"time"

	"vyala/internal/findings"
)

var knownHybridPQCGroups = map[uint16]string{
	0x11EC: "X25519MLKEM768",
	0xFE30: "X25519Kyber512Draft00",
	0xFE31: "X25519Kyber768Draft00",
}

func ScanTLSProbes(endpoints []string) ([]findings.Finding, error) {
	var results []findings.Finding
	for _, raw := range endpoints {
		endpoint := strings.TrimSpace(raw)
		if endpoint == "" {
			continue
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
	// Extract hostname for SNI (Server Name Indication)
	hostname := endpoint
	if idx := strings.LastIndex(endpoint, ":"); idx != -1 {
		hostname = endpoint[:idx]
	}

	// FIX: Use a strict 5-second timeout to prevent hanging on blackholed IPs
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", endpoint, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         hostname, // Crucial for SNI
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	state := conn.ConnectionState()
	var results []findings.Finding
	cipherName := tls.CipherSuiteName(state.CipherSuite)

	switch {
	case state.Version < tls.VersionTLS13:
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
	}

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