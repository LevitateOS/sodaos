package installer

import (
	"fmt"
	"net/url"
	"testing"
)

func TestZZUrlErr(t *testing.T) {
	for _, s := range []string{"https://", "https://?x", "https:///path", "https:foo", "//", "https://a b"} {
		u, err := url.Parse(s)
		fmt.Printf("UE %q err=%v url=%+v\n", s, err, u)
	}
}
