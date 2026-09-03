# dry

[![MIT License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/moncho/dryblob/master/LICENSE)
![Build Status](https://github.com/moncho/dry/actions/workflows/go.yml/badge.svg)
![Release](https://github.com/moncho/dry/actions/workflows/release.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/moncho/dry)](https://goreportcard.com/report/github.com/moncho/dry)
[![GoDoc](https://godoc.org/github.com/moncho/dry?status.svg)](https://godoc.org/github.com/moncho/dry)
[![Github All Releases](https://img.shields.io/github/downloads/moncho/dry/total.svg)]()
[![Release](https://img.shields.io/github/release/moncho/dry.svg?style=flat-square)](https://github.com/moncho/dry/releases/latest)
[![dry](https://snapcraft.io/dry/badge.svg)](https://snapcraft.io/dry)

**Dry** is a terminal application for **Docker** and **Docker Compose**.

It lets you browse containers, images, networks, and volumes, and manage Compose projects — bring them up, take them down, see what has drifted from the compose file — without leaving the terminal. It can be used with both local or remote **Docker** daemons.

Besides showing information, it can be used to manage Docker. Most of the commands that the official **Docker CLI** provides, are available in **dry** with the same behaviour. A list of available commands and their keybindings can be found in **dry**'s help screen or in this README.

It can also be used as a monitoring tool for **Docker** containers, and, when the daemon is running a **Swarm** cluster, to manage Nodes, Services, and Stacks — see [Docker Swarm](#docker-swarm) below.

**Dry** is installed as a single binary and does not require external libraries.

The demo below shows a **dry** session.

[![asciicast](https://asciinema.org/a/35825.png)](https://asciinema.org/a/35825?autoplay=1&speed=1.5)

## **dry** keybinds

### Global

Keybinding           | Description
---------------------|---------------------------------------
<kbd>%</kbd>         | filter list
<kbd>F1</kbd>        | sort list
<kbd>F5</kbd>        | refresh list
<kbd>F7</kbd>        | toggle showing Docker daemon information
<kbd>F8</kbd>        | show docker disk usage
<kbd>F9</kbd>        | show last 10 docker events
<kbd>F10</kbd>       | show docker info
<kbd>1</kbd>         | show container list
<kbd>2</kbd>         | show image list
<kbd>3</kbd>         | show network list
<kbd>4</kbd>         | show volumes list
<kbd>5</kbd>         | show node list (on Swarm mode)
<kbd>6</kbd>         | show service list (on Swarm mode)
<kbd>7</kbd>         | show stacks list (on Swarm mode)
<kbd>8</kbd>         | show compose projects list
<kbd>ArrowUp</kbd> or <kbd>k</kbd> | move the cursor one line up
<kbd>ArrowDown</kbd> or <kbd>j</kbd> | move the cursor one line down
<kbd>g</kbd>         | move the cursor to the top
<kbd>G</kbd>         | move the cursor to the bottom
<kbd>:</kbd>         | open command palette
<kbd>Space</kbd>     | open Quick Peek for the current selection
<kbd>Ctrl+0</kbd>   | cycle color theme (dark/light)
<kbd>q</kbd>         | quit dry

### Workspace mode (`--workspace`)

`dry --workspace` enables the Phase 1 workspace shell. This keeps the current list view visible together with a passive context pane and a bottom activity pane.

- `Tab` and `Shift+Tab` switch focus between `Navigator`, `Context`, and `Activity` in the full workspace layout.
- When `Context` is focused, you can scroll it with the usual navigation keys.
- `p` / `P` pins or unpins the current preview.
- `:` opens the command palette with global and context-aware actions.
- `Space` opens `Quick Peek`, a temporary side panel with recent logs or inspect/details for the current selection.
- `f` toggles follow mode in the activity pane.
- Existing primary actions stay on their original keys. For example, `Enter` still opens the container command menu and still inspects resources in views where `Enter` already meant inspect/drill-down.
- Embedded activity logs start from a recent tail instead of replaying the full log history.
- On narrow or short terminals, workspace mode falls back to a compact single-pane layout and lets `Tab` switch between navigator and activity.


#### Container commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>Enter</kbd>     | show container command menu (includes Attach for running containers)
<kbd>F2</kbd>        | toggle on/off showing stopped containers
<kbd>i</kbd>         | inspect
<kbd>l</kbd>         | container logs
<kbd>e</kbd>         | remove
<kbd>s</kbd>         | stats
<kbd>x</kbd>         | exec a command in the selected container (default `/bin/sh`)
<kbd>Ctrl+e</kbd>    | remove all stopped containers
<kbd>Ctrl+k</kbd>    | kill
<kbd>Ctrl+r</kbd>    | start/restart
<kbd>Ctrl+t</kbd>    | stop


#### Image commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>i</kbd>         | history
<kbd>Ctrl+d</kbd>    | remove dangling images
<kbd>Ctrl+e</kbd>    | remove image
<kbd>Ctrl+f</kbd>    | remove image (force)
<kbd>Ctrl+u</kbd>    | remove unused images
<kbd>Enter</kbd>     | inspect

#### Network commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>Ctrl+e</kbd>    | remove network
<kbd>Enter</kbd>     | inspect

#### Volume commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>Ctrl+a</kbd>    | remove all volumes
<kbd>Ctrl+e</kbd>    | remove volume
<kbd>Ctrl+f</kbd>    | remove volume (force)
<kbd>Ctrl+u</kbd>    | remove unused volumes
<kbd>Enter</kbd>     | inspect

#### Service commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>i</kbd>         | inspect service
<kbd>l</kbd>         | service logs
<kbd>Ctrl+r</kbd>    | remove service
<kbd>Ctrl+s</kbd>    | scale service
<kbd>Ctrl+u</kbd>    | update service
<kbd>Enter</kbd>     | show service tasks

#### Compose Projects commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>Enter</kbd>     | show project services
<kbd>u</kbd>         | bring the selected project (or service) up
<kbd>d</kbd>         | take the selected project down
<kbd>c</kbd>         | show the project's rendered compose configuration
<kbd>l</kbd>         | project logs
<kbd>Ctrl+t</kbd>    | stop project containers
<kbd>Ctrl+r</kbd>    | restart project containers
<kbd>Ctrl+e</kbd>    | remove project containers

#### Compose Services commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>Enter</kbd>     | inspect service
<kbd>Esc</kbd>       | back to projects
<kbd>u</kbd>         | bring the selected service up
<kbd>c</kbd>         | show the project's rendered compose configuration
<kbd>l</kbd>         | service logs
<kbd>Ctrl+s</kbd>    | start service containers
<kbd>Ctrl+t</kbd>    | stop service containers
<kbd>Ctrl+r</kbd>    | restart service containers
<kbd>Ctrl+e</kbd>    | remove service containers

#### Moving around buffers

Keybinding           | Description
---------------------|---------------------------------------
<kbd>ArrowUp</kbd> or <kbd>k</kbd> | move the cursor one line up
<kbd>ArrowDown</kbd> or <kbd>j</kbd> | move the cursor one line down
<kbd>g</kbd>         | move the cursor to the beginning of the buffer
<kbd>G</kbd>         | move the cursor to the end of the buffer
<kbd>n</kbd>         | after search, move forwards to the next search hit
<kbd>N</kbd>         | after search, move backwards to the previous search hit
<kbd>s</kbd>         | search
<kbd>pg up</kbd>     | move the cursor "screen size" lines up
<kbd>pg down</kbd>   | move the cursor "screen size" lines down

## Installation

The easiest way to install the latest binaries for Linux and Mac is to run this in a shell:

```sh
curl -sSf https://moncho.github.io/dry/dryup.sh | sudo sh
sudo chmod 755 /usr/local/bin/dry
```

### Binaries

If you dont like to **curl | sh**, binaries are provided.

* **darwin** [amd64](https://github.com/moncho/dry/releases/latest/download/dry-darwin-amd64) / [arm64](https://github.com/moncho/dry/releases/latest/download/dry-darwin-arm64)
* **freebsd** [386](https://github.com/moncho/dry/releases/latest/download/dry-freebsd-386) / [amd64](https://github.com/moncho/dry/releases/latest/download/dry-freebsd-amd64)
* **linux** [386](https://github.com/moncho/dry/releases/latest/download/dry-linux-386) / [amd64](https://github.com/moncho/dry/releases/latest/download/dry-linux-amd64) / [arm64](https://github.com/moncho/dry/releases/latest/download/dry-linux-arm64) / [armv6](https://github.com/moncho/dry/releases/latest/download/dry-linux-armv6) / [armv7](https://github.com/moncho/dry/releases/latest/download/dry-linux-armv7)
* **windows** [amd64](https://github.com/moncho/dry/releases/latest/download/dry-windows-amd64)

#### Mac OS X / Homebrew

If you're on OS X and want to use homebrew:

```
brew tap moncho/dry
brew install dry
```

#### Docker

```docker run --rm -it -v /var/run/docker.sock:/var/run/docker.sock -e DOCKER_HOST=$DOCKER_HOST moncho/dry```

#### Arch Linux

```yay -S dry-bin```

### Usage

Open a console, type ```dry```. It will try to connect to:

* A Docker host given as a parameter (**-H**).
* if none given, a Docker host defined in the **$DOCKER_HOST** environment variable.
* if not defined, to **unix:///var/run/docker.sock**.

**dry** does not read `docker context`, so with Docker Desktop, colima or
Rancher Desktop, whose active context is not that socket, name the host with
**-H** or **$DOCKER_HOST**. The same value is what **dry** hands the
`docker compose` plugin, so the two always agree about which daemon they are
talking to.

If no connection with a Docker host succeeds, **dry** will exit.

#### Connecting over SSH

**dry** can talk to a remote Docker daemon through SSH, the same way the
docker CLI does:

```
DOCKER_HOST=ssh://user@host dry
```

Host, port, and user are resolved like the `ssh` command resolves them:
`Hostname`, `Port`, `User`, and `IdentityFile` directives from
`~/.ssh/config` are honored, the port defaults to 22, the user defaults to
the current OS user, and the remote Docker socket defaults to
`/var/run/docker.sock` (a different socket path can be given in the URL,
e.g. `ssh://user@host:2222/run/user/1000/docker.sock`).

Authentication tries, in order: the identity files configured for the host
in `~/.ssh/config` (or, when none is configured, the `~/.ssh/id_*` keys),
keys held by a running SSH agent (`SSH_AUTH_SOCK`), and a password given
in the URL. Passphrase protected key files are used through the agent.

The remote host key is verified against `~/.ssh/known_hosts` and
`/etc/ssh/ssh_known_hosts`, and the connection fails if the host is
unknown or its key does not match. Connect once with `ssh host` to record
the key. Verification can be disabled with
`DRY_SSH_INSECURE_SKIP_HOST_KEY_CHECK=1`, which leaves the connection open
to interception and should only be used against hosts you fully control.

```dry -T light``` launches dry with the light color theme. Available themes: `dark` (default), `light`.

```dry --workspace``` launches the experimental Phase 1 workspace layout.

```dry -p``` launches dry with [pprof](https://golang.org/pkg/net/http/pprof/) package active.

### Docker Compose

Compose projects show up in their own view (key <kbd>8</kbd>), each project row followed by its services indented under it; pressing <kbd>Enter</kbd> on a project opens its services, networks and volumes in the Compose Services view.

Keybinding       | Description
-----------------|---------------------------------------
<kbd>u</kbd>      | bring the selected project or service up
<kbd>d</kbd>      | take the selected project down (behind a confirmation prompt)
<kbd>c</kbd>      | show the project's rendered compose configuration

`u` and `c` work in both the Compose Projects and Compose Services views; `d` works in Compose Projects only.

Projects are discovered two ways: from container labels, including stopped
containers, and from a compose file in the directory **dry** was started in.
A project known only from a file is listed with no running containers, and
pressing `u` brings it up.

Everything compose-specific needs the `docker compose` CLI plugin: the
`u`/`d`/`c` keys, the SYNC column and that directory scan. Without it the
views still list projects found from container labels, and **dry** probes
once at startup, so installing it needs a restart. An empty Compose view has
three causes, and these tell them apart:

* `docker compose version` fails: no plugin, so no scan either.
* the directory holds none of `compose.yaml`, `compose.yml`,
  `docker-compose.yml`, `docker-compose.yaml`. The scan looks in the
  directory **dry** started in, never below it.
* `docker compose config` in that directory fails. A file that does not
  resolve, a YAML error or a required variable with no value, is dropped
  without a message.

The scan passes the one file it picked, so an override file beside it is not
applied and `u` on a scanned project creates something different from
`docker compose up` in the same directory. It matches an existing project by
name, and compose takes that name from the directory's basename unless the
file sets `name:`. A project brought up with `-p` therefore needs
`COMPOSE_PROJECT_NAME=<name> dry` to match, or the scan lists it a second
time under the directory's name.

The SYNC column reports whether a service's running containers match its
compose file:

Label   | Meaning
--------|-----------------------------------------------------------------
`ok`    | the containers were created from the compose file as it is now
`drift` | the compose file changed since the containers were created, the next `u` recreates them

A blank cell is none of those. SYNC is per service, so it is always empty on
a project row, and on the section, network and volume rows of the Compose
Services view. On a service row it means one of:

* no plugin, or the check has not run yet;
* the project's compose file is not on this machine, see below;
* the first check for this project failed. The message bar names it once,
  and a later identical failure keeps the label it already had rather than
  banners again;
* the file no longer defines a service whose containers are still running;
* the file defines it behind a profile, which reports no hash. Start with
  `COMPOSE_PROFILES=dev dry` to fix both that and what `u` brings up.

<kbd>F5</kbd> re-runs the check: the directory scan and every project's drift
in Compose Projects, the selected project's resources and drift in Compose
Services. A refresh that arrives while a check is already running is deferred
rather than dropped, so it lands once that one finishes.

The command palette (<kbd>:</kbd>) adds Force Recreate, on a service row in
the Compose Services view: it recreates that service's containers even when
nothing has drifted, which `u` would skip.

#### Compose files and the machine **dry** runs on

`u`, `c` and Force Recreate have to read the compose file, and the plugin
reads it from the filesystem **dry** runs on. The paths come from the
project's `com.docker.compose.project.config_files` label, written by
whichever machine ran `compose up`, or from the directory scan. Every
recorded path has to be absolute and present there; when one is not, **dry**
says `Project <name> has no compose file on this host` and refuses before
running compose.

`docker compose ls -a` prints each project's recorded files, which shows
which of three cases you are in. (It talks to whatever your shell's
`docker context` or `DOCKER_HOST` points at, which is not necessarily the
daemon **dry** is connected to.)

* **No path recorded**, from an older Compose: start **dry** in the
  project's directory and the scan fills in the file it finds.
* **A relative path**, which an older Compose recorded as given: bring the
  project down and up again from its directory.
* **An absolute path that is not there**: the project was brought up on
  another machine, or the file has since moved. A project brought up from a
  different directory *on this machine* is not this case, its recorded path
  is absolute and still valid.

Either put the files back where the label says, or run **dry** on the machine
that has them. To do the latter, name the daemon with `DOCKER_HOST` or `-H`:
both reach the plugin. **dry** does not read `docker context` and always
passes the plugin a `DOCKER_HOST`, so with Docker Desktop or colima name the
host explicitly rather than relying on the context.

To put the files back, recreate the project directory too, not just the
compose file: **dry** passes the project's
`com.docker.compose.project.working_dir` label as `--project-directory`, and
relative `build:` contexts and bind-mount sources resolve under it. That
label is printed by:

```sh
docker inspect $(docker ps -aq --filter label=com.docker.compose.project=<name> | head -1) \
  --format '{{index .Config.Labels "com.docker.compose.project.working_dir"}}'
```

The paths are re-checked on every invocation, so the keys start working the
moment the files are in place.

`d` (down) needs the plugin but no file, because compose removes a project by
its container labels, so it keeps working when the paths do not. The `Ctrl+`
lifecycle keys need neither: they act on containers through the Docker API,
in [Compose Projects](#compose-projects-commands) and
[Compose Services](#compose-services-commands).

### Docker Swarm

**dry** also works with Docker Swarm. When the connected daemon reports an active swarm, three extra views become available: Nodes (<kbd>5</kbd>), Services (<kbd>6</kbd>), and Stacks (<kbd>7</kbd>). Without an active swarm these views, their keybindings, and their command-palette entries are hidden.

### Contributing

All contributions are welcome.

* Fork the project.
* Make changes on a topic branch.
* Pull request.

## Copyright and license

Code released under the MIT license. See
[LICENSE](https://github.com/moncho/dry/blob/master/LICENSE) for the full license text.

## Credits

Built on top of:

* [Bubbletea](https://github.com/charmbracelet/bubbletea)
* [Bubbles](https://github.com/charmbracelet/bubbles)
* [Lipgloss](https://github.com/charmbracelet/lipgloss)
* [Docker](https://github.com/docker/docker)
* [Docker CLI](https://github.com/docker/cli)

## Alternatives
See [Awesome Docker list](https://github.com/veggiemonk/awesome-docker/blob/master/README.md#terminal) for similar tools to work with Docker.
