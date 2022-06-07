ARG BASE_IMAGE=alpine

FROM golang:bullseye as builder
ADD . /github.com/densify-dev/Container-Optimization-Data-Forwarder
WORKDIR /github.com/densify-dev/Container-Optimization-Data-Forwarder/cmd/dataCollection
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -gcflags=-trimpath="${GOPATH}" -asmflags=-trimpath="${GOPATH}" -ldflags="-w -s" -o ./dataCollection .

FROM ${BASE_IMAGE}:latest
ARG BASE_IMAGE
ENV BASE_IMG=${BASE_IMAGE}
ARG VERSION
ARG RELEASE

### Required OpenShift Labels
LABEL name="Container-Optimization-Data-Forwarder" \
      vendor="Densify" \
      maintainer="support@densify.com" \
      version="${VERSION}" \
      release="${RELEASE}" \
      summary="Densify container data collection" \
      description="Collects data from Prometheus and sends to Densify server for analysis"

# BASE_IMAGE specifics - add non-root user and remove the ability to install packages
RUN case ${BASE_IMG} in \
    alpine* ) \
        addgroup -g 3000 densify && \
        adduser -h /home/densify -s /bin/sh -u 3000 -G densify -g "" -D densify && \
        rm -f /sbin/apk ;; \
    *ubi* ) \
        microdnf install -y shadow-utils && \
        groupadd -g 3000 densify && \
        adduser -d /home/densify -m -s /bin/bash -u 3000 -g densify densify && \
        microdnf remove -y shadow-utils && \
        rm -f /bin/microdnf ;; \
    debian* ) \
        mkdir -p /home/densify && \
        addgroup --gid 3000 densify && \
        useradd --home /home/densify --shell /bin/bash --uid 3000 --gid 3000 --password "" densify && \
        chown densify:densify /home/densify && \
        rm -f /usr/bin/apt /usr/bin/apt-get ;; \
    * ) \
        echo "unsupported base image ${BASE_IMG}" && \
        exit 1 ;; \
    esac

### add licenses to this directory
COPY --chown=densify:densify --chmod=644 ./LICENSE /licenses/LICENSE
### keep /config as this is how it is mounted in versions < 3.0
COPY --chown=densify:densify --chmod=644 ./config /config

WORKDIR /home/densify
RUN mkdir -p data/node data/container data/hpa data/cluster data/node_group data/crq data/rq && chown -R densify:densify /home/densify/data && chmod -R 777 /home/densify/data && ln -s /config config
COPY --chown=densify:densify --chmod=755 ./tools bin
COPY --chown=densify:densify --chmod=755 --from=builder /github.com/densify-dev/Container-Optimization-Data-Forwarder/cmd/dataCollection/dataCollection bin
USER densify
CMD ["/home/densify/bin/entry.sh"]
