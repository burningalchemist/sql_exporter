# Copyright 2015 The Prometheus Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# To distinguish between native Windows and Windows Subsystem for Linux (WSL),
# we have to check how PATH is separated. For WSL and Unix-based systems it's
# a colon; for native Windows it's a semicolon.
ifeq '$(findstring ;,$(PATH))' ';'
	GOPATH = $(firstword $(subst ;, ,$(shell $(GO) env GOPATH)))
	PREFIX = $(shell cd)
endif

GO     := go
GOPATH ?= $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
PROMU  := $(GOPATH)/bin/promu
PROMU_VERSION := v0.17.0
pkgs    = $(shell $(GO) list ./... | grep -v /vendor/)

PREFIX              ?= $(shell pwd)
BIN_DIR             ?= $(shell pwd)
DOCKER_IMAGE_NAME   ?= sql-exporter
DOCKER_IMAGE_TAG    ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))


all: format build test

style:
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

test:
	@echo ">> running tests"
	@$(GO) test -short $(pkgs)

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build: promu
	@echo ">> building binaries"
	@$(PROMU) build --prefix $(PREFIX)

drivers-%:
	@echo ">> generating drivers.go with selected drivers"
	@$(GO) get github.com/dave/jennifer/jen
	@$(GO) run drivers_gen.go -- $*
	@$(GO) get ./...
	@$(GO) mod tidy

tarball: promu
	@echo ">> building release tarball"
	@$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)

crossbuild: promu
	@echo ">> building crossbuild release"
	@$(PROMU) crossbuild

crossbuild-tarballs: promu
	@echo ">> building crossbuild release tarballs"
	@$(PROMU) crossbuild tarballs

crossbuild-checksum: promu
	@echo ">> calculating checksums for released packages"
	@$(PROMU) checksum .tarballs

crossbuild-release: promu crossbuild crossbuild-tarballs crossbuild-checksum

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

# Override for native Windows, where the path separator is a semicolon.
ifeq '$(findstring ;,$(PATH))' ';'
promu:
	@set GOOS=windows
	@set GOARCH=$(subst AMD64,amd64,$(patsubst i%86,386,$(shell echo %PROCESSOR_ARCHITECTURE%)))
	@$(GO) install github.com/prometheus/promu@$(PROMU_VERSION)
else
promu:
	@GOOS=$(shell uname -s | tr A-Z a-z) \
		GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m))) \
		$(GO) install github.com/prometheus/promu@$(PROMU_VERSION)
endif

.PHONY: all style format build test vet tarball docker promu
