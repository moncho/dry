# Bubbletea v2 Migration Spec

Migrate dry from tcell + gizak/termui to Bubbletea v2, Bubbles, and Lipgloss v2. The goal is to preserve all existing functionality while modernizing the UI framework and improving the look of the application.

## New Dependencies

| Package | Import Path | Replaces |
|---------|-------------|----------|
| Bubbletea v2 | `charm.land/bubbletea/v2` | `github.com/gdamore/tcell`, event loop, screen management |
| Bubbles v2 | `charm.land/bubbles/v2` | Custom table/list/input/viewport widgets |
| Lipgloss v2 | `charm.land/lipgloss/v2` | `github.com/gizak/termui`, markup system, color/theme system |

Note: v2 of all three Charmbracelet libraries moved to the `charm.land` vanity domain. The source remains on GitHub but the Go module paths changed.

### Prerequisites

- **Go version**: Upgrade `go.mod` from `go 1.23` to `go 1.26`. Bubbletea v2 and Bubbles v2 require Go 1.24.2 minimum; we target 1.26 for current toolchain support. This also requires updating:
  - CI workflow `.github/workflows/go.yml` (currently uses Go 1.22)
  - CI workflow `.github/workflows/go-lint.yml`
  - Dockerfile (currently uses alpine and downloads a pre-built binary; if switching to source build, needs Go 1.26 image)

### Dependencies to Remove

- `github.com/gdamore/tcell` (and `tcell/termbox` compat layer)
- `github.com/gizak/termui`

---

## Architectural Changes

### Current Architecture

```
main.go → app.NewDry() → app.RenderLoop(dry)
                              │
              ┌───────────────┼───────────────┐
              │               │               │
         render goroutine  event loop    message goroutine
         (renderChan)      (PollEvent)   (output chan)
              │               │
              ▼               ▼
         render(dry)     handler.handle(ev, nextHandler)
              │               │
              ▼               ▼
    screen.RenderBufferer  eventHandler chain
    screen.Flush()         (forwarder for modals)
```

- Imperative rendering: widgets produce `Buffer` (cell maps), screen merges them
- Event channels: `chan *tcell.EventKey` piped to handlers and modal sub-views
- Global singletons: `ui.ActiveScreen`, package-level `widgets`, `refreshScreen()`

### Target Architecture (Bubbletea v2)

```
main.go → tea.NewProgram(NewModel()) → p.Run()
                                         │
                              ┌──────────┼──────────┐
                              │          │          │
                           Init()    Update()    View()
                              │          │          │
                              ▼          ▼          ▼
                         tea.Cmd    tea.Msg      tea.View
                        (docker     dispatch     (string
                         connect)   to active     building
                                    sub-model     with lipgloss)
```

- Declarative rendering: `View()` returns styled strings, Bubbletea handles terminal
- Message dispatch: type-switch on `tea.Msg` in `Update()`, forward to active sub-model
- No globals: all state lives in the model tree; Docker daemon passed through models

### Top-Level Model

```go
type model struct {
    // State
    view         viewMode
    previousView viewMode
    width        int
    height       int
    showHeader   bool
    ready        bool

    // Docker
    daemon docker.ContainerDaemon
    config Config

    // Sub-models (one per view)
    containers   containersModel
    images       imagesModel
    networks     networksModel
    volumes      volumesModel
    monitor      monitorModel
    diskUsage    diskUsageModel
    containerMenu containerMenuModel
    nodes        nodesModel
    services     servicesModel
    stacks       stacksModel
    nodeTasks    nodeTasksModel
    serviceTasks serviceTasksModel
    stackTasks   stackTasksModel

    // Shared components
    header    headerModel
    help      help.Model
    messageBar messageBarModel
    prompt    promptModel

    // Overlay state
    overlay overlayType  // none, less, prompt, stream
    less    lessModel
}
```

### Message Types

Define custom messages for Docker operations and internal communication:

```go
// Docker data messages
type containersLoadedMsg struct{ containers []docker.Container }
type imagesLoadedMsg struct{ images []types.ImageSummary }
type networksLoadedMsg struct{ networks []types.NetworkResource }
type volumesLoadedMsg struct{ volumes []*volume.Volume }
type containerStatsMsg struct{ id string; stats docker.Stats }
type dockerEventMsg struct{ event events.Message }
type dockerConnectedMsg struct{ daemon docker.ContainerDaemon }
type dockerErrorMsg struct{ err error }

// Operation result messages
type operationSuccessMsg struct{ message string }
type operationErrorMsg struct{ err error }

// Internal messages
type refreshMsg struct{}
type statusMessageMsg struct{ text string; expiry time.Duration }
```

---

## Package-by-Package Migration Plan

### Phase 1: Foundation (`ui/` package rewrite)

The `ui/` package is the lowest layer. It must be rewritten first since everything depends on it.

#### 1.1 Remove: `ui/screen.go`

The `Screen` struct wrapping `tcell.Screen` is eliminated. Its responsibilities move to:

| Screen method | Bubbletea equivalent |
|---------------|---------------------|
| `NewScreen()` / `Close()` | `tea.NewProgram()` / `tea.Quit` |
| `Clear()` / `Flush()` | Automatic (Bubbletea handles rendering) |
| `Resize()` | `tea.WindowSizeMsg` in `Update()` |
| `Fill()` / `RenderRune()` | `lipgloss.Style.Render()` in `View()` |
| `RenderBufferer()` | Eliminated (no more Buffer system) |
| `RenderLine()` / `Render()` | `lipgloss` styled strings |
| `ShowCursor()` / `HideCursor()` | `tea.View.Cursor` field |
| `Sync()` | Not needed |
| `Dimensions()` | `tea.WindowSizeMsg` width/height stored in model |

**Global `ActiveScreen` is eliminated.** Window dimensions are stored in the top-level model and passed down to sub-models.

#### 1.2 Remove: `ui/events.go`, `ui/render.go`, `ui/focus.go`

