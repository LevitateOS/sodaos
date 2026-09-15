package nativefinalization

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/nativebuild"
	rd "github.com/levitateos/sodaos/internal/releasedelivery"
	"github.com/stretchr/testify/require"
)

// Explicit noninteractive local B5 evidence against a retained candidate+media.
// Synthetic keys only; never production custody or GHCR.
func TestLocalSignedFinalMetadata(t *testing.T) {
	out := os.Getenv("SODA_B5_FINAL_OUT")
	candidateSrc := os.Getenv("SODA_B5_CANDIDATE")
	if out == "" || candidateSrc == "" {
		t.Skip("explicit SODA_B5_FINAL_OUT and SODA_B5_CANDIDATE required")
	}
	require.NoError(t, nativebuild.FreshDirectory(out))
	require.NoError(t, os.Chmod(out, 0o700))

	stage := filepath.Join(out, "candidate")
	require.NoError(t, stageCandidate(candidateSrc, stage))
	mediaPath := filepath.Join(out, "media.json")
	require.NoError(t, writeReconstructedMedia(stage, mediaPath))

	evidence := filepath.Join(out, "qualification.json")
	require.NoError(t, nativebuild.WriteNew(evidence, []byte(`{"scope":"local-only","note":"retained-candidate B5 mechanism evidence; not protected P9"}`+"\n"), 0o600))

	n := rd.Native{Home: out}
	require.NoError(t, rd.CheckNative(t.Context(), n))
	trust, keys := syntheticReleaseTrust(t, n, out, stage)

	sum, err := nativebuild.HashFile(evidence)
	require.NoError(t, err)
	q := rd.Qualification{
		Serial:   1,
		Class:    "normal",
		Scope:    "local-only",
		Notes:    "B5 local signed final metadata against retained candidate; fixture keys only",
		Evidence: map[string]string{"qualification.json": "sha256:" + sum},
	}
	prepared := filepath.Join(out, "prepared")
	digest, err := rd.Prepare(trust, stage, mediaPath, q, prepared)
	require.NoError(t, err)
	signed := filepath.Join(out, "signed")
	permit := rd.Permit{Format: 1, Repository: trust.Prefix + "-release", Digest: digest, Expires: time.Now().Add(time.Hour).Unix()}
	require.NoError(t, rd.Sign(t.Context(), n, trust, permit, "oci", prepared, signed, keys))
	ref := permit.Repository + "@" + digest
	require.NoError(t, rd.VerifyCopy(t.Context(), n, trust, ref, "dir:"+filepath.Join(signed, "signed"), filepath.Join(out, "verified")))

	var release rd.Release
	require.NoError(t, rd.ReadDocument(filepath.Join(out, "verified/image"), digest, &release))
	require.NotEmpty(t, release.Media)
	_, _, err = release.Validate(trust)
	require.NoError(t, err)

	require.Error(t, rd.Sign(t.Context(), n, trust, rd.Permit{Format: 1, Repository: trust.Prefix + "-release", Digest: rd.Hash([]byte("wrong")), Expires: permit.Expires}, "oci", prepared, filepath.Join(out, "wrong-digest"), keys))
	require.NoDirExists(t, filepath.Join(out, "wrong-digest/signed"))

	receipt := map[string]any{
		"reference": ref,
		"digest":    digest,
		"scope":     "local signed final release metadata with embedded media.json; fixture keys; not production custody or GHCR",
		"candidate": candidateSrc,
		"refused":   []string{"wrong digest never signed"},
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	require.NoError(t, err)
	require.NoError(t, nativebuild.WriteNew(filepath.Join(out, "final.json"), append(data, '\n'), 0o600))
}

func stageCandidate(src, dest string) error {
	if err := nativebuild.FreshDirectory(dest); err != nil {
		return err
	}
	names := []string{
		"payload.json", "candidate.json", "app-inputs.json", "packages.txt",
		"host.oci", "forgejo-context", "images",
	}
	for _, name := range names {
		if err := linkOrCopy(filepath.Join(src, name), filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	source := filepath.Join(filepath.Dir(src), "inputs", "source.tar")
	if _, err := os.Stat(source); err != nil {
		source = filepath.Join(src, "source.tar")
	}
	return linkOrCopy(source, filepath.Join(dest, "source.tar"))
}

func linkOrCopy(src, dest string) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dest, rel)
			if info.IsDir() {
				return os.MkdirAll(target, 0o700)
			}
			return os.Link(path, target)
		})
	}
	return os.Link(src, dest)
}

