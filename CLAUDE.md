# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**dry** is a terminal-based Docker management application written in Go. It provides a TUI (Terminal User Interface) for managing Docker containers, images, networks, volumes, and Docker Swarm resources. The application works with both local and remote Docker daemons and is distributed as a single binary.

## Development Commands

### Build & Run
- `make run` - Run the application directly
- `make build` - Build binary locally
- `make install` - Install to GOPATH

### Testing & Quality
- `make test` - Run tests with coverage (excludes vendor and mock packages)
- `make benchmark` - Run benchmark tests
- `make lint` - Run complete code quality checks (revive, gofmt, misspell)
- `make fmt` - Format code using gofmt

### Cross-Platform Building
- `make cross` - Cross-compile for all supported platforms
- `make release` - Build release binaries for distribution

## Architecture

### Core Package Structure
- **`main.go`** - Entry point with CLI parsing and app initialization
- **`app/`** - Application coordination layer and event handling logic
- **`docker/`** - Docker API abstraction and daemon communication
- **`ui/`** - Low-level terminal UI primitives and screen management  
- **`appui/`** - High-level UI widgets and view components
- **`version/`** - Version information (populated at build time)

### Key Architectural Patterns

**Interface-Based Design**: The codebase uses `ContainerDaemon` interface to abstract Docker API interactions, with sub-interfaces for different resource types (ContainerAPI, ImageAPI, NetworkAPI, etc.). This enables testing with mocks in the `mocks/` directory.

**Event-Driven Architecture**: Docker events are streamed through `docker.GlobalRegistry` and trigger UI refreshes. The system uses refresh throttling to prevent excessive updates while maintaining real-time responsiveness.

**Widget-Based UI**: Each view (containers, images, networks, volumes, Swarm resources) is implemented as a separate widget registered in a widget registry. The UI layer separates low-level terminal primitives (`ui/`) from high-level components (`appui/`).

**Layered Architecture**: Clear separation between Docker API (`docker/`), application logic (`app/`), and UI layers (`ui/`, `appui/`), with minimal cross-layer dependencies.

### Testing Strategy
- Interface mocking for Docker API interactions
- Golden file testing for UI components (see `appui/testdata/`)
- Unit tests focus on core business logic
- Benchmark tests for performance-critical paths

### Cross-Platform Support
- Builds for: darwin, freebsd, linux, windows
- Architectures: amd64, 386, arm, arm64
- Static linking for portable distribution
- TLS/SSH support for secure remote Docker connections

## Key Dependencies
- `github.com/docker/docker` - Official Docker client library
- `github.com/gdamore/tcell` - Terminal cell manipulation
- `github.com/gizak/termui` - Terminal UI widgets
- `github.com/jessevdk/go-flags` - CLI flag parsing
- `github.com/sirupsen/logrus` - Structured logging

## Development Notes
- Go version 1.23+ required
- Uses `revive.toml` for linting configuration
- Version information is injected at build time via ldflags
- The application supports profiling with `-p` flag for performance analysis