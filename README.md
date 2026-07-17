<div align="center">

# 🔐 VYALA

**Post-quantum cryptography vulnerability scanner for developers.**

A CI-native tool that detects quantum-vulnerable crypto in your codebase and posts actionable findings directly on pull requests.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat)](https://www.apache.org/licenses/LICENSE-2.0)(LICENSE)
[![Semgrep](https://img.shields.io/badge/powered%20by-Semgrep-1B1B1B?style=flat&logo=semgrep&logoColor=white)](https://semgrep.dev)
[![NIST FIPS 203/204/205](https://img.shields.io/badge/NIST-FIPS%20203%20%7C%20204%20%7C%20205-002868?style=flat)](https://csrc.nist.gov/pubs/fips/203/final)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](CONTRIBUTING.md)

</div>

---

## What it does

- 🔍 **Scans Python, JavaScript/TypeScript** for RSA, ECDSA/ECDH, DH, and MD5/SHA‑1 in security contexts
- ⚙️ **Runs as a CLI** (`vyala --path .`) or as a **GitHub Action** on pull requests
- ⚡ **Diff-aware** — only scans changed lines, keeps PR comments under 30 seconds
- 🆔 **Stable finding IDs** — the same code always produces the same ID, so re-scans update the existing PR comment instead of spamming
- 📋 **CBOM output** — structured JSON (Cryptographic Bill of Materials) for compliance and tracking, with each finding mapped to its NIST-standard replacement

---

## Quick start

### CLI

```bash
# Install semgrep
pip install semgrep

# Download the vyala binary, or build from source
go build -o vyala ./cmd/scanner

# Scan a repo
./vyala --path /your/repo
```

Add `--json report.json` to also write a full CBOM report.

### GitHub Action

```yaml
# .github/workflows/vyala.yml
name: VYALA Scan
on: [pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: your-org/vyala-action@v1
        with:
          severity-threshold: medium
```

Findings show up as a single, auto-updating comment on the pull request.

---

## Example output

```
========== PQC Vulnerability Scan Report ==========

HIGH Severity (1 finding):
  - api/auth.py:14 | RSA | rsa_keygen
    Suggested: Replace with ML-KEM (FIPS 203) using a hybrid construction.

MEDIUM Severity (1 finding):
  - utils.py:22 | ECC | ecc_usage
    Suggested: Replace with ML-DSA (FIPS 204) for signing operations.

=====================================================
```

Every finding is also written to a structured CBOM in JSON — see [`docs/cbom-schema.md`](docs/cbom-schema.md).

---

## Why VYALA

Enterprise PQC discovery tools take large organizations 12–24 months and $50K–$150K/year to produce a cryptographic inventory. VYALA gives developers a first report in minutes, for free, directly inside the workflow they already use — then keeps watching on every pull request after that.

---

## Roadmap

- [x] Core scanner engine (Python, JS/TS)
- [x] Diff-aware scanning + stable finding IDs
- [ ] GitHub Action + PR comment posting
- [ ] Dependency & IaC/TLS config scanning
- [ ] Dashboard + continuous monitoring
- [ ] AI-assisted fix suggestions, grounded in FIPS 203/204/205

## License

[Apache 2.0](LICENSE)

## Contributing

Issues and pull requests welcome — see [`CONTRIBUTING.md`](CONTRIBUTING.md).