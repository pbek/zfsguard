// Package monitor provides the background monitoring service for zfsguard.
package monitor

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pbek/zfsguard/internal/config"
	"github.com/pbek/zfsguard/internal/notify"
	"github.com/pbek/zfsguard/internal/report"
	"github.com/pbek/zfsguard/internal/zfs"
)

// Service is the monitoring service that checks ZFS and SMART health.
type Service struct {
	cfg      config.Config
	notifier *notify.Notifier
}

// New creates a new monitoring service.
func New(cfg config.Config) *Service {
	return &Service{
		cfg:      cfg,
		notifier: notify.New(cfg.Notify),
	}
}

// RunOnce performs a single health check cycle.
func (s *Service) RunOnce() error {
	log.Println("Running health check...")

	var issues []string
	var pools []zfs.PoolStatus
	var disks []zfs.SMARTStatus
	var poolErr, diskErr error

	if s.cfg.Monitor.CheckZFS {
		pools, poolErr = zfs.PoolStatuses()
		if poolErr != nil {
			log.Printf("ZFS check error: %v", poolErr)
			issues = append(issues, fmt.Sprintf("ZFS check failed: %v", poolErr))
		} else {
			hasErrors, summary, err := zfs.CheckZFSErrors()
			if err != nil {
				log.Printf("ZFS check error: %v", err)
				poolErr = err
				issues = append(issues, fmt.Sprintf("ZFS check failed: %v", err))
			} else if hasErrors {
				log.Printf("ZFS issues found: %s", summary)
				issues = append(issues, "ZFS: "+summary)
			} else {
				log.Printf("ZFS: %s", summary)
			}
		}
	}

	if s.cfg.Monitor.CheckSMART {
		disks, diskErr = zfs.CheckSMART(s.cfg.Monitor.SMARTDevices)
		if diskErr != nil {
			log.Printf("SMART check error: %v", diskErr)
			issues = append(issues, fmt.Sprintf("SMART check failed: %v", diskErr))
		} else {
			// Check for unhealthy disks
			for _, d := range disks {
				if !d.Healthy {
					summary := fmt.Sprintf("Device %s: %s", d.Device, d.Summary)
					log.Printf("SMART issues found: %s", summary)
					issues = append(issues, "SMART: "+summary)
				}
			}
			if len(issues) == 0 || !containsSMART(issues) {
				log.Println("SMART: All disks healthy")
			}
		}
	}

	// Write health report to disk
	if s.cfg.Monitor.ReportPath != "" {
		r := report.FromChecks(pools, poolErr, disks, diskErr)
		if err := report.Write(s.cfg.Monitor.ReportPath, r); err != nil {
			log.Printf("Failed to write health report: %v", err)
		} else {
			log.Printf("Health report written to %s", s.cfg.Monitor.ReportPath)
		}
	}

	if len(issues) > 0 {
		msg := strings.Join(issues, "\n") + "\n"
		if err := s.notifier.Send("ZFSGuard Alert", msg); err != nil {
			log.Printf("Failed to send notification: %v", err)
			return err
		}
		log.Println("Alert notification sent")
	} else {
		log.Println("All checks passed")
	}

	return nil
}

func containsSMART(issues []string) bool {
	for _, s := range issues {
		if strings.HasPrefix(s, "SMART:") {
			return true
		}
	}
	return false
}

// Run starts the monitoring loop.
func (s *Service) Run() error {
	interval := time.Duration(s.cfg.Monitor.IntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 60 * time.Minute
	}

	log.Printf("Starting ZFSGuard monitor (interval: %s)", interval)
	log.Printf("ZFS checks: %v, SMART checks: %v", s.cfg.Monitor.CheckZFS, s.cfg.Monitor.CheckSMART)
	if s.cfg.Monitor.ReportPath != "" {
		log.Printf("Health report path: %s", s.cfg.Monitor.ReportPath)
	}

	if len(s.cfg.Notify.ShoutrrrURLs) > 0 {
		log.Printf("Configured %d notification service(s)", len(s.cfg.Notify.ShoutrrrURLs))
	}
	if s.cfg.Notify.Desktop {
		log.Println("Desktop notifications enabled")
	}

	// Run immediately on start
	if err := s.RunOnce(); err != nil {
		log.Printf("Initial check error: %v", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.RunOnce(); err != nil {
			log.Printf("Check error: %v", err)
		}
	}

	return nil
}
