//go:build !soda_host_image

// Package installlayout fixes native paths at build time. The existing writable
// installer and the bootable host image have separate packaging contracts; runtime
// callers cannot select a helper path or broaden trusted unit locations.
package installlayout

const Libexec = "/usr/local/libexec/soda"
const Sbin = "/usr/local/sbin"
const ProjectUnit = "/etc/systemd/system/soda-project@.service"
const Release = ""
