# dry

[![Join the chat at https://gitter.im/moncho/dry](https://badges.gitter.im/moncho/dry.svg)](https://gitter.im/moncho/dry?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![MIT License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/moncho/dryblob/master/LICENSE)
[![wercker status](https://app.wercker.com/status/66c3ab71a46c0c8841f34a526fc23189/s/master "wercker status")](https://app.wercker.com/project/bykey/66c3ab71a46c0c8841f34a526fc23189)
[![Go Report Card](https://goreportcard.com/badge/github.com/moncho/dry)](https://goreportcard.com/report/github.com/moncho/dry)
[![GoDoc](https://godoc.org/github.com/moncho/dry?status.svg)](https://godoc.org/github.com/moncho/dry)
[![Coverage Status](https://coveralls.io/repos/github/moncho/dry/badge.svg)](https://coveralls.io/github/moncho/dry)

**dry** is a terminal application to manage **Docker** containers and images. It aims to be an alternative to the official **Docker CLI** when it is needed to repeatedly execute commands on existing containers and images, and also as a tool to monitor **Docker** containers from a terminal.

The demo below shows a **dry** session.

[![asciicast](https://asciinema.org/a/35825.png)](https://asciinema.org/a/35825?autoplay=1&speed=1.5)

Upon connecting to a **Docker** host (local or remote), the main screen will show the list of containers and version information as reported by the **Docker Engine**. At all times, information about containers, images and networks shown by **dry** is up-to-date.

Most of the commands that can be executed with the official **Docker CLI** on containers, images and networks are available in **dry**, with the exact same behaviour. A list of available commands and their keybinds can be found in **dry** help screen.

Besides this, it:

* Shows real-time information about containers.
* Can sort the container, image and network lists.
* Can navigate and search the output of ***info***, ***inspect*** and ***logs*** commands.
* Makes easier to cleanup old images and containers.

## **dry** keybinds

### Global

```
[F1]        sort list
[F5]        refresh list
[F8]        show docker disk usage
[F9]        show last 10 docker events
[F10]       show docker info
[1]         show container list
[2]         show image list
[3]         show network list
[ArrowUp]   move the cursor one line up
[ArrowDown] move the cursor one line down
[g]         move the cursor to the top
[G]         move the cursor to the bottom
[q]         quit dry
```

#### Container commands

```
[Enter]     show container command menu
[F2]        toggle on/off showing stopped containers
[F3]        filter containers
[i]         inspect
[Ctrl]+[k]  kill
[l]         logs
[e]         remove
[Ctrl]+[e]  remove all stopped containers
[Ctrl]+[r]  start/restart
[s]         stats
[Ctrl]+[t]  stop
```

#### Image commands

```
[i]         history
[Ctrl]+[d]    remove dangling images
[Ctrl]+[e]    remove image
[Ctrl]+[f]    remove image (force)
[Enter]     inspect
```

#### Network commands

```
[Ctrl]+[e]    remove network
[Enter]     inspect
```

#### Moving around buffers

```
[ArrowUp]   move the cursor one line up
[ArrowDown] move the cursor one line down
[g]         move the cursor to the beginning of the buffer
[G]         move the cursor to the end of the buffer
[n]         after search, move forwards to the next search hit
[N]         after search, move backwards to the previous search hit
[s]         search
[pg up]     move the cursor "screen size" lines up
[pg down]   move the cursor "screen size" lines down
```

## Installation

The easiest way to install the latest binaries for Linux and Mac is to run this in your shell:

```$ curl -sSf https://moncho.github.io/dry/dryup.sh | sudo sh```

### Binaries

If you dont like to **curl | sh**, binaries are provided.

* **darwin** [386](https://github.com/moncho/dry/releases/download/v0.7-beta.3/dry-darwin-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.7-beta.3/dry-darwin-amd64)
* **freebsd** [386](https://github.com/moncho/dry/releases/download/v0.7-beta.3/dry-freebsd-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.7-beta.3/dry-freebsd-amd64)
* **linux** [386](https://github.com/moncho/dry/releases/download/v0.7-beta.3/dry-linux-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.7-beta.3/dry-linux-amd64)
* **windows** [386](https://github.com/moncho/dry/releases/download/v0.7-beta.3/dry-windows-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.7-beta.3/dry-windows-amd64)

#### Mac OS X / Homebrew

If you're on OS X and want to use homebrew:

```
brew tap moncho/dry
brew install dry
```

#### Docker

```docker run -it -v  /var/run/docker.sock:/var/run/docker.sock moncho/dry ```

#### Arch Linux

```yaourt -S dry-bin```

### Usage

Open a console, type ```dry```. It will try to connect to:

* A Docker host given as a parameter (**-H**).
* if none given, a Docker host defined in the **$DOCKER_HOST** environment variable.
* if not defined, to **unix:///var/run/docker.sock**.

If no connection with a Docker host succeeds, **dry** will exit immediately.

```dry -p``` launches dry with [pprof](https://golang.org/pkg/net/http/pprof/) package active.

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

* [termbox](https://github.com/nsf/termbox-go)
* [termui](https://github.com/gizak/termui)
* [Docker engine-api](https://github.com/docker/engine-api)

Also reused some code and ideas from the [Docker project](https://github.com/docker/docker).
