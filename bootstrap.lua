-- bootstrap.lua
-- Invoked from Go after all embedded Prometheus modules are preloaded.
-- Globals are injected by the Go layer:
--   __prom_source__      string  - Lua source to obfuscate
--   __prom_filename__    string  - logical filename for diagnostics
--   __prom_preset__      string  - "Minify" | "Weak" | "Medium" | "Strong" | ...
--   __prom_seed__        number  - 0 selects time-based seed inside Prometheus
--   __prom_prefix__      string  - var name prefix; "" leaves preset default
--   __prom_lua_version__ string  - "Lua51" | "LuaU"; "" leaves preset default
--   __prom_pretty__      boolean - pretty print toggle
--
-- The obfuscated output is written back to __prom_result__.

-- gopher-lua does not populate the `arg` global the way the standalone Lua
-- interpreter does. Prometheus modules read `arg` at module load time
-- (config.lua scans it for CLI flags), so define an empty table to satisfy
-- the `for _, v in pairs(arg) do` loops before any require runs.
arg = arg or {}

-- Silence Prometheus's progress logger unless verbose mode is requested.
-- Prometheus writes status lines via print(); override it before loading
-- the modules so even module-load-time messages are suppressed.
if not __prom_verbose__ then
    _G.print = function() end
end

local Prometheus = require("prometheus")

if Prometheus.colors then
    Prometheus.colors.enabled = false
end

local preset = Prometheus.Presets[__prom_preset__]
if not preset then
    error("unknown preset: " .. tostring(__prom_preset__))
end

local cfg = {}
for k, v in pairs(preset) do
    cfg[k] = v
end

if __prom_seed__ and __prom_seed__ ~= 0 then
    cfg.Seed = __prom_seed__
end
if __prom_prefix__ and __prom_prefix__ ~= "" then
    cfg.VarNamePrefix = __prom_prefix__
end
if __prom_lua_version__ and __prom_lua_version__ ~= "" then
    cfg.LuaVersion = __prom_lua_version__
end
if __prom_pretty__ then
    cfg.PrettyPrint = true
end

local pipeline = Prometheus.Pipeline:fromConfig(cfg)
__prom_result__ = pipeline:apply(__prom_source__, __prom_filename__)
