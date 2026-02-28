// Package report provides the shared health report format used by the monitor
// service (writer) and the TUI (reader). The monitor writes a JSON report to
// a well-known path after each health check cycle. The TUI reads it to display
// health data without needing elevated privileges.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pbek/zfsguard/internal/zfs"
)

// DefaultPath is the default location for the health report file.
const DefaultPath = "/var/lib/zfsguard/health-report.json"

// HealthReport is the top-level structure written to disk as JSON.
type HealthReport struct {
	Timestamp time.Time    `json:"timestamp"`
	Pools     []PoolReport `json:"pools"`
	Disks     []DiskReport `json:"disks"`
	PoolError string       `json:"pool_error,omitempty"`
	DiskError string       `json:"disk_error,omitempty"`
}

// PoolReport mirrors zfs.PoolStatus with JSON tags.
type PoolReport struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Errors string `json:"errors"`
	Raw    string `json:"raw"`
}

// DiskReport mirrors zfs.SMARTStatus with JSON tags.
type DiskReport struct {
	Device  string `json:"device"`
	Healthy bool   `json:"healthy"`
	Summary string `json:"summary"`
	Raw     string `json:"raw"`
}

// FromChecks builds a HealthReport from raw ZFS and SMART check results.
func FromChecks(
	pools []zfs.PoolStatus,
	poolErr error,
	disks []zfs.SMARTStatus,
	diskErr error,
) HealthReport {
	r := HealthReport{
		Timestamp: time.Now(),
	}

	if poolErr != nil {
		r.PoolError = poolErr.Error()
	}
	for _, p := range pools {
		r.Pools = append(r.Pools, PoolReport{
			Name:   p.Name,
			State:  p.State,
			Errors: p.Errors,
			Raw:    p.Raw,
		})
	}

	if diskErr != nil {
		r.DiskError = diskErr.Error()
	}
	for _, d := range disks {
		r.Disks = append(r.Disks, DiskReport{
			Device:  d.Device,
			Healthy: d.Healthy,
			Summary: d.Summary,
			Raw:     d.Raw,
		})
	}

	return r
}

// Write atomically writes the report as JSON to the given path.
// It writes to a temporary file first and renames to avoid partial reads.
func Write(path string, r HealthReport) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal health report: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory %s: %w", dir, err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write health report: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename health report: %w", err)
	}

	return nil
}

// Read loads a HealthReport from the given path.
func Read(path string) (HealthReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return HealthReport{}, fmt.Errorf("failed to read health report: %w", err)
	}

	var r HealthReport
	if err := json.Unmarshal(data, &r); err != nil {
		return HealthReport{}, fmt.Errorf("failed to parse health report: %w", err)
	}

	return r, nil
}
