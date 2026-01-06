# AGENTS.md - AI Coding Agent Guidelines

This document provides guidelines for AI coding agents working on the Subscription Tracker TUI codebase.

## Project Overview

A terminal-based subscription tracking application built with Go using:
- **Bubbletea** - TUI framework (Elm architecture: Model-Update-View)
- **Lipgloss** - Terminal styling
- **SQLC** - Type-safe SQL code generation
- **SQLite** - Data storage
- **golang-migrate** - Database migrations

## Build/Lint/Test Commands

```bash
# Build
make build              # Build to bin/subscription-tracking-tui
make build-all          # Cross-compile for linux/darwin/windows

# Run
make run                # Run the application directly

# Test
make test               # Run all tests: go test -v ./...
go test -v ./internal/service/             # Run tests in a specific package
go test -v ./internal/service/ -run TestSubscriptionService_Create  # Run single test
go test -v ./... -run TestName             # Run test by name across all packages

# Test with coverage
make test-coverage      # Generate coverage.html report

# Code quality
make fmt                # Format code with gofmt
make vet                # Run go vet ./...
make lint               # Run golangci-lint (requires installation)

# Dependencies
make deps               # Download and tidy dependencies

# SQLC code generation
make sqlc               # Regenerate internal/db/ from db/sqlc/queries.sql

# Clean
make clean              # Remove build artifacts
```

## Project Structure

```
subscription-tracking-tui/
├── main.go                     # Entry point
├── Makefile                    # Build commands
├── sqlc.yaml                   # SQLC configuration
├── db/
│   ├── migrations/             # SQL migration files (embedded)
│   │   └── embed.go            # Embeds .sql files via go:embed
│   └── sqlc/
│       └── queries.sql         # SQL queries -> generates internal/db/
├── internal/
│   ├── app/                    # Application initialization, DI container
│   │   └── app.go
│   ├── db/                     # SQLC generated code (DO NOT EDIT)
│   │   ├── db.go
│   │   ├── models.go
│   │   └── queries.sql.go
│   ├── service/                # Business logic layer
│   │   ├── subscription.go     # CRUD for subscriptions
│   │   ├── spending.go         # Spending calculations
│   │   ├── config.go           # App configuration
│   │   ├── export.go           # CSV/JSON export
│   │   ├── sync.go             # GitHub Gist sync
│   │   ├── crypto.go           # AES-256 encryption
│   │   └── *_test.go           # Tests
│   └── tui/                    # Terminal UI (Bubbletea)
│       ├── model.go            # Main model, Init/Update/View
│       ├── styles.go           # Lipgloss styles
│       └── *.go                # View-specific files
```

## Code Style Guidelines

### Formatting
- Use `gofmt` for all formatting (run `make fmt`)
- No external formatter configuration - standard Go formatting

### Import Organization
Group imports in this order with blank lines between groups:
```go
import (
    // 1. Standard library
    "context"
    "fmt"

    // 2. Third-party packages
    "github.com/charmbracelet/bubbletea"
    _ "github.com/mattn/go-sqlite3"

    // 3. Local packages
    "subscription-tracker/internal/db"
    "subscription-tracker/internal/service"
)
```

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Files | snake_case | `subscription_test.go`, `testutil_test.go` |
| Exported types/functions | PascalCase | `SubscriptionService`, `NewSpendingService` |
| Unexported identifiers | camelCase | `queries`, `configService` |
| Constants | camelCase (unexported) or PascalCase (exported) | `pbkdf2Iterations`, `ViewList` |
| Constructor functions | `New<Type>()` | `NewSubscriptionService(queries)` |
| Input/Output structs | `<Action><Entity>Input/Output` | `CreateSubscriptionInput` |

### Struct Definitions
- Use unexported fields with receiver methods
- Create Input/Output structs for complex operations with validation

```go
type SubscriptionService struct {
    queries *db.Queries  // unexported field
}

type CreateSubscriptionInput struct {
    Name            string
    Amount          float64
    Currency        string
    BillingCycle    string
    NextRenewalDate string
}

func (i *CreateSubscriptionInput) Validate() error {
    if i.Name == "" {
        return fmt.Errorf("name is required")
    }
    // ... more validation
    return nil
}
```

### Error Handling
- Wrap errors with context using `fmt.Errorf` and `%w` verb
- Return errors, don't panic
- Clean up resources on error paths

```go
if err != nil {
    return nil, fmt.Errorf("failed to open database: %w", err)
}

if err := runMigrations(database); err != nil {
    database.Close()  // Clean up before returning
    return nil, fmt.Errorf("failed to run migrations: %w", err)
}
```

### Constants and Enums
- Use `const` blocks for related constants
- Use `iota` for enum-like values

```go
const (
    pbkdf2Iterations = 100000
    saltSize         = 32
)

type View int
const (
    ViewList View = iota
    ViewAdd
    ViewEdit
)
```

## Testing Patterns

### Test File Location
- Tests live alongside source files: `foo.go` -> `foo_test.go`
- Use external test package for black-box testing: `package service_test`

### Table-Driven Tests
```go
func TestSubscriptionService_Create(t *testing.T) {
    tdb := setupTestDB(t)
    ctx := context.Background()

    tests := []struct {
        name    string
        input   service.CreateSubscriptionInput
        wantErr bool
    }{
        {
            name:    "valid monthly subscription",
            input:   service.CreateSubscriptionInput{...},
            wantErr: false,
        },
        // more cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sub, err := tdb.SubscriptionService.Create(ctx, tt.input)
            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            // assertions...
        })
    }
}
```

### Test Helpers
- Use `t.Helper()` in test helper functions
- Use `t.Cleanup()` for resource cleanup
- Use in-memory SQLite for database tests

```go
func setupTestDB(t *testing.T) *testDB {
    t.Helper()
    database, err := sql.Open("sqlite3", ":memory:")
    // ...
    t.Cleanup(func() {
        database.Close()
    })
    return tdb
}
```

## TUI Architecture (Bubbletea)

The TUI follows the Elm architecture pattern:

### Model
Central state container in `internal/tui/model.go`:
```go
type Model struct {
    view          View
    subscriptions []db.Subscription
    // ... state fields
}
```

### Messages
Custom message types for async operations:
```go
type subscriptionsLoadedMsg struct {
    subscriptions []db.Subscription
}
type errMsg struct{ err error }
type successMsg struct{ message string }
```

### Update
Handle messages and return new model + commands:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // handle keys
    case subscriptionsLoadedMsg:
        m.subscriptions = msg.subscriptions
    }
    return m, nil
}
```

### View
Render current state to string:
```go
func (m Model) View() string {
    switch m.view {
    case ViewList:
        return m.viewList()
    // ...
    }
}
```

## SQLC Guidelines

### Adding New Queries
1. Add query to `db/sqlc/queries.sql` with annotation:
   ```sql
   -- name: GetSubscription :one
   SELECT * FROM subscriptions WHERE id = ?;
   
   -- name: ListSubscriptions :many
   SELECT * FROM subscriptions ORDER BY name ASC;
   ```
2. Run `make sqlc` to regenerate `internal/db/`
3. Never edit files in `internal/db/` directly

### Database Migrations
- Add new migration files in `db/migrations/`
- Use sequential numbering: `003_feature_name.up.sql`, `003_feature_name.down.sql`
- Migrations run automatically on startup via `internal/app/app.go`

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/bubbles` | TUI components (textinput, etc.) |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/golang-migrate/migrate/v4` | Database migrations |
| `github.com/mattn/go-sqlite3` | SQLite driver (requires CGO) |
| `golang.org/x/crypto` | PBKDF2 key derivation |
