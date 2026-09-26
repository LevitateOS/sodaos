package build

import (
	"context"
	"strings"
	"testing"
)

func TestCoreOSISOResolvesLiveTriple(t *testing.T) {
	streamFixtureServer(t, fixtureStreamDoc(t, nil), fixtureIndexDoc(t, nil), 200)
	for _, arch := range []string{"x86_64", "aarch64"} {
		release, img, err := ResolveCoreOSISO(context.Background(), arch)
		if err != nil || release != "44.20260901.1.0" || img.SHA256 != strings.Repeat("a", 64) {
			t.Fatal("missing resolved ISO input", err)
		}
		if !strings.HasSuffix(img.URL, ".iso") || img.SignatureURL != img.URL+".sig" {
			t.Fatalf("resolved ISO triple malformed: %+v", img)
		}
		_, qemu, err := ResolveCoreOSQEMU(context.Background(), arch)
		if err != nil {
			t.Fatal(err)
		}
		if qemu.URL == img.URL {
			t.Fatal("QEMU image admitted as ISO")
		}
	}
	if _, _, err := ResolveCoreOSISO(context.Background(), "riscv64"); err == nil {
		t.Fatal("unknown architecture admitted")
	}
}

func TestCoreOSISORejectsUnsafeStream(t *testing.T) {
	for name, mutate := range map[string]func(string, string, map[string]string){
		"http": func(_, _ string, disk map[string]string) {
			disk["location"] = strings.Replace(disk["location"], "https://", "http://", 1)
		},
		"credentials": func(_, _ string, disk map[string]string) {
			disk["location"] = strings.Replace(disk["location"], "https://", "https://secret@", 1)
		},
		"signature": func(_, _ string, disk map[string]string) { disk["signature"] = disk["location"] },
		"checksum":  func(_, _ string, disk map[string]string) { disk["sha256"] = "missing" },
	} {
		t.Run(name, func(t *testing.T) {
			streamFixtureServer(t, fixtureStreamDoc(t, mutate), fixtureIndexDoc(t, nil), 200)
			if _, _, err := ResolveCoreOSISO(context.Background(), "x86_64"); err == nil {
				t.Fatal("unsafe ISO metadata accepted")
			}
		})
	}
}
