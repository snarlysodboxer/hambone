#!/bin/bash

if [ -z "$GOPATH" ]; then
    echo "GOPATH env var must be set"
    exit 1
fi

set -e

docker run -it --rm -v $GOPATH:/go --workdir /go/src/github.com/snarlysodboxer/hambone --network host -e CGO_ENABLED=1 -e GOOS=linux --entrypoint go golang:1.10.1 "$@"

