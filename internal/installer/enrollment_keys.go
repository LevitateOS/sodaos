package installer

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// Existing files are appended through their validated open inode, never replaced.
// A native editor may still change that inode or its pathname concurrently. Such
// changes cause an uncertain result; Soda never restores a snapshot over them.
// Production passes the protected native root home and UID 0. The UID and narrow
// write seam permit synthetic tests of the exact filesystem race/failure boundary.
var errEnrollmentWriteUncertain = errors.New("authorized-key import may have completed; the file changed or the write could not be confirmed; inspect native access before another import")

func appendEnrollmentKey(ctx context.Context, home, key string, uid uint32) error {
	return appendEnrollmentKeyWithWriter(ctx, home, key, uid, func(file *os.File, data []byte) (int, error) {
		// Unlike a looping stream writer, one syscall preserves the bounded
		// append boundary and exposes short writes for explicit uncertainty.
		return unix.Write(int(file.Fd()), data)
	})
}

func appendEnrollmentKeyWithWriter(ctx context.Context, home, key string, uid uint32, write func(*os.File, []byte) (int, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := PublicKey(key)
	if err != nil || normalized != key {
		return errors.New("normalized public key required")
	}
	homeFD, err := unix.Open(home, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(homeFD)
	if err := enrollmentSafeDirectory(homeFD, uid); err != nil {
		return err
	}
	if err := unix.Mkdirat(homeFD, ".ssh", 0700); err != nil && err != unix.EEXIST {
		return err
	}
	dirFD, err := unix.Openat(homeFD, ".ssh", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(dirFD)
	if err := enrollmentSafeDirectory(dirFD, uid); err != nil {
		return err
	}
	// This lock serializes Soda writers only, not unrelated native editors.
	if err := unix.Flock(dirFD, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return errors.New("authorized keys are busy")
	}
	defer unix.Flock(dirFD, unix.LOCK_UN)

	fd, err := unix.Openat(dirFD, "authorized_keys", unix.O_RDWR|unix.O_APPEND|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil && err != unix.ENOENT {
		return err
	}
	if err == nil {
		file := os.NewFile(uintptr(fd), "authorized_keys")
		defer file.Close()
		var before unix.Stat_t
		if err := unix.Fstat(fd, &before); err != nil {
			return err
		}
		if before.Mode&unix.S_IFMT != unix.S_IFREG || before.Uid != uid || before.Nlink != 1 || before.Mode&0022 != 0 || before.Size > 1<<20 {
			return errors.New("existing authorized_keys must be a bounded, singly linked, safely owned regular file")
		}
		existing, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
		if err != nil || len(existing) > 1<<20 {
			return errors.New("cannot inspect bounded authorized keys")
		}
		for _, line := range bytes.Split(existing, []byte{'\n'}) {
			// Do not duplicate an existing restricted key as unrestricted.
			fields := strings.Fields(string(line))
			for i := 0; i+1 < len(fields); i++ {
				if fields[i]+" "+fields[i+1] == key {
					return errors.New("this key already exists; verify its existing access policy")
				}
			}
		}
		var current unix.Stat_t
		if err := unix.Fstatat(dirFD, "authorized_keys", &current, unix.AT_SYMLINK_NOFOLLOW); err != nil || current.Dev != before.Dev || current.Ino != before.Ino || current.Size != before.Size || current.Mtim != before.Mtim || current.Ctim != before.Ctim {
			return errors.New("authorized keys changed before the append; no append started")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// One bounded O_APPEND write never rewrites old bytes. Always separate
		// the key with a newline: an in-place native edit between the check and
		// write must not merge our key with its last unterminated line.
		addition := []byte("\n" + key + "\n")
		n, err := write(file, addition)
		if err != nil || n != len(addition) {
			return errEnrollmentWriteUncertain
		}
		if err := file.Sync(); err != nil {
			return errEnrollmentWriteUncertain
		}
		expected := append(bytes.Clone(existing), addition...)
		if err := enrollmentConfirmKeyFile(dirFD, file, uid, expected); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return errEnrollmentWriteUncertain
		}
		return nil
	}

	// A missing file is published complete and exclusively. Linkat refuses a
	// concurrently created path, including a symlink; there is no overwrite or
	// rename fallback. Only our exact private temporary name is removed.
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	tempName := ".soda-enrollment-" + hex.EncodeToString(random[:])
	fd, err = unix.Openat(dirFD, tempName, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	defer unix.Unlinkat(dirFD, tempName, 0)
	file := os.NewFile(uintptr(fd), tempName)
	defer file.Close()
	contents := []byte(key + "\n")
	n, err := write(file, contents)
	if err != nil || n != len(contents) {
		return errors.New("new authorized-key file could not be written; no publication attempted")
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := unix.Linkat(dirFD, tempName, dirFD, "authorized_keys", 0); err != nil {
		return errors.New("authorized-key publication refused; any concurrently created file was preserved")
	}
	if err := unix.Unlinkat(dirFD, tempName, 0); err != nil {
		return errEnrollmentWriteUncertain
	}
	if err := unix.Fsync(dirFD); err != nil {
		return errEnrollmentWriteUncertain
	}
	if err := enrollmentConfirmKeyFile(dirFD, file, uid, contents); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return errEnrollmentWriteUncertain
	}
	return nil
}

// Confirmation is observational, not a lock against future native edits. If a
// native editor replaced the pathname while we appended, its new file remains
// untouched; the append may instead have reached the old, even unlinked, inode.
func enrollmentConfirmKeyFile(dirFD int, file *os.File, uid uint32, expected []byte) error {
	actual := make([]byte, len(expected)+1)
	n, err := file.ReadAt(actual, 0)
	if err != io.EOF || n != len(expected) || !bytes.Equal(actual[:n], expected) {
		return errEnrollmentWriteUncertain
	}
	var opened, named unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &opened); err != nil {
		return errEnrollmentWriteUncertain
	}
	if err := unix.Fstatat(dirFD, "authorized_keys", &named, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errEnrollmentWriteUncertain
	}
	if opened.Dev != named.Dev || opened.Ino != named.Ino || named.Mode&unix.S_IFMT != unix.S_IFREG || named.Uid != uid || named.Nlink != 1 || named.Mode&0022 != 0 || named.Size != int64(len(expected)) {
		return errEnrollmentWriteUncertain
	}
	return nil
}

func enrollmentSafeDirectory(fd int, uid uint32) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != uid || st.Mode&0022 != 0 {
		return errors.New("real safely owned directory required")
	}
	return nil
}

// CoreOS uses a native /root symlink. Resolve that native root-home alias once,
// then require real root-owned non-writable ancestors; never follow .ssh links.
func enrollmentRootHome() (string, error) {
	home, err := filepath.EvalSymlinks("/root")
	if err != nil {
		return "", err
	}
	for current := home; ; current = filepath.Dir(current) {
		fd, err := unix.Open(current, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return "", err
		}
		err = enrollmentSafeDirectory(fd, 0)
		unix.Close(fd)
		if err != nil {
			return "", err
		}
		if current == "/" {
			break
		}
	}
	return home, nil
}
