package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// The library adds its own tables without rewriting the existing appliance data.
// Its current projection and immutable revisions commit with the MCP receipt.
func initializeLibrarySchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS factory_library_items(id TEXT PRIMARY KEY,repository BIGINT NOT NULL,revision BIGINT NOT NULL,kind TEXT NOT NULL,removed BOOLEAN NOT NULL DEFAULT FALSE,data JSONB NOT NULL);
CREATE INDEX IF NOT EXISTS factory_library_repository ON factory_library_items(repository,id);
CREATE INDEX IF NOT EXISTS factory_library_search ON factory_library_items USING GIN(to_tsvector('simple',coalesce(data->>'title','') || ' ' || coalesce(data->>'body','') || ' ' || coalesce(data->>'scope','')));
CREATE TABLE IF NOT EXISTS factory_library_revisions(item TEXT NOT NULL REFERENCES factory_library_items(id),revision BIGINT NOT NULL,repository BIGINT NOT NULL,data JSONB NOT NULL,PRIMARY KEY(item,revision));`)
	return err
}

type libraryQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readLibraryItem(ctx context.Context, q libraryQuerier, repository int64, id string, lock bool) (factory.LibraryItem, error) {
	var item factory.LibraryItem
	var raw []byte
	query := `SELECT data FROM factory_library_items WHERE repository=$1 AND id=$2`
	if lock {
		query += " FOR UPDATE"
	}
	if err := q.QueryRowContext(ctx, query, repository, id).Scan(&raw); err != nil {
		return item, err
	}
	err := json.Unmarshal(raw, &item)
	return item, err
}

func (s *Store) LibrarySearch(ctx context.Context, repository int64, query, kind, cursor string, includeRemoved bool) (factory.LibraryPage, error) {
	page := factory.LibraryPage{Items: []factory.LibraryItem{}, Complete: true}
	if repository < 1 || len(query) > 512 || (cursor != "" && !factory.ValidID(cursor)) || (kind != "" && kind != "knowledge" && kind != "decision") {
		return page, errors.New("invalid library search")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_library_items WHERE repository=$1 AND ($2 OR NOT removed) AND ($3='' OR kind=$3) AND id>$4 AND ($5='' OR to_tsvector('simple',coalesce(data->>'title','') || ' ' || coalesce(data->>'body','') || ' ' || coalesce(data->>'scope','')) @@ plainto_tsquery('simple',$5)) ORDER BY id LIMIT 51`, repository, includeRemoved, kind, cursor, query)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	bytes := 0
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return page, err
		}
		if len(page.Items) == 50 || (bytes+len(raw) > factory.MCPMaxPageBytes && len(page.Items) > 0) {
			page.Complete = false
			page.NextCursor = page.Items[len(page.Items)-1].ID
			break
		}
		var item factory.LibraryItem
		if err = json.Unmarshal(raw, &item); err != nil {
			return page, err
		}
		page.Items = append(page.Items, item)
		bytes += len(raw)
	}
	return page, rows.Err()
}

// LibraryInventory reads one bounded collection-scoped inventory page
// with reference analysis over exactly the inventoried items. Total
// always names the scope size and Complete is false whenever items
// remain past NextCursor, so a partial page is never mistaken for a
// complete inventory. Stale inputs refuse: an invalid cursor, kind or
// repository errors instead of returning a silently partial page.
func (s *Store) LibraryInventory(ctx context.Context, repository int64, kind, cursor string, includeRemoved bool) (factory.LibraryInventory, error) {
	return libraryInventory(ctx, s.db, repository, kind, cursor, includeRemoved)
}

