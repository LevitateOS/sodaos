package nativebuild

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCoreOSISOInputsAreSeparateFromQEMU(t *testing.T) {
	for _, arch := range []string{"x86_64", "aarch64"} {
		lock, img, err := ReadCoreOSISO("../../appliance/locks/coreos-iso.json", arch)
		if err != nil || lock.Release != "44.20260817.3.2" || img.SHA256 == "" {
			t.Fatal("missing selected ISO input", err)
		}
		if _, _, err := ReadCoreOSISO("../../appliance/locks/coreos-qemu.json", arch); err == nil {
			t.Fatal("QEMU lock admitted as ISO")
		}
		if _, _, err := ReadCoreOS("../../appliance/locks/coreos-iso.json", arch); err == nil {
			t.Fatal("ISO lock admitted as QEMU")
		}
	}
	if _, _, err := ReadCoreOSISO("../../appliance/locks/coreos-iso.json", "riscv64"); err == nil {
		t.Fatal("unknown architecture admitted")
	}
}

func TestCoreOSISORejectsUnsafeMetadata(t *testing.T) {
	original, _, err := ReadCoreOSISO("../../appliance/locks/coreos-iso.json", "x86_64")
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*CoreOSImage){
		"http":        func(i *CoreOSImage) { i.URL = "http://example.test/coreos.iso" },
		"credentials": func(i *CoreOSImage) { i.URL = "https://secret@example.test/coreos.iso" },
		"signature":   func(i *CoreOSImage) { i.SignatureURL = i.URL },
		"checksum":    func(i *CoreOSImage) { i.SHA256 = "missing" },
		"compressed":  func(i *CoreOSImage) { i.UncompressedSHA256 = i.SHA256 },
	} {
		t.Run(name, func(t *testing.T) {
			img := original.Architectures["x86_64"]
			change(&img)
			lock := CoreOSLock{MetadataURL: original.MetadataURL, Release: original.Release, Architectures: map[string]CoreOSImage{"x86_64": img}}
			data, _ := json.Marshal(lock)
			path := filepath.Join(t.TempDir(), "lock.json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := ReadCoreOSISO(path, "x86_64"); err == nil {
				t.Fatal("unsafe ISO metadata accepted")
			}
		})
	}
}
