package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
)

// Complete-inventory bounds (D13): every open non-PR native issue of
// the repository, ordered by issue index, with the exact delivered
// context needed to interpret them. Missing permissions or pages,
// excess size and unstable evidence visibly hold the gate; nothing is
// truncated and an empty page is never confused with a complete empty
// inventory.
const (
	maxCompletionInventoryPages = 256
	maxCollectorRestarts        = 3
)

// Inventory hold reasons name the resumable step for the cycle wait
// reason. They carry no evidence bytes.
const (
	HoldInventoryPage        = "inventory_page_invalid"
	HoldInventoryRevision    = "inventory_revision_changed"
	HoldInventoryOversize    = "inventory_evidence_oversize"
	HoldInventoryUnavailable = "inventory_unavailable"
	HoldInventoryInProgress  = "inventory_collecting"
)

// NativeInventoryIssue is one enumerated open non-PR native issue with
// its canonical evidence bytes. The producer (the FW32 native-page
// joint over the FW29 repository_issues family plus existing individual
// families, under one stable revision) orders items by ascending issue
// index and bounds each Data; the collector bounds the total.
type NativeInventoryIssue struct {
	Data  string
	Index int64
}

// NativeInventoryPage is one bounded enumeration page. A terminal page
// carries complete=true with an empty cursor; a nonterminal page
// carries nonempty items with the cursor equal to the final returned
// index, strictly greater than the input. An empty incomplete page is
// invalid, as is a nonempty cursor that is not a canonical positive
// int64 decimal index.
type NativeInventoryPage struct {
	Issues     []NativeInventoryIssue
	NextCursor string
	Revision   int64
	Complete   bool
	Idle       bool
}

// CompletionIssueEnumerator supplies native pages plus the idle
// revision bracket. Control owns this interface because the native
// snapshot adapter lives outside the control import boundary; the FW32
// handoff binds it to the FW29 repository_issues family. Tests supply
// deterministic fixtures; production never consults a live model here.
type CompletionIssueEnumerator interface {
	ObserveRevision(ctx context.Context) (int64, bool, error)
	EnumerateIssues(ctx context.Context, repository int64, cursor string) (NativeInventoryPage, error)
}

// SealedInventory is one durable complete inventory: every open
// non-PR native issue of the repository at one idle native revision,
// ordered by issue index, with the bound evidence and the local
// bindings that interpret it. An empty issue set seals as a valid
// complete empty inventory with its own digest: each completion gets
// its own decision even when none remain. Bodies carries the parsed
// substance behind every collected Data payload, index-aligned with
// Indexes, so the seal persists bodies — not just the digest — for the
// funded decision.
type SealedInventory struct {
	Local      factory.CompletionLocal
	Indexes    []int64
	Bodies     []factory.CompletionIssueEvidence
	Digest     string
	Repository int64
	Generation int64
	Revision   int64
	Issues     int
	Evidence   int
}

// InventoryCollector resumes one bounded collection across passes. The
// cursor restarts from empty whenever the native revision moves so a
// concurrent insertion can never be omitted; stable revisions make
// progress instead. Progress serializes into the cycle row so restarts
// and crashes resume instead of rescanning.
type InventoryCollector struct {
	Running  [32]byte
	Issues   []NativeInventoryIssue
	Local    factory.CompletionLocal
	Cursor   string
	Revision int64
	Restarts int
}

// parseInventoryCursor validates a nonempty cursor as a canonical
// positive int64 decimal index.
func parseInventoryCursor(cursor string) (int64, error) {
	index, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || index < 1 || strconv.FormatInt(index, 10) != cursor {
		return 0, errors.New("invalid inventory cursor")
	}
	return index, nil
}

// checkInventoryPage validates one page against the cursor that
// fetched it and the collector bounds.
func checkInventoryPage(page NativeInventoryPage, cursor string, collected int, evidence int) error {
	if page.Revision < 0 || !page.Idle {
		return errors.New(HoldInventoryRevision)
	}
	if page.Complete {
		if page.NextCursor != "" {
			return errors.New(HoldInventoryPage)
		}
	} else {
		if len(page.Issues) == 0 {
			return errors.New(HoldInventoryPage)
		}
		last, err := parseInventoryCursor(page.NextCursor)
		if err != nil {
			return errors.New(HoldInventoryPage)
		}
		if cursor != "" {
			current, err := parseInventoryCursor(cursor)
			if err != nil || last <= current {
				return errors.New(HoldInventoryPage)
			}
		}
		// The cursor must also match the final returned index: pages
		// chain exactly, so a resumed scan can never skip an issue.
		if last != page.Issues[len(page.Issues)-1].Index {
			return errors.New(HoldInventoryPage)
		}
	}
	var base int64
	if cursor != "" {
		var err error
		base, err = parseInventoryCursor(cursor)
		if err != nil {
			return errors.New(HoldInventoryPage)
		}
	}
	prior := base
	for _, issue := range page.Issues {
		if issue.Index <= prior {
			return errors.New(HoldInventoryPage)
		}
		prior = issue.Index
		evidence += len(issue.Data)
		if evidence > factory.MaxReassessmentEvidenceBytes {
			return errors.New(HoldInventoryOversize)
		}
	}
	if collected+len(page.Issues) > factory.MaxReassessmentIssues {
		return errors.New(HoldInventoryOversize)
	}
	return nil
}

