# dry

[![MIT License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/moncho/dryblob/master/LICENSE)
[![Build Status](https://travis-ci.org/moncho/dry.svg?branch=master)](https://travis-ci.org/moncho/dry)
![Docker Build](https://github.com/moncho/dry/workflows/.github/workflows/docker.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/moncho/dry)](https://goreportcard.com/report/github.com/moncho/dry)
[![GoDoc](https://godoc.org/github.com/moncho/dry?status.svg)](https://godoc.org/github.com/moncho/dry)
[![Coverage Status](https://coveralls.io/repos/github/moncho/dry/badge.svg?branch=master)](https://coveralls.io/github/moncho/dry?branch=master)
[![Github All Releases](https://img.shields.io/github/downloads/moncho/dry/total.svg)]()
[![Join the chat at https://gitter.im/moncho/dry](https://badges.gitter.im/moncho/dry.svg)](https://gitter.im/moncho/dry?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Release](https://img.shields.io/github/release/moncho/dry.svg?style=flat-square)](https://github.com/moncho/dry/releases/latest)

**Dry** is a terminal application to manage **Docker** and **Docker Swarm**.

It shows information about Containers, Images and Networks, and, if running a **Swarm** cluster, it shows information about Nodes, Service, Stacks and the rest of **Swarm** constructs. It can be used with both local or remote **Docker** daemons.

Besides showing information, it can be used to manage Docker. Most of the commands that the official **Docker CLI** provides, are available in **dry** with the same behaviour. A list of available commands and their keybindings can be found in **dry**'s help screen or in this README.

Lastly, it can also be used as a monitoring tool for **Docker** containers.

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
<kbd>i</kbd>         | inspect
<kbd>l</kbd>         | container logs
<kbd>e</kbd>         | remove
<kbd>s</kbd>         | stats
<kbd>Ctrl+e</kbd>    | remove all stopped containers
<kbd>Ctrl+k</kbd>    | kill
<kbd>Ctrl+l</kbd>    | container logs with Docker timestamps
<kbd>Ctrl+r</kbd>    | start/restart
<kbd>Ctrl+t</kbd>    | stop


#### Image commands

Keybinding           | Description
---------------------|---------------------------------------
<kbd>i</kbd>         | history
<kbd>r</kbd>         | run command in new container
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
<kbd>Ctrl+l</kbd>    | service logs with Docker timestamps
<kbd>Ctrl+r</kbd>    | remove service
<kbd>Ctrl+s</kbd>    | scale service
<kbd>Ctrl+u</kbd>    | update service
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

The easiest way to install the latest binaries for Linux and Mac is to run this in a shell:

```sh
curl -sSf https://moncho.github.io/dry/dryup.sh | sudo sh
sudo chmod 755 /usr/local/bin/dry
```

### Binaries

If you dont like to **curl | sh**, binaries are provided.

* **darwin** [386](https://github.com/moncho/dry/releases/download/v0.11.1/dry-darwin-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.11.1/dry-darwin-amd64)
* **freebsd** [386](https://github.com/moncho/dry/releases/download/v0.11.1/dry-freebsd-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.11.1/dry-freebsd-amd64)
* **linux** [386](https://github.com/moncho/dry/releases/download/v0.11.1/dry-linux-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.11.1/dry-linux-amd64)
* **windows** [386](https://github.com/moncho/dry/releases/download/v0.11.1/dry-windows-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.11.1/dry-windows-amd64)

#### Mac OS X / Homebrew

If you're on OS X and want to use homebrew:

```
brew tap moncho/dry
brew install dry
```

#### Docker

```docker run --rm -it -v /var/run/docker.sock:/var/run/docker.sock -e DOCKER_HOST=$DOCKER_HOST moncho/dry```

#### Arch Linux

```yaourt -S dry-bin```

### Usage

Open a console, type ```dry```. It will try to connect to:

* A Docker host given as a parameter (**-H**).
* if none given, a Docker host defined in the **$DOCKER_HOST** environment variable.
* if not defined, to **unix:///var/run/docker.sock**.

If no connection with a Docker host succeeds, **dry** will exit.

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

* [tcell](https://github.com/gdamore/tcell)
* [termui](https://github.com/gizak/termui)
* [Docker](https://github.com/docker/docker)
* [Docker CLI](github.com/docker/cli/cli)

## Alternatives
See [Awesome Docker list](https://github.com/veggiemonk/awesome-docker/blob/master/README.md#terminal) for similar tools to work with Docker.
