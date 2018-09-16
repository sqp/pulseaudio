# pulseaudio: native pulseaudio client for go through dbus.

[![Build Status](https://travis-ci.org/sqp/pulseaudio.svg?branch=master)](https://travis-ci.org/sqp/pulseaudio)
[![codecov](https://codecov.io/gh/sqp/pulseaudio/branch/master/graph/badge.svg)](https://codecov.io/gh/sqp/pulseaudio)
[![golangci](https://golangci.com/badges/github.com/sqp/pulseaudio.svg)](https://golangci.com/r/github.com/sqp/pulseaudio)
[![Go Report Card](https://goreportcard.com/badge/sqp/pulseaudio)](https://goreportcard.com/report/sqp/pulseaudio)

[![License](https://img.shields.io/:license-ISC-brightgreen.svg)](https://raw.githubusercontent.com/sqp/pulseaudio/master/LICENSE)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/sqp/pulseaudio)


pulseaudio is a simple library that controls a pulseaudio server through its D-Bus interface.

### Features

* Control your audio cards and streams in pure go.
* Native implementation of the pulseaudio D-Bus protocol.
* Small lib pretty close to the DBus API rather than trying to abstract everything.
* Splitted interface to allow clients to implement only what they need.
* Test coverage 86%, missing 9 lines in errors paths harder to test.
* Only one dependency, the dbus library: github.com/godbus/dbus
* [Permissive software licence](https://en.wikipedia.org/wiki/Permissive_software_licence): [ISC](https://raw.githubusercontent.com/sqp/pulseaudio/master/LICENSE)

### Installation

This packages requires Go 1.7 (for the dbus lib). If you installed it and set up your GOPATH, just run:

```
go get -u github.com/sqp/pulseaudio
```

### Usage

The complete package documentation is available at [godoc.org](http://godoc.org/github.com/sqp/pulseaudio).
See also:
* [The package example](https://godoc.org/github.com/sqp/pulseaudio/#example_) with a short overview of the basic usage. 
* A real use in [a cairo-dock applet](https://github.com/sqp/godock/blob/master/services/Audio/audio.go).

### Note

You will have to enable the dbus module of your pulseaudio server.
This can now be done with ```pulseaudio.LoadModule()``` function (requires the pacmd command, in package ```pulseaudio-utils``` on debian).

or as a permanent config by adding this line in ```/etc/pulse/default.pa```
```
    load-module module-dbus-protocol
```
If system-wide daemon mode is used, the file to edit is ```/etc/pulse/system.pa```

### Evolutions

* The base API has been stable for years and there's no plan to improve it for now.
* A higher level API could be designed to cover simple frequent needs.
Open an issue to discuss it if you want.
* The lib may at some point move to a community repo. This could be an
opportunity to change a little the API, so we'll need some feedback.

### Feedback

Please [open an issue](https://github.com/sqp/pulseaudio/issues) or submit a pull request if:
* You tried or use this library, let us know if you saw things to improve, especially in the doc if you're a native English speaker.
* You want your code to be listed as example.