# Embedded Prometheus patches

The Lua source under `./luasrc/` is a copy of upstream
[Prometheus](https://github.com/prometheus-lua/Prometheus) (v0.2.11.x). The
following minimal patches were applied to make the source run under
[gopher-lua](https://github.com/yuin/gopher-lua):

## `luasrc/prometheus/compiler/emit.lua`

Inside `Compiler:emitContainerFuncBody`, the original code does:

```lua
local block = { id = id, index = i, block = Ast.Block(blockstats, block.scope) }
table.insert(blocks, block)
blocks[id] = block   -- removed
```

The `blocks[id] = block` write is **never read** anywhere in the file. In the
reference Lua 5.1 / LuaJIT runtimes the dead write is harmless, but
gopher-lua's table implementation can promote large numeric keys into the
array portion of the table, leaving `nil` holes that `#blocks` and the
subsequent `table.sort(blocks, ...)` then trip over (`attempt to index a
non-table object(nil) with key 'id'`).

The line is commented out in our embedded copy. No other code is affected.

## `luasrc/prometheus/steps/NumbersToExpressions.lua`

The original generators do `tonumber(tostring(diff)) ± tonumber(tostring(val2))`
to verify a candidate decomposition. When `val + val2` (or `val - val2`)
overflows to inf/nan — possible with extreme numeric literals in the input —
`tonumber(tostring(inf))` returns `nil`, and the next arithmetic op then
errors with "cannot perform sub operation between nil and number".

Patched the three generators (Addition, Subtraction, Modulo) to detect the
nil round-trip and return `false`, which falls through to the next generator
(or to a plain NumberExpression).

## `bootstrap.lua` — `tonumber` shim

gopher-lua's `tonumber` rejects decimal scientific-notation literals that
lack an explicit decimal point: `tonumber("1e10")`, `tonumber("2e5")`, even
`tonumber("2e0")` all return `nil`. Standard Lua / LuaJIT accept these.

This breaks the Prometheus tokenizer, which converts every number literal
in the source via `tonumber`. Any `Ne...` literal in the input becomes a
`NumberExpression` with `value = nil`, and every subsequent step that does
arithmetic on `node.value` crashes (e.g. `NumbersToExpressions.lua:50:
cannot perform add operation between nil and number`).

We override `tonumber` in `bootstrap.lua` to retry the conversion with
`.0` injected before the `e` when the raw call returns nil.

The same shim also handles two related gopher-lua quirks:

- `tonumber(s, 16)` and `tonumber(s, 2)` return nil when the value
  overflows int64 (e.g. `0xFFFFFFFFFFFFFFFF`). We fall back to a manual
  base conversion using float multiplication, which overflows to
  `math.huge` rather than producing nil.
- Decimal scientific literals that overflow float64 (e.g. `1e500`) get
  mapped to signed `math.huge`, matching standard Lua / LuaJIT semantics
  instead of returning nil.

---

If you spot additional gopher-lua compatibility issues, please open an issue
on this repository (do **not** report them upstream — they only manifest with
gopher-lua's table semantics).
