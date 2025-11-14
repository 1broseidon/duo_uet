# Contributing to User Experience Toolkit

## Development Setup

### Prerequisites
- Go 1.25.0 or higher
- Git

### Getting Started

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run tests:
   ```bash
   go test ./...
   ```

## Pre-commit Hooks

This project uses Git pre-commit hooks to ensure code quality before commits.

### Automatic Test Runner

The pre-commit hook automatically runs `go test ./...` before each commit. If any tests fail, the commit will be blocked.

**Example output on success:**
```
Running pre-commit checks...

🧪 Running go test ./...
✅ All tests passed
   ok  	user_experience_toolkit/internal/config
   ok  	user_experience_toolkit/internal/handlers
   ...
```

**Example output on failure:**
```
Running pre-commit checks...

🧪 Running go test ./...
❌ Tests failed! Commit aborted.

Test output:
--- FAIL: TestSomething (0.00s)
    ...

Fix the failing tests before committing.
To skip this check, use: git commit --no-verify
```

### Bypassing Pre-commit Hooks

In rare cases where you need to commit without running tests (not recommended):
```bash
git commit --no-verify -m "Your commit message"
```

**Note:** Only use `--no-verify` in exceptional circumstances, such as:
- Work-in-progress commits on a feature branch
- Urgent hotfixes where tests will be fixed in a follow-up commit
- When the CI/CD pipeline will catch the issues

## Running Tests

### Run all tests
```bash
go test ./...
```

### Run tests with coverage
```bash
go test ./... -cover
```

### Run tests for a specific package
```bash
go test ./internal/config -v
```

### Generate coverage report
```bash
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Code Quality Tools

### Run go vet
```bash
go vet ./...
```

### Run staticcheck
```bash
# Install staticcheck first
go install honnef.co/go/tools/cmd/staticcheck@latest

# Run staticcheck
staticcheck ./...
```

### Check cyclomatic complexity
```bash
# Install gocyclo first
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

# Find functions with complexity > 15
gocyclo -over 15 .
```

## Code Style

This project follows the [Effective Go](https://go.dev/doc/effective_go) guidelines:

- Use `gofmt` for formatting (automatically done by most editors)
- Use MixedCaps for exported names, mixedCaps for unexported
- Always handle errors - never ignore them
- Document all exported functions and types
- Keep functions focused and simple (cyclomatic complexity < 15)

## Pull Request Process

1. Ensure all tests pass locally
2. Update documentation if needed
3. Add tests for new functionality
4. Ensure code follows Go best practices
5. Create a pull request with a clear description

## Testing Guidelines

- Aim for 80%+ coverage on core business logic packages
- Write table-driven tests when testing multiple scenarios
- Use meaningful test names that describe the scenario
- Mock external dependencies (HTTP clients, databases, etc.)

## Commit Message Format

Follow conventional commit format:
```
<type>: <description>

[optional body]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks
