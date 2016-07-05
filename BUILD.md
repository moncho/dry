### Build instructions

If you just run what you compile, use the source.

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
