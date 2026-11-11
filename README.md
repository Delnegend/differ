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
# 1. Install (Linux x64; all six platform builds are on the releases page)
URL=$(curl -fsSL https://api.github.com/repos/Delnegend/differ/releases/latest | grep -o 'https://[^"]*linux-amd64\.tar\.xz')
curl -fsSL -o differ.tar.xz "$URL" && tar -xJf differ.tar.xz --strip-components=1

# 2. Compress a sequence (first image kept whole, the rest become diffs)
./differ -diff frame01.png frame02.png frame03.png

# 3. Rebuild the exact originals from base + diffs
./differ -join frame01.BASE.png frame02.DIFF.png frame03.DIFF.png
```

Prefer building from source? `go build -o differ .` needs Go 1.24+ and nothing else.

## Highlights

- **Lossless where it matters**: `A: 255` changed pixels overwrite the base while untouched pixels pass through; fully transparent source pixels are the documented edge, everything else round-trips.
- **Frame-delta compression, no codec**: only changed pixels are stored per frame, like H.264 inter-frames without the container.
- **No output directory to manage**: `.BASE` and `.DIFF` files land next to their sources with self-describing names.
- **One static binary**: Go, no runtime, no config.

## Documentation

- **[Architecture](docs/architecture.md)**: Frame-delta compression pipeline, file layout, and lossless reconstruction invariants.
- **[Development](docs/development.md)**: Building from source, running tests, and semver tag release workflow.
- **[Deployment](docs/deployment.md)**: Installation via release archives, binary verification, and environment setup.

## License

Dual-licensed under [MIT](LICENSE-MIT) or [Apache-2.0](LICENSE-APACHE), at your option.
