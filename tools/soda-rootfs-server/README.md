# soda-rootfs-server

Serve selected public installer rootfs images to development VMs over HTTP.
This is a builder-side operator service for the retained libvirt fixture network.
It reads from `/home/soda-rootfs` and binds `192.168.122.1:8080`.

## Use a prepared builder

The builder must have that bridge address, a free port 8080 and a readable
rootfs-only directory. [Candidate setup](../candidate-setup/README.md) prepares
the directory; installing or starting the server is a separate operator action.
The [service template](soda-rootfs-server.service) runs an installed binary at
`/usr/local/lib/soda/soda-rootfs-server` with a dynamic service user and network
restrictions. Inspect an already installed service with:

```sh
systemctl status soda-rootfs-server.service
journalctl -u soda-rootfs-server.service
```

For an explicitly selected development builder, the source can also run in the
foreground from the repository root:

```sh
go run ./tools/soda-rootfs-server
```

There are no command-line options or environment overrides for the address or
directory. Arguments, including `--help`, are ignored and start the server.
A listen failure prints to stderr and exits `1`.

## Fetch an image

Only regular files named `<64 lowercase hexadecimal characters>-rootfs.img`
are served. Copy the exact public image from the selected media output into
this dedicated directory, then use its actual digest in the request:

```sh
curl --fail --head http://192.168.122.1:8080/ROOTFS_SHA256-rootfs.img
```

Replace `ROOTFS_SHA256` with the file's 64-character digest. Successful `GET`
streams the file and `HEAD` returns its size; responses use `no-store`. Invalid
names, missing files, subdirectories and symlinks return `404`; methods other
than `GET` and `HEAD` return `501`. The handler offers no directory listing or
authentication. Serve only public installer bytes, and keep VM disks and private
inputs outside this directory.

See [release tools](../../lib/soda-release-tools/README.md) and
[native support](../../docs/development/native-support.md#local-host-content-image-candidate)
for media production and rootfs pickup.
