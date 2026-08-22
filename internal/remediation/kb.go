package remediation

// KB returns grounded migration guidance for a finding category. These are
// curated facts — the planner is told to treat them as authoritative and to
// cite them, which keeps a 7B model from hallucinating crypto APIs.
func KB(category string) string {
	if v, ok := kbByCategory[category]; ok {
		return v
	}
	return kbGeneral
}

const kbGeneral = `NIST finalized FIPS 203 (ML-KEM, key encapsulation), FIPS 204 (ML-DSA, signatures)
and FIPS 205 (SLH-DSA, stateless hash-based signatures) in August 2024.
Recommended transition pattern: HYBRID constructions (classical + PQC) so security
holds if either component breaks. Signatures are identity-forgery risks (not HNDL);
key exchange and encryption are harvest-now-decrypt-later risks — prioritize those.`

var kbByCategory = map[string]string{
	"rsa_keygen": `RSA key generation is quantum-vulnerable (Shor's algorithm).
Migration: ML-KEM (FIPS 203) hybrid KEM for encryption/key exchange; ML-DSA
(FIPS 204) or SLH-DSA (FIPS 205) for signatures.
Library pointers:
- Go:      crypto/mlkem (stdlib, Go 1.24+); signatures via github.com/cloudflare/circl (sign/mldsa)
- Python:  liboqs-python; oqs-provider with OpenSSL 3.5+
- Java:    JDK 24+ JEP 496 (ML-KEM) / JEP 497 (ML-DSA); BouncyCastle pqc packages
- Rust:    ml-kem crate, oqs crate
- JS/TS:   @noble/post-quantum
- C/C++:   liboqs; OpenSSL 3.5 ships ML-KEM/ML-DSA natively
- C#:      .NET 10 System.Security.Cryptography PQC APIs; BouncyCastle.Crypto Pqc*`,

	"rsa_signing": `RSA signatures are forgeable by a future CRQC. Not an HNDL risk, but breaks
long-lived identities (code signing certs, JWTs, root of trust chains).
Migration: ML-DSA (FIPS 204) preferred; SLH-DSA (FIPS 205) where conservative
hash-based security is required. During transition use composite/hybrid
signatures (classical + ML-DSA verified together). Watch certificate chain
compatibility — peers must support the new signature algorithm.`,

	"rsa_encryption": `RSA-OAEP / PKCS#1 v1.5 ciphertexts captured today can be decrypted once a
cryptographically relevant quantum computer exists. This is a direct HNDL risk.
Migration: replace RSA encryption with ML-KEM (FIPS 203), preferably hybrid
X25519+ML-KEM-768. Keep AES-GCM for bulk data; only the key transport/wrap
changes to KEM-based encapsulation (e.g. HPKE with ML-KEM).`,

	"ecc_usage": `Elliptic-curve operations (ECDH/ECDSA on P-256/X25519 etc.) are broken by
Shor's algorithm. ECDH key agreement is HNDL-exposed like RSA encryption.
Migration: hybrid X25519+ML-KEM-768 for agreement; ML-DSA (FIPS 204) for ECDSA
replacements. Library pointers per language as in general guidance.`,

	"dh_keyexchange": `Finite-field Diffie-Hellman is broken by Shor's algorithm; established shared
secrets recorded today are recoverable later (HNDL).
Migration: ML-KEM (FIPS 203) hybrid encapsulation. For TLS prefer X25519MLKEM768
(IANA 0x11EC). Remove ffdhe/DH parameter usage rather than enlarging key sizes -
bigger DH keys do not resist quantum attack meaningfully.`,

	"dsa_signing": `Classic DSA is deprecated (NIST SP 800-131A rev2 disallows >2048-bit after 2023/2024
transition guidance, NIST IR 8547) and quantum-vulnerable regardless.
Migration: ML-DSA (FIPS 204). Do not invest in DSA hardening - plan removal.`,

	"vulnerable_tls_kex": `TLS endpoints not negotiating a post-quantum hybrid group expose all traffic
to harvest-now-decrypt-later capture. The deployed standard is X25519MLKEM768
(IANA codepoint 0x11EC, RFC 10024), already default in Chrome/Firefox and major CDNs.
Server config examples (OpenSSL 3.5+):
- nginx:    ssl_ecdh_curve X25519MLKEM768:X25519:prime256v1;
- Apache:   SSLOpenSSLConfCmd Groups X25519MLKEM768:X25519:P-256
- HAProxy:  ssl-default-bind-curves X25519MLKEM768:X25519:secp256r1
Keep classical groups as fallback for legacy clients; ordering decides priority.`,

	"vulnerable_tls_certificate": `TLS certificates signed with RSA/ECDSA face future identity forgery, not HNDL.
No public CA issues ML-DSA certificates at scale yet; this finding is a
plan-and-track item, not an emergency. Track CA/Browser Forum PQC ballot progress
and inventory which internal/private CAs could pilot ML-DSA first.`,

	"vulnerable_dependency": `The dependency itself implements classical crypto internally. Two remediation
paths: (1) upgrade/migrate away from the package to a PQC-capable alternative,
or (2) if the package is unavoidable, isolate its usage behind an interface so
the crypto callsite can be swapped to a hybrid implementation. Track upstream
PQC adoption issues before rewriting integration code.`,

	"iac_misconfiguration": `Infrastructure TLS policy predates PQC. Update managed-policy documents to allow
TLS 1.3 with hybrid key exchange groups (X25519MLKEM768) once the platform
supports it (AWS ALB/NLB, Azure App Gateway and Cloudflare are rolling out
hybrid negotiation). Verify with: openssl s_client -groups X25519MLKEM768:X25519 -connect host:443`,

	"unresolved_algorithm_call": `The algorithm argument could not be resolved statically. Manual review required:
check what string/constant reaches this call site, then apply the matching
migration guidance (ML-KEM for key exchange/encryption, ML-DSA/SLH-DSA for signatures).`,
}
