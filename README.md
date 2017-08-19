# dry

[![MIT License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/moncho/dryblob/master/LICENSE)
[![wercker status](https://app.wercker.com/status/66c3ab71a46c0c8841f34a526fc23189/s/master "wercker status")](https://app.wercker.com/project/bykey/66c3ab71a46c0c8841f34a526fc23189)
[![Go Report Card](https://goreportcard.com/badge/github.com/moncho/dry)](https://goreportcard.com/report/github.com/moncho/dry)
[![GoDoc](https://godoc.org/github.com/moncho/dry?status.svg)](https://godoc.org/github.com/moncho/dry)
[![Coverage Status](https://coveralls.io/repos/github/moncho/dry/badge.svg?branch=master)](https://coveralls.io/github/moncho/dry?branch=master)
[![Github All Releases](https://img.shields.io/github/downloads/moncho/dry/total.svg)]()
[![Join the chat at https://gitter.im/moncho/dry](https://badges.gitter.im/moncho/dry.svg)](https://gitter.im/moncho/dry?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

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

Keybinding           | Description
---------------------|---------------------------------------
<kbd>F1</kbd>        | sort list
<kbd>F5</kbd>        | refresh list
<kbd>F8</kbd>        | show docker disk usage
<kbd>F9</kbd>        | show last 10 docker events
<kbd>F10</kbd>       | show docker info
<kbd>1</kbd>         | show container list
<kbd>2</kbd>         | show image list
<kbd>3</kbd>         | show network list
<kbd>4</kbd>         | show node list (on Swarm mode)
<kbd>5</kbd>         | show service list (on Swarm mode)
<kbd>ArrowUp</kbd>   | move the cursor one line up
<kbd>ArrowDown</kbd> | move the cursor one line down
<kbd>g</kbd>         | move the cursor to the top
<kbd>G</kbd>         | move the cursor to the bottom
<kbd>q</kbd>         | quit dry


#### Container commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>Enter</kbd>     | show container command menu
<kbd>F2</kbd>        | toggle on/off showing stopped containers
<kbd>F3</kbd>        | filter containers
<kbd>i</kbd>         | inspect
<kbd>l</kbd>         | container logs
<kbd>e</kbd>         | remove
<kbd>s</kbd>         | stats
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
<kbd>Enter</kbd>     | inspect


#### Network commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>Ctrl+e</kbd>    | remove network
<kbd>Enter</kbd>     | inspect

#### Service commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>l</kbd>         | service logs
<kbd>Ctrl+r</kbd>    | remove service
<kbd>Enter</kbd>     | show service tasks


#### Moving around buffers

Keybinding           | Description
---------------------|---------------------------------------
<kbd>ArrowUp</kbd>   | move the cursor one line up
<kbd>ArrowDown</kbd> | move the cursor one line down
<kbd>g</kbd>         | move the cursor to the beginning of the buffer
<kbd>G</kbd>         | move the cursor to the end of the buffer
<kbd>n</kbd>         | after search, move forwards to the next search hit
<kbd>N</kbd>         | after search, move backwards to the previous search hit
<kbd>s</kbd>         | search
<kbd>pg up</kbd>     | move the cursor "screen size" lines up
<kbd>pg down</kbd>   | move the cursor "screen size" lines down


## Installation

The easiest way to install the latest binaries for Linux and Mac is to run this in your shell:

```curl -sSf https://moncho.github.io/dry/dryup.sh | sudo sh```

### Binaries

If you dont like to **curl | sh**, binaries are provided.

* **darwin** [386](https://github.com/moncho/dry/releases/download/v0.8-beta.3/dry-darwin-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.8-beta.3/dry-darwin-amd64)
* **freebsd** [386](https://github.com/moncho/dry/releases/download/v0.8-beta.3/dry-freebsd-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.8-beta.3/dry-freebsd-amd64)
* **linux** [386](https://github.com/moncho/dry/releases/download/v0.8-beta.3/dry-linux-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.8-beta.3/dry-linux-amd64)
* **windows** [386](https://github.com/moncho/dry/releases/download/v0.8-beta.3/dry-windows-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.8-beta.3/dry-windows-amd64)

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
