# Changelog

All notable features are listed below. For full usage and configuration details, see [README.md](README.md).

## [0.1.0] - 2026-02-28

### Added

#### TUI Snapshot Manager (`zfsguard`)

- List all ZFS snapshots with dataset, name, used space, referenced size, and creation time
- Create snapshots with an interactive form (dataset selection, timestamp name, Esc to cancel)
- Delete selected snapshots with confirmation dialog
- Bulk select snapshots with space/x, select all with `a`
- Delete all snapshots with a single key (`D`)
- Filter snapshots by name with `/` search
- Refresh snapshot list after creation or on demand (`r`)
- Scrollable viewport with page up/down support
- Vim-style keybindings (j/k navigation)
- Colored output with dataset and snapshot name highlighting
- Privilege escalation handled transparently (run with `sudo` for destructive operations)
- **Health report view** (`h` key): Press `h` from the snapshot list to open an in-TUI health report panel showing ZFS pool states and SMART disk health, read from the JSON file written by `zfsguard-monitor`
- Scroll health report with `j`/`k`/`PgUp`/`PgDn`; press `r` to reload; press `h` or `Esc` to return to the snapshot list
- Report shows timestamp and age so you know how fresh the data is
- Gracefully handles missing report file (monitor not yet run, or report path not configured)
- `--config` flag to specify a config file path (same resolution logic as `zfsguard-monitor`)

#### Health Monitor (`zfsguard-monitor`)

- Runs as a systemd service checking ZFS pool health and SMART disk health at configurable intervals
- Reports pool state degradation, data errors, and SMART failures
- Sends alerts via shoutrrr supporting 15+ notification services
- Supports Discord, Slack, Telegram, Pushover, Gotify, ntfy, and more
- Sends local Linux desktop notifications via `notify-send`
- Oneshot mode for cron-based setups (`--oneshot`)
- **JSON health report**: After each check cycle the monitor writes a structured JSON report to `report_path` (default `/var/lib/zfsguard/health-report.json`) capturing pool status, SMART results, error counts, and a UTC timestamp
- Atomic write via temp file + rename to prevent partial reads by the TUI
- `report_path` config option under `monitor` to customise the file location

#### NixOS Module

- `services.zfsguard.settings.monitor.report_path` option (default `/var/lib/zfsguard/health-report.json`)
- `StateDirectory = "zfsguard"` on the systemd service — automatically creates and owns `/var/lib/zfsguard`
- `/var/lib/zfsguard` added to `ReadWritePaths` so the strict `ProtectSystem = "strict"` hardening still allows writing the report
