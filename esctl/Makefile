BIN_DIR := bin
CMDS := $(wildcard cmd/*)
BINARIES := $(patsubst cmd/%,${BIN_DIR}/%,${CMDS})
VERSION := $(shell git describe --tags 2>/dev/null || echo "v0.0.0")
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X github.com/matthew-hollick/elasticsearch-cli/internal/version.Version=${VERSION} \
                     -X github.com/matthew-hollick/elasticsearch-cli/internal/version.Commit=${COMMIT} \
                     -X 'github.com/matthew-hollick/elasticsearch-cli/internal/version.BuildTime=${BUILD_TIME}'"

.PHONY: all build clean test lint fmt

all: build

build: ${BINARIES}

${BIN_DIR}/%: cmd/%/main.go
	@mkdir -p ${BIN_DIR}
	mise exec -- go build ${LDFLAGS} -o $@ $<

clean:
	rm -rf ${BIN_DIR}

test:
	mise exec -- go test -v -coverprofile=coverage.out ./...

lint:
	mise exec -- golangci-lint run

fmt:
	mise exec -- gofmt -s -w .
	mise exec -- goimports -w .

install-tools:
	mise exec -- go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	mise exec -- go install golang.org/x/tools/cmd/goimports@latest
