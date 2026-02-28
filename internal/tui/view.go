package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00BFFF")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#888888"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF88"))

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#333355")).
			Foreground(lipgloss.Color("#FFFFFF"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	checkMark = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF88")).
			Render("[x]")

	uncheckMark = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Render("[ ]")

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF88")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Bold(true).
			Padding(0, 1)

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF8800")).
			Padding(1, 2).
			Width(60)

	createDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00BFFF")).
				Padding(1, 2).
				Width(60)

	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00"))

	countStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	datasetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF"))

	snapNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))
)

func (m Model) View() string {
	var b strings.Builder

	// Title bar
	title := titleStyle.Render(" ZFSGuard - ZFS Snapshot Manager ")
	b.WriteString(title)
	b.WriteString("\n")

	switch m.currentView {
	case viewCreate:
		b.WriteString(m.viewCreate())
	case viewConfirmDelete:
		b.WriteString(m.viewList())
		b.WriteString("\n")
		b.WriteString(m.viewConfirmDelete())
	case viewConfirmDeleteAll:
		b.WriteString(m.viewList())
		b.WriteString("\n")
		b.WriteString(m.viewConfirmDeleteAll())
	default:
		b.WriteString(m.viewList())
	}

	// Status bar
	if m.statusMsg != "" {
		if m.statusErr {
			b.WriteString(errorStyle.Render(m.statusMsg))
		} else {
			b.WriteString(statusStyle.Render(m.statusMsg))
		}
		b.WriteString("\n")
	}

	// Error
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Help
	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(m.help.FullHelpView(m.keys.FullHelp()))
	} else {
		b.WriteString("\n")
		b.WriteString(m.help.ShortHelpView(m.keys.ShortHelp()))
	}

	return b.String()
}

func (m Model) viewList() string {
	var b strings.Builder

	indices := m.visibleIndices()
	total := len(m.snapshots)
	visible := len(indices)
	selected := len(m.selectedSnapshots())

	// Filter bar
	if m.filterActive {
		b.WriteString(filterStyle.Render("Filter: "))
		b.WriteString(m.filterInput.View())
		b.WriteString("\n")
	} else if m.filterText != "" {
		b.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s ", m.filterText)))
		b.WriteString(countStyle.Render(fmt.Sprintf("(%d/%d matched)", visible, total)))
		b.WriteString("\n")
	}

	// Count line
	info := fmt.Sprintf(" %d snapshots", total)
	if selected > 0 {
		info += fmt.Sprintf(" | %d selected", selected)
	}
	b.WriteString(countStyle.Render(info))
	b.WriteString("\n")

	// Column headers
	header := fmt.Sprintf("  %-5s %-30s %-10s %-10s %s", "Sel", "Name", "Used", "Refer", "Created")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(headerStyle.Render(strings.Repeat("─", min(m.width, 90))))
	b.WriteString("\n")

	if len(indices) == 0 {
		if total == 0 {
			b.WriteString(normalStyle.Render("  No snapshots found. Press 'c' to create one."))
		} else {
			b.WriteString(normalStyle.Render("  No snapshots match the filter."))
		}
		b.WriteString("\n")
		return b.String()
	}

	// Viewport
	vpHeight := m.viewportHeight()
	start := m.offset
	end := start + vpHeight
	if end > len(indices) {
		end = len(indices)
	}

	for vi := start; vi < end; vi++ {
		idx := indices[vi]
		snap := m.snapshots[idx]

		check := uncheckMark
		if snap.Selected {
			check = checkMark
		}

		// Format the name with colored dataset@snapshot
		nameParts := strings.SplitN(snap.Name, "@", 2)
		var formattedName string
		if len(nameParts) == 2 {
			formattedName = datasetStyle.Render(
				nameParts[0],
			) + normalStyle.Render(
				"@",
			) + snapNameStyle.Render(
				nameParts[1],
			)
		} else {
			formattedName = snap.Name
		}

		created := ""
		if !snap.Creation.IsZero() {
			created = snap.Creation.Format("2006-01-02 15:04")
		}

		// Build the row (uncolored for width calculation)
		plainName := snap.Name
		if len(plainName) > 30 {
			plainName = plainName[:27] + "..."
			nameParts := strings.SplitN(snap.Name[:27], "@", 2)
			if len(nameParts) == 2 {
				formattedName = datasetStyle.Render(
					nameParts[0],
				) + normalStyle.Render(
					"@",
				) + snapNameStyle.Render(
					nameParts[1],
				) + normalStyle.Render(
					"...",
				)
			}
		}

		// Build the colored line with formatted dataset@snapshot coloring
		coloredLine := fmt.Sprintf("  %s ", check)
		padLen := 30 - len(plainName)
		if padLen < 0 {
			padLen = 0
		}
		coloredLine += formattedName + strings.Repeat(" ", padLen)
		coloredLine += fmt.Sprintf(" %-10s %-10s %s", snap.Used, snap.Refer, created)

		if vi == m.cursor {
			b.WriteString(cursorStyle.Render(coloredLine))
		} else if snap.Selected {
			b.WriteString(selectedStyle.Render(coloredLine))
		} else {
			b.WriteString(coloredLine)
		}
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(indices) > vpHeight {
		pct := float64(m.offset) / float64(len(indices)-vpHeight) * 100
		scrollInfo := fmt.Sprintf(" [%d-%d of %d] %.0f%%", start+1, end, len(indices), pct)
		b.WriteString(countStyle.Render(scrollInfo))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) viewCreate() string {
	var b strings.Builder

	b.WriteString("\n")

	content := "Create New Snapshot\n\n"

	if len(m.datasets) == 0 {
		content += "No datasets found.\nPress Esc to go back.\n"
	} else {
		content += "Dataset (Tab/Shift+Tab to cycle):\n"
		ds := m.datasets[m.createDataset%len(m.datasets)]
		content += datasetStyle.Render("  "+ds) + "\n\n"
		content += "Snapshot name:\n"
		content += "  " + m.createInput.View() + "\n\n"
		content += fmt.Sprintf("Will create: %s@%s\n\n", ds, m.createInput.Value())
		content += "Enter to confirm | Esc to cancel"
	}

	b.WriteString(createDialogStyle.Render(content))
	return b.String()
}

func (m Model) viewConfirmDelete() string {
	selected := m.selectedSnapshots()
	content := fmt.Sprintf("Delete %d selected snapshot(s)?\n\n", len(selected))
	for i, s := range selected {
		if i >= 10 {
			content += fmt.Sprintf("  ... and %d more\n", len(selected)-10)
			break
		}
		content += fmt.Sprintf("  - %s\n", s.Name)
	}
	content += "\nThis action requires elevated privileges.\n"
	content += "Press 'y' to confirm, 'n'/Esc to cancel"
	return dialogStyle.Render(content)
}

func (m Model) viewConfirmDeleteAll() string {
	content := fmt.Sprintf("DELETE ALL %d SNAPSHOTS?\n\n", len(m.snapshots))
	content += "This will destroy every snapshot on the system.\n"
	content += "This action is IRREVERSIBLE and requires elevated privileges.\n\n"
	content += "Press 'y' to confirm, 'n'/Esc to cancel"
	return dialogStyle.Render(content)
}
