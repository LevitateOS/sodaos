package image

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtensionAssetStageKeepsRecordedOutputAndRefusesUnsafeFiles(t *testing.T) {
	for _, bad := range []string{"", "../outside.js", "entry.js\nentry.js", "linked.js"} {
		t.Run(bad, func(t *testing.T) {
			root := t.TempDir()
			native := filepath.Join(root, "native")
			built := filepath.Join(native, "soda-extension-assets")
			if err := os.MkdirAll(filepath.Join(built, "assets"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(built, "assets/entry.js"), []byte("export function mount() {}"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("entry.js", filepath.Join(built, "assets/linked.js")); err != nil {
				t.Fatal(err)
			}
			inventory := "[\"entry.js\"]\n"
			switch bad {
			case "../outside.js":
				inventory = "[\"../outside.js\"]\n"
			case "entry.js\nentry.js":
				inventory = "[\"entry.js\",\"entry.js\"]\n"
			case "linked.js":
				inventory = "[\"linked.js\"]\n"
			}
			if err := os.WriteFile(filepath.Join(built, "files.json"), []byte(inventory), 0o600); err != nil {
				t.Fatal(err)
			}
			packageDir := filepath.Join(root, "package")
			err := stageExtensionAssets(native, packageDir)
			if bad != "" {
				if err == nil {
					t.Fatal("unsafe generated asset admitted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			from, _ := os.Stat(filepath.Join(built, "assets/entry.js"))
			to, err := os.Stat(filepath.Join(packageDir, "assets/entry.js"))
			if err != nil || !os.SameFile(from, to) {
				t.Fatal("staged asset differs from the browser build")
			}
		})
	}
}
