package releasedelivery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

type Ledger struct {
	Format     int
	Repository string
	Phase      string // idle, pending, complete. Pending never automatically replays a write.
	Digest     string
	State      Highwater
}

func InitLedger(path string, t Trust, repo string) error {
	if t.Validate() != nil {
		return ErrRefused
	}
	if _, e := t.Role(repo); e != nil {
		return e
	}
	if e := nativebuild.PrivateDestination(path); e != nil {
		return e
	}
	state := EmptyState()
	state.TrustEpoch = t.Epoch
	return writeJSON(path, Ledger{Format: 1, Repository: repo, Phase: "idle", State: state})
}

func validLedgerPhase(phase, digest string) bool {
	switch phase {
	case "idle":
		return digest == ""
	case "pending", "complete":
		return Digest(digest)
	default:
		return false
	}
}

func (l Ledger) Validate(t Trust) error {
	if l.Format != 1 || l.State.Validate() != nil || l.State.TrustEpoch > t.Epoch {
		return ErrRefused
	}
	if _, e := t.Role(l.Repository); e != nil {
		return e
	}
	if !validLedgerPhase(l.Phase, l.Digest) {
		return ErrRefused
	}
	return nil
}

func upload(ctx context.Context, r Runner, t Trust, ref, signed, dest, auth, out string) error {
	repo, _, e := t.Reference(ref)
	if e != nil {
		return e
	}
	policy, e := policyFor(t, repo, "dir", signed)
	if e != nil {
		return e
	}
	pp := filepath.Join(out, "upload-policy.json")
	if e = writeJSON(pp, policy); e != nil {
		return e
	}
	reg, e := registryConfig(out, t)
	if e != nil {
		return e
	}
	// Credentials go only to the destination. Uploads have no generic retry flags.
	_, e = r.Run(ctx, "--command-timeout=10m", "--policy", pp, "--registries.d", reg, "copy", "--preserve-digests", "--src-no-creds", "--dest-authfile", auth, "dir:"+signed, "docker://"+dest)
	return e
}

func observePublished(ctx context.Context, r Runner, t Trust, p Permit, role, out string) error {
	ref := p.Repository + "@" + p.Digest
	if channel(role) {
		actual, e := discover(ctx, r, t, role)
		if e != nil {
			return e
		}
		if actual != ref {
			return errors.New("publication not observed at intended channel; held, not replayed")
		}
	}
	return VerifyCopy(ctx, r, t, ref, "docker://"+ref, filepath.Join(out, "observed"))
}

// Publish is used unattended by a protected worker. The caller must serialize ALL
// writers to each repository through its one persistent ledger. GHCR has no CAS
// tag update here: a second external tag writer is not a supported topology.
// On uncertain results, --observe can complete the receipt without replaying any
// upload. If the desired commit cannot be observed, the job remains held.
func admitPublishLedger(t Trust, p Permit, ledgerPath, out string) (string, Ledger, error) {
	if t.Validate() != nil {
		return "", Ledger{}, ErrRefused
	}
	role, err := t.Role(p.Repository)
	if err != nil || p.Format != 1 || !Digest(p.Digest) {
		return "", Ledger{}, ErrRefused
	}
	var ledger Ledger
	if err = ReadJSON(ledgerPath, &ledger); err != nil {
		return "", Ledger{}, err
	}
	if ledger.Validate(t) != nil || ledger.Repository != p.Repository {
		return "", Ledger{}, ErrRefused
	}
	if err = nativebuild.FreshDirectory(out); err != nil {
		return "", Ledger{}, err
	}
	return role, ledger, nil
}

func shouldObserve(observe bool, ledger Ledger, p Permit) bool {
	return observe || (ledger.Phase == "complete" && ledger.Digest == p.Digest)
}

func observeOnly(ctx context.Context, r Runner, t Trust, p Permit, role, ledgerPath, out string, ledger Ledger) error {
	if ledger.Digest != p.Digest || ledger.Phase == "idle" {
		return ErrRefused
	}
	if err := observePublished(ctx, r, t, p, role, out); err != nil {
		return err
	}
	ledger.Phase = "complete"
	ledger.State.TrustEpoch = t.Epoch
	if err := saveState(ledgerPath, ledger); err != nil {
		return err
	}
	return writeJSON(filepath.Join(out, "receipt.json"), map[string]string{
		"Reference": p.Repository + "@" + p.Digest,
		"Outcome":   "publication observed; no write replayed",
	})
}

func validateChannelHistory(current Seen, previous string) error {
	if previous != "absent" && !Digest(previous) {
		return ErrRefused
	}
	if current.Sequence == 0 && previous != "absent" {
		return errors.New("existing channel requires preserved publisher history; no implicit adoption")
	}
	if current.Sequence != 0 && previous != current.Digest {
		return errors.New("protected permit disagrees with publisher history")
	}
	return nil
}

func admitChannelOffer(ctx context.Context, r Runner, t Trust, p Permit, role, copy, out string, current Highwater) (Highwater, Channel, error) {
	if err := validateChannelHistory(current.Channels[role], p.Previous); err != nil {
		return Highwater{}, Channel{}, err
	}
	var offer Channel
	if err := ReadDocument(copy, p.Digest, &offer); err != nil {
		return Highwater{}, Channel{}, err
	}
	next, err := AdmitChannel(t, current, offer, p.Digest, role, nowUTC())
	if err != nil {
		return Highwater{}, Channel{}, err
	}
	// Every advertised architecture and every signed artifact must be publicly
	// retrievable before publishing even the immutable channel document.
	next, err = verifyReleases(ctx, r, t, next, offer, "", out)
	if err != nil {
		return Highwater{}, Channel{}, err
	}
	return next, offer, nil
}

func admitSigned(ctx context.Context, r Runner, t Trust, p Permit, role, signed, auth, ledgerPath, out string, ledger Ledger) (string, Channel, Ledger, error) {
	if p.Validate(t, nowUTC()) != nil || PrivateFile(auth) != nil {
		return "", Channel{}, Ledger{}, ErrRefused
	}
	ref := p.Repository + "@" + p.Digest
	check := filepath.Join(out, "signed-check")
	if err := VerifyCopy(ctx, r, t, ref, "dir:"+signed, check); err != nil {
		return "", Channel{}, Ledger{}, err
	}
	copy := filepath.Join(check, "image")
	var offer Channel
	next := ledger.State
	if channel(role) {
		var err error
		next, offer, err = admitChannelOffer(ctx, r, t, p, role, copy, out, next)
		if err != nil {
			return "", Channel{}, Ledger{}, err
		}
	}
	next.TrustEpoch = t.Epoch
	ledger.Phase = "pending"
	ledger.Digest = p.Digest
	ledger.State = next
	if err := saveState(ledgerPath, ledger); err != nil {
		return "", Channel{}, Ledger{}, err
	}
	return copy, offer, ledger, nil
}

func commitImmutable(ctx context.Context, r Runner, t Trust, ref, copy, auth, out string) error {
	immutable := filepath.Join(out, "immutable-upload")
	if err := os.Mkdir(immutable, 0o700); err != nil {
		return err
	}
	if err := upload(ctx, r, t, ref, copy, immutableTag(ref), auth, immutable); err != nil {
		return err
	}
	return VerifyCopy(ctx, r, t, ref, "docker://"+ref, filepath.Join(out, "registry-roundtrip"))
}

func channelTagExists(ctx context.Context, r Runner, repository, role string) (bool, error) {
	// The immutable upload has created the repository. Native list-tags now
	// distinguishes an absent channel tag from auth/network/transport failure;
	// no brittle parsing of skopeo's stderr and no implicit bootstrap on error.
	tags, err := r.Run(ctx, "--command-timeout=2m", "list-tags", "--no-creds", "docker://"+repository)
	if err != nil {
		return false, err
	}
	var list struct {
		Repository string
		Tags       []string
	}
	if err = decode(tags, &list); err != nil || list.Repository != repository || len(list.Tags) == 0 {
		return false, ErrRefused
	}
	for _, tag := range list.Tags {
		if tag == role {
			return true, nil
		}
	}
	return false, nil
}

func verifyPreviousChannel(ctx context.Context, r Runner, t Trust, p Permit, role, out string, offer Channel) error {
	previous, err := discover(ctx, r, t, role)
	if err != nil {
		return err
	}
	_, d, _ := strings.Cut(previous, "@")
	if d != p.Previous {
		return errors.New("channel changed since protected admission")
	}
	var old Channel
	if err = fetchDocument(ctx, r, t, previous, filepath.Join(out, "previous-channel"), &old); err != nil {
		return err
	}
	// Previous may be expired; it is authenticated history, not a current offer.
	if old.Format != 1 || old.Name != role || old.Sequence >= offer.Sequence || old.Issued > offer.Issued {
		return ErrRefused
	}
	return nil
}

func promoteChannel(ctx context.Context, r Runner, t Trust, p Permit, role, ref, copy, auth, out string, offer Channel, ledger Ledger) error {
	exists, err := channelTagExists(ctx, r, p.Repository, role)
	if err != nil {
		return err
	}
	if exists {
		if err = verifyPreviousChannel(ctx, r, t, p, role, out, offer); err != nil {
			return err
		}
	} else if p.Previous != "absent" {
		return errors.New("expected channel missing; no automatic bootstrap")
	}
	if p.Validate(t, nowUTC()) != nil {
		return ErrRefused
	}
	if _, err = AdmitChannel(t, ledger.State, offer, p.Digest, role, nowUTC()); err != nil {
		return err
	}
	promotion := filepath.Join(out, "promotion")
	if err = os.Mkdir(promotion, 0o700); err != nil {
		return err
	}
	// The signature attachment and immutable document already round-tripped.
	// Updating this tag last cannot point at an incompletely uploaded offer.
	return upload(ctx, r, t, ref, copy, p.Repository+":"+role, auth, promotion)
}

func finalizePublication(ctx context.Context, r Runner, t Trust, p Permit, role, ref, ledgerPath, out string, ledger Ledger) error {
	if err := observePublished(ctx, r, t, p, role, out); err != nil {
		return err
	}
	ledger.Phase = "complete"
	if err := saveState(ledgerPath, ledger); err != nil {
		return err
	}
	return writeJSON(filepath.Join(out, "receipt.json"), map[string]string{
		"Reference": ref,
		"Outcome":   "published and anonymously signature-verified; no activation",
	})
}

// Publish is used unattended by a protected worker. The caller must serialize ALL
// writers to each repository through its one persistent ledger. GHCR has no CAS
// tag update here: a second external tag writer is not a supported topology.
// On uncertain results, --observe can complete the receipt without replaying any
// upload. If the desired commit cannot be observed, the job remains held.
func Publish(ctx context.Context, r Runner, t Trust, p Permit, signed, auth, ledgerPath, out string, observe bool) error {
	lock, err := lockState(ledgerPath)
	if err != nil {
		return err
	}
	defer unlock(lock)
	role, ledger, err := admitPublishLedger(t, p, ledgerPath, out)
	if err != nil {
		return err
	}
	if shouldObserve(observe, ledger, p) {
		return observeOnly(ctx, r, t, p, role, ledgerPath, out, ledger)
	}
	if ledger.Phase == "pending" {
		return errors.New("uncertain publication held; use observation, never blind replay")
	}
	ref := p.Repository + "@" + p.Digest
	copy, offer, ledger, err := admitSigned(ctx, r, t, p, role, signed, auth, ledgerPath, out, ledger)
	if err != nil {
		return err
	}
	if err = commitImmutable(ctx, r, t, ref, copy, auth, out); err != nil {
		return err
	}
	if channel(role) {
		if err = promoteChannel(ctx, r, t, p, role, ref, copy, auth, out, offer, ledger); err != nil {
			return err
		}
	}
	return finalizePublication(ctx, r, t, p, role, ref, ledgerPath, out, ledger)
}
