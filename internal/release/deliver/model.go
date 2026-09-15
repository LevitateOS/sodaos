// Package deliver binds native Sigstore verification to Soda release and
// channel semantics. It never installs an image, changes host trust or reboots.
package deliver

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/strictjson"
)

var (
	ErrUnavailable = errors.New("release transport unavailable; preserve attempt and observe before retrying publication")
	ErrRefused     = errors.New("release authority or completeness refused")
)

func Hash(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }
func Digest(s string) bool {
	return strings.HasPrefix(s, "sha256:") && build.Digest(strings.TrimPrefix(s, "sha256:"))
}
func channel(s string) bool        { return s == "candidate" || s == "preview" || s == "stable" }
func decode(b []byte, v any) error { return strictjson.Decode(bytes.NewReader(b), v) }

// Keys are public PEM, not paths to mutable builder-selected trust. Separate
// channel roles prevent an artifact/preview signing key from approving stable.
type Trust struct {
	Format           int
	Prefix           string
	Epoch            uint64
	Keys             map[string][]string // artifact, candidate, preview, stable; rotation overlap within a role
	NotBefore        int64
	MaxAgeSeconds    int64
	ClockSkewSeconds int64
	MinimumSequence  map[string]uint64
}

func validTrustTiming(t Trust) bool {
	return t.NotBefore > 0 && t.MaxAgeSeconds >= 60 && t.MaxAgeSeconds <= 7*86400 && t.ClockSkewSeconds >= 0 && t.ClockSkewSeconds <= 300
}

func validTrustEnvelope(t Trust) bool {
	return t.Format == 1 && ValidRepositoryPrefix(t.Prefix) && t.Epoch != 0 && len(t.Keys) == 4 && len(t.MinimumSequence) == 3 && validTrustTiming(t)
}

func parseTrustPublicKey(key string) ([]byte, error) {
	block, rest := pem.Decode([]byte(key))
	if block == nil || block.Type != "PUBLIC KEY" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, ErrRefused
	}
	pub, e := x509.ParsePKIXPublicKey(block.Bytes)
	if e != nil {
		return nil, ErrRefused
	}
	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok || ec.Curve != elliptic.P256() {
		return nil, errors.New("native P-256 Sigstore public key required")
	}
	return block.Bytes, nil
}

func admitTrustRoleKeys(keys []string, seen map[string]bool) error {
	if len(keys) < 1 || len(keys) > 4 {
		return ErrRefused
	}
	for _, key := range keys {
		der, err := parseTrustPublicKey(key)
		if err != nil {
			return err
		}
		fingerprint := Hash(der)
		if seen[fingerprint] {
			return errors.New("signer roles must not share keys")
		}
		seen[fingerprint] = true
	}
	return nil
}

func (t Trust) Validate() error {
	if !validTrustEnvelope(t) {
		return ErrRefused
	}
	seen := map[string]bool{}
	for _, role := range []string{"artifact", "candidate", "preview", "stable"} {
		if err := admitTrustRoleKeys(t.Keys[role], seen); err != nil {
			return err
		}
		if role != "artifact" && t.MinimumSequence[role] == 0 {
			return ErrRefused
		}
	}
	return nil
}

func (t Trust) Role(repository string) (string, error) {
	for _, n := range append([]string{"host", "release", "media"}, Names...) {
		if repository == t.Prefix+"-"+n {
			return "artifact", nil
		}
	}
	for _, c := range []string{"candidate", "preview", "stable"} {
		if repository == t.Prefix+"-channel-"+c {
			return c, nil
		}
	}
	return "", ErrRefused
}

func (t Trust) Reference(ref string) (string, string, error) {
	repo, d, ok := strings.Cut(ref, "@")
	if !ok || !Digest(d) {
		return "", "", ErrRefused
	}
	role, e := t.Role(repo)
	return repo, role, e
}

type Candidate struct {
	Format                                                            int
	Host                                                              build.Image
	HostReference, HostArchiveSHA256, PayloadSHA256, Migration, Notes string
}

func validCandidateHost(c Candidate, p Payload, arch string) bool {
	return c.HostReference == p.RepositoryPrefix+"-host@"+c.Host.Manifest && Digest(c.Host.Manifest) && Digest(c.Host.Config) && build.Digest(c.HostArchiveSHA256) && c.Host.Architecture == arch && c.Host.Revision == p.Revision && c.Host.BaseName == p.Base
}