func libraryInventory(ctx context.Context, q libraryQuerier, repository int64, kind, cursor string, includeRemoved bool) (factory.LibraryInventory, error) {
	out := factory.LibraryInventory{Items: []factory.LibraryItem{}, Complete: true}
	if repository < 1 || (cursor != "" && !factory.ValidID(cursor)) || (kind != "" && kind != "knowledge" && kind != "decision") {
		return out, errors.New("invalid library inventory")
	}
	var total int64
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM factory_library_items WHERE repository=$1 AND ($2 OR NOT removed) AND ($3='' OR kind=$3)`, repository, includeRemoved, kind).Scan(&total); err != nil {
		return out, err
	}
	out.Total = total
	rows, err := q.QueryContext(ctx, `SELECT data FROM factory_library_items WHERE repository=$1 AND ($2 OR NOT removed) AND ($3='' OR kind=$3) AND id>$4 ORDER BY id LIMIT 51`, repository, includeRemoved, kind, cursor)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	bytes := 0
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return out, err
		}
		if len(out.Items) == 50 || (bytes+len(raw) > factory.MCPMaxPageBytes && len(out.Items) > 0) {
			out.Complete = false
			out.NextCursor = out.Items[len(out.Items)-1].ID
			break
		}
		var item factory.LibraryItem
		if err = json.Unmarshal(raw, &item); err != nil {
			return out, err
		}
		out.Items = append(out.Items, item)
		bytes += len(raw)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) == 0 {
		return out, nil
	}
	revRows, err := q.QueryContext(ctx, `SELECT item,COUNT(*) FROM factory_library_revisions WHERE repository=$1 GROUP BY item`, repository)
	if err != nil {
		return out, err
	}
	defer revRows.Close()
	counts := map[string]int64{}
	for revRows.Next() {
		var item string
		var count int64
		if err = revRows.Scan(&item, &count); err != nil {
			return out, err
		}
		counts[item] = count
	}
	if err = revRows.Err(); err != nil {
		return out, err
	}
	idRows, err := q.QueryContext(ctx, `SELECT id,removed FROM factory_library_items WHERE repository=$1`, repository)
	if err != nil {
		return out, err
	}
	defer idRows.Close()
	removedByID := map[string]bool{}
	for idRows.Next() {
		var id string
		var removed bool
		if err = idRows.Scan(&id, &removed); err != nil {
			return out, err
		}
		removedByID[id] = removed
	}
	if err = idRows.Err(); err != nil {
		return out, err
	}
	inPage := map[string]factory.LibraryItem{}
	for _, item := range out.Items {
		inPage[item.ID] = item
		if n, ok := counts[item.ID]; ok {
			if out.Revisions == nil {
				out.Revisions = map[string]int64{}
			}
			out.Revisions[item.ID] = n
		}
	}
	addPair := func(seen map[string]bool, dst *[][]string, a, b string) {
		if a == b {
			return
		}
		x, y := a, b
		if x > y {
			x, y = y, x
		}
		key := x + "\x00" + y
		if seen[key] {
			return
		}
		seen[key] = true
		*dst = append(*dst, []string{x, y})
	}
	seenDuplicates := map[string]bool{}
	seenContradictions := map[string]bool{}
	byTitle := map[string]string{}
	for _, item := range out.Items {
		for _, rel := range item.Relations {
			// A target that is gone — or removed from ordinary
			// retrieval — dangles for maintenance purposes even
			// though its row survives for history. A removed
			// target still present in the page keeps its
			// reference analysis; a missing one has none.
			if removed, ok := removedByID[rel.Target]; !ok || removed {
				out.Orphans = append(out.Orphans, factory.LibraryOrphanRef{FromID: item.ID, Relation: rel.Kind, TargetID: rel.Target})
				if !ok {
					continue
				}
			}
			if rel.Kind == "contradicts" {
				addPair(seenContradictions, &out.Contradictions, item.ID, rel.Target)
			}
			if rel.Kind == "duplicate_of" {
				addPair(seenDuplicates, &out.Duplicates, item.ID, rel.Target)
			}
			if _, ok := inPage[rel.Target]; ok {
				if out.Incoming == nil {
					out.Incoming = map[string][]factory.LibraryIncomingRef{}
				}
				out.Incoming[rel.Target] = append(out.Incoming[rel.Target], factory.LibraryIncomingRef{FromID: item.ID, FromRevision: item.Revision, Relation: rel.Kind})
			}
		}
		title := strings.ToLower(strings.TrimSpace(item.Title))
		if title == "" {
			continue
		}
		key := item.Kind + "\x00" + title
		if first, ok := byTitle[key]; ok {
			addPair(seenDuplicates, &out.Duplicates, first, item.ID)
		} else {
			byTitle[key] = item.ID
		}
	}
	for _, refs := range out.Incoming {
		slices.SortFunc(refs, func(a, b factory.LibraryIncomingRef) int {
			if a.FromID != b.FromID {
				return strings.Compare(a.FromID, b.FromID)
			}
			return strings.Compare(a.Relation, b.Relation)
		})
	}
	sortPairs := func(pairs [][]string) {
		slices.SortFunc(pairs, func(a, b []string) int {
			if a[0] != b[0] {
				return strings.Compare(a[0], b[0])
			}
			return strings.Compare(a[1], b[1])
		})
	}
	sortPairs(out.Duplicates)
	sortPairs(out.Contradictions)
	slices.SortFunc(out.Orphans, func(a, b factory.LibraryOrphanRef) int {
		if a.FromID != b.FromID {
			return strings.Compare(a.FromID, b.FromID)
		}
		if a.Relation != b.Relation {
			return strings.Compare(a.Relation, b.Relation)
		}
		return strings.Compare(a.TargetID, b.TargetID)
	})
	return out, nil
}

func (s *Store) LibraryRead(ctx context.Context, repository int64, id string, includeRemoved bool) (factory.LibraryRead, error) {
	out := factory.LibraryRead{History: []factory.LibraryRevision{}}
	item, err := readLibraryItem(ctx, s.db, repository, id, false)
	if err != nil {
		return out, err
	}
	if item.Removed && !includeRemoved {
		return out, ErrNotFound
	}
	out.Item = item
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_library_revisions WHERE repository=$1 AND item=$2 ORDER BY revision DESC LIMIT 20`, repository, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	bytes := 0
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return out, err
		}
		if bytes+len(raw) > factory.MCPMaxPageBytes {
			break
		}
		var revision factory.LibraryRevision
		if err = json.Unmarshal(raw, &revision); err != nil {
			return out, err
		}
		out.History = append(out.History, revision)
		bytes += len(raw)
	}
	return out, rows.Err()
}

