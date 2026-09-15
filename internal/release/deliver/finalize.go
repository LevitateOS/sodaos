// Finalize connects protected final release signing after native
// qualification. It reuses Prepare/Sign/Publish and never rebuilds tested
// candidate or media bytes.
package deliver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
)

// Config is operator admission for final signing. It is never builder output.
// AuthFile+Ledger+Channel enable optional channel-last publication; omit all three
// for signed final metadata only.
type Config struct {
	Trust, Signer    string
	Serial           uint64
	Class, Notes     string
	AuthFile, Ledger string
	Channel          string // channel JSON published last after the release is signed
	ChannelSigner    string // restricted signer for the channel role when publishing
}

func admitPublicationInputs(c Config) error {
	set := 0
	for _, p := range []string{c.AuthFile, c.Ledger, c.Channel, c.ChannelSigner} {
		if p != "" {
			set++
		}
	}
	if set == 0 {
		return nil
	}
	if set != 4 {
		return errors.New("publication requires auth-file, ledger, channel and channel-signer together")
	}
	for _, p := range []string{c.AuthFile, c.Ledger, c.Channel, c.ChannelSigner} {
		if err := PrivateFile(p); err != nil {
			return err
		}
	}
	return nil
}

func admitConfig(c Config) error {
	if c.Serial == 0 || (c.Class != "normal" && c.Class != "emergency") || c.Notes == "" || len(c.Notes) > 16384 {
		return errors.New("exact release serial, class and notes required")
	}
	if _, err := os.Stat(c.Trust); err != nil {
		return errors.New("public release trust configuration required")
	}
	if err := PrivateFile(c.Signer); err != nil {
		return errors.New("restricted signer configuration required")
	}
	return admitPublicationInputs(c)
}

func LoadConfig(path string) (Config, error) {
	var c Config
	if os.Geteuid() != 0 {
		return c, errors.New("protected finalization admission required")
	}
	if err := PrivateFile(path); err != nil {
		return c, err
	}
	if err := ReadJSON(path, &c); err != nil {
		return c, err
	}
	return c, admitConfig(c)
}

func loadTrust(path string) (Trust, error) {
	var trust Trust
	if err := ReadJSON(path, &trust); err != nil || trust.Validate() != nil {
		return trust, errors.New("public release trust configuration refused")
	}
	return trust, nil
}

func loadSigner(path string) (SecretFiles, error) {
	var keys SecretFiles
	if err := ReadJSON(path, &keys); err != nil {
		return keys, errors.New("restricted signer configuration refused")
	}
	if err := PrivateFile(keys.Key); err != nil || PrivateFile(keys.Passphrase) != nil {
		return keys, errors.New("restricted signer key inputs required")
	}
	return keys, nil
}

func qualificationFromEvidence(c Config, evidencePath string) (Qualification, error) {
	sum, err := build.HashFile(evidencePath)
	if err != nil {
		return Qualification{}, err
	}
	return Qualification{
		Serial:   c.Serial,
		Class:    c.Class,
		Scope:    "native-install-upgrade-recovery",
		Notes:    c.Notes,
		Evidence: map[string]string{"qualification.json": "sha256:" + sum},
	}, nil
}

func documentDigest(oci string) (string, error) {
	b, err := os.ReadFile(filepath.Join(oci, "index.json"))
	if err != nil {
		return "", err
	}
	var index struct {
		Manifests []struct {
			Digest string `json:"digest"`
		} `json:"manifests"`
	}
	if json.Unmarshal(b, &index) != nil || len(index.Manifests) != 1 || !Digest(index.Manifests[0].Digest) {
		return "", ErrRefused
	}
	return index.Manifests[0].Digest, nil
}

func signDigest(ctx context.Context, r Runner, trust Trust, keys SecretFiles, repo, prepared, out string) (string, error) {
	digest, err := documentDigest(prepared)
	if err != nil {
		return "", err
	}
	permit := Permit{Format: 1, Repository: repo, Digest: digest, Expires: time.Now().Add(time.Hour).Unix()}
	if err := Sign(ctx, r, trust, permit, "oci", prepared, out, keys); err != nil {
		return "", err
	}
	return digest, nil
}

func bindChannelReleases(offer *Channel, releaseRef string) error {
	if len(offer.Releases) == 0 {
		return ErrRefused
	}
	for arch := range offer.Releases {
		offer.Releases[arch] = releaseRef
	}
	return nil
}

func channelRepository(trust Trust, name string) (string, error) {
	if name != "candidate" && name != "preview" && name != "stable" {
		return "", ErrRefused
	}
	return trust.Prefix + "-channel-" + name, nil
}

func publishChannelLast(ctx context.Context, r Runner, trust Trust, c Config, releaseRef, out string) error {
	var offer Channel
	if err := ReadJSON(c.Channel, &offer); err != nil {
		return err
	}
	if err := bindChannelReleases(&offer, releaseRef); err != nil {
		return err
	}
	prepared := filepath.Join(out, "channel-prepared")
	digest, err := WriteDocument(prepared, offer)
	if err != nil {
		return err
	}
	keys, err := loadSigner(c.ChannelSigner)
	if err != nil {
		return err
	}
	repo, err := channelRepository(trust, offer.Name)
	if err != nil {
		return err
	}
	signed := filepath.Join(out, "channel-signed")
	if _, err := signDigest(ctx, r, trust, keys, repo, prepared, signed); err != nil {
		return err
	}
	permit := Permit{Format: 1, Repository: repo, Digest: digest, Previous: "absent", Expires: time.Now().Add(time.Hour).Unix()}
	return Publish(ctx, r, trust, permit, filepath.Join(signed, "signed"), c.AuthFile, c.Ledger, filepath.Join(out, "publish"), false)
}

func writeFinalReceipt(out, ref, digest string, published bool) error {
	receipt := map[string]any{"reference": ref, "digest": digest, "scope": "signed final release metadata", "published": published}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return build.WriteNew(filepath.Join(out, "final.json"), append(data, '\n'), 0o600)
}

func prepareAndSign(ctx context.Context, r Runner, trust Trust, keys SecretFiles, q Qualification, candidate, media, out string) (string, string, error) {
	prepared := filepath.Join(out, "prepared")
	digest, err := Prepare(trust, candidate, media, q, prepared)
	if err != nil {
		return "", "", err
	}
	signed := filepath.Join(out, "signed")
	got, err := signDigest(ctx, r, trust, keys, trust.Prefix+"-release", prepared, signed)
	if err != nil {
		return "", "", err
	}
	if got != digest {
		return "", "", ErrRefused
	}
	ref, err := ReferenceForDocument(trust, "release", digest)
	return ref, digest, err
}

// Finalize prepares and signs final release metadata from unchanged candidate and
// media bindings. Optional channel-last publication runs only when admitted.
func Finalize(ctx context.Context, r Runner, c Config, candidate, media, evidence, out string) (string, error) {
	if err := build.FreshDirectory(out); err != nil {
		return "", err
	}
	trust, err := loadTrust(c.Trust)
	if err != nil {
		return "", err
	}
	keys, err := loadSigner(c.Signer)
	if err != nil {
		return "", err
	}
	q, err := qualificationFromEvidence(c, evidence)
	if err != nil {
		return "", err
	}
	ref, digest, err := prepareAndSign(ctx, r, trust, keys, q, candidate, media, out)
	if err != nil {
		return "", err
	}
	published := c.AuthFile != ""
	if published {
		if err := publishChannelLast(ctx, r, trust, c, ref, out); err != nil {
			return "", err
		}
	}
	return ref, writeFinalReceipt(out, ref, digest, published)
}
