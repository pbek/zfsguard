// zfsguard is a TUI for managing ZFS snapshots.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pbek/zfsguard/internal/config"
	"github.com/pbek/zfsguard/internal/tui"
	"github.com/pbek/zfsguard/internal/version"
)

func main() {
	// Simple flag parsing (avoid importing flag to keep startup fast)
	var configPath string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version", "-v":
			fmt.Println(version.String("zfsguard"))
			os.Exit(0)
		case "--config":
			if i+1 < len(args) {
				i++
				configPath = args[i]
			}
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	m := tui.NewModel(cfg.Monitor.ReportPath)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
