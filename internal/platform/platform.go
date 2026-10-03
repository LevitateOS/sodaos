// Package platform fixes native paths at build time. The single image
// layout owns every path; runtime callers cannot select a helper path or
// broaden trusted unit locations.
package platform

const (
	Libexec     = "/usr/libexec/soda"
	Sbin        = "/usr/bin"
	ProjectUnit = "/usr/lib/systemd/system/soda-project@.service"
	Release     = "/usr/share/soda/release.json"
)
