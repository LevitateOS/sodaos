package image

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// stageExtensionAssets copies only the files emitted by the separate Soda
// browser build. Bun resolves the JavaScript/CSS graph and records its output;
// this stage never guesses source dependencies from the old public payload.
func stageExtensionAssets(native, packageDir string) error {
	root := filepath.Join(native, "soda-extension-assets")
	files, err := extensionAssetInventory(root)
	if err != nil {
		return err
	}
	for i, name := range files {
		if !safeExtensionAssetName(name) || i > 0 && files[i-1] == name {
			return errors.New("unsafe Soda extension asset inventory")
		}
		if err := linkExtensionAsset(root, packageDir, name); err != nil {
			return err
		}
	}
	return nil
}

func extensionAssetInventory(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, "files.json"))
	if err != nil {
		return nil, err
	}
	var files []string
	if err = json.Unmarshal(data, &files); err != nil {
		return nil, err
	}
	if len(files) == 0 || !slices.IsSorted(files) {
		return nil, errors.New("sorted Soda extension asset inventory required")
	}
	return files, nil
}

func safeExtensionAssetName(name string) bool {
	return name != "" && path.Clean(name) == name && !path.IsAbs(name) && !strings.HasPrefix(name, "../") && !strings.Contains(name, "\\")
}

func linkExtensionAsset(root, packageDir, name string) error {
	from := filepath.Join(root, "assets", filepath.FromSlash(name))
	info, err := os.Lstat(from)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("regular Soda extension asset required")
	}
	to := filepath.Join(packageDir, "assets", filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.Link(from, to)
}
