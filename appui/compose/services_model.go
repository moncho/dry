package compose

import (
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// serviceRow wraps a ComposeService as a TableRow.
type serviceRow struct {
	service docker.ComposeService
	columns []string
}

func newServiceRow(s docker.ComposeService, sync docker.ServiceSync) serviceRow {
	return serviceRow{
		service: s,
		columns: []string{
			"  " + s.Name,
			fmt.Sprintf("%d", s.Containers),
			fmt.Sprintf("%d", s.Running),
			fmt.Sprintf("%d", s.Exited),
			s.Image,
			colorHealth(s.Health),
			colorSync(sync),
			s.Ports,
		},
	}
}

func (r serviceRow) Columns() []string { return r.columns }

// ID prefixes the service name the way the network and volume rows do:
// service names come from a user-controlled label, so an unprefixed
// "net:x" would collide with the network x.
func (r serviceRow) ID() string { return "svc:" + r.service.Name }

// sectionRow is a visual group header separating resource types.
type sectionRow struct {
	label   string
	columns []string
}

func newSectionRow(label string, count int) sectionRow {
	title := appui.ColorFg(fmt.Sprintf("%s (%d)", label, count), appui.DryTheme.Info)
	return sectionRow{
		label:   label,
		columns: []string{title, "", "", "", "", "", "", ""},
	}
}

func (r sectionRow) Columns() []string { return r.columns }
func (r sectionRow) ID() string        { return "section:" + r.label }

// networkRow wraps a ComposeNetwork as a TableRow.
type networkRow struct {
	network docker.ComposeNetwork
	columns []string
}

func newNetworkRow(n docker.ComposeNetwork) networkRow {
	return networkRow{
		network: n,
		columns: []string{"  " + n.Name, "", "", "", n.Driver, n.Scope, "", ""},
	}
}

func (r networkRow) Columns() []string { return r.columns }
func (r networkRow) ID() string        { return "net:" + r.network.Name }

// volumeRow wraps a ComposeVolume as a TableRow.
type volumeRow struct {
	volume  docker.ComposeVolume
	columns []string
}

func newVolumeRow(v docker.ComposeVolume) volumeRow {
	return volumeRow{
		volume:  v,
		columns: []string{"  " + v.Name, "", "", "", v.Driver, "", "", ""},
	}
}

func (r volumeRow) Columns() []string { return r.columns }
func (r volumeRow) ID() string        { return "vol:" + r.volume.Name }

// ServicesLoadedMsg carries loaded compose resources for a project. Gen is
// the load's generation, so a load that lands after a newer one is ignored;
// two are in flight after an f5 during a container event. Generations start
// at 1, so a message built by hand with Gen unset is older than every real
// load and is dropped once any load has been applied.
type ServicesLoadedMsg struct {
	Services []docker.ComposeService
	Networks []docker.ComposeNetwork
	Volumes  []docker.ComposeVolume
	Project  string
	Gen      uint64
}

// ServicesModel is the Compose project resources view.
type ServicesModel struct {
	table        appui.TableModel
	filter       appui.FilterInputModel
	project      string
	serviceCount int
	services     []docker.ComposeService
	networks     []docker.ComposeNetwork
	volumes      []docker.ComposeVolume
	drift        map[string]map[string]docker.ServiceSync
	loading      bool
}

// NewServicesModel creates a compose services list model.
func NewServicesModel() ServicesModel {
	columns := []appui.Column{
		{Title: "NAME"},
		{Title: "CONTAINERS", Width: 12, Fixed: true},
		{Title: "RUNNING", Width: 10, Fixed: true},
		{Title: "EXITED", Width: 10, Fixed: true},
		{Title: "IMAGE/DRIVER"},
		{Title: "HEALTH/SCOPE", Width: 14, Fixed: true},
		{Title: "SYNC", Width: 7, Fixed: true},
		{Title: "PORTS"},
	}
	m := ServicesModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
		// Waiting, not empty: a model that has never been given a project
		// has nothing to show for the same reason one mid-load does.
		loading: true,
	}
	m.syncEmptyMessage()
	return m
}

// FilterActive returns true when the filter input is active.
func (m ServicesModel) FilterActive() bool { return m.filter.Active() }

// Filtered reports whether a filter is narrowing the list, open or not. It
// tells "no services" apart from "the filter is hiding them".
func (m ServicesModel) Filtered() bool { return m.table.FilterText() != "" }

// ServiceRowCount is how many service rows this view shows before
// filtering: the ones with containers plus the ones only the compose file
// knows about. It is zero while the first load is in flight, because
// SetProject clears the resource slices along with raising the flag, and
// refreshRows recounts from those slices; the workspace panel depends on
// that, so a change to either has a test on it.
func (m ServicesModel) ServiceRowCount() int { return m.serviceCount }

// Loading reports that the first load for this project has not arrived
// yet, the state where calling the project empty would be a guess. An f5
// or an event-driven reload does not raise it: the rows on screen stay
// answerable while it runs.
func (m ServicesModel) Loading() bool { return m.loading }

