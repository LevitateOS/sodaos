package releasedelivery

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/stretchr/testify/require"
)

func publicKey(t *testing.T) string {
	t.Helper()
	k, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, e)
	b, e := x509.MarshalPKIXPublicKey(&k.PublicKey)
	require.NoError(t, e)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b}))
}
func testTrust(t *testing.T) Trust {
	now := time.Now().Unix()
	tr := Trust{Format: 1, Prefix: "ghcr.io/example/sodaos", Epoch: 1, NotBefore: now - 600, MaxAgeSeconds: 3600, ClockSkewSeconds: 10, Keys: map[string][]string{}, MinimumSequence: map[string]uint64{"candidate": 1, "preview": 1, "stable": 1}}
	for _, role := range []string{"artifact", "candidate", "preview", "stable"} {
		tr.Keys[role] = []string{publicKey(t)}
	}
	require.NoError(t, tr.Validate())
	return tr
}
func testRelease(t *testing.T, tr Trust) Release {
	p := appliancerelease.Payload{Format: 3, CoreOS: "44.20260817.3.2", Revision: strings.Repeat("a", 40), Architecture: "x86_64", Schema: 10, RepositoryPrefix: tr.Prefix, Base: "quay.io/fedora/fedora-coreos@sha256:" + strings.Repeat("b", 64), PresentationSHA256: strings.Repeat("c", 64), HostPackagesSHA256: strings.Repeat("d", 64), Images: map[string]appliancerelease.Image{}}
	p.ID = p.CoreOS + ".soda-" + p.Revision[:12]
	for i, n := range appliancerelease.Names {
		d := Hash([]byte(n))
		p.Images[n] = appliancerelease.Image{Reference: tr.Prefix + "-" + n + "@" + d, Manifest: d, Config: Hash([]byte{byte(i)}), ArchiveSHA256: strings.Repeat("1", 64)}
	}
	pb, e := marshal(p)
	require.NoError(t, e)
	c := Candidate{Format: 1, Host: nativebuild.Image{Manifest: Hash([]byte("host")), Config: Hash([]byte("host-config")), Architecture: "amd64", Revision: p.Revision, Source: "https://github.com/LevitateOS/sodaos", BaseName: p.Base, BaseDigest: "sha256:" + strings.Repeat("b", 64)}, HostArchiveSHA256: strings.Repeat("f", 64), PayloadSHA256: strings.TrimPrefix(Hash(pb), "sha256:"), Migration: "no upgrade qualified", Notes: "synthetic fixture"}
	c.HostReference = tr.Prefix + "-host@" + c.Host.Manifest
	cb, e := marshal(c)
	require.NoError(t, e)
	r := Release{Format: 1, Serial: 10, Class: "normal", Payload: pb, Candidate: cb, Qualification: "local-only", Evidence: map[string]string{"synthetic": Hash([]byte("receipt"))}, Notes: "fixture", Provenance: map[string]string{}}
	for _, n := range []string{"source.tar", "app-inputs.json", "packages.txt", "presentation.json"} {
		r.Provenance[n] = Hash([]byte(n))
	}
	r.Provenance["packages.txt"] = "sha256:" + p.HostPackagesSHA256
	r.Provenance["presentation.json"] = "sha256:" + p.PresentationSHA256
	_, _, e = r.Validate(tr)
	require.NoError(t, e)
	return r
}
func TestSharedLayoutPayloadKeepsDeliveryArchiveBindings(t *testing.T) {
	tr := testTrust(t)
	r := testRelease(t, tr)
	var p appliancerelease.Payload
	var c Candidate
	require.NoError(t, decode(r.Payload, &p))
	require.NoError(t, decode(r.Candidate, &c))
	p.Schema++
	var err error
	r.Payload, err = marshal(p)
	require.NoError(t, err)
	// Changed payloads cannot reuse an earlier candidate binding.
	_, _, err = r.Validate(tr)
	require.Error(t, err)
	c.PayloadSHA256 = strings.TrimPrefix(Hash(r.Payload), "sha256:")
	r.Candidate, err = marshal(c)
	require.NoError(t, err)
	_, _, err = r.Validate(tr)
	require.NoError(t, err)
	for _, image := range p.Images {
		require.Equal(t, strings.Repeat("1", 64), image.ArchiveSHA256)
	}
}

