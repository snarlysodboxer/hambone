#!/bin/bash

if [ -z "$GOPATH" ]; then
    echo "GOPATH env var must be set"
    exit 1
fi

set -e

case "$1" in
local)
    CGO_ENABLED=1 GOOS=darwin go build -buildmode=plugin -o plugins/engine/k8s-api/k8s-api.so plugins/engine/k8s-api/k8s-api.go
    echo "Built engine plugin"
    CGO_ENABLED=1 GOOS=darwin go build -buildmode=plugin -o plugins/state/memory/memory.so plugins/state/memory/memory.go
    echo "Built state plugin"
    CGO_ENABLED=1 GOOS=darwin go build -buildmode=plugin -o plugins/render/default/default.so plugins/render/default/default.go
    echo "Built render plugin"
    ;;
engine)
    docker run -it --rm -v $GOPATH:/go --workdir /go/src/github.com/snarlysodboxer/hambone -e CGO_ENABLED=1 -e GOOS=linux --entrypoint go golang:1.10.1 build -buildmode=plugin -o plugins/engine/k8s-api/k8s-api.so plugins/engine/k8s-api/k8s-api.go
    echo "Built engine plugin"
    ;;
state)
    docker run -it --rm -v $GOPATH:/go --workdir /go/src/github.com/snarlysodboxer/hambone -e CGO_ENABLED=1 -e GOOS=linux --entrypoint go golang:1.10.1 build -buildmode=plugin -o plugins/state/memory/memory.so plugins/state/memory/memory.go
    echo "Built state plugin"
    ;;
render)
    docker run -it --rm -v $GOPATH:/go --workdir /go/src/github.com/snarlysodboxer/hambone -e CGO_ENABLED=1 -e GOOS=linux --entrypoint go golang:1.10.1 build -buildmode=plugin -o plugins/render/default/default.so plugins/render/default/default.go
    echo "Built render plugin"
    ;;
*)
    docker run -it --rm -v $GOPATH:/go --workdir /go/src/github.com/snarlysodboxer/hambone -e CGO_ENABLED=1 -e GOOS=linux --entrypoint go golang:1.10.1 build -buildmode=plugin -o plugins/engine/k8s-api/k8s-api.so plugins/engine/k8s-api/k8s-api.go
    echo "Built engine plugin"
    docker run -it --rm -v $GOPATH:/go --workdir /go/src/github.com/snarlysodboxer/hambone -e CGO_ENABLED=1 -e GOOS=linux --entrypoint go golang:1.10.1 build -buildmode=plugin -o plugins/state/memory/memory.so plugins/state/memory/memory.go
    echo "Built state plugin"
    docker run -it --rm -v $GOPATH:/go --workdir /go/src/github.com/snarlysodboxer/hambone -e CGO_ENABLED=1 -e GOOS=linux --entrypoint go golang:1.10.1 build -buildmode=plugin -o plugins/render/default/default.so plugins/render/default/default.go
    echo "Built render plugin"
    ;;
esac


