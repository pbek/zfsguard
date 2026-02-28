// zfsguard is a TUI for managing ZFS snapshots.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pbek/zfsguard/internal/tui"
	"github.com/pbek/zfsguard/internal/version"
)

func main() {
	// Simple --version / -v flag check (avoid importing flag to keep startup fast)
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Println(version.String("zfsguard"))
			os.Exit(0)
		}
	}

	m := tui.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
