# Subscription Tracker TUI

A terminal-based application for tracking personal subscriptions and managing monthly spending. Built with Go using [Bubbletea](https://github.com/charmbracelet/bubbletea) for the TUI, [SQLC](https://sqlc.dev/) for type-safe SQL, and SQLite for storage.

## Features

- **Subscription Management** - Add, edit, and delete subscriptions with monthly or yearly billing cycles
- **Renewal Date Tracking** - Track when each subscription renews; auto-advances dates when they pass
- **Spending Summary** - View monthly spending with configurable billing periods based on your payday
- **Remaining Budget** - Set your monthly salary to see how much money remains after subscriptions
- **Export** - Export your data to CSV or JSON
- **Encrypted Cloud Sync** - Sync across devices using GitHub Gist with AES-256 encryption

## Installation

### Prerequisites

- Go 1.21 or later
- GCC (for SQLite)

### Build from Source

```bash
git clone https://github.com/yourusername/subscription-tracking-tui.git
cd subscription-tracking-tui
go build -o subscription-tracker .
```

### Run

```bash
./subscription-tracker
```

Data is stored in `~/.local/share/subscription-tracker/subscriptions.db`

## Usage

### Keyboard Shortcuts

#### Main List View

| Key | Action |
|-----|--------|
| `↑/k` | Move cursor up |
| `↓/j` | Move cursor down |
| `a` | Add new subscription |
| `e` | Edit selected subscription |
| `d` | Delete selected subscription |
| `s` | View spending summary |
| `x` | Export subscriptions |
| `c` | Configuration (payday, salary) |
| `y` | Sync to GitHub Gist |
| `r` | Refresh list |
| `?` | Show help |
| `q` | Quit |

#### Add/Edit Form

| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `←/→` | Toggle billing cycle |
| `Ctrl+S` | Save |
| `Esc` | Cancel |

#### Spending View

| Key | Action |
|-----|--------|
| `←/→` | Change month |
| `Esc` | Back to list |

#### Sync View

| Key | Action |
|-----|--------|
| `Ctrl+P` | Push to GitHub Gist |
| `Ctrl+L` | Pull from GitHub Gist |
| `Esc` | Cancel |

## Configuration

Press `c` from the main list to configure:

- **Payday (1-28)** - The day of the month you get paid. This determines when your billing period starts. For example, if you get paid on the 22nd, setting this to 22 means your "January" spending covers Dec 22 - Jan 21.

- **Monthly Salary** - Your monthly income. Used to calculate remaining money after subscriptions in the spending summary.

## Spending Summary

The spending summary shows:

- **Date Range** - The exact dates covered by the billing period
- **Monthly Subscriptions** - All monthly subscriptions that renew during this period
- **Yearly Subscriptions** - Only yearly subscriptions with renewal dates in this period
- **Total** - Combined spending for the period
- **Remaining** - Your salary minus total subscriptions (if salary is configured)

## Encrypted Cloud Sync

Sync your subscription data across multiple computers using GitHub Gist with end-to-end encryption.

### How It Works

1. Your data is encrypted locally using AES-256-GCM before leaving your machine
2. The encrypted data is uploaded to a private GitHub Gist
3. On another computer, you pull the gist and decrypt with your password
4. GitHub only ever sees encrypted data

### Setup

1. Create a GitHub Personal Access Token:
   - Go to https://github.com/settings/tokens
   - Generate a new token with the `gist` scope
   
2. In the app, press `y` to open the sync view

3. Enter:
   - **Password** - Choose a strong password (use the same on all devices)
   - **GitHub Token** - Your personal access token
   - **Gist ID** - Leave empty for first push, or enter existing ID to sync

4. Press `Ctrl+P` to push or `Ctrl+L` to pull

### Security

- **AES-256-GCM** encryption (military-grade)
- **PBKDF2** key derivation with 100,000 iterations
- **Random salt and nonce** for each encryption
- Your password never leaves your machine
- GitHub only stores encrypted, unreadable data

## Project Structure

```
subscription-tracking-tui/
├── main.go                 # Entry point
├── sqlc.yaml              # SQLC configuration
├── db/
│   ├── migrations/        # SQL migration files
│   └── sqlc/
│       └── queries.sql    # SQL queries for SQLC
├── internal/
│   ├── app/               # Application initialization
│   ├── db/                # SQLC generated code
│   ├── service/           # Business logic
│   │   ├── subscription.go
│   │   ├── spending.go
│   │   ├── config.go
│   │   ├── export.go
│   │   ├── sync.go
│   │   └── crypto.go
│   └── tui/               # Terminal UI
│       ├── model.go
│       ├── list.go
│       ├── add.go
│       ├── edit.go
│       ├── spending.go
│       ├── config.go
│       ├── sync.go
│       └── styles.go
```

## Development

### Run Tests

```bash
go test ./...
```

### Regenerate SQLC

```bash
sqlc generate
```

### Run Migrations

Migrations run automatically on startup using [golang-migrate](https://github.com/golang-migrate/migrate).

## License

MIT