func writeLibraryRevision(ctx context.Context, tx *sql.Tx, item factory.LibraryItem, reason, actor, submission string) error {
	item.UpdatedAt = time.Now().Unix()
	raw, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_library_items(id,repository,revision,kind,removed,data) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(id) DO UPDATE SET revision=excluded.revision,removed=excluded.removed,data=excluded.data`, item.ID, item.Repository, item.Revision, item.Kind, item.Removed, string(raw))
	if err != nil {
		return err
	}
	revision, err := json.Marshal(factory.LibraryRevision{Item: item, Reason: reason, Actor: actor, Submission: submission})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_library_revisions(item,revision,repository,data) VALUES($1,$2,$3,$4)`, item.ID, item.Revision, item.Repository, string(revision))
	return err
}

func createCommunicationLibraryTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, submission factory.MCPSubmission, content *factory.FeedCommunication, postID string, receipt *factory.MCPReceipt) error {
	if submission.Tool != factory.MCPToolLearned && submission.Tool != factory.MCPToolRecordDecision {
		return nil
	}
	if content.RevisesItem != "" {
		return amendCommunicationLibraryTx(ctx, tx, cap, submission, content, postID, receipt)
	}
	item := factory.LibraryItem{ID: factory.MCPInputDigest([]byte("library\n" + postID))[:32], Repository: cap.Repository, Revision: 1, Kind: "knowledge", Title: content.Title, Body: content.Summary, Scope: content.Scope, EvidenceRefs: content.EvidenceRefs, OwnerRole: string(cap.Profile), Assignment: cap.AssignmentID, SourcePost: postID, State: "observed"}
	if submission.Tool == factory.MCPToolRecordDecision {
		item.Kind = "decision"
		item.State = "current"
		item.Decision = content.Decision
		// The actual decision owner defaults to the logging worker;
		// naming another owner references a choice made by an
		// authorized person/controller. The revision actor below
		// always stays the logging worker.
		if owner := strings.TrimSpace(content.DecisionOwner); owner != "" {
			item.OwnerRole = owner
		}
		item.Rationale = content.Rationale
		item.Alternatives = content.Alternatives
		if item.Scope == "" {
			item.Scope = cap.Subject
		}
	}
	if err := writeLibraryRevision(ctx, tx, item, "Recorded by "+string(cap.Profile), "worker:"+cap.AssignmentID, submission.SubmissionID); err != nil {
		return err
	}
	content.LibraryItemID = item.ID
	content.LibraryRevision = 1
	receipt.LibraryItemID = item.ID
	return nil
}

