UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Race detector works on macOS (any arch) and Linux x86_64
# Disabled on Linux ARM64 due to ThreadSanitizer VMA limitation
ifeq ($(UNAME_S)-$(filter aarch64 arm%,$(UNAME_M)),Linux-$(UNAME_M))
  RACE :=
else
  RACE := -race
endif

# Pinned to the same version the CI lint job uses (.github/workflows/ci.yml)
GOLANGCI_LINT_VERSION := v2.12.2

COVDIR := $(CURDIR)/.coverage

default: build coverage

build:
	go build ./...
	go vet ./...

# Unit tests only: fast, no Docker.
test:
	go test $(RACE) -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -func=coverage.out

# Integration tests only: a separate module (tests/), needs Docker. testcontainers
# builds the dev Postgres image and runs the migrate-grant-query cycle end to end.
integration:
	cd tests && go vet ./... && go test $(RACE) -timeout 10m ./...

# ALL tests (unit + integration) reported as ONE merged coverage profile of the
# library. Each run writes binary coverage data into its own -test.gocoverdir;
# go tool covdata merges them properly (unlike concatenating text profiles, which
# double-counts blocks covered by both suites). -coverpkg points the integration
# run back at the library so its coverage lands in the same package set.
coverage:
	rm -rf $(COVDIR) && mkdir -p $(COVDIR)/unit $(COVDIR)/integration
	go test $(RACE) -coverpkg=./... ./... -args -test.gocoverdir=$(COVDIR)/unit
	cd tests && go vet ./... && go test $(RACE) -timeout 10m \
		-coverpkg=github.com/llingr/llingr-pgx/... ./... \
		-args -test.gocoverdir=$(COVDIR)/integration
	go tool covdata textfmt -i=$(COVDIR)/unit,$(COVDIR)/integration -o coverage.out
	go tool cover -func=coverage.out

# Mirrors the CI lint job (same image, same version); nothing installed on the host.
lint:
	docker run --rm -v "$(PWD)":/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run ./...

# Fails if either module's go.mod/go.sum is untidy (CI does not check this).
tidy:
	go mod tidy -diff
	cd tests && go mod tidy -diff

clean:
	rm -rf $(COVDIR) coverage.out

all: build coverage lint tidy

.PHONY: default build test integration coverage lint tidy clean all