// foldInventoryPage appends one validated page to the running digest.
func foldInventoryPage(running [32]byte, repository, generation int64, issues []NativeInventoryIssue) [32]byte {
	h := sha256.New()
	h.Write(running[:])
	for _, issue := range issues {
		h.Write([]byte(strconv.FormatInt(issue.Index, 10)))
		h.Write([]byte{0})
		h.Write([]byte(issue.Data))
		h.Write([]byte{0})
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// sealInventoryDigest binds the sealed set: repository, generation,
// bracket revision, ordered indexes and the folded evidence.
func sealInventoryDigest(repository, generation, revision int64, running [32]byte, indexes []int64) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d\x00%d\x00%d\x00", repository, generation, revision)))
	h.Write(running[:])
	for _, index := range indexes {
		h.Write([]byte(strconv.FormatInt(index, 10) + "\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Collect advances one bounded collection: it brackets an idle native
// revision, resumes from the saved cursor, validates every page shape,
// and either seals the complete inventory or reports the hold reason
// with updated resumable progress. A revision move restarts from empty
// (bounded restarts per call); anything else holds without consuming
// the restart budget. Empty complete inventories seal valid.
func (c *InventoryCollector) Collect(ctx context.Context, enum CompletionIssueEnumerator, repository, generation int64, local factory.CompletionLocal) (*SealedInventory, *InventoryCollector, error) {
	if err := local.Validate(); err != nil {
		return nil, c, errors.New(HoldInventoryUnavailable)
	}
	if enum == nil {
		return nil, c, errors.New(HoldInventoryUnavailable)
	}
	revision, idle, err := enum.ObserveRevision(ctx)
	if err != nil || !idle {
		return nil, c, errors.New(HoldInventoryRevision)
	}
	if c.Revision != 0 && c.Revision != revision {
		if c.Restarts >= maxCollectorRestarts {
			return nil, c, errors.New(HoldInventoryRevision)
		}
		c = &InventoryCollector{Revision: revision, Restarts: c.Restarts + 1}
	}
	if c.Revision == 0 {
		c = &InventoryCollector{Revision: revision, Local: local, Restarts: c.Restarts}
	}
	next := *c
	next.Local = local
	evidence := 0
	for _, issue := range next.Issues {
		evidence += len(issue.Data)
	}
	for pages := 0; pages < maxCompletionInventoryPages; pages++ {
		if err := ctx.Err(); err != nil {
			return nil, &next, err
		}
		page, err := enum.EnumerateIssues(ctx, repository, next.Cursor)
		if err != nil {
			// The producer names exact holds visibly: oversize,
			// malformed and unstable evidence keep their reason
			// instead of collapsing into unavailable.
			switch err.Error() {
			case HoldInventoryOversize, HoldInventoryPage, HoldInventoryRevision:
				return nil, &next, err
			default:
				return nil, &next, errors.New(HoldInventoryUnavailable)
			}
		}
		if page.Revision != revision || !page.Idle {
			if next.Restarts >= maxCollectorRestarts {
				return nil, &next, errors.New(HoldInventoryRevision)
			}
			// Re-bracket before restarting: the saved cursor belongs
			// to the moved revision and resuming it could omit the
			// concurrent insertion.
			fresh, idle, err := enum.ObserveRevision(ctx)
			if err != nil || !idle {
				return nil, &next, errors.New(HoldInventoryRevision)
			}
			revision = fresh
			next = InventoryCollector{Revision: revision, Local: local, Restarts: next.Restarts + 1}
			evidence = 0
			continue
		}
		if err := checkInventoryPage(page, next.Cursor, len(next.Issues), evidence); err != nil {
			return nil, &next, err
		}
		next.Running = foldInventoryPage(next.Running, repository, generation, page.Issues)
		next.Issues = append(next.Issues, page.Issues...)
		for _, issue := range page.Issues {
			evidence += len(issue.Data)
		}
		if page.Complete {
			confirmed, idle, err := enum.ObserveRevision(ctx)
			if err != nil || !idle || confirmed != revision {
				if next.Restarts >= maxCollectorRestarts {
					return nil, &next, errors.New(HoldInventoryRevision)
				}
				next = InventoryCollector{Revision: confirmed, Local: local, Restarts: next.Restarts + 1}
				evidence = 0
				continue
			}
			indexes := make([]int64, 0, len(next.Issues))
			for _, issue := range next.Issues {
				indexes = append(indexes, issue.Index)
			}
			sort.Slice(indexes, func(i, j int) bool { return indexes[i] < indexes[j] })
			// The seal parses every payload into its substance: a
			// non-canonical body refuses the seal instead of
			// persisting bytes the decision cannot assess.
			byIndex := make(map[int64]factory.CompletionIssueEvidence, len(next.Issues))
			for _, issue := range next.Issues {
				body, err := factory.ParseCompletionIssueEvidence(issue.Data)
				if err != nil {
					return nil, &next, errors.New(HoldInventoryPage)
				}
				if body.Index != issue.Index {
					return nil, &next, errors.New(HoldInventoryPage)
				}
				byIndex[issue.Index] = body
			}
			bodies := make([]factory.CompletionIssueEvidence, 0, len(indexes))
			for _, index := range indexes {
				bodies = append(bodies, byIndex[index])
			}
			// The seal supersedes progress: resuming from a sealed
			// collector would re-observe a closed evidence set.
			return &SealedInventory{
				Local: next.Local, Indexes: indexes, Bodies: bodies,
				Digest:     sealInventoryDigest(repository, generation, revision, next.Running, indexes),
				Repository: repository, Generation: generation, Revision: revision,
				Issues: len(next.Issues), Evidence: evidence,
			}, nil, nil
		}
		next.Cursor = page.NextCursor
	}
	return nil, &next, errors.New(HoldInventoryInProgress)
}
