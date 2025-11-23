# gh-sweep 🧹

> A powerful Terminal User Interface (TUI) for GitHub repository management, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

**gh-sweep** helps you manage multiple GitHub repositories interactively from your terminal. It fills gaps in the GitHub ecosystem by providing cross-repo visibility, bulk operations, and intelligent analysis.

## Features

### Core Features (Phase 1)
- **🌳 Interactive Branch Management**: Visualize branch relationships, create stacked PRs, batch delete with dependency analysis
- **🛡️ Branch Protection Rules**: Compare and sync protection rules across repositories
- **💬 Unresolved PR Comments**: Search, filter, and review unresolved comments with advanced filters and caching

### Coming Soon
- **⚡ GitHub Actions Analytics**: Performance trends, flaky test detection, error log extraction (Phase 2)
- **⚙️ Cross-Repo Settings**: Visual diff and sync of repository settings (Phase 2)
- **🔗 Webhook Management**: Org-wide overview and debugging (Phase 2)
- **👥 Collaborator Management**: Time-boxed access grants for contractors/trials (Phase 3)
- **🔐 Secrets Audit**: Visibility into secrets usage and compliance (Phase 3)
- **📦 Release Overview**: Multi-repo release dashboard and version comparison (Phase 3)
- **🔌 Integrations**: Linear, mani, ghq, and other local git tools (Phase 4)
- **📊 Analytics**: CI runs, AI reviews, comment stats, contributor metrics (Phase 5)

## Why gh-sweep?

**Fills Real Gaps:**
- ✅ No interactive TUI exists for branch protection management
- ✅ No TUI for unresolved PR comment review with advanced filtering
- ✅ No tool bridges GitHub Actions metadata with AI-friendly error extraction
- ✅ Cross-repo settings comparison is CLI-only elsewhere

**Complements Existing Tools:**
- Use **Renovate** for dependency updates → Use **gh-sweep** to visualize health
- Use **Pulumi/Terraform** for IaC → Use **gh-sweep** to detect drift
- Use **BuildPulse** for ML-based flaky tests → Use **gh-sweep** for simple statistics

See [anti-phases.md](.phases/anti-phases.md) for what we don't do and recommended alternatives.

## Installation

### From Source (Development)
```bash
git clone https://github.com/KyleKing/gh-sweep.git
cd gh-sweep

# Using mise (recommended)
mise install
mise run build

# Or using go directly
go build -o gh-sweep
```

### Using Go Install
```bash
go install github.com/KyleKing/gh-sweep@latest
```

### Homebrew (Coming Soon)
```bash
brew install KyleKing/tap/gh-sweep
```

## Quick Start

```bash
# Configure GitHub token (uses gh CLI if available)
export GITHUB_TOKEN="ghp_..."

# Or authenticate with gh CLI
gh auth login

# Launch interactive branch management
gh-sweep branches

# Review unresolved PR comments
gh-sweep comments --repo owner/repo

# Compare branch protection rules
gh-sweep protection --repos "owner/repo1,owner/repo2"

# Launch full TUI
gh-sweep
```

## Configuration

Create `.gh-sweep.yaml` in your home directory or project root:

```yaml
# Default GitHub organization
default_org: your-org

# Repositories to manage
repositories:
  - owner/repo1
  - owner/repo2

# Cache settings
cache:
  ttl: 1h
  path: ~/.cache/gh-sweep

# Filters
filters:
  # Exclude bot users from comment search
  exclude_users:
    - dependabot
    - renovate

# Linear integration (optional)
linear:
  api_key: lin_api_...
  workspace: your-workspace

# mani integration (optional)
mani:
  config_path: ./mani.yaml
```

## Usage Examples

### Branch Management
```bash
# Interactive branch visualization
gh-sweep branches

# Show branch tree for specific repo
gh-sweep branches --repo owner/repo --tree

# Create stacked PRs from selected branches
gh-sweep branches --stacked-prs
```

