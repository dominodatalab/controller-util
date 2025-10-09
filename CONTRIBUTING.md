# Contributing to controller-util

Thank you for your interest in contributing to controller-util! This document provides guidelines and instructions for contributing.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help create a welcoming environment for all contributors

## Getting Started

### Prerequisites

- Go 1.25 or later
- Git
- Basic understanding of Kubernetes controllers and controller-runtime

### Development Setup

1. **Fork and clone the repository**

```bash
git clone https://github.com/dominodatalab/controller-util.git
cd controller-util
```

1. **Install development tools**

```bash
make tools
```

1. **Run tests to verify setup**

```bash
make test
```

## Development Workflow

### Making Changes

1. **Create a feature branch**

```bash
git checkout -b feature/your-feature-name
```

1. **Make your changes**
   - Write clear, concise code
   - Follow Go best practices and idioms
   - Add tests for new functionality
   - Update documentation as needed

1. **Run checks locally**

```bash
make fmt      # Format code
make lint     # Run linters
make test     # Run tests
make ci       # Run all CI checks
```

1. **Commit your changes**

```bash
git add .
git commit -m "feat: add new feature description"
```

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test additions or changes
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

1. **Push and create a Pull Request**

```bash
git push origin feature/your-feature-name
```

## Pull Request Guidelines

### Before Submitting

- [ ] All tests pass (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linters pass (`make lint`)
- [ ] Documentation is updated
- [ ] Commit messages follow conventional commits
- [ ] Changes are focused and atomic

### PR Description

Include:

- **What**: Brief description of the change
- **Why**: Motivation and context
- **How**: Implementation approach (if non-obvious)
- **Testing**: How you tested the changes
- **Related Issues**: Link to any related issues

### Review Process

1. Automated CI checks must pass
2. At least one maintainer approval required
3. Address review feedback promptly
4. Keep PR updated with main branch

## Coding Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` and `goimports` for formatting (automated via `make fmt`)
- Run `golangci-lint` and fix issues (automated via `make lint`)

### Code Organization

```plain
controller-util/
├── action/         # Resource action utilities
├── collection/     # Collection helpers
├── core/           # Core reconciler and component interfaces
├── metadata/       # Kubernetes metadata helpers
```

### Testing

- Write table-driven tests where appropriate
- Use meaningful test names that describe the scenario
- Mock external dependencies
- Aim for high code coverage on new features

```go
func TestComponentName(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    interface{}
        wantErr bool
    }{
        {
            name:  "valid case",
            input: validInput,
            want:  expectedOutput,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Documentation

- Add package-level godoc comments for all packages
- Document exported types, functions, and methods
- Keep README.md updated with new features

## Project Structure

### Adding New Components

When adding new utilities or components:

1. Choose appropriate package (`action/`, `core/`, etc.)
2. Add comprehensive tests
3. Document with godoc comments
4. Update README.md if user-facing

## Reporting Issues

### Bug Reports

Include:

- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (Go version, K8s version)
- Relevant logs or error messages

### Feature Requests

Include:

- Use case description
- Proposed API or interface
- Alternative approaches considered
- Willingness to implement

## Getting Help

- Review existing [documentation](./README.md)
- Check [existing issues](https://github.com/dominodatalab/controller-util/issues)
- Ask questions in pull requests or issues

## Release Process

(For maintainers)

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create git tag: `git tag -a v0.x.0 -m "Release v0.x.0"`
4. Push tag: `git push origin v0.x.0`
5. GitHub Actions will handle the release

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see [LICENSE](./LICENSE)).

## Questions?

Feel free to open an issue for any questions or clarifications needed.

Thank you for contributing! 🎉
