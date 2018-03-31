# mawt - A ray of light

This software implements a server that parses input from a Niantic Techthulu and uses it to build OPC frames that are then sent to another TCP/IP server.

## Build instructions



https://github.com/direnv/direnv

```
go get -u github.com/golang/dep/cmd/dep
go get github.com/karlmutch/duat
go install github.com/karlmutch/duat/semver
go install github.com/karlmutch/duat/github-release
go install github.com/karlmutch/duat/image-release
go install github.com/karlmutch/duat/stencil
```

## Running mawt


