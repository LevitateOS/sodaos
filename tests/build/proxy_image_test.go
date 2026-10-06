// Port of test_proxy_image.py: proxy image pin checks.
package build

import (
	"regexp"
	"strings"
	"testing"
)

func TestProxyImageIsDigestPinned(t *testing.T) {
	var images []string
	for _, line := range strings.Split(ReadFile(t, "system/host/services/soda-proxy.container"), "\n") {
		if strings.HasPrefix(line, "Image=") {
			images = append(images, strings.TrimSpace(strings.SplitN(line, "=", 2)[1]))
		}
	}
	Require(t, len(images) == 1, "Image= entries = %d, want 1", len(images))
	matched, err := regexp.MatchString(`^docker\.io/library/caddy:2\.10\.2@sha256:[0-9a-f]{64}$`, images[0])
	Require(t, err == nil, "regex: %v", err)
	Check(t, matched, "proxy image %q is not digest-pinned", images[0])
}
