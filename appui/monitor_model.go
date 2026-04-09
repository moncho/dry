package appui

import (
	"context"
	"fmt"
	"image/color"
	"regexp"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/docker/go-units"
	"github.com/moncho/dry/docker"
)

const monitorHistoryWindow = 3 * time.Minute

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

type MonitorSeries struct {
	CPU    []MonitorPoint
	Memory []MonitorPoint
}

type MonitorPoint struct {
	At    time.Time
	Value float64
}

// MonitorModel shows live container stats.
type MonitorModel struct {
	table   TableModel
	daemon  docker.ContainerDaemon
	stats   map[string]*docker.Stats
	history map[string]MonitorSeries
	cancels map[string]context.CancelFunc
	active  bool
	width   int
	height  int
}

var monitorSortableFields = []int{0, 1, 2, 3, 4}

// NewMonitorModel creates a monitor model.
func NewMonitorModel() MonitorModel {
	columns := []Column{
		{Title: "CONTAINER", Width: IDColumnWidth + 2, Fixed: true},
		{Title: "CPU LOAD", Width: 20, Fixed: true},
		{Title: "MEM USAGE/LIMIT", Width: 30, Fixed: true},
		{Title: "MEM LOAD", Width: 20, Fixed: true},
		{Title: "NET RX/TX", Width: 26, Fixed: true},
		{Title: "BLOCK R/W", Width: 26, Fixed: true},
	}
	return MonitorModel{
		table:   NewTableModel(columns),
		stats:   make(map[string]*docker.Stats),
		history: make(map[string]MonitorSeries),
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
	m.table.SetSize(w, maxInt(h-WidgetHeaderLines-1, 1)) // header + summary line
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
	m.history = make(map[string]MonitorSeries)
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
	m.history = make(map[string]MonitorSeries)
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

// UpdateStats stores stats and records history without rebuilding the table.
// Call FlushTable to apply accumulated updates to the table.
// Returns a command to continue listening on the same channel.
func (m *MonitorModel) UpdateStats(cid string, stats *docker.Stats, ch <-chan *docker.Stats) tea.Cmd {
	m.stats[cid] = stats
	m.recordHistory(cid, stats)
	// Re-subscribe to the same channel
	return listenContainerStats(cid, ch)
}

// FlushTable rebuilds the table rows from the current stats.
func (m *MonitorModel) FlushTable() {
	m.refreshTable()
}

// RemoveContainer removes a container from monitoring.
func (m *MonitorModel) RemoveContainer(cid string) {
	if cancel, ok := m.cancels[cid]; ok {
		cancel()
		delete(m.cancels, cid)
	}
	delete(m.stats, cid)
	delete(m.history, cid)
	m.refreshTable()
}

func (m *MonitorModel) refreshTable() {
	var rows []TableRow
	for cid, s := range m.stats {
		if s == nil || s.Error != nil {
			continue
		}
		rows = append(rows, monitorRow{
			cid:     cid,
			columns: monitorRowColumns(s),
		})
	}
	m.table.SetRows(rows)
	m.table.Sort()
}

// SelectedStats returns the stats entry under the cursor, or nil.
func (m MonitorModel) SelectedStats() *docker.Stats {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	mr, ok := row.(monitorRow)
	if !ok {
		return nil
	}
	return m.stats[mr.cid]
}

func (m MonitorModel) SelectedSeries() MonitorSeries {
	row := m.table.SelectedRow()
	if row == nil {
		return MonitorSeries{}
	}
	mr, ok := row.(monitorRow)
	if !ok {
		return MonitorSeries{}
	}
	return m.SeriesFor(mr.cid)
}

func (m MonitorModel) SeriesFor(cid string) MonitorSeries {
	series, ok := m.history[cid]
	if !ok {
		return MonitorSeries{}
	}
	return MonitorSeries{
		CPU:    append([]MonitorPoint(nil), series.CPU...),
		Memory: append([]MonitorPoint(nil), series.Memory...),
	}
}

func (m MonitorModel) StatsByID(cid string) *docker.Stats {
	return m.stats[cid]
}

func (m MonitorModel) RowCount() int {
	return m.table.RowCount()
}

// StatsCount returns the number of containers with stored stats.
func (m MonitorModel) StatsCount() int {
	return len(m.stats)
}

// Update handles key events.
func (m MonitorModel) Update(msg tea.Msg) (MonitorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1":
			m.nextSort()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *MonitorModel) nextSort() {
	current := m.table.SortField()
	idx := slices.Index(monitorSortableFields, current)
	if idx < 0 {
		m.table.SetSortField(monitorSortableFields[0])
		m.table.Sort()
		return
	}
	next := monitorSortableFields[(idx+1)%len(monitorSortableFields)]
	m.table.SetSortField(next)
	m.table.Sort()
}

// View renders the monitor.
func (m MonitorModel) View() string {
	header := RenderWidgetHeader(WidgetHeaderOpts{
		Icon:     "◉",
		Title:    "Monitor",
		Total:    len(m.stats),
		Filtered: m.table.RowCount(),
		Width:    m.width,
		Accent:   DryTheme.Info,
	})
	return header + monitorSummaryLine(m.stats, m.width) + "\n" + m.renderTableView()
}

// RefreshTableStyles re-applies theme styles to the inner table.
func (m *MonitorModel) RefreshTableStyles() {
	m.table.RefreshStyles()
}

// monitorBar renders a half-block progress bar with a numeric label for a
// percentage value (0–100). Using ▌/█ gives double the resolution of a
// full-block bar and guarantees a visible indicator when pct > 0.
func monitorBar(pct float64, width int) string {
	return monitorBarWithBase(pct, width, DryTheme.Info)
}

func monitorBarWithBase(pct float64, width int, base color.Color) string {
	fg := monitorLoadColor(pct, base)

	frac := pct / 100
	if frac > 1 {
		frac = 1
	}

	// Two "slots" per character: left half (▌) and full (█).
	slots := float64(width) * 2
	filled := frac * slots
	full := int(filled) / 2 // number of full-block characters
	half := int(filled) % 2 // 1 if there's a trailing half-block

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

func monitorLoadColor(pct float64, base color.Color) color.Color {
	switch {
	case pct >= 90:
		return DryTheme.Error
	case pct >= 75:
		return DryTheme.Warning
	case pct >= 35:
		return base
	default:
		return DryTheme.Success
	}
}

func monitorContainerCell(s *docker.Stats) string {
	dot := ColorFg("●", monitorLoadColor(maxFloat(s.CPUPercentage, s.MemoryPercentage), DryTheme.Info))
	return dot + " " + ColorFg(s.CID, DryTheme.Fg)
}

func monitorMemoryCell(s *docker.Stats) string {
	used := ColorFg(units.BytesSize(s.Memory), monitorLoadColor(s.MemoryPercentage, DryTheme.Secondary))
	sep := ColorFg(" / ", DryTheme.FgSubtle)
	limit := ColorFg(units.BytesSize(s.MemoryLimit), DryTheme.FgMuted)
	return used + sep + limit
}

func monitorIOCell(leftLabel string, left float64, rightLabel string, right float64, leftColor, rightColor color.Color) string {
	leftText := ColorFg(leftLabel, leftColor) + " " + ColorFg(units.BytesSize(left), leftColor)
	rightText := ColorFg(rightLabel, rightColor) + " " + ColorFg(units.BytesSize(right), rightColor)
	return leftText + ColorFg("  ", DryTheme.FgSubtle) + rightText
}

func monitorSummaryLine(stats map[string]*docker.Stats, width int) string {
	bg := lipgloss.NewStyle().Background(DryTheme.Bg)
	labelStyle := lipgloss.NewStyle().
		Foreground(DryTheme.FgSubtle).
		Background(DryTheme.Bg)
	valueStyle := lipgloss.NewStyle().
		Foreground(DryTheme.Fg).
		Background(DryTheme.Bg).
		Bold(true)
	sepStyle := lipgloss.NewStyle().
		Foreground(DryTheme.FgSubtle).
		Background(DryTheme.Bg)

	var hottestCPU, hottestMem *docker.Stats
	for _, s := range stats {
		if s == nil || s.Error != nil {
			continue
		}
		if hottestCPU == nil || s.CPUPercentage > hottestCPU.CPUPercentage {
			hottestCPU = s
		}
		if hottestMem == nil || s.MemoryPercentage > hottestMem.MemoryPercentage {
			hottestMem = s
		}
	}

	parts := []string{
		labelStyle.Render("live ") + valueStyle.Render(fmt.Sprintf("%d", len(stats))),
	}
	if hottestCPU != nil {
		parts = append(parts,
			labelStyle.Render("hot cpu ")+ColorFg(hottestCPU.CID, DryTheme.Info)+
				labelStyle.Render(" ")+ColorFg(fmt.Sprintf("%.1f%%", hottestCPU.CPUPercentage), monitorLoadColor(hottestCPU.CPUPercentage, DryTheme.Info)),
		)
	}
	if hottestMem != nil {
		parts = append(parts,
			labelStyle.Render("hot mem ")+ColorFg(hottestMem.CID, DryTheme.Secondary)+
				labelStyle.Render(" ")+ColorFg(fmt.Sprintf("%.1f%%", hottestMem.MemoryPercentage), monitorLoadColor(hottestMem.MemoryPercentage, DryTheme.Secondary)),
		)
	}

	line := bg.Render(" ") + strings.Join(parts, sepStyle.Render("  •  "))
	return padMonitorLine(line, width, bg)
}

func padMonitorLine(line string, width int, bg lipgloss.Style) string {
	w := ansi.StringWidth(line)
	if w > width {
		return ansi.Truncate(line, width, "")
	}
	if w < width {
		line += bg.Render(strings.Repeat(" ", width-w))
	}
	return line
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func monitorRowColumns(s *docker.Stats) []string {
	return []string{
		monitorContainerCell(s),
		monitorBar(s.CPUPercentage, 12),
		monitorMemoryCell(s),
		monitorBarWithBase(s.MemoryPercentage, 12, DryTheme.Secondary),
		monitorIOCell("↓", s.NetworkRx, "↑", s.NetworkTx, DryTheme.Info, DryTheme.Tertiary),
		monitorIOCell("R", s.BlockRead, "W", s.BlockWrite, DryTheme.Warning, DryTheme.FgMuted),
	}
}

func (m MonitorModel) renderTableView() string {
	view := m.table.View()
	row := m.table.SelectedRow()
	if row != nil {
		mr, ok := row.(monitorRow)
		if ok {
			lines := strings.Split(view, "\n")
			selectedLine := m.table.Cursor() + 1
			if selectedLine > 0 && selectedLine < len(lines) {
				lines[selectedLine] = m.renderSelectedRow(lines[selectedLine], mr)
			}
			view = strings.Join(lines, "\n")
		}
	}
	if m.width <= 0 {
		return view
	}
	lines := strings.Split(view, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], m.width, "")
	}
	return strings.Join(lines, "\n")
}

func (m MonitorModel) renderSelectedRow(line string, row monitorRow) string {
	_ = row
	line = stripMonitorSelectedBackground(line)
	return strings.Replace(line, "●", "▶", 1)
}

func stripMonitorSelectedBackground(line string) string {
	line = monitorCursorLineBGPattern().ReplaceAllString(line, "")
	line = strings.ReplaceAll(line, "\x1b[49m", "")
	return line
}

func monitorCursorLineBGPattern() *regexp.Regexp {
	r, g, b := rgb8(DryTheme.CursorLineBg)
	return regexp.MustCompile(fmt.Sprintf(`\x1b\[[0-9;]*48;2;%d;%d;%d[0-9;]*m`, r, g, b))
}

func rgb8(c color.Color) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

func (m *MonitorModel) recordHistory(cid string, stats *docker.Stats) {
	if stats == nil {
		return
	}
	series := m.history[cid]
	now := time.Now()
	series.CPU = appendMonitorSample(series.CPU, MonitorPoint{At: now, Value: stats.CPUPercentage})
	series.Memory = appendMonitorSample(series.Memory, MonitorPoint{At: now, Value: stats.MemoryPercentage})
	m.history[cid] = series
}

func appendMonitorSample(samples []MonitorPoint, value MonitorPoint) []MonitorPoint {
	samples = append(samples, value)
	if value.At.IsZero() || len(samples) < 2 {
		return samples
	}
	cutoff := value.At.Add(-monitorHistoryWindow)
	start := 0
	for start < len(samples)-1 && samples[start].At.Before(cutoff) {
		start++
	}
	if start == 0 {
		return samples
	}
	return append([]MonitorPoint(nil), samples[start:]...)
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
