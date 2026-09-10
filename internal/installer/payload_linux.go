package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

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
	var stat unix.Statfs_t
	if err := unix.Statfs("/run/media/iso", &stat); err != nil || uint64(stat.Type) != uint64(unix.ISOFS_SUPER_MAGIC) || stat.Flags&unix.ST_RDONLY == 0 {
		return 0, errors.New("read-only ISO installation media required")
	}
	return payloadRequirementAt(filepath.Join(mediaPayloadRoot, media.Architecture), media, verifyTrustedBundle)
}

func payloadRequirementAt(source string, media mediaIdentity, verify func(string, string, string) (nativebuild.Inventory, error)) (uint64, error) {
	if media.Architecture != architecture() || media.Release == "" || media.InstallerVersion != "coreos-installer 0.26.0" || !nativebuild.Revision(media.Revision) || !nativebuild.Digest(media.BundleSHA256) {
		return 0, errors.New("incomplete or mismatched media payload identity")
	}
	inventory, err := verify(source, media.BundleSHA256, media.Architecture)
	if err != nil || inventory.Revision != media.Revision || inventory.Architecture != media.Architecture {
		return 0, errors.New("media payload does not match its trusted identity")
	}
	var total uint64
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case info.Mode().IsRegular():
			size := uint64(info.Size())
			if ^uint64(0)-total < size {
				return errors.New("media payload size overflow")
			}
			total += size
		case info.IsDir(), info.Mode()&os.ModeSymlink != 0:
		default:
			return errors.New("unsupported media payload file type")
		}
		return nil
	})
	if err != nil || total == 0 {
		return 0, errors.New("cannot measure verified media payload")
	}
	return total, nil
}

type installedBlockDevice struct {
	Name        string                 `json:"name"`
	KName       string                 `json:"kname"`
	PKName      string                 `json:"pkname"`
	Type        string                 `json:"type"`
	Size        uint64                 `json:"size"`
	Model       string                 `json:"model"`
	Serial      string                 `json:"serial"`
	WWN         string                 `json:"wwn"`
	MajorMinor  string                 `json:"maj:min"`
	ReadOnly    bool                   `json:"ro"`
	Mountpoints []*string              `json:"mountpoints"`
	FSType      string                 `json:"fstype"`
	Label       string                 `json:"label"`
	PartLabel   string                 `json:"partlabel"`
	UUID        string                 `json:"uuid"`
	PartUUID    string                 `json:"partuuid"`
	Children    []installedBlockDevice `json:"children"`
}

type installedRoot struct {
	Device     string
	MajorMinor string
}

func discoverInstalledRoot(ctx context.Context, selected Disk, run commandRunner) (installedRoot, error) {
	data, err := run(ctx, "lsblk", []string{"--json", "--bytes", "--paths", "--output", "NAME,KNAME,PKNAME,TYPE,SIZE,MODEL,SERIAL,WWN,MAJ:MIN,RO,MOUNTPOINTS,FSTYPE,LABEL,PARTLABEL,UUID,PARTUUID", selected.Device.Name}, nil)
	if err != nil {
		return installedRoot{}, errors.New("cannot inspect the installed disk")
	}
	sequence, err := diskSequence(selected.Device)
	if err != nil {
		return installedRoot{}, errors.New("installed disk kernel identity changed")
	}
	return installedRootFromJSON(data, selected, sequence, checkInstalledHolders)
}