// amendCommunicationLibraryTx revises a recorded knowledge entry under
// compare-and-swap: the expected revision must match the current row,
// so concurrent amendments serialize instead of silently overwriting.
// An amendment replaces the entry body and restores the observed
// state; a challenge keeps the body and marks it contested. Either
// records a full revision with its actor and submission, preserving
// one identity across work, feed and retrieval. Removed entries and
// decisions (R32.5) refuse here.
func amendCommunicationLibraryTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, submission factory.MCPSubmission, content *factory.FeedCommunication, postID string, receipt *factory.MCPReceipt) error {
	item, err := readLibraryItem(ctx, tx, cap.Repository, content.RevisesItem, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("amended entry is not available in this project")
		}
		return err
	}
	if item.Removed {
		return errors.New("removed entry cannot be revised")
	}
	if item.Kind != "knowledge" {
		return errors.New("decisions amend through their own owner")
	}
	if item.Revision != content.ExpectedRevision {
		return errors.New("knowledge revision changed under this amendment")
	}
	verb := "Amended"
	// A challenge keeps the recorded body and only marks it contested;
	// the counter-statement lives on the feed post and in the revision
	// reason. An amendment replaces the body and restores observed.
	if content.Challenge {
		item.State = "contested"
		verb = "Challenged"
	} else {
		item.Title, item.Body, item.Scope, item.EvidenceRefs = content.Title, content.Summary, content.Scope, content.EvidenceRefs
		item.State = "observed"
	}
	item.Revision++
	if err := writeLibraryRevision(ctx, tx, item, verb+" by "+string(cap.Profile)+": "+content.Title, "worker:"+cap.AssignmentID, submission.SubmissionID); err != nil {
		return err
	}
	content.LibraryItemID = item.ID
	content.LibraryRevision = item.Revision
	receipt.LibraryItemID = item.ID
	return nil
}

