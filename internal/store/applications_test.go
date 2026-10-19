package store

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/levitateos/sodaos/internal/application"
)

func applicationFixture() application.Record {
	return application.Record{Application: application.Spec{ID: "a" + strings.Repeat("1", 24), Revision: 1, Name: "Example", ArtifactSHA256: strings.Repeat("2", 64), BundleSHA256: strings.Repeat("3", 64), ToolsSHA256: strings.Repeat("4", 64), RuntimeImage: "sha256:" + strings.Repeat("5", 64), Image: "sha256:" + strings.Repeat("6", 64), Data: "a" + strings.Repeat("7", 24), OwnerTeam: "12345678-1234-1234-1234-123456789abc", Origin: "https://app.example.test", Port: 20001}, AdministratorID: 41}
}

func TestApplicationProjectAssociationNeedsTheApplicationAdministrator(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	record := applicationFixture()
	if err := s.CreateApplication(ctx, record); err != nil {
		t.Fatal(err)
	}
	if err := s.SetApplicationProject(ctx, record.Application.ID, 42, 7); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another account changed project: %v", err)
	}
	if err := s.SetApplicationProject(ctx, record.Application.ID, 41, 7); err != nil {
		t.Fatal(err)
	}
	repository, err := s.ApplicationProject(ctx, record.Application.ID)
	if err != nil || repository != 7 {
		t.Fatalf("source project lost: %d %v", repository, err)
	}
	if _, err = s.Application(ctx, record.Application.ID, 42); !errors.Is(err, ErrNotFound) {
		t.Fatal("project link expanded application permission")
	}
}

func TestApplicationPersistenceAndAdministration(t *testing.T) {
	s, dsn := postgresFixture(t, nil)
	ctx := context.Background()
	r := applicationFixture()
	if err := s.CreateApplication(ctx, r); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Application(ctx, r.Application.ID, 42); !errors.Is(err, ErrNotFound) {
		t.Fatalf("private record leaked: %v", err)
	}
	got, err := s.Application(ctx, r.Application.ID, 41)
	if err != nil || !reflect.DeepEqual(got, r) {
		t.Fatalf("initial unknown %+v %v", got, err)
	}
	o := application.Observation{ID: r.Application.ID, Revision: 1, Container: strings.Repeat("a", 64), Image: r.Application.Image, Origin: r.Application.Origin, Running: true, Ready: true}
	if err := s.ObserveApplication(ctx, r.Application.ID, 41, 1, nil, &o); err != nil {
		t.Fatal(err)
	}
	if err := s.ObserveApplication(ctx, r.Application.ID, 41, 1, nil, nil); err == nil {
		t.Fatal("stale observation CAS accepted")
	}
	bad := o
	bad.Image = "sha256:" + strings.Repeat("b", 64)
	if err := s.ObserveApplication(ctx, r.Application.ID, 41, 1, &o, &bad); err == nil {
		t.Fatal("unadmitted image observation accepted")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	got, err = s.Application(ctx, r.Application.ID, 41)
	if err != nil || !application.SameObservation(got.Observation, &o) {
		t.Fatalf("reopen observation %+v %v", got, err)
	}
	if err := s.ObserveApplication(ctx, r.Application.ID, 41, 1, &o, nil); err != nil {
		t.Fatal(err)
	}
	got, err = s.Application(ctx, r.Application.ID, 41)
	if err != nil || got.Observation != nil {
		t.Fatal("uncertain native result inferred readiness")
	}
}

func TestApplicationReplacementCASAndHistory(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	r := applicationFixture()
	if err := s.CreateApplication(ctx, r); err != nil {
		t.Fatal(err)
	}
	a, b := r.Application, r.Application
	a.Revision = 2
	b.Revision = 2
	a.Image = "sha256:" + strings.Repeat("a", 64)
	b.Image = "sha256:" + strings.Repeat("b", 64)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, next := range []application.Spec{a, b} {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- s.ReplaceApplication(ctx, 41, 1, next) }()
	}
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("CAS winners %d", success)
	}
	got, err := s.Application(ctx, r.Application.ID, 41)
	if err != nil || got.Application.Revision != 2 || got.Observation != nil {
		t.Fatalf("replacement %+v %v", got, err)
	}
	old, err := s.ApplicationRevision(ctx, r.Application.ID, 41, 1)
	if err != nil || old != r.Application {
		t.Fatalf("prior identity lost %+v %v", old, err)
	}
	changed := got.Application
	changed.Revision = 3
	changed.Data = "a" + strings.Repeat("c", 24)
	if err := s.ReplaceApplication(ctx, 41, 2, changed); err == nil {
		t.Fatal("private data silently changed")
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE application_revisions SET spec='{}' WHERE application_id=$1`, r.Application.ID); err == nil {
		t.Fatal("revision history mutable")
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE applications SET administrator_id=42 WHERE id=$1`, r.Application.ID); err == nil {
		t.Fatal("administrator grant mutable")
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM application_revisions WHERE application_id=$1`, r.Application.ID).Scan(&n); err != nil || n != 2 {
		t.Fatalf("history count %d %v", n, err)
	}
}

func TestApplicationEndpointAndDataIsolation(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	r := applicationFixture()
	if err := s.CreateApplication(ctx, r); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"hostname", "port", "data"} {
		next := r
		next.Application.ID = "a" + strings.Repeat("a", 24)
		next.Application.Data = "a" + strings.Repeat("b", 24)
		next.Application.Port = 20002
		next.Application.Origin = "https://other.example.test"
		switch field {
		case "hostname":
			next.Application.Origin = "https://app.example.test:8443"
		case "port":
			next.Application.Port = r.Application.Port
		case "data":
			next.Application.Data = r.Application.Data
		}
		if err := s.CreateApplication(ctx, next); err == nil {
			t.Fatalf("shared %s admitted", field)
		}
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM applications`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("partial failed creation retained %d %v", n, err)
	}
}