func writeReconstructedMedia(candidate, out string) error {
	var payload struct {
		Revision, Architecture string
	}
	var candidateMeta struct {
		Host struct {
			Manifest string
		}
		PayloadSHA256 string
	}
	pb, err := os.ReadFile(filepath.Join(candidate, "payload.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(pb, &payload); err != nil {
		return err
	}
	cb, err := os.ReadFile(filepath.Join(candidate, "candidate.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(cb, &candidateMeta); err != nil {
		return err
	}
	iso := filepath.Join(os.Getenv("SODA_B5_CANDIDATE"), "media", "minimal.iso")
	rootfsEntries, err := filepath.Glob(filepath.Join(os.Getenv("SODA_B5_CANDIDATE"), "media", "*-rootfs.img"))
	if err != nil || len(rootfsEntries) != 1 {
		return rd.ErrRefused
	}
	isoSum, err := nativebuild.HashFile(iso)
	if err != nil {
		return err
	}
	isoStat, err := os.Stat(iso)
	if err != nil {
		return err
	}
	rootSum, err := nativebuild.HashFile(rootfsEntries[0])
	if err != nil {
		return err
	}
	rootStat, err := os.Stat(rootfsEntries[0])
	if err != nil {
		return err
	}
	media := rd.MediaBinding{
		Revision: payload.Revision, Architecture: payload.Architecture,
		HostManifest: candidateMeta.Host.Manifest, PayloadSHA256: candidateMeta.PayloadSHA256,
		RootfsURL: "https://example.invalid/" + filepath.Base(rootfsEntries[0]),
		ISO:       rd.MediaFile{Path: "minimal.iso", SHA256: isoSum, Bytes: isoStat.Size()},
		Rootfs:    rd.MediaFile{Path: filepath.Base(rootfsEntries[0]), SHA256: rootSum, Bytes: rootStat.Size()},
	}
	data, err := json.MarshalIndent(media, "", "  ")
	if err != nil {
		return err
	}
	return nativebuild.WriteNew(out, append(data, '\n'), 0o600)
}

func syntheticReleaseTrust(t *testing.T, n rd.Native, out, candidate string) (rd.Trust, rd.SecretFiles) {
	t.Helper()
	var payload struct{ RepositoryPrefix string }
	pb, err := os.ReadFile(filepath.Join(candidate, "payload.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(pb, &payload))
	pass := filepath.Join(out, "synthetic-passphrase")
	var random [32]byte
	_, err = rand.Read(random[:])
	require.NoError(t, err)
	require.NoError(t, nativebuild.WriteNew(pass, []byte(hex.EncodeToString(random[:])), 0o600))
	now := time.Now().Unix()
	trust := rd.Trust{
		Format: 1, Prefix: payload.RepositoryPrefix, Epoch: 1, NotBefore: now - 600,
		MaxAgeSeconds: 3600, ClockSkewSeconds: 10,
		Keys:            map[string][]string{},
		MinimumSequence: map[string]uint64{"candidate": 1, "preview": 1, "stable": 1},
	}
	var artifactKeys rd.SecretFiles
	for _, role := range []string{"artifact", "candidate", "preview", "stable"} {
		prefix := filepath.Join(out, "synthetic-"+role)
		_, err = n.Run(t.Context(), "generate-sigstore-key", "--output-prefix", prefix, "--passphrase-file", pass)
		require.NoError(t, err)
		pub, err := rd.ReadFile(prefix+".pub", 16384)
		require.NoError(t, err)
		trust.Keys[role] = []string{string(pub)}
		if role == "artifact" {
			artifactKeys = rd.SecretFiles{Key: prefix + ".private", Passphrase: pass}
		}
	}
	require.NoError(t, trust.Validate())
	require.NoError(t, writePrivateJSON(filepath.Join(out, "synthetic-trust.json"), trust))
	return trust, artifactKeys
}

func writePrivateJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return nativebuild.WriteNew(path, append(data, '\n'), 0o600)
}
