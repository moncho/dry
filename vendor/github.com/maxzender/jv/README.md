[![Go Report Card](https://goreportcard.com/badge/maxzender/jv)](https://goreportcard.com/report/maxzender/jv)
[![Build Status](https://travis-ci.org/maxzender/jv.svg?branch=master)](https://travis-ci.org/maxzender/jv)

# jv
jv (for jsonviewer) helps you view your JSON.

[![asciicast](https://asciinema.org/a/123606.png)](https://asciinema.org/a/123606)

## Installation
```
go get -u github.com/maxzender/jv
```

## Example usage
```
jv file.json
```
Or by reading from `stdin`:
```
jv < file.json
echo '{"foo": "bar"}' | jv
```
