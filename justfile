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

# Release version lives in VERSION; `just bump` delegates to the version recipe.
bump version:
    @just version bump "{{version}}"

# Language-agnostic version contract for Delnegend/actions/resolve-release.
version action ver="":
    #!/usr/bin/env bash
    set -euo pipefail
    case "{{action}}" in
      get)
        cat VERSION
        ;;
      bump)
        [ -n "{{ver}}" ] || { echo "usage: just version bump X.Y.Z" >&2; exit 1; }
        NEW="{{ver}}"
        OLD="$(cat VERSION 2>/dev/null || true)"
        if [ "$NEW" = "$OLD" ]; then
          echo "error: bump $NEW is a no-op (VERSION unchanged)" >&2
          exit 1
        fi
        printf '%s\n' "$NEW" > VERSION
        git diff -- VERSION
        ;;
      *) echo "usage: just version get|bump X.Y.Z" >&2; exit 1 ;;
    esac
