package nativebuild

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ReadCoreOSISO admits only uncompressed live ISOs, not QEMU or metal disk images.
func ReadCoreOSISO(lock, arch string) (CoreOSLock, CoreOSImage, error) {
	var l CoreOSLock
	if _, err := OCIArchitecture(arch); err != nil {
		return l, CoreOSImage{}, err
	}
	if err := ReadJSON(lock, &l); err != nil {
		return l, CoreOSImage{}, err
	}
	img, ok := l.Architectures[arch]
	if !ok || l.Release == "" || !httpsURL(l.MetadataURL) || !httpsURL(img.URL) || !strings.HasSuffix(img.URL, ".iso") || img.SignatureURL != img.URL+".sig" || !Digest(img.SHA256) || img.UncompressedSHA256 != "" {
		return l, img, errors.New("invalid uncompressed CoreOS ISO lock")
	}
	return l, img, nil
}

// FetchCoreOSISO retains a new, signature-verified upstream ISO. No customization,
// boot, key import or disk installation is implicit.
func FetchCoreOSISO(ctx context.Context, lock, arch, keyring, signer, out string) (VerifiedBase, error) {
	var result VerifiedBase
	if err := RequireNative(arch); err != nil {
		return result, err
	}
	l, img, err := ReadCoreOSISO(lock, arch)
	if err != nil {
		return result, err
	}
	if !regexp.MustCompile(`^(?:[A-Fa-f0-9]{40}|[A-Fa-f0-9]{64})$`).MatchString(signer) {
		return result, errors.New("full trusted signer fingerprint required")
	}
	if _, err = HashFile(keyring); err != nil {
		return result, err
	}
	keyring, err = filepath.Abs(keyring)
	if err != nil {
		return result, err
	}
	if _, err = exec.LookPath("gpgv"); err != nil {
		return result, err
	}
	if err = FreshDirectory(out); err != nil {
		return result, err
	}
	dest := filepath.Join(out, "coreos.iso")
	if err = download(ctx, img.URL, dest, 8<<30); err != nil {
		return result, err
	}
	if sum, e := HashFile(dest); e != nil || sum != img.SHA256 {
		return result, errors.New("CoreOS ISO checksum mismatch")
	}
	if err = download(ctx, img.SignatureURL, dest+".sig", 1<<20); err != nil {
		return result, err
	}
	if err = verifyCoreOSSignature(ctx, dest, dest+".sig", keyring, signer, out); err != nil {
		return result, err
	}
	if err = os.Chmod(dest, 0444); err != nil {
		return result, err
	}
	result = VerifiedBase{dest, img.SHA256, arch, l.Release, strings.ToUpper(signer)}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, err
	}
	return result, WriteNew(filepath.Join(out, "verified-iso.json"), append(data, '\n'), 0600)
}