- `EventChannel()` (polls `tcell.PollEvent`) → eliminated; Bubbletea handles event polling
- `EventSource` struct → eliminated; events arrive as `tea.Msg` in `Update()`
- `Focusable` interface → replaced by Bubbletea model focus patterns (active sub-model receives messages)
- `ScreenTextRenderer` → eliminated; rendering is string-based

#### 1.3 Rewrite: `ui/markup.go` → Lipgloss Style Rendering

The custom markup tag system (`<white>`, `<blue>`, `<b>`, etc.) must be replaced.

**Option A (recommended): Incremental — keep markup as internal format, convert to lipgloss at render time.**

Create a `RenderMarkup(s string) string` function that:
1. Parses the existing `<tag>text</>` format
2. Applies `lipgloss.Style` for each tag
3. Returns a styled string

This allows existing markup usage throughout the codebase to work initially, with gradual migration to direct lipgloss usage.

**Tag → lipgloss.Style mapping:**

| Markup Tag | Lipgloss Equivalent |
|-----------|-------------------|
| `<white>` | `lipgloss.NewStyle().Foreground(lipgloss.Color("255"))` |
| `<blue>` | `lipgloss.NewStyle().Foreground(lipgloss.Color("33"))` |
| `<red>` | `lipgloss.NewStyle().Foreground(lipgloss.Color("196"))` |
| `<green>` | `lipgloss.NewStyle().Foreground(lipgloss.Color("190"))` |
| `<yellow>` | `lipgloss.NewStyle().Foreground(lipgloss.Color("11"))` |
| `<cyan>` | `lipgloss.NewStyle().Foreground(lipgloss.Color("14"))` |
| `<darkgrey>` | `lipgloss.NewStyle().Foreground(lipgloss.Color("240"))` |
| `<b>` | `lipgloss.NewStyle().Bold(true)` |
| `<u>` | `lipgloss.NewStyle().Underline(true)` |
| `<r>` | `lipgloss.NewStyle().Reverse(true)` |
| `</>` | Reset to default |

#### 1.4 Rewrite: `ui/theme.go` + `appui/theme.go`

Replace `ColorTheme` struct with lipgloss styles:

```go
type Theme struct {
    // Base
    Fg     lipgloss.Color
    Bg     lipgloss.Color
    DarkBg lipgloss.Color

    // Semantic styles
    Header       lipgloss.Style
    Footer       lipgloss.Style
    Selected     lipgloss.Style
    StatusBar    lipgloss.Style
    Prompt       lipgloss.Style
    Key          lipgloss.Style
    Info         lipgloss.Style
    Error        lipgloss.Style

    // Table styles
    TableHeader  lipgloss.Style
    TableRow     lipgloss.Style
    TableRowAlt  lipgloss.Style
    TableCursor  lipgloss.Style

    // Container status styles
    Running      lipgloss.Style
    Stopped      lipgloss.Style
    Paused       lipgloss.Style
}
```

Use `lipgloss.AdaptiveColor` for light/dark terminal support.

#### 1.5 Rewrite: `ui/less.go` → Less Model (viewport + textinput)

The Less viewer becomes a Bubbletea sub-model:

```go
type lessModel struct {
    viewport    viewport.Model
    searchInput textinput.Model
    filterInput textinput.Model

    content     []string       // all lines
    filtered    []string       // filtered lines
    searchResult *search.Result
    following   bool
    searching   bool
    filtering   bool
    width       int
    height      int
}
```

**Key mappings remain the same** but handled via `Update()`:
- `Esc` → return `closeOverlayMsg{}`
- `Up/Down/PgUp/PgDown/g/G` → forward to `viewport.Update()`
- `/` → activate search input
- `F` → activate filter input
- `f` → toggle follow mode
- `n/N` → next/previous search hit

**Search highlighting**: Use `lipgloss.StyleRanges()` to highlight matched text in the viewport content.

**Follow mode**: When following, `GotoBottom()` on every content update.

#### 1.6 Rewrite: `ui/input.go` → Eliminate

Replace with `textinput.Model` from Bubbles. All usages of `InputBox` migrate to the `textinput` Bubble. The Bubbles textinput already handles:
- Cursor movement (Home/End, Ctrl+A/E, Ctrl+B/F)
- Delete (Backspace, Delete, Ctrl+K, Ctrl+D)
- Text insertion
- Visual cursor rendering

#### 1.7 Rewrite: `ui/cursor.go`

The `Cursor` struct (tracks position with min/max bounds) can remain as a utility for list/table navigation, but `ShowCursor`/`HideCursor` screen calls are removed — cursor display is handled by Bubbletea's `tea.View.Cursor` field.

#### 1.8 Rewrite: `ui/view.go` → Eliminate or simplify

The `View` struct (internal text buffer with line storage and cursor) was used by the Less viewer. Replace with `viewport.Model` which handles scrollable content natively.

#### 1.9 Remove: `ui/tcell.go`, `ui/termbox.go`

These files exist solely to bridge tcell/termbox. Eliminated entirely.

#### 1.10 Remove: `ui/key.go`

The custom `Key` struct holding `[]tcell.Key` is replaced by `key.Binding` from Bubbles.

---

### Phase 2: Termui Primitives (`ui/termui/` rewrite)

#### 2.1 Remove: `ui/termui/widget.go`

The `Widget` interface (`Buffer()`, `Mount()`, `Name()`, `Unmount()`) is replaced by the Bubbletea `Model` interface (`Init()`, `Update()`, `View()`).

However, the Mount/Unmount lifecycle pattern is still useful for lazy data loading. Implement it as:
- `Mount()` becomes a `tea.Cmd` that fetches data from Docker and returns a loaded message
- `Unmount()` is implicit — when a view is switched away, its data goes stale; re-entering triggers a new mount command

#### 2.2 Remove: `ui/termui/row.go`, `ui/termui/header.go`, `ui/termui/par_column.go`, `ui/termui/gauge_column.go`

