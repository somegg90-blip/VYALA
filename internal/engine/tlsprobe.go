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

// knownPQCSafeGroups lists TLS key-exchange groups that are quantum-safe:
// the final PQ/T hybrids registered by RFC 10024, the pre-standard Kyber
// draft codepoints still seen in the wild, and pure ML-KEM groups.
var knownPQCSafeGroups = map[tls.CurveID]string{
	tls.X25519MLKEM768:  "X25519MLKEM768",       // RFC 10024 (0x11EC)
	tls.CurveID(0x11EB): "SecP256r1MLKEM768",    // RFC 10024
	tls.CurveID(0x11ED): "SecP384r1MLKEM1024",   // RFC 10024
	tls.CurveID(0xFE30): "X25519Kyber512Draft00", // pre-standard draft
	tls.CurveID(0xFE31): "X25519Kyber768Draft00", // pre-standard draft
	tls.CurveID(0x6399): "X25519Kyber768Draft00", // IANA-assigned draft (obsoleted)
	tls.CurveID(0x639A): "X25519Kyber512Draft00", // IANA-assigned draft (obsoleted)
	tls.CurveID(0x0200): "MLKEM512",             // pure ML-KEM
	tls.CurveID(0x0201): "MLKEM768",
	tls.CurveID(0x0202): "MLKEM1024",
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
	// Extract hostname for SNI (Server Name Indication). SplitHostPort handles
	// both host:port and [IPv6-literal]:port correctly.
	hostname := endpoint
	if h, _, err := net.SplitHostPort(endpoint); err == nil {
		hostname = h
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
		if _, isPQCSafe := knownPQCSafeGroups[state.CurveID]; !isPQCSafe {
			results = append(results, findings.Finding{
				ID:                   findings.GenerateFindingID("tls-kex", endpoint, 1),
				Type:                 "tls",
				File:                 endpoint,
				Algorithm:            fmt.Sprintf("Classical key exchange group (TLS 1.3, group ID 0x%04X)", uint16(state.CurveID)),
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