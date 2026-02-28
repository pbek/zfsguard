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

#### Health Monitor (`zfsguard-monitor`)

- Runs as a systemd service checking ZFS pool health and SMART disk health at configurable intervals
- Reports pool state degradation, data errors, and SMART failures
- Sends alerts via shoutrrr supporting 15+ notification services
- Supports Discord, Slack, Telegram, Pushover, Gotify, ntfy, and more
- Sends local Linux desktop notifications via `notify-send`
- Oneshot mode for cron-based setups (`--oneshot`)
