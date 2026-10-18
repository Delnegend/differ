<div align="center">

# Differ

**Shrink a run of near-identical images: store the first frame whole and every following frame as its pixel diff, then rebuild the originals exactly.**

[![CI](https://img.shields.io/github/actions/workflow/status/Delnegend/differ/ci.yml?branch=main&style=flat-square)](https://github.com/Delnegend/differ/actions)
[![Release](https://img.shields.io/github/v/release/Delnegend/differ?style=flat-square)](https://github.com/Delnegend/differ/releases)
[![License](https://img.shields.io/badge/license-MIT%20%7C%20Apache--2.0-blue?style=flat-square)](LICENSE-MIT)

</div>

---

## Quick Start

```bash
# 1. Build
go build -o differ .

# 2. Compress a sequence (first image kept whole, the rest become diffs)
./differ -diff frame01.png frame02.png frame03.png

# 3. Rebuild the exact originals from base + diffs
./differ -join frame01.BASE.png frame02.DIFF.png frame03.DIFF.png
```

- **Lossless where it matters** — `A: 255` changed pixels overwrite the base while untouched pixels pass through; fully transparent source pixels are the documented edge, everything else round-trips.

- **Frame-delta compression, no codec** — only changed pixels are stored per frame, like H.264 inter-frames without the container.
- **No output directory to manage** — `.BASE` and `.DIFF` files land next to their sources with self-describing names.
- **One static binary** — Go, no runtime, no config.

## Layout

| File | Purpose |
|---|---|
| `main.go` | The whole tool: `-diff` and `-join` modes, PNG encode/decode |
| `go.mod` | Module `differ`, Go 1.24 |

## Releases

- Weekly runs cut a semver tag and publish six archives (`linux`/`darwin` amd64+arm64, `windows` amd64+arm64); versions live in git tags only, there is no manifest to bump.
- A run dispatched with a commit SHA test-builds it and uploads the same six archives without publishing anything. Dispatch modes are shared across projects ([Delnegend/actions](https://github.com/Delnegend/actions/blob/main/README.md)).

## License

Dual-licensed under [MIT](LICENSE-MIT) or [Apache-2.0](LICENSE-APACHE), at your option.
