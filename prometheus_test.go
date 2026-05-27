package prometheus

import (
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
