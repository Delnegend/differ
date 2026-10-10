# Differ — dev workflows

# Verify formatting, vet, and tests (CI gate: `Check (just check)`)
check:
    #!/usr/bin/env bash
    set -euo pipefail
    [ -z "$(gofmt -l .)" ]
    go vet ./...
    if [ -n "$(find . -name '*_test.go' -print -quit)" ]; then
        go test ./...
    else
        echo "no tests yet"
    fi

# Cross-compile all six release targets (mirrors release.yml)
build:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p dist/linux-amd64 && GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/differ .;
    mkdir -p dist/linux-arm64 && GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/differ .;
    mkdir -p dist/darwin-amd64 && GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/differ .;
    mkdir -p dist/darwin-arm64 && GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/differ .;
    mkdir -p dist/windows-amd64 && GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/differ.exe .;
    mkdir -p dist/windows-arm64 && GOOS=windows GOARCH=arm64 go build -o dist/windows-arm64/differ.exe .

# Go has no manifest: versions live in git tags, and the release workflow tags
# HEAD directly. Intentional no-op — the step exists so `new-tag` keeps the
# same shape as every other repo.
bump version:
    @echo "Go versions are tracked via git tags; nothing to bump"
