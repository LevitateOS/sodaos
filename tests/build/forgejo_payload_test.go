// Port of test_forgejo_payload.py: exact presentation payload manifest.
package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func payloadFiles(t *testing.T) map[string]string {
	t.Helper()
	raw, ok := ReadJSON(t, "frontend/forgejo/payload.json").(map[string]any)
	Require(t, ok, "payload manifest is not an object")
	files := make(map[string]string, len(raw))
	for dest, source := range raw {
		text, ok := source.(string)
		Require(t, ok, "payload origin for %s is not a string", dest)
		files[dest] = text
	}
	return files
}

func TestForgejoPayloadExactSourcesTemplateClosureAndNotices(t *testing.T) {
	files := payloadFiles(t)
	templates := map[string]bool{}
	root := filepath.Join(RepoRoot, "frontend/forgejo/templates")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			templates["templates/"+filepath.ToSlash(rel)] = true
		}
		return nil
	})
	Require(t, err == nil, "walk templates: %v", err)
	manifest := map[string]bool{}
	for name := range files {
		if strings.HasPrefix(name, "templates/") {
			manifest[name] = true
		}
	}
	Require(t, len(templates) == len(manifest), "template closure differs: disk=%d manifest=%d", len(templates), len(manifest))
	for name := range templates {
		Check(t, manifest[name], "template %s missing from manifest", name)
	}
	templateRef := regexp.MustCompile(`\{\{-?\s*template\s+"(custom/soda/[^"]+)"`)
	for dest, source := range files {
		for _, part := range strings.Split(dest, "/") {
			Check(t, part != "..", "dest %s escapes", dest)
		}
		Check(t, !filepath.IsAbs(dest), "dest %s is absolute", dest)
		for _, part := range strings.Split(source, "/") {
			Check(t, part != "..", "source %s escapes", source)
		}
		if strings.HasPrefix(source, "@build/") {
			Check(t, strings.HasPrefix(source, "@build/terminal-assets/") ||
				strings.HasPrefix(source, "@build/forgejo-locales/") ||
				strings.HasPrefix(source, "@build/forgejo-js/"),
				"unexpected build origin %s", source)
			continue
		}
		full := filepath.Join(RepoRoot, source)
		info, err := os.Stat(full)
		Require(t, err == nil && !info.IsDir(), "source %s is not a file", source)
		link, err := os.Lstat(full)
		Require(t, err == nil, "lstat %s: %v", source, err)
		Check(t, link.Mode()&os.ModeSymlink == 0, "source %s is a symlink", source)
		if strings.HasSuffix(full, ".tmpl") {
			data, err := os.ReadFile(full)
			Require(t, err == nil, "read %s: %v", source, err)
			for _, match := range templateRef.FindAllStringSubmatch(string(data), -1) {
				_, present := files["templates/"+match[1]+".tmpl"]
				Check(t, present, "template ref %s missing from manifest", match[1])
			}
		}
	}
	for _, font := range []string{"barlow", "fraunces", "ibm-plex-mono"} {
		_, present := files["public/assets/soda/fonts/"+font+"/LICENSE"]
		Check(t, present, "font %s LICENSE missing", font)
	}
	_, present := files["public/assets/soda/forgejo/forgejo-LICENSE"]
	Check(t, present, "forgejo LICENSE missing")
	Check(t, files["public/assets/soda/forgejo/lit.js"] == "@build/forgejo-js/lit.js", "lit.js origin changed")
	Check(t, files["public/assets/soda/forgejo/lit.LICENSE"] == "system/licenses/lit-LICENSE", "lit license origin changed")
	_, present = files["options/locale/locale_en-US.ini"]
	Check(t, present, "en-US locale missing")
}

func extensionPages(t *testing.T, manifest string) []any {
	t.Helper()
	raw, ok := ReadJSON(t, manifest).(map[string]any)
	Require(t, ok, "%s is not an object", manifest)
	pages, ok := raw["pages"].([]any)
	Require(t, ok, "%s pages is not a list", manifest)
	return pages
}

// The required soda package carries only the spaces page; the tailnet
// page lives in the separately activatable optional package under its
// admin-gated declaration, never in the core manifest.
func TestForgejoPayloadExtensionPagesDeclaredInNativePackage(t *testing.T) {
	ids := map[string]bool{}
	for _, entry := range extensionPages(t, "system/containers/extension/extension.json") {
		ids[entry.(map[string]any)["id"].(string)] = true
	}
	Check(t, len(ids) == 1 && ids["spaces"], "core page ids = %v", ids)
	optional := map[string]any{}
	for _, entry := range extensionPages(t, "system/containers/extension/tailnet-package/extension.json") {
		page := entry.(map[string]any)
		optional[page["id"].(string)] = page
	}
	tailnet, ok := optional["tailnet"].(map[string]any)
	Require(t, ok && len(optional) == 1, "optional pages = %v", optional)
	Check(t, tailnet["scope"] == "admin" && tailnet["route"] == "", "optional tailnet page = %v", tailnet)
	if _, gated := tailnet["permission"]; !gated {
		t.Fatalf("optional tailnet page carries no permission gate: %v", tailnet)
	}
}

func TestForgejoPayloadOperatorSettingsUseDeclaredTailnetRoute(t *testing.T) {
	declared := map[string]bool{}
	for _, entry := range extensionPages(t, "system/containers/extension/tailnet-package/extension.json") {
		page := entry.(map[string]any)
		if page["id"] == "tailnet" && page["scope"] == "admin" && page["route"] == "" {
			_, hasPermission := page["permission"]
			declared["tailnet"] = hasPermission
		}
	}
	Check(t, len(declared) == 1 && declared["tailnet"], "tailnet page declaration = %v", declared)
}