func installedRootFromJSON(data []byte, selected Disk, sequence string, holders func(string) error) (installedRoot, error) {
	var tree struct {
		Devices []installedBlockDevice `json:"blockdevices"`
	}
	if json.Unmarshal(data, &tree) != nil || len(tree.Devices) != 1 {
		return installedRoot{}, errors.New("ambiguous installed disk inventory")
	}
	disk := tree.Devices[0]
	if disk.Name != selected.Device.Name || disk.KName != selected.Device.KName || disk.Type != "disk" || disk.Size != selected.Device.Size || disk.Model != selected.Device.Model || disk.Serial != selected.Device.Serial || disk.WWN != selected.Device.WWN || disk.MajorMinor != selected.Device.MajorMinor || disk.ReadOnly || !validInstalledDevice(disk.Name, disk.KName, disk.MajorMinor) {
		return installedRoot{}, errors.New("installed disk identity changed")
	}
	if sequence != selected.Sequence {
		return installedRoot{}, errors.New("installed disk kernel identity changed")
	}
	if err := holders(disk.Name); err != nil {
		return installedRoot{}, err
	}
	var roots []installedBlockDevice
	for _, child := range disk.Children {
		if child.Type != "part" || child.PKName != disk.Name || !validInstalledDevice(child.Name, child.KName, child.MajorMinor) || child.ReadOnly || mounted(child.Mountpoints) || len(child.Children) != 0 {
			return installedRoot{}, errors.New("unsupported installed disk partition inventory")
		}
		if err := holders(child.Name); err != nil {
			return installedRoot{}, err
		}
		if child.FSType == "xfs" && child.Label == "root" && child.PartLabel == "root" {
			roots = append(roots, child)
		}
	}
	if len(roots) != 1 {
		return installedRoot{}, errors.New("exact installed CoreOS root partition required")
	}
	root := roots[0]
	if root.ReadOnly || root.UUID == "" || root.PartUUID == "" || len(root.Children) != 0 || mounted(root.Mountpoints) {
		return installedRoot{}, errors.New("installed CoreOS root is unavailable")
	}
	return installedRoot{Device: root.Name, MajorMinor: root.MajorMinor}, nil
}

func validInstalledDevice(name, kname, majorMinor string) bool {
	return regexp.MustCompile(`^/dev/[a-zA-Z0-9_-]+$`).MatchString(name) && kname == name && regexp.MustCompile(`^[0-9]+:[0-9]+$`).MatchString(majorMinor)
}

func mounted(points []*string) bool {
	for _, point := range points {
		if point != nil && *point != "" {
			return true
		}
	}
	return false
}

func checkInstalledHolders(device string) error {
	entries, err := os.ReadDir(filepath.Join("/sys/class/block", filepath.Base(device), "holders"))
	if err != nil || len(entries) != 0 {
		return errors.New("installed CoreOS root has holders or cannot be inspected")
	}
	return nil
}

type payloadCopyOps struct {
	source      string
	discover    func(context.Context, Disk, commandRunner) (installedRoot, error)
	mount       func(installedRoot) (string, func() error, error)
	recheck     func(context.Context, Disk, installedRoot, string, commandRunner) error
	physicalVar func(string) (string, string, error)
	prepare     func(string) (string, error)
	available   func(string) (uint64, error)
	bundle      func(string, string, string, string) error
	verify      func(string, string, string) (nativebuild.Inventory, error)
	label       func(context.Context, string, string, commandRunner) error
	sync        func(string) error
	requirement func(string, mediaIdentity) (uint64, error)
}

func copyInstalledPayload(ctx context.Context, selected Disk, media mediaIdentity, measured uint64, run commandRunner) error {
	ops := payloadCopyOps{
		source:      filepath.Join(mediaPayloadRoot, media.Architecture),
		discover:    discoverInstalledRoot,
		mount:       mountInstalledRoot,
		recheck:     recheckInstalledRoot,
		physicalVar: coreOSPhysicalVar,
		prepare:     preparePayloadState,
		available: func(path string) (uint64, error) {
			var stat unix.Statfs_t
			if err := unix.Statfs(path, &stat); err != nil || stat.Bsize <= 0 || stat.Bavail > ^uint64(0)/uint64(stat.Bsize) {
				return 0, errors.New("cannot measure installed root free space")
			}
			return stat.Bavail * uint64(stat.Bsize), nil
		},
		bundle: nativebuild.Bundle,
		verify: verifyTrustedBundle,
		label: func(ctx context.Context, stateroot, state string, run commandRunner) error {
			// /var/lib can legitimately be absent until first boot. Relabel the
			// fixed directory and its new child as logical /var/lib below the
			// physical OSTree stateroot. The validated offline ISO carries the
			// same release policy as the installed image.
			_, err := run(ctx, "setfiles", []string{"-F", "-r", stateroot, "/etc/selinux/targeted/contexts/files/file_contexts", filepath.Dir(state)}, nil)
			if err != nil {
				return errors.New("cannot label installed payload")
			}
			return nil
		},
		sync: syncFilesystem,
		requirement: func(source string, media mediaIdentity) (uint64, error) {
			return payloadRequirement(media)
		},
	}
	return copyInstalledPayloadWith(ctx, selected, media, measured, run, ops)
}

