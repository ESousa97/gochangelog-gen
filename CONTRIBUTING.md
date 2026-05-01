# Contributing to gochangelog-gen

Thank you for your interest in contributing. This document describes the development workflow, coding standards, and contribution process.

## Prerequisites

- Go >= 1.22
- [golangci-lint](https://golangci-lint.run/welcome/install/) (latest)
- [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) (`go install golang.org/x/tools/cmd/goimports@latest`)
- (Optional) [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) (`go install golang.org/x/vuln/cmd/govulncheck@latest`)

## Development Setup

```bash
git clone https://github.com/esousa97/gochangelog-gen.git
cd gochangelog-gen
go mod download
make build
```

## Code Style

This project follows standard Go conventions:

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Google Go Style Guide](https://google.github.io/styleguide/go/)

All code must pass `gofmt`, `goimports`, and `golangci-lint` without warnings.

## Testing

```bash
# Run all tests
make test

# Run tests with race detector
make test-race

# Run tests with coverage
make test-cover
```

All new code should include tests. Use table-driven tests following Go conventions.

## Commit Convention

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>
```

| Type | Description |
|---|---|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Refactoring without behavior change |
| `docs` | Documentation changes |
| `test` | Adding or fixing tests |
| `chore` | Maintenance, dependencies, configs |
| `ci` | CI/CD changes |
| `perf` | Performance improvements |
| `security` | Security fixes |

## Pull Request Process

1. Fork the repository
2. Create a feature branch from `main` (`git checkout -b feat/my-feature`)
3. Make your changes following the code style guidelines
4. Add or update tests as needed
5. Run the full validation suite: `make validate`
6. Commit with a Conventional Commit message
7. Push and open a Pull Request against `main`

### PR Checklist

- [ ] Code follows Go conventions and passes `golangci-lint`
- [ ] Tests added or updated for the changes
- [ ] Documentation updated if public API changed
- [ ] `go mod tidy` produces no diff
- [ ] CI is passing

## Areas Where Contributions Are Welcome

- New commit type mappings and changelog categories
- Monorepo support
- Additional output formats (JSON, HTML)
- Improved template customization
- Documentation improvements
- Test coverage improvements

## Questions?

Open a [Discussion](https://github.com/esousa97/gochangelog-gen/discussions) for questions or ideas.
