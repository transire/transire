# Contributing to Transire

Thank you for your interest in contributing to Transire! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Contributor License Agreement](#contributor-license-agreement)
- [Submitting Changes](#submitting-changes)
- [Areas We Need Help](#areas-we-need-help)

## Getting Started

### Prerequisites

- **Go**: 1.21 or higher
- **Node.js**: 18+ (for CDK deployment testing)
- **Git**: Latest version
- **golangci-lint**: (optional but recommended)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/transire.git
   cd transire
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/transire/transire.git
   ```

## Development Setup

### 1. Install Dependencies

```bash
go mod download
```

### 2. Install Git Hooks (Recommended)

We provide pre-commit hooks that run the same checks as CI to catch issues early:

```bash
./scripts/install-hooks.sh
```

This installs a pre-commit hook that automatically runs:
- Code formatting checks (`gofmt`)
- Go vet
- golangci-lint (if installed)
- All tests with race detector
- CLI build verification

**Note**: You can bypass the hook in emergencies with `git commit --no-verify`, but this is not recommended.

### 3. Install golangci-lint (Optional but Recommended)

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.62.2
```

Or on macOS:
```bash
brew install golangci-lint
```

### 4. Build the CLI

```bash
go build -o transire-cli cmd/transire/main.go
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

Use prefixes:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Adding tests

### 2. Make Your Changes

Follow the [code style guidelines](#code-style) below.

### 3. Run Tests Locally

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/transire
```

### 4. Verify Code Quality

The pre-commit hook will run these automatically, but you can run them manually:

```bash
# Format code
gofmt -w .

# Run go vet
go vet ./...

# Run linter
golangci-lint run

# Build CLI
go build -o transire-cli cmd/transire/main.go
```

### 5. Commit Your Changes

If you installed the pre-commit hook, it will automatically run checks before allowing the commit:

```bash
git add .
git commit -m "feat: add support for X"
```

**Commit Message Format:**
```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance tasks

Examples:
```
feat: add support for custom CDK extensions

fix: resolve panic when queue handler returns empty array

docs: update README with VPC configuration examples

refactor: simplify runtime detection logic
```

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## Testing

### Unit Tests

Write unit tests for all new functionality:

```go
// pkg/transire/example_test.go
package transire

import "testing"

func TestExample(t *testing.T) {
    // Test implementation
}
```

### E2E Tests

For significant features, add E2E tests in the examples:

```go
// examples/simple-api/e2e_test.go
package main

import "testing"

func TestAPIEndpoint(t *testing.T) {
    // E2E test implementation
}
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/transire

# With verbose output
go test -v ./...

# With coverage
go test -cover ./...
```

## Code Quality

### Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (automatically done by pre-commit hook)
- Use meaningful variable and function names
- Keep functions small and focused
- Add comments for exported functions and complex logic
- Avoid premature optimization

### Package Structure

- **pkg/transire/**: Public SDK - user-facing APIs only
- **internal/**: Implementation code not exposed to users
- **internal/providers/**: Cloud-specific implementations
- **cmd/transire/**: CLI entry point
- **examples/**: Example applications

### Interface Design

- Keep interfaces small and focused
- Prefer composition over inheritance
- Use standard library interfaces where possible
- Define interfaces in `pkg/transire/interfaces.go`

### Error Handling

- Always handle errors explicitly
- Provide context in error messages
- Use `fmt.Errorf` with `%w` for error wrapping
- Don't panic in library code

Example:
```go
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
```

### Linting

The project uses golangci-lint with configuration in `.golangci.yml`. The pre-commit hook runs this automatically.

## Contributor License Agreement

Before contributing to Transire, you must sign our Contributor License Agreement (CLA). This ensures that the project can be maintained and distributed freely under the Mozilla Public License 2.0.

### Why do we require a CLA?

The CLA ensures that:
- You grant Transire maintainers the right to use your contributions
- You confirm that you have the legal right to grant this license
- Your contributions will be licensed under MPL 2.0
- The project can be maintained and distributed freely

### Signing the CLA

**For Individual Contributors:**
1. Review the CLA terms in [.github/CLA.md](.github/CLA.md)
2. [Sign the Individual CLA](https://github.com/transire/transire/issues/new?template=individual-cla.md&title=Individual%20CLA%20for%20@USERNAME) (replace @USERNAME with your GitHub username)
3. Fill out all required information accurately
4. Submit the issue

**For Corporate Contributors:**
1. Review the CLA terms in [.github/CLA.md](.github/CLA.md)
2. [Sign the Corporate CLA](https://github.com/transire/transire/issues/new?template=corporate-cla.md&title=Corporate%20CLA%20for%20COMPANY_NAME) (replace COMPANY_NAME with your organization)
3. The authorized signatory must complete all corporate information
4. List all employees authorized to contribute
5. Submit the issue

### After Signing

Once you've signed the CLA:
- Your GitHub username will be added to our CLA database
- The CLA bot will automatically approve your future pull requests
- You can start contributing to Transire!

**Note:** The CLA check is automated. When you open a pull request, if you haven't signed the CLA, the bot will comment with instructions. Simply comment with "I have read the CLA Document and I hereby sign the CLA" on your PR to complete the process.

## Submitting Changes

### Before Submitting

- [ ] Signed the CLA (if this is your first contribution)
- [ ] All tests pass locally
- [ ] Code is formatted with `gofmt`
- [ ] `go vet` passes
- [ ] golangci-lint passes
- [ ] All Go files have proper MPL 2.0 license headers
- [ ] CLI builds successfully
- [ ] Added tests for new functionality
- [ ] Updated documentation if needed
- [ ] Pre-commit hook passes (or checks run manually)

### Pull Request Guidelines

1. **Title**: Use a clear, descriptive title
   - Good: "Add support for VPC configuration in Lambda"
   - Bad: "Fix stuff"

2. **Description**: Provide context and details
   - What does this PR do?
   - Why is this change needed?
   - How does it work?
   - Any breaking changes?

3. **Testing**: Describe how you tested the changes
   - Manual testing steps
   - New test cases added
   - Edge cases considered

4. **Documentation**: Update relevant documentation
   - README.md
   - CLAUDE.md (for architecture changes)
   - Code comments
   - Examples

### CI Checks

All PRs must pass CI checks:
- Tests on Go 1.21, 1.22, 1.23
- golangci-lint
- Code formatting
- Build verification

### Review Process

1. Maintainers will review your PR
2. Address any feedback or requested changes
3. Once approved, a maintainer will merge your PR

## Areas We Need Help

We'd especially appreciate contributions in these areas:

### High Priority

- **Additional Cloud Providers**
  - Google Cloud Platform (Cloud Run)
  - Azure (Azure Functions)

- **Language SDKs**
  - Rust SDK
  - Python SDK
  - TypeScript/Node.js SDK

- **Documentation**
  - Tutorials and guides
  - Video walkthroughs
  - Blog posts

- **Testing**
  - Integration tests
  - Load testing
  - Security testing

### Medium Priority

- **Features**
  - WebSocket support
  - GraphQL integration
  - Built-in observability (traces, metrics)
  - Request validation middleware

- **Infrastructure**
  - OpenTofu as CDK alternative
  - Terraform modules
  - Pulumi support

- **Developer Experience**
  - IDE plugins
  - Debugging tools
  - Better error messages

### Good First Issues

Look for issues labeled `good first issue` on GitHub for beginner-friendly tasks.

## Questions?

- **Issues**: [GitHub Issues](https://github.com/transire/transire/issues)
- **Discussions**: [GitHub Discussions](https://github.com/transire/transire/discussions)

## Code of Conduct

Be respectful, inclusive, and professional. We're all here to build something great together.

## License

By contributing to Transire, you agree that your contributions will be licensed under the Mozilla Public License 2.0.

---

Thank you for contributing to Transire! 🚀
