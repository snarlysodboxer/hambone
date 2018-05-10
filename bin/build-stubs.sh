#!/bin/bash

set -e

mkdir -p generated

docker run -it --rm -v $(pwd):/code -w /code --entrypoint /usr/bin/protoc snarlysodboxer/protoc-grpc-gateway:0.0.1 -I/usr/include -I./protos -I./protos/vendor --go_out=,plugins=grpc:./generated protos/hambone.proto
echo "Generated Golang gRPC stub"

