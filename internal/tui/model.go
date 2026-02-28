// Package tui provides the terminal user interface for zfsguard.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pbek/zfsguard/internal/zfs"
)

// view represents the current screen.
type view int

const (
	viewList view = iota
	viewCreate
	viewConfirmDelete
	viewConfirmDeleteAll
	viewStatus
)

// keyMap defines the keybindings for the TUI.
type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Select     key.Binding
	SelectAll  key.Binding
	Delete     key.Binding
	DeleteAll  key.Binding
	Create     key.Binding
	Refresh    key.Binding
	Confirm    key.Binding
	Cancel     key.Binding
	Quit       key.Binding
	Help       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	FilterMode key.Binding
}

var keys = keyMap{
	Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/up", "move up")),
	Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/down", "move down")),
	Select:     key.NewBinding(key.WithKeys(" ", "x"), key.WithHelp("space/x", "toggle select")),
	SelectAll:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select/deselect all")),
	Delete:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete selected")),
	DeleteAll:  key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete ALL snapshots")),
	Create:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "create snapshot")),
	Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Confirm:    key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
	Cancel:     key.NewBinding(key.WithKeys("n", "escape"), key.WithHelp("n/esc", "cancel")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
	PageUp:     key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("PgUp", "page up")),
	PageDown:   key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("PgDn", "page down")),
	FilterMode: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Select, k.Delete, k.Create, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Select, k.SelectAll, k.FilterMode},
		{k.Create, k.Delete, k.DeleteAll, k.Refresh},
		{k.Help, k.Quit},
	}
}

// Model is the main TUI model.
type Model struct {
	snapshots []zfs.Snapshot
	datasets  []string
	cursor    int
	offset    int // scroll offset for viewport
	height    int // terminal height
	width     int // terminal width

	currentView view
	statusMsg   string
	statusErr   bool

	// Create snapshot form
	createInput   textinput.Model
	createDataset int // index into datasets

	help     help.Model
	showHelp bool
	keys     keyMap

	// Filter
	filterInput  textinput.Model
	filterActive bool
	filterText   string
	filtered     []int // indices into snapshots that match the filter

	err error
}

// messages
type snapshotsLoadedMsg struct {
	snapshots []zfs.Snapshot
	datasets  []string
}
type errMsg struct{ err error }
type statusMsg struct {
	msg   string
	isErr bool
}
type clearStatusMsg struct{}

// NewModel creates a new TUI model.
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "snapshot-name"
	ti.CharLimit = 128
	ti.Width = 40

	fi := textinput.New()
	fi.Placeholder = "type to filter..."
	fi.CharLimit = 128
	fi.Width = 40

	return Model{
		keys:        keys,
		help:        help.New(),
		createInput: ti,
		filterInput: fi,
		height:      24,
		width:       80,
	}
}

func (m Model) Init() tea.Cmd {
	return loadSnapshots
}

func loadSnapshots() tea.Msg {
	snaps, err := zfs.ListSnapshots()
	if err != nil {
		return errMsg{err}
	}
	datasets, _ := zfs.ListDatasets()
	return snapshotsLoadedMsg{snapshots: snaps, datasets: datasets}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.help.Width = msg.Width
		return m, nil

	case snapshotsLoadedMsg:
		m.snapshots = msg.snapshots
		m.datasets = msg.datasets
		m.err = nil
		m.applyFilter()
		m.clampCursor()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case statusMsg:
		m.statusMsg = msg.msg
		m.statusErr = msg.isErr
		return m, tea.Tick(4*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		m.statusMsg = ""
		m.statusErr = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Update text inputs if active
	if m.currentView == viewCreate {
		var cmd tea.Cmd
		m.createInput, cmd = m.createInput.Update(msg)
		return m, cmd
	}
	if m.filterActive {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.filterText = m.filterInput.Value()
		m.applyFilter()
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle filter mode input
	if m.filterActive {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "escape"))):
			m.filterActive = false
			m.filterInput.Blur()
			if msg.String() == "escape" {
				m.filterText = ""
				m.filterInput.SetValue("")
				m.applyFilter()
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.filterText = m.filterInput.Value()
			m.applyFilter()
			m.clampCursor()
			return m, cmd
		}
	}

	// Handle create view input
	if m.currentView == viewCreate {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			return m.executeCreate()
		case key.Matches(msg, keys.Cancel):
			m.currentView = viewList
			m.createInput.Blur()
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			m.createDataset = (m.createDataset + 1) % max(len(m.datasets), 1)
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			m.createDataset = (m.createDataset - 1 + max(len(m.datasets), 1)) % max(
				len(m.datasets),
				1,
			)
			return m, nil
		default:
			var cmd tea.Cmd
			m.createInput, cmd = m.createInput.Update(msg)
			return m, cmd
		}
	}

	// Handle confirm delete
	if m.currentView == viewConfirmDelete || m.currentView == viewConfirmDeleteAll {
		switch {
		case key.Matches(msg, keys.Confirm):
			return m.executeDelete()
		case key.Matches(msg, keys.Cancel):
			m.currentView = viewList
			return m, nil
		}
		return m, nil
	}

	// Main list view
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, keys.Up):
		m.cursor--
		m.clampCursor()
		m.ensureVisible()
		return m, nil

	case key.Matches(msg, keys.Down):
		m.cursor++
		m.clampCursor()
		m.ensureVisible()
		return m, nil

	case key.Matches(msg, keys.PageUp):
		m.cursor -= m.viewportHeight()
		m.clampCursor()
		m.ensureVisible()
		return m, nil

	case key.Matches(msg, keys.PageDown):
		m.cursor += m.viewportHeight()
		m.clampCursor()
		m.ensureVisible()
		return m, nil

	case key.Matches(msg, keys.Select):
		if idx := m.currentIndex(); idx >= 0 {
			m.snapshots[idx].Selected = !m.snapshots[idx].Selected
		}
		return m, nil

	case key.Matches(msg, keys.SelectAll):
		m.toggleSelectAll()
		return m, nil

	case key.Matches(msg, keys.FilterMode):
		m.filterActive = true
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, keys.Create):
		m.currentView = viewCreate
		m.createInput.SetValue(time.Now().Format("2006-01-02_15-04-05"))
		m.createInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, keys.Delete):
		selected := m.selectedSnapshots()
		if len(selected) == 0 {
			// Delete the one under cursor
			if idx := m.currentIndex(); idx >= 0 {
				m.snapshots[idx].Selected = true
			} else {
				return m, nil
			}
		}
		m.currentView = viewConfirmDelete
		return m, nil

	case key.Matches(msg, keys.DeleteAll):
		if len(m.snapshots) == 0 {
			return m, nil
		}
		m.currentView = viewConfirmDeleteAll
		return m, nil

	case key.Matches(msg, keys.Refresh):
		return m, loadSnapshots
	}

	return m, nil
}