The row/column grid system based on `gizaktermui.Buffer` merging is replaced by lipgloss string layout:

| Old | New |
|-----|-----|
| `Row` with `[]GridBufferer` columns | `lipgloss.JoinHorizontal()` of styled column strings |
| `TableHeader` with fixed/proportional columns | `lipgloss/table` headers or lipgloss-styled header row |
| `ParColumn` (text cell) | `lipgloss.NewStyle().Width(w).Render(text)` |
| `GaugeColumn` (progress bar cell) | `progress.Model` from Bubbles, rendered inline |

**Column width calculation**: The current `TableHeader.calculateColumnWidths()` logic (some columns fixed, others proportional) should be preserved. Implement as a utility that returns `[]int` widths given the total width and column definitions.

#### 2.3 Rewrite: `ui/termui/textinput.go` → Eliminate

Replace with `textinput.Model` from Bubbles.

#### 2.4 Remove: `ui/termui/par_markup.go`, `ui/termui/textbuilder.go`

Markup paragraphs become `lipgloss`-styled strings via the `RenderMarkup()` function or direct lipgloss usage.

#### 2.5 Remove: `ui/termui/keyvalue.go`

Key-value display becomes simple lipgloss rendering:
```go
keyStyle.Render("Key: ") + valueStyle.Render("Value")
```

#### 2.6 Remove: `ui/termui/types.go`, `ui/termui/table.go`, `ui/termui/cursor.go`, `ui/termui/stringer.go`

These utility types are either eliminated or replaced by lipgloss/bubbles equivalents.

---

### Phase 3: App Widgets (`appui/` rewrite)

Each widget becomes a Bubbletea sub-model with `Init()`, `Update()`, `View()`.

#### 3.1 Table-based Widgets (Containers, Images, Networks, Volumes, Swarm*)

All table-based widgets follow the same pattern. Use either:

**Option A: `bubbles/table`** — provides built-in navigation, cursor, scrolling.
**Option B: `lipgloss/table`** — pure rendering, manual navigation logic.
**Option C (recommended): Custom table model** — wraps `lipgloss/table` for rendering with custom navigation, sorting, filtering logic.

The custom approach is recommended because:
- The Bubbles table lacks built-in filtering and sorting
- dry's tables need row-level styling (running=green, stopped=red)
- dry's tables need custom column types (gauges for monitor)
- dry needs virtual scrolling with a visible window

**Shared table model structure:**

```go
type tableModel struct {
    columns     []column
    rows        []tableRow
    filtered    []tableRow
    cursor      int
    offset      int      // scroll offset
    width       int
    height      int
    sortField   int
    sortAsc     bool
    filterText  string
    filterInput textinput.Model
    filtering   bool
    focused     bool
    styles      tableStyles
}

type column struct {
    title string
    width int       // 0 = proportional
    fixed bool
}

type tableRow interface {
    Columns() []string
    Style() lipgloss.Style  // row-level style (e.g., running vs stopped)
    ID() string
}
```

**View rendering** uses `lipgloss/table`:
```go
func (m tableModel) View() string {
    t := table.New().
        Headers(headerStrings...).
        StyleFunc(m.styleFunc).
        Width(m.width)
    for _, row := range m.visibleRows() {
        t.Row(row.Columns()...)
    }
    return t.Render()
}
```

**Keybindings** (same as current):
- `Up/k` / `Down/j` — cursor navigation
- `g` / `G` — top / bottom
- `F1` — cycle sort field
- `%` — activate filter input
- `F5` — refresh (send mount command)

#### 3.2 ContainersWidget → `containersModel`

```go
type containersModel struct {
    table      tableModel
    daemon     docker.ContainerDaemon
    showAll    bool
}
```

Replaces: `appui/containers.go`, `appui/container_row.go`

Columns: Status indicator, ID, Image, Command, Status, Ports, Names

Row styling: green for running, red for exited, yellow for paused.

#### 3.3 DockerImagesWidget → `imagesModel`

```go
type imagesModel struct {
    table  tableModel
    daemon docker.ContainerDaemon
}
```

Replaces: `appui/images.go`, `appui/image_row.go`

#### 3.4 DockerNetworksWidget → `networksModel`

Replaces: `appui/networks.go`, `appui/network_row.go`

#### 3.5 VolumesWidget → `volumesModel`

Replaces: `appui/volumes.go`, `appui/volume_row.go`

#### 3.6 Monitor → `monitorModel`

```go
type monitorModel struct {
    table       tableModel
    daemon      docker.ContainerDaemon
    refreshRate time.Duration
    statsChans  map[string]context.CancelFunc
}
```

Replaces: `appui/monitor.go`, `appui/stats_row.go`

**Critical change**: The current monitor has its own `refreshLoop()` goroutine that calls `screen.RenderBufferer()` and `screen.Flush()` directly. In Bubbletea, the monitor cannot self-render — it must go through the standard `Update()`/`View()` cycle.

**View lifecycle**: Since `Init()` runs only once at program startup (not when the user navigates to monitor view), use a view activation message pattern to replace the current `Mount()`/`Unmount()` lifecycle:

```go
// Parent model sends these when switching views
type viewActivatedMsg struct{ view viewMode }
type viewDeactivatedMsg struct{ view viewMode }

// In the parent's Update(), when switching to monitor:
case tea.KeyPressMsg:
    if msg.String() == "m" {
        oldView := m.view
        m.view = Monitor
        return m, tea.Batch(
            func() tea.Msg { return viewDeactivatedMsg{view: oldView} },
            func() tea.Msg { return viewActivatedMsg{view: Monitor} },
        )
    }

// In monitorModel.Update():
case viewActivatedMsg:
    // Start stats channels for each container
    var cmds []tea.Cmd
    for _, container := range m.containers {
        ch := m.daemon.OpenChannel(container)
        cmds = append(cmds, waitForStats(ch, container.ID))
    }
    cmds = append(cmds, m.tickCmd()) // start periodic refresh
    return m, tea.Batch(cmds...)

case viewDeactivatedMsg:
    // Cancel all stats subscriptions
    for id, cancel := range m.statsChans {
        cancel()
        delete(m.statsChans, id)
    }
    return m, nil
```

**Stats flow**:
1. On activation, start one blocking command per container that reads from its stats channel
2. Each stats update arrives as `containerStatsMsg`, model updates its row data
3. A `tea.Tick(refreshRate, ...)` triggers periodic re-renders (re-issued after each tick)
4. On deactivation, cancel all stats contexts so the blocking commands return
5. `View()` renders the table including progress bar columns for CPU/Memory

This pattern applies to all views, not just the monitor — any view that needs to load data on entry should respond to `viewActivatedMsg`.

**CPU/Memory gauges**: Use `progress.Model` from Bubbles or render custom gauge strings with lipgloss (e.g., `[████████░░] 80%`).

#### 3.7 ContainerMenuWidget → `containerMenuModel`

```go
type containerMenuModel struct {
    list   list.Model  // bubbles/list for command selection
    container docker.Container
}
```

Replaces: `appui/container_menu.go`

The Bubbles list component provides navigation, filtering, and styling out of the box. Define menu items implementing `list.DefaultItem`.

#### 3.8 DockerInfo → `headerModel`

```go
type headerModel struct {
    info   types.Info
    styles Theme
    width  int
}
```

Replaces: `appui/docker_info.go`

Renders a bordered info block using `lipgloss` with border and padding. No interactivity, just `View()`.

#### 3.9 DiskUsage → `diskUsageModel`

Replaces: `appui/disk_usage.go`

Currently renders via `tabwriter` + `fmt.Stringer`. Replace with `lipgloss/table` for a better-looking table.

#### 3.10 Prompt → `promptModel`

```go
type promptModel struct {
    textInput textinput.Model
    message   string
    active    bool
    callback  func(string) tea.Cmd
}
```

Replaces: `appui/prompt.go`

Rendered as a centered, bordered overlay using `lipgloss.Place()`.

#### 3.11 ImageRunWidget → integrated into `promptModel`

Same pattern as Prompt but with a wider input field and different placeholder text.

#### 3.12 WidgetHeader → lipgloss inline rendering

Replaces: `appui/header.go`

Simple styled key-value pairs rendered with `lipgloss.JoinHorizontal()`.

#### 3.13 Swarm Widgets (`appui/swarm/`)

Same table model pattern as containers/images/networks:
- `NodesWidget` → `nodesModel`
- `ServicesWidget` → `servicesModel`
- `StacksWidget` → `stacksModel`
- `NodeTasksWidget` → `nodeTasksModel`
- `ServiceTasksWidget` → `serviceTasksModel`
- `StacksTasksWidget` → `stackTasksModel`

Each gets its own column definitions and row type but shares the `tableModel` infrastructure.

**Two files with unique logic require specific attention:**

**`appui/swarm/tasks.go`** — This is a shared base widget (`TasksWidget`) embedded by `ServiceTasksWidget`, `NodeTasksWidget`, and `StacksTasksWidget`. It contains the reusable sorting (4 sort modes with sort indicators in headers), filtering (`filterRows()`), and pagination (`calculateVisibleRows()`) logic for all three task views. In the migration, this shared logic maps directly to the shared `tableModel` — the sort/filter/scroll behavior should be implemented once in `tableModel` and reused by all task sub-models. The three embedding widgets become thin wrappers that define their column layout and data source.

**`appui/swarm/service_info.go`** — This is **not** a table widget. It is a fixed-height (6 lines) detail info panel that renders formatted service metadata using `olekukonko/tablewriter` for key-value layout, wrapped in a `MarkupPar`. It should become a simple `View()` method on the services model (or a helper function) that returns a lipgloss-styled info block with a border. The `tablewriter` formatting can be retained or replaced with `lipgloss/table` for visual consistency. No interactivity needed — this is purely a display component shown above the service tasks list.

#### 3.14 ExpiringMessageWidget → `messageBarModel`

```go
type messageBarModel struct {
    text    string
    expiry  time.Time
    style   lipgloss.Style
    width   int
}
```

Replaces: `ui/expiring_message.go`

On receiving a `statusMessageMsg`, sets text and expiry. `View()` renders if not expired. A `tea.Tick` cmd checks for expiry.

#### 3.15 Less/Stream wrappers (`appui/less.go`, `appui/stream.go`)

These functions create a `Less` viewer with Docker-fetched content (inspect JSON, logs, events). They become commands that:
1. Fetch data from Docker (returns a `tea.Cmd`)
2. On data receipt, populate a `lessModel` and set it as the active overlay

For streaming content (container logs), use a re-issuing command pattern. Each command reads one chunk and returns it as a message; the `Update()` handler re-issues the command to keep reading:

```go
type logLineMsg struct {
    id   string
    line string
}

type logDoneMsg struct{ id string }

func readLogLine(reader io.Reader, id string) tea.Cmd {
    return func() tea.Msg {
        buf := make([]byte, 4096)
        n, err := reader.Read(buf)
        if err != nil {
            return logDoneMsg{id: id}
        }
        return logLineMsg{id: id, line: string(buf[:n])}
    }
}

// In Update():
case logLineMsg:
    m.less.appendContent(msg.line)
    return m, readLogLine(m.logReader, msg.id) // re-issue to keep reading
case logDoneMsg:
    // Stream ended, stop re-issuing
```

Alternatively, use a channel-based approach where the stream goroutine writes to a channel and a blocking command reads from it:

```go
func startLogStream(daemon docker.ContainerDaemon, id string) tea.Cmd {
    return func() tea.Msg {
        reader, err := daemon.Logs(id)
        if err != nil {
            return operationErrorMsg{err}
        }
        ch := make(chan string, 64)
        go func() {
            defer close(ch)
            scanner := bufio.NewScanner(reader)
            for scanner.Scan() {
                ch <- scanner.Text()
            }
        }()
        // Return the channel wrapped in a message; subsequent reads
        // use waitForLogLine(ch) commands
        return logStreamStartedMsg{id: id, ch: ch}
    }
}

// Blocking command that reads one line from the channel
func waitForLogLine(ch <-chan string, id string) tea.Cmd {
    return func() tea.Msg {
        line, ok := <-ch
        if !ok {
            return logDoneMsg{id: id}
        }
        return logLineMsg{id: id, line: line}
    }
}

// In Update():
case logStreamStartedMsg:
    m.logChan = msg.ch
    return m, waitForLogLine(msg.ch, msg.id)
case logLineMsg:
    m.less.appendContent(msg.line)
    return m, waitForLogLine(m.logChan, msg.id) // re-issue
case logDoneMsg:
    m.logChan = nil // stream ended
```

This avoids needing access to `*tea.Program` from inside commands — all communication flows through messages and blocking channel reads, which is the idiomatic Bubbletea pattern.

#### 3.16 Content Renderers (`appui/inspect.go`, `appui/events.go`, `appui/info.go`, `appui/top.go`, `appui/image_history.go`)

These files define renderer functions that produce text content for the Less viewer:

| File | Function | Produces |
|------|----------|----------|
| `appui/inspect.go` | `NewJSONRenderer()` | Pretty-printed JSON of inspected Docker objects |
| `appui/events.go` | `NewDockerEventsRenderer()` | Formatted Docker event stream |
| `appui/info.go` | `NewDockerInfoRenderer()` | Docker daemon info text |
| `appui/top.go` | `NewDockerTopRenderer()` | Container process list |
| `appui/image_history.go` | `DockerImageHistoryRenderer()` | Image layer history |

Currently these return a `fmt.Stringer` or write to an `io.Writer`, and the calling code feeds the output into a `Less` viewer via `appui.Less()` or `appui.Stream()`.

In Bubbletea, these become `tea.Cmd` factories that:
1. Call the Docker daemon to fetch data
2. Format it as a string
3. Return a `showLessMsg{content, title}` that the top-level `Update()` uses to populate and activate the `lessModel` overlay

```go
// Example: inspect becomes a command
func inspectContainerCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
    return func() tea.Msg {
        json, err := daemon.InspectContainer(id)
        if err != nil {
            return operationErrorMsg{err}
        }
        content := formatJSON(json) // pretty-print
        return showLessMsg{content: content, title: "Inspect " + id}
    }
}

// Example: events becomes a streaming command
func dockerEventsCmd(daemon docker.ContainerDaemon) tea.Cmd {
    return func() tea.Msg {
        events := daemon.EventLog()
        content := formatEvents(events)
        return showLessMsg{content: content, title: "Docker Events"}
    }
}
```

The existing formatting logic in these files can be largely preserved — only the wrapper that connects them to the UI changes.

---

### Phase 4: App Core (`app/` rewrite)

#### 4.1 Rewrite: `app/loop.go` → Eliminate

The explicit render loop (`RenderLoop()`) is eliminated entirely. Bubbletea manages the render loop internally.

| Current | Bubbletea |
|---------|-----------|
| `renderChan` goroutine | Automatic after every `Update()` |
| `refreshScreen()` | Return a command or message that triggers `Update()` |
| `refreshIfView(v)` | Check view mode in `Update()` before processing |
| Event polling goroutine | Bubbletea internal |
| Message display goroutine | Handle in `Update()` with timer commands |

#### 4.2 Rewrite: `app/events.go` → Message Dispatch

