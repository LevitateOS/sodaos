package host

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"strings"

	"github.com/levitateos/sodaos/internal/projectos"
)

const profileInspectFormat = `{"Id":{{json .ID}},"Architecture":{{json .Architecture}},"Os":{{json .Os}},"Labels":{{json .Labels}}}`

// ResolveProfile inspects only the configured installed image. It never pulls,
// runs a container, changes a tag or accepts an image/architecture from the caller.
func (c *Client) ResolveProfile(ctx context.Context) (projectos.Profile, error) {
	var p projectos.Profile
	err := c.call(ctx, "/profile", struct{}{}, &p)
	if err == nil {
		err = p.Validate()
		if p.Architecture != runtime.GOARCH {
			err = errors.New("project image is not native to this backend")
		}
	}
	return p, err
}
func (d *Daemon) resolveProfile(ctx context.Context) (projectos.Profile, error) {
	var p projectos.Profile
	raw, err := d.podman(ctx, nil, "image", "inspect", "--format", profileInspectFormat, d.Config.Image)
	if err != nil {
		return p, err
	}
	// Podman's ImageData exposes Labels at the top level, not container Config.
	var image struct {
		ID, Architecture, OS string
		Labels               map[string]string
	}
	if len(raw) > 65536 || json.Unmarshal(raw, &image) != nil {
		return p, errors.New("invalid installed image inspection")
	}
	if !strings.HasPrefix(image.ID, "sha256:") {
		image.ID = "sha256:" + image.ID
	}
	p = projectos.Profile{ID: image.Labels["org.soda.profile"], Distribution: image.Labels["org.soda.distribution"], Version: image.Labels["org.soda.distribution.version"], Interface: image.Labels["org.soda.interface"], Architecture: image.Architecture, Image: image.ID, Revision: image.Labels["org.opencontainers.image.revision"]}
	if image.OS != "linux" || image.Architecture != runtime.GOARCH {
		return p, errors.New("project image is not native Linux")
	}
	return p, p.Validate()
}
