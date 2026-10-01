package project

import "testing"

func TestFactoryOutputValidation(t *testing.T) {
	valid := FactoryOutput{Project: "p123456789012345678901234", ID: "0123456789abcdef0123456789abcdef", Offset: 0, Limit: 1024}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid output: %v", err)
	}
	for name, mutate := range map[string]func(*FactoryOutput){
		"project": func(p *FactoryOutput) { p.Project = "nope" },
		"run":     func(p *FactoryOutput) { p.ID = "short" },
		"cursor":  func(p *FactoryOutput) { p.Offset = -1 },
		"huge":    func(p *FactoryOutput) { p.Offset = MaxFactoryOutputOffset + 1 },
		"empty":   func(p *FactoryOutput) { p.Limit = 0 },
		"wide":    func(p *FactoryOutput) { p.Limit = MaxFactoryOutputRead + 1 },
	} {
		broken := valid
		mutate(&broken)
		if err := broken.Validate(); err == nil {
			t.Fatalf("accepted invalid %s", name)
		}
	}
}
