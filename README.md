<p align="center">
  <h2 align="center">SQL Exporter for Prometheus</h2>
</p>

<p align="center" style="text-decoration: none; border: none;">
  <a
    href="https://hub.docker.com/r/burningalchemist/sql_exporter"
    style="text-decoration: none"
  >
    <img
      alt="Docker Pulls"
      src="https://img.shields.io/docker/pulls/burningalchemist/sql_exporter"
    ></a>
  <a
    href="https://github.com/burningalchemist/sql_exporter/releases/latest"
    style="text-decoration: none"
  >
    <img
      alt="Downloads"
      src="https://img.shields.io/github/downloads/burningalchemist/sql_exporter/total"
    ></a>
  <a
    href="https://artifacthub.io/packages/helm/sql-exporter/sql-exporter"
    style="text-decoration: none"
  >
    <img
      alt="Artifact HUB"
      src="https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/sql-exporter"
    /></a>
  <a
    href="https://github.com/burningalchemist/sql_exporter/actions/workflows/govulncheck-binary.yml"
    style="text-decoration: none"
  >
    <img
      alt="Audit Released Binary (linux/amd64)"
      src="https://github.com/burningalchemist/sql_exporter/actions/workflows/govulncheck-binary.yml/badge.svg"
    /></a>
</p>

<p align="center">
  <a href="https://burningalchemist.github.io/sql_exporter">Documentation</a>
  ·
  <a href="https://burningalchemist.github.io/sql_exporter/pages/config.html">Configuration Reference</a>
</p>

**SQL Exporter** is a flexible database-agnostic exporter that exposes metrics gathered from databases in Prometheus
format.

## Features

- Database-agnostic with broad driver support
- Scrape-synchronous collection with optional caching
- Staggered cache warmup
- Fully configuration-driven metrics
- Flexible metric mapping
- Multi-target (jobs) mode
- Selective job scraping
- Secret manager integration
- TLS/BasicAuth support
- Request rate limiting
- Live reload of configuration without restart
- Automatic metrics
- Builtin validation and config check

## Installation

Head over to the [documentation website](https://burningalchemist.github.io/sql_exporter/pages/config/general.html) for installation instructions and configuration options.

## Support

If you have an issue using sql_exporter, please check [Discussions](https://github.com/burningalchemist/sql_exporter/discussions) or
closed [Issues](https://github.com/burningalchemist/sql_exporter/issues?q=is%3Aissue+is%3Aclosed) first. Chances are
someone else has already encountered the same problem and there is a solution. If not, feel free to create a new
discussion.

## Special Thanks

- [@free](https://github.com/free) for creating the original implementation;
- [@allanger](https://github.com/allanger) for contributing the helm chart and the dev/release environment around it;
- [@itdesign-sponsoring](https://github.com/itdesign-sponsoring) for sponsoring the development of the project;
- all the contributors who have helped improve this project over the years.
