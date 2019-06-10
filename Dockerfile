FROM golang:alpine as builder
RUN apk update && apk upgrade && \
    apk add --no-cache bash git
RUN go get github.com/prometheus/common/model github.com/prometheus/client_golang/api github.com/spf13/viper github.com/json-iterator/go
RUN mkdir /go/src/build
ADD . /go/src/github.com/densify-dev/Container-Optimization-Data-Forwarder
WORKDIR /go/src/github.com/densify-dev/Container-Optimization-Data-Forwarder/cmd/dataCollection
RUN go build -o dataCollection .
FROM alpine
CMD ["./Forwarder", "-c", "-n", "k8s_transfer_v2", "-l", "k8s_transfer_v2", "-o", "upload", "-r", "-C", "config"]
RUN mkdir data data/node data/container
COPY ./config config
COPY ./tools .
COPY --from=builder /go/src/github.com/densify-dev/Container-Optimization-Data-Forwarder/cmd/dataCollection .