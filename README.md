# govm — Go Version Manager

> **govm** (pronounced "go-vee-em") lets you install, switch, and manage multiple versions of Go side-by-side — inspired by `nvm` / `fnm` for Node.js, but written in Go for the Go community.

> ⚠️ **Under active development.** APIs and behaviors may change between releases. Feedback and contributions welcome.

[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![CI](https://github.com/abramsz/govm/actions/workflows/ci.yml/badge.svg)](https://github.com/abramsz/govm/actions/workflows/ci.yml)

## Quickstart

```bash
# Install a version
govm install 1.23.4

# Switch to it
eval "$(govm use 1.23.4)"
go version  # → go version go1.23.4 ...

# Set as global default
govm default 1.23.4

# List installed versions
govm list
```

## Features

- **🔀 Multi-version management** — install any Go version side-by-side
- **🖥️ Per-session switching** — `eval "$(govm use 1.23.4)"` or just `govm use 1.23.4` with wrapper
- **🔐 SHA256 verified downloads** — every archive is checked against go.dev's checksum
- **⚡ Version aliases** — `govm config alias stable 1.26.4; govm install stable`
- **🏗️ Cross-architecture install** — `govm install 1.23.4 --arch arm64` on any host
- **🐚 5 shell integrations** — bash, zsh, fish, PowerShell, cmd.exe
- **🔒 Concurrent-safe** — file locking prevents install races
- **📦 Single binary** — zero runtime dependencies
- **🔄 Offline activation** — switching versions works without network
- **🆔 Version injection** — `go build -ldflags="-X 'main.version=v1.0.0'"`

## Commands

| Command | Description |
|---------|-------------|
| `govm install <version>` | Download and install a Go version |
| `govm install --reinstall <version>` | Re-download an already-installed version |
| `govm install --arch <arch> <version>` | Install for a specific architecture |
| `govm list` | Show installed versions (`*` marks default) |
| `govm list-remote` | Show all versions available on go.dev (cached 1h) |
| `govm list-remote --all` | Include unstable / release-candidate versions |
| `govm use <version>` | Activate a version (outputs shell script) |
| `govm default <version>` | Set the global default via symlink |
| `govm current` | Show the currently active default version |
| `govm which` | Show the path to the active Go binary |
| `govm uninstall <version>` | Remove an installed version |
| `govm env` | Print shell initialization script |
| `govm config` | View JSON configuration |
| `govm config set <key> <val>` | Set a config value (mirror, default) |
| `govm config alias <name> <ver>` | Set or remove a version alias |
| `govm version` | Print the govm version |
| `govm manual` | Print full Markdown user manual |
| `govm completion bash/zsh/fish` | Generate shell completion scripts |

## Setup (one-time)

### Bash / Zsh

```bash
# ~/.bashrc or ~/.zshrc
eval "$(govm env --shell bash)"
govm default 1.23.4
```

### Fish

```fish
# ~/.config/fish/conf.d/govm.fish
govm env --shell fish | source
govm default 1.23.4
```

### PowerShell

```powershell
# $profile
govm env --shell powershell | Out-String | Invoke-Expression
govm default 1.23.4
```

### cmd.exe

```batch
:: AutoRun or startup script
for /f "tokens=*" %%i in ('govm env --shell cmd') do call %%i
```

### Post-setup

After setup, `govm use <version>` works directly without `eval`:

```bash
govm use 1.23.4   # ← wrapper function handles eval
govm use stable   # ← aliases work too
```

## Version aliases

```bash
# Define aliases
govm config alias stable 1.26.4
govm config alias latest 1.26.4

# Use them anywhere a version is expected
govm install stable       # → installs 1.26.4
govm use latest           # → activates 1.26.4
govm default stable       # → sets 1.26.4 as default

# Remove an alias
govm config alias stable ""

# View config
govm config
```

## Cross-architecture installation

```bash
# Download ARM64 version from an AMD64 machine
govm install 1.23.4 --arch arm64

# Verify
file ~/.govm/versions/1.23.4/go/bin/go
# → ... ARM64 executable
```

## Shell completions

```bash
# Generate and source
source <(govm completion bash)     # bash
source <(govm completion zsh)      # zsh
govm completion fish | source      # fish
```

Completions dynamically list remote versions (for `install`) and installed versions (for `use`, `default`, `uninstall`).

## Configuration file

`~/.govm/config.json` is auto-managed. You rarely need to edit it by hand:

```json
{
  "default_version": "1.23.4",
  "aliases": {
    "stable": "1.26.4",
    "latest": "1.26.4"
  },
  "mirror": "https://go-mirror.example.com/dl/"
}
```

## How it works

```
~/.govm/                          # GOVM_HOME
  current → versions/1.23.4/go    # Symlink to default GOROOT
  versions/
    1.23.4/
      go/                         # Extracted GOROOT
        bin/go
        ...
    1.22.10/
      go/
  cache/
    versions.json                 # Cached version list (TTL 1h)
  config.json                     # Aliases, mirror, settings
  .lock                           # Advisory lock (concurrency safety)
```

## Building from source

```bash
git clone https://github.com/abramsz/govm
cd govm
go build -o govm .

# With version injection
go build -ldflags="-X 'main.version=v1.0.0'" -o govm .
```

Requires Go 1.22+.

## Test

```bash
go test -short ./...          # fast (skips network tests)
go test -count=1 ./...        # full suite (includes list-remote API calls)
```

**~60 tests across 7 packages**, with coverage in shell (92%), store (61%), download (45%), and config (89%).

## Architecture

| Package | Responsibility |
|---------|---------------|
| `pkg/versions` | Fetch & parse version list from `go.dev/dl/?mode=json` |
| `pkg/download` | Download with progress, SHA256 verify, tar.gz/zip extract |
| `pkg/store` | Local store management, symlink, file locking |
| `pkg/shell` | Shell script generation (5 shells) |
| `pkg/config` | JSON config file, alias resolution |
| `pkg/manual` | Embedded markdown user manual |
| `cmd/` | Cobra CLI commands (13 commands) |

## Roadmap

- [ ] `.go-version` auto-switch (project-local version files)
- [ ] `govm exec <version> <cmd>` — run a command under a specific Go version
- [ ] `govm unuse` — restore system Go
- [ ] Architecture aliases (`--arch arm64`) ✅ *done*
- [ ] Shell completions ✅ *done*
- [ ] Version aliases ✅ *done*
- [ ] `govm manual` — AI-friendly markdown docs ✅ *done*

## License

MIT
