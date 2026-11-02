#
# SPDX-License-Identifier: BSD-3-Clause
#

PKGS := $(shell go list ./... | grep -v example | grep -v tools)
ROOT_DIR := $(shell git rev-parse --show-toplevel)
GOLANGCI_VERSION := "v2.6.2"

.PHONY: all test test-integration etag-sim build lint fmt generate check-generate clean

all: lint build test

test:
	go test -v $(PKGS) -cover -race

# Opt-in ETag / If-Match test (//go:build integration). Starts a local
# etag-aware fake unless GOFISH_TEST_REDFISH_URL points at a live service.
test-integration:
	go test -tags=integration -count=1 -v ./test/integration

# Long-running Redfish fake that returns ETag and honors If-Match.
#   make etag-sim
#   GOFISH_TEST_REDFISH_URL=http://127.0.0.1:8000 make test-integration
etag-sim:
	go run ./tools/etag-sim

build:
	go build

lint:
	docker run --rm \
                -v "$(ROOT_DIR)":/src \
                -w /src \
                "golangci/golangci-lint:$(GOLANGCI_VERSION)" \
                golangci-lint run -v

fmt:
	docker run --rm \
                -v "$(ROOT_DIR)":/src \
                -w /src \
                "golangci/golangci-lint:$(GOLANGCI_VERSION)" \
                golangci-lint fmt .

OUTPUT_DIR ?= ../..
# Schema bundles the committed code is generated from. Bump these to update.
REDFISH_TAG ?= 2026.2
SWORDFISH_TAG ?= v1.2.9_Release

generate:
	cd tools/generator && \
    go run ./cmd/generate-schemas --output-dir $(OUTPUT_DIR) \
		--redfish-tag $(REDFISH_TAG) --swordfish-tag $(SWORDFISH_TAG)

# Fails if regenerating changes or adds any file under schemas/ or serviceroot.go.
check-generate: generate
	git diff --exit-code -- schemas serviceroot.go
	test -z "$$(git status --porcelain -- schemas serviceroot.go)"

clean:
	go clean
