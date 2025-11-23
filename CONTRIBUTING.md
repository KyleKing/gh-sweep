# Contributing to gh-sweep

Thank you for considering contributing to gh-sweep! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites
- Go 1.21 or higher
- [mise](https://mise.jdx.dev/) (recommended for task management)
- Git
- GitHub account with personal access token

### Development Setup

1. **Fork and clone the repository:**
```bash
git clone https://github.com/YOUR_USERNAME/gh-sweep.git
cd gh-sweep
```

2. **Install dependencies:**
```bash
# Using mise (recommended)
mise install

# Or manually install Go dependencies
go mod download
```

3. **Configure GitHub token:**
```bash
# Option 1: Use gh CLI
gh auth login

# Option 2: Set environment variable
export GITHUB_TOKEN="ghp_your_token_here"
```

4. **Run tests to verify setup:**
```bash
mise run test
```

## Development Workflow

### Task Commands

We use `mise` for task management. Common commands:

```bash
# Run tests
mise run test

# Run linter
mise run lint

# Format code
mise run format

# Type check
mise run typecheck

# Run development build
mise run dev

# Build production binary
mise run build

# Run CI checks locally (test + lint + typecheck)
mise run ci
```

### Code Style

We follow standard Go conventions:

- **Formatting:** Use `gofumpt` (stricter than `gofmt`)
- **Linting:** `golangci-lint` with configuration in `.golangci.yml`
- **Imports:** Use `goimports` for automatic import management

Run before committing:
```bash
mise run format
mise run lint
```

### Testing

#### Unit Tests
```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/github/...

# With coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### TUI Tests
We use [teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest) for testing Bubble Tea components:

```go
func TestBranchListView(t *testing.T) {
    m := NewBranchListModel()
    tm := teatest.NewTestModel(t, m)

    // Send key press
    tm.Send(tea.KeyMsg{Type: tea.KeyDown})

    // Verify state
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("branch-name"))
    })
}
```

#### Integration Tests
Integration tests that call real GitHub APIs should be tagged:

```go
//go:build integration

func TestFetchRealBranches(t *testing.T) {
    // ... test code
}
```

Run integration tests separately:
```bash
go test -tags=integration ./...
```

### Documentation

- **Code comments:** Use godoc-style comments for exported functions/types
- **README updates:** Update `README.md` for user-facing changes
- **Phase docs:** Update relevant `.phases/*.md` files for design changes

## Contributing Guidelines

### Submitting Issues

**Bug Reports:**
- Use the bug report template
- Include Go version, OS, and gh-sweep version
- Provide minimal reproduction steps
- Include relevant logs/screenshots

**Feature Requests:**
- Check existing issues and phase docs first
- Explain the use case and problem being solved
- Consider if it fits gh-sweep's philosophy (see [anti-phases.md](.phases/anti-phases.md))

### Pull Requests

1. **Create a feature branch:**
```bash
git checkout -b feature/your-feature-name
```

2. **Make your changes:**
- Write tests for new functionality
- Update documentation
- Follow code style guidelines
- Keep commits atomic and well-described

3. **Test your changes:**
```bash
mise run test
mise run lint
mise run typecheck
```

4. **Commit with conventional commits:**
```bash
# Format: <type>(<scope>): <description>
git commit -m "feat(branches): add stacked PR creation"
git commit -m "fix(comments): resolve caching race condition"
git commit -m "docs(readme): update installation instructions"
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance tasks

5. **Push and create PR:**
```bash
git push origin feature/your-feature-name
```

Then open a PR on GitHub with:
- Clear description of changes
- Link to related issues
- Screenshots/demos for UI changes
- Test results

### PR Review Process

1. **Automated checks:**
   - Tests must pass
   - Linter must pass
   - Coverage should not decrease significantly

2. **Code review:**
   - At least one maintainer approval required
   - Address review comments
   - Keep discussion focused and respectful

3. **Merge:**
   - Squash and merge (default)
   - Use meaningful commit message
   - Delete branch after merge

## Architecture Guidelines

### Project Structure

```
gh-sweep/
├── cmd/                  # Cobra CLI commands
│   ├── root.go          # Main entry point
│   ├── branches.go      # Branch management command
│   └── ...
├── internal/            # Private application code
│   ├── tui/            # Bubble Tea components
│   │   ├── model.go    # Main model
│   │   └── components/ # Reusable TUI components
│   ├── github/         # GitHub API client
│   ├── cache/          # Caching layer
│   └── config/         # Configuration management
├── pkg/                # Public libraries (if any)
└── .phases/            # Design documentation
```

### Bubble Tea Patterns

Follow the Elm Architecture:

```go
type Model struct {
    // State
}

func (m Model) Init() tea.Cmd {
    // Initialize
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle messages
}

func (m Model) View() string {
    // Render UI
}
```

**Best practices:**
- Keep models focused (single responsibility)
- Use composition for complex UIs
- Separate business logic from rendering
- Test models with teatest

### GitHub API Guidelines

- **Use go-gh library:** `github.com/cli/go-gh/v2`
- **Handle rate limits:** Implement exponential backoff
- **Cache aggressively:** Use SQLite for persistence
- **Respect ETags:** For conditional requests
- **Paginate correctly:** Handle large result sets

Example:
```go
func FetchBranches(client *github.Client, repo string) ([]Branch, error) {
    // Check cache first
    if cached, found := cache.Get(repo); found {
        return cached, nil
    }

    // Fetch from API with pagination
    branches, err := client.ListBranches(repo, &github.ListOptions{
        PerPage: 100,
    })

    // Cache result
    cache.Set(repo, branches, 1*time.Hour)

    return branches, err
}
```

### Error Handling

- **Use standard errors:** `fmt.Errorf("context: %w", err)`
- **Provide context:** Include relevant information
- **User-friendly messages:** In TUI, show actionable errors
- **Log details:** Use structured logging for debugging

## Release Process

Maintainers handle releases:

1. Update version in `cmd/version.go`
2. Update `CHANGELOG.md`
3. Create git tag: `git tag v1.2.3`
4. Push tag: `git push origin v1.2.3`
5. GitHub Actions builds and publishes release

## Community

- **Discussions:** Use GitHub Discussions for questions
- **Issues:** Use GitHub Issues for bugs and features
- **Code of Conduct:** Be respectful and inclusive

## Questions?

- Check [README.md](README.md) first
- Review [phase documentation](.phases/)
- Search existing issues
- Ask in GitHub Discussions

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
