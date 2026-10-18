package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// repairEpisodeChargeID derives the deterministic cycle-1
// investigator-run charge identity for one bound stall episode.
// Redetecting the same episode replays the identical charge, so the
// store spends once and replays return the recorded receipt.
func repairEpisodeChargeID(episodeID string) string {
	sum := sha256.Sum256([]byte("repair-charge-v1\x00" + episodeID + "\x00" + factory.RepairChargeRun + "\x001"))
	return hex.EncodeToString(sum[:16])
}

// AdmitStallEpisodeRun admits one cycle-1 investigator-run charge for a
// classified stall repository result under the repository's explicit
// repair grant. The caller passes the detection round's progress
// generation: the bound episode must match it exactly, so a stale
// episode from an older round redetects instead of spending and a
// generation mismatch is impossible by construction. Non-stall
// results admit nothing and report unspent; a stall result with an
// invalid bound event fails closed instead of spending against an
// unnamed episode. Later cycles need materially new evidence through
// the Strategist proposal path, never this joint.
func AdmitStallEpisodeRun(ctx context.Context, db *store.Store, result StallRepositoryResult, generation int64, activeMinutes int, now time.Time) (bool, error) {
	if result.Kind != StallRepositoryStall || !result.Investigate || result.Event == nil {
		return false, nil
	}
	event := result.Event
	if err := event.Validate(); err != nil {
		return false, errors.New("stall result carries an invalid bound episode")
	}
	if event.ProgressGeneration != generation {
		return false, errors.New("stall episode no longer matches its detection round: redetect before spending")
	}
	grant, err := db.RepairGrantCommand(ctx, result.Repository)
	if err != nil {
		return false, err
	}
	charge := factory.RepairCharge{
		ID: repairEpisodeChargeID(event.EpisodeID), GrantID: grant.ID,
		Kind: factory.RepairChargeRun, EpisodeID: event.EpisodeID, Cycle: 1,
		EvidenceDigest: event.InventoryDigest, ActiveMinutes: activeMinutes,
		SpentUnix: now.Unix(),
	}
	_, spent, err := db.AdmitStallInvestigation(ctx, result.Repository, charge, now)
	if err != nil {
		return false, err
	}
	return spent, nil
}
