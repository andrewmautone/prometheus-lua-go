// Package prometheus provides Go bindings for the Prometheus Lua obfuscator.
//
// Based on Prometheus by Elias Oelschner,
// https://github.com/prometheus-lua/Prometheus
//
// The original Lua source is embedded and executed inside a gopher-lua VM,
// so no external Lua runtime is required. The library is pure Go and
// builds natively on Linux and Windows.
package prometheus

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

//go:embed all:luasrc
var luaFS embed.FS

//go:embed bootstrap.lua
var bootstrapLua string

// Preset selects one of the built-in Prometheus obfuscation presets.
type Preset string

const (
	PresetMinify Preset = "Minify"
	PresetWeak   Preset = "Weak"
	PresetMedium Preset = "Medium"
	PresetStrong Preset = "Strong"
	// PresetOTClient is a wrapper-only preset tuned for LuaJIT-based
	// clients (OTClient and forks). It runs ConstantArray +
	// NumbersToExpressions + WrapInFunction, omitting AntiTamper, Vmify,
	// and EncryptStrings — those three rely on runtime invariants that
	// diverge between LuaJIT and stock Lua and silently break the
	// obfuscated wrapper at load time.
	PresetOTClient Preset = "OTClient"
)

// LuaVersion selects the source dialect Prometheus parses and emits.
type LuaVersion string

const (
	Lua51 LuaVersion = "Lua51"
	LuaU  LuaVersion = "LuaU"
)

// Config controls obfuscation behavior. Zero-valued fields receive defaults.
type Config struct {
	// Preset selects the built-in obfuscation profile. Default: PresetMedium.
	Preset Preset
	// LuaVersion selects the input/output Lua dialect. Default: Lua51.
	LuaVersion LuaVersion
	// Seed for deterministic output. 0 (default) uses the current time.
	Seed int64
	// VarNamePrefix is prepended to every renamed local. Default: "".
	VarNamePrefix string
	// PrettyPrint enables formatted output. Default: false.
	PrettyPrint bool
	// Verbose enables Prometheus's progress logging to stdout. Default: false (silent).
	Verbose bool
}

// Obfuscator obfuscates Lua source using a fixed configuration. A new
// gopher-lua VM is spawned for each Obfuscate call, so Obfuscator instances
// are safe for concurrent use across goroutines.
type Obfuscator struct {
	cfg Config
}

// New returns an Obfuscator initialized with cfg. Unset fields fall back to
// PresetMedium and Lua51.
func New(cfg Config) *Obfuscator {
	if cfg.Preset == "" {
		cfg.Preset = PresetMedium
	}
	if cfg.LuaVersion == "" {
		cfg.LuaVersion = Lua51
	}
	return &Obfuscator{cfg: cfg}
}

// Obfuscate transforms source and returns the obfuscated Lua code.
func (o *Obfuscator) Obfuscate(source string) (string, error) {
	return o.ObfuscateNamed(source, "input.lua")
}

// ObfuscateNamed is Obfuscate with an explicit logical filename used in
// diagnostics emitted by Prometheus.
func (o *Obfuscator) ObfuscateNamed(source, filename string) (string, error) {
	// gopher-lua defaults (RegistrySize=8192, RegistryMaxSize=0) overflow when
	// Prometheus's unparser walks a large TableExpression — e.g. an i18n locale
	// file with thousands of fields. RegistryMaxSize > 0 enables auto-grow up
	// to the cap; CallStackSize headroom covers deeply nested ASTs.
	L := lua.NewState(lua.Options{
		CallStackSize:    1024,
		RegistrySize:     1024 * 128,
		RegistryMaxSize:  1024 * 1024,
		RegistryGrowStep: 1024,
	})
	defer L.Close()

	if err := registerEmbeddedModules(L); err != nil {
		return "", fmt.Errorf("prometheus: register modules: %w", err)
	}

	L.SetGlobal("__prom_source__", lua.LString(source))
	L.SetGlobal("__prom_filename__", lua.LString(filename))
	L.SetGlobal("__prom_preset__", lua.LString(string(o.cfg.Preset)))
	L.SetGlobal("__prom_seed__", lua.LNumber(o.cfg.Seed))
	L.SetGlobal("__prom_prefix__", lua.LString(o.cfg.VarNamePrefix))
	L.SetGlobal("__prom_lua_version__", lua.LString(string(o.cfg.LuaVersion)))
	L.SetGlobal("__prom_pretty__", lua.LBool(o.cfg.PrettyPrint))
	L.SetGlobal("__prom_verbose__", lua.LBool(o.cfg.Verbose))

	if err := L.DoString(bootstrapLua); err != nil {
		return "", fmt.Errorf("prometheus: %w", err)
	}

	result := L.GetGlobal("__prom_result__")
	if result.Type() != lua.LTString {
		return "", fmt.Errorf("prometheus: pipeline did not return a string")
	}
	return result.String(), nil
}

func registerEmbeddedModules(L *lua.LState) error {
	return fs.WalkDir(luaFS, "luasrc", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".lua") {
			return nil
		}

		rel := strings.TrimPrefix(path, "luasrc/")
		rel = strings.TrimSuffix(rel, ".lua")
		modName := strings.ReplaceAll(rel, "/", ".")

		data, err := fs.ReadFile(luaFS, path)
		if err != nil {
			return err
		}
		src := string(data)
		chunkName := "@" + path

		L.PreloadModule(modName, func(L *lua.LState) int {
			fn, loadErr := L.Load(strings.NewReader(src), chunkName)
			if loadErr != nil {
				L.RaiseError("prometheus: load %s: %v", modName, loadErr)
				return 0
			}
			L.Push(fn)
			L.Call(0, 1)
			return 1
		})
		return nil
	})
}
