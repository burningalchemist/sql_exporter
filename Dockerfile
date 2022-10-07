ARG  ARCH="amd64"
ARG  OS="linux"
FROM quay.io/prometheus/golang-builder AS builder

# Get sql_exporter
ADD .   /go/src/github.com/burningalchemist/sql_exporter
WORKDIR /go/src/github.com/burningalchemist/sql_exporter

# Do makefile
RUN make

# Make image and copy build sql_exporter
FROM        quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL       maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"
COPY        --from=builder /go/src/github.com/burningalchemist/sql_exporter/sql_exporter  /bin/sql_exporter

# Create user and group
RUN addgroup sql_exporter && \
    adduser -G sql_exporter -D sql_exporter && \
    mkdir -p /var/lib/sql_exporter && \
    chown -R sql_exporter:sql_exporter /var/lib/sql_exporter

EXPOSE      9399
USER        sql_exporter
WORKDIR     /var/lib/sql_exporter
ENTRYPOINT  [ "/bin/sql_exporter" ]
