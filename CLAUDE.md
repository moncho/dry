# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**dry** is a terminal UI (TUI) application for managing Docker containers and images, written in Go. It connects to a Docker daemon and provides an interactive interface for monitoring and managing containers, images, networks, volumes, and Docker Swarm resources.

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

Update golden files for widget tests:
```bash
go test ./appui/ -update
go test ./appui/swarm/ -update
```

Version is stored in `APPVERSION` and injected at build time via ldflags into `version.VERSION` and `version.GITCOMMIT`.

## Architecture

### Layered Structure

1. **Entry point** (`main.go`) — CLI flags, screen init, Docker connection, starts render loop
2. **App core** (`app/`) — `Dry` struct, render loop, event handling, view modes
3. **App widgets** (`appui/`) — High-level Docker resource widgets (containers, images, networks, volumes, monitor, swarm)
4. **Docker layer** (`docker/`) — `ContainerDaemon` interface abstracting the Docker API
5. **UI primitives** (`ui/`) — Low-level TUI framework: Screen, Less viewer, InputBox, Cursor, Markup; `ui/termui/` has table primitives (Row, TableHeader, TextInput)

### Core Types and Patterns

**`Dry`** (`app/dry.go`): Central application struct. Holds the Docker daemon, screen, event channels, output channel, and current view mode. Protected by `sync.RWMutex`.

**View modes** (`app/view.go`): 16 modes as `uint16` enum — `Main`, `Images`, `Networks`, `Volumes`, `Monitor`, `DiskUsage`, `Nodes`, `Services`, `Stacks`, etc. The current view determines which widget renders and which event handler is active.

**Widget registry** (`app/widget_registry.go`): Package-level singleton `widgets` holds all persistent widgets. Initialized in `initRegistry()` which also registers Docker event listeners for auto-refresh.

**Render loop** (`app/loop.go`): Runs on the main goroutine. A render goroutine reads from `renderChan`; the main loop reads keyboard events from `ui.EventChannel()` and delegates to the current `eventHandler`. Package-level `refreshScreen()` and `refreshIfView()` functions trigger re-renders.

**Event handler chain** (`app/events.go`): Handlers implement `handle(event, nextHandler)` where `nextHandler` swaps the active handler. `baseEventHandler` handles global keys; view-specific handlers embed it. Modal views (prompts, Less viewer) use `eventHandlerForwarder` to temporarily redirect input.

**Widget lifecycle**: All widgets follow Mount/Unmount — `Mount()` loads data from Docker, `Buffer()` renders to a `termui.Buffer`, `Unmount()` marks for reload. `AppWidget` interface combines `termui.Widget`, `EventableWidget`, `FilterableWidget`, `SortableWidget`.

**Rendering** (`app/render.go`): `render()` switches on view mode, mounts the appropriate widget, collects `Bufferer`s (header, main widget, footer), renders them via `screen.RenderBufferer()`, then renders any active overlay widgets and flushes.

### Docker Integration

**`ContainerDaemon`** (`docker/api.go`): Interface composed of `ContainerAPI`, `ImageAPI`, `NetworkAPI`, `VolumesAPI`, `SwarmAPI`, `ContainerRuntime`.

**Event-driven refresh** (`docker/event_listener.go`): `GlobalRegistry` dispatches Docker events to registered callbacks by source type (container, image, network, etc.), triggering widget unmount/refresh with 250ms throttle.

### Key Conventions

- **Concurrency**: Mutexes protect shared state throughout (Dry, widgets, screen, cursor, container store). Docker operations use 10-second timeouts.
- **Markup**: Custom tag system for colored text (`<white>`, `<blue>`, `<red>`, `<b>`, `<darkgrey>`) parsed by `ui/markup.go`.
- **Theming**: Colors defined in `appui/theme.go` via `DryTheme` (defaults to `Dark256`).
- **Mocks**: `mocks/docker_daemon.go` provides `DockerDaemonMock` with canned data (10 running + 10 stopped containers). `docker/mock/` has additional internal mocks.
- **Golden files**: Widget rendering tests use golden files in `appui/testdata/` and `appui/swarm/testdata/`.
- **Destructive Docker actions** (kill, remove, stop) show a confirmation prompt before executing.
