package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

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

func parseInitialInstalledDisk(data []byte) (installedBlockDevice, error) {
	var tree struct {
		Devices []installedBlockDevice `json:"blockdevices"`
	}
	if json.Unmarshal(data, &tree) != nil || len(tree.Devices) != 1 {
		return installedBlockDevice{}, errors.New("ambiguous installed disk inventory")
	}
	return tree.Devices[0], nil
}

func verifyInitialInstalledDisk(disk installedBlockDevice, selected Disk, sequence string, holders func(string) error) error {
	if !sameInstalledDevice(disk, selected.Device) {
		return errors.New("installed disk identity changed")
	}
	if sequence != selected.Sequence {
		return errors.New("installed disk kernel identity changed")
	}
	return holders(disk.Name)
}

func validInitialPartition(child installedBlockDevice, diskName string) bool {
	if child.Type != "part" || child.PKName != diskName || len(child.Children) != 0 {
		return false
	}
	return validInstalledDevice(child.Name, child.KName, child.MajorMinor) && !child.ReadOnly && !mounted(child.Mountpoints)
}

func isCoreOSRootPartition(child installedBlockDevice) bool {
	return child.FSType == "xfs" && child.Label == "root" && child.PartLabel == "root"
}

func validCoreOSRootPartition(root installedBlockDevice) bool {
	if root.ReadOnly || root.UUID == "" || root.PartUUID == "" || len(root.Children) != 0 {
		return false
	}
	return !mounted(root.Mountpoints)
}

func findInstalledRootPartition(disk installedBlockDevice, holders func(string) error) (installedRoot, error) {
	var roots []installedBlockDevice
	for _, child := range disk.Children {
		if !validInitialPartition(child, disk.Name) {
			return installedRoot{}, errors.New("unsupported installed disk partition inventory")
		}
		if err := holders(child.Name); err != nil {
			return installedRoot{}, err
		}
		if isCoreOSRootPartition(child) {
			roots = append(roots, child)
		}
	}
	if len(roots) != 1 {
		return installedRoot{}, errors.New("exact installed CoreOS root partition required")
	}
	root := roots[0]
	if !validCoreOSRootPartition(root) {
		return installedRoot{}, errors.New("installed CoreOS root is unavailable")
	}
	return installedRoot{Device: root.Name, MajorMinor: root.MajorMinor}, nil
}

