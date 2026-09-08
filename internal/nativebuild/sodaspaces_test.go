package nativebuild

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSodaspacesPayloadRequiredAndReadable(t *testing.T) {
	for _, name := range sodaspacesFiles {
		t.Run(name, func(t *testing.T) {
			root := fixtureBundle(t)
			p := filepath.Join(root, name)
			if err := os.Chmod(p, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := tree(root); err == nil {
				t.Fatal("accepted unreadable Soda file")
			}
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if _, err := tree(root); err == nil {
				t.Fatal("accepted missing Soda file")
			}
		})
	}
	root := fixtureBundle(t)
	p := filepath.Join(root, "rootfs/var/lib/soda/forgejo/gitea/templates/custom")
	if err := os.Chmod(p, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := tree(root); err == nil {
		t.Fatal("accepted unreadable templates directory")
	}
}

func TestSodaspacesDoesNotAdmitArbitraryTemplatesOrData(t *testing.T) {
	for _, name := range []string{"templates/base/head.tmpl", "templates/custom/extra_tabs.tmpl", "templates/custom/header.tmpl/extra", "conf/app.ini", "gitea.db"} {
		if allowedPayload("rootfs/var/lib/soda/forgejo/gitea/" + name) {
			t.Fatalf("accepted %s", name)
		}
	}
	root := fixtureBundle(t)
	p := filepath.Join(root, "rootfs/var/lib/soda/forgejo/gitea/templates/custom/footer.tmpl")
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("header.tmpl", p); err != nil {
		t.Fatal(err)
	}
	if _, err := tree(root); err == nil {
		t.Fatal("accepted symlinked hook")
	}
}
