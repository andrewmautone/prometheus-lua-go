# prometheus-lua-go

[![Go Reference](https://pkg.go.dev/badge/github.com/andrewmautone/prometheus-lua-go.svg)](https://pkg.go.dev/github.com/andrewmautone/prometheus-lua-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/andrewmautone/prometheus-lua-go)](https://goreportcard.com/report/github.com/andrewmautone/prometheus-lua-go)
[![License](https://img.shields.io/badge/license-Prometheus-blue.svg)](LICENSE)
[![Tests](https://github.com/andrewmautone/prometheus-lua-go/actions/workflows/test.yml/badge.svg)](https://github.com/andrewmautone/prometheus-lua-go/actions/workflows/test.yml)

> **Pure-Go bindings for the [Prometheus](https://github.com/prometheus-lua/Prometheus) Lua obfuscator.**
> No CGo. No external Lua runtime. Single static binary on Linux and Windows.

> Based on Prometheus by Elias Oelschner, https://github.com/prometheus-lua/Prometheus

---

## What it does

`prometheus-lua-go` lets you obfuscate Lua source code directly from Go.

The full Prometheus Lua codebase is embedded into your binary via `//go:embed`
and executed inside an in-process [gopher-lua](https://github.com/yuin/gopher-lua)
VM. There are no shell-outs, no temp files, no native dependencies — `go build`
and ship.

Use it to:

- Bundle a Lua obfuscator inside a Go CLI or web service
- Obfuscate game scripts, plugin scripts, or distributed Lua snippets as part
  of a build pipeline
- Apply Prometheus transformations (control-flow flattening, constant array,
  string encryption, VM-ification, anti-tamper, …) from any Go program

---

## Install

```bash
go get github.com/andrewmautone/prometheus-lua-go
```

Requires Go 1.22+. That's it — no Lua install, no compiler toolchain.

---

## Quick start

```go
package main

import (
    "fmt"
    "log"

    prometheus "github.com/andrewmautone/prometheus-lua-go"
)

func main() {
    o := prometheus.New(prometheus.Config{
        Preset: prometheus.PresetMedium,
        Seed:   42,
    })

    source := `print("Hello, World!")`

    out, err := o.Obfuscate(source)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(out)
}
```

Run it:

```bash
go run ./examples/basic
```

---

## Presets

| Constant            | Lua name    | Description                                                                |
| ------------------- | ----------- | -------------------------------------------------------------------------- |
| `PresetMinify`      | `Minify`    | Minify only. No obfuscation. No runtime cost.                              |
| `PresetWeak`        | `Weak`      | Light obfuscation. Highly readable, very low runtime cost.                 |
| `PresetMedium`      | `Medium`    | Balanced — constant arrays, string splitting, proxy locals. (Default.)     |
| `PresetStrong`      | `Strong`    | Aggressive — adds VM-ification, encryption, anti-tamper. Heavier runtime.  |
| `PresetOTClient`    | `OTClient`  | Light, safe for LuaJIT-based clients (OTClient, LÖVE, Tibia clients). Omits AntiTamper / Vmify / EncryptStrings, which depend on runtime invariants that diverge between LuaJIT and stock Lua. |

You can also pass any preset key supported by upstream Prometheus as a plain
string (for example `prometheus.Preset("Vmify")` for the isolated Vmify step).

---

## Configuration

```go
type Config struct {
    Preset        Preset     // Default: PresetMedium
    LuaVersion    LuaVersion // "Lua51" (default) or "LuaU"
    Seed          int64      // 0 = time-based (non-deterministic)
    VarNamePrefix string     // e.g. "_obf_"
    PrettyPrint   bool       // Format the output
    Verbose       bool       // Print Prometheus progress to stdout
}
```

### Deterministic output

Pass a non-zero `Seed` to get bit-identical output across runs:

```go
o := prometheus.New(prometheus.Config{
    Preset: prometheus.PresetMedium,
    Seed:   12345,
})
```

### LuaU (Roblox-flavored Lua)

```go
o := prometheus.New(prometheus.Config{
    Preset:     prometheus.PresetWeak,
    LuaVersion: prometheus.LuaU,
})
```

LuaU support in Prometheus is still experimental upstream — see the
[Prometheus docs](https://levno-710.gitbook.io/prometheus/) for caveats.

---

## API

```go
// Build an Obfuscator.
func New(cfg Config) *Obfuscator

// Obfuscate source code. filename is used in diagnostics only.
func (o *Obfuscator) Obfuscate(source string) (string, error)
func (o *Obfuscator) ObfuscateNamed(source, filename string) (string, error)
```

`Obfuscator` is **safe for concurrent use** — every call spins up its own
isolated Lua VM, so you can fan out across goroutines without locks.

---

## How it works

```
┌────────────────────────────────────────────────────────────────┐
│  Your Go program                                               │
│                                                                │
│      Obfuscator.Obfuscate(src)                                 │
│              │                                                 │
│              ▼                                                 │
│   ┌──────────────────────────────────────────────────────┐     │
│   │  gopher-lua VM (in-process, pure Go)                 │     │
│   │  ┌────────────────────────────────────────────────┐  │     │
│   │  │  Prometheus Lua codebase (//go:embed)          │  │     │
│   │  │  - prometheus.lua, pipeline, parser, steps, …  │  │     │
│   │  │  - all 80 .lua files preloaded via             │  │     │
│   │  │    package.preload                             │  │     │
│   │  └────────────────────────────────────────────────┘  │     │
│   │  bootstrap.lua → Pipeline:fromConfig:apply(src)      │     │
│   └──────────────────────────────────────────────────────┘     │
│              │                                                 │
│              ▼                                                 │
│      obfuscated string                                         │
└────────────────────────────────────────────────────────────────┘
```

The Lua source is shipped inside the Go binary; no filesystem access is
required at runtime. A fresh `*lua.LState` is created per call, so the VM
state is never shared between calls — that's what makes the library
trivially concurrent.

---

## Cross-platform

Tested on:

- Linux (amd64, arm64)
- Windows (amd64)

The library is pure Go with no CGo and no native dependencies, so any
target supported by both Go and gopher-lua works (macOS, FreeBSD, WASM with
caveats, etc.).

---

## Limitations

- **Lua 5.1 dialect.** Prometheus parses and emits Lua 5.1 (or LuaU). Your
  input must be valid Lua 5.1.
- **Performance.** gopher-lua is ~5-10× slower than reference Lua / LuaJIT.
  For a typical 1 KB script the obfuscation pass takes a few hundred
  milliseconds. For build-time use this is fine; for hot-path use, cache
  results.
- **First-call cost.** Each call loads ~80 Lua modules into a new VM
  (~50–200ms cold). If you obfuscate many small snippets back-to-back,
  consider batching them through a single longer Lua script via a future
  `BatchObfuscate` API (PRs welcome).

---

## Versioning

This wrapper tracks Prometheus upstream loosely. The embedded Lua source is
copied from a tagged upstream release; the version of the embedded source is
recorded in `NOTICE`.

This wrapper itself is versioned using semantic versioning starting at
`v0.1.0`.

---

## Contributing

Issues and PRs welcome — especially:

- Bumping the embedded Prometheus version
- Exposing the full `Steps` API (custom step lists, per-step settings)
- A `BatchObfuscate` that reuses one VM
- Additional examples (build pipelines, CI integration)

Run the test suite:

```bash
go test ./...
```

---

## Attribution

This project wraps and redistributes the Prometheus Lua obfuscator.

> **Based on Prometheus by Elias Oelschner, https://github.com/prometheus-lua/Prometheus**

The Prometheus License (see [`LICENSE`](LICENSE)) requires this attribution
to be reproduced in any product or service that integrates this library —
in the UI, About screen, `--version` output, or documentation. Please carry
the line above (or equivalent) when you ship.

---

## License

[Prometheus License](LICENSE) — a modified MIT-style license that permits
commercial use under the attribution conditions above.

Third-party components:

- [Prometheus](https://github.com/prometheus-lua/Prometheus) by Elias Oelschner — Prometheus License
- [gopher-lua](https://github.com/yuin/gopher-lua) by Yusuke Inuzuka — MIT License

See [`NOTICE`](NOTICE) for full third-party attribution.