The `eventHandler` interface chain is replaced by the top-level model's `Update()` method:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        // Handle overlay first (prompt, less, stream)
        if m.overlay != overlayNone {
            return m.updateOverlay(msg)
        }
        // Global keys
        switch msg.String() {
        case "ctrl+c", "Q":
            return m, tea.Quit
        case "?", "h", "H":
            return m.showHelp()
        case "1": m.view = Main
        case "2": m.view = Images
        // ... etc
        case "F7": m.showHeader = !m.showHeader
        case "F8": m.view = DiskUsage
        // ... etc
        }
        // Delegate to active view's sub-model
        return m.updateActiveView(msg)

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        // Propagate to all sub-models
        return m.propagateResize()

    case dockerEventMsg:
        return m.handleDockerEvent(msg)

    // ... other message types
    }
}
```

The `nextHandler` callback pattern (used to swap handlers for modal focus) becomes:
- Set `m.overlay = overlayLess` (or `overlayPrompt`, `overlayStream`)
- The top-level `Update()` checks overlay first and routes messages to the overlay model
- When overlay closes, set `m.overlay = overlayNone`

#### 4.3 Rewrite: `app/container_events.go` through `app/stacktasks_events.go`

Each `*ScreenEventHandler` file becomes a method on its corresponding model:

```go
func (m *containersModel) Update(msg tea.Msg) (containersModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        switch msg.String() {
        case "F1":
            m.table.nextSort()
        case "F2":
            m.showAll = !m.showAll
            return *m, m.loadContainers()
        case "F5":
            return *m, m.loadContainers()
        case "ctrl+k":
            // Return command to show kill confirmation prompt
            return *m, showPromptCmd("Kill container?", m.killContainer)
        case "enter":
            return *m, showContainerMenuCmd(m.selectedContainer())
        case "i", "I":
            return *m, inspectContainerCmd(m.selectedContainer())
        case "l":
            return *m, showLogsCmd(m.selectedContainer())
        // ... etc
        }
    }
    // Forward to table for navigation
    var cmd tea.Cmd
    m.table, cmd = m.table.Update(msg)
    return *m, cmd
}
```

#### 4.4 Rewrite: `app/render.go` → Eliminate

The `render()` function with its view-mode switch becomes the top-level `View()`:

```go
func (m model) View() tea.View {
    // Full-screen overlays (less, stream) replace the main content entirely.
    // Bubbletea View() returns a single string — there is no z-index or layering.
    if m.overlay == overlayLess || m.overlay == overlayStream {
        v := tea.NewView(m.less.View())
        v.AltScreen = true
        return v
    }

    var sections []string

    // Header
    if m.showHeader {
        sections = append(sections, m.header.View())
    }

    // Main content area
    switch m.view {
    case Main:
        sections = append(sections, m.containers.View())
    case Images:
        sections = append(sections, m.images.View())
    case Networks:
        sections = append(sections, m.networks.View())
    case Volumes:
        sections = append(sections, m.volumes.View())
    case Monitor:
        sections = append(sections, m.monitor.View())
    case DiskUsage:
        sections = append(sections, m.diskUsage.View())
    // ... swarm views
    }

    // Footer (keybinding help)
    sections = append(sections, m.renderFooter())

    // Status message bar
    sections = append(sections, m.messageBar.View())

    content := lipgloss.JoinVertical(lipgloss.Left, sections...)

    // Centered prompt overlay: render on top of dimmed main content using lipgloss.Place()
    if m.overlay == overlayPrompt {
        prompt := m.prompt.View()
        content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, prompt,
            lipgloss.WithWhitespaceChars(" "),
            lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Faint(true)),
        )
    }

    v := tea.NewView(content)
    v.AltScreen = true
    return v
}
```

#### 4.5 Rewrite: `app/dry.go`

The `Dry` struct is replaced by the top-level `model`. Key changes:

| Dry field | Model equivalent |
|-----------|-----------------|
| `dockerDaemon` | `daemon` field in model |
| `dockerEvents` / `dockerEventsDone` | Docker event subscription as `tea.Cmd` |
| `output chan string` | `statusMessageMsg` messages |
| `screen *ui.Screen` | Eliminated (Bubbletea manages screen) |
| `view viewMode` | `view` field in model |
| `sync.RWMutex` | Not needed (Bubbletea is single-threaded update loop) |

**Thread safety**: Bubbletea processes messages sequentially in a single goroutine, so the `sync.RWMutex` on `Dry` and widgets is no longer needed. Docker API calls happen in `tea.Cmd` functions (separate goroutines) and return results as messages.

#### 4.6 Rewrite: `app/misc.go`

Helper functions like `inspect()`, `less()`, `stream()` become command factories:

```go
func inspectCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
    return func() tea.Msg {
        content, err := daemon.Inspect(id)
        if err != nil {
            return operationErrorMsg{err}
        }
        return showLessMsg{content: content, title: "Inspect " + id}
    }
}
```

#### 4.7 Rewrite: `app/widget_registry.go` → Eliminate

The `widgetRegistry` singleton is eliminated. All widgets are fields on the top-level model. Docker event-driven refresh is handled via the throttled event dispatch pattern described in Phase 6.1 (debounce `dockerEventMsg` by `SourceType` with 250ms delay, then batch-refresh affected widgets).

#### 4.8 Rewrite: `app/view.go` and handler map

View mode constants remain the same. The `viewsToHandlers` map (declared in `app/events.go`, initialized via `initHandlers()` in `app/dry.go`) is eliminated — routing happens in the top-level `Update()` switch on `m.view`.

---

### Phase 5: Entry Point (`main.go`)

```go
func main() {
    // Parse flags (keep existing flag parsing)
    opts := parseFlags()

    // Build config
    config := app.Config{...}

    // Create initial model
    m := app.NewModel(config)

    // Create and run program
    p := tea.NewProgram(m)

    finalModel, err := p.Run()
    if err != nil {
        log.Fatal(err)
    }
}
```

The loading screen with whale animation becomes the initial `View()` when `m.ready == false`, with `Init()` returning a command that connects to Docker. Once connected, a `dockerConnectedMsg` sets `m.ready = true` and triggers the main UI render.

---

### Phase 6: Docker Layer (`docker/` package)

The `docker/` package requires minimal changes since it's primarily an API abstraction.

#### 6.1 Event Listener Changes

`docker/event_listener.go`: The current `GlobalRegistry` callback pattern dispatches Docker events to registered callbacks by `SourceType` (container, image, network, volume, daemon, service, node, secret, plugin). Callbacks run in goroutines and call `refreshScreen()`. Importantly, the current code throttles refresh at 250ms intervals (`refreshInterval` in `app/dry.go`) to avoid excessive re-renders on busy Docker hosts.

In Bubbletea, Docker events need to flow as messages. Two options:

**Option A (recommended)**: Subscribe via a re-issuing command pattern:
```go
func listenDockerEvents(daemon docker.ContainerDaemon) tea.Cmd {
    return func() tea.Msg {
        event := <-daemon.Events()
        return dockerEventMsg{event}
    }
}

// In Init():
return listenDockerEvents(m.daemon)

// In Update():
case dockerEventMsg:
    cmd := m.handleDockerEvent(msg)
    return m, tea.Batch(cmd, listenDockerEvents(m.daemon)) // re-issue to keep listening
```

**Option B**: Use a channel-based blocking command (avoids needing `*tea.Program` access):
```go
func listenDockerEventsViaChannel(ch <-chan events.Message) tea.Cmd {
    return func() tea.Msg {
        event, ok := <-ch
        if !ok {
            return dockerEventsClosedMsg{}
        }
        return dockerEventMsg{event}
    }
}

// In Init(), start the event channel and issue the first blocking read:
func (m model) Init() tea.Cmd {
    m.eventsChan = m.daemon.Events()
    return listenDockerEventsViaChannel(m.eventsChan)
}

// In Update(), re-issue after each event:
case dockerEventMsg:
    cmd := m.handleDockerEvent(msg)
    return m, tea.Batch(cmd, listenDockerEventsViaChannel(m.eventsChan))
```

**Throttling**: The `GlobalRegistry` callback system can be simplified — instead of registering per-source-type callbacks, the top-level `Update()` handles `dockerEventMsg` and dispatches by `SourceType`. To preserve the 250ms throttle and avoid excessive re-renders on busy Docker hosts, implement debouncing in the model:

```go
type model struct {
    // ...
    pendingRefresh map[docker.SourceType]bool
    refreshTimer   bool // whether a debounce tick is pending
}

