package image

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/release/build"
)

const defaultRootfsOptions = "-zlzma,level=6 -Efragments -C1048576 --quiet"
const fastRootfsOptions = "-zlzma,level=1 -Efragments -C1048576 --quiet"
const imageConfigPath = "usr/share/coreos-assembler/image.json"

// The caller has admitted the development/media request. Only this metadata field
// changes; upstream Assembler reads it from the resulting distinct host candidate.
func setMediaCompression(config map[string]json.RawMessage, mode string) error {
	if mode == "" {
		return nil // production and ordinary development preserve upstream defaults
	}
	var fs, options string
	if mode != "fast" || json.Unmarshal(config["live-rootfs-fstype"], &fs) != nil || json.Unmarshal(config["live-rootfs-fsoptions"], &options) != nil || fs != "erofs" || options != defaultRootfsOptions {
		return errors.New("fast media requires the reviewed upstream EROFS/LZMA defaults")
	}
	config["live-rootfs-fsoptions"], _ = json.Marshal(fastRootfsOptions)
	return nil
}

func recordImageConfig(context, out, observed string) error {
	expected, err := os.ReadFile(filepath.Join(context, "rootfs", imageConfigPath))
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(expected)) != observed {
		return errors.New("host image configuration differs from admitted metadata")
	}
	return build.WriteNew(filepath.Join(out, "image-config.json"), expected, 0o644)
}

func rootfsSettings(out string) (filesystem, options string, err error) {
	var config struct {
		Filesystem string `json:"live-rootfs-fstype"`
		Options    string `json:"live-rootfs-fsoptions"`
	}
	data, err := os.ReadFile(filepath.Join(out, "image-config.json"))
	if err != nil {
		return "", "", err
	}
	err = json.Unmarshal(data, &config)
	return config.Filesystem, config.Options, err
}
