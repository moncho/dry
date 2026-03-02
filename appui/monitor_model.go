package appui

import (
	"context"
	"fmt"
	"sort"

	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/docker/go-units"
	"github.com/moncho/dry/docker"
)

// MonitorStatsMsg carries stats update for a container.
type MonitorStatsMsg struct {
	CID     string
	Stats   *docker.Stats
	StatsCh <-chan *docker.Stats // channel to re-subscribe
}

// MonitorErrorMsg carries a stats error.
type MonitorErrorMsg struct {
	CID string
	Err error
}

// monitorRow displays stats for one container.
type monitorRow struct {
	cid     string
	columns []string
}

func (r monitorRow) Columns() []string { return r.columns }
func (r monitorRow) ID() string        { return r.cid }

// MonitorModel shows live container stats.
type MonitorModel struct {
	table   TableModel
	daemon  docker.ContainerDaemon
	stats   map[string]*docker.Stats
	cancels map[string]context.CancelFunc
	active  bool
	width   int
	height  int
}

// NewMonitorModel creates a monitor model.
func NewMonitorModel() MonitorModel {
	columns := []Column{
		{Title: "CONTAINER", Width: IDColumnWidth, Fixed: true},
		{Title: "CPU %", Width: 20, Fixed: true},
		{Title: "MEM USAGE/LIMIT", Width: 30, Fixed: true},
		{Title: "MEM %", Width: 20, Fixed: true},
		{Title: "NET I/O", Width: 26, Fixed: true},
		{Title: "BLOCK I/O", Width: 26, Fixed: true},
		{Title: "PIDS", Width: 10, Fixed: true},
		{Title: "COMMAND"},
	}
	return MonitorModel{
		table:   NewTableModel(columns),
		stats:   make(map[string]*docker.Stats),
		cancels: make(map[string]context.CancelFunc),
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *MonitorModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *MonitorModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table.SetSize(w, h-2) // -2 for header + blank line
}

// Active returns whether monitoring is active.
func (m MonitorModel) Active() bool {
	return m.active
}

// Start begins monitoring all running containers.
// Returns commands that will stream stats.
func (m *MonitorModel) Start() []tea.Cmd {
	m.StopAll()
	m.active = true
	m.stats = make(map[string]*docker.Stats)
	m.cancels = make(map[string]context.CancelFunc)

	if m.daemon == nil {
		return nil
	}

	containers := m.daemon.Containers(
		[]docker.ContainerFilter{docker.ContainerFilters.Running()},
		docker.SortByContainerID,
	)

	var cmds []tea.Cmd
	for _, c := range containers {
		ch, cancel, err := m.startContainerStats(c)
		if err != nil {
			continue
		}
		m.cancels[c.ID] = cancel
		cmds = append(cmds, listenContainerStats(c.ID, ch))
	}
	return cmds
}

// StopAll stops monitoring all containers.
func (m *MonitorModel) StopAll() {
	for id, cancel := range m.cancels {
		cancel()
		delete(m.cancels, id)
	}
	m.active = false
}

func (m *MonitorModel) startContainerStats(c *docker.Container) (<-chan *docker.Stats, context.CancelFunc, error) {
	sc, err := m.daemon.StatsChannel(c)
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch := sc.Start(ctx)
	return ch, cancel, nil
}

// UpdateStats updates stats for a container and refreshes the table.
// Returns a command to continue listening on the same channel.
func (m *MonitorModel) UpdateStats(cid string, stats *docker.Stats, ch <-chan *docker.Stats) tea.Cmd {
	m.stats[cid] = stats
	m.refreshTable()
	// Re-subscribe to the same channel
	return listenContainerStats(cid, ch)
}

// RemoveContainer removes a container from monitoring.
func (m *MonitorModel) RemoveContainer(cid string) {
	if cancel, ok := m.cancels[cid]; ok {
		cancel()
		delete(m.cancels, cid)
	}
	delete(m.stats, cid)
	m.refreshTable()
}

func (m *MonitorModel) refreshTable() {
	ids := make([]string, 0, len(m.stats))
	for cid := range m.stats {
		ids = append(ids, cid)
	}
	sort.Strings(ids)

	var rows []TableRow
	for _, cid := range ids {
		s := m.stats[cid]
		if s == nil || s.Error != nil {
			continue
		}
		rows = append(rows, monitorRow{
			cid: cid,
			columns: []string{
				s.CID,
				monitorBar(s.CPUPercentage, 12),
				ColorFg(fmt.Sprintf("%s / %s",
					units.BytesSize(s.Memory),
					units.BytesSize(s.MemoryLimit)), DryTheme.FgMuted),
				monitorBar(s.MemoryPercentage, 12),
				ColorFg(fmt.Sprintf("%s / %s",
					units.BytesSize(s.NetworkRx),
					units.BytesSize(s.NetworkTx)), DryTheme.Tertiary),
				ColorFg(fmt.Sprintf("%s / %s",
					units.BytesSize(s.BlockRead),
					units.BytesSize(s.BlockWrite)), DryTheme.FgMuted),
				ColorFg(fmt.Sprintf("%d", s.PidsCurrent), DryTheme.FgMuted),
				s.Command,
			},
		})
	}
	m.table.SetRows(rows)
}

// Update handles key events.
func (m MonitorModel) Update(msg tea.Msg) (MonitorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1":
			m.table.NextSort()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the monitor.
func (m MonitorModel) View() string {
	header := RenderWidgetHeader(WidgetHeaderOpts{
		Title:    "Monitor",
		Total:    len(m.stats),
		Filtered: m.table.RowCount(),
		Width:    m.width,
		Accent:   DryTheme.Info,
	})
	return header + "\n" + m.table.View()
}

// RefreshTableStyles re-applies theme styles to the inner table.
func (m *MonitorModel) RefreshTableStyles() {
	m.table.RefreshStyles()
}

// monitorBar renders a half-block progress bar with a numeric label for a
// percentage value (0–100). Using ▌/█ gives double the resolution of a
// full-block bar and guarantees a visible indicator when pct > 0.
func monitorBar(pct float64, width int) string {
	fg := DryTheme.Info
	if pct > 80 {
		fg = DryTheme.Warning
	}

	frac := pct / 100
	if frac > 1 {
		frac = 1
	}

	// Two "slots" per character: left half (▌) and full (█).
	slots := float64(width) * 2
	filled := frac * slots
	full := int(filled) / 2  // number of full-block characters
	half := int(filled) % 2  // 1 if there's a trailing half-block

	// Always show at least a half-block when pct > 0.
	if pct > 0 && full == 0 && half == 0 {
		half = 1
	}

	empty := width - full - half

	var b strings.Builder
	fillStyle := lipgloss.NewStyle().Foreground(fg)
	halfStyle := lipgloss.NewStyle().Foreground(fg).Background(DryTheme.Border)
	emptyStyle := lipgloss.NewStyle().Foreground(DryTheme.Border)

	if full > 0 {
		b.WriteString(fillStyle.Render(strings.Repeat("█", full)))
	}
	if half > 0 {
		b.WriteString(halfStyle.Render("▌"))
	}
	if empty > 0 {
		b.WriteString(emptyStyle.Render(strings.Repeat("─", empty)))
	}

	label := ColorFg(fmt.Sprintf(" %5.1f%%", pct), fg)
	return b.String() + label
}

// listenContainerStats creates a command that reads from a stats channel.
func listenContainerStats(cid string, ch <-chan *docker.Stats) tea.Cmd {
	return func() tea.Msg {
		stats, ok := <-ch
		if !ok {
			return MonitorErrorMsg{CID: cid, Err: fmt.Errorf("stats channel closed")}
		}
		if stats.Error != nil {
			return MonitorErrorMsg{CID: cid, Err: stats.Error}
		}
		return MonitorStatsMsg{CID: cid, Stats: stats, StatsCh: ch}
	}
}
