# Deployment & Operations

## Install

Download the latest prebuilt release archive for Linux x86_64:

```bash
URL=$(curl -fsSL https://api.github.com/repos/Delnegend/differ/releases/latest | grep -o 'https://[^"]*linux-amd64\.tar\.xz')
curl -fsSL -o differ.tar.xz "$URL" && tar -xJf differ.tar.xz --strip-components=1
```

For other architectures or platforms, build from source using Go 1.24+:

```bash
go build -o differ .
```

## Binary Verification

Verify the installed binary executes correctly:

```bash
./differ -h
```
