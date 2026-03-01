# ZFSGuard

[Changelog](CHANGELOG.md) · [Releases](https://github.com/pbek/zfsguard/releases)

A beautiful terminal user interface (TUI) for managing ZFS snapshots with a background health monitoring service for ZFS and SMART errors. Sends alerts to dozens of notification services and your local Linux desktop.

![ZFSGuard TUI screenshot](doc/zfsguard.webp)

## Features

### TUI Snapshot Manager (`zfsguard`)

- **List** all ZFS snapshots with dataset, name, used space, referenced size, and creation time
- **Create** snapshots with an interactive form (select dataset, name auto-populated with timestamp, `Esc` to cancel)
- **Delete** selected snapshots with confirmation dialog
- **Bulk select** snapshots with space/x, select all with `a`
- **Delete all** snapshots with a single key (`D`)
- **Filter** snapshots by name with `/` search
- **Refresh** snapshot list after creation or on demand (`r`)
- **Health report** view (`h`) showing ZFS pool states and SMART disk results, sourced from the monitor's JSON output
- **Scrollable** viewport with page up/down support
- **Vim-style** keybindings (j/k navigation)
- Colored output with dataset and snapshot name highlighting
- Privilege escalation handled transparently (run with `sudo` for destructive operations)

### Health Monitor (`zfsguard-monitor`)

- Runs as a **systemd service** checking ZFS pool health and SMART disk health at configurable intervals
- Reports pool state degradation, data errors, and SMART failures
- **Writes a JSON health report** after each check cycle for the TUI to display
- Sends alerts via **[shoutrrr](https://containrrr.dev/shoutrrr/)** supporting 15+ notification services:
  - Discord, Slack, Telegram, Pushover, Gotify, ntfy
  - Email (SMTP), Microsoft Teams, Matrix, Mattermost
  - Pushbullet, Rocket.Chat, Zulip, generic webhooks, and more
- Sends **local Linux desktop notifications** via `notify-send`
- Oneshot mode for cron-based setups (`--oneshot`)

## Installation

### Pre-built binaries

Download from [GitHub Releases](https://github.com/pbek/zfsguard/releases):

```bash
# Download and install
curl -Lo zfsguard https://github.com/pbek/zfsguard/releases/latest/download/zfsguard-linux-amd64
curl -Lo zfsguard-monitor https://github.com/pbek/zfsguard/releases/latest/download/zfsguard-monitor-linux-amd64
chmod +x zfsguard zfsguard-monitor
sudo mv zfsguard zfsguard-monitor /usr/local/bin/
```

### Build from source

Requires Go 1.22+:

```bash
go install github.com/pbek/zfsguard/cmd/zfsguard@latest
go install github.com/pbek/zfsguard/cmd/zfsguard-monitor@latest
```

Or clone and build:

```bash
git clone https://github.com/pbek/zfsguard.git
cd zfsguard
VERSION=$(cat VERSION)
go build -ldflags "-X github.com/pbek/zfsguard/internal/version.Version=$VERSION" ./cmd/zfsguard
go build -ldflags "-X github.com/pbek/zfsguard/internal/version.Version=$VERSION" ./cmd/zfsguard-monitor
```

### NixOS Flake

Add to your `flake.nix`:

```nix
{
  inputs.zfsguard.url = "github:pbek/zfsguard";

  outputs = { self, nixpkgs, zfsguard, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      modules = [
        zfsguard.nixosModules.default
        {
          services.zfsguard = {
            enable = true;
            settings = {
              monitor = {
                interval_minutes = 30;
                check_zfs = true;
                check_smart = true;
                # report_path = "/var/lib/zfsguard/health-report.json"; # default
              };
              notify = {
                desktop = true;
                shoutrrr_urls = [
                  "ntfy://ntfy.sh/my-zfs-alerts"
                  # "discord://token@id"
                  # "telegram://token@telegram?channels=channel"
                ];
              };
            };
          };
        }
      ];
    };
  };
}
```

This sets up:

- The `zfsguard` TUI in your system PATH
- A `zfsguard-monitor.service` systemd unit that monitors ZFS and SMART health
- Automatic dependency on `zfs`, `smartmontools`, and `libnotify`

### Development shell

```bash
nix develop
```

Provides: `go`, `gopls`, `golangci-lint`, `goreleaser`

## Usage

### TUI

```bash
# Run the TUI (read-only operations work without root)
zfsguard

# Run with sudo for snapshot creation/deletion
sudo zfsguard

# Use a specific config file
zfsguard --config /path/to/config.yaml

# Show version
zfsguard --version
```

#### Keybindings

| Key             | Action                 |
| --------------- | ---------------------- |
| `j` / `k`       | Move cursor up/down    |
| `PgUp` / `PgDn` | Page up/down           |
| `Space` / `x`   | Toggle select snapshot |
| `a`             | Select/deselect all    |
| `/`             | Filter by name         |
| `c`             | Create snapshot        |
| `d`             | Delete selected        |
| `D`             | Delete ALL snapshots   |
| `r`             | Refresh snapshot list  |
| `h`             | Open health report     |
| `?`             | Toggle full help       |
| `q`             | Quit                   |

#### Health report view

Press `h` from the snapshot list to open the health report panel. It displays the ZFS pool states and SMART disk results collected by the last monitor run, along with the report timestamp and age.

| Key             | Action                  |
| --------------- | ----------------------- |
| `j` / `k`       | Scroll up/down          |
| `PgUp` / `PgDn` | Page up/down            |
| `r`             | Reload report from disk |
| `h` / `Esc`     | Return to snapshot list |
| `q`             | Quit                    |

The health report is read from `monitor.report_path` in the config (default `/var/lib/zfsguard/health-report.json`). If the file does not exist yet (e.g. the monitor has not run or the path is not configured), the TUI shows a descriptive message rather than an error.

#### Create snapshot dialog

- `Tab` / `Shift+Tab` to cycle through datasets
- Type a snapshot name (auto-filled with current timestamp)
- `Enter` to confirm, `Esc` to cancel

### Monitor

```bash
# Run continuously (default: check every 60 minutes)
sudo zfsguard-monitor

# Run a single check and exit
sudo zfsguard-monitor --oneshot

# Use a specific config file
sudo zfsguard-monitor --config /path/to/config.yaml

# Show version
zfsguard-monitor --version
```

## Configuration

Create a config file at `~/.config/zfsguard/config.yaml` or `/etc/zfsguard/config.yaml`:

```yaml
monitor:
  interval_minutes: 60
  check_zfs: true
  check_smart: true
  # smart_devices:
  #   - /dev/sda
  #   - /dev/sdb
  # report_path: /var/lib/zfsguard/health-report.json

notify:
  shoutrrr_urls:
    - "ntfy://ntfy.sh/my-zfs-alerts"
    # - "discord://token@id"
    # - "telegram://token@telegram?channels=channel"
    # - "gotify://gotify.example.com/token"
    # - "smtp://user:pass@mail.example.com:587/?to=admin@example.com"
  desktop: true

defaults:
  snapshot_prefix: "zfsguard"
```

See [`config.example.yaml`](config.example.yaml) for a fully commented example.

### Notification services

ZFSGuard uses [shoutrrr](https://containrrr.dev/shoutrrr/) for notification integration. See the [shoutrrr documentation](https://containrrr.dev/shoutrrr/services/overview/) for the full list of supported services and URL formats.

Desktop notifications use `notify-send` (from `libnotify`) and work on any Linux desktop environment supporting the freedesktop notification specification.

## Versioning

The application version is defined in the [`VERSION`](VERSION) file at the repository root. This single source of truth is used by:

- **Go binaries** via `-ldflags -X github.com/pbek/zfsguard/internal/version.Version=...`
- **Nix flake** via `builtins.readFile ./VERSION`
- **GoReleaser** via the git tag (tags should match the `VERSION` file content)

To bump the version:

1. Update the `VERSION` file
2. Commit and tag: `git tag v$(cat VERSION)`
3. Push with tags: `git push origin main --tags`

## Elevated privileges

ZFS snapshot operations require root or delegated ZFS permissions:

- **Listing** snapshots works without root
- **Creating** and **deleting** snapshots requires root or ZFS delegation
- The monitor service runs as a systemd service (typically as root) to access both ZFS and SMART data

To delegate ZFS permissions to a user without full root:

```bash
# Allow user 'myuser' to create and destroy snapshots on 'tank'
zfs allow myuser create,destroy,snapshot,mount tank
```

## Project structure

```
zfsguard/
├── cmd/
│   ├── zfsguard/           # TUI binary
│   │   └── main.go
│   └── zfsguard-monitor/   # Monitor service binary
│       └── main.go
├── internal/
│   ├── config/             # Configuration loading (YAML)
│   │   └── config.go
│   ├── monitor/            # Health monitoring service
│   │   └── monitor.go
│   ├── notify/             # Notification dispatching (shoutrrr + desktop)
│   │   └── notify.go
│   ├── report/             # Health report JSON types + read/write
│   │   └── report.go
│   ├── tui/                # Terminal UI (bubbletea + lipgloss)
│   │   ├── model.go
│   │   └── view.go
│   ├── version/            # Build version info (set via ldflags)
│   │   └── version.go
│   └── zfs/                # ZFS and SMART CLI wrappers
│       ├── zfs.go
│       └── smart.go
├── VERSION                 # Single source of truth for app version
├── flake.nix               # Nix flake with package + NixOS module
├── .goreleaser.yml         # GoReleaser config for release builds
├── config.example.yaml     # Example configuration
├── .github/workflows/
│   ├── ci.yml              # CI: test + lint on push/PR
│   └── release.yml         # Release: GoReleaser on v* tags
├── LICENSE                 # GPL-3.0
├── README.md
└── AGENTS.md
```

## CI/CD

### CI workflow (`.github/workflows/ci.yml`)

Runs on push to `main`/`release` and on pull requests:

- `go vet`
- `go test -v -race ./...`
- `golangci-lint`
- Build with version from `VERSION` file

### Release workflow (`.github/workflows/release.yml`)

Runs on `v*` tags:

- Uses [GoReleaser](https://goreleaser.com/) to build release binaries for `linux/amd64` and `linux/arm64`
- Packages both `zfsguard` and `zfsguard-monitor` into archives with LICENSE, README, and example config
- Creates a GitHub Release with changelog and checksums

To create a release:

```bash
# Update VERSION file, then:
git add VERSION
git commit -m "release: v0.2.0"
git tag v0.2.0
git push origin main --tags
```

## License

[GPL-3.0-or-later](LICENSE)
