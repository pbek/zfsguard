// Package zfs provides SMART disk health checking utilities.
package zfs

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// SMARTStatus holds the SMART health status of a disk.
type SMARTStatus struct {
	Device  string
	Healthy bool
	Summary string
	Raw     string
}

// CheckSMART runs smartctl on the given devices and returns their health status.
// If devices is empty, it attempts to auto-detect devices.
func CheckSMART(devices []string) ([]SMARTStatus, error) {
	if len(devices) == 0 {
		var err error
		devices, err = detectDevices()
		if err != nil {
			return nil, fmt.Errorf("failed to detect devices: %w", err)
		}
	}

	var statuses []SMARTStatus
	for _, dev := range devices {
		status := checkDevice(dev)
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// CheckSMARTErrors checks all detected disks and returns a summary of any issues.
func CheckSMARTErrors(devices []string) (hasErrors bool, summary string, err error) {
	statuses, err := CheckSMART(devices)
	if err != nil {
		return false, "", err
	}

	var issues []string
	for _, s := range statuses {
		if !s.Healthy {
			issues = append(issues, fmt.Sprintf("Device %s: %s", s.Device, s.Summary))
		}
	}

	if len(issues) > 0 {
		return true, strings.Join(issues, "\n"), nil
	}
	return false, "All disks healthy", nil
}

func detectDevices() ([]string, error) {
	cmd := exec.Command("smartctl", "--scan")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var devices []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			devices = append(devices, fields[0])
		}
	}
	return devices, nil
}

func checkDevice(device string) SMARTStatus {
	cmd := exec.Command("smartctl", "-H", device)
	out, err := cmd.CombinedOutput()
	raw := string(out)

	status := SMARTStatus{
		Device: device,
		Raw:    raw,
	}

	if err != nil {
		// smartctl returns non-zero for unhealthy disks
		status.Healthy = false
		status.Summary = "smartctl returned an error"
	}

	// Parse the output for PASSED/FAILED
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "PASSED") || strings.Contains(line, "OK") {
			status.Healthy = true
			status.Summary = "PASSED"
			return status
		}
		if strings.Contains(line, "FAILED") {
			status.Healthy = false
			status.Summary = "FAILED - " + line
			return status
		}
	}

	if status.Summary == "" {
		status.Summary = "Unable to determine health status"
		status.Healthy = true // assume healthy if we can't determine
	}
	return status
}