// SetSize updates the table dimensions.
func (m *ServicesModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-2-filterH)
	m.filter.SetWidth(w)
}

// SetServices replaces the resource list with services, networks, and volumes.
func (m *ServicesModel) SetServices(services []docker.ComposeService, networks []docker.ComposeNetwork, volumes []docker.ComposeVolume, project string) {
	m.project = project
	m.loading = false
	m.services = services
	m.networks = networks
	m.volumes = volumes
	m.refreshRows()
}

// SetDrift records per-service sync status, keyed by project then service.
func (m *ServicesModel) SetDrift(drift map[string]map[string]docker.ServiceSync) {
	m.drift = drift
	m.refreshRows()
}

// refreshRows rebuilds the resource rows from the current services,
// networks, volumes, and drift status.
func (m *ServicesModel) refreshRows() {
	// Ascending, the one direction any dry table sorts in.
	field, asc := m.table.SortField(), true
	// Each section is sorted on its own and the sections keep their order.
	// The table's flat sort moves rows between sections, leaving Services
	// with none under it and service rows below Volumes.
	section := func(rows []appui.TableRow) []appui.TableRow {
		sort.SliceStable(rows, func(i, j int) bool {
			return appui.CompareRowsByColumn(rows[i], rows[j], field, asc)
		})
		return rows
	}
	var rows []appui.TableRow

	// Held back while loading: the drift map may still describe this project
	// from an earlier cycle.
	var notCreated []docker.ComposeService
	if !m.loading {
		notCreated = notCreatedServices(m.project, m.services, m.drift[m.project])
	}
	m.serviceCount = len(m.services) + len(notCreated)
	if m.serviceCount > 0 {
		rows = append(rows, newSectionRow("Services", m.serviceCount))
		services := mergeByName(m.services, notCreated)
		serviceRows := make([]appui.TableRow, 0, len(services))
		for _, s := range services {
			serviceRows = append(serviceRows, newServiceRow(s, m.drift[s.Project][s.Name]))
		}
		rows = append(rows, section(serviceRows)...)
	}
	if len(m.networks) > 0 {
		rows = append(rows, newSectionRow("Networks", len(m.networks)))
		networkRows := make([]appui.TableRow, 0, len(m.networks))
		for _, n := range m.networks {
			networkRows = append(networkRows, newNetworkRow(n))
		}
		rows = append(rows, section(networkRows)...)
	}
	if len(m.volumes) > 0 {
		rows = append(rows, newSectionRow("Volumes", len(m.volumes)))
		volumeRows := make([]appui.TableRow, 0, len(m.volumes))
		for _, v := range m.volumes {
			volumeRows = append(volumeRows, newVolumeRow(v))
		}
		rows = append(rows, section(volumeRows)...)
	}

	selected := m.selectedID()
	m.table.SetRows(rows)
	// A rebuild has no direction of travel: prefer the row above, since
	// everything below a removed row has shifted up.
	m.restoreSelection(selected, false)
	m.syncEmptyMessage()
}

// selectedID identifies the row the cursor is on, or "" when no key can act
// on it. A header is deliberately unidentified: following one across a
// rebuild would put the cursor straight back on it.
func (m ServicesModel) selectedID() string {
	row := m.table.SelectedRow()
	if row == nil || !selectableRow(row) {
		return ""
	}
	return row.ID()
}

// restoreSelection puts the cursor back on the row it was on, and when that
// row is gone leaves it wherever SetRows clamped it, on the last row if the
// list shrank past it, with ensureSelectableRow moving it off a header.
// Following the row rather than the index is what keeps the selection still
// while the list moves under it: by index, a reload, a sort or a filter
// keystroke slides it silently onto a neighbour, often a different kind of
// resource.
func (m *ServicesModel) restoreSelection(id string, forward bool) {
	if id != "" && m.table.SelectRowByID(id) {
		return
	}
	m.ensureSelectableRow(forward)
}

// SetProject clears the resource list and records the project being loaded.
// The view switches before the resources arrive, so without this it goes on
// rendering the previous project's rows, under the previous project's title,
// until the load lands. Clearing both together is what makes the empty view
// mean "loading" rather than "this project is empty".
func (m *ServicesModel) SetProject(project string) {
	m.project = project
	m.services = nil
	m.networks = nil
	m.volumes = nil
	// Until the load lands, empty means "not loaded yet".
	m.loading = true
	// The filter goes with the project it was typed for: carried over, it
	// hides rows the user never filtered, usually all of them.
	m.filter.Clear()
	m.table.SetFilter("")
	m.refreshRows()
}

// composeResource is what a row resolves to. Exactly one field is set, or
// none at all for a row that is only decoration.
type composeResource struct {
	service *docker.ComposeService
	network *docker.ComposeNetwork
	volume  *docker.ComposeVolume
}

// empty reports that the row resolves to nothing a key could act on.
func (r composeResource) empty() bool {
	return r.service == nil && r.network == nil && r.volume == nil
}

