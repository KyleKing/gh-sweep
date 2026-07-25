# AI Agent Guidelines for gh-sweep

Guidelines for AI coding assistants working on this Go project. Project-specific architecture, patterns, and domain context live in [DESIGN.md](DESIGN.md).

## Package Structure

```
gh-sweep/

├── cmd/gh-sweep/     # CLI entry point

├── internal/         # Private packages (not importable by other modules)
│   ├── cli/          # Cobra commands
│   ├── tui/          # Bubble Tea app and components
│   └── ...

└── go.mod
```

- One package, one purpose; short lowercase names with no underscores (`httputil`, not `http_util`)
- Avoid grab-bag packages (`util`, `common`, `misc`)
- `internal/` prevents external imports at the compiler level
- Group related types and their methods in one file named after the primary type (`user.go`, `user_test.go`); keep `main.go` thin

## Code Style

- Favor small, composable, single-responsibility functions and composition over inheritance
- Define interfaces where they are consumed, not where implemented, and keep them to 1-3 methods
- Accept a `context.Context` as the first argument for cancellable or I/O-bound work
- Name with MixedCaps and keep acronyms uppercase (`ServeHTTP`, `userID`, `GetHTTPClient`)
- Use the functional-options pattern for constructors with optional configuration:

```go
type Option func(*Server)

func WithTimeout(d time.Duration) Option {
    return func(s *Server) { s.timeout = d }
}

func NewServer(addr string, opts ...Option) *Server {
    s := &Server{addr: addr, timeout: 30 * time.Second}
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

## Error Handling

- Return errors rather than panicking outside truly unrecoverable states
- Wrap with context: `fmt.Errorf("doing something: %w", err)`
- Inspect with `errors.Is` / `errors.As`; define custom types for domain-specific errors
- Validate at system boundaries and trust internal code (parse, don't validate)

## Comments and Documentation

- Code should be self-explanatory; do not comment what the code plainly does
- Doc-comment exported symbols, describing non-obvious behavior and invariants rather than restating types
- Skip docstrings on self-explanatory private helpers

## Testing

- Prefer table-driven tests with subtests via `t.Run`
- Use the `_test` package suffix for black-box tests and place tests next to the code they cover

```go
tests := []struct {
    name     string
    a, b     int
    expected int
}{
    {"positive", 2, 3, 5},
    {"negative", -1, -1, -2},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        if got := Add(tt.a, tt.b); got != tt.expected {
            t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.expected)
        }
    })
}
```

## Anti-Patterns to Avoid

- Naked returns, functions over ~50 lines, and deep nesting (prefer early returns)
- Interface pollution (define interfaces only once a consumer needs them)
- Ignored errors (`_ = doThing()` is almost always wrong)
- Shared global state; pass dependencies explicitly

## Workflow

- Run `mise run ci` (tests + build) before committing; `mise run format` or `hk fix` auto-fixes lint and formatting
- Conventional commits are enforced by commitizen
- Do not stage, commit, or push without explicit instruction