case dockerEventMsg:
    source := docker.SourceType(msg.event.Type)
    m.pendingRefresh[source] = true
    if !m.refreshTimer {
        m.refreshTimer = true
        return m, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
            return flushRefreshMsg{}
        })
    }
    return m, nil

case flushRefreshMsg:
    m.refreshTimer = false
    var cmds []tea.Cmd
    for source := range m.pendingRefresh {
        switch source {
        case docker.ContainerSource:
            cmds = append(cmds, m.containers.refresh())
        case docker.ImageSource:
            cmds = append(cmds, m.images.refresh())
        case docker.NetworkSource:
            cmds = append(cmds, m.networks.refresh())
        case docker.VolumeSource:
            cmds = append(cmds, m.volumes.refresh())
        case docker.ServiceSource:
            cmds = append(cmds, m.services.refresh())
        case docker.NodeSource:
            cmds = append(cmds, m.nodes.refresh())
        }
    }
    m.pendingRefresh = make(map[docker.SourceType]bool)
    return m, tea.Batch(cmds...)
```

This preserves the existing throttle behavior while using Bubbletea's message-based architecture.

#### 6.2 Stats Channels

Container stats currently flow through channels to the Monitor's goroutines. In Bubbletea, stats use the same channel-based blocking command pattern:

```go
// Blocking command: reads one stats update from the Docker stats channel
func waitForStats(ch <-chan docker.Stats, containerID string) tea.Cmd {
    return func() tea.Msg {
        stats, ok := <-ch
        if !ok {
            return statsChannelClosedMsg{id: containerID}
        }
        return containerStatsMsg{id: containerID, stats: stats}
    }
}
```

- On `viewActivatedMsg`, the monitor opens a stats channel per container and issues one `waitForStats` command per channel (see section 3.6)
- Each `containerStatsMsg` updates the row data, then re-issues `waitForStats` to keep reading
- On `viewDeactivatedMsg`, canceling the stats contexts closes the channels, causing the blocking commands to return `statsChannelClosedMsg`

---

## File Inventory

### Files to Delete (37 files)

```
ui/screen.go              - tcell screen wrapper
ui/tcell.go               - tcell initialization
ui/termbox.go             - termbox compatibility rendering
ui/events.go              - EventSource struct
ui/render.go              - EventChannel(), ScreenTextRenderer
ui/focus.go               - Focusable interface
ui/key.go                 - Key struct wrapping tcell.Key
ui/input.go               - InputBox (replaced by textinput Bubble)
ui/view.go                - View buffer (replaced by viewport Bubble)
ui/list.go                - Custom list (replaced by list Bubble)
ui/par.go                 - NewPar() termui paragraph helper
ui/termui/widget.go       - Widget interface
ui/termui/row.go          - Row grid layout
ui/termui/header.go       - TableHeader
ui/termui/par_column.go   - ParColumn
ui/termui/gauge_column.go - GaugeColumn
ui/termui/par_markup.go   - MarkupPar
ui/termui/textbuilder.go  - TextBuilder
ui/termui/keyvalue.go     - KeyValuePar
ui/termui/textinput.go    - TextInput
ui/termui/types.go        - SizableBufferer
ui/termui/table.go        - Table interface
ui/termui/cursor.go       - cursor interface
ui/termui/stringer.go     - Buffer to string
app/loop.go               - RenderLoop
app/widget_registry.go    - widgetRegistry singleton
app/render.go             - render() function
appui/screen.go           - Screen and ScreenBuffererRender interfaces (tcell/termui-dependent)
appui/ui_events.go        - AppWidget interface (composes termui.Widget)
appui/input.go            - Input() function (trivial termui wrapper)
appui/monitor_header.go   - MonitorTableHeader (termui-dependent)
appui/row.go              - Base Row type (termui-dependent)
appui/container_row.go    - ContainerRow (termui-dependent)
appui/image_row.go        - ImageRow (termui-dependent)
appui/network_row.go      - NetworkRow (termui-dependent)
appui/volume_row.go       - VolumeRow (termui-dependent)
appui/stats_row.go        - ContainerStatsRow (termui-dependent)
```

Note: All corresponding `*_test.go` files for deleted files should also be deleted.

### Files to Rewrite (50+ files)

```
main.go                        - Entry point
app/dry.go                     - Core struct → model
app/events.go                  - Event handler chain → Update dispatch (also contains viewsToHandlers map)
app/view.go                    - View modes (mostly keep, remove handler map)
app/container_events.go        - → containersModel.Update()
app/image_events.go            - → imagesModel.Update()
app/network_events.go          - → networksModel.Update()
app/volume_events.go           - → volumesModel.Update()
app/monitor_events.go          - → monitorModel.Update()
app/cmenu_events.go            - → containerMenuModel.Update()
app/df_events.go               - → diskUsageModel.Update()
app/service_events.go          - → servicesModel.Update()
app/node_events.go             - → nodesModel.Update()
app/stack_events.go            - → stacksModel.Update()
app/nodetasks_events.go        - → nodeTasksModel.Update()
app/servicetasks_events.go     - → serviceTasksModel.Update()
app/stacktasks_events.go       - → stackTasksModel.Update()
app/misc.go                    - Helper functions → command factories
app/filter_event.go            - showFilterInput() → filter handled in table model
app/help_texts.go              - Help text strings → key.Binding descriptions + help.KeyMap
appui/containers.go            - → containersModel
appui/images.go                - → imagesModel
appui/networks.go              - → networksModel
appui/volumes.go               - → volumesModel
appui/monitor.go               - → monitorModel
appui/container_menu.go        - → containerMenuModel
appui/container_details.go     - ContainerDetailsWidget → lipgloss styled view
appui/docker_info.go           - → headerModel.View()
appui/disk_usage.go            - → diskUsageModel.View()
appui/prompt.go                - → promptModel
appui/header.go                - → inline lipgloss rendering
appui/stream.go                - → streamModel / re-issuing command pattern
appui/less.go                  - → lessModel overlay
appui/events.go                - NewDockerEventsRenderer → command that fetches events for lessModel
appui/info.go                  - NewDockerInfoRenderer → command that fetches info for lessModel
appui/inspect.go               - NewJSONRenderer → command that fetches JSON for lessModel
appui/top.go                   - NewDockerTopRenderer → command that fetches top for lessModel
appui/image_history.go         - DockerImageHistoryRenderer → command for lessModel
appui/run_image.go             - ImageRunWidget → integrated into promptModel
appui/container.go             - NewContainerInfo() → lipgloss styled formatter
appui/row_filter.go            - RowFilter, FilterableRow → integrated into tableModel
appui/ui.go                    - Constants (MainScreenHeaderSize etc.) → adapt for lipgloss layout
appui/testing.go               - testScreen helper → adapt for string-based View() testing
appui/swarm/*.go               - All swarm widgets (12 source files) → swarm models
ui/less.go                     - → lessModel
ui/markup.go                   - → lipgloss-based RenderMarkup()
ui/theme.go                    - → lipgloss Theme
ui/cursor.go                   - Simplify (remove tcell deps)
ui/color.go                    - → lipgloss.Color values
ui/colorize.go                 - → lipgloss helpers
ui/expiring_message.go         - ExpiringMessageWidget → messageBarModel
appui/theme.go                 - → lipgloss Theme
```

Note: All corresponding `*_test.go` files will also need rewriting to test `View()` string output instead of `Buffer()` cell maps.

### Files to Keep (mostly unchanged)

```
app/config.go               - Config struct (no UI dependencies)
docker/                      - Docker abstraction layer (minimal changes)
  api.go                    - ContainerDaemon interface (no changes)
  daemon.go                 - DockerDaemon implementation (no changes)
  container.go              - Container operations (no changes)
  images.go                 - Image operations (no changes)
  network.go                - Network operations (no changes)
  volumes.go                - Volume operations (no changes)
  swarm.go                  - Swarm operations (no changes)
  formatter/                - Docker formatters (no changes)
  event_listener.go         - Adjust callback pattern (see Phase 6)
  mock/                     - Internal mocks (no changes)
docker/container_store.go    - No changes
search/                      - Search functionality (no changes)
terminal/                    - ANSI parser (no changes)
tls/                         - TLS config (no changes)
version/                     - Version vars (no changes)
mocks/                       - DockerDaemonMock (no changes)
appui/appui.go               - invalidRow error type (no UI dependencies)
ui/screen_dimension.go       - Dimensions struct (may simplify, no tcell deps)
```

### New Files to Create

```
app/model.go           - Top-level Bubbletea model
app/messages.go        - Custom tea.Msg types
app/commands.go        - tea.Cmd factories for Docker operations
app/keys.go            - key.Binding definitions + KeyMap implementations
app/overlay.go         - Overlay (less, prompt, stream) management
appui/table_model.go   - Shared table model with sort/filter/scroll
appui/styles.go        - Lipgloss theme and style definitions
appui/gauge.go         - CPU/memory gauge rendering with lipgloss
```

---

## Testing Strategy

### Golden File Tests

All existing golden file tests in `appui/testdata/` and `appui/swarm/testdata/` will need to be regenerated. The output format changes from cell-map-based rendering to string-based rendering. Run with `-update` flag after migration.

### Unit Tests

- Widget model tests: verify `Update()` state transitions for key events
- Widget view tests: verify `View()` output strings (golden file comparison)
- Command tests: verify `tea.Cmd` factories return correct messages
- Table model tests: verify sort, filter, scroll behavior

### Integration Tests

- Use `tea.NewProgram(model, tea.WithoutRenderer(), tea.WithInput(reader))` for headless testing
- Verify full key sequences produce expected view transitions

### Mock Compatibility

The `mocks/docker_daemon.go` mock is unchanged — it implements `docker.ContainerDaemon` which has no UI dependencies.

---

## Migration Order (Recommended)

Execute in this order to maintain a compilable codebase at each step:

1. **Add new dependencies**: `go get` bubbletea v2, bubbles, lipgloss v2
2. **Phase 1**: Rewrite `ui/` — theme, markup-to-lipgloss converter, less model, remove screen/events/tcell
3. **Phase 2**: Rewrite `ui/termui/` — replace with lipgloss-based table/layout utilities
4. **Phase 3**: Rewrite `appui/` widgets — each widget becomes a Bubbletea model
5. **Phase 4**: Rewrite `app/` core — model, update dispatch, view composition
6. **Phase 5**: Rewrite `main.go` — create `tea.Program` and run
7. **Phase 6**: Adjust `docker/` event listener integration
8. **Regenerate golden files** and fix tests
9. **Remove old dependencies**: `go mod tidy`

Note: Because this is a near-total rewrite of the UI layer, it may be more practical to build the new Bubbletea app alongside the old code in a parallel package tree (e.g., `app2/`, `appui2/`, `ui2/`), get it working, then swap. This avoids a long period where the codebase does not compile.

---

## Visual Improvements

With lipgloss, the following visual enhancements become easy:

1. **Rounded borders** on info panels and overlays (`lipgloss.RoundedBorder()`)
2. **Alternating row colors** in tables (`StyleFunc` with row index)
3. **Better gauges** for CPU/memory with gradient fills (`progress.Model` with `WithGradient`)
4. **Styled status indicators** (colored dots or badges for container status)
5. **Centered overlays** with shadow/dim background for prompts and confirmations
6. **Responsive layout** adapting to terminal width via lipgloss `Width()` and `MaxWidth()`
7. **Better help display** using the `help` Bubble with categorized keybindings
8. **Styled search highlights** using `lipgloss.StyleRanges()` for search results in the Less viewer