func validCandidateProvenance(c Candidate, p Payload, payload []byte) bool {
	return c.Format == 1 && c.PayloadSHA256 == strings.TrimPrefix(Hash(payload), "sha256:") && c.Host.BaseDigest == strings.Split(p.Base, "@")[1] && c.Host.Source == "https://github.com/LevitateOS/sodaos" && c.Migration != "" && c.Notes != ""
}

func (c Candidate) Validate(p Payload, payload []byte) error {
	arch, e := build.OCIArchitecture(p.Architecture)
	if e != nil || p.Validate() != nil {
		return ErrRefused
	}
	if !validCandidateHost(c, p, arch) || !validCandidateProvenance(c, p, payload) {
		return ErrRefused
	}
	return nil
}

// MediaFile is the exact ISO/rootfs binding already sealed in media.json.
type MediaFile struct {
	Path, SHA256 string
	Bytes        int64
}

// MediaBinding is decoded from embedded media.json bytes. Extra fields are
// ignored; required ISO/rootfs/location bindings are checked against the candidate.
type MediaBinding struct {
	Revision, Architecture, HostManifest, PayloadSHA256, RootfsURL string
	ISO, Rootfs                                                    MediaFile
}

// Exact existing metadata bytes are embedded, not independently re-maintained
// component inventories. Native qualification is evidence, not self-authorizing:
// the protected signer must admit the exact prepared document digest separately.
type Release struct {
	Format        int
	Serial        uint64
	Class         string // normal or emergency
	Payload       []byte
	Candidate     []byte
	Media         []byte // exact media.json bytes: ISO/rootfs hash/size/location
	Provenance    map[string]string
	Qualification string // local-only or native-install-upgrade-recovery
	Evidence      map[string]string
	Notes         string
}

func validReleaseIdentity(r Release) bool {
	return r.Format == 1 && r.Serial != 0 && (r.Class == "normal" || r.Class == "emergency") && (r.Qualification == "local-only" || r.Qualification == "native-install-upgrade-recovery")
}

func validReleaseNotesAndEvidence(r Release) bool {
	return len(r.Evidence) > 0 && len(r.Evidence) <= 64 && len(r.Notes) > 0 && len(r.Notes) <= 16384
}

func decodeReleasePayloads(r Release) (Payload, Candidate, error) {
	var p Payload
	var c Candidate
	if decode(r.Payload, &p) != nil || decode(r.Candidate, &c) != nil {
		return p, c, ErrRefused
	}
	return p, c, nil
}

func validMediaFile(f MediaFile) bool {
	return f.Path != "" && !strings.Contains(f.Path, "..") && !strings.ContainsAny(f.Path, "\n\r\x00") && build.Digest(f.SHA256) && f.Bytes > 0
}

