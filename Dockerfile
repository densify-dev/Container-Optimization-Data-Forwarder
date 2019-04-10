FROM golang:alpine as builder
RUN apk update && apk upgrade && \
    apk add --no-cache bash git
WORKDIR /go/src/build
RUN mkdir /go/src/build
ADD ./densify /go/src/build
RUN go get
RUN go build -o main .
FROM alpine
COPY ./densify .
COPY --from=builder /go/src/build/main .
WORKDIR .
CMD ["./Forwarder", "-c", "-n", "k8s_transfer_v2", "-l", "k8s_transfer_v2", "-o", "upload", "-r", "-C", "config"]