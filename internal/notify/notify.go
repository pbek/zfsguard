// Package notify provides notification dispatching for zfsguard.
// It uses shoutrrr for remote services and notify-send for local desktop.
package notify

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/containrrr/shoutrrr"
	"github.com/pbek/zfsguard/internal/config"
)

// Notifier dispatches notifications to configured services.
type Notifier struct {
	cfg config.NotifyConfig
}

// New creates a new Notifier from the given config.
func New(cfg config.NotifyConfig) *Notifier {
	return &Notifier{cfg: cfg}
}

// Send sends a notification with the given title and message to all
// configured notification services.
func (n *Notifier) Send(title, message string) error {
	var errs []string

	// Send to shoutrrr services
	for _, url := range n.cfg.ShoutrrrURLs {
		formatted := fmt.Sprintf("[%s] %s", title, message)
		if err := shoutrrr.Send(url, formatted); err != nil {
			errs = append(errs, fmt.Sprintf("shoutrrr (%s): %v", maskURL(url), err))
		}
	}

	// Send desktop notification
	if n.cfg.Desktop {
		if err := sendDesktop(title, message); err != nil {
			errs = append(errs, fmt.Sprintf("desktop: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

// sendDesktop sends a local Linux desktop notification using notify-send.
func sendDesktop(title, message string) error {
	// Try notify-send first (most common on Linux desktops)
	if path, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command(path, "--app-name=zfsguard", "--urgency=critical", title, message)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("notify-send failed: %s: %w", string(out), err)
		}
		return nil
	}

	return fmt.Errorf("no desktop notification tool found (install libnotify/notify-send)")
}

// maskURL masks sensitive parts of notification URLs for error messages.
func maskURL(url string) string {
	// Show only the scheme and first few chars
	if idx := strings.Index(url, "://"); idx >= 0 {
		scheme := url[:idx]
		rest := url[idx+3:]
		if len(rest) > 8 {
			rest = rest[:8] + "..."
		}
		return scheme + "://" + rest
	}
	if len(url) > 12 {
		return url[:12] + "..."
	}
	return url
}
