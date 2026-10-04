package installer

import (
	"fmt"
	"net/netip"
	"testing"
)

func TestZZV6(t *testing.T) {
	for _, s := range []string{"1.2.3.4::5", "1.2.3.4::", "::1.2.3.4", "1::2.3.4.5",
		"0:0:0:0:0:0:13.1.68.3", "0:0:0:0:0:13.1.68.3:0", "1:2:3:4:5:6:7::", "1:2:3:4:5:6:7:8::",
		"::1:2:3:4:5:6:7", "1:2:3:4:5:6:7:8:9", "abcd:ef01:2345:6789:abcd:ef01:2345:6789",
		"1:2:3:4:5:6:7:8:", ":1:2:3:4:5:6:7:8", "1:2:3:4::5:6:7:8", "::ffff", "ffff::",
		"1:2:3:4:5:6:255.255.255.255", "1:2:3:4:5:255.255.255.255:7"} {
		a, err := netip.ParseAddr(s)
		if err != nil {
			fmt.Printf("V6 %q ERR\n", s)
			continue
		}
		fmt.Printf("V6 %q str=%q\n", s, a.String())
	}
}
