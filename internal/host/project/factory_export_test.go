package project

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

func TestFactoryExportGatesOnRecordedReceipt(t *testing.T) {
	f := openTestFactory(t, "")
	ctx := context.Background()
	run := factoryTestRun()
	req := domain.FactoryExport{
		Project: run.Project, ID: run.ID, Role: run.Role,
		Preparation: run.Preparation, Candidate: strings.Repeat("f", 40),
	}
	if _, err := f.Export(ctx, domain.FactoryExport{Project: "bad", ID: run.ID, Role: run.Role, Preparation: run.Preparation, Candidate: req.Candidate}); err == nil {
		t.Fatal("invalid export address admitted")
	}
	if _, err := f.Export(ctx, req); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("unknown run: %v", err)
	}
	if err := f.storeReceipt(factoryReceipt{Run: run, Phase: domain.FactoryApproved}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Export(ctx, req); err == nil || strings.Contains(err.Error(), "factory run not found") {
		t.Fatalf("running run: %v", err)
	}
	second := factoryTestRun()
	second.ID = strings.Repeat("d", 32)
	recorded := strings.Repeat("c", 64)
	binding := &identity.Binding{Kind: identity.Factory, ID: second.ID, Project: recorded}
	if err := f.storeReceipt(factoryReceipt{Run: second, Phase: domain.FactoryCompleted, Binding: binding}); err != nil {
		t.Fatal(err)
	}
	wrong := domain.FactoryExport{
		Project: second.Project, ID: second.ID, Role: domain.RoleReviewer,
		Preparation: second.Preparation, Candidate: strings.Repeat("f", 40),
	}
	if _, err := f.Export(ctx, wrong); !errors.Is(err, identity.ErrDenied) {
		t.Fatalf("role mismatch: %v", err)
	}
}

func TestFactoryExportEncodesBoundedBundle(t *testing.T) {
	f := openTestFactory(t, "")
	ctx := context.Background()
	run := factoryTestRun()
	recorded := strings.Repeat("c", 64)
	binding := &identity.Binding{Kind: identity.Factory, ID: run.ID, Project: recorded}
	if err := f.storeReceipt(factoryReceipt{Run: run, Phase: domain.FactoryCompleted, Binding: binding}); err != nil {
		t.Fatal(err)
	}
	candidate := strings.Repeat("f", 40)
	fake := f.terminal.Exec.(*factoryFakeExec)
	fake.run = func(_ context.Context, _ []byte, _ string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "inspect"):
			return []byte(`{"id":"` + recorded + `","running":true,"project":"` + run.Project + `","owner":"7","privileged":false,"userns":"private","mappings":{}}`), nil
		case strings.Contains(joined, "exec --user "):
			return []byte("bundle-bytes"), nil
		default:
			return nil, nil
		}
	}
	state, err := f.Export(ctx, domain.FactoryExport{
		Project: run.Project, ID: run.ID, Role: run.Role,
		Preparation: run.Preparation, Candidate: candidate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.ID != run.ID || state.Candidate != candidate || state.Phase != domain.FactoryCompleted {
		t.Fatalf("state: %+v", state)
	}
	decoded, err := base64.StdEncoding.DecodeString(state.Bundle)
	if err != nil || string(decoded) != "bundle-bytes" {
		t.Fatalf("bundle: %q %v", state.Bundle, err)
	}
}
