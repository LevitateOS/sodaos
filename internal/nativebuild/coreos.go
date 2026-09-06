package nativebuild

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type CoreOSImage struct{ URL, SignatureURL, SHA256, UncompressedSHA256 string }
type CoreOSLock struct {
	MetadataURL, Release string
	Architectures        map[string]CoreOSImage
}
type VerifiedBase struct{ Path, SHA256, Architecture, Release, Signer string }

func httpsURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}
func ReadCoreOS(lock, arch string) (CoreOSLock, CoreOSImage, error) {
	var l CoreOSLock
	if _, err := OCIArchitecture(arch); err != nil {
		return l, CoreOSImage{}, err
	}
	if err := ReadJSON(lock, &l); err != nil {
		return l, CoreOSImage{}, err
	}
	img, ok := l.Architectures[arch]
	if !ok || l.Release == "" || !httpsURL(l.MetadataURL) || !httpsURL(img.URL) || !httpsURL(img.SignatureURL) || !Digest(img.SHA256) || !Digest(img.UncompressedSHA256) {
		return l, img, errors.New("invalid CoreOS lock")
	}
	return l, img, nil
}

// FetchCoreOS uses an explicitly supplied trusted keyring and signer. It does
// not download/import trust roots or permit an unsigned-image fallback.
func FetchCoreOS(ctx context.Context, lock, arch, keyring, signer, out string) (VerifiedBase, error) {
	var result VerifiedBase
	if err := RequireNative(arch); err != nil {
		return result, err
	}
	l, img, err := ReadCoreOS(lock, arch)
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
	for _, name := range []string{"gpgv", "xz"} {
		if _, err = exec.LookPath(name); err != nil {
			return result, err
		}
	}
	if err = FreshDirectory(out); err != nil {
		return result, err
	}
	archive := filepath.Join(out, "coreos.qcow2.xz")
	sig := archive + ".sig"
	if err = download(ctx, img.URL, archive, 8<<30); err != nil {
		return result, err
	}
	if sum, e := HashFile(archive); e != nil || sum != img.SHA256 {
		return result, errors.Join(e, errors.New("compressed CoreOS checksum mismatch"))
	}
	if err = download(ctx, img.SignatureURL, sig, 1<<20); err != nil {
		return result, err
	}
	home := filepath.Join(out, "gnupg")
	if err = os.Mkdir(home, 0700); err != nil {
		return result, err
	}
	cmd := exec.CommandContext(ctx, "gpgv", "--homedir", home, "--keyring", keyring, "--status-fd=1", sig, archive)
	status, err := cmd.Output()
	if err != nil {
		return result, errors.New("CoreOS signature verification failed")
	}
	if !validSignature(status, signer) {
		return result, errors.New("signature does not match selected signer")
	}
	dest := filepath.Join(out, "coreos.qcow2")
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return result, err
	}
	hash := sha256.New()
	cmd = exec.CommandContext(ctx, "xz", "--decompress", "--stdout", "--", archive)
	cmd.Stdout = &limitWriter{w: io.MultiWriter(f, hash), remaining: 64 << 30}
	cmd.WaitDelay = 2 * time.Second
	err = errors.Join(cmd.Run(), f.Close())
	if err != nil {
		return result, errors.New("CoreOS decompression failed")
	}
	if hex.EncodeToString(hash.Sum(nil)) != img.UncompressedSHA256 {
		return result, errors.New("uncompressed CoreOS checksum mismatch")
	}
	if err = os.Chmod(dest, 0444); err != nil {
		return result, err
	}
	result = VerifiedBase{dest, img.UncompressedSHA256, arch, l.Release, strings.ToUpper(signer)}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, err
	}
	err = WriteNew(filepath.Join(out, "verified-base.json"), append(data, '\n'), 0600)
	return result, err
}
func validSignature(status []byte, signer string) bool {
	valid := false
	for _, line := range strings.Split(string(status), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] != "[GNUPG:]" {
			continue
		}
		switch f[1] {
		case "BADSIG", "ERRSIG", "EXPSIG", "EXPKEYSIG", "REVKEYSIG", "KEYEXPIRED", "SIGEXPIRED", "NO_PUBKEY", "NODATA":
			return false
		}
		if f[1] == "VALIDSIG" && (len(f) == 11 || len(f) == 12) && (strings.EqualFold(f[2], signer) || (len(f) == 12 && strings.EqualFold(f[11], signer))) {
			valid = true
		}
	}
	return valid
}

type limitWriter struct {
	w         io.Writer
	remaining int64
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, errors.New("input exceeds size limit")
	}
	n, err := w.w.Write(p)
	w.remaining -= int64(n)
	return n, err
}
func download(ctx context.Context, source, dest string, max int64) error {
	client := http.Client{Timeout: 30 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || !httpsURL(req.URL.String()) {
			return errors.New("unsafe download redirect")
		}
		return nil
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("CoreOS download failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("CoreOS download HTTP failure")
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(&limitWriter{w: f, remaining: max}, resp.Body)
	return errors.Join(err, f.Close())
}
