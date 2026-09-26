package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

// Squid owns the upstream HTTP implementation. Worker networks have no DNS or
// default route; CONNECT permits only selected public TLS destinations.
func proxyConfiguration(domains []string) string {
	return `http_port 3128
acl CONNECT method CONNECT
acl SSL_ports port 443
acl permitted dstdomain ` + strings.Join(domains, " ") + `
acl nonpublic dst 0.0.0.0/8 10.0.0.0/8 100.64.0.0/10 127.0.0.0/8 169.254.0.0/16 172.16.0.0/12 192.168.0.0/16 224.0.0.0/4 240.0.0.0/4 ::/128 ::1/128 fc00::/7 fe80::/10 ff00::/8
http_access deny !CONNECT
http_access deny !SSL_ports
http_access deny nonpublic
http_access allow permitted
http_access deny all
cache deny all
cache_mem 0 MB
access_log none
cache_log /dev/stderr
cache_store_log none
pid_filename /tmp/squid.pid
coredump_dir /tmp
shutdown_lifetime 1 seconds
`
}

func (w *Runtime) createNetwork(ctx context.Context, r *factory.Run, resource *factory.Resource) error {
	if _, err := w.Exec.Run(ctx, nil, "podman", "network", "create", "--internal", "--disable-dns", "--ipv6", "--opt", "isolate=true", "--label", ownership(r.ID), resource.Name); err != nil {
		return err
	}
	id, exists, err := w.observeResource(ctx, *r, *resource)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("created network is missing")
	}
	resource.ID = id
	return nil
}

func (w *Runtime) createProxy(ctx context.Context, r *factory.Run, resource *factory.Resource, network string) error {
	path := filepath.Join(w.Config.Root, r.ID, "squid.conf")
	if err := os.WriteFile(path, []byte(proxyConfiguration(w.Config.AllowedDomains)), 0o600); err != nil {
		return err
	}
	args := w.containerArguments(r, resource)
	args = append(args, "--network=podman", "--network="+network, "--tmpfs", "/tmp:rw,nosuid,nodev,size=32m", "--volume", path+":/etc/soda-egress.conf:ro,Z", w.Config.ProxyImage, "/usr/sbin/squid", "-N", "-f", "/etc/soda-egress.conf")
	return w.createContainer(ctx, resource, args)
}
