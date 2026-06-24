# Kazi Ancestry — Makefile

APP_NAME = Kazi Ancestry
BIN := kazi-ancestry

# Where to push the docker image.
REGISTRY ?= masudjuly02

# Git metadata for versioning.
git_branch       := $(shell git rev-parse --abbrev-ref HEAD)
git_tag          := $(shell git describe --exact-match --abbrev=0 2>/dev/null || echo "")
commit_hash      := $(shell git rev-parse --verify HEAD)
commit_timestamp := $(shell git show -s --format=%cd --date=format:'%Y-%m-%dT%H:%M:%S' HEAD)

VERSION          := $(shell git describe --tags --always --dirty)
version_strategy := commit_hash
ifdef git_tag
	VERSION := $(git_tag)
	version_strategy := tag
endif

DOCKER_IMAGE := $(REGISTRY)/$(BIN)
GO_VERSION   ?= 1.26

# ── Build ─────────────────────────────────────────────────────────────────────

all: # @HELP builds the binary
all: build

.PHONY: build
build: # @HELP compiles the binary into bin/
	go build -o bin/$(BIN) .

.PHONY: run
run: # @HELP builds and runs the server locally (native Go)
	go run . serve

.PHONY: seed
seed: # @HELP seeds the database (use ARGS=--reseed to regenerate ids)
	go run . seed $(ARGS)

# ── Format ────────────────────────────────────────────────────────────────────

.PHONY: fmt
fmt: # @HELP formats and reorders Go imports
	@which goimports-reviser >/dev/null 2>&1 || go install github.com/incu6us/goimports-reviser/v3@latest
	goimports-reviser -recursive -company-prefixes=github.com/masudur-rahman -imports-order=std,project,company,general,blanked -format -excludes vendor ./...

# ── Test & Quality ────────────────────────────────────────────────────────────

.PHONY: test
test: # @HELP runs tests with the race detector and coverage
	go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep -E "^total:" | awk '{print "Coverage: " $$3}'

.PHONY: test-short
test-short: # @HELP runs tests without long-running cases
	go test -short ./...

.PHONY: coverage-html
coverage-html: test # @HELP opens the coverage report in a browser
	go tool cover -html=coverage.out

.PHONY: vet
vet: # @HELP runs go vet
	go vet ./...

.PHONY: lint
lint: # @HELP runs golangci-lint
	@which golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./... --timeout=5m

.PHONY: vulncheck
vulncheck: # @HELP runs govulncheck for known vulnerabilities
	@which govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

.PHONY: check
check: # @HELP runs vet + lint + tests
check: vet lint test

.PHONY: tidy
tidy: # @HELP tidies and verifies Go modules
	go mod tidy
	go mod verify

.PHONY: verify
verify: # @HELP fails if go.mod/go.sum are not tidy
verify: tidy
	@if ! git diff --exit-code go.mod go.sum; then \
		echo "go.mod/go.sum are out of date; run 'go mod tidy'"; exit 1; \
	fi

# ── Docker ────────────────────────────────────────────────────────────────────

.PHONY: docker-build
docker-build: # @HELP builds a single-arch Docker image
	DOCKER_BUILDKIT=1 docker build \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg BUILD_DATE=$(commit_timestamp) \
	  --build-arg GIT_COMMIT=$(commit_hash) \
	  $(DOCKER_CACHE_ARGS) \
	  -t $(DOCKER_IMAGE):$(VERSION) \
	  -f Dockerfile .

.PHONY: docker-run
docker-run: # @HELP runs the image, mounting local config + .env (needs a reachable Postgres)
	@if [ -z "$$(docker images -q $(DOCKER_IMAGE):$(VERSION))" ]; then $(MAKE) docker-build; fi
	docker run --rm -p 5294:5294 \
	  --env-file .env \
	  --volume $(CURDIR)/.configs/.kazi-ancestry.yaml:/app/.configs/.kazi-ancestry.yaml:ro \
	  $(DOCKER_IMAGE):$(VERSION)

.PHONY: docker-compose-up
docker-compose-up: # @HELP runs Postgres + app via Docker Compose
	docker compose up --build

.PHONY: docker-build-push
docker-build-push: # @HELP builds and pushes a multi-arch image to the registry
	docker buildx build --platform linux/amd64,linux/arm64 \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg BUILD_DATE=$(commit_timestamp) \
	  --build-arg GIT_COMMIT=$(commit_hash) \
	  --output "type=image,push=true" \
	  --tag $(DOCKER_IMAGE):$(VERSION) \
	  --tag $(DOCKER_IMAGE):latest .

.PHONY: release
release: # @HELP builds and pushes the release image (multi-arch)
release: docker-build-push

# ── Misc ──────────────────────────────────────────────────────────────────────

.PHONY: version
version: # @HELP prints version + build information
	@echo "Application Name:    $(APP_NAME)"
	@echo "Version:             $(VERSION)  ($(version_strategy))"
	@echo "Git Tag:             $(git_tag)"
	@echo "Git Branch:          $(git_branch)"
	@echo "Commit Hash:         $(commit_hash)"
	@echo "Commit Timestamp:    $(commit_timestamp)"
	@echo "Go Version:          $(shell go version | cut -d ' ' -f 3)"
	@echo "Platform:            $(shell go env GOOS)/$(shell go env GOARCH)"

.PHONY: clean
clean: # @HELP removes built binaries and temporary files
	rm -rf bin coverage.out

help: # @HELP prints this message
help:
	@echo "VARIABLES:"
	@echo "  BIN = $(BIN)"
	@echo "  REGISTRY = $(REGISTRY)"
	@echo "  VERSION = $(VERSION)"
	@echo
	@echo "TARGETS:"
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST)    \
	    | awk '                                   \
	        BEGIN {FS = ": *# *@HELP"};           \
	        { printf "  %-22s %s\n", $$1, $$2 };  \
	    '
