// fixture serves the locally built system image and files it after a
// development media build. The ISO is only the boot menu; the installer
// downloads the big rootfs file from here. Loopback development media only:
// any other mode or address stays fully operator-managed.
package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// defaultRootfsDir points the pickup folder at the checkout's ignored
// artifacts instead of the removed system path: everything the wrapper
// serves and files stays on the roomy disk, never on root.
func defaultRootfsDir(o *options) error {
	if o.rootfsDir != "" {
		return nil
	}
	source, err := os.Getwd()
	if err != nil {
		return err
	}
	o.rootfsDir = filepath.Join(source, ".artifacts", "rootfs")
	return os.MkdirAll(o.rootfsDir, 0o755)
}

// fixtureWanted reports whether the wrapper should serve the pickup address
// itself: development media with a loopback URL. Public URLs stay
// operator-managed.
func fixtureWanted(mode, rootfsURL string) bool {
	if mode != "media" {
		return false
	}
	u, err := url.Parse(rootfsURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// fixtureAddr splits the URL into a listen address for the file server.
func fixtureAddr(rootfsURL string) (string, error) {
	u, err := url.Parse(rootfsURL)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	if host == "localhost" {
		host = "127.0.0.1"
	}
	port := u.Port()
	if port == "" {
		return "", errors.New("rootfs URL needs an explicit port")
	}
	return net.JoinHostPort(host, port), nil
}

// serveFixture serves dir over HTTP until stop is called. A busy port means
// the operator already serves it; that is not an error. It reports the
// actual listen address so tests can use ephemeral ports.
func serveFixture(addr, dir string) (stop func(), listen string, err error) {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, "", fmt.Errorf("pickup directory %s missing; rerun the setup script", dir)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(dir)))
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			return func() {}, addr, nil
		}
		return nil, "", err
	}
	go func() {
		_ = server.Serve(ln)
	}()
	return func() {
		_ = server.Close()
	}, ln.Addr().String(), nil
}

// copyBuiltRootfs files the produced image into the served directory and
// reports the pickup file name. The build output stays the source of truth.
func copyBuiltRootfs(outDir, serveDir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(outDir, "artifacts", "media", "*-rootfs.img"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", errors.New("no rootfs image in build output; the media build did not produce one")
	}
	name := filepath.Base(matches[0])
	dst := filepath.Join(serveDir, name)
	if err := copyFile(matches[0], dst); err != nil {
		return "", err
	}
	if user := os.Getenv("SUDO_USER"); user != "" {
		_ = chownName(dst, user)
	}
	return name, nil
}

// chownName hands a filed artifact back to the invoking operator.
// Best effort: the file stays usable when the lookup fails.
func chownName(path, name string) error {
	u, err := user.Lookup(name)
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}
	return os.Chown(path, uid, gid)
}

func copyFile(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("occupied pickup file %s refused", dst)
	} else if !os.IsNotExist(err) {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	return errors.Join(copyErr, closeErr)
}
