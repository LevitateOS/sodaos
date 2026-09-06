package nativebuild

import (
	"debug/elf"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func inspectELF(path, arch string) error {
	want := elf.EM_X86_64
	if arch == "aarch64" {
		want = elf.EM_AARCH64
	} else if arch != "x86_64" {
		return errors.New("unknown native architecture")
	}
	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if f.Class != elf.ELFCLASS64 || f.Data != elf.ELFDATA2LSB || f.Machine != want || (f.Type != elf.ET_EXEC && f.Type != elf.ET_DYN) {
		return errors.New("native executable format/platform mismatch")
	}
	return nil
}
func inspectBinaries(root, arch string, files map[string]File) error {
	for name, entry := range files {
		if entry.Directory || entry.Link != "" {
			continue
		}
		if name != "tools/soda-artifacts" && !strings.HasPrefix(name, "rootfs/usr/local/libexec/soda/") {
			continue
		}
		if name == "rootfs/usr/local/libexec/soda/soda-console-welcome" {
			continue
		} // delivered shell hook, not Go
		if os.FileMode(entry.Mode)&0111 == 0 {
			return errors.New("native command is not executable")
		}
		if err := inspectELF(filepath.Join(root, name), arch); err != nil {
			return err
		}
	}
	return nil
}
