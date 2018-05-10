FROM golang:1.10 AS builder
ENV CGO_ENABLED=0 GOOS=linux
RUN go get k8s.io/kubectl/cmd/kustomize
ARG KUBECTL_VERSION=v1.7.3
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl && chmod +x ./kubectl
COPY . $GOPATH/src/github.com/snarlysodboxer/hambone
RUN cd $GOPATH/src/github.com/snarlysodboxer/hambone && go build

FROM alpine:3.7
MAINTAINER david amick <docker@davidamick.com>
ENV PATH="/usr/local/bin:${PATH}"
WORKDIR /hambone
COPY --from=builder /go/kubectl /usr/local/bin/kubectl
COPY --from=builder /go/bin/kustomize /usr/local/bin/kustomize
COPY --from=builder /go/src/github.com/snarlysodboxer/hambone/hambone /usr/local/bin/hambone
ENTRYPOINT ["hambone"]
CMD []