// resourceRow is a row that resolves to one of the view's resources.
// Selectability and the Selected* accessors read this same resolution, so
// they cannot disagree about what a row is. The cursor can still be left on
// a header when a filter leaves nothing else.
type resourceRow interface {
	appui.TableRow
	resource() composeResource
}

func (r serviceRow) resource() composeResource { return composeResource{service: &r.service} }
func (r networkRow) resource() composeResource { return composeResource{network: &r.network} }
func (r volumeRow) resource() composeResource  { return composeResource{volume: &r.volume} }

// resourceOf resolves a row, returning the empty resource for decoration.
func resourceOf(row appui.TableRow) composeResource {
	if r, ok := row.(resourceRow); ok {
		return r.resource()
	}
	return composeResource{}
}

// selectableRow reports whether a key can act on the given row.
func selectableRow(row appui.TableRow) bool {
	return !resourceOf(row).empty()
}

// ensureSelectableRow moves the cursor off a section header onto the nearest
// row a key can act on, preferring the direction the user was travelling. It
// falls back to the other one when that runs out, which is what carries a
// fresh load past row 0, always a header. With no selectable row at all, a
// filter matching only a header, the cursor stays put.
func (m *ServicesModel) ensureSelectableRow(forward bool) {
	rows := m.table.FilteredRows()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(rows) || selectableRow(rows[cursor]) {
		return
	}
	steps := []int{1, -1}
	if !forward {
		steps = []int{-1, 1}
	}
	for _, step := range steps {
		for i := cursor + step; i >= 0 && i < len(rows); i += step {
			if selectableRow(rows[i]) {
				m.table.SetCursor(i)
				return
			}
		}
	}
}

// syncEmptyMessage pushes the current reason into the table; see the same
// method on ProjectsModel for why it is not done in View.
func (m *ServicesModel) syncEmptyMessage() { m.table.SetEmptyMessage(m.emptyMessage()) }

// emptyMessage says which of the three empty states this is.
func (m ServicesModel) emptyMessage() string {
	switch {
	case m.loading:
		return "Loading the project's resources..."
	case m.table.FilterText() != "":
		return "Nothing here matches the filter"
	default:
		return "This project has no services, networks or volumes"
	}
}

// SelectedService returns the service under the cursor, or nil.
func (m ServicesModel) SelectedService() *docker.ComposeService {
	return resourceOf(m.table.SelectedRow()).service
}

// SelectedNetwork returns the network under the cursor, or nil.
func (m ServicesModel) SelectedNetwork() *docker.ComposeNetwork {
	return resourceOf(m.table.SelectedRow()).network
}

// SelectedVolume returns the volume under the cursor, or nil.
func (m ServicesModel) SelectedVolume() *docker.ComposeVolume {
	return resourceOf(m.table.SelectedRow()).volume
}

// Update handles key events.
func (m ServicesModel) Update(msg tea.Msg) (ServicesModel, tea.Cmd) {
	if m.filter.Active() {
		var cmd tea.Cmd
		selected := m.selectedID()
		m.filter, cmd = m.filter.Update(msg)
		m.table.SetFilter(m.filter.Value())
		m.restoreSelection(selected, true)
		m.syncEmptyMessage()
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1":
			// SetSortField moves the indicator without sorting; the
			// rebuild sorts inside each section. refreshRows follows the
			// selected row, whose index means nothing after a reorder.
			m.table.NextSortField()
			m.refreshRows()
			return m, nil
		case "%":
			cmd := m.filter.Activate()
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	m.ensureSelectableRow(!movesUp(msg))
	return m, cmd
}

// movesUp reports whether msg is an upward navigation key, the only thing
// that makes header-skipping search backwards. GotoTop is deliberately not
// one: it lands on row 0, always a header, so the skip continues forward.
// Exhaustive for NewTableModel's keymap, which is where the bindings are.
func movesUp(msg tea.Msg) bool {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false
	}
	switch key.String() {
	case "up", "k", "pgup":
		return true
	}
	return false
}

// View renders the services list.
func (m ServicesModel) View() string {
	title := "Compose Resources"
	if m.project != "" {
		title = fmt.Sprintf("Compose: %s", m.project)
	}
	filtered := m.serviceCount
	if m.table.FilterText() != "" {
		filtered = m.countFilteredServices()
	}
	header := appui.RenderWidgetHeader(appui.WidgetHeaderOpts{
		Icon:     "\U0001f433",
		Title:    title,
		Total:    m.serviceCount,
		Filtered: filtered,
		Filter:   m.table.FilterText(),
		Width:    m.table.Width(),
		Accent:   appui.DryTheme.Info,
	})
	result := header + "\n" + m.table.View()
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

// countFilteredServices counts how many service rows survive the current filter.
func (m ServicesModel) countFilteredServices() int {
	count := 0
	for _, row := range m.table.FilteredRows() {
		if _, ok := row.(serviceRow); ok {
			count++
		}
	}
	return count
}

// RefreshTableStyles re-applies theme styles to the inner table.
func (m *ServicesModel) RefreshTableStyles() {
	m.table.RefreshStyles()
}
