FROM quay.io/prometheus/golang-builder AS builder

# Get sql_exporter
ADD .   /go/src/github.com/burningalchemist/sql_exporter
WORKDIR /go/src/github.com/burningalchemist/sql_exporter

# Do makefile
RUN make

# Make image and copy build sql_exporter
FROM        gcr.io/distroless/static:nonroot
MAINTAINER  The Prometheus Authors <prometheus-developers@googlegroups.com>
COPY        --from=builder /go/src/github.com/burningalchemist/sql_exporter/sql_exporter  /bin/sql_exporter

USER        65534
EXPOSE      9399
ENTRYPOINT  [ "/bin/sql_exporter" ]
