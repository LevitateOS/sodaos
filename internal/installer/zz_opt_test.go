package installer

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestZZOpt(t *testing.T) {
	ed := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKQ0MsA1tWa7risZNVfq58qNB9BByJfSJUQpWI9KvglV"
	for _, v := range []string{",,, " + ed, ",,," + ed, "\"\" " + ed, " , " + ed, ed + " ", " " + ed + " "} {
		key, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(v))
		typ := "";
		if err == nil {
			typ = key.Type()
		}
		fmt.Printf("OPT %q err=%v type=%q comment=%q options=%q rest=%q\n", v, err != nil, typ, comment, options, rest)
		_ = strings.TrimSpace
	}
}
