package prometheus

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

const helloLua = `print("Hello, World!")`

func TestObfuscate_DefaultConfig(t *testing.T) {
	o := New(Config{})
	out, err := o.Obfuscate(helloLua)
	if err != nil {
		t.Fatalf("Obfuscate: %v", err)
	}
	if out == "" {
		t.Fatal("empty output")
	}
}

func TestObfuscate_TransformsCode(t *testing.T) {
	o := New(Config{Preset: PresetMedium, Seed: 12345})
	out, err := o.Obfuscate(helloLua)
	if err != nil {
		t.Fatalf("Obfuscate: %v", err)
	}
	if strings.Contains(out, `print("Hello, World!")`) {
		t.Errorf("output still contains original source:\n%s", out)
	}
}

func TestObfuscate_AllPresets(t *testing.T) {
	for _, p := range []Preset{PresetMinify, PresetWeak, PresetMedium, PresetStrong} {
		p := p
		t.Run(string(p), func(t *testing.T) {
			o := New(Config{Preset: p, Seed: 1})
			out, err := o.Obfuscate(helloLua)
			if err != nil {
				t.Fatalf("preset %s: %v", p, err)
			}
			if out == "" {
				t.Fatalf("preset %s: empty output", p)
			}
		})
	}
}

func TestObfuscate_DeterministicSeed(t *testing.T) {
	o := New(Config{Preset: PresetMedium, Seed: 4242})
	a, err := o.Obfuscate(helloLua)
	if err != nil {
		t.Fatal(err)
	}
	b, err := o.Obfuscate(helloLua)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("seed did not produce deterministic output\nA:\n%s\nB:\n%s", a, b)
	}
}

func TestObfuscate_UnknownPreset(t *testing.T) {
	o := New(Config{Preset: "Bogus"})
	if _, err := o.Obfuscate(helloLua); err == nil {
		t.Fatal("expected error for unknown preset")
	}
}

// TestObfuscate_LargeLocaleTable exercises the unparser's joinParts ->
// table.concat path on a TableExpression with thousands of fields (i18n
// locale style). With the default gopher-lua state (RegistryMaxSize=0)
// this path runs out of registry slots and crashes. The fix in
// ObfuscateNamed enables registry auto-grow.
//
// Skipped under -short because gopher-lua takes several seconds even for
// this reduced size. Increase the entry count to stress larger workloads.
func TestObfuscate_LargeLocaleTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-table test in short mode")
	}
	const entries = 1500
	var b strings.Builder
	b.Grow(entries * 96)
	b.WriteString("return {\n")
	for i := 0; i < entries; i++ {
		fmt.Fprintf(&b, "  [%q] = %q,\n",
			fmt.Sprintf("ui.label.section_%d.key_%d", i/100, i),
			fmt.Sprintf("Translation %d.", i),
		)
	}
	b.WriteString("}\n")

	o := New(Config{Preset: PresetMinify, Seed: 1})
	out, err := o.Obfuscate(b.String())
	if err != nil {
		t.Fatalf("Obfuscate large table (%d entries): %v", entries, err)
	}
	if out == "" {
		t.Fatal("empty output")
	}
}

// TestObfuscate_ScientificNumberLiteral covers two number-literal forms
// that exposed bugs in our wrapper:
//
//   - "5e3" (no decimal point) hits a gopher-lua tonumber quirk: it
//     returns nil for any "NNNeKK" form lacking a "." between mantissa and
//     exponent. bootstrap.lua now shims tonumber to retry with ".0e".
//   - "1e500" parses to +Inf and overflows arithmetic inside the
//     NumbersToExpressions generators; those generators now bail when the
//     round-trip via tostring/tonumber produces nil.
func TestObfuscate_ScientificNumberLiteral(t *testing.T) {
	cases := []struct {
		name, src string
	}{
		{"plain_scientific", `local KILLS = 5e3; local DAY = 86400; return KILLS * DAY`},
		{"extreme_literal", `local huge = 1e500; local norm = 4294967295; return huge + norm`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			o := New(Config{Preset: PresetMedium, Seed: 1})
			out, err := o.Obfuscate(tc.src)
			if err != nil {
				t.Fatalf("Obfuscate %s: %v", tc.name, err)
			}
			if out == "" {
				t.Fatal("empty output")
			}
		})
	}
}

func TestObfuscate_Concurrent(t *testing.T) {
	o := New(Config{Preset: PresetWeak, Seed: 7})
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := o.Obfuscate(helloLua)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent obfuscate: %v", err)
		}
	}
}
