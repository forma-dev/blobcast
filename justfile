#!/usr/bin/env -S just --justfile
# ^ A shebang isn't required, but allows a justfile to be executed
#   like a script, with `./justfile test`, for example.

VERSION := env_var_or_default("VERSION", `git describe --tags --always --dirty=-dev`)
GIT_COMMIT := env_var_or_default("GIT_COMMIT", `git rev-parse HEAD`)
BUILD_TIME := env_var_or_default("BUILD_TIME", `date -u +%Y-%m-%dT%H:%M:%SZ`)

build:
    go build \
      -ldflags="-w -s \
                -X github.com/forma-dev/blobcast/pkg/version.Version={{VERSION}} \
                -X github.com/forma-dev/blobcast/pkg/version.GitCommit={{GIT_COMMIT}} \
                -X github.com/forma-dev/blobcast/pkg/version.BuildTime={{BUILD_TIME}}" \
      -o build/blobcast .

clean:
    rm -rf build/

install:
    @if [ ! -f build/blobcast ]; then \
        echo "Binary not found, building..."; \
        just build; \
    else \
        echo "Binary already exists, skipping build"; \
    fi
    @sudo cp build/blobcast /usr/local/bin/
    @echo "Binary installed to /usr/local/bin/blobcast"

info:
    @echo "Version: {{VERSION}}"
    @echo "Git Commit: {{GIT_COMMIT}}"
    @echo "Build Time: {{BUILD_TIME}}"
