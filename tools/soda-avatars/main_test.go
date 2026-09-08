package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogUsesAllPartsAndProductionSamples(t *testing.T) {
	parent := t.TempDir()
	first, err := generate(parent)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generate(parent)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("preview overwrote prior output")
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(first), "svg", "*.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 44+32+100 {
		t.Fatalf("got %d images, want all 44 parts, 32 palettes, 100 robots", len(files))
	}
	for _, file := range files {
		a, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(filepath.Dir(second), "svg", filepath.Base(file)))
		if err != nil || string(a) != string(b) {
			t.Fatalf("catalog changed across runs: %s: %v", file, err)
		}
	}
	html, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`id="grid-24"`, `id="grid-32"`, `id="grid-64"`, `id="grid-128"`, `id="dark"`, `id="circle"`} {
		if !strings.Contains(string(html), want) {
			t.Errorf("missing %s", want)
		}
	}
}
