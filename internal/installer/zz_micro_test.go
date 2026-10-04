package installer

import (
	"fmt"
	"net/netip"
	"testing"
)

func TestZZMicro(t *testing.T) {
	for _, s := range []string{"2001:db8:0:1:1:1:1:1", "1:0:0:2:0:0:0:3", "1:0:0:2:0:0:3:4",
		"0:0:1:2:3:4:5:6", "1:2:3:4:5:6:0:0", "0:1:2:3:4:5:6:7", "1:2:3:4:5:6:7:0",
		"::ffff:10.0.0.1", "::ffff:0:1", "1::", "::1:0", "64:ff9b:0:0:0:0:1:2",
		"fe80::1%", "fe80::1%eth0", "::ffff:0.0.0.0", "0:0:0:0:0:0:0:1", "0:0:0:0:0:0:13.1.68.3"} {
		a, err := netip.ParseAddr(s)
		if err != nil {
			fmt.Printf("M %q ERR\n", s)
			continue
		}
		fmt.Printf("M %q str=%q priv=%t is4in6=%t zone=%q\n", s, a.String(), a.IsPrivate(), a.Is4In6(), a.Zone())
	}
	for _, s := range []string{"10.0.0.0/00", "10.0.0.0/000", "10.0.0.0/0008", "fd00::/129", "fd00::/128", "fd00::/00", "::ffff:1.2.3.4/128", "::ffff:1.2.3.4/96"} {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			fmt.Printf("MP %q ERR\n", s)
			continue
		}
		fmt.Printf("MP %q ok masked=%t\n", s, p == p.Masked())
	}
}
