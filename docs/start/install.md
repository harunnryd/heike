---
title: Install
sidebar_position: 3
---

## Script Install

```sh
curl -fsSL https://raw.githubusercontent.com/harunnryd/heike/main/install.sh | sh
```

Alternative URL:

```sh
curl -fsSL https://heike.tech/install | sh
```

## Build From Source

```sh
git clone https://github.com/harunnryd/heike.git
cd heike
go build -o heike ./cmd/heike
./heike version
```

## Verify

```sh
./heike version
```

If the binary is installed in `PATH`, you can also run `heike version`.
