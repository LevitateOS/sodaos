package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
)

func (s *Store) ActivitySettings(ctx context.Context, repository int64) (factory.ActivitySettings, int64, error) {
	settings := factory.DefaultActivitySettings()
	var revision int64
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT revision,data FROM factory_activity_settings WHERE repository=$1`, repository).Scan(&revision, &raw)
	if errors.Is(err, ErrNotFound) {
		return settings, 0, nil
	}
	if err != nil {
		return settings, 0, err
	}
	err = json.Unmarshal(raw, &settings)
	return settings, revision, err
}

func (s *Store) SetActivitySettings(ctx context.Context, repository, actor, expected int64, settings factory.ActivitySettings) error {
	if repository <= 0 || actor <= 0 || expected < 0 || expected == math.MaxInt64 {
		return errors.New("invalid activity settings revision")
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	query := `UPDATE factory_activity_settings SET revision=revision+1,actor_id=$2,data=$4 WHERE repository=$1 AND revision=$3`
	if expected == 0 {
		query = `INSERT INTO factory_activity_settings(repository,actor_id,revision,data) SELECT $1,$2,1,$4::jsonb WHERE $3::bigint=0 ON CONFLICT DO NOTHING`
	}
	result, err := s.db.ExecContext(ctx, query, repository, actor, expected, string(raw))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrStaleRevision
	}
	return nil
}

// RepositoryActivity derives bounded local facts. Pending native effects and
// committed merges still awaiting original issue closure never become events.
func (s *Store) RepositoryActivity(ctx context.Context, repository int64, path string) (factory.RepositoryActivity, error) {
	settings, revision, err := s.ActivitySettings(ctx, repository)
	out := factory.RepositoryActivity{Repository: strconv.FormatInt(repository, 10), RepositoryPath: path, Revision: revision, Settings: settings, Items: []factory.ActivityItem{}}
	if err != nil {
		return out, err
	}
	add := func(id, kind string, issue, pr, t int64, summary string) {
		if issue <= 0 || t < 0 {
			return
		}
		url := path + "/issues/" + strconv.FormatInt(issue, 10)
		p := ""
		if pr > 0 {
			p = strconv.FormatInt(pr, 10)
		}
		if kind == "pr_created" {
			if pr <= 0 {
				return
			}
			url = path + "/pulls/" + p
		}
		out.Items = append(out.Items, factory.ActivityItem{ID: id, Kind: kind, Issue: strconv.FormatInt(issue, 10), PR: p, CreatedUnix: t, Summary: summary, URL: url})
	}
	if settings.IssueCreated {
		rows, e := s.db.QueryContext(ctx, `SELECT data FROM (SELECT DISTINCT ON (t.data->>'issue_id') t.data FROM factory_intent_tasks t JOIN factory_intent_revisions r ON r.id=t.revision JOIN factory_intent_generations g ON g.revision=t.revision WHERE r.data->>'repository'=$1 AND g.data->>'state'='completed' ORDER BY t.data->>'issue_id',t.revision) facts ORDER BY COALESCE((data->>'created_unix')::bigint,0) DESC,data->>'issue_id' LIMIT 60`, strconv.FormatInt(repository, 10))
		if e != nil {
			return out, e
		}
		for rows.Next() {
			var raw []byte
			if e = rows.Scan(&raw); e != nil {
				break
			}
			var b factory.IntentTaskBinding
			if e = json.Unmarshal(raw, &b); e != nil {
				break
			}
			if b.IssueID > 0 && b.AcceptanceID != "" {
				add(fmt.Sprintf("issue:%d", b.IssueID), "issue_created", b.IssueIndex, 0, b.CreatedUnix, fmt.Sprintf("Issue #%d created from accepted product intent.", b.IssueIndex))
			}
		}
		e = errors.Join(e, rows.Err(), rows.Close())
		if e != nil {
			return out, e
		}
	}
	if settings.PRCreated {
		rows, e := s.db.QueryContext(ctx, `SELECT data FROM factory_publications WHERE repository=$1 AND stage='published' ORDER BY seq DESC LIMIT 60`, repository)
		if e != nil {
			return out, e
		}
		seen := map[int64]bool{}
		for rows.Next() {
			var raw []byte
			if e = rows.Scan(&raw); e != nil {
				break
			}
			var p factory.Publication
			if e = json.Unmarshal(raw, &p); e != nil {
				break
			}
			if p.Outcome != factory.Succeeded || p.Reason != factory.PublishReasonLinked || p.FinishedUnix <= 0 || p.Publish.Effect != factory.OpEffectCommitted || p.Publish.Completion != factory.OpCompletionComplete || p.Repository != repository || p.Stage != factory.PublicationPublished || p.PRCreate.Effect != factory.OpEffectCommitted || p.PRCreate.Completion != factory.OpCompletionComplete || p.PRCreate.PRNumber != p.PRNumber || p.PRCreate.PRID != p.PRID || p.PRID <= 0 || p.PRNumber <= 0 || seen[p.PRID] {
				continue
			}
			seen[p.PRID] = true
			add(fmt.Sprintf("pr:%d", p.PRID), "pr_created", p.Issue, p.PRNumber, p.FinishedUnix, fmt.Sprintf("Pull request #%d created for issue #%d.", p.PRNumber, p.Issue))
		}
		e = errors.Join(e, rows.Err(), rows.Close())
		if e != nil {
			return out, e
		}
	}
	if settings.VerifiedMerge {
		rows, e := s.db.QueryContext(ctx, `SELECT data FROM factory_merges WHERE repository=$1 AND stage='merged' ORDER BY seq DESC LIMIT 60`, repository)
		if e != nil {
			return out, e
		}
		for rows.Next() {
			var raw []byte
			if e = rows.Scan(&raw); e != nil {
				break
			}
			var m factory.Merge
			if e = json.Unmarshal(raw, &m); e != nil {
				break
			}
			occ, e2 := strconv.ParseInt(m.OriginalClosingOccurrence, 10, 64)
			if m.ID == "" || m.Outcome != factory.Succeeded || m.Reason != factory.MergeReasonMerged || m.FinishedUnix <= 0 || m.ClosedUnix <= 0 || m.Operation.IssueID != m.IssueID || m.IssueID <= 0 || m.Repository != repository || m.Stage != factory.MergeMerged || m.Operation.Effect != factory.OpEffectCommitted || m.Operation.Completion != factory.OpCompletionComplete || m.Operation.PRNumber != m.PRNumber || m.Operation.PRID != m.PRID || m.Operation.MergedCommit != m.MergedCommit || m.MergedCommit == "" || m.MergedCommit != m.HeadOID || m.PRID <= 0 || m.PRNumber <= 0 || m.OriginalClosedUnix <= 0 || m.MergedUnix <= 0 || e2 != nil || occ <= 0 {
				continue
			}
			add("merge:"+m.ID, "verified_merge", m.Issue, m.PRNumber, m.FinishedUnix, fmt.Sprintf("Pull request #%d merged; original issue #%d closure confirmed.", m.PRNumber, m.Issue))
		}
		e = errors.Join(e, rows.Err(), rows.Close())
		if e != nil {
			return out, e
		}
	}
	sort.Slice(out.Items, func(i, j int) bool {
		if out.Items[i].CreatedUnix != out.Items[j].CreatedUnix {
			return out.Items[i].CreatedUnix > out.Items[j].CreatedUnix
		}
		return out.Items[i].ID < out.Items[j].ID
	})
	if len(out.Items) > 60 {
		out.Items = out.Items[:60]
	}
	return out, nil
}
