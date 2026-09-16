package build

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

// FetchCoreOSISO retains a new, signature-verified upstream ISO. No customization,
// boot, key import or disk installation is implicit. The ISO resolves live
// from the stable stream; no stored version is consulted.
func admitCoreOSISOFetch(arch, signer, keyring string) (string, error) {
	if err := RequireNative(arch); err != nil {
		return "", err
	}
	if !regexp.MustCompile(`^(?:[A-Fa-f0-9]{40}|[A-Fa-f0-9]{64})$`).MatchString(signer) {
		return "", errors.New("full trusted signer fingerprint required")
	}
	if _, err := HashFile(keyring); err != nil {
		return "", err
	}
	keyring, err := filepath.Abs(keyring)
	if err != nil {
		return "", err
	}
	if _, err = exec.LookPath("gpgv"); err != nil {
		return "", err
	}
	return keyring, nil
}

func fetchVerifiedISO(ctx context.Context, img CoreOSImage, dest, keyring, signer, out string) error {
	if err := download(ctx, img.URL, dest, 8<<30); err != nil {
		return err
	}
	if sum, e := HashFile(dest); e != nil || sum != img.SHA256 {
		return errors.New("CoreOS ISO checksum mismatch")
	}
	if err := download(ctx, img.SignatureURL, dest+".sig", 1<<20); err != nil {
		return err
	}
	return verifyCoreOSSignature(ctx, dest, dest+".sig", keyring, signer, out)
}

func FetchCoreOSISO(ctx context.Context, arch, keyring, signer, out string) (VerifiedBase, error) {
	var result VerifiedBase
	keyring, err := admitCoreOSISOFetch(arch, signer, keyring)
	if err != nil {
		return result, err
	}
	release, img, err := ResolveCoreOSISO(ctx, arch)
	if err != nil {
		return result, err
	}
	if err = FreshDirectory(out); err != nil {
		return result, err
	}
	dest := filepath.Join(out, "coreos.iso")
	if err = fetchVerifiedISO(ctx, img, dest, keyring, signer, out); err != nil {
		return result, err
	}
	if err = os.Chmod(dest, 0o444); err != nil {
		return result, err
	}
	result = VerifiedBase{dest, img.SHA256, arch, release, strings.ToUpper(signer)}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, err
	}
	return result, WriteNew(filepath.Join(out, "verified-iso.json"), append(data, '\n'), 0o600)
}
