package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

func payloadMedia() mediaIdentity {
	return mediaIdentity{
		Architecture:     architecture(),
		Release:          "42.20260901.3.0",
		InstallerVersion: "coreos-installer 0.26.0",
		Revision:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		BundleSHA256:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
}

func payloadInventory(media mediaIdentity) nativebuild.Inventory {
	return nativebuild.Inventory{Revision: media.Revision, Architecture: media.Architecture}
}

func TestPayloadRequirementMeasuresVerifiedRegularBytes(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "one"), []byte("123"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested/two"), []byte("4567"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nested/two", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	media := payloadMedia()
	verify := func(path, digest, arch string) (nativebuild.Inventory, error) {
		if path != root || digest != media.BundleSHA256 || arch != media.Architecture {
			t.Fatal("wrong verification identity")
		}
		return payloadInventory(media), nil
	}
	got, err := payloadRequirementAt(root, media, verify)
	if err != nil || got != 7 {
		t.Fatalf("requirement = %d, %v", got, err)
	}
	media.Revision = "short"
	if _, err := payloadRequirementAt(root, media, verify); err == nil {
		t.Fatal("invalid media revision accepted")
	}
}

func installedDiskFixture(t *testing.T, mountpoint string) (Disk, []byte, installedRoot) {
	t.Helper()
	selected := Disk{Device: BlockDevice{Name: "/dev/vda", KName: "/dev/vda", Type: "disk", Size: 10 << 30,
		Model: "fixture", Serial: "serial", WWN: "wwn", MajorMinor: "252:0"}, Sequence: "17"}
	root := installedBlockDevice{Name: "/dev/vda4", KName: "/dev/vda4", PKName: "/dev/vda", Type: "part",
		Size: 8 << 30, MajorMinor: "252:4", Mountpoints: []*string{}, FSType: "xfs", Label: "root",
		PartLabel: "root", UUID: "filesystem", PartUUID: "partition"}
	if mountpoint != "" {
		root.Mountpoints = []*string{&mountpoint}
	}
	disk := installedBlockDevice{Name: "/dev/vda", KName: "/dev/vda", Type: "disk", Size: selected.Device.Size,
		Model: selected.Device.Model, Serial: selected.Device.Serial, WWN: selected.Device.WWN,
		MajorMinor: selected.Device.MajorMinor, Mountpoints: []*string{}, Children: []installedBlockDevice{
			{Name: "/dev/vda3", KName: "/dev/vda3", PKName: "/dev/vda", Type: "part", Size: 1 << 30,
				MajorMinor: "252:3", Mountpoints: []*string{}, FSType: "ext4", Label: "boot", PartLabel: "boot"},
			root,
		}}
	data, err := json.Marshal(struct {
		Devices []installedBlockDevice `json:"blockdevices"`
	}{[]installedBlockDevice{disk}})
	if err != nil {
		t.Fatal(err)
	}
	return selected, data, installedRoot{Device: root.Name, MajorMinor: root.MajorMinor}
}

func TestInstalledRootDiscoveryAndExactPostMountRecheck(t *testing.T) {
	selected, before, expected := installedDiskFixture(t, "")
	holders := []string{}
	root, err := installedRootFromJSON(before, selected, selected.Sequence, func(name string) error {
		holders = append(holders, name)
		return nil
	})
	if err != nil || root != expected {
		t.Fatalf("discovery = %#v, %v", root, err)
	}
	if len(holders) != 3 || holders[0] != "/dev/vda" || holders[1] != "/dev/vda3" || holders[2] != expected.Device {
		t.Fatalf("holders targets = %#v", holders)
	}
	mountpoint := "/run/soda-installed-root-fixture"
	_, after, _ := installedDiskFixture(t, mountpoint)
	if err := recheckInstalledRootFromJSON(after, selected, expected, mountpoint, selected.Sequence); err != nil {
		t.Fatal(err)
	}
	other := "/unexpected"
	var tree struct {
		Devices []installedBlockDevice `json:"blockdevices"`
	}
	if err := json.Unmarshal(after, &tree); err != nil {
		t.Fatal(err)
	}
	tree.Devices[0].Children[1].Mountpoints = []*string{&mountpoint, &other}
	tampered, _ := json.Marshal(tree)
	if err := recheckInstalledRootFromJSON(tampered, selected, expected, mountpoint, selected.Sequence); err == nil {
		t.Fatal("additional root mount accepted")
	}
	if _, err := installedRootFromJSON(before, selected, "18", func(string) error { return nil }); err == nil {
		t.Fatal("changed disk sequence accepted")
	}
}

func payloadCopyFixture(t *testing.T) (Disk, mediaIdentity, uint64, string, payloadCopyOps, *[]string) {
	t.Helper()
	mountpoint := t.TempDir()
	physicalVar := filepath.Join(mountpoint, "ostree/deploy/fedora-coreos/var")
	if err := os.MkdirAll(filepath.Join(physicalVar, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(mountpoint, "ostree/repo"), 0755); err != nil {
		t.Fatal(err)
	}
	selected, _, root := installedDiskFixture(t, "")
	media := payloadMedia()
	measured := uint64(4096)
	events := []string{}
	ops := payloadCopyOps{
		source: "/synthetic/read-only-media/bundle/" + media.Architecture,
		requirement: func(string, mediaIdentity) (uint64, error) {
			events = append(events, "requirement")
			return measured, nil
		},
		discover: func(context.Context, Disk, commandRunner) (installedRoot, error) {
			events = append(events, "discover")
			return root, nil
		},
		mount: func(installedRoot) (string, func() error, error) {
			events = append(events, "mount")
			return mountpoint, func() error { events = append(events, "unmount"); return nil }, nil
		},
		physicalVar: func(string) (string, string, error) {
			return filepath.Dir(physicalVar), physicalVar, nil
		},
		prepare: func(string) (string, error) {
			state := filepath.Join(physicalVar, "lib/soda-installer")
			return state, os.Mkdir(state, 0700)
		},
		recheck: func(context.Context, Disk, installedRoot, string, commandRunner) error {
			events = append(events, "recheck")
			return nil
		},
		available: func(string) (uint64, error) { return measured + payloadFreeReserve, nil },
		bundle: func(source, destination, arch, revision string) error {
			events = append(events, "bundle")
			if source != opsSource(media) || arch != media.Architecture || revision != media.Revision {
				t.Fatal("wrong bundle copy identity")
			}
			return os.Mkdir(destination, 0700)
		},
		verify: func(string, string, string) (nativebuild.Inventory, error) {
			events = append(events, "verify")
			return payloadInventory(media), nil
		},
		label: func(context.Context, string, string, commandRunner) error {
			events = append(events, "label")
			return nil
		},
		sync: func(string) error {
			state := filepath.Join(physicalVar, "lib/soda-installer")
			if len(events) > 0 && events[len(events)-1] == "label" {
				if _, err := os.Lstat(filepath.Join(state, "media-copy.json")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("completion receipt existed before payload data sync")
				}
			}
			events = append(events, "sync")
			return nil
		},
	}
	return selected, media, measured, physicalVar, ops, &events
}

func opsSource(media mediaIdentity) string {
	return "/synthetic/read-only-media/bundle/" + media.Architecture
}

func TestCopyInstalledPayloadWritesCompletionAfterDataSync(t *testing.T) {
	selected, media, measured, physicalVar, ops, events := payloadCopyFixture(t)
	if err := copyInstalledPayloadWith(context.Background(), selected, media, measured, nil, ops); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(physicalVar, "lib/soda-installer")
	if _, err := os.Stat(filepath.Join(state, "media-copy-started.json")); err != nil {
		t.Fatal(err)
	}
	bundle, digest, revision, err := installedPayloadAtWith(state, media.Architecture, media.Release, ops.verify, func(string) error { return nil })
	if err != nil || bundle != filepath.Join(state, "bundle", media.Architecture) || digest != media.BundleSHA256 || revision != media.Revision {
		t.Fatalf("installed payload = %q, %q, %q, %v", bundle, digest, revision, err)
	}
	if _, _, _, err := installedPayloadAtWith(state, media.Architecture, "different-release", ops.verify, func(string) error { return nil }); err == nil {
		t.Fatal("receipt for another installed release accepted")
	}
	want := []string{"requirement", "discover", "mount", "recheck", "bundle", "verify", "label", "sync", "sync", "sync", "unmount", "verify"}
	if len(*events) != len(want) {
		t.Fatalf("events = %#v", *events)
	}
	for index := range want {
		if (*events)[index] != want[index] {
			t.Fatalf("events = %#v", *events)
		}
	}
}

func TestCopyInstalledPayloadRefusesSpaceAndPreservesIncompleteState(t *testing.T) {
	selected, media, measured, physicalVar, ops, _ := payloadCopyFixture(t)
	ops.available = func(string) (uint64, error) { return measured + payloadFreeReserve - 1, nil }
	if err := copyInstalledPayloadWith(context.Background(), selected, media, measured, nil, ops); err == nil {
		t.Fatal("insufficient reserve accepted")
	}
	if _, err := os.Lstat(filepath.Join(physicalVar, "lib/soda-installer")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("state was written despite failed space preflight")
	}

	selected, media, measured, physicalVar, ops, _ = payloadCopyFixture(t)
	ops.sync = func(string) error { return errors.New("synthetic sync failure") }
	if err := copyInstalledPayloadWith(context.Background(), selected, media, measured, nil, ops); err == nil {
		t.Fatal("sync failure accepted")
	}
	state := filepath.Join(physicalVar, "lib/soda-installer")
	if _, err := os.Stat(filepath.Join(state, "media-copy-started.json")); err != nil {
		t.Fatal("start receipt not preserved")
	}
	if _, err := os.Lstat(filepath.Join(state, "media-copy.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("completion receipt published after failed payload sync")
	}
}

func TestCopyInstalledPayloadInvalidatesCompletionWhenUnmountFails(t *testing.T) {
	selected, media, measured, physicalVar, ops, _ := payloadCopyFixture(t)
	mountpoint := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(physicalVar))))
	ops.mount = func(installedRoot) (string, func() error, error) {
		return mountpoint, func() error { return errors.New("synthetic unmount failure") }, nil
	}
	if err := copyInstalledPayloadWith(context.Background(), selected, media, measured, nil, ops); err == nil {
		t.Fatal("unmount failure accepted")
	}
	state := filepath.Join(physicalVar, "lib/soda-installer")
	if _, err := os.Lstat(filepath.Join(state, "media-copy.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("completion receipt remained after unmount failure")
	}
	if _, err := os.Stat(filepath.Join(state, "media-copy-incomplete.json")); err != nil {
		t.Fatal("invalidated completion evidence missing")
	}
}

func TestCopyInstalledPayloadInvalidatesCompletionAfterPublicationError(t *testing.T) {
	selected, media, measured, physicalVar, ops, _ := payloadCopyFixture(t)
	syncs := 0
	ops.sync = func(string) error {
		syncs++
		if syncs == 3 {
			return errors.New("synthetic final sync failure")
		}
		return nil
	}
	if err := copyInstalledPayloadWith(context.Background(), selected, media, measured, nil, ops); err == nil {
		t.Fatal("post-publication sync failure accepted")
	}
	state := filepath.Join(physicalVar, "lib/soda-installer")
	if _, err := os.Lstat(filepath.Join(state, "media-copy.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("completion receipt remained after final sync failure")
	}
	if _, err := os.Stat(filepath.Join(state, "media-copy-incomplete.json")); err != nil {
		t.Fatal("invalidated completion evidence missing")
	}
}

func TestCopyInstalledPayloadStopsAfterCancellationBoundary(t *testing.T) {
	selected, media, measured, _, ops, events := payloadCopyFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	ops.requirement = func(string, mediaIdentity) (uint64, error) {
		*events = append(*events, "requirement")
		cancel()
		return measured, nil
	}
	if err := copyInstalledPayloadWith(ctx, selected, media, measured, nil, ops); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
	if len(*events) != 1 || (*events)[0] != "requirement" {
		t.Fatalf("native work continued after cancellation: %#v", *events)
	}
}
