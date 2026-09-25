# Decentralized.Host build, test and release gates.
GO        ?= go
BIN       := bin
PEBBLE    := github.com/letsencrypt/pebble/v2
PEBBLE_V  := v2.10.1

.PHONY: all build tools vet fmt fmt-check test race integration conformance chaos gates clean

all: build

build:
	$(GO) build -trimpath -o $(BIN)/ ./cmd/...

# ACME test server for the M4/Pebble integration tests (pinned version).
tools:
	GOBIN=$(CURDIR)/tools/bin $(GO) install $(PEBBLE)/cmd/pebble@$(PEBBLE_V) $(PEBBLE)/cmd/pebble-challtestsrv@$(PEBBLE_V)

vet:
	$(GO) vet ./...

fmt:
	gofmt -w cmd pkg tests conformance web

fmt-check:
	@out=$$(gofmt -l cmd pkg tests conformance web); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

test:
	$(GO) test ./pkg/... ./cmd/... ./conformance/...

race:
	$(GO) test -race ./pkg/...

integration:
	$(GO) test -count=1 -timeout 30m ./tests/integration/...

conformance: build
	$(BIN)/dh-conformance check
	$(BIN)/dh-conformance run -self
	$(BIN)/dh-conformance run -adapter "$(BIN)/dh-conformance adapter"
	$(BIN)/dh-conformance run -adapter "python3 conformance/python/adapter.py"

chaos: build
	$(BIN)/dh chaos run --scenario all --out ./chaos-reports

# Release gates: formatting, vet, build, unit tests with -race, conformance, integration.
gates: fmt-check vet build race conformance integration

clean:
	rm -rf $(BIN)
