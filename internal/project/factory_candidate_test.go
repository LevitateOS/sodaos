package project

import (
	"testing"
	"time"
)

func TestFactoryCandidateRequiresFreshBoundedPreparation(t *testing.T) {
	in := FactoryCandidate{Preparation: testPreparation(), SourcePreparation: "f111111111111111111111111", Bundle: []byte("candidate"), Deadline: time.Now().Add(time.Minute)}
	in.Preparation.Role = RoleReviewer
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*FactoryCandidate){
		"coder checkout":   func(v *FactoryCandidate) { v.Preparation.Role = RoleCoder },
		"same checkout":    func(v *FactoryCandidate) { v.SourcePreparation = v.Preparation.ID },
		"source path":      func(v *FactoryCandidate) { v.SourcePreparation = "../source" },
		"missing bundle":   func(v *FactoryCandidate) { v.Bundle = nil },
		"oversized bundle": func(v *FactoryCandidate) { v.Bundle = make([]byte, MaxSourceBundle+1) },
		"missing approval": func(v *FactoryCandidate) { v.Preparation.Approval = AdminApproval{} },
		"missing deadline": func(v *FactoryCandidate) { v.Deadline = time.Time{} },
	} {
		t.Run(name, func(t *testing.T) {
			bad := in
			mutate(&bad)
			if bad.Validate() == nil {
				t.Fatal("invalid candidate preparation accepted")
			}
		})
	}
}
