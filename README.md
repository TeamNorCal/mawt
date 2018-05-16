# mawt - A ray of light

<repo-version>0.1.0-feature-01-techthulu-events-1f3aTX</repo-version>

This software implements a server that parses input from a Niantic Tecthulhu and uses it to build OPC frames that are then sent to another TCP/IP server.

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
sudo apt-get install golang-go
sudo apt-get install -y portaudio19-dev libasound2-dev libvorbis-dev alsa-utils alsa-tools alsa-oss alsaplayer mpg321 alsaplayer-alsa alsa-base
go run build.go
```

## Running mawt

```shell
LOGXI=*=DBG /home/pi/mawt/bin/mawt
```

## Running the simulator using scenario files

```shell
LOGXI=*=DBG go run cmd/simulator/*.go -path assets/simulator/portal_builds
```


## fcserver configuration

fcserver should be run using the following config.json file

```shell
{
    "listen": ["0.0.0.0", 7890],
    "relay":  [null, 7891],
    "verbose": true,

    "color": {
        "gamma": 2.5,
        "whitepoint": [1.0, 1.0, 1.0]
    },

    "devices": [
        {
            "type": "fadecandy",
            "serial": "AMWPGCSIYCRCKYHL",
            "map": [
                [ 1, 0, 0, 64 ],
                [ 2, 0, 64, 64 ],
                [ 3, 0, 128, 64 ],
                [ 4, 0, 192, 64 ],
                [ 6, 0, 256, 64 ],
                [ 7, 0, 320, 64 ],
                [ 8, 0, 384, 64 ]
                [ 9, 0, 448, 64 ]
            ]
        }
    ]
}
```
