// Port of test_sodaspaces.py: staging/preflight in temporary filesystems.
package build

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

var sodaspacesFiles = []string{
	"templates/custom/header.tmpl",
	"templates/custom/footer.tmpl",
	"public/assets/soda/forgejo/repository-actions.js",
	"public/assets/soda/forgejo/notification-preview.js",
}

func testVMBinary(t *testing.T) string {
	t.Helper()
	return CargoBinary(t, "soda-test-vm", "soda-test-vm", "--bin", "soda-test-vm")
}

func stageBinary(t *testing.T) string {
	t.Helper()
	return CargoBinary(t, "soda-stage-render", "soda-stage", "--bin", "soda-stage")
}

func TestSodaspacesSpacesEntryIsPackagedByTheExtension(t *testing.T) {
	extension := ReadJSON(t, "system/containers/extension/extension.json").(map[string]any)
	manifest := payloadFiles(t)
	var spaces map[string]any
	for _, entry := range extension["pages"].([]any) {
		page := entry.(map[string]any)
		if page["id"] == "spaces" {
			spaces = page
		}
	}
	Require(t, spaces != nil, "spaces page missing")
	Check(t, spaces["entry"] == "soda-spaces-entry.js", "entry = %v", spaces["entry"])
	info, err := os.Stat(filepath.Join(RepoRoot, "frontend/spaces/soda-spaces-entry.ts"))
	Check(t, err == nil && !info.IsDir(), "spaces entry missing")
	_, present := manifest["public/assets/sodaspaces.js"]
	Check(t, !present, "retired sodaspaces.js in manifest")
	_, present = manifest["public/assets/soda-spaces-entry.js"]
	Check(t, !present, "retired entry in manifest")
}

func TestSodaspacesVMWebTunnelUsesOnlyTheNativeBrowserOrigin(t *testing.T) {
	dir := TempDir(t)
	ssh := filepath.Join(dir, "ssh")
	WriteFile(t, ssh, []byte("#!/bin/sh\nprintf \"%s\\n\" \"$@\"\n"), 0o755)
	env := SetEnv(os.Environ(), "PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	result := Run(t, RunOpt{Env: env, Timeout: 10 * time.Second}, testVMBinary(t), "web-tunnel")
	Require(t, result.Code == 0, "web-tunnel failed: %s", result.Stderr)
	Check(t, strings.Contains(result.Stdout, "Forgejo + Sodaspaces https://localhost:24444"), "stdout=%q", result.Stdout)
	Check(t, strings.Contains(result.Stdout, "127.0.0.1:24444:127.0.0.1:24444"), "stdout=%q", result.Stdout)
	Check(t, !strings.Contains(result.Stdout, "24443"), "stdout=%q", result.Stdout)
}

func TestSodaspacesAccessProbeRejectsPrivateBadRequestBeforeNativeCommands(t *testing.T) {
	dir := TempDir(t)
	Require(t, os.Chmod(dir, 0o700) == nil, "chmod root")
	request := filepath.Join(dir, "target.json")
	WriteFile(t, request, []byte(`{"SYNTHETIC_PRIVATE_MARKER":true}`), 0o600)
	result := Run(t, RunOpt{Cwd: RepoRoot, Timeout: 180 * time.Second},
		"go", "run", "./tools/soda-installed-probes", "developer-access", dir)
	Check(t, result.Code == 1, "exit = %d", result.Code)
	Check(t, strings.Contains(result.Stderr, "Developer access incomplete"), "stderr=%q", result.Stderr)
	Check(t, !strings.Contains(result.Stdout+result.Stderr, "SYNTHETIC_PRIVATE_MARKER"), "marker leaked")
	entries, err := os.ReadDir(dir)
	Require(t, err == nil, "read dir: %v", err)
	Require(t, len(entries) == 1 && entries[0].Name() == "target.json", "dir changed: %v", entries)
}

func copyTree(t *testing.T, src, dest string) {
	t.Helper()
	Require(t, os.MkdirAll(dest, 0o755) == nil, "mkdir %s", dest)
	entries, err := os.ReadDir(src)
	Require(t, err == nil, "read %s: %v", src, err)
	for _, entry := range entries {
		from, to := filepath.Join(src, entry.Name()), filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			copyTree(t, from, to)
			continue
		}
		data, err := os.ReadFile(from)
		Require(t, err == nil, "read %s: %v", from, err)
		info, err := entry.Info()
		Require(t, err == nil, "stat %s: %v", from, err)
		WriteFile(t, to, data, info.Mode().Perm())
	}
}