func (m *Model) executeCreate() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.createInput.Value())
	if name == "" {
		m.statusMsg = "Snapshot name cannot be empty"
		m.statusErr = true
		return m, nil
	}

	dataset := ""
	if len(m.datasets) > 0 && m.createDataset < len(m.datasets) {
		dataset = m.datasets[m.createDataset]
	}
	if dataset == "" {
		m.statusMsg = "No dataset selected"
		m.statusErr = true
		return m, nil
	}

	fullName := dataset + "@" + name
	m.currentView = viewList
	m.createInput.Blur()

	return m, func() tea.Msg {
		if err := zfs.CreateSnapshot(fullName); err != nil {
			return statusMsg{msg: fmt.Sprintf("Failed to create: %v", err), isErr: true}
		}
		return statusMsg{msg: fmt.Sprintf("Created snapshot: %s", fullName), isErr: false}
	}
}

func (m *Model) executeDelete() (tea.Model, tea.Cmd) {
	var toDelete []string

	if m.currentView == viewConfirmDeleteAll {
		for i := range m.snapshots {
			toDelete = append(toDelete, m.snapshots[i].Name)
		}
	} else {
		for _, s := range m.selectedSnapshots() {
			toDelete = append(toDelete, s.Name)
		}
	}

	m.currentView = viewList

	return m, func() tea.Msg {
		results := zfs.DestroySnapshots(toDelete)
		var failed int
		for _, err := range results {
			if err != nil {
				failed++
			}
		}
		// Reload snapshots
		snaps, _ := zfs.ListSnapshots()
		datasets, _ := zfs.ListDatasets()

		if failed > 0 {
			return snapshotsLoadedMsg{snapshots: snaps, datasets: datasets}
		}
		return snapshotsLoadedMsg{snapshots: snaps, datasets: datasets}
	}
}

func (m *Model) selectedSnapshots() []zfs.Snapshot {
	var selected []zfs.Snapshot
	for _, s := range m.snapshots {
		if s.Selected {
			selected = append(selected, s)
		}
	}
	return selected
}

func (m *Model) toggleSelectAll() {
	indices := m.visibleIndices()
	allSelected := true
	for _, idx := range indices {
		if !m.snapshots[idx].Selected {
			allSelected = false
			break
		}
	}
	for _, idx := range indices {
		m.snapshots[idx].Selected = !allSelected
	}
}

func (m *Model) visibleIndices() []int {
	if len(m.filtered) > 0 || m.filterText != "" {
		return m.filtered
	}
	indices := make([]int, len(m.snapshots))
	for i := range m.snapshots {
		indices[i] = i
	}
	return indices
}

func (m *Model) currentIndex() int {
	indices := m.visibleIndices()
	if m.cursor >= 0 && m.cursor < len(indices) {
		return indices[m.cursor]
	}
	return -1
}

func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.filtered = nil
		return
	}
	m.filtered = nil
	lower := strings.ToLower(m.filterText)
	for i, s := range m.snapshots {
		if strings.Contains(strings.ToLower(s.Name), lower) {
			m.filtered = append(m.filtered, i)
		}
	}
}

func (m *Model) clampCursor() {
	total := len(m.visibleIndices())
	if total == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
}

func (m *Model) viewportHeight() int {
	// Reserve lines for header, footer, help
	reserved := 6
	if m.showHelp {
		reserved += 8
	}
	if m.statusMsg != "" {
		reserved++
	}
	if m.filterActive || m.filterText != "" {
		reserved++
	}
	h := m.height - reserved
	if h < 3 {
		h = 3
	}
	return h
}

func (m *Model) ensureVisible() {
	vpHeight := m.viewportHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vpHeight {
		m.offset = m.cursor - vpHeight + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
