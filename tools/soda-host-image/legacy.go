package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

type buildCapture = nativebuild.BuildCapture

// Compatibility assembly only. The shell retains its clean-checkout lock,
// inventory/seal and ISO handoff. No common compilation/asset/image command is
// independently maintained there, and this profile cannot emit a host candidate.
func legacyNative(source, out, arch, revision string, execution nativebuild.BuildExecution, next func(string) error) error {
	for _, name := range []string{"bin", "tools"} {
		if e := os.Mkdir(filepath.Join(out, name), 0755); e != nil {
			return e
		}
	}
	p := nativebuild.Production{Source: source, Native: out, Out: out, Arch: arch, Revision: revision, Execute: execution.Execute, Capture: execution.Capture, Next: next}
	for _, name := range []string{"soda-artifacts", "soda-acceptance"} {
		if e := p.Compile(name, "./tools/"+name, filepath.Join(out, "tools", name)); e != nil {
			return e
		}
	}
	names, e := nativebuild.SodaCommands(source)
	if e != nil {
		return e
	}
	for _, name := range names {
		if e = p.Compile(name, "./cmd/"+name, filepath.Join(out, "bin", name)); e != nil {
			return e
		}
	}
	if e = p.Assets(); e != nil {
		return e
	}
	if _, e = p.Images(""); e != nil {
		return e
	}
	fmt.Println("Legacy native components prepared at", out, "; caller still owns metadata and sealing; no host image, publication or installation")
	return nil
}
