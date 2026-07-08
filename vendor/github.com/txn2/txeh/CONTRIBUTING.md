# Contributing to txeh

Thank you for your interest in contributing to txeh!

## Development Setup

### Prerequisites

1. **Go 1.21+** required
2. Install development tools (pin to specific versions for reproducibility):

```bash
# Linting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2

# Security scanning
go install github.com/securego/gosec/v2/cmd/gosec@v2.19.0
go install golang.org/x/vuln/cmd/govulncheck@v1.0.4

# Import organization
go install golang.org/x/tools/cmd/goimports@v0.18.0
```

### Clone and Build

```bash
git clone https://github.com/txn2/txeh.git
cd txeh
make deps
make build
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Make changes with tests
4. Run `make verify` before committing
5. Submit PR with description using the PR template

## Code Style

### Formatting
- Run `make fmt` before committing
- Code must pass `golangci-lint` with project config

### Documentation
- All exported functions need doc comments
- Doc comments must start with the name being documented
- Comments should end with a period

### Error Handling
- Always check and handle errors (no `_` for errors)
- Wrap errors at boundaries: `fmt.Errorf("context: %w", err)`
- Use `errors.Is` and `errors.As`, never string comparison

### Naming
- Follow [Effective Go](https://go.dev/doc/effective_go) naming conventions
- Error messages should be lowercase, no punctuation
- Avoid stuttering (e.g., `http.HTTPServer` → `http.Server`)

## Commit Messages

Use conventional commits:
- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `test:` Test additions/changes
- `refactor:` Code refactoring
- `perf:` Performance improvements
- `security:` Security fixes
- `chore:` Maintenance tasks

Examples:
```
feat: add support for IPv6 CIDR ranges
fix: handle empty hostname in AddHost
docs: update README with new CLI flags
test: add coverage for RemoveAddress edge cases
```

## Testing

### Requirements
- Add tests for new functionality
- Maintain 82%+ coverage
- Run `go test -race ./...` to check for race conditions

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run linting
make lint

# Run full verification suite
make verify

# Run specific test
go test -v -run TestSpecific ./...
```

## Verification Checklist

Before submitting a PR, ensure:

- [ ] `make verify` passes
- [ ] New code has tests
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventions

## Reporting Issues

- Check existing issues before creating a new one
- Use the issue templates provided
- Include version, OS, and reproduction steps

## Security Vulnerabilities

Please report security vulnerabilities privately. See [SECURITY.md](SECURITY.md) for details.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
