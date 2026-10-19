package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// recordFactoryFeedEventTx is private to the canonical store writers. A worker
// cannot call it or choose a Factory component through its MCP report tools.
// Events share the write's transaction and replay identity, so a card only
// exists when its observed effect committed, and a retry cannot duplicate it.
func recordFactoryFeedEventTx(ctx context.Context, tx *sql.Tx, repository, issue int64, assignment, identity, status string, content factory.FeedCommunication, at int64) error {
	component, ok := factory.FeedComponentByID(content.ComponentID)
	if !ok || component.Authority != "factory" || repository <= 0 || identity == "" {
		return errors.New("invalid canonical Factory feed event")
	}
	content.Kind = "system"
	if at <= 0 {
		at = time.Now().Unix()
	}
	key := fmt.Sprintf("factory:%d:%s:%s", repository, content.ComponentID, identity)
	digest := sha256.Sum256([]byte(key))
	id := hex.EncodeToString(digest[:16])
	key = hex.EncodeToString(digest[:])
	data, err := json.Marshal(map[string]any{
		"actor": "factory", "author_name": "Factory", "issue": issue,
		"assignment": assignment, "at": at, "status": status, "communication": content, "reply_to": content.ReplyTo, "event_identity": identity,
	})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_feed_comments(id,repository,revision,author,body,idempotency,data)
		VALUES($1,$2,0,'factory',$3,$4,$5) ON CONFLICT(repository,author,idempotency) DO NOTHING`,
		id, repository, content.Title+"\n\n"+content.Summary, key, string(data))
	return err
}

// A run may predate dispatch records (for example a supervised operator run).
// Only a recorded project/repository binding is sufficient for a scoped card.
func factoryRunFeedScopeTx(ctx context.Context, tx *sql.Tx, run, project string) (repository, issue int64, assignment string, err error) {
	err = tx.QueryRowContext(ctx, `SELECT repository,issue,id FROM factory_assignments WHERE run=$1`, run).Scan(&repository, &issue, &assignment)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `SELECT repository_id FROM projects WHERE id=$1`, project).Scan(&repository)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
	}
	return
}
