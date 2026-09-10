package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

type BlockDevice struct {
	Name        string        `json:"name"`
	KName       string        `json:"kname"`
	Type        string        `json:"type"`
	Size        uint64        `json:"size"`
	Model       string        `json:"model"`
	Serial      string        `json:"serial"`
	WWN         string        `json:"wwn"`
	MajorMinor  string        `json:"maj:min"`
	ReadOnly    bool          `json:"ro"`
	Mountpoints []*string     `json:"mountpoints"`
	FSType      string        `json:"fstype"`
	UUID        string        `json:"uuid"`
	PartUUID    string        `json:"partuuid"`
	Children    []BlockDevice `json:"children"`
}

type Disk struct {
	Device   BlockDevice
	Sequence string
	Blocked  string
}

func deviceNames(d BlockDevice) bool {
	return regexp.MustCompile(`^/dev/[a-zA-Z0-9_-]+$`).MatchString(d.Name) && d.KName == d.Name && regexp.MustCompile(`^[0-9]+:[0-9]+$`).MatchString(d.MajorMinor)
}

func unused(d BlockDevice) string {
	if !deviceNames(d) || d.Size == 0 || d.Mountpoints == nil {
		return "incomplete or unsupported device inventory"
	}
	if d.ReadOnly {
		return "read-only device"
	}
	for _, mount := range d.Mountpoints {
		if mount != nil && *mount != "" {
			return "mounted filesystem or active swap"
		}
	}
	switch d.FSType {
	case "iso9660", "udf":
		return "installation/optical media"
	case "", "ext2", "ext3", "ext4", "xfs", "vfat", "exfat", "ntfs", "swap":
		// Ordinary single-device filesystems still require all use checks below.
	default:
		return "unrecognized, multi-device or encrypted storage requires separate operator handling"
	}
	for _, child := range d.Children {
		if child.Type != "part" {
			return "active mapped/stacked device"
		}
		if reason := unused(child); reason != "" {
			return reason
		}
	}
	return ""
}

func scanDisks(ctx context.Context, run commandRunner) ([]Disk, error) {
	data, err := run(ctx, "lsblk", []string{"--json", "--bytes", "--paths", "--output", "NAME,KNAME,TYPE,SIZE,MODEL,SERIAL,WWN,MAJ:MIN,RO,MOUNTPOINTS,FSTYPE,UUID,PARTUUID"}, nil)
	if err != nil {
		return nil, errors.New("cannot inventory block devices")
	}
	var tree struct {
		Devices []BlockDevice `json:"blockdevices"`
	}
	if err = json.Unmarshal(data, &tree); err != nil {
		return nil, errors.New("cannot decode block device inventory")
	}
	var disks []Disk
	for _, device := range tree.Devices {
		if device.Type != "disk" {
			continue
		}
		disk := Disk{Device: device, Blocked: unused(device)}
		if disk.Blocked == "" {
			disk.Sequence, err = diskSequence(device)
			if err != nil {
				disk.Blocked = "cannot establish kernel disk identity"
			}
			if err := checkHolders(device); err != nil {
				disk.Blocked = "device has holders or cannot be inspected"
			}
		}
		disks = append(disks, disk)
	}
	return disks, nil
}

func diskSequence(device BlockDevice) (string, error) {
	if !deviceNames(device) {
		return "", errors.New("invalid kernel device identity")
	}
	data, err := os.ReadFile(filepath.Join("/sys/class/block", filepath.Base(device.Name), "diskseq"))
	if err != nil {
		return "", err
	}
	seq := strings.TrimSpace(string(data))
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(seq) {
		return "", errors.New("invalid disk sequence")
	}
	return seq, nil
}

func checkHolders(device BlockDevice) error {
	entries, err := os.ReadDir(filepath.Join("/sys/class/block", filepath.Base(device.Name), "holders"))
	if err != nil || len(entries) != 0 {
		return errors.New("device has holders or cannot be inspected")
	}
	for _, child := range device.Children {
		if err := checkHolders(child); err != nil {
			return err
		}
	}
	return nil
}

func sameDisk(selected Disk, observed []Disk) error {
	for _, current := range observed {
		if current.Device.Name == selected.Device.Name {
			if selected.Blocked != "" || current.Blocked != "" || selected.Sequence == "" || !reflect.DeepEqual(selected, current) {
				return errors.New("disk identity, partition inventory or use changed; no installation started")
			}
			return nil
		}
	}
	return errors.New("selected disk disappeared; no installation started")
}

func diskSummary(d BlockDevice) string {
	return fmt.Sprintf("%q | %.1f GiB | model %q | serial %q | WWN %q", d.Name, float64(d.Size)/(1<<30), d.Model, d.Serial, d.WWN)
}
