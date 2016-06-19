# dry

[![Join the chat at https://gitter.im/moncho/dry](https://badges.gitter.im/moncho/dry.svg)](https://gitter.im/moncho/dry?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![License MIT](https://img.shields.io/badge/license-MIT-lightgrey.svg?style=flat)](https://github.com/moncho/dry#license-mit)
[![wercker status](https://app.wercker.com/status/66c3ab71a46c0c8841f34a526fc23189/s/master "wercker status")](https://app.wercker.com/project/bykey/66c3ab71a46c0c8841f34a526fc23189)
[![Go Report Card](https://goreportcard.com/badge/github.com/moncho/dry)](https://goreportcard.com/report/github.com/moncho/dry)
[![codecov](https://codecov.io/gh/moncho/dry/branch/master/graph/badge.svg)](https://codecov.io/gh/moncho/dry)


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

### **dry** keybinds

#### Global
```
[F1]        sort list
[F5]        refresh list
[F9]        show last 10 docker events
[F10]       show docker info
[1]         show container list
[2]         show image list
[3]         show network list
[ArrowUp]   move the cursor one line up
[ArrowDown] move the cursor one line down
[q]         quit dry
```

#### Container commands
```
[F2]        toggle on/off showing stopped containers
[Enter]     inspect
[Ctrl]+[k]    kill
[l]         logs
[e]         remove
[Ctrl]+[e]    remove all stopped containers
[Ctrl]+[r]    start/restart
[s]         stats
[Ctrl]+[t]    stop
```

#### Image commands
```
[i]         history
[Ctrl]+[e]    remove image
[Ctrl]+[f]    remove image (force)
[Enter]     inspect
```
#### Network commands
```
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

```
$ curl -sSf https://moncho.github.io/dry/dryup.sh | sh
```

Most likely you will have to sudo it:

```
$ curl -sSf https://moncho.github.io/dry/dryup.sh | sudo sh
```

#### Binaries

If you dont like to **curl | sh**, binaries are provided.

- **darwin** [386](https://github.com/moncho/dry/releases/download/v0.5-alpha.4/dry-darwin-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.5-alpha.4/dry-darwin-amd64)
- **freebsd** [386](https://github.com/moncho/dry/releases/download/v0.5-alpha.4/dry-freebsd-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.5-alpha.4/dry-freebsd-amd64)
- **linux** [386](https://github.com/moncho/dry/releases/download/v0.5-alpha.4/dry-linux-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.5-alpha.4/dry-linux-amd64)
- **windows** [386](https://github.com/moncho/dry/releases/download/v0.5-alpha.4/dry-windows-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.5-alpha.4/dry-windows-amd64)

#### Go

And if you just run what you compile, use the source.

Make sure that **$GOPATH** exists. Go get this project.
```
$ go get github.com/moncho/dry
$ cd $GOPATH/src/github.com/moncho/dry
```
This project uses [godep](https://github.com/tools/godep) to handle its dependencies.
```
$ go get github.com/tools/godep
$ godep restore
```
Build **dry**.
```
$ make install
```

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

Also reused some code and ideas from the [Docker project](https://github.com/docker/docker).


[![Bitdeli Badge](https://d2weczhvl823v0.cloudfront.net/moncho/dry/trend.png)](https://bitdeli.com/free "Bitdeli Badge")
