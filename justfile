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

# Cross-compile the Linux release target (mirrors release.yml)
build:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p dist/linux-amd64 && GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/differ .;

# Go has no manifest: versions live in git tags, and the release workflow tags
# HEAD directly. Intentional no-op — the step exists so `new-tag` keeps the
# same shape as every other repo.
bump version:
    @echo "Go versions are tracked via git tags; nothing to bump"
