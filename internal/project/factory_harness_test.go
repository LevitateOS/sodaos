package project

import (
	"strings"
	"testing"
)

func TestFactoryHarnessPinValidate(t *testing.T) {
	good := FactoryHarnessPin{
		Harness: FactoryHarnessCodex, Version: "1.2.3",
		SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64),
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("pin refused: %v", err)
	}
	for name, mutate := range map[string]func(*FactoryHarnessPin){
		"harness": func(p *FactoryHarnessPin) { p.Harness = "muse" },
		"version": func(p *FactoryHarnessPin) { p.Version = "" },
		"sha":     func(p *FactoryHarnessPin) { p.SHA256 = "short" },
		"image":   func(p *FactoryHarnessPin) { p.Image = "localhost/soda:latest" },
	} {
		next := good
		mutate(&next)
		if err := next.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}
