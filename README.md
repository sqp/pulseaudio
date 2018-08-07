# pulseaudio: native pulseaudio client for go through dbus.

[![License](https://img.shields.io/:license-ISC-brightgreen.svg)](https://raw.githubusercontent.com/sqp/pulseaudio/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/sqp/pulseaudio)](https://goreportcard.com/report/sqp/pulseaudio) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/sqp/pulseaudio)


pulseaudio is a simple library that controls a pulseaudio server through its Dbus interface.

### Features

* Control your audio cards and streams in pure go
* Complete native implementation of the pulseaudio D-Bus protocol
* Splitted interface to allow clients to implement only what they need

### Installation

This packages requires Go 1.1. If you installed it and set up your GOPATH, just run:

```
go get github.com/sqp/pulseaudio
```

### Usage

The complete package documentation is available at [godoc.org](http://godoc.org/github.com/sqp/pulseaudio).
See also:
* [the client example](https://github.com/sqp/pulseaudio/blob/master/example/client.go) is a short overview of the basic usage. 
* [a real use](https://github.com/sqp/godock/blob/master/services/Audio/audio.go) in a cairo-dock applet. 

### Note

You will have to enable the dbus module of your pulseaudio server.
This can now be done with ```pulseaudio.LoadModule()``` function.

or by adding this line in ```/etc/pulse/default.pa```
```
    load-module module-dbus-protocol
```
if system-wide daemon mode is used, edit the file ```/etc/pulse/system.pa```



### Evolutions

* The base API has been stable for years and there's no plan to improve it for now.
* A higher level API could be designed to cover simple frequent needs.
Open a issue to discuss it if you want.

### License

pulseaudio is available under the ISC License; see LICENSE for the full text.
