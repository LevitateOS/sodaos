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
func (l Ledger) Validate(t Trust) error {
	if l.Format != 1 || l.State.Validate() != nil || l.State.TrustEpoch > t.Epoch {
		return ErrRefused
	}
	if _, e := t.Role(l.Repository); e != nil {
		return e
	}
	if l.Phase != "idle" && l.Phase != "pending" && l.Phase != "complete" {
		return ErrRefused
	}
	if l.Phase == "idle" {
		if l.Digest != "" {
			return ErrRefused
		}
	} else if !Digest(l.Digest) {
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
func Publish(ctx context.Context, r Runner, t Trust, p Permit, signed, auth, ledgerPath, out string, observe bool) error {
	if t.Validate() != nil {
		return ErrRefused
	}
	role, e := t.Role(p.Repository)
	if e != nil || p.Format != 1 || !Digest(p.Digest) {
		return ErrRefused
	}
	lock, e := lockState(ledgerPath)
	if e != nil {
		return e
	}
	defer unlock(lock)
	var ledger Ledger
	if e = ReadJSON(ledgerPath, &ledger); e != nil {
		return e
	}
	if ledger.Validate(t) != nil || ledger.Repository != p.Repository {
		return ErrRefused
	}
	if e = nativebuild.FreshDirectory(out); e != nil {
		return e
	}
	if observe || (ledger.Phase == "complete" && ledger.Digest == p.Digest) {
		if ledger.Digest != p.Digest || ledger.Phase == "idle" {
			return ErrRefused
		}
		if e = observePublished(ctx, r, t, p, role, out); e != nil {
			return e
		}
		ledger.Phase = "complete"
		ledger.State.TrustEpoch = t.Epoch
		if e = saveState(ledgerPath, ledger); e != nil {
			return e
		}
		return writeJSON(filepath.Join(out, "receipt.json"), map[string]string{"Reference": p.Repository + "@" + p.Digest, "Outcome": "publication observed; no write replayed"})
	}
	if ledger.Phase == "pending" {
		return errors.New("uncertain publication held; use observation, never blind replay")
	}
	if p.Validate(t, nowUTC()) != nil || PrivateFile(auth) != nil {
		return ErrRefused
	}
	ref := p.Repository + "@" + p.Digest
	check := filepath.Join(out, "signed-check")
	if e = VerifyCopy(ctx, r, t, ref, "dir:"+signed, check); e != nil {
		return e
	}
	copy := filepath.Join(check, "image")
	var offer Channel
	next := ledger.State
	if channel(role) {
		if p.Previous != "absent" && !Digest(p.Previous) {
			return ErrRefused
		}
		if ledger.State.Channels[role].Sequence == 0 && p.Previous != "absent" {
			return errors.New("existing channel requires preserved publisher history; no implicit adoption")
		}
		if ledger.State.Channels[role].Sequence != 0 && p.Previous != ledger.State.Channels[role].Digest {
			return errors.New("protected permit disagrees with publisher history")
		}
		if e = ReadDocument(copy, p.Digest, &offer); e != nil {
			return e
		}
		next, e = AdmitChannel(t, next, offer, p.Digest, role, nowUTC())
		if e != nil {
			return e
		}
		// Every advertised architecture and every signed artifact must be publicly
		// retrievable before publishing even the immutable channel document.
		next, e = verifyReleases(ctx, r, t, next, offer, "", out)
		if e != nil {
			return e
		}
	}
	next.TrustEpoch = t.Epoch
	ledger.Phase = "pending"
	ledger.Digest = p.Digest
	ledger.State = next
	if e = saveState(ledgerPath, ledger); e != nil {
		return e
	}
	immutable := filepath.Join(out, "immutable-upload")
	if e = os.Mkdir(immutable, 0700); e != nil {
		return e
	}
	if e = upload(ctx, r, t, ref, copy, immutableTag(ref), auth, immutable); e != nil {
		return e
	}
	if e = VerifyCopy(ctx, r, t, ref, "docker://"+ref, filepath.Join(out, "registry-roundtrip")); e != nil {
		return e
	}
	if channel(role) {
		// The immutable upload has created the repository. Native list-tags now
		// distinguishes an absent channel tag from auth/network/transport failure;
		// no brittle parsing of skopeo's stderr and no implicit bootstrap on error.
		tags, e := r.Run(ctx, "--command-timeout=2m", "list-tags", "--no-creds", "docker://"+p.Repository)
		if e != nil {
			return e
		}
		var list struct {
			Repository string
			Tags       []string
		}
		if e = decode(tags, &list); e != nil || list.Repository != p.Repository || len(list.Tags) == 0 {
			return ErrRefused
		}
		exists := false
		for _, tag := range list.Tags {
			if tag == role {
				exists = true
			}
		}
		if exists {
			previous, e := discover(ctx, r, t, role)
			if e != nil {
				return e
			}
			_, d, _ := strings.Cut(previous, "@")
			if d != p.Previous {
				return errors.New("channel changed since protected admission")
			}
			var old Channel
			if e = fetchDocument(ctx, r, t, previous, filepath.Join(out, "previous-channel"), &old); e != nil {
				return e
			}
			// Previous may be expired; it is authenticated history, not a current offer.
			if old.Format != 1 || old.Name != role || old.Sequence >= offer.Sequence || old.Issued > offer.Issued {
				return ErrRefused
			}
		} else if p.Previous != "absent" {
			return errors.New("expected channel missing; no automatic bootstrap")
		}
		if p.Validate(t, nowUTC()) != nil {
			return ErrRefused
		}
		if _, e = AdmitChannel(t, ledger.State, offer, p.Digest, role, nowUTC()); e != nil {
			return e
		}
		promotion := filepath.Join(out, "promotion")
		if e = os.Mkdir(promotion, 0700); e != nil {
			return e
		}
		// The signature attachment and immutable document already round-tripped.
		// Updating this tag last cannot point at an incompletely uploaded offer.
		if e = upload(ctx, r, t, ref, copy, p.Repository+":"+role, auth, promotion); e != nil {
			return e
		}
	}
	if e = observePublished(ctx, r, t, p, role, out); e != nil {
		return e
	}
	ledger.Phase = "complete"
	if e = saveState(ledgerPath, ledger); e != nil {
		return e
	}
	return writeJSON(filepath.Join(out, "receipt.json"), map[string]string{"Reference": ref, "Outcome": "published and anonymously signature-verified; no activation"})
}
