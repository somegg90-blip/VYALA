# Contributing to VYALA
First off, thank you for considering contributing to VYALA! It’s people like you that make the open-source community such a great place.

VYALA aims to be the standard for developer-first, CI/CD-native Post-Quantum Cryptography discovery. We welcome contributions of all kinds, including new Semgrep rules, language support, bug fixes, and documentation improvements.

## 🛡️ Reporting Security Vulnerabilities
Do NOT open a public GitHub issue for security vulnerabilities in VYALA itself. Instead, please review our security policy (or message the maintainers directly) so we can address it responsibly.

## 🐛 Reporting Bugs or Feature Requests

- Ensure the bug/feature hasn't already been reported by searching the [Issues](https://chat.z.ai/issues).
- If you can't find an existing issue, [open a new one](https://chat.z.ai/issues/new).
- Please include as much detail as possible:
- A clear title and description.
- The exact steps to reproduce the bug.
- The expected behavior vs. what actually happened.
- Your OS and Go version (if applicable).

## 💻 Development Setup
VYALA is built in Go and relies on Semgrep for AST pattern matching. To set up your local environment:

1. **Prerequisites:**

- [Go](https://golang.org/doc/install) (v1.21 or higher)
- [Python 3](https://www.python.org/downloads/) & `pip`
- [PostgreSQL](https://www.postgresql.org/download/) (only if you are working on the backend/dashboard features)
2. **Install Semgrep:**

```
pip install semgrep
```

1. **Clone and Build:** git clone https://github.com/somegg90-blip/vyala.git
cd vyala
go build -o vyala ./cmd/scanner/                                                   **Run Tests:**
We use Go's built-in testing framework. Before submitting a PR, ensure all tests pass:        go test ./...                                                                       ## 📝 How to Add a New Detection Rule
VYALA uses Semgrep YAML rules located in `internal/rules/`. If you want to add detection for a new vulnerable cryptographic pattern:

1. Create a new `.yaml` file in `internal/rules/` (e.g., `go-rsa-signing.yaml`).
2. Follow the Semgrep rule syntax. Include the `metadata` fields (`algorithm`, `category`, `suggested_replacement`) so VYALA can map it to a NIST standard.
3. Add test fixtures in the `testdata/` directory to ensure your rule works.
4. Run `go test ./internal/engine/ -v` to verify your rule triggers correctly.

## 🔄 Pull Request Process

1. Fork the repository and create your branch from `main`.
2. If you've added code that should be tested, add tests in the `*_test.go` files.
3. Ensure the test suite passes (`go test ./...`).
4. Make sure your code lints cleanly (e.g., `go vet ./...`).
5. Open a Pull Request with a clear title and description of your changes. Reference any related issues (e.g., `Closes #12`).

Thank you for contributing!                    
