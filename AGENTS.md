# AGENTS.md - Guidance for AI coding agents

This file provides context and instructions for AI agents working on the ZFSGuard codebase.

## Project overview

ZFSGuard is a Go project with two binaries:

1. **`zfsguard`** - A TUI (terminal user interface) for managing ZFS snapshots (list, create, delete, bulk delete)
2. **`zfsguard-monitor`** - A background service that monitors ZFS pool and SMART disk health, sending alerts via notification services

## Tech stack

- **Language**: Go (1.22+)
- **TUI framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-architecture TUI framework)
- **TUI styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss) (terminal CSS)
- **TUI components**: [Bubbles](https://github.com/charmbracelet/bubbles) (text input, help, key bindings)
- **Notifications**: [Shoutrrr](https://github.com/containrrr/shoutrrr) (15+ notification services)
- **Desktop notifications**: `notify-send` via `os/exec`
- **Config**: YAML via `gopkg.in/yaml.v3`
- **Packaging**: Nix flake with NixOS module
- **Releases**: [GoReleaser](https://goreleaser.com/)
- **CI**: GitHub Actions

## Architecture

```
cmd/
  zfsguard/           → TUI entry point
  zfsguard-monitor/   → Monitor service entry point

internal/
  zfs/                → ZFS and SMART CLI wrappers (exec zfs/zpool/smartctl)
  tui/                → Bubble Tea model + view (Elm architecture)
  monitor/            → Health check loop with notification dispatch
  notify/             → Notification abstraction (shoutrrr + desktop)
  config/             → YAML config loading with defaults
  version/            → Build version info (set via ldflags)
```

### Key patterns

- **Elm architecture** in the TUI: `Model` holds state, `Update` handles messages, `View` renders. All side effects return as `tea.Cmd` functions.
- **ZFS interaction** is via `os/exec` calling `zfs`, `zpool`, and `smartctl` CLI tools. There is no library binding - this is intentional for portability and simplicity.
- **Notification dispatch** goes through the `Notifier` type which fans out to all configured shoutrrr URLs + optional desktop notification.
- **Version management**: The `VERSION` file at the repo root is the single source of truth. It is read by `flake.nix` (via `builtins.readFile`) and injected into Go binaries at build time via `-ldflags -X github.com/pbek/zfsguard/internal/version.Version=...`. GoReleaser uses the git tag.

## Building

```bash
VERSION=$(cat VERSION)
go build -ldflags "-X github.com/pbek/zfsguard/internal/version.Version=$VERSION" ./cmd/zfsguard
go build -ldflags "-X github.com/pbek/zfsguard/internal/version.Version=$VERSION" ./cmd/zfsguard-monitor
go test ./...
go vet ./...
```

Or with Nix:

```bash
nix build
nix develop  # enter dev shell with go, gopls, golangci-lint, goreleaser
```

## Testing

- Unit tests can be added to any `_test.go` file in the respective package
- The `internal/zfs` package wraps CLI tools, so tests there should mock `exec.Command` or use build tags to skip on systems without ZFS
- The TUI can be tested using Bubble Tea's test utilities
- Run: `go test -v -race ./...`

## Common tasks

### Adding a new notification service

Shoutrrr handles this. Just add the URL to the config. If you need a custom notification channel not supported by shoutrrr, add it to `internal/notify/notify.go` in the `Send` method.

### Adding a new health check

1. Add the check function to `internal/zfs/` (or a new file in `internal/`)
2. Call it from `internal/monitor/monitor.go` in `RunOnce()`
3. Add the config option to `internal/config/config.go`
4. Update the NixOS module options in `flake.nix`

### Modifying the TUI

The TUI follows Bubble Tea's Elm architecture:

- `internal/tui/model.go` - State, messages, `Update` (input handling)
- `internal/tui/view.go` - `View` (rendering with Lip Gloss styles)

To add a new view:

1. Add a `view` constant in `model.go`
2. Add the view rendering in `view.go`
3. Handle transitions in `handleKey` in `model.go`

### Updating the NixOS module

The NixOS module is in `flake.nix` under `nixosModules.default`. It:

- Defines `services.zfsguard` options
- Creates a systemd service running `zfsguard-monitor`
- Generates a YAML config file from the NixOS options
- Adds ZFS, smartmontools, and libnotify to the service PATH

When adding new config options, update both `internal/config/config.go` and the NixOS module options.

### Bumping the version

1. Edit the `VERSION` file with the new version number (e.g. `0.2.0`)
2. The flake.nix reads from `VERSION` automatically
3. GoReleaser gets the version from the git tag
4. CI injects the version via ldflags when building

## Code style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `golangci-lint` for linting
- Keep packages small and focused
- Prefer returning errors over panicking
- Use `fmt.Errorf` with `%w` for error wrapping
- Internal packages go in `internal/` to prevent external imports

## Security notes

- The monitor service needs root for `zpool status` and `smartctl`
- The TUI needs root for `zfs snapshot` and `zfs destroy`
- `zfs list` works without root
- Notification URLs may contain secrets - never log full URLs (see `maskURL` in `notify.go`)
- The NixOS module applies systemd hardening (ProtectHome, ProtectSystem, PrivateTmp)

## Release process

1. Ensure all tests pass on `main`
2. Update the `VERSION` file
3. Commit: `git commit -am "release: v0.2.0"`
4. Tag with semver: `git tag v0.2.0`
5. Push: `git push origin main --tags`
6. GitHub Actions runs GoReleaser to build binaries and create a GitHub Release
7. Update `vendorHash` in `flake.nix` if dependencies changed