### Comment Review
```bash
# Search unresolved comments
gh-sweep comments --repo owner/repo

# Filter by author
gh-sweep comments --author username

# Filter by date range
gh-sweep comments --since 2024-01-01

# Fuzzy search in comment text
gh-sweep comments --search "TODO|FIXME"
```

### Branch Protection
```bash
# Compare protection rules
gh-sweep protection --repos "owner/repo1,owner/repo2"

# Apply template to multiple repos
gh-sweep protection --template templates/default.yaml --apply

# Show drift from baseline
gh-sweep protection --baseline owner/baseline-repo
```

## Development

### Prerequisites
- Go 1.21+
- [mise](https://mise.jdx.dev/) (recommended) or go task runner
- GitHub personal access token with repo scope

### Setup
```bash
# Clone repository
git clone https://github.com/KyleKing/gh-sweep.git
cd gh-sweep

# Install dependencies
mise install

# Run tests
mise run test

# Run linter
mise run lint

# Format code
mise run format

# Run development build
mise run dev
```

### Project Structure
```
gh-sweep/
├── cmd/                  # CLI commands (Cobra)
├── internal/
│   ├── tui/             # Bubble Tea TUI components
│   ├── github/          # GitHub API client
│   ├── cache/           # Caching layer (SQLite)
│   └── config/          # Configuration management
├── .phases/             # Phase documentation
├── .github/workflows/   # CI/CD
└── README.md
```

### Running Tests
```bash
# Run all tests
mise run test

# Run specific package tests
go test ./internal/github/...

# Run with coverage
go test -cover ./...

# Run TUI tests (using teatest)
go test ./internal/tui/...
```

## Documentation

- **[Phase 1: MVP](.phases/phase_1_mvp.md)** - Branch management, protection rules, comment review
- **[Phase 2: Actions & Settings](.phases/phase_2_actions_and_settings.md)** - GitHub Actions analytics, settings comparison
- **[Phase 3: Access & Releases](.phases/phase_3_access_and_releases.md)** - Collaborator management, secrets audit, releases
- **[Phase 4: Integrations](.phases/phase_4_integrations.md)** - Linear, mani, local git tools
- **[Phase 5: Analytics](.phases/phase_5_analytics.md)** - CI runs, AI reviews, contributor metrics
- **[Anti-Phases](.phases/anti-phases.md)** - What we don't do and alternatives

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## Roadmap

- [x] Phase 1 planning and documentation
- [ ] Phase 1 implementation (MVP)
- [ ] Phase 2 implementation (Actions & Settings)
- [ ] Phase 3 implementation (Access & Releases)
- [ ] Phase 4 implementation (Integrations)
- [ ] Phase 5 implementation (Analytics)

## Alternatives & Related Tools

**When NOT to use gh-sweep:**
- Automated dependency updates → Use [Renovate](https://github.com/renovatebot/renovate)
- Infrastructure as Code → Use [Pulumi](https://www.pulumi.com/blog/managing-github-with-pulumi/) or [Terraform](https://registry.terraform.io/providers/integrations/github/)
- Stale issue automation → Use [GitHub Actions](https://github.com/actions/stale)
- Real-time CI monitoring → Use [watchgha](https://github.com/nedbat/watchgha)

See [anti-phases.md](.phases/anti-phases.md) for detailed comparison and usage guidance.

**Related TUI tools:**
- [gh-dash](https://github.com/dlvhdr/gh-dash) - GitHub dashboard for PRs and issues
- [lazygit](https://github.com/jesseduffield/lazygit) - Git TUI
- [watchgha](https://github.com/nedbat/watchgha) - Real-time GitHub Actions monitoring

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm
- Inspired by [gh-dash](https://github.com/dlvhdr/gh-dash)
- Python Rich CLI reference: [dotfiles PR#5](https://github.com/KyleKing/dotfiles/pull/5)
