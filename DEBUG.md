### How to debug

Debugging can be done using [godebug](https://github.com/mailgun/godebug).

Install it, then insert a breakpoint anywhere you want:
```
_ = "breakpoint"
```
And run the debugger:
```
godebug run -instrument github.com/moncho/dry,github.com/moncho/dry/app,github.com/moncho/dry/docker,github.com/moncho/dry/ui,github.com/moncho/dry/appui main.go
```
