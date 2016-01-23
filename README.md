# dry
[![License MIT](https://img.shields.io/badge/license-MIT-lightgrey.svg?style=flat)](https://github.com/moncho/dry#license-mit)
[![wercker status](https://app.wercker.com/status/66c3ab71a46c0c8841f34a526fc23189/s/master "wercker status")](https://app.wercker.com/project/bykey/66c3ab71a46c0c8841f34a526fc23189)
[![Go Report Card](http://goreportcard.com/badge/moncho/dry)](http://goreportcard.com/report/moncho/dry)


**dry** is terminal application that connects to a **Docker** host, shows the list of containers and allows to interact with them.

[![asciicast](https://asciinema.org/a/32898.png)](https://asciinema.org/a/32898?autoplay=1)

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

- **darwin** [386](https://github.com/moncho/dry/releases/download/v0.3-beta.7/dry-darwin-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.3-beta.7/dry-darwin-amd64)
- **freebsd** [386](https://github.com/moncho/dry/releases/download/v0.3-beta.7/dry-freebsd-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.3-beta.7/dry-freebsd-amd64)
- **linux** [386](https://github.com/moncho/dry/releases/download/v0.3-beta.7/dry-linux-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.3-beta.7/dry-linux-amd64)
- **windows** [386](https://github.com/moncho/dry/releases/download/v0.3-beta.7/dry-windows-386) / [amd64](https://github.com/moncho/dry/releases/download/v0.3-beta.7/dry-windows-amd64)

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

### Debug

Debugging can be done using [godebug](https://github.com/mailgun/godebug).

Install it, then insert a breakpoint anywhere you want:
```
_ = "breakpoint"
```
And run the debugger:
```
godebug run -instrument github.com/moncho/dry,github.com/moncho/dry/app,github.com/moncho/dry/docker,github.com/moncho/dry/ui,github.com/moncho/dry/appui main.go
```
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
