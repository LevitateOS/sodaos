package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/levitateos/sodaos/internal/nativebuild"
	"golang.org/x/sys/unix"
)

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

type installedPayloadTarget struct {
	stateroot   string
	physicalVar string
	state       string
	started     payloadReceipt
}

func verifyMediaPayloadRequirement(ctx context.Context, ops payloadCopyOps, media mediaIdentity, measured uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := ops.requirement(ops.source, media)
	if err != nil || current != measured {
		return errors.New("verified media payload changed after disk installation")
	}
	return ctx.Err()
}

func discoverAndMountRoot(ctx context.Context, ops payloadCopyOps, selected Disk, run commandRunner) (installedRoot, string, func() error, error) {
	if err := ctx.Err(); err != nil {
		return installedRoot{}, "", nil, err
	}
	root, err := ops.discover(ctx, selected, run)
	if err != nil {
		return installedRoot{}, "", nil, err
	}
	if err := ctx.Err(); err != nil {
		return installedRoot{}, "", nil, err
	}
	mountpoint, cleanup, err := ops.mount(root)
	if err != nil {
		return installedRoot{}, "", nil, err
	}
	return root, mountpoint, cleanup, nil
}

func prepareInstalledPayloadState(ctx context.Context, ops payloadCopyOps, mountpoint string, media mediaIdentity, measured uint64) (installedPayloadTarget, error) {
	var target installedPayloadTarget
	if err := ctx.Err(); err != nil {
		return target, err
	}
	stateroot, physicalVar, err := ops.physicalVar(mountpoint)
	if err != nil {
		return target, err
	}
	available, err := ops.available(physicalVar)
	if err != nil || measured > ^uint64(0)-payloadFreeReserve || available < measured+payloadFreeReserve {
		return target, errors.New("installed CoreOS root lacks payload space plus required reserve; disk layout was not changed")
	}
	state, err := ops.prepare(physicalVar)
	if err != nil {
		return target, err
	}
	started := payloadReceipt{
		Schema:       payloadReceiptSchema,
		Release:      media.Release,
		Architecture: media.Architecture,
		Revision:     media.Revision,
		BundleSHA256: media.BundleSHA256,
		Bytes:        measured,
	}
	if err := writePayloadReceipt(filepath.Join(state, "media-copy-started.json"), started); err != nil {
		return target, err
	}
	target = installedPayloadTarget{
		stateroot:   stateroot,
		physicalVar: physicalVar,
		state:       state,
		started:     started,
	}
	return target, nil
}

func copyAndVerifyBundle(
	ctx context.Context,
	ops payloadCopyOps,
	target installedPayloadTarget,
	media mediaIdentity,
	run commandRunner,
	mountpoint string,
) error {
	bundle := filepath.Join(target.state, "bundle", media.Architecture)
	if err := os.Mkdir(filepath.Dir(bundle), 0o700); err != nil {
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
	if err := ops.label(ctx, target.stateroot, target.state, run); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := syncDirectories(bundle, filepath.Dir(bundle), target.state, filepath.Dir(target.state), target.physicalVar, target.stateroot); err != nil {
		return err
	}
	if err := ops.sync(mountpoint); err != nil {
		return errors.New("cannot persist installed payload data")
	}
	return ctx.Err()
}

func publishPayloadCompletion(
	ctx context.Context,
	ops payloadCopyOps,
	mountpoint, state string,
	started payloadReceipt,
	completion *string,
) error {
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
	completed := filepath.Join(state, "media-copy.json")
	if err := os.Rename(ready, completed); err != nil {
		*completion = ""
		return errors.New("cannot publish installed payload completion")
	}
	*completion = completed
	if err := syncDirectories(state); err != nil {
		return err
	}
	if err := ops.sync(mountpoint); err != nil {
		return errors.New("cannot persist installed payload completion")
	}
	return nil
}

func finalizeInstalledPayload(result error, completion, state, mountpoint string, cleanup func() error) error {
	if result != nil && completion != "" {
		result = errors.Join(result, invalidatePayloadCompletion(completion, state, mountpoint))
		completion = ""
	}
	cleanupErr := cleanup()
	if cleanupErr != nil && completion != "" {
		cleanupErr = errors.Join(cleanupErr, invalidatePayloadCompletion(completion, state, mountpoint))
	}
	return errors.Join(result, cleanupErr)
}

func copyInstalledPayloadWith(ctx context.Context, selected Disk, media mediaIdentity, measured uint64, run commandRunner, ops payloadCopyOps) (result error) {
	if err := verifyMediaPayloadRequirement(ctx, ops, media, measured); err != nil {
		return err
	}
	root, mountpoint, cleanup, err := discoverAndMountRoot(ctx, ops, selected, run)
	if err != nil {
		return err
	}
	state := ""
	completion := ""
	defer func() {
		result = finalizeInstalledPayload(result, completion, state, mountpoint, cleanup)
	}()
	if err := ops.recheck(ctx, selected, root, mountpoint, run); err != nil {
		return err
	}
	target, err := prepareInstalledPayloadState(ctx, ops, mountpoint, media, measured)
	if err != nil {
		return err
	}
	state = target.state
	if err := copyAndVerifyBundle(ctx, ops, target, media, run, mountpoint); err != nil {
		return err
	}
	return publishPayloadCompletion(ctx, ops, mountpoint, state, target.started, &completion)
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

func writePayloadReceipt(path string, receipt payloadReceipt) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
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