func validMediaURL(url string) bool {
	return url != "" && !strings.ContainsAny(url, "\n\r\x00 ") && (strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://"))
}

func validMediaBinding(m MediaBinding, p Payload, c Candidate) bool {
	if m.Revision != p.Revision || m.Architecture != p.Architecture || m.HostManifest != c.Host.Manifest || m.PayloadSHA256 != c.PayloadSHA256 {
		return false
	}
	return validMediaFile(m.ISO) && validMediaFile(m.Rootfs) && validMediaURL(m.RootfsURL)
}

func decodeReleaseMedia(r Release, p Payload, c Candidate) error {
	if len(r.Media) == 0 || len(r.Media) > 1<<20 {
		return ErrRefused
	}
	var m MediaBinding
	if json.Unmarshal(r.Media, &m) != nil || !validMediaBinding(m, p, c) {
		return ErrRefused
	}
	return nil
}

func validReleaseProvenance(r Release, p Payload) bool {
	if len(r.Provenance) != 4 {
		return false
	}
	for _, n := range []string{"source.tar", "app-inputs.json", "packages.txt", "presentation.json"} {
		if !Digest(r.Provenance[n]) {
			return false
		}
	}
	return r.Provenance["packages.txt"] == "sha256:"+p.HostPackagesSHA256 && r.Provenance["presentation.json"] == "sha256:"+p.PresentationSHA256
}

func validReleaseEvidence(evidence map[string]string) bool {
	for n, h := range evidence {
		if len(n) == 0 || len(n) > 128 || strings.ContainsAny(n, "\n\r\x00") || !Digest(h) {
			return false
		}
	}
	return true
}

func (r Release) Validate(t Trust) (Payload, Candidate, error) {
	var p Payload
	var c Candidate
	if !validReleaseIdentity(r) || !validReleaseNotesAndEvidence(r) {
		return p, c, ErrRefused
	}
	p, c, err := decodeReleasePayloads(r)
	if err != nil {
		return p, c, err
	}
	if p.RepositoryPrefix != t.Prefix || c.Validate(p, r.Payload) != nil {
		return p, c, ErrRefused
	}
	if !validReleaseProvenance(r, p) || !validReleaseEvidence(r.Evidence) || decodeReleaseMedia(r, p, c) != nil {
		return p, c, ErrRefused
	}
	return p, c, nil
}

func (r Release) References(t Trust) ([]string, error) {
	p, c, e := r.Validate(t)
	if e != nil {
		return nil, e
	}
	refs := []string{c.HostReference}
	for _, n := range Names {
		refs = append(refs, p.Images[n].Reference)
	}
	return refs, nil
}

type Channel struct {
	Format          int
	Name            string
	Sequence        uint64
	Issued, Expires int64
	Withdrawn       bool
	Releases        map[string]string // advertised architecture -> signed release OCI digest reference
}
type Seen struct {
	Sequence uint64
	Digest   string
	Issued   int64
}
type Highwater struct {
	Format     int
	TrustEpoch uint64
	CheckedAt  int64
	Channels   map[string]Seen
	Serials    map[string]uint64
	Releases   map[string]string // same serial must retain exactly the same release digest
}

func EmptyState() Highwater {
	return Highwater{Format: 1, Channels: map[string]Seen{}, Serials: map[string]uint64{}, Releases: map[string]string{}}
}

func validHighwaterMaps(s Highwater) bool {
	return s.Channels != nil && s.Serials != nil && s.Releases != nil && len(s.Channels) <= 3 && len(s.Serials) <= 2 && len(s.Releases) == len(s.Serials)
}

func validHighwaterChannels(s Highwater) bool {
	for c, v := range s.Channels {
		if !channel(c) || v.Sequence == 0 || !Digest(v.Digest) || v.Issued <= 0 {
			return false
		}
	}
	return true
}

func validHighwaterSerials(s Highwater) bool {
	for a, n := range s.Serials {
		if _, e := build.OCIArchitecture(a); e != nil || n == 0 {
			return false
		}
		if !Digest(s.Releases[a]) {
			return false
		}
	}
	return true
}

func (s Highwater) Validate() error {
	if s.Format != 1 || !validHighwaterMaps(s) || s.CheckedAt < 0 {
		return ErrRefused
	}
	if !validHighwaterChannels(s) || !validHighwaterSerials(s) {
		return ErrRefused
	}
	return nil
}

func validateReleaseReference(t Trust, arch, ref string) error {
	if _, err := build.OCIArchitecture(arch); err != nil {
		return ErrRefused
	}
	repo, role, err := t.Reference(ref)
	if err != nil || role != "artifact" || repo != t.Prefix+"-release" {
		return ErrRefused
	}
	return nil
}

func validateChannelReleases(t Trust, c Channel) error {
	if c.Withdrawn {
		if len(c.Releases) != 0 {
			return ErrRefused
		}
		return nil
	}
	if len(c.Releases) < 1 || len(c.Releases) > 2 {
		return ErrRefused
	}
	for a, ref := range c.Releases {
		if err := validateReleaseReference(t, a, ref); err != nil {
			return err
		}
	}
	return nil
}

func validateChannelIdentity(t Trust, c Channel, digest, wanted string) error {
	if t.Validate() != nil || !channel(wanted) || !Digest(digest) {
		return ErrRefused
	}
	if c.Format != 1 || c.Name != wanted || c.Sequence < t.MinimumSequence[wanted] {
		return ErrRefused
	}
	return nil
}

func validateChannelTiming(t Trust, s Highwater, c Channel, now time.Time) error {
	nowUnix := now.Unix()
	if nowUnix < t.NotBefore || nowUnix < s.CheckedAt-t.ClockSkewSeconds {
		return ErrRefused
	}
	if c.Issued < t.NotBefore || c.Issued > nowUnix+t.ClockSkewSeconds {
		return ErrRefused
	}
	if c.Expires <= nowUnix || c.Expires <= c.Issued || c.Expires-c.Issued > t.MaxAgeSeconds {
		return ErrRefused
	}
	return nil
}

func validateChannelProgression(old Seen, c Channel, digest string) error {
	if c.Sequence < old.Sequence || c.Issued < old.Issued {
		return ErrRefused
	}
	if c.Sequence == old.Sequence && digest != old.Digest {
		return ErrRefused
	}
	return nil
}

func advanceHighwaterChannel(s Highwater, epoch uint64, wanted, digest string, seq uint64, issued, nowUnix int64) Highwater {
	b, _ := json.Marshal(s)
	var next Highwater
	_ = json.Unmarshal(b, &next)
	next.TrustEpoch = epoch
	next.CheckedAt = max(s.CheckedAt, nowUnix)
	next.Channels[wanted] = Seen{Sequence: seq, Digest: digest, Issued: issued}
	return next
}

// AdmitChannel is pure. Persist its result as soon as the signed channel is
// validated, even when a later artifact is unavailable. Equal, unexpired offers
// permit safe observation retries; same-sequence substitutions never do.
func AdmitChannel(t Trust, s Highwater, c Channel, digest, wanted string, now time.Time) (Highwater, error) {
	if s.Validate() != nil || s.TrustEpoch > t.Epoch {
		return s, ErrRefused
	}
	if err := validateChannelIdentity(t, c, digest, wanted); err != nil {
		return s, err
	}
	if err := validateChannelTiming(t, s, c, now); err != nil {
		return s, err
	}
	if err := validateChannelReleases(t, c); err != nil {
		return s, err
	}
	if err := validateChannelProgression(s.Channels[wanted], c, digest); err != nil {
		return s, err
	}
	return advanceHighwaterChannel(s, t.Epoch, wanted, digest, c.Sequence, c.Issued, now.Unix()), nil
}

func admitChannelRef(c Channel, arch, ref string) bool {
	return channel(c.Name) && c.Format == 1 && !c.Withdrawn && c.Releases[arch] == ref
}

func admitReleasePayload(t Trust, s Highwater, c Channel, arch, ref string, r Release) error {
	p, _, e := r.Validate(t)
	if e != nil || s.Validate() != nil || !admitChannelRef(c, arch, ref) {
		return ErrRefused
	}
	if p.Architecture != arch {
		return ErrRefused
	}
	if c.Name != "candidate" && r.Qualification != "native-install-upgrade-recovery" {
		return ErrRefused
	}
	return nil
}

func admitReleaseDigest(s Highwater, arch, ref string, r Release) (string, error) {
	_, d, _ := strings.Cut(ref, "@")
	if r.Serial < s.Serials[arch] || (r.Serial == s.Serials[arch] && s.Releases[arch] != d) {
		return "", ErrRefused
	}
	return d, nil
}

func admitReleaseRef(t Trust, s Highwater, c Channel, arch, ref string, r Release) (string, error) {
	if err := admitReleasePayload(t, s, c, arch, ref, r); err != nil {
		return "", err
	}
	return admitReleaseDigest(s, arch, ref, r)
}

func AdmitRelease(t Trust, s Highwater, c Channel, arch, ref string, r Release) (Highwater, error) {
	d, e := admitReleaseRef(t, s, c, arch, ref, r)
	if e != nil {
		return s, e
	}
	b, _ := json.Marshal(s)
	var next Highwater
	_ = json.Unmarshal(b, &next)
	next.Serials[arch] = r.Serial
	next.Releases[arch] = d
	return next, nil
}

// Permit is produced by the protected qualification/promotion owner, never by
// trusting a build's "passed" label. The future scheduler supplies it unattended.
// Each exact digest/repository is admitted separately; no arbitrary sign target.
type Permit struct {
	Format             int
	Repository, Digest string
	Previous           string // channel publication only: exact previous manifest, or "absent" for bootstrap
	Expires            int64
}

func (p Permit) Validate(t Trust, now time.Time) error {
	_, e := t.Role(p.Repository)
	if e != nil || p.Format != 1 || !Digest(p.Digest) || p.Expires <= now.Unix() || p.Expires-now.Unix() > 86400 {
		return ErrRefused
	}
	return nil
}

func marshal(v any) ([]byte, error) {
	b, e := json.MarshalIndent(v, "", "  ")
	return append(b, '\n'), e
}
func errorAt(operation string) error { return fmt.Errorf("%s: %w", operation, ErrRefused) }