// LibraryPreview checks the exact items a proposed mutation would touch. It
// returns current revisions and references, without reserving or applying them.
func (s *Store) LibraryPreview(ctx context.Context, cap factory.MCPCapability, tool factory.MCPTool, change factory.LibraryChange) (map[string]any, error) {
	if string(cap.Profile) != "library_admin" {
		return nil, errors.New("library maintenance profile required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	items, err := applyLibraryChangeTx(ctx, tx, cap, tool, change, "preview", true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"tool": tool, "component_id": factory.LibraryComponent(tool), "items": items, "applied": false}, nil
}

// libraryAffectedRefsTx collects every relation another repository
// item holds toward a changed item: the maintenance blast radius the
// feed post carries. It reads the just-written rows in the same
// transaction, so the mapping reflects the applied operation.
func libraryAffectedRefsTx(ctx context.Context, tx *sql.Tx, repository int64, changed map[string]bool) ([]factory.LibraryAffectedRef, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,data->'relations' FROM factory_library_items WHERE repository=$1`, repository)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []factory.LibraryAffectedRef
	for rows.Next() {
		var id string
		var raw []byte
		if err = rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var relations []factory.LibraryRelation
		if err = json.Unmarshal(raw, &relations); err != nil {
			return nil, err
		}
		for _, rel := range relations {
			if changed[rel.Target] {
				out = append(out, factory.LibraryAffectedRef{ItemID: rel.Target, FromID: id, Relation: rel.Kind})
			}
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b factory.LibraryAffectedRef) int {
		if a.ItemID != b.ItemID {
			return strings.Compare(a.ItemID, b.ItemID)
		}
		if a.FromID != b.FromID {
			return strings.Compare(a.FromID, b.FromID)
		}
		return strings.Compare(a.Relation, b.Relation)
	})
	return out, nil
}

func recordMCPLibraryTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, submission factory.MCPSubmission, receipt *factory.MCPReceipt) error {
	if !factory.MCPIsLibraryMutation(submission.Tool) {
		return nil
	}
	if string(cap.Profile) != "library_admin" || !factory.MCPToolAdmitted(cap.Profile, submission.Tool) {
		return errors.New("library maintenance profile required")
	}
	change, err := factory.DecodeLibraryChange(submission.Tool, submission.Payload)
	if err != nil {
		return err
	}
	items, err := applyLibraryChangeTx(ctx, tx, cap, submission.Tool, change, submission.SubmissionID, false)
	if err != nil {
		return err
	}
	component, _ := factory.FeedComponentByID(factory.LibraryComponent(submission.Tool))
	content := factory.FeedCommunication{Kind: "library_maintenance", ComponentID: component.ID, Title: change.Title, Summary: change.Summary, EvidenceRefs: change.EvidenceRefs, Completed: change.Completed, Remaining: change.Remaining}
	if content.Title == "" {
		content.Title = component.Label
	}
	if content.Summary == "" {
		content.Summary = change.Reason
	}
	for _, item := range items {
		state := item.State
		if item.Removed {
			state = "removed"
		}
		content.Items = append(content.Items, factory.FeedCommunicationItem{Label: item.Title, Detail: item.ID + " · revision " + strconv.FormatInt(item.Revision, 10), State: state})
	}
	if len(items) > 0 {
		primary := items[0]
		// Split and consolidate retain removed source revisions alongside
		// active replacements. The primary receipt links an active result;
		// random item-ID ordering must not sometimes select its retired source.
		for _, item := range items {
			if !item.Removed {
				primary = item
				break
			}
		}
		content.LibraryItemID = primary.ID
		content.LibraryRevision = primary.Revision
		receipt.LibraryItemID = primary.ID
		receipt.LibraryRevision = primary.Revision
		refs := make([]factory.LibraryRef, 0, len(items))
		changed := make(map[string]bool, len(items))
		for _, item := range items {
			refs = append(refs, factory.LibraryRef{ID: item.ID, Revision: item.Revision})
			changed[item.ID] = true
		}
		slices.SortFunc(refs, func(a, b factory.LibraryRef) int { return strings.Compare(a.ID, b.ID) })
		content.LibraryItems = refs
		affected, err := libraryAffectedRefsTx(ctx, tx, cap.Repository, changed)
		if err != nil {
			return err
		}
		content.AffectedRefs = affected
	}
	if change.Content != nil {
		content.After = change.Content.Body
	}
	content.Before = change.Reason
	if err = recordMCPFeedContentTx(ctx, tx, cap, submission, content, receipt); err != nil {
		return err
	}
	if !factory.MCPIsTerminal(submission.Tool) {
		receipt.State = factory.MCPReceiptAccepted
	}
	return nil
}

func applyLibraryChangeTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, tool factory.MCPTool, c factory.LibraryChange, submission string, preview bool) ([]factory.LibraryItem, error) {
	refs := map[string]int64{}
	add := func(ref factory.LibraryRef) error {
		if revision, ok := refs[ref.ID]; ok && revision != ref.Revision {
			return errors.New("conflicting expected revisions")
		}
		refs[ref.ID] = ref.Revision
		return nil
	}
	if c.ID != "" {
		if err := add(factory.LibraryRef{ID: c.ID, Revision: c.Revision}); err != nil {
			return nil, err
		}
	}
	if c.Target != nil {
		if err := add(*c.Target); err != nil {
			return nil, err
		}
	}
	for _, ref := range c.Sources {
		if err := add(ref); err != nil {
			return nil, err
		}
	}
	ids := make([]string, 0, len(refs))
	for id := range refs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	items := map[string]factory.LibraryItem{}
	for _, id := range ids {
		item, err := readLibraryItem(ctx, tx, cap.Repository, id, true)
		if err != nil {
			return nil, err
		}
		if item.Revision != refs[id] {
			return nil, fmt.Errorf("library item %s changed; read its current revision", id)
		}
		items[id] = item
	}
	changed := map[string]factory.LibraryItem{}
	active := func(item factory.LibraryItem) error {
		if item.Removed {
			return errors.New("removed library item must be restored first")
		}
		return nil
	}
	put := func(item factory.LibraryItem) { item.Revision++; changed[item.ID] = item }
	content := func(item factory.LibraryItem, body factory.LibraryContent) factory.LibraryItem {
		item.Title = body.Title
		item.Body = body.Body
		item.Scope = body.Scope
		item.EvidenceRefs = mergeLibraryRefs(item.EvidenceRefs, body.EvidenceRefs)
		return item
	}
	switch factory.LibraryComponent(tool) {
	case "LA01", "LA08", "LA09":
		return []factory.LibraryItem{}, nil
	case "LA02":
		item := items[c.ID]
		if err := active(item); err != nil {
			return nil, err
		}
		if item.Kind == "decision" && c.Content.Scope != item.Scope {
			return nil, errors.New("editing cannot change a decision's authorized scope")
		}
		put(content(item, *c.Content))
	case "LA03":
		target := items[c.Target.ID]
		if err := active(target); err != nil {
			return nil, err
		}
		if target.Kind == "decision" && c.Content.Scope != target.Scope {
			return nil, errors.New("consolidation cannot change a decision's authorized scope")
		}
		target = content(target, *c.Content)
		for _, source := range c.Sources {
			item := items[source.ID]
			if item.ID == target.ID {
				return nil, errors.New("consolidation source is also the target")
			}
			if err := active(item); err != nil {
				return nil, err
			}
			if item.Kind != target.Kind {
				return nil, errors.New("cannot consolidate knowledge with decisions")
			}
			if item.Kind == "decision" && (item.Decision != target.Decision || item.Scope != target.Scope || item.OwnerRole != target.OwnerRole) {
				return nil, errors.New("conflicting decisions need their owner to record a replacement")
			}
			target.EvidenceRefs = mergeLibraryRefs(target.EvidenceRefs, item.EvidenceRefs)
			target.DerivedFrom = mergeLibraryRefs(target.DerivedFrom, []string{item.ID})
			item.Removed = true
			item.Replacements = []string{target.ID}
			put(item)
		}
		put(target)
	case "LA04":
		source := items[c.ID]
		if err := active(source); err != nil {
			return nil, err
		}
		replacements := []string{}
		for i, part := range c.Items {
			if source.Kind == "decision" && part.Scope != source.Scope {
				return nil, errors.New("split decision parts must preserve authorized scope")
			}
			item := content(source, part)
			item.ID = factory.MCPInputDigest([]byte(cap.AssignmentID + "\n" + submission + "\n" + strconv.Itoa(i)))[:32]
			item.Revision = 1
			item.DerivedFrom = mergeLibraryRefs(item.DerivedFrom, []string{source.ID})
			item.Replacements = nil
			changed[item.ID] = item
			replacements = append(replacements, item.ID)
		}
		source.Removed = true
		source.Replacements = replacements
		put(source)
	case "LA05":
		item := items[c.ID]
		remove := string(tool) == "factory.remove_library_item"
		if item.Removed == remove {
			return nil, errors.New("item already has the requested visibility")
		}
		item.Removed = remove
		put(item)
	case "LA06":
		item, target := items[c.ID], items[c.Target.ID]
		if err := active(item); err != nil {
			return nil, err
		}
		if err := active(target); err != nil {
			return nil, err
		}
		if item.ID == target.ID || item.Kind != target.Kind {
			return nil, errors.New("replacement must be a different item of the same kind")
		}
		if item.Kind == "decision" && (item.OwnerRole != target.OwnerRole || item.Scope != target.Scope || target.Decision == "" || target.State != "current") {
			return nil, errors.New("decision replacement needs a current recorded choice from its owner for the same scope")
		}
		item.State = "superseded"
		item.Replacements = []string{target.ID}
		item.EvidenceRefs = mergeLibraryRefs(item.EvidenceRefs, c.EvidenceRefs)
		put(item)
	case "LA07":
		item, target := items[c.ID], items[c.Target.ID]
		if err := active(item); err != nil {
			return nil, err
		}
		if err := active(target); err != nil {
			return nil, err
		}
		if item.ID == target.ID {
			return nil, errors.New("item cannot relate to itself")
		}
		relations := []factory.LibraryRelation{}
		for _, relation := range item.Relations {
			if relation.Kind != c.Relation || relation.Target != target.ID {
				relations = append(relations, relation)
			}
		}
		if !c.RemoveRelation {
			relations = append(relations, factory.LibraryRelation{Kind: c.Relation, Target: target.ID, Evidence: c.EvidenceRefs})
		}
		item.Relations = relations
		put(item)
	default:
		return nil, errors.New("unknown library operation")
	}
	result := []factory.LibraryItem{}
	for _, item := range changed {
		result = append(result, item)
	}
	slices.SortFunc(result, func(a, b factory.LibraryItem) int { return strings.Compare(a.ID, b.ID) })
	if !preview {
		for _, item := range result {
			if err := writeLibraryRevision(ctx, tx, item, c.Reason, "worker:"+cap.AssignmentID, submission); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func mergeLibraryRefs(a, b []string) []string {
	out := slices.Clone(a)
	for _, v := range b {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}
