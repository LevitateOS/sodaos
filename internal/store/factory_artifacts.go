package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/levitateos/sodaos/internal/factory"
	"strconv"
)

func initializeArtifactSchema(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = ensureFactoryArtifacts(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
func ensureFactoryArtifacts(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS factory_feed_artifacts(id TEXT PRIMARY KEY,repository BIGINT NOT NULL,assignment TEXT NOT NULL,name TEXT NOT NULL,media_type TEXT NOT NULL,content BYTEA NOT NULL,complete BOOLEAN NOT NULL DEFAULT FALSE,digest TEXT,CHECK(octet_length(content)<=8388608),CHECK(digest IS NULL OR digest ~ '^[a-f0-9]{64}$'))`); err != nil {
		return err
	}
	// Convergent shape for lazy tables created before the digest
	// column existed; versioned schema tables take the format bump
	// instead.
	_, err := tx.ExecContext(ctx, `ALTER TABLE factory_feed_artifacts ADD COLUMN IF NOT EXISTS digest TEXT CHECK(digest IS NULL OR digest ~ '^[a-f0-9]{64}$')`)
	return err
}
func artifactID(cap factory.MCPCapability, upload string) string {
	return factory.MCPInputDigest([]byte("artifact\n" + cap.AssignmentID + "\n" + upload))[:32]
}
func artifactHref(repo int64, id string) string {
	return "/api/factory/artifacts/" + id + "?repository=" + strconv.FormatInt(repo, 10)
}
func recordMCPArtifactTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, sub factory.MCPSubmission, receipt *factory.MCPReceipt) error {
	if sub.Tool != factory.MCPToolPublishArtifact {
		return nil
	}
	u, chunk, err := factory.DecodeArtifactUpload(sub.Payload)
	if err != nil {
		return err
	}
	id := artifactID(cap, u.UploadID)
	if u.Offset == 0 {
		var count int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_feed_artifacts WHERE assignment=$1`, cap.AssignmentID).Scan(&count); err != nil {
			return err
		}
		if count >= 32 {
			return errors.New("at most 32 artifacts per assignment")
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO factory_feed_artifacts(id,repository,assignment,name,media_type,content,complete) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, cap.Repository, cap.AssignmentID, u.Name, u.MediaType, chunk, u.Final)
	} else {
		var result sql.Result
		result, err = tx.ExecContext(ctx, `UPDATE factory_feed_artifacts SET content=content || $1,complete=$2 WHERE id=$3 AND repository=$4 AND assignment=$5 AND name=$6 AND media_type=$7 AND octet_length(content)=$8 AND NOT complete`, chunk, u.Final, id, cap.Repository, cap.AssignmentID, u.Name, u.MediaType, u.Offset)
		if err == nil {
			n, _ := result.RowsAffected()
			if n != 1 {
				return errors.New("artifact upload offset or metadata changed")
			}
		}
	}
	if err != nil {
		return err
	}
	receipt.ArtifactID = id
	if u.Final {
		// The digest attests the complete bytes the worker uploaded;
		// finish citations reference this value so settlement can
		// resolve them back to this assignment's artifact.
		var content []byte
		if err := tx.QueryRowContext(ctx, `SELECT content FROM factory_feed_artifacts WHERE id=$1 AND repository=$2 AND complete`, id, cap.Repository).Scan(&content); err != nil {
			return err
		}
		digest := sha256.Sum256(content)
		receipt.ArtifactDigest = hex.EncodeToString(digest[:])
		if _, err := tx.ExecContext(ctx, `UPDATE factory_feed_artifacts SET digest=$1 WHERE id=$2 AND repository=$3`, receipt.ArtifactDigest, id, cap.Repository); err != nil {
			return err
		}
		feedContent, err := artifactCommunicationTx(ctx, tx, cap, sub)
		if err != nil {
			return err
		}
		return recordMCPFeedContentTx(ctx, tx, cap, sub, *feedContent, receipt)
	}
	return nil
}
func artifactCommunicationTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, sub factory.MCPSubmission) (*factory.FeedCommunication, error) {
	var u factory.ArtifactUpload
	if err := json.Unmarshal(sub.Payload, &u); err != nil {
		return nil, err
	}
	if !u.Final {
		return nil, nil
	}
	id := artifactID(cap, u.UploadID)
	var a factory.FeedArtifact
	a.ID = id
	a.Href = artifactHref(cap.Repository, id)
	if err := tx.QueryRowContext(ctx, `SELECT name,media_type,octet_length(content),COALESCE(digest,'') FROM factory_feed_artifacts WHERE id=$1 AND repository=$2 AND complete`, id, cap.Repository).Scan(&a.Name, &a.MediaType, &a.Size, &a.Digest); err != nil {
		return nil, err
	}
	return &factory.FeedCommunication{Kind: "artifact", Title: a.Name, Summary: fmt.Sprintf("Artifact available · %d bytes", a.Size), Artifacts: []factory.FeedArtifact{a}}, nil
}
func resolveCommunicationArtifactsTx(ctx context.Context, tx *sql.Tx, repo int64, content *factory.FeedCommunication) error {
	for i, ref := range content.Artifacts {
		a := factory.FeedArtifact{ID: ref.ID, Href: artifactHref(repo, ref.ID)}
		if err := tx.QueryRowContext(ctx, `SELECT name,media_type,octet_length(content),COALESCE(digest,'') FROM factory_feed_artifacts WHERE id=$1 AND repository=$2 AND complete`, ref.ID, repo).Scan(&a.Name, &a.MediaType, &a.Size, &a.Digest); err != nil {
			return errors.New("artifact is unavailable in this project")
		}
		content.Artifacts[i] = a
	}
	return nil
}

// ResolveArtifactDigest maps one cited content digest to a complete
// artifact of the assignment. A digest with no complete row here is
// unknown, foreign or unfinished, and the caller must refuse it.
func (s *Store) ResolveArtifactDigest(ctx context.Context, repository int64, assignment, digest string) (string, error) {
	if repository < 1 || !factory.ValidID(assignment) || !factory.ValidDigest(digest) {
		return "", errors.New("artifact citation is invalid")
	}
	var id string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_artifacts WHERE repository=$1 AND assignment=$2 AND digest=$3 AND complete`, repository, assignment, digest).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return id, nil
}

func (s *Store) FeedArtifact(ctx context.Context, repo int64, id string) (factory.FeedArtifact, []byte, error) {
	a := factory.FeedArtifact{ID: id, Href: artifactHref(repo, id)}
	var content []byte
	err := s.db.QueryRowContext(ctx, `SELECT name,media_type,content,COALESCE(digest,'') FROM factory_feed_artifacts WHERE id=$1 AND repository=$2 AND complete`, id, repo).Scan(&a.Name, &a.MediaType, &content, &a.Digest)
	a.Size = len(content)
	return a, content, err
}