func copyInstalledPayloadWith(ctx context.Context, selected Disk, media mediaIdentity, measured uint64, run commandRunner, ops payloadCopyOps) (result error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := ops.requirement(ops.source, media)
	if err != nil || current != measured {
		return errors.New("verified media payload changed after disk installation")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := ops.discover(ctx, selected, run)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	mountpoint, cleanup, err := ops.mount(root)
	if err != nil {
		return err
	}
	state := ""
	completion := ""
	defer func() {
		if result != nil && completion != "" {
			result = errors.Join(result, invalidatePayloadCompletion(completion, state, mountpoint))
			completion = ""
		}
		cleanupErr := cleanup()
		if cleanupErr != nil && completion != "" {
			cleanupErr = errors.Join(cleanupErr, invalidatePayloadCompletion(completion, state, mountpoint))
		}
		result = errors.Join(result, cleanupErr)
	}()
	if err := ops.recheck(ctx, selected, root, mountpoint, run); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	stateroot, physicalVar, err := ops.physicalVar(mountpoint)
	if err != nil {
		return err
	}
	available, err := ops.available(physicalVar)
	if err != nil || measured > ^uint64(0)-payloadFreeReserve || available < measured+payloadFreeReserve {
		return errors.New("installed CoreOS root lacks payload space plus required reserve; disk layout was not changed")
	}
	state, err = ops.prepare(physicalVar)
	if err != nil {
		return err
	}
	started := payloadReceipt{Schema: payloadReceiptSchema, Release: media.Release,
		Architecture: media.Architecture, Revision: media.Revision,
		BundleSHA256: media.BundleSHA256, Bytes: measured}
	if err := writePayloadReceipt(filepath.Join(state, "media-copy-started.json"), started); err != nil {
		return err
	}
	bundle := filepath.Join(state, "bundle", media.Architecture)
	if err := os.Mkdir(filepath.Dir(bundle), 0700); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ops.bundle(ops.source, bundle, media.Architecture, media.Revision); err != nil {
		return errors.New("verified media payload copy failed; partial state was preserved")
	}
	if _, err := ops.verify(bundle, media.BundleSHA256, media.Architecture); err != nil {
		return errors.New("installed payload verification failed; partial state was preserved")
	}
	if err := ops.label(ctx, stateroot, state, run); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := syncDirectories(bundle, filepath.Dir(bundle), state, filepath.Dir(state), physicalVar, stateroot); err != nil {
		return err
	}
	if err := ops.sync(mountpoint); err != nil {
		return errors.New("cannot persist installed payload data")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ready := filepath.Join(state, "media-copy-ready.json")
	if err := writePayloadReceipt(ready, started); err != nil {
		return err
	}
	if err := syncDirectories(state); err != nil {
		return err
	}
	if err := ops.sync(mountpoint); err != nil {
		return errors.New("cannot persist ready installed payload receipt")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	completion = filepath.Join(state, "media-copy.json")
	if err := os.Rename(ready, completion); err != nil {
		completion = ""
		return errors.New("cannot publish installed payload completion")
	}
	if err := syncDirectories(state); err != nil {
		return err
	}
	if err := ops.sync(mountpoint); err != nil {
		return errors.New("cannot persist installed payload completion")
	}
	return nil
}

func invalidatePayloadCompletion(completion, state, mountpoint string) error {
	invalid := filepath.Join(state, "media-copy-incomplete.json")
	if err := os.Rename(completion, invalid); err != nil {
		return errors.New("cannot invalidate failed installed payload completion")
	}
	if err := syncDirectories(state); err != nil {
		return err
	}
	if err := syncFilesystem(mountpoint); err != nil {
		return errors.New("cannot persist failed installed payload state")
	}
	return nil
}

func recheckInstalledRoot(ctx context.Context, selected Disk, root installedRoot, mountpoint string, run commandRunner) error {
	data, err := run(ctx, "lsblk", []string{"--json", "--bytes", "--paths", "--output", "NAME,KNAME,PKNAME,TYPE,SIZE,MODEL,SERIAL,WWN,MAJ:MIN,RO,MOUNTPOINTS,FSTYPE,LABEL,PARTLABEL,UUID,PARTUUID", selected.Device.Name}, nil)
	if err != nil {
		return errors.New("cannot recheck installed disk after mounting its root")
	}
	sequence, err := diskSequence(selected.Device)
	if err != nil {
		return errors.New("installed disk kernel identity changed while mounting its root")
	}
	return recheckInstalledRootFromJSON(data, selected, root, mountpoint, sequence)
}

func recheckInstalledRootFromJSON(data []byte, selected Disk, root installedRoot, mountpoint, sequence string) error {
	var tree struct {
		Devices []installedBlockDevice `json:"blockdevices"`
	}
	if json.Unmarshal(data, &tree) != nil || len(tree.Devices) != 1 {
		return errors.New("ambiguous installed disk inventory after mount")
	}
	disk := tree.Devices[0]
	if disk.Name != selected.Device.Name || disk.KName != selected.Device.KName || disk.Type != "disk" || disk.Size != selected.Device.Size || disk.Model != selected.Device.Model || disk.Serial != selected.Device.Serial || disk.WWN != selected.Device.WWN || disk.MajorMinor != selected.Device.MajorMinor || disk.ReadOnly || !validInstalledDevice(disk.Name, disk.KName, disk.MajorMinor) {
		return errors.New("installed disk identity changed while mounting its root")
	}
	if sequence != selected.Sequence {
		return errors.New("installed disk kernel identity changed while mounting its root")
	}
	matches := 0
	for _, child := range disk.Children {
		if child.Type != "part" || child.PKName != disk.Name || !validInstalledDevice(child.Name, child.KName, child.MajorMinor) || child.ReadOnly || len(child.Children) != 0 {
			return errors.New("installed disk partition inventory changed while root was mounted")
		}
		if child.Name != root.Device {
			if mounted(child.Mountpoints) {
				return errors.New("another installed disk partition became mounted")
			}
			continue
		}
		matches++
		if child.MajorMinor != root.MajorMinor || child.KName != child.Name || child.PKName != disk.Name || child.Type != "part" || child.ReadOnly || child.FSType != "xfs" || child.Label != "root" || child.PartLabel != "root" || child.UUID == "" || child.PartUUID == "" || len(child.Children) != 0 || !onlyMountedAt(child.Mountpoints, mountpoint) {
			return errors.New("mounted CoreOS root identity changed")
		}
	}
	if matches != 1 {
		return errors.New("mounted CoreOS root disappeared or became ambiguous")
	}
	return nil
}

func onlyMountedAt(points []*string, expected string) bool {
	found := 0
	for _, point := range points {
		if point == nil || *point == "" {
			continue
		}
		if *point != expected {
			return false
		}
		found++
	}
	return found == 1
}

func mountInstalledRoot(root installedRoot) (string, func() error, error) {
	mountpoint, err := os.MkdirTemp("/run", "soda-installed-root-")
	if err != nil {
		return "", nil, err
	}
	if err := unix.Mount(root.Device, mountpoint, "xfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
		_ = os.Remove(mountpoint)
		return "", nil, errors.New("cannot mount installed CoreOS root")
	}
	if err := verifyMountedRoot(root, mountpoint); err != nil {
		if unmountErr := unix.Unmount(mountpoint, 0); unmountErr == nil {
			_ = os.Remove(mountpoint)
		}
		return "", nil, err
	}
	cleanup := func() error {
		if err := unix.Unmount(mountpoint, 0); err != nil {
			return errors.New("cannot unmount installed CoreOS root; do not remove installation media")
		}
		// The disk is safely unmounted. A failure to remove only this empty,
		// run-owned directory does not invalidate the persisted payload.
		_ = os.Remove(mountpoint)
		return nil
	}
	return mountpoint, cleanup, nil
}

func verifyMountedRoot(root installedRoot, mountpoint string) error {
	parts := strings.Split(root.MajorMinor, ":")
	if len(parts) != 2 {
		return errors.New("invalid installed root device identity")
	}
	major, majorErr := strconv.ParseUint(parts[0], 10, 32)
	minor, minorErr := strconv.ParseUint(parts[1], 10, 32)
	var source, target unix.Stat_t
	if majorErr != nil || minorErr != nil || unix.Stat(root.Device, &source) != nil || source.Mode&unix.S_IFMT != unix.S_IFBLK || unix.Stat(mountpoint, &target) != nil || unix.Major(uint64(source.Rdev)) != uint32(major) || unix.Minor(uint64(source.Rdev)) != uint32(minor) || uint64(target.Dev) != uint64(source.Rdev) {
		return errors.New("mounted CoreOS root identity mismatch")
	}
	return nil
}

func coreOSPhysicalVar(root string) (string, string, error) {
	deploy := filepath.Join(root, "ostree/deploy")
	entries, err := os.ReadDir(deploy)
	if err != nil || len(entries) != 1 {
		return "", "", errors.New("exactly one installed OSTree stateroot required")
	}
	stateroot := filepath.Join(deploy, entries[0].Name())
	physicalVar := filepath.Join(stateroot, "var")
	for _, path := range []string{root, filepath.Join(root, "ostree"), filepath.Join(root, "ostree/repo"), deploy, stateroot, physicalVar} {
		info, err := os.Lstat(path)
		stat, ok := infoSys(info)
		if err != nil || !ok || !info.IsDir() || info.Mode().Perm()&0022 != 0 || stat.Uid != 0 {
			return "", "", errors.New("real protected installed OSTree /var required")
		}
	}
	return stateroot, physicalVar, nil
}

func preparePayloadState(physicalVar string) (string, error) {
	lib := filepath.Join(physicalVar, "lib")
	info, err := os.Lstat(lib)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(lib, 0755); err != nil {
			return "", errors.New("cannot create installed /var/lib")
		}
		info, err = os.Lstat(lib)
	}
	stat, ok := infoSys(info)
	if err != nil || !ok || !info.IsDir() || info.Mode().Perm()&0022 != 0 || stat.Uid != 0 {
		return "", errors.New("real protected installed /var/lib required")
	}
	state := filepath.Join(lib, "soda-installer")
	if err := os.Mkdir(state, 0700); err != nil {
		return "", errors.New("installed payload state already exists or cannot be created")
	}
	return state, nil
}

func infoSys(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
}

func writePayloadReceipt(path string, receipt payloadReceipt) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	return errors.Join(writeErr, file.Sync(), file.Close())
}

func syncDirectories(paths ...string) error {
	for _, path := range paths {
		directory, err := os.Open(path)
		if err != nil {
			return err
		}
		err = errors.Join(directory.Sync(), directory.Close())
		if err != nil {
			return err
		}
	}
	return nil
}

func syncFilesystem(root string) error {
	directory, err := os.Open(root)
	if err != nil {
		return err
	}
	defer directory.Close()
	return unix.Syncfs(int(directory.Fd()))
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

func installedPayloadAtWith(root, arch, release string, verify func(string, string, string) (nativebuild.Inventory, error), protect func(string) error) (bundle, digest, revision string, err error) {
	if err := protect(root); err != nil {
		return "", "", "", errors.New("protected installed payload state required")
	}
	for _, name := range []string{"media-copy-ready.json", "media-copy-incomplete.json"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			return "", "", "", errors.New("incomplete installer media-copy state requires operator inspection")
		}
	}
	receiptPath := filepath.Join(root, "media-copy.json")
	info, err := os.Lstat(receiptPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return "", "", "", errors.New("private regular installer media-copy receipt required")
	}
	data, err := readRegular(receiptPath, 4096)
	if err != nil {
		return "", "", "", errors.New("completed installer media-copy receipt required")
	}
	var receipt payloadReceipt
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return "", "", "", errors.New("completed installer media-copy receipt required")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", "", "", errors.New("completed installer media-copy receipt required")
	}
	if receipt.Schema != payloadReceiptSchema || receipt.Architecture != arch || receipt.Release != release || receipt.Bytes == 0 || !nativebuild.Digest(receipt.BundleSHA256) || !nativebuild.Revision(receipt.Revision) {
		return "", "", "", errors.New("invalid installer media-copy receipt")
	}
	bundle = filepath.Join(root, "bundle", arch)
	inventory, err := verify(bundle, receipt.BundleSHA256, arch)
	if err != nil || inventory.Architecture != arch || inventory.Revision != receipt.Revision {
		return "", "", "", errors.New("installed payload does not match its completed receipt")
	}
	return bundle, receipt.BundleSHA256, receipt.Revision, nil
}
