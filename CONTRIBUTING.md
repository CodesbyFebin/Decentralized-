# Contributing to Decentralized.Host

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to the Decentralized.Host qualification system.

## Code of Conduct

Please review our [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing. We are committed to providing a welcoming and inclusive environment for all contributors.

## Getting Started

### Prerequisites
- Go 1.21 or later
- PostgreSQL 12+ or MySQL 8.0+ (for testing)
- Docker and Docker Compose (for local deployment testing)
- Git knowledge

### Development Setup
```bash
# Clone the repository
git clone https://github.com/CodesbyFebin/Decentralized-.git
cd Decentralized-

# Install dependencies
go mod download
go mod tidy

# Run tests
make test

# Build the system
make build

# Run locally with Docker Compose
docker-compose up -d
```

## How to Contribute

### Types of Contributions

1. **Bug Reports**: Report issues via GitHub Issues with reproduction steps
2. **Feature Requests**: Propose new features through GitHub Discussions
3. **Code Improvements**: Submit pull requests with enhancements
4. **Documentation**: Improve README, API docs, or deployment guides
5. **Testing**: Add test coverage or report edge cases
6. **Performance**: Optimize code or improve system benchmarks

### Workflow

1. **Fork the repository** and create a feature branch
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our code standards (see below)

3. **Test thoroughly**
   ```bash
   make test
   make lint
   ```

4. **Commit with clear messages**
   ```bash
   git commit -m "Brief description of changes"
   ```

5. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Open a Pull Request** using the provided template

### Code Standards

- **Go Code Style**: Follow [Effective Go](https://golang.org/doc/effective_go)
- **Naming**: Use clear, descriptive names for functions and variables
- **Comments**: Add comments for exported functions and complex logic
- **Error Handling**: Always handle errors appropriately
- **Testing**: Write tests for new functionality
- **Formatting**: Run `go fmt` on all code

### Commit Message Format

```
<type>: <brief description>

<optional detailed description>

Fixes #issue_number (if applicable)
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions or fixes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build, dependencies, or tooling

### Pull Request Guidelines

- Use the PR template provided in `.github/pull_request_template.md`
- Link related issues using `Fixes #123` or `Relates to #456`
- Ensure all tests pass and CI checks are green
- Request review from maintainers
- Be responsive to feedback and iterate as needed

## Areas for Contribution

### High Priority
- [ ] Performance optimizations in phase 4r components
- [ ] Enhanced GraphQL resolvers (phase 4s)
- [ ] Additional health checks and monitoring (phase 4t)
- [ ] Kubernetes operator implementation
- [ ] Cloud provider integrations

### Medium Priority
- [ ] Documentation improvements
- [ ] Example configurations
- [ ] Tool integrations (Prometheus, ELK, etc.)
- [ ] Additional test scenarios

### Community Help Wanted
- Areas marked with `good first issue` label
- Documentation spelling/grammar improvements
- Issue triage and labeling

## Testing Requirements

- Unit tests for new functions (minimum 80% coverage)
- Integration tests for critical paths
- Chaos scenario validation for resilience features
- Performance benchmarks for optimization changes

```bash
# Run all tests
make test

# Run specific test
go test ./pkg/providers -run TestName -v

# Run with coverage
make test-coverage

# Run chaos scenarios
make chaos-test
```

## Documentation

When submitting changes that affect functionality:
- Update relevant documentation in DEPLOYMENT.md or API_DOCUMENTATION.md
- Add comments to code for complex logic
- Update README.md if user-facing features change
- Add examples for new APIs

## Performance Considerations

- Benchmark critical paths before/after changes
- Consider memory impact for long-running operations
- Profile using pprof for optimization PRs
- Document performance characteristics in PR description

## Licensing

By contributing, you agree that your contributions will be licensed under the same license as the project. Ensure all code is your original work or properly attributed.

## Questions?

- Check existing [Issues](https://github.com/CodesbyFebin/Decentralized-/issues)
- Review [Documentation](QUALIFICATION_SYSTEM.md)
- Open a Discussion for questions

## Recognition

Contributors will be recognized in:
- Commit history
- Release notes
- Contributors section of README.md

Thank you for helping improve Decentralized.Host!

---

**Last Updated**: 2026-09-30
