# Contributing to Decentralized.Host

Thank you for your interest in contributing! This document provides guidelines for participating in the project.

## Getting Started

### Prerequisites
- Go 1.26 or later
- Git
- Docker (optional, for testing container runtime)
- Python 3 (optional, for conformance tests)

### Development Setup

```bash
# Clone the repository
git clone https://github.com/CodesbyFebin/Decentralized-
cd Decentralized-

# Install dependencies
go mod download

# Build binaries
make build

# Run unit tests
make test

# Start a dev cluster
./bin/dh dev up --dir ./devcluster
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

Use descriptive branch names:
- `feature/` for new features
- `fix/` for bug fixes
- `docs/` for documentation
- `test/` for test improvements
- `refactor/` for code refactoring

### 2. Make Your Changes

**Code Style:**
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Run `gofmt` and `goimports` on your changes
- Keep functions small and focused
- Add comments for exported functions
- Use meaningful variable names

**Testing:**
- Add tests for new features
- Update existing tests for bug fixes
- Ensure all tests pass: `make test`
- Run race detector: `make race`
- Test in a dev cluster: `./bin/dh dev up --dir ./devcluster`

### 3. Commit Your Changes

Use clear, descriptive commit messages:

```
[type]: [scope] [short description]

[detailed explanation if needed]

[related issue(s): #123, #456]
```

Examples:
- `feat: identity - add key rotation support`
- `fix: storage - prevent concurrent repairs causing conflicts`
- `docs: runbooks - add failover procedure`
- `test: chaos - add network partition recovery scenario`

### 4. Push and Create a Pull Request

```bash
git push origin feature/your-feature-name
```

**In your PR description:**
- What does this change do?
- Why is it needed?
- How was it tested?
- Any breaking changes?
- Links to related issues

## Testing Requirements

All contributions must pass:

```bash
# Unit tests
make test

# Race detector
make race

# Integration tests (if modifying control plane, node, or storage)
make integration

# Conformance (if modifying protocol)
make conformance

# Chaos (if modifying robustness)
make chaos
```

## Areas for Contribution

### 🚀 High-Impact Areas
- **Edge proxy improvements:** L7 routing, rate limiting, connection pooling
- **Scheduler enhancements:** Advanced placement strategies, affinity rules
- **Observability:** Metrics, tracing, dashboards
- **Documentation:** Runbooks, examples, architecture diagrams
- **Performance:** Optimization in critical paths

### 🐛 Bugs & Issues
Check [GitHub Issues](https://github.com/CodesbyFebin/Decentralized-/issues) for:
- `good-first-issue` — Great starting points for new contributors
- `help-wanted` — Areas where community input is needed
- `bug` — Issues to fix

### 📖 Documentation
- Improve runbooks in `docs/runbooks/`
- Clarify protocol documentation in `docs/protocol/`
- Add architecture diagrams to `docs/`
- Update README with examples

### ✅ Testing
- Add chaos scenarios in `pkg/chaos/`
- Expand conformance vectors in `conformance/`
- Improve integration tests in `tests/integration/`
- Add edge-case unit tests

## Code Review Process

1. **Automated checks** run on every PR:
   - Build passes (`make build`)
   - Tests pass (`make test`)
   - No linting errors (`gofmt`, `goimports`)

2. **Code review** by maintainers:
   - Correctness and design
   - Performance implications
   - Security considerations
   - Documentation completeness

3. **Approval and merge:**
   - At least one maintainer approval required
   - All CI checks must pass
   - Squash commits if needed

## Reporting Issues

Found a bug? Have a feature request? Please open an issue:

**For bugs:**
- Describe the issue clearly
- Include steps to reproduce
- Attach relevant logs or error messages
- List your environment (Go version, OS, etc.)

**For features:**
- Explain the use case
- Describe the desired behavior
- Suggest implementation approach (optional)
- Link to related issues

## Security Policy

### Reporting Security Vulnerabilities

**Do not open a public issue for security vulnerabilities.** Instead:

1. Email [security@decentralized-host.dev](mailto:codesbyfebin@gmail.com?subject=Security%20Vulnerability) with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

2. Allow 48 hours for response
3. We'll work with you to verify and patch the issue
4. You'll be credited in the security advisory

### Security Best Practices

When contributing:
- ✅ Validate all inputs (user, network, file system)
- ✅ Use constant-time comparison for cryptographic values
- ✅ Avoid timing attacks in security-critical paths
- ✅ Log security events for audit purposes
- ✅ Test with the race detector (`make race`)

## Architecture & Design Principles

Before making large changes, review:

1. **[Architecture](docs/architecture.md)** — System design
2. **[Trust Model](docs/trust-model.md)** — Security boundaries
3. **[Decisions](docs/decisions/)** — Design rationale
4. **[Protocol Spec](docs/protocol/dh-v1.md)** — Wire format and semantics

Key principles:
- **Sovereignty over consensus** — Hosts make local decisions
- **Signed intent over silent mutations** — All work is cryptographically bound
- **Honesty over assertion** — Observed state is measured, not claimed
- **P2P over gateway** — Mesh topology, not centralized routing
- **Content-addressed storage** — BLAKE3 hashes, not paths

## Documentation

### Writing Good Code Comments

```go
// Package storage provides content-addressed storage with BLAKE3 hashing.
// Objects are deduplicated via FastCDC chunking and repaired via Merkle
// anti-entropy gossip.
package storage

// Put adds an object to storage, computing its BLAKE3 hash and storing
// chunks across the mesh. Returns the object hash.
func (s *Store) Put(ctx context.Context, data []byte) (string, error) {
    // ...
}
```

### Writing Good Runbooks

See `docs/runbooks/` for examples. Runbooks should:
- Have clear prerequisites
- Use step-by-step procedures
- Include expected output
- Link to troubleshooting guides
- Provide rollback procedures

## Community Guidelines

### Code of Conduct

We're committed to providing a welcoming and inclusive environment. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

### Be Respectful

- Assume good intent
- Critique ideas, not people
- Welcome diverse perspectives
- Be patient with new contributors

### Participate Constructively

- Engage in discussions
- Provide actionable feedback
- Help others learn
- Share knowledge generously

## Licensing

By contributing to Decentralized.Host, you agree that your contributions will be licensed under the same license as the project (AGPL v3). See [LICENSE](LICENSE) for details.

## Questions?

- **GitHub Issues:** For bugs and features
- **GitHub Discussions:** For questions and ideas
- **Email:** codesbyfebin@gmail.com

---

Thank you for contributing to making infrastructure more sovereign! 🚀
