# dry
[![License MIT](https://img.shields.io/badge/license-MIT-lightgrey.svg?style=flat)](https://github.com/moncho/dry#license-mit)
[![wercker status](https://app.wercker.com/status/66c3ab71a46c0c8841f34a526fc23189/s/master "wercker status")](https://app.wercker.com/project/bykey/66c3ab71a46c0c8841f34a526fc23189)
[![Go Report Card](http://goreportcard.com/badge/moncho/dry)](http://goreportcard.com/report/moncho/dry)


**dry** is terminal application to manage **Docker** containers and images. It aims to be used as an alternative to the official **Docker CLI** when a lot of interactions with containers and images need to be done and to continuously monitor **Docker** containers from a terminal.

Upon connecting to a **Docker** host (local or remote), the main screen will show the list of containers and version information as reported by the **Docker Engine**. At all times, information about containers and images shown by **dry** is up-to-date, even if commands are executed from outside the tool.

From the main screen, commands to start, stop, remove and several others can be executed, all of them behaving like the equivalent command from the official **Docker CLI**. A detailed list of available commands can be found in this README, command information can also be read while running **dry** by showing help [H].

Switching to the image screen can be done by clicking [1]. Once there, **Docker** images are shown and again a few commands are available. Clicking [1] will show again the main screen (the container list).

Take a look at the demo below too see what can be done with **dry**.

[![asciicast](https://asciinema.org/a/35825.png)](https://asciinema.org/a/35825?autoplay=1)

Available commands (and their keybind in **dry**) from the official [**Docker** cli](https://docs.docker.com/engine/reference/commandline/cli/):

|Command | Key|
|---|---|
|***info***     | **[F10]**|
|***inspect***  | **[Enter]**|
|***kill***     | **[k]**|
|***logs***     | **[l]**|
|***ps***       ||
|***rm***       | **[e]**|
|***start***    | **[r]**|
|***stats***    | **[s]**|
|***stop***     | **[t]**|

Besides this, it:

* Shows real-time information about containers.
* Allows to sort the container list. **[F1]**
* Can search and navigate the output of ***info***, ***inspect***, ***logs*** commands.  
* Allows to easily remove all stopped containers. **[ctrl + E]**

### Installation

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

- **darwin** [386](https://github.com/moncho/dry/releases/download/v0.4-beta.1/dry-darwin-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.4-beta.1/dry-darwin-amd64)
- **freebsd** [386](https://github.com/moncho/dry/releases/download/v0.4-beta.1/dry-freebsd-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.4-beta.1/dry-freebsd-amd64)
- **linux** [386](https://github.com/moncho/dry/releases/download/v0.4-beta.1/dry-linux-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.4-beta.1/dry-linux-amd64)
- **windows** [386](https://github.com/moncho/dry/releases/download/v0.4-beta.1/dry-windows-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.4-beta.1/dry-windows-amd64)

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
* [go-dockerclient](https://github.com/fsouza/go-dockerclient)

Also reused some code and ideas from the [Docker project](https://github.com/docker/docker) and [mop](https://github.com/michaeldv/mop).


[![Bitdeli Badge](https://d2weczhvl823v0.cloudfront.net/moncho/dry/trend.png)](https://bitdeli.com/free "Bitdeli Badge")