func chmodTree(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if entry.IsDir() {
			mode = 0o700
		}
		return os.Chmod(path, mode)
	})
	Require(t, err == nil, "chmod tree: %v", err)
}

func checkMode(t *testing.T, path string, dir bool) {
	t.Helper()
	want := os.FileMode(0o644)
	if dir {
		want = 0o755
	}
	Check(t, fileMode(t, path) == want, "%s mode = %o, want %o", path, fileMode(t, path), want)
}

func TestSodaspacesActualStageRecipeWithSyntheticBuildInputs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("soda-stage requires native Linux x86_64")
	}
	stage := stageBinary(t)
	tmp := TempDir(t)
	checkout, err := filepath.EvalSymlinks(tmp)
	Require(t, err == nil, "resolve tempdir: %v", err)
	copyTree(t, filepath.Join(RepoRoot, "assets"), filepath.Join(checkout, "assets"))
	chmodTree(t, filepath.Join(checkout, "assets"))
	copyTree(t, filepath.Join(RepoRoot, "appliance"), filepath.Join(checkout, "appliance"))
	copyTree(t, filepath.Join(RepoRoot, "frontend"), filepath.Join(checkout, "frontend"))
	copyTree(t, filepath.Join(RepoRoot, "system", "host", "config"), filepath.Join(checkout, "system/host/config"))
	copyTree(t, filepath.Join(RepoRoot, "system", "licenses"), filepath.Join(checkout, "system/licenses"))
	// The branding payload manifest rides along inside the copied assets/.
	payloadSrc, err := os.ReadFile(filepath.Join(RepoRoot, "assets/branding/forgejo/forgejo-payload.json"))
	Require(t, err == nil, "read payload: %v", err)
	for _, name := range []string{"LICENSE", "NOTICE"} {
		data, err := os.ReadFile(filepath.Join(RepoRoot, name))
		Require(t, err == nil, "read %s: %v", name, err)
		WriteFile(t, filepath.Join(checkout, name), data, 0o644)
	}
	build := filepath.Join(checkout, ".artifacts/native/x86_64")
	Require(t, os.MkdirAll(filepath.Join(build, "project-tools/bin"), 0o755) == nil, "mkdir tools")
	for _, name := range []string{"muse", "muse-native", "soda-identity-compose"} {
		WriteFile(t, filepath.Join(build, "project-tools/bin", name), []byte("synthetic; never executed"), 0o644)
	}
	Require(t, os.Mkdir(filepath.Join(build, "forgejo-js"), 0o755) == nil, "mkdir forgejo-js")
	var manifest map[string]string
	Require(t, json.Unmarshal(payloadSrc, &manifest) == nil, "parse payload")
	for _, origin := range manifest {
		if rest, ok := strings.CutPrefix(origin, "@build/forgejo-js/"); ok {
			_ = rest
			WriteFile(t, filepath.Join(build, strings.TrimPrefix(origin, "@build/")), []byte("// synthetic compiled browser fixture\n"), 0o644)
		}
	}
	Require(t, os.Mkdir(filepath.Join(build, "forgejo-locales"), 0o755) == nil, "mkdir locales")
	WriteFile(t, filepath.Join(build, "forgejo-locales/locale_en-US.ini"), []byte("synthetic full-catalog output; not native proof"), 0o644)
	Require(t, os.Mkdir(filepath.Join(build, "terminal-assets"), 0o755) == nil, "mkdir terminal-assets")
	lockSrc, err := os.ReadFile(filepath.Join(RepoRoot, "tools/release-assets/terminal-assets.lock.json"))
	Require(t, err == nil, "read lock: %v", err)
	lockPath := filepath.Join(checkout, "tools/release-assets/terminal-assets.lock.json")
	WriteFile(t, lockPath, lockSrc, 0o644)
	lockData, err := os.ReadFile(lockPath)
	Require(t, err == nil, "read lock: %v", err)
	var lock []map[string]any
	Require(t, json.Unmarshal(lockData, &lock) == nil, "parse lock")
	for _, item := range lock {
		for _, f := range item["files"].([]any) {
			asset := f.(map[string]any)
			data := []byte("synthetic asset " + asset["file"].(string))
			WriteFile(t, filepath.Join(build, "terminal-assets", asset["file"].(string)), data, 0o644)
			sum := sha256.Sum256(data)
			asset["sha256"] = hex.EncodeToString(sum[:])
		}
	}
	encoded, err := json.Marshal(lock)
	Require(t, err == nil, "encode lock: %v", err)
	WriteFile(t, lockPath, encoded, 0o644)
	for _, page := range []string{"tailscale", "runners"} {
		WriteFile(t, filepath.Join(checkout, "cockpit/dist", "soda-"+page, "index.html"), []byte("synthetic Cockpit package"), 0o644)
	}
	host := filepath.Join(checkout, "host-context")
	marker := filepath.Join(host, "rootfs/usr/libexec/soda/soda-dashboard")
	WriteFile(t, marker, []byte("already compiled program; never executed"), 0o644)
	forgejoContext := filepath.Join(checkout, "forgejo-context")
	Require(t, os.Mkdir(forgejoContext, 0o700) == nil, "mkdir forgejo-context")
	Require(t, os.Chmod(host, 0o700) == nil, "chmod host")
	vendorRoot := filepath.Join(host, "rootfs")

	vendorStage := func(t *testing.T, argv ...string) ProcResult {
		t.Helper()
		if argv == nil {
			argv = []string{"--arch", "x86_64", "--host-context", host, "--forgejo-context", forgejoContext}
		}
		previous := syscall.Umask(0o077)
		defer syscall.Umask(previous)
		return Run(t, RunOpt{Cwd: checkout, Timeout: 180 * time.Second}, stage, argv...)
	}

	first := vendorStage(t, nil...)
	Require(t, first.Code == 0, "stage failed: %s", tail(first.Stderr, 2000))
	Check(t, strings.TrimSpace(first.Stdout) == vendorRoot, "stdout=%q", first.Stdout)
	for _, absent := range []string{
		filepath.Join(build, "rootfs"),
		filepath.Join(vendorRoot, "var"),
		filepath.Join(vendorRoot, "usr/local"),
	} {
		_, err := os.Stat(absent)
		Check(t, os.IsNotExist(err), "%s exists", absent)
	}
	override := filepath.Join(vendorRoot, "etc/cockpit/users.override.json")
	_, err = os.Lstat(override)
	Check(t, os.IsNotExist(err), "override exists")
	markerText, err := os.ReadFile(marker)
	Require(t, err == nil, "read marker: %v", err)
	Check(t, string(markerText) == "already compiled program; never executed", "marker changed")
	for _, dir := range []string{host, forgejoContext} {
		Check(t, fileMode(t, dir) == 0o700, "%s mode changed", dir)
	}
	presentation := filepath.Join(forgejoContext, "forgejo")
	original := func(origin string) string {
		if rest, ok := strings.CutPrefix(origin, "@build/"); ok {
			return filepath.Join(build, rest)
		}
		return filepath.Join(checkout, origin)
	}
	err = filepath.WalkDir(presentation, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		checkMode(t, path, entry.IsDir())
		return nil
	})
	Require(t, err == nil, "walk presentation: %v", err)
	checkMode(t, presentation, true)
	for _, name := range sodaspacesFiles {
		want, err := os.ReadFile(original(manifest[name]))
		Require(t, err == nil, "read origin %s: %v", name, err)
		got, err := os.ReadFile(filepath.Join(presentation, name))
		Require(t, err == nil, "read staged %s: %v", name, err)
		Check(t, string(got) == string(want), "staged %s differs", name)
	}
	for name, origin := range manifest {
		staged := filepath.Join(presentation, name)
		want, err := os.ReadFile(original(origin))
		Require(t, err == nil, "read origin %s: %v", name, err)
		got, err := os.ReadFile(staged)
		Require(t, err == nil, "read staged %s: %v", name, err)
		Check(t, string(got) == string(want), "staged %s differs", name)
		checkMode(t, staged, false)
		for parent := filepath.Dir(staged); parent != presentation; parent = filepath.Dir(parent) {
			checkMode(t, parent, true)
		}
	}
	for _, theme := range []string{"dark", "light"} {
		source, err := os.ReadFile(filepath.Join(checkout, "assets/branding/forgejo/css/theme-soda-"+theme+".css"))
		Require(t, err == nil, "read theme source: %v", err)
		Check(t, strings.Contains(string(source), `@import "../../../css/theme-forgejo-`+theme+`.css";`), "source import changed")
		staged, err := os.ReadFile(filepath.Join(presentation, "public/assets/css/theme-soda-"+theme+".css"))
		Require(t, err == nil, "read staged theme: %v", err)
		Check(t, strings.Contains(string(staged), `@import "theme-forgejo-`+theme+`.css";`), "staged import wrong")
		Check(t, !strings.Contains(string(staged), "css/theme-forgejo-"), "staged import not rewritten")
	}
	brand := filepath.Join(vendorRoot, "etc/cockpit/branding")
	compareBytes := func(t *testing.T, got, want string) {
		t.Helper()
		a, err := os.ReadFile(got)
		Require(t, err == nil, "read %s: %v", got, err)
		b, err := os.ReadFile(want)
		Require(t, err == nil, "read %s: %v", want, err)
		Check(t, string(a) == string(b), "%s differs", got)
	}
	compareBytes(t, filepath.Join(brand, "soda-symbol-brutalist.svg"), filepath.Join(RepoRoot, "assets/branding/source/soda-symbol-brutalist.svg"))
	_, err = os.Stat(filepath.Join(brand, "fonts/barlow-condensed/LICENSE"))
	Check(t, err == nil, "barlow license missing")
	compareBytes(t, filepath.Join(brand, "apple-touch-icon.png"), filepath.Join(RepoRoot, "assets/branding/forgejo/apple-touch-icon.png"))
	icon, err := os.ReadFile(filepath.Join(brand, "favicon.ico"))
	Require(t, err == nil, "read favicon: %v", err)
	Check(t, binary.LittleEndian.Uint16(icon[0:]) == 0 && binary.LittleEndian.Uint16(icon[2:]) == 1 && binary.LittleEndian.Uint16(icon[4:]) == 2,
		"favicon header wrong")
	for i, entry := range [][2]any{{16, "favicon-16.png"}, {32, "favicon.png"}} {
		size := entry[0].(int)
		name := entry[1].(string)
		off := 6 + 16*i
		w, h, colors, reserved := icon[off], icon[off+1], icon[off+2], icon[off+3]
		planes, bits := binary.LittleEndian.Uint16(icon[off+4:]), binary.LittleEndian.Uint16(icon[off+6:])
		length, offset := binary.LittleEndian.Uint32(icon[off+8:]), binary.LittleEndian.Uint32(icon[off+12:])
		Check(t, w == byte(size) && h == byte(size) && colors == 0 && reserved == 0 && planes == 1 && bits == 32,
			"favicon entry %d wrong", i)
		want, err := os.ReadFile(filepath.Join(RepoRoot, "assets/branding/forgejo", name))
		Require(t, err == nil, "read %s: %v", name, err)
		Check(t, string(icon[offset:offset+length]) == string(want), "favicon %s differs", name)
	}
	for _, absent := range []string{"login-background-light.svg", "soda-symbol.svg"} {
		_, err := os.Stat(filepath.Join(brand, absent))
		Check(t, os.IsNotExist(err), "%s exists", absent)
	}
	err = filepath.WalkDir(brand, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		checkMode(t, path, entry.IsDir())
		return nil
	})
	Require(t, err == nil, "walk branding: %v", err)
	compareBytes(t, filepath.Join(vendorRoot, "usr/share/soda/fastfetch/sodaos.txt"), filepath.Join(RepoRoot, "assets/branding/terminal/sodaos.txt"))
	fastfetch, err := os.ReadFile(filepath.Join(vendorRoot, "etc/fastfetch/config.jsonc"))
	Require(t, err == nil, "read fastfetch config: %v", err)
	Check(t, strings.Contains(string(fastfetch), "/usr/share/soda/fastfetch/"), "fastfetch path missing")
	Check(t, fileMode(t, filepath.Join(vendorRoot, "etc/soda/forgejo.env")) == 0o600, "forgejo.env mode wrong")
	for _, tool := range []string{"muse", "muse-native", "soda-identity-compose"} {
		Check(t, fileMode(t, filepath.Join(vendorRoot, "usr/share/soda/muse-tools", tool)) == 0o755, "%s mode wrong", tool)
	}
	err = filepath.WalkDir(filepath.Join(presentation, "public/assets"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		checkMode(t, path, entry.IsDir())
		rel, err := filepath.Rel(presentation, path)
		Require(t, err == nil, "rel: %v", err)
		for _, part := range strings.Split(rel, string(os.PathSeparator)) {
			Check(t, part != "soda-runners" && part != "soda-tailscale", "retired package in %s", path)
		}
		return nil
	})
	Require(t, err == nil, "walk assets: %v", err)

	inventory := func(t *testing.T, root string) map[string]string {
		t.Helper()
		out := map[string]string{}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out[rel] = string(data)
			return nil
		})
		Require(t, err == nil, "inventory %s: %v", root, err)
		return out
	}
	beforeVendor, beforePresentation := inventory(t, vendorRoot), inventory(t, presentation)
	rerun := vendorStage(t, nil...)
	Check(t, rerun.Code == 2, "rerun exit = %d: %s", rerun.Code, tail(rerun.Stderr, 2000))
	Check(t, strings.Contains(rerun.Stderr, "occupied Forgejo presentation refused"), "stderr=%q", rerun.Stderr)
	afterVendor, afterPresentation := inventory(t, vendorRoot), inventory(t, presentation)
	Check(t, len(afterVendor) == len(beforeVendor) && len(afterPresentation) == len(beforePresentation), "rerun changed inventory")
	for key, value := range beforeVendor {
		Check(t, afterVendor[key] == value, "vendor %s changed", key)
	}
	for key, value := range beforePresentation {
		Check(t, afterPresentation[key] == value, "presentation %s changed", key)
	}
	missing := vendorStage(t, "--arch", "x86_64", "--host-context", host)
	Check(t, missing.Code == 2, "missing exit = %d: %s", missing.Code, tail(missing.Stderr, 2000))
	Check(t, strings.Contains(missing.Stderr, "the following arguments are required: --forgejo-context"), "stderr=%q", missing.Stderr)
	link := filepath.Join(checkout, "linked-forgejo-context")
	Require(t, os.Symlink(forgejoContext, link) == nil, "symlink context")
	linked := vendorStage(t, "--arch", "x86_64", "--host-context", host, "--forgejo-context", link)
	Check(t, linked.Code == 2, "linked exit = %d: %s", linked.Code, tail(linked.Stderr, 2000))
	Check(t, strings.Contains(linked.Stderr, "real prepared host and fresh Forgejo context directories required"), "stderr=%q", linked.Stderr)
	Check(t, fileMode(t, filepath.Join(checkout, "assets/branding/source/soda-symbol-brutalist.svg")) == 0o600, "source mode normalized")
}
