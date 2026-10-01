package factory

import "testing"

func TestRunViewValidation(t *testing.T) {
	valid := RunView{RunID: NewID(), Repository: 7, Issue: 42, Attempt: "attempt-1"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid view: %v", err)
	}
	unbound := RunView{RunID: valid.RunID, Repository: 7}
	if err := unbound.Validate(); err != nil {
		t.Fatalf("unbound view: %v", err)
	}
	for name, mutate := range map[string]func(*RunView){
		"run":        func(v *RunView) { v.RunID = "short" },
		"repository": func(v *RunView) { v.Repository = 0 },
		"issue":      func(v *RunView) { v.Issue = -1 },
		"attempt":    func(v *RunView) { v.Attempt = "../escape" },
	} {
		broken := valid
		mutate(&broken)
		if err := broken.Validate(); err == nil {
			t.Fatalf("accepted invalid %s", name)
		}
	}
}
