package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func (s *Store) identityAtomic(ctx context.Context, operation func(*tx) error) error {
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = operation(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func appendIdentityEvent(ctx context.Context, tx *tx, event identity.Event) error {
	if event.OwnerID == 0 {
		var generation int64
		if err := tx.queryRow(ctx, `SELECT owner_id,generation FROM identity_connections WHERE id=?`, event.ConnectionID).Scan(&event.OwnerID, &generation); err != nil {
			return err
		}
		if event.Generation == 0 {
			event.Generation = generation
		}
	}
	if event.ActorID == 0 {
		event.ActorID = event.OwnerID
	}
	event.Time = time.Now().UTC()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = tx.exec(ctx, `INSERT INTO identity_events(owner_id,connection_id,data) VALUES(?,?,?)`, event.OwnerID, event.ConnectionID, data)
	return err
}

func leaseEvent(l identity.Lease, action string) identity.Event {
	return identity.Event{Action: action, ActorID: l.ActorID, ConnectionID: l.ConnectionID, LeaseID: l.ID, ProjectID: l.ProjectID, GrantID: l.GrantID, ExecutionID: l.ExecutionID, Kind: l.Kind, Generation: l.Generation}
}

func (s *Store) IdentityEvents(ctx context.Context, owner int64, id string) ([]identity.Event, error) {
	if owner <= 0 {
		return nil, identity.ErrDenied
	}
	rows, err := s.query(ctx, `SELECT id,data FROM identity_events WHERE owner_id=? AND connection_id=? ORDER BY id DESC LIMIT 200`, owner, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []identity.Event{}
	for rows.Next() {
		var id int64
		var data []byte
		var event identity.Event
		if err = rows.Scan(&id, &data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &event); err != nil {
			return nil, err
		}
		event.ID = id
		out = append(out, event)
	}
	return out, rows.Err()
}
