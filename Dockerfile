FROM golang:1.13.5-stretch AS builder
ENV CGO_ENABLED=0 GOOS=linux GO111MODULE=on
RUN go get sigs.k8s.io/kustomize/v3/cmd/kustomize@v3.2.0
ARG KUBECTL_VERSION=v1.15.0
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl && chmod +x ./kubectl
COPY . $GOPATH/src/github.com/snarlysodboxer/hambone
RUN cd $GOPATH/src/github.com/snarlysodboxer/hambone && go build

FROM alpine:3.11
MAINTAINER david amick <docker@davidamick.com>
ENV PATH="/usr/local/bin:${PATH}"
WORKDIR /hambone
COPY --from=builder /go/kubectl /usr/local/bin/kubectl
COPY --from=builder /go/bin/kustomize /usr/local/bin/kustomize
COPY --from=builder /go/src/github.com/snarlysodboxer/hambone/hambone /usr/local/bin/hambone
ENTRYPOINT ["hambone"]
CMD []

