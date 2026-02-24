# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**dry** is a terminal UI (TUI) application for managing Docker containers and images, written in Go. It connects to a Docker daemon and provides an interactive interface for monitoring and managing containers, images, networks, volumes, and Docker Swarm resources. The UI is built with Bubbletea v2 (`charm.land/bubbletea/v2`).

## Build & Development Commands

```bash
make build          # Build binary
make run            # Run from source (go run ./main.go)
make install        # Install to $GOPATH/bin
make test           # Run tests with coverage (excludes vendor/mock dirs)
make lint           # Run revive, gofmt, misspell
make fmt            # Auto-format Go files
make benchmark      # Run benchmarks
```

Run a single test:
```bash
go test -v -run TestName ./path/to/package/
```

Version is stored in `APPVERSION` and injected at build time via ldflags into `version.VERSION` and `version.GITCOMMIT`.

## Architecture

### Layered Structure

1. **Entry point** (`main.go`) — CLI flags, config, creates `app.NewModel(cfg)` and runs `tea.NewProgram(m).Run()`
2. **App core** (`app/`) — Top-level Bubbletea model (`model`), commands, messages, key bindings, view modes
3. **App widgets** (`appui/`) — Sub-models for each view: containers, images, networks, volumes, monitor, disk usage, plus shared TableModel and overlay models (less, prompt, container menu)
4. **Swarm widgets** (`appui/swarm/`) — Sub-models for nodes, services, stacks, tasks
5. **Docker layer** (`docker/`) — `ContainerDaemon` interface abstracting the Docker API
6. **UI primitives** (`ui/`) — Markup parser, colorize helpers, theme struct, color constants

### Bubbletea Elm Architecture

The app follows Bubbletea's Elm pattern: `Init() → Cmd`, `Update(Msg) → (Model, Cmd)`, `View() → View`.

**`model`** (`app/model.go`): The single top-level `tea.Model`. Holds all sub-models, the Docker daemon, overlay state, and view mode. Verified with `var _ tea.Model = model{}`.

**View modes** (`app/view.go`): 12 modes as `viewMode` enum — `Main`, `Images`, `Networks`, `Volumes`, `DiskUsage`, `Monitor`, `Nodes`, `Services`, `Stacks`, `ServiceTasks`, `StackTasks`, `Tasks`.

**Message flow** (`app/messages.go`): All custom `tea.Msg` types. Docker operations run in `tea.Cmd` functions and return result messages (e.g., `containersLoadedMsg`, `operationSuccessMsg`).

**Commands** (`app/commands.go`): `tea.Cmd` factories that bridge UI ↔ docker/ — `loadContainersCmd`, `inspectContainerCmd`, `showContainerLogsCmd`, etc.

### Sub-Model Pattern

Each view has a sub-model in `appui/` with `Update(msg) → (Model, Cmd)` and `View() → string`. The top-level model delegates key events to the active sub-model. Sub-models do NOT implement `Init()` — they're not full `tea.Model`s.

**`TableModel`** (`appui/table_model.go`): Shared reusable table with cursor navigation (j/k/g/G/PgUp/PgDown), column sorting, text filtering, scroll windowing. Used by every list view.

**Overlay system**: `overlayType` enum determines which overlay is active:
- `overlayLess` — Scrollable text viewer (help, logs, inspect, events, info)
- `overlayPrompt` — y/N confirmation for destructive actions
- `overlayInputPrompt` — Text input (e.g., service scale)
- `overlayContainerMenu` — Command menu for selected container

### Docker Integration

**`ContainerDaemon`** (`docker/api.go`): Interface composed of `ContainerAPI`, `ImageAPI`, `NetworkAPI`, `VolumesAPI`, `SwarmAPI`, `ContainerRuntime`.

**Event-driven refresh**: Docker events arrive via `listenDockerEvents` command, are throttled with a 250ms debounce (`pendingRefresh` map + `flushRefreshMsg`), then trigger data reload for the active view.

### Key Conventions

- **Import paths**: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`
- **Key events**: `tea.KeyPressMsg` — `.String()` returns key names (e.g., `"esc"` not `"escape"`, `"f1"` not `"F1"`)
- **Markup**: Custom tag system for colored text (`<white>`, `<blue>`, `<red>`, `<b>`, `<darkgrey>`) parsed by `ui/markup.go`, rendered via lipgloss
- **Theming**: Colors defined in `appui/theme.go` via `DryTheme`; styles in `appui/styles.go`
- **Mocks**: `mocks/docker_daemon.go` provides `DockerDaemonMock` with canned data (10 running + 10 stopped containers)
- **Destructive Docker actions** (kill, remove, stop) show a confirmation prompt before executing
- **ID safety**: Use `shortID(id)` helper in `app/commands.go` for safe truncation to 12 chars
