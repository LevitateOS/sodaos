//go:build branding

// Optional native renderer verification; requires rsvg-convert and Go.
// Ported from soda-os bc1d3e0. Not run during source-only work.
package scripts

import (
	"github.com/stretchr/testify/require"
	"os/exec"
	"testing"
)

func TestForgejoBrandingMatchesSVGMaster(t *testing.T) {
	check := exec.Command("scripts/render-forgejo-branding.sh", "--check")
	check.Dir = ".."
	output, err := check.CombinedOutput()
	require.NoErrorf(t, err, "Forgejo branding is stale:\n%s", output)
}