func testChannel(tr Trust, name string) Channel {
	now := time.Now().Unix()
	return Channel{Format: 1, Name: name, Sequence: 1, Issued: now - 30, Expires: now + 600, Releases: map[string]string{"x86_64": tr.Prefix + "-release@" + Hash([]byte("release"))}}
}
func TestAuthorityFreshnessReplayAndWithdrawal(t *testing.T) {
	tr := testTrust(t)
	c := testChannel(tr, "stable")
	d := Hash([]byte("channel"))
	now := time.Now()
	state, e := AdmitChannel(tr, EmptyState(), c, d, "stable", now)
	require.NoError(t, e)
	_, e = AdmitChannel(tr, state, c, d, "stable", now)
	require.NoError(t, e, "same exact fresh offer is safe to observe again")
	for _, mutate := range []func(*Channel){
		func(c *Channel) { c.Name = "preview" }, func(c *Channel) { c.Sequence = 0 }, func(c *Channel) { c.Expires = now.Unix() },
		func(c *Channel) { c.Issued = now.Unix() + 100 }, func(c *Channel) { c.Expires = c.Issued + 7200 },
		func(c *Channel) { c.Releases = map[string]string{"armv7": "no"} }, func(c *Channel) { c.Releases = nil },
		func(c *Channel) {
			c.Releases = map[string]string{"x86_64": "ghcr.io/other/release@" + Hash([]byte("bad"))}
		},
		func(c *Channel) { c.Withdrawn = true },
	} {
		copy := c
		mutate(&copy)
		_, e = AdmitChannel(tr, state, copy, d, "stable", now)
		require.Error(t, e)
	}
	_, e = AdmitChannel(tr, state, c, Hash([]byte("substitution")), "stable", now)
	require.Error(t, e)
	_, e = AdmitChannel(tr, state, c, d, "stable", now.Add(-time.Hour))
	require.Error(t, e)
	newer := tr
	newer.Epoch = 2
	state, e = AdmitChannel(newer, state, c, d, "stable", now)
	require.NoError(t, e)
	_, e = AdmitChannel(tr, state, c, d, "stable", now)
	require.Error(t, e, "trust rollback refused")
	withdrawn := c
	withdrawn.Sequence++
	withdrawn.Withdrawn = true
	withdrawn.Releases = nil
	state, e = AdmitChannel(newer, state, withdrawn, Hash([]byte("withdrawn")), "stable", now)
	require.NoError(t, e)
	_, e = AdmitChannel(newer, state, c, d, "stable", now)
	require.Error(t, e, "withdrawal cannot be replaced by replaying the old offer")
	require.Equal(t, uint64(2), state.Channels["stable"].Sequence)
}
func TestReleaseBindingQualificationAndDowngrade(t *testing.T) {
	tr := testTrust(t)
	r := testRelease(t, tr)
	c := testChannel(tr, "candidate")
	ref := c.Releases["x86_64"]
	state, e := AdmitRelease(tr, EmptyState(), c, "x86_64", ref, r)
	require.NoError(t, e)
	for _, name := range []string{"preview", "stable"} {
		copy := c
		copy.Name = name
		_, e = AdmitRelease(tr, state, copy, "x86_64", ref, r)
		require.Error(t, e, "local qualification cannot become a native preview/stable release")
	}
	_, e = AdmitRelease(tr, state, c, "aarch64", ref, r)
	require.Error(t, e)
	old := r
	old.Serial--
	_, e = AdmitRelease(tr, state, c, "x86_64", ref, old)
	require.Error(t, e)
	fork := c
	fork.Releases = map[string]string{"x86_64": tr.Prefix + "-release@" + Hash([]byte("fork"))}
	_, e = AdmitRelease(tr, state, fork, "x86_64", fork.Releases["x86_64"], r)
	require.Error(t, e)
	// A protected worker may admit native evidence only after the actual M3 tests;
	// changing the classification still cannot create missing/altered payload data.
	r.Qualification = "native-install-upgrade-recovery"
	r.Class = "emergency"
	c.Name = "stable"
	_, e = AdmitRelease(tr, state, c, "x86_64", ref, r)
	require.NoError(t, e)
	var p appliancerelease.Payload
	require.NoError(t, decode(r.Payload, &p))
	delete(p.Images, "proxy")
	r.Payload, _ = marshal(p)
	_, _, e = r.Validate(tr)
	require.Error(t, e)
	r = testRelease(t, tr)
	r.Candidate = append(r.Candidate, []byte(` {"extra":1}`)...)
	_, _, e = r.Validate(tr)
	require.Error(t, e)
}
func TestTrustRolesRotationAndPreservedVendorPolicy(t *testing.T) {
	tr := testTrust(t)
	base := []byte(`{"default":[{"type":"reject"}],"transports":{"docker":{"quay.io/fedora":[{"type":"signedBy","keyPath":"/vendor/key","keyType":"GPGKeys"}]},"containers-storage":{"": [{"type":"insecureAcceptAnything"}]}}}`)
	merged, e := MergePolicy(tr, base)
	require.NoError(t, e)
	var p map[string]any
	require.NoError(t, json.Unmarshal(merged, &p))
	docker := p["transports"].(map[string]any)["docker"].(map[string]any)
	require.Contains(t, docker, "quay.io/fedora")
	require.Contains(t, string(merged), "/vendor/key")
	require.Len(t, docker, 11)
	require.Contains(t, string(merged), "exactRepository")
	require.Contains(t, string(merged), "sigstoreSigned")
	_, e = MergePolicy(tr, merged)
	require.ErrorContains(t, e, "explicit review")
	tr.Keys["stable"] = append(tr.Keys["stable"], publicKey(t))
	require.NoError(t, tr.Validate(), "rotation overlap within the same role")
	tr.Keys["preview"] = tr.Keys["stable"]
	require.Error(t, tr.Validate(), "preview must not inherit stable signing authority")
}
func TestPrivateStateRefusesResetCorruptionAndConcurrentMutation(t *testing.T) {
	tr := testTrust(t)
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0700))
	path := filepath.Join(root, "state.json")
	require.NoError(t, InitState(path, tr))
	require.Error(t, InitState(path, tr), "no implicit recovery/reset")
	f, e := lockState(path)
	require.NoError(t, e)
	defer unlock(f)
	_, e = lockState(path)
	require.Error(t, e)
	require.NoError(t, saveState(path, EmptyState()))
	var state Highwater
	require.NoError(t, ReadJSON(path, &state))
	require.NoError(t, state.Validate())
	alias := filepath.Join(root, "alias")
	require.NoError(t, os.Symlink(path, alias))
	require.Error(t, PrivateFile(alias))
	require.NoError(t, os.Chmod(path, 0644))
	require.Error(t, PrivateFile(path))
}
