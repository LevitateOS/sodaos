//go:build linux

package terminal

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

// gitListener accepts only a listening IPv4 loopback socket in the pinned
// caller's network namespace. The received file remains owned by the caller.
func gitListener(file *os.File, peer MusePeer) (net.Listener, error) {
	if file == nil || !musePeerAlive(peer) {
		return nil, identity.ErrDenied
	}
	fd := int(file.Fd())
	if err := gitSocketAddress(fd); err != nil {
		return nil, err
	}
	if err := gitSocketNamespace(fd, peer); err != nil {
		return nil, err
	}
	return net.FileListener(file)
}

func gitSocketAddress(fd int) error {
	kind, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_TYPE)
	if err != nil || kind != unix.SOCK_STREAM {
		return identity.ErrDenied
	}
	listening, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ACCEPTCONN)
	if err != nil || listening != 1 {
		return identity.ErrDenied
	}
	address, err := unix.Getsockname(fd)
	v4, ok := address.(*unix.SockaddrInet4)
	if err != nil || !ok || v4.Addr != [4]byte{127, 0, 0, 1} || v4.Port == 0 {
		return identity.ErrDenied
	}
	return nil
}

func gitSocketNamespace(fd int, peer MusePeer) error {
	namespace, err := unix.IoctlRetInt(fd, unix.SIOCGSKNS)
	if err != nil {
		return identity.ErrDenied
	}
	defer unix.Close(namespace)
	actual, err := os.Readlink("/proc/self/fd/" + strconv.Itoa(namespace))
	if err != nil {
		return identity.ErrDenied
	}
	expected, err := os.Readlink("/proc/" + strconv.Itoa(peer.PID) + "/ns/net")
	if err != nil || actual != expected || !musePeerAlive(peer) {
		return identity.ErrDenied
	}
	return nil
}

func gitCapability() (string, error) {
	var value [32]byte
	_, err := rand.Read(value[:])
	return hex.EncodeToString(value[:]), err
}

func gitRelay(capability string, proxy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := r.Header.Values("X-Soda-Git-Invocation")
		if len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), []byte(capability)) != 1 {
			http.Error(w, "Git unavailable.", http.StatusForbidden)
			return
		}
		r.Header.Del("X-Soda-Git-Invocation")
		proxy.ServeHTTP(w, r)
	})
}

func gitServe(ctx context.Context, listener net.Listener, server *http.Server) {
	go func() { <-ctx.Done(); _ = server.Close() }()
	go func() { _ = server.Serve(listener) }()
}
