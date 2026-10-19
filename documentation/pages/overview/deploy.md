# Deployment

SQL Exporter can be deployed in a variety of ways, depending on your environment and preferences. Below are some common
deployment methods:

- Download a prebuilt binary for your operating system and architecture. You can find the latest releases on the
  [GitHub releases page](https://github.com/burningalchemist/sql_exporter/releases/latest);

- Download a prebuilt package for your operating system. SQL Exporter is available as a package for Debian and
  Fedora-based distributions (for `linux/amd64`). You can find the latest releases on the [GitHub releases
  page](https://github.com/burningalchemist/sql_exporter/releases/latest);

- Pull a container image. SQL Exporter is available as a Docker image on [Docker
  Hub](https://hub.docker.com/r/burningalchemist/sql_exporter);

- Use a Helm chart. More information about the Helm chart can be found in the chart
[README](https://github.com/burningalchemist/sql_exporter/blob/master/helm/README.md), and also at
[ArtifactHub](https://artifacthub.io/packages/helm/sql-exporter/sql-exporter). The Helm chart also comes with
provenance files to verify the authenticity of the chart and its contents.

## Build from source

Prerequisites:

- Go Compiler
- GNU Make

By default we produce a binary with all the supported drivers with the following command:

```sh
make build
```
