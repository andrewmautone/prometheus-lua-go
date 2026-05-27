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

---

If you spot additional gopher-lua compatibility issues, please open an issue
on this repository (do **not** report them upstream — they only manifest with
gopher-lua's table semantics).
