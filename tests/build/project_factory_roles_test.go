// Behavioral coverage for the project-factory-roles helper. Every case
// drives the compiled Rust binary with the factory redirected to a
// test-owned directory plus a recording stand-in for git; the embedded
// interpreter driver is gone and no interpreter runs anywhere here.
package build

import (
	"bytes"
	"testing"
)

func TestRolesRejectsMalformedRequests(t *testing.T) {
	fenv := rolesSetup(t)
	for _, body := range []string{
		"not json",
		"",
		"[1, 2]",
		"null",
		`{"op": "frobnicate"}`,
		`{"id": "x"}`,
		`{"op": "ensure", "extra": 1}`,
		`{"op": "hold", "revision": true}`,
		`{"op": "approve"}`,
		`{"op": "stop", "id": "../escape"}`,
	} {
		rolesRefused(t, fenv, []byte(body))
	}
	rolesRefused(t, fenv, bytes.Repeat([]byte("x"), 4*1024*1024+65536+1))
}
