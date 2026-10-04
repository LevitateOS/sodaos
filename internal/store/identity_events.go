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
