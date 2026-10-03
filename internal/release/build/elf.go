package build

import (
	"debug/elf"
	"errors"
)

func inspectELF(path, arch string) error {
	if arch != "x86_64" {
		return errors.New("unknown native architecture")
	}
	want := elf.EM_X86_64
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
