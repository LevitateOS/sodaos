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

func openAndLockSSHDirectory(home string, uid uint32) (int, func(), error) {
	homeFD, err := unix.Open(home, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, nil, err
	}
	defer unix.Close(homeFD)
	if err := enrollmentSafeDirectory(homeFD, uid); err != nil {
		return -1, nil, err
	}
	if err := unix.Mkdirat(homeFD, ".ssh", 0o700); err != nil && err != unix.EEXIST {
		return -1, nil, err
	}
	dirFD, err := unix.Openat(homeFD, ".ssh", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, nil, err
	}
	if err := enrollmentSafeDirectory(dirFD, uid); err != nil {
		unix.Close(dirFD)
		return -1, nil, err
	}
	if err := unix.Flock(dirFD, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		unix.Close(dirFD)
		return -1, nil, errors.New("authorized keys are busy")
	}
	cleanup := func() {
		unix.Flock(dirFD, unix.LOCK_UN)
		unix.Close(dirFD)
	}
	return dirFD, cleanup, nil
}

func validateAuthorizedKeysStat(st unix.Stat_t, uid uint32) error {
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uid || st.Nlink != 1 || st.Mode&0o022 != 0 || st.Size > 1<<20 {
		return errors.New("existing authorized_keys must be a bounded, singly linked, safely owned regular file")
	}
	return nil
}

func hasAuthorizedKey(data []byte, key string) bool {
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		fields := strings.Fields(string(line))
		for i := 0; i+1 < len(fields); i++ {
			if fields[i]+" "+fields[i+1] == key {
				return true
			}
		}
	}
	return false
}

func inspectExistingAuthorizedKeys(file *os.File, key string, uid uint32) (unix.Stat_t, []byte, error) {
	var before unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &before); err != nil {
		return unix.Stat_t{}, nil, err
	}
	if err := validateAuthorizedKeysStat(before, uid); err != nil {
		return unix.Stat_t{}, nil, err
	}
	existing, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(existing) > 1<<20 {
		return unix.Stat_t{}, nil, errors.New("cannot inspect bounded authorized keys")
	}
	if hasAuthorizedKey(existing, key) {
		return unix.Stat_t{}, nil, errors.New("this key already exists; verify its existing access policy")
	}
	return before, existing, nil
}

func verifyAuthorizedKeysUnchanged(dirFD int, before unix.Stat_t) error {
	var current unix.Stat_t
	if err := unix.Fstatat(dirFD, "authorized_keys", &current, unix.AT_SYMLINK_NOFOLLOW); err != nil ||
		current.Dev != before.Dev || current.Ino != before.Ino ||
		current.Size != before.Size || current.Mtim != before.Mtim || current.Ctim != before.Ctim {
		return errors.New("authorized keys changed before the append; no append started")
	}
	return nil
}

func appendExistingKey(ctx context.Context, dirFD int, file *os.File, key string, uid uint32, write func(*os.File, []byte) (int, error)) error {
	before, existing, err := inspectExistingAuthorizedKeys(file, key, uid)
	if err != nil {
		return err
	}
	if err := verifyAuthorizedKeysUnchanged(dirFD, before); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
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

func writeTempKeyFile(dirFD int, tempName string, contents []byte, write func(*os.File, []byte) (int, error)) (*os.File, error) {
	fd, err := unix.Openat(dirFD, tempName, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), tempName)
	n, err := write(file, contents)
	if err != nil || n != len(contents) {
		file.Close()
		return nil, errors.New("new authorized-key file could not be written; no publication attempted")
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func linkAndConfirmKeyFile(ctx context.Context, dirFD int, tempName string, file *os.File, uid uint32, contents []byte) error {
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

func createExclusiveKeyFile(ctx context.Context, dirFD int, key string, uid uint32, write func(*os.File, []byte) (int, error)) error {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	tempName := ".soda-enrollment-" + hex.EncodeToString(random[:])
	contents := []byte(key + "\n")
	file, err := writeTempKeyFile(dirFD, tempName, contents, write)
	if err != nil {
		return err
	}
	defer file.Close()
	defer unix.Unlinkat(dirFD, tempName, 0)
	if err := ctx.Err(); err != nil {
		return err
	}
	return linkAndConfirmKeyFile(ctx, dirFD, tempName, file, uid, contents)
}

func appendEnrollmentKeyWithWriter(ctx context.Context, home, key string, uid uint32, write func(*os.File, []byte) (int, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := PublicKey(key)
	if err != nil || normalized != key {
		return errors.New("normalized public key required")
	}
	dirFD, closeSSH, err := openAndLockSSHDirectory(home, uid)
	if err != nil {
		return err
	}
	defer closeSSH()

	fd, err := unix.Openat(dirFD, "authorized_keys", unix.O_RDWR|unix.O_APPEND|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil && err != unix.ENOENT {
		return err
	}
	if err == nil {
		file := os.NewFile(uintptr(fd), "authorized_keys")
		defer file.Close()
		return appendExistingKey(ctx, dirFD, file, key, uid, write)
	}
	return createExclusiveKeyFile(ctx, dirFD, key, uid, write)
}

func confirmKeyFileContents(file *os.File, expected []byte) error {
	actual := make([]byte, len(expected)+1)
	n, err := file.ReadAt(actual, 0)
	if err != io.EOF || n != len(expected) || !bytes.Equal(actual[:n], expected) {
		return errEnrollmentWriteUncertain
	}
	return nil
}

func validateKeyFileMatch(opened, named unix.Stat_t, uid uint32, expectedLen int) error {
	if opened.Dev != named.Dev || opened.Ino != named.Ino || named.Nlink != 1 {
		return errEnrollmentWriteUncertain
	}
	if named.Mode&unix.S_IFMT != unix.S_IFREG || named.Uid != uid || named.Mode&0o022 != 0 || named.Size != int64(expectedLen) {
		return errEnrollmentWriteUncertain
	}
	return nil
}

func confirmKeyFileStats(dirFD int, file *os.File, uid uint32, expectedLen int) error {
	var opened, named unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &opened); err != nil {
		return errEnrollmentWriteUncertain
	}
	if err := unix.Fstatat(dirFD, "authorized_keys", &named, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errEnrollmentWriteUncertain
	}
	return validateKeyFileMatch(opened, named, uid, expectedLen)
}

// Confirmation is observational, not a lock against future native edits. If a
// native editor replaced the pathname while we appended, its new file remains
// untouched; the append may instead have reached the old, even unlinked, inode.
func enrollmentConfirmKeyFile(dirFD int, file *os.File, uid uint32, expected []byte) error {
	if err := confirmKeyFileContents(file, expected); err != nil {
		return err
	}
	return confirmKeyFileStats(dirFD, file, uid, len(expected))
}

func enrollmentSafeDirectory(fd int, uid uint32) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != uid || st.Mode&0o022 != 0 {
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
