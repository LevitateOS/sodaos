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
	coder := in
	coder.Preparation.Role, coder.Bundle = RoleCoder, nil
	if err := coder.Validate(); err != nil {
		t.Fatalf("protected-source coder Retry refused: %v", err)
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

func TestFactoryCandidateBindsProtectedCoderExport(t *testing.T) {
	p := FactoryCandidate{Preparation: testPreparation(), SourcePreparation: "f111111111111111111111111", Deadline: time.Now().Add(time.Minute)}
	p.Preparation.Role = RoleCoder
	p.SourceRun = &FactoryExport{Project: p.Preparation.Project, ID: "22222222222222222222222222222222", Role: RoleCoder, Preparation: p.SourcePreparation, Candidate: p.Preparation.SourceCommit}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*FactoryCandidate){
		"foreign project":      func(v *FactoryCandidate) { v.SourceRun.Project = "p" + "333333333333333333333333" },
		"different candidate":  func(v *FactoryCandidate) { v.SourceRun.Candidate = "4444444444444444444444444444444444444444" },
		"reviewer export":      func(v *FactoryCandidate) { v.SourceRun.Role = RoleReviewer },
		"reviewer preparation": func(v *FactoryCandidate) { v.Preparation.Role = RoleReviewer; v.Bundle = []byte("bundle") },
		"caller bundle":        func(v *FactoryCandidate) { v.Bundle = []byte("bundle") },
		"invalid run":          func(v *FactoryCandidate) { v.SourceRun.ID = "invalid" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := p
			export := *p.SourceRun
			bad.SourceRun = &export
			mutate(&bad)
			if bad.Validate() == nil {
				t.Fatal("unbound source export accepted")
			}
		})
	}
}
