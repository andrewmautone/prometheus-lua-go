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

-- gopher-lua bug: tonumber rejects decimal literals in scientific notation
-- without an explicit decimal point. Standard Lua accepts "1e10", but
-- gopher-lua returns nil for any "NNNeKK" form unless there is a "." between
-- the mantissa and the "e". Source code being obfuscated commonly contains
-- literals like 1e6, 5e3, etc.; without this shim the Prometheus tokenizer
-- stores nil as the number value and every downstream step crashes.
-- Wrap tonumber globally and inject ".0" before "e" when the raw call fails.
do
    local rawTonumber = tonumber
    local floor = math.floor

    -- Manual base conversion for digits up to a given base. Avoids the
    -- int64 overflow that gopher-lua's strconv-backed tonumber hits on
    -- big hex / binary literals (e.g. 0xFFFFFFFFFFFFFFFF). Uses
    -- multiplication on floats, which gracefully overflows to math.huge
    -- rather than returning nil.
    local function manualBaseConvert(s, base, charPattern)
        local body = s:match("^%s*(" .. charPattern .. "+)%s*$")
        if not body then
            return nil
        end
        local v = 0
        for i = 1, #body do
            local c = body:byte(i)
            local d
            if c >= 48 and c <= 57 then
                d = c - 48
            elseif c >= 65 and c <= 70 then
                d = c - 55
            elseif c >= 97 and c <= 102 then
                d = c - 87
            else
                return nil
            end
            if d >= base then
                return nil
            end
            v = v * base + d
        end
        return v
    end

    _G.tonumber = function(s, base)
        local n = rawTonumber(s, base)
        if n ~= nil or type(s) ~= "string" then
            return n
        end
        if base == 16 then
            return manualBaseConvert(s, 16, "%x")
        end
        if base == 2 then
            return manualBaseConvert(s, 2, "[01]")
        end
        if base ~= nil then
            return nil
        end
        -- base == nil: decimal / scientific path.
        local fixed, replaced = s:gsub("^(%s*[%+%-]?%d+)([eE])", "%1.0%2")
        if replaced > 0 then
            n = rawTonumber(fixed)
            if n ~= nil then
                return n
            end
        end
        -- Out-of-range scientific literal: gopher-lua's strconv.ParseFloat
        -- rejects values that overflow a float64. Standard Lua returns inf.
        local sign = s:match("^%s*([%+%-]?)%d[%d%.]*[eE][%+%-]?%d+%s*$")
        if sign ~= nil then
            if sign == "-" then
                return -math.huge
            end
            return math.huge
        end
        return nil
    end
end

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

-- Strip Vmify when the source uses `goto` / `::label::`.
-- The parser/unparser now understand goto and labels (see PATCHES.md),
-- but the Vmify step compiles the entire AST into a bespoke VM whose
-- compiler does not model arbitrary goto. Rather than crashing, we
-- detect goto/label in the source and remove Vmify from the active
-- Steps list. The file still gets every other obfuscation pass
-- (EncryptStrings, ConstantArray, ProxifyLocals, AntiTamper, etc.).
local function hasGotoOrLabel(src)
    if src:match("[^%w_]goto%s+[%a_]") or src:match("^goto%s+[%a_]") then
        return true
    end
    if src:match("::%s*[%a_][%w_]*%s*::") then
        return true
    end
    return false
end

if hasGotoOrLabel(__prom_source__) and cfg.Steps then
    local filtered = {}
    for _, step in ipairs(cfg.Steps) do
        if step.Name ~= "Vmify" then
            filtered[#filtered + 1] = step
        end
    end
    cfg.Steps = filtered
end

local pipeline = Prometheus.Pipeline:fromConfig(cfg)
__prom_result__ = pipeline:apply(__prom_source__, __prom_filename__)
