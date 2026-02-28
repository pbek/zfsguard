// Package monitor provides the background monitoring service for zfsguard.
package monitor

import (
	"fmt"
	"log"
	"time"

	"github.com/pbek/zfsguard/internal/config"
	"github.com/pbek/zfsguard/internal/notify"
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

	if s.cfg.Monitor.CheckZFS {
		hasErrors, summary, err := zfs.CheckZFSErrors()
		if err != nil {
			log.Printf("ZFS check error: %v", err)
			issues = append(issues, fmt.Sprintf("ZFS check failed: %v", err))
		} else if hasErrors {
			log.Printf("ZFS issues found: %s", summary)
			issues = append(issues, "ZFS: "+summary)
		} else {
			log.Printf("ZFS: %s", summary)
		}
	}

	if s.cfg.Monitor.CheckSMART {
		hasErrors, summary, err := zfs.CheckSMARTErrors(s.cfg.Monitor.SMARTDevices)
		if err != nil {
			log.Printf("SMART check error: %v", err)
			issues = append(issues, fmt.Sprintf("SMART check failed: %v", err))
		} else if hasErrors {
			log.Printf("SMART issues found: %s", summary)
			issues = append(issues, "SMART: "+summary)
		} else {
			log.Printf("SMART: %s", summary)
		}
	}

	if len(issues) > 0 {
		msg := ""
		for _, issue := range issues {
			msg += issue + "\n"
		}
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

// Run starts the monitoring loop.
func (s *Service) Run() error {
	interval := time.Duration(s.cfg.Monitor.IntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 60 * time.Minute
	}

	log.Printf("Starting ZFSGuard monitor (interval: %s)", interval)
	log.Printf("ZFS checks: %v, SMART checks: %v", s.cfg.Monitor.CheckZFS, s.cfg.Monitor.CheckSMART)

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
