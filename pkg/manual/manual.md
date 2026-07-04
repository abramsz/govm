# govm — Go Version Manager

> **Version:** {{VERSION}}
> **Source:** github.com/abramsz/govm
> **Language:** Go
> **Platforms:** Linux, macOS, Windows

govm is a CLI tool for installing, switching, and managing multiple versions of the Go programming language on a single machine. It is inspired by nvm (Node Version Manager) and fnm (Fast Node Manager) but purpose-built for the Go ecosystem.

## Table of Contents

1. [Quickstart](#quickstart)
2. [How It Works](#how-it-works)
3. [Commands](#commands)
   - [install](#govm-install-version)
   - [list](#govm-list)
   - [list-remote](#govm-list-remote)
   - [use](#govm-use-version)
   - [default](#govm-default-version)
   - [current](#govm-current)
   - [which](#govm-which)
   - [uninstall](#govm-uninstall-version)
   - [env](#govm-env)
   - [config / config alias](#govm-config)
   - [version](#govm-version)
   - [manual](#govm-manual)
4. [Version Aliases](#version-aliases)
5. [Shell Integration](#shell-integration)
6. [Directory Layout](#directory-layout)
7. [Environment Variables](#environment-variables)
8. [Architecture](#architecture)
9. [How govm use Works (The Eval Pattern)](#how-govm-use-works-the-eval-pattern)
10. [Command-Line Completion](#command-line-completion)
11. [FAQ / Troubleshooting](#faq--troubleshooting)

---

## Quickstart

```bash
# 1. Install a Go version
govm install 1.23.4

# 2. Switch to it (current shell)
eval "$(govm use 1.23.4)"

# 3. Verify
go version  # → go version go1.23.4 ...

# 4. Set as global default (new shells)
govm default 1.23.4

# 5. List installed versions
govm list
```

---

## How It Works

govm downloads official Go distribution archives from `go.dev/dl/` and extracts them into a local directory (`~/.govm/versions/<version>/go/`). When you activate a version, govm:

1. **Sets `GOROOT`** to the extracted Go root
2. **Prepends `$GOROOT/bin` to `PATH`**
3. **Exports `GOVM_VERSION`** for shell prompt integration

The activated Go installation is self-contained — no system files are modified.

The key design goal is **zero state mutation outside `~/.govm/`**. No `sudo`, no `apt`, no system-wide symlinks. Everything lives under one directory.

---

## Commands

### govm install \<version\>

Download and install a Go version from go.dev. When no default is set, the first install is automatically set as the global default.

**Usage:** `govm install <version>`

**Flags:**

| Flag | Description |
|------|-------------|
| `--reinstall` | Re-download and re-extract even if already installed. |
| `-a, --arch string` | Target architecture: amd64, arm64, 386, etc. (default: host arch). |

**Examples:**

```bash
govm install 1.23.4
govm install 1.22.10
govm install --reinstall 1.23.4
govm install 1.23.4 --arch arm64
```

**Notes:**

On first install (when no default is set), the new version is automatically set as the global default. Use `--reinstall` to overwrite an existing installation. Use `--arch` to download a version for a different architecture than the current host (e.g., download ARM64 binaries from an AMD64 machine).

---

### govm list

List all locally installed Go versions, sorted newest-first using semantic versioning. The currently active default version is marked with `*`.

**Examples:**

```bash
govm list
```

Output:

```
 * 1.25.11
   1.23.4
   1.22.10
```

If no versions are installed, a suggestion to run `govm list-remote` is shown.

---

### govm list-remote

List all Go versions available for download from go.dev. Only stable releases are shown by default.

**Flags:**

| Flag | Description |
|------|-------------|
| `-a, --all` | Include unstable, beta, and release-candidate versions. |

**Examples:**

```bash
govm list-remote
govm list-remote --all
```

**Notes:**

Results are cached locally for 1 hour in `~/.govm/cache/versions.json`. Subsequent calls within the TTL are instant.

---

### govm use \<version\>

Activate a Go version in the current shell session. This command **outputs a shell script** that must be evaluated by the calling shell — see [The Eval Pattern](#how-govm-use-works-the-eval-pattern).

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --shell string` | Output syntax: bash, zsh, fish, powershell, cmd (auto-detect if empty). |

**Examples:**

```bash
eval "$(govm use 1.23.4)"
govm use --shell bash 1.23.4
govm use --shell powershell 1.23.4
```

**Output by shell:**

| Shell | Syntax |
|-------|--------|
| bash/zsh | `export GOROOT='...'; export PATH='...'` |
| fish | `set -gx GOROOT ...; set -gx PATH ...` |
| PowerShell | `$env:GOROOT = '...'; $env:Path = '...'` |
| cmd.exe | `set GOROOT=...; set PATH=...` |

If the wrapper function from `govm env` is installed, `govm use` works directly without manual eval.

---

### govm default \<version\>

Set the global default Go version. This updates the symlink at `~/.govm/current` to point to the specified version's GOROOT. All new shell sessions that have `~/.govm/current/bin` on `PATH` will use this version.

```bash
govm default 1.23.4
```

Requires the version to be already installed.

---

### govm current

Print the version number of the currently active default Go installation.

```bash
govm current
# → 1.23.4
```

Errors if no default version has been set.

---

### govm which

Print the full file path to the currently active Go binary.

```bash
govm which
# → /home/user/.govm/versions/1.23.4/go/bin/go
```

On Windows the path ends with `go.exe`. Errors if no default version has been set.

---

### govm uninstall \<version\>

Remove an installed Go version from the local store. Refuses to uninstall the currently active default version — switch to another version first.

```bash
govm uninstall 1.22.10
```

---

### govm env

Print a shell initialization script that sets up `PATH` and a `govm` wrapper function. This is the recommended way to set up govm in your shell config.

**Flags:**

| Flag | Description |
|------|-------------|
| `-s, --shell string` | Target shell: bash, zsh, fish, powershell, cmd (auto-detect). |
| `--use-on-cd` | Reserved for future .go-version auto-switching. |

**Examples:**

```bash
eval "$(govm env --shell bash)"
```

The generated script:

1. Adds the govm binary directory to `PATH`
2. Adds `~/.govm/current/bin` to `PATH`
3. Defines a `govm()` shell function that intercepts `govm use` and automatically wraps it in `eval`
4. Hardcodes the govm binary path so it works even when govm is not on `PATH` yet

---

### govm config

View or modify the global govm configuration file (`~/.govm/config.json`).

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `govm config` | Print the full configuration as JSON. |
| `govm config set <key> <value>` | Set a config value (`mirror`, `default`). |
| `govm config alias <name> <version>` | Set or remove a version alias. |

**Examples:**

```bash
govm config set mirror https://go-mirror.example.com/dl/
govm config alias stable 1.26.4
govm config alias latest 1.26.4
govm config alias stable ""   # remove alias
govm config
```

---

### govm version

Print the version number of govm. The version can be set at build time via `-ldflags`.

```bash
govm version       # → 0.0.0-dev (or v1.0.0 for releases)
govm --version     # → govm version 0.0.0-dev
```

Build with a specific version:

```bash
go build -ldflags="-X 'main.version=v1.0.0'" -o govm .
```

---

### govm manual

Print a comprehensive Markdown user manual to stdout. Designed for both human readers and large language models (LLMs).

```bash
govm manual > docs/manual.md   # save to file
govm manual | head              # read in terminal
```

---

## Version Aliases

govm supports human-friendly aliases for Go versions. Aliases let you refer to versions by name instead of number.

**Examples:**

```bash
# Define aliases
govm config alias stable 1.26.4
govm config alias latest 1.26.4

# Use aliases anywhere a version is expected
govm install stable       # installs 1.26.4
govm use latest           # activates 1.26.4
govm default stable       # sets 1.26.4 as global default
govm uninstall old-version

# Aliases support chaining: "latest" → "stable" → "1.26.4"
govm config alias latest stable

# Remove an alias
govm config alias stable ""

# List all aliases
govm config
```

**Notes:**

- Aliases are stored in `~/.govm/config.json` under the `aliases` key.
- Alias resolution is recursive with cycle detection (max 10 hops).
- The `resolveVersion()` function is applied to the version argument of `install`, `use`, `default`, and `uninstall`.

---

## Shell Integration

Add one line to your shell configuration file.

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
# Add to $profile
govm env --shell powershell | Out-String | Invoke-Expression
govm default 1.23.4
```

### cmd.exe

```batch
:: Add to AutoRun or create a startup script
for /f "tokens=*" %%i in ('govm env --shell cmd') do call %%i
```

### Post-setup

After setup, `govm use <version>` works directly without `eval`:

```bash
govm use 1.23.4  # wrapper function handles eval automatically
```

---

## Directory Layout

```
~/.govm/                        # Govm home ($GOVM_HOME)
  current → versions/1.23.4/go  # Symlink to active GOROOT
  versions/
    1.23.4/
      go/                       # Extracted GOROOT
        bin/go
        ...
    1.22.10/
      go/
  cache/
    versions.json               # Cached version list from go.dev
  config.json                   # Persistent configuration (aliases, mirror)
  .lock                         # Advisory lock file
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOVM_HOME` | `~/.govm` | Override the data directory. Useful for CI, testing, or multi-user setups. |
| `GOVM_VERSION` | _(set by `govm use`)_ | The currently activated Go version. |
| `GOROOT` | _(set by `govm use`)_ | The Go root of the activated version. |

---

## Architecture

govm is organized into four internal packages:

| Package | Responsibility |
|---------|---------------|
| `pkg/versions` | Fetch and parse the Go version list from `go.dev/dl/?mode=json`. Provides the `Release` and `File` types. |
| `pkg/download` | Find the matching archive for the current OS/arch, download it with progress indication, verify SHA256, and extract tar.gz or zip archives. |
| `pkg/store` | Manage the local file system store (`~/.govm/`): install, list, uninstall versions; manage the `current` symlink; file locking for concurrency safety. |
| `pkg/shell` | Generate shell scripts for version activation (`export` / `$env:` / `set`) and shell initialization (`eval "$(govm env)"`). Supports bash, zsh, fish, PowerShell, and cmd.exe. |
| `pkg/config` | Load, save, and manage the `~/.govm/config.json` configuration file including version aliases. |

### CLI Layer

The `cmd/` package contains Cobra commands that wire the packages together:

| File | Command |
|------|---------|
| `root.go` | `govm` — root command, sets up cache directory |
| `install.go` | `govm install [--reinstall] [--arch]` |
| `list.go` | `govm list` |
| `list_remote.go` | `govm list-remote [--all]` |
| `use.go` | `govm use [--shell]` |
| `set_default.go` | `govm default` |
| `uninstall.go` | `govm uninstall` |
| `current.go` | `govm current` |
| `which.go` | `govm which` |
| `env.go` | `govm env [--shell]` |
| `config.go` | `govm config set/alias` |
| `version.go` | `govm version` |
| `manual.go` | `govm manual` |
| `term.go` | `isTerminal()` helper |
| `completion.go` | Shell completion functions (`ValidArgsFunction`) |

---

## How govm use Works (The Eval Pattern)

A fundamental Unix process model constraint: **a child process cannot modify the environment of its parent shell**. This means `govm use` cannot directly change the calling shell's `PATH` or `GOROOT`.

The solution is the **eval pattern**:

```bash
# govm outputs shell commands to stdout; the parent shell evaluates them
eval "$(govm use 1.23.4)"
```

When the wrapper function from `govm env` is installed, this happens automatically:

```bash
# The shell function intercepts `govm use` and wraps it in eval
govm() {
  case "$1" in
    use)
      eval "$("$_govm_exe" use --shell bash "$2")"
      ;;
    *)
      "$_govm_exe" "$@"
      ;;
  esac
}
```

Each shell has its own eval mechanism:

| Shell | Pattern |
|-------|---------|
| bash/zsh | `eval "$(govm use ...)"` |
| fish | `govm use ... | source` |
| PowerShell | `govm use ... | Out-String | Invoke-Expression` |
| cmd.exe | `for /f "tokens=*" %i in ('govm use ...') do call %i` |

---

## Command-Line Completion

govm includes built-in shell completion via Cobra:

```bash
# Generate and source completion scripts
source <(govm completion bash)     # bash
source <(govm completion zsh)      # zsh
govm completion fish | source      # fish
```

Dynamic completions (triggered by pressing TAB):

| Command | Completes with |
|---------|---------------|
| `govm install <TAB>` | Remote versions from go.dev |
| `govm use <TAB>` | Locally installed versions |
| `govm default <TAB>` | Locally installed versions |
| `govm uninstall <TAB>` | Locally installed versions |

---

## FAQ / Troubleshooting

### Q: `govm use` does nothing or shows 'not installed'

Run `govm install <version>` first. Then use `govm use` with the wrapper function from `govm env`, or wrap it manually: `eval "$(govm use <version>)"`.

### Q: New shell doesn't see the default version

Make sure `~/.govm/current/bin` is on `PATH`. Add to your shell config:

```bash
export PATH="$HOME/.govm/current/bin:$PATH"
```

Or better: use `eval "$(govm env)"` which does this automatically.

### Q: 'Cannot uninstall the currently active default version'

This is a safety guard. Switch to another version first:

```bash
govm default 1.22.10  # switch away
govm uninstall 1.23.4 # now safe to remove
```

### Q: `govm list-remote` is slow

The first call fetches from go.dev. Results are cached for 1 hour in `~/.govm/cache/versions.json`. Subsequent calls within the TTL are instant.

### Q: Does govm work offline?

Listing remote versions requires the network. Activating an already-installed version (`govm use`, `govm default`) works fully offline.

### Q: Can I change the data directory?

Set the `GOVM_HOME` environment variable:

```bash
export GOVM_HOME=/custom/path
govm install 1.23.4
```

### Q: How is this different from `go install golang.org/dl/go1.23.4@latest`?

The `golang.org/dl` tool downloads and runs specific Go versions, but cannot switch between them globally. govm provides a complete version manager with global defaults, shell integration, and per-session switching.

### Q: Downgrade or side-grade?

govm fully supports installing any available version — older or newer. All versions are independent:

```bash
govm install 1.22.10  # older release
govm install 1.25.11  # newer release
govm use 1.22.10       # switch freely
```

---

*Generated by `govm manual`. For the latest version, visit the project repository.*
