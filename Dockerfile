FROM golang:alpine as builder
RUN apk update && apk upgrade && \
    apk add --no-cache bash git
ADD . /github.com/densify-dev/Container-Optimization-Data-Forwarder
WORKDIR /github.com/densify-dev/Container-Optimization-Data-Forwarder/cmd/dataCollection
RUN go build -o dataCollection .
FROM alpine
CMD ["./Forwarder", "-c", "-n", "k8s_transfer_v3", "-l", "k8s_transfer_v3", "-o", "upload", "-r", "-C", "config"]
RUN mkdir data data/node data/container data/hpa data/cluster
RUN chmod 777 -R data
COPY ./config config
COPY ./tools .
COPY --from=builder /github.com/densify-dev/Container-Optimization-Data-Forwarder/cmd/dataCollection .