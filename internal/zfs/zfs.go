// Package zfs provides functions to interact with ZFS via CLI commands.
package zfs

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Snapshot represents a ZFS snapshot.
type Snapshot struct {
	Name      string
	Dataset   string
	ShortName string
	Used      string
	Refer     string
	Creation  time.Time
	Selected  bool
}

// PoolStatus holds the health status of a ZFS pool.
type PoolStatus struct {
	Name   string
	State  string
	Errors string
	Raw    string
}

// ListSnapshots returns all ZFS snapshots on the system.
func ListSnapshots() ([]Snapshot, error) {
	cmd := exec.Command(
		"zfs",
		"list",
		"-t",
		"snapshot",
		"-H",
		"-o",
		"name,used,refer,creation",
		"-s",
		"creation",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	return parseSnapshots(string(out))
}

// ListDatasets returns all ZFS datasets on the system.
func ListDatasets() ([]string, error) {
	cmd := exec.Command("zfs", "list", "-H", "-o", "name")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets: %w", err)
	}

	var datasets []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			datasets = append(datasets, line)
		}
	}
	return datasets, nil
}

// CreateSnapshot creates a new ZFS snapshot with the given name.
// name should be in the format "dataset@snapname".
func CreateSnapshot(name string) error {
	cmd := exec.Command("zfs", "snapshot", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create snapshot %q: %s: %w", name, string(out), err)
	}
	return nil
}

// DestroySnapshot destroys a ZFS snapshot.
func DestroySnapshot(name string) error {
	cmd := exec.Command("zfs", "destroy", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to destroy snapshot %q: %s: %w", name, string(out), err)
	}
	return nil
}

// DestroySnapshots destroys multiple ZFS snapshots.
// Returns a map of snapshot name to error (nil if successful).
func DestroySnapshots(names []string) map[string]error {
	results := make(map[string]error, len(names))
	for _, name := range names {
		results[name] = DestroySnapshot(name)
	}
	return results
}

// PoolStatuses returns the status of all ZFS pools.
func PoolStatuses() ([]PoolStatus, error) {
	cmd := exec.Command("zpool", "list", "-H", "-o", "name,health")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list pools: %w", err)
	}

	var statuses []PoolStatus
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			status := PoolStatus{
				Name:  fields[0],
				State: fields[1],
			}
			// Get detailed status for error info
			detail, err := poolDetail(fields[0])
			if err == nil {
				status.Errors = detail.Errors
				status.Raw = detail.Raw
			}
			statuses = append(statuses, status)
		}
	}
	return statuses, nil
}

func poolDetail(pool string) (PoolStatus, error) {
	cmd := exec.Command("zpool", "status", pool)
	out, err := cmd.Output()
	if err != nil {
		return PoolStatus{}, err
	}

	raw := string(out)
	status := PoolStatus{Raw: raw}

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "errors:") {
			status.Errors = strings.TrimPrefix(line, "errors:")
			status.Errors = strings.TrimSpace(status.Errors)
		}
		if strings.HasPrefix(line, "state:") {
			status.State = strings.TrimPrefix(line, "state:")
			status.State = strings.TrimSpace(status.State)
		}
	}
	return status, nil
}

func parseSnapshots(output string) ([]Snapshot, error) {
	var snapshots []Snapshot
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Fields are tab-separated from -H flag.
		// name, used, refer are tab fields, but creation is the rest of the line
		// Format: name\tused\trefer\tcreation_string
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		used := strings.TrimSpace(parts[1])
		refer := strings.TrimSpace(parts[2])
		creationStr := strings.TrimSpace(parts[3])

		// Parse dataset and short name
		atIdx := strings.Index(name, "@")
		var dataset, shortName string
		if atIdx >= 0 {
			dataset = name[:atIdx]
			shortName = name[atIdx+1:]
		} else {
			dataset = name
			shortName = name
		}

		// Parse creation time - ZFS outputs like "Mon Jan  2 15:04 2006"
		creation, _ := parseZFSTime(creationStr)

		snapshots = append(snapshots, Snapshot{
			Name:      name,
			Dataset:   dataset,
			ShortName: shortName,
			Used:      used,
			Refer:     refer,
			Creation:  creation,
		})
	}
	return snapshots, nil
}

func parseZFSTime(s string) (time.Time, error) {
	// ZFS outputs creation in several formats depending on locale.
	// Common format: "Thu Feb 27 10:30 2025"
	formats := []string{
		"Mon Jan _2 15:04 2006",
		"Mon Jan  2 15:04 2006",
		"Mon Jan 2 15:04 2006",
		time.ANSIC,
		time.UnixDate,
	}
	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

// CheckZFSErrors checks all pools for errors and returns a summary.
func CheckZFSErrors() (hasErrors bool, summary string, err error) {
	statuses, err := PoolStatuses()
	if err != nil {
		return false, "", err
	}

	var issues []string
	for _, s := range statuses {
		if s.State != "ONLINE" {
			issues = append(issues, fmt.Sprintf("Pool %q is in state: %s", s.Name, s.State))
		}
		if s.Errors != "" && s.Errors != "No known data errors" {
			issues = append(issues, fmt.Sprintf("Pool %q has errors: %s", s.Name, s.Errors))
		}
	}

	if len(issues) > 0 {
		return true, strings.Join(issues, "\n"), nil
	}
	return false, "All pools healthy", nil
}
