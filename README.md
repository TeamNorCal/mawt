# mawt - A ray of light

<repo-version>0.1.0-feature-01-techthulu-events-1f3aTX</repo-version>

This software implements a server that parses input from a Niantic Techthulu and uses it to build OPC frames that are then sent to another TCP/IP server.

## Source code preparation

```shell
cd ~
mkdir -p mawt/src/github.com/TeamNorCal/
export GOPATH=~/mawt
export PATH=$GOPATH/bin:$PATH
cd mawt/src/github.com/TeamNorCal/
git clone https://github.com/TeamNorCal/mawt
cd mawt
```

```shell
go get -u github.com/golang/dep
go install github.com/golang/dep/cmd/dep
go get github.com/karlmutch/duat
go install github.com/karlmutch/duat/cmd/semver
go install github.com/karlmutch/duat/cmd/github-release
go install github.com/karlmutch/duat/cmd/stencil
```

## Build instructions

```shell
go run build.go
```

## Running mawt


