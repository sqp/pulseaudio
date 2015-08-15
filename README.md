pulseaudio
----------


pulseaudio is a simple library that controls a pulseaudio server through its 
Dbus interface.

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

The complete package documentation and some simple examples are available at
[godoc.org](http://godoc.org/github.com/sqp/pulseaudio).
Also, the
[client example](https://github.com/sqp/pulseaudio/blob/master/example/client.go) file
gives a short overview over the basic usage. 

Please note that the API is considered unstable for now and may change without
further notice.

### License

pulseaudio is available under the ISC License; see LICENSE for the full text.
