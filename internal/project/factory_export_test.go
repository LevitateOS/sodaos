package project

import (
	"strings"
	"testing"
)

func TestFactoryExportValidates(t *testing.T) {
	valid := FactoryExport{
		Project: "p123456789012345678901234", ID: "0123456789abcdef0123456789abcdef",
		Role: RoleCoder, Preparation: "f123456789012345678901234",
		Candidate: strings.Repeat("a", 40),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid export refused: %v", err)
	}
	for name, mutate := range map[string]func(*FactoryExport){
		"project":     func(p *FactoryExport) { p.Project = "nope" },
		"run":         func(p *FactoryExport) { p.ID = "short" },
		"role":        func(p *FactoryExport) { p.Role = "coder" },
		"preparation": func(p *FactoryExport) { p.Preparation = "nope" },
		"candidate":   func(p *FactoryExport) { p.Candidate = "xyz" },
	} {
		mutated := valid
		mutate(&mutated)
		if err := mutated.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}
