//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const interfaceScript = `set -eu; test ! -L /run; test -d /run; test ! -L /run/soda-muse-interface; if test -e /run/soda-muse-interface; then test -d /run/soda-muse-interface; fi; mkdir -p /run/soda-muse-interface`

func prepareInterface(ctx context.Context, target observation) error {
	return prepareLaunchInterface(ctx, target, "muse")
}

func prepareLaunchInterface(ctx context.Context, target observation, kind string) error {
	if err := confirmProject(ctx, target); err != nil {
		return err
	}
	_, err := podman(ctx, nil, "exec", "--user", "0:0", target.ID, "/bin/sh", "-ceu", strings.ReplaceAll(interfaceScript, "soda-muse-interface", "soda-"+kind+"-interface"))
	return err
}

func publicSocketDirectory(socket string) (string, error) {
	if !filepath.IsAbs(socket) || filepath.Base(socket) != "launch.sock" {
		return "", errors.New("explicit public launch socket required")
	}
	root := filepath.Dir(socket)
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "launch.sock" {
		return "", errors.New("launch directory must contain only the public socket")
	}
	if err := validatePublicSocket(socket); err != nil {
		return "", err
	}
	if err := validateInterfaceDirectory(root); err != nil {
		return "", err
	}
	return root, nil
}

func attachInterface(ctx context.Context, target observation, socket string) error {
	return attachLaunchInterface(ctx, target, socket, "muse")
}

func attachLaunchInterface(ctx context.Context, target observation, socket, kind string) error {
	source, err := publicSocketDirectory(socket)
	if err != nil {
		return err
	}
	sourceFD, err := unix.Open(source, unix.O_PATH|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(sourceFD) }()
	tree, err := unix.OpenTree(sourceFD, "", unix.OPEN_TREE_CLONE|unix.OPEN_TREE_CLOEXEC|unix.AT_EMPTY_PATH)
	if err != nil {
		return fmt.Errorf("clone public interface mount: %w", err)
	}
	defer func() { _ = unix.Close(tree) }()
	attr := unix.MountAttr{Attr_set: unix.MOUNT_ATTR_RDONLY | unix.MOUNT_ATTR_NOSUID | unix.MOUNT_ATTR_NODEV | unix.MOUNT_ATTR_NOEXEC}
	if err := unix.MountSetattr(tree, "", unix.AT_EMPTY_PATH, &attr); err != nil {
		return fmt.Errorf("restrict public interface mount: %w", err)
	}
	return attachProjectMount(ctx, target, tree, kind)
}

func attachProjectMount(ctx context.Context, target observation, tree int, kind string) error {
	root := "/proc/" + strconv.Itoa(target.PID)
	targetFD, err := unix.Open(root+"/root/run/soda-"+kind+"-interface", unix.O_PATH|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(targetFD) }()
	namespace, err := unix.Open(root+"/ns/mnt", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(namespace) }()
	if err := confirmProject(ctx, target); err != nil {
		return err
	}
	return moveProjectMount(namespace, tree, targetFD)
}

func moveProjectMount(namespace, tree, target int) error {
	// This standalone process exits on the locked thread. Enter only the mount
	// namespace: initial user namespace root authority is required for transfer.
	runtime.LockOSThread()
	if err := unix.Unshare(unix.CLONE_FS); err != nil {
		return fmt.Errorf("separate maintenance filesystem context: %w", err)
	}
	if err := unix.Setns(namespace, unix.CLONE_NEWNS); err != nil {
		return fmt.Errorf("enter project mount namespace: %w", err)
	}
	if err := unix.MoveMount(tree, "", target, "", unix.MOVE_MOUNT_F_EMPTY_PATH|unix.MOVE_MOUNT_T_EMPTY_PATH); err != nil {
		return fmt.Errorf("attach public interface mount: %w", err)
	}
	return nil
}

func validatePublicSocket(socket string) error {
	info, err := os.Lstat(socket)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return errors.New("public launch socket is unavailable")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || info.Mode().Perm() != 0o666 {
		return errors.New("public launch socket must be root-owned and public")
	}
	return nil
}

func validateInterfaceDirectory(root string) error {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
		return errors.New("public launch directory must be a protected directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || info.Mode().Perm()&0o055 != 0o055 {
		return errors.New("public launch directory must be root-owned and accessible")
	}
	return nil
}