func installedRootFromJSON(data []byte, selected Disk, sequence string, holders func(string) error) (installedRoot, error) {
	disk, err := parseInitialInstalledDisk(data)
	if err != nil {
		return installedRoot{}, err
	}
	if err := verifyInitialInstalledDisk(disk, selected, sequence, holders); err != nil {
		return installedRoot{}, err
	}
	return findInstalledRootPartition(disk, holders)
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

func parseInstalledDiskInventory(data []byte) (installedBlockDevice, error) {
	var tree struct {
		Devices []installedBlockDevice `json:"blockdevices"`
	}
	if json.Unmarshal(data, &tree) != nil || len(tree.Devices) != 1 {
		return installedBlockDevice{}, errors.New("ambiguous installed disk inventory after mount")
	}
	return tree.Devices[0], nil
}

func sameInstalledDiskMetadata(disk installedBlockDevice, dev BlockDevice) bool {
	if disk.Name != dev.Name || disk.KName != dev.KName || disk.Type != "disk" || disk.Size != dev.Size {
		return false
	}
	return disk.MajorMinor == dev.MajorMinor
}

func sameInstalledDiskIdentity(disk installedBlockDevice, dev BlockDevice) bool {
	if disk.Model != dev.Model || disk.Serial != dev.Serial || disk.WWN != dev.WWN {
		return false
	}
	return !disk.ReadOnly && validInstalledDevice(disk.Name, disk.KName, disk.MajorMinor)
}

func sameInstalledDevice(disk installedBlockDevice, dev BlockDevice) bool {
	return sameInstalledDiskMetadata(disk, dev) && sameInstalledDiskIdentity(disk, dev)
}

func verifyInstalledDiskIdentity(disk installedBlockDevice, selected Disk, sequence string) error {
	if !sameInstalledDevice(disk, selected.Device) {
		return errors.New("installed disk identity changed while mounting its root")
	}
	if sequence != selected.Sequence {
		return errors.New("installed disk kernel identity changed while mounting its root")
	}
	return nil
}

func verifyInstalledPartitionShape(child installedBlockDevice, diskName string) error {
	if child.Type != "part" || child.PKName != diskName || len(child.Children) != 0 {
		return errors.New("installed disk partition inventory changed while root was mounted")
	}
	if child.ReadOnly || !validInstalledDevice(child.Name, child.KName, child.MajorMinor) {
		return errors.New("installed disk partition inventory changed while root was mounted")
	}
	return nil
}

func sameMountedRootAttributes(child installedBlockDevice, root installedRoot, diskName string) bool {
	if child.MajorMinor != root.MajorMinor || child.KName != child.Name || child.PKName != diskName {
		return false
	}
	return child.Type == "part" && !child.ReadOnly && child.FSType == "xfs"
}

func verifyMountedRootPartition(child installedBlockDevice, root installedRoot, diskName, mountpoint string) error {
	if !sameMountedRootAttributes(child, root, diskName) {
		return errors.New("mounted CoreOS root identity changed")
	}
	if child.Label != "root" || child.PartLabel != "root" || child.UUID == "" || child.PartUUID == "" {
		return errors.New("mounted CoreOS root identity changed")
	}
	if len(child.Children) != 0 || !onlyMountedAt(child.Mountpoints, mountpoint) {
		return errors.New("mounted CoreOS root identity changed")
	}
	return nil
}

func verifyInstalledPartitions(disk installedBlockDevice, root installedRoot, mountpoint string) error {
	matches := 0
	for _, child := range disk.Children {
		if err := verifyInstalledPartitionShape(child, disk.Name); err != nil {
			return err
		}
		if child.Name != root.Device {
			if mounted(child.Mountpoints) {
				return errors.New("another installed disk partition became mounted")
			}
			continue
		}
		matches++
		if err := verifyMountedRootPartition(child, root, disk.Name, mountpoint); err != nil {
			return err
		}
	}
	if matches != 1 {
		return errors.New("mounted CoreOS root disappeared or became ambiguous")
	}
	return nil
}

func recheckInstalledRootFromJSON(data []byte, selected Disk, root installedRoot, mountpoint, sequence string) error {
	disk, err := parseInstalledDiskInventory(data)
	if err != nil {
		return err
	}
	if err := verifyInstalledDiskIdentity(disk, selected, sequence); err != nil {
		return err
	}
	return verifyInstalledPartitions(disk, root, mountpoint)
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

func mountedRootMatches(root installedRoot, mountpoint string) bool {
	parts := strings.Split(root.MajorMinor, ":")
	if len(parts) != 2 {
		return false
	}
	major, majorErr := strconv.ParseUint(parts[0], 10, 32)
	minor, minorErr := strconv.ParseUint(parts[1], 10, 32)
	var source, target unix.Stat_t
	return majorErr == nil && minorErr == nil && unix.Stat(root.Device, &source) == nil && source.Mode&unix.S_IFMT == unix.S_IFBLK && unix.Stat(mountpoint, &target) == nil && unix.Major(uint64(source.Rdev)) == uint32(major) && unix.Minor(uint64(source.Rdev)) == uint32(minor) && uint64(target.Dev) == uint64(source.Rdev)
}

func verifyMountedRoot(root installedRoot, mountpoint string) error {
	if !mountedRootMatches(root, mountpoint) {
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
		if err != nil || !ok || !info.IsDir() || info.Mode().Perm()&0o022 != 0 || stat.Uid != 0 {
			return "", "", errors.New("real protected installed OSTree /var required")
		}
	}
	return stateroot, physicalVar, nil
}

func preparePayloadState(physicalVar string) (string, error) {
	lib := filepath.Join(physicalVar, "lib")
	info, err := os.Lstat(lib)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(lib, 0o755); err != nil {
			return "", errors.New("cannot create installed /var/lib")
		}
		info, err = os.Lstat(lib)
	}
	stat, ok := infoSys(info)
	if err != nil || !ok || !info.IsDir() || info.Mode().Perm()&0o022 != 0 || stat.Uid != 0 {
		return "", errors.New("real protected installed /var/lib required")
	}
	state := filepath.Join(lib, "soda-installer")
	if err := os.Mkdir(state, 0o700); err != nil {
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
