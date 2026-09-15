package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/nativebuild"
	"golang.org/x/sys/unix"
)

const (
	mediaPayloadRoot     = "/run/media/iso/soda/bundle"
	installedPayloadRoot = "/var/lib/soda-installer"
	payloadReceiptSchema = 1
	payloadFreeReserve   = uint64(1 << 30)
)

type payloadReceipt struct {
	Schema       int
	Release      string
	Architecture string
	Revision     string
	BundleSHA256 string
	Bytes        uint64
}

func payloadRequirement(media mediaIdentity) (uint64, error) {
	if media.Format == 2 {
		return candidateRequirement(media, "/")
	}
	var stat unix.Statfs_t
	if err := unix.Statfs("/run/media/iso", &stat); err != nil || uint64(stat.Type) != uint64(unix.ISOFS_SUPER_MAGIC) || stat.Flags&unix.ST_RDONLY == 0 {
		return 0, errors.New("read-only ISO installation media required")
	}
	return payloadRequirementAt(filepath.Join(mediaPayloadRoot, media.Architecture), media, verifyTrustedBundle)
}

func validMediaPayloadIdentity(media mediaIdentity) bool {
	return media.Architecture == architecture() && media.Release != "" && media.InstallerVersion == "coreos-installer 0.26.0" && nativebuild.Revision(media.Revision) && nativebuild.Digest(media.BundleSHA256)
}

func addPayloadSize(total uint64, info os.FileInfo) (uint64, error) {
	switch {
	case info.Mode().IsRegular():
		size := uint64(info.Size())
		if ^uint64(0)-total < size {
			return 0, errors.New("media payload size overflow")
		}
		return total + size, nil
	case info.IsDir(), info.Mode()&os.ModeSymlink != 0:
		return total, nil
	default:
		return 0, errors.New("unsupported media payload file type")
	}
}

type payloadSizeAcc struct{ total uint64 }

func (a *payloadSizeAcc) walk(_ string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	info, err := entry.Info()
	if err != nil {
		return err
	}
	a.total, err = addPayloadSize(a.total, info)
	return err
}

func payloadRequirementAt(source string, media mediaIdentity, verify func(string, string, string) (nativebuild.Inventory, error)) (uint64, error) {
	if !validMediaPayloadIdentity(media) {
		return 0, errors.New("incomplete or mismatched media payload identity")
	}
	inventory, err := verify(source, media.BundleSHA256, media.Architecture)
	if err != nil || inventory.Revision != media.Revision || inventory.Architecture != media.Architecture {
		return 0, errors.New("media payload does not match its trusted identity")
	}
	var acc payloadSizeAcc
	err = filepath.WalkDir(source, acc.walk)
	if err != nil || acc.total == 0 {
		return 0, errors.New("cannot measure verified media payload")
	}
	return acc.total, nil
}

func installedPayload() (bundle, digest, revision string, err error) {
	identity, err := readRegular("/usr/lib/os-release", 16384)
	if err != nil {
		return "", "", "", errors.New("cannot read installed CoreOS release identity")
	}
	release := osRelease(identity)["IMAGE_VERSION"]
	if release == "" {
		return "", "", "", errors.New("installed CoreOS release identity is incomplete")
	}
	return installedPayloadAt(installedPayloadRoot, architecture(), release, verifyTrustedBundle)
}

func installedPayloadAt(root, arch, release string, verify func(string, string, string) (nativebuild.Inventory, error)) (bundle, digest, revision string, err error) {
	return installedPayloadAtWith(root, arch, release, verify, protectedBundle)
}

func rejectIncompleteMediaCopy(root string) error {
	for _, name := range []string{"media-copy-ready.json", "media-copy-incomplete.json"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			return errors.New("incomplete installer media-copy state requires operator inspection")
		}
	}
	return nil
}

func decodePayloadReceipt(data []byte, arch, release string) (payloadReceipt, error) {
	var receipt payloadReceipt
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return receipt, errors.New("completed installer media-copy receipt required")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return receipt, errors.New("completed installer media-copy receipt required")
	}
	if receipt.Schema != payloadReceiptSchema || receipt.Architecture != arch || receipt.Release != release || receipt.Bytes == 0 || !nativebuild.Digest(receipt.BundleSHA256) || !nativebuild.Revision(receipt.Revision) {
		return receipt, errors.New("invalid installer media-copy receipt")
	}
	return receipt, nil
}

func readMediaCopyReceipt(root, arch, release string) (payloadReceipt, error) {
	var receipt payloadReceipt
	receiptPath := filepath.Join(root, "media-copy.json")
	info, err := os.Lstat(receiptPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return receipt, errors.New("private regular installer media-copy receipt required")
	}
	data, err := readRegular(receiptPath, 4096)
	if err != nil {
		return receipt, errors.New("completed installer media-copy receipt required")
	}
	return decodePayloadReceipt(data, arch, release)
}

func installedPayloadAtWith(root, arch, release string, verify func(string, string, string) (nativebuild.Inventory, error), protect func(string) error) (bundle, digest, revision string, err error) {
	if err := protect(root); err != nil {
		return "", "", "", errors.New("protected installed payload state required")
	}
	if err := rejectIncompleteMediaCopy(root); err != nil {
		return "", "", "", err
	}
	receipt, err := readMediaCopyReceipt(root, arch, release)
	if err != nil {
		return "", "", "", err
	}
	bundle = filepath.Join(root, "bundle", arch)
	inventory, err := verify(bundle, receipt.BundleSHA256, arch)
	if err != nil || inventory.Architecture != arch || inventory.Revision != receipt.Revision {
		return "", "", "", errors.New("installed payload does not match its completed receipt")
	}
	return bundle, receipt.BundleSHA256, receipt.Revision, nil
}
