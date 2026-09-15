// soda-avatars is a development-only catalog generator. Its output is never
// staged on the appliance; the SVG renderer is shared with the production API.
package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	dicebear "github.com/dicebear/dicebear-go/v10"
	"github.com/levitateos/sodaos/internal/avatar"
)

type (
	card    struct{ Label, File string }
	section struct {
		ID, Title string
		Size      int
		Cards     []card
	}
)
type sheet struct{ Sections []section }

func main() {
	name, err := generate(".artifacts/avatars")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(name)
}

func avatarBaseline(definition struct {
	Components map[string]struct{ Variants map[string]json.RawMessage }
	Colors     map[string]struct{ Values []string }
},
) func() map[string]any {
	return func() map[string]any {
		return map[string]any{
			"seed": "soda-robot-v1:00000000000000000000000000000000", "size": 128,
			"headVariant": []string{"wide"}, "faceVariant": []string{"rounded"},
			"eyesVariant": []string{"ovals"}, "mouthVariant": []string{"smile"},
			"earcapsVariant": []string{"round"}, "accessoryVariant": []string{"plain"},
			"backgroundColor": []string{definition.Colors["background"].Values[0]},
			"accentColor":     []string{definition.Colors["accent"].Values[0]},
		}
	}
}

func writeAvatarSVG(dir, name, svg string) error {
	return os.WriteFile(filepath.Join(dir, "svg", name+".svg"), []byte(svg), 0o644)
}

func variantSection(s *dicebear.Style, definition struct {
	Components map[string]struct{ Variants map[string]json.RawMessage }
	Colors     map[string]struct{ Values []string }
}, dir, name string, baseline func() map[string]any,
) (section, error) {
	part := section{ID: name, Title: name, Size: 128}
	keys := []string{}
	for key := range definition.Components[name].Variants {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		opts := baseline()
		opts[name+"Variant"] = []string{key}
		a, err := dicebear.NewAvatar(s, opts)
		if err != nil {
			return part, err
		}
		file := name + "-" + key
		if err := writeAvatarSVG(dir, file, a.SVG()); err != nil {
			return part, err
		}
		part.Cards = append(part.Cards, card{key, "svg/" + file + ".svg"})
	}
	return part, nil
}

func colorSection(s *dicebear.Style, definition struct {
	Components map[string]struct{ Variants map[string]json.RawMessage }
	Colors     map[string]struct{ Values []string }
}, dir string, baseline func() map[string]any,
) (section, error) {
	colors := section{ID: "colors", Title: "Every background / accent pair", Size: 128}
	for i, bg := range definition.Colors["background"].Values {
		for j, accent := range definition.Colors["accent"].Values {
			opts := baseline()
			opts["backgroundColor"], opts["accentColor"] = []string{bg}, []string{accent}
			a, err := dicebear.NewAvatar(s, opts)
			if err != nil {
				return colors, err
			}
			file := fmt.Sprintf("palette-%d-%d", i, j)
			if err := writeAvatarSVG(dir, file, a.SVG()); err != nil {
				return colors, err
			}
			colors.Cards = append(colors.Cards, card{bg + " / " + accent, "svg/" + file + ".svg"})
		}
	}
	return colors, nil
}

func robotSamples(dir string) ([]card, error) {
	samples := []card{}
	for i := range 100 {
		hash := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("soda-avatar-preview-%03d", i))))
		svg, err := avatar.Render(hash, 128)
		if err != nil {
			return nil, err
		}
		file := fmt.Sprintf("robot-%03d", i)
		if err := writeAvatarSVG(dir, file, svg); err != nil {
			return nil, err
		}
		samples = append(samples, card{fmt.Sprintf("%03d", i), "svg/" + file + ".svg"})
	}
	return samples, nil
}

func writeAvatarIndex(dir string, data sheet) error {
	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}
	if err := template.Must(template.New("sheet").Parse(page)).Execute(f, data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func catalogAvatarSheet(s *dicebear.Style, definition struct {
	Components map[string]struct{ Variants map[string]json.RawMessage }
	Colors     map[string]struct{ Values []string }
}, dir string,
) (sheet, error) {
	baseline := avatarBaseline(definition)
	data := sheet{}
	for _, name := range []string{"head", "face", "eyes", "mouth", "earcaps", "accessory"} {
		part, err := variantSection(s, definition, dir, name, baseline)
		if err != nil {
			return data, err
		}
		data.Sections = append(data.Sections, part)
	}
	colors, err := colorSection(s, definition, dir, baseline)
	if err != nil {
		return data, err
	}
	data.Sections = append(data.Sections, colors)
	samples, err := robotSamples(dir)
	if err != nil {
		return data, err
	}
	for _, size := range []int{128, 64, 32, 24} {
		data.Sections = append(data.Sections, section{fmt.Sprintf("grid-%d", size), fmt.Sprintf("100 stable robots · %dpx", size), size, samples})
	}
	return data, nil
}

func prepareAvatarWorkspace(parent string) (string, error) {
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(parent, "preview-")
	if err != nil {
		return "", err
	}
	return dir, os.Mkdir(filepath.Join(dir, "svg"), 0o755)
}

func generate(parent string) (string, error) {
	if err := avatar.Validate(); err != nil {
		return "", err
	}
	s, err := dicebear.NewStyle([]byte(avatar.Definition()))
	if err != nil {
		return "", err
	}
	var definition struct {
		Components map[string]struct{ Variants map[string]json.RawMessage }
		Colors     map[string]struct{ Values []string }
	}
	if err := json.Unmarshal([]byte(avatar.Definition()), &definition); err != nil {
		return "", err
	}
	dir, err := prepareAvatarWorkspace(parent)
	if err != nil {
		return "", err
	}
	data, err := catalogAvatarSheet(s, definition, dir)
	if err != nil {
		return "", err
	}
	if err := writeAvatarIndex(dir, data); err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(dir, "index.html"))
}

const page = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Soda robot v1 · artwork inspection</title>
<style>
:root{font:15px system-ui;color:#172554;background:#fffdf8;color-scheme:light}
body{margin:0;padding:32px;max-width:1500px;margin-inline:auto}
h1{font-size:32px;margin-bottom:8px}p{max-width:85ch;line-height:1.5}nav{display:flex;gap:16px;flex-wrap:wrap}a{color:inherit}
.controls{background:inherit;padding:12px 0;display:flex;gap:24px;border-bottom:1px solid #8f857d}
body.dark{background:#141a24;color:#f0f4f8;color-scheme:dark}html:has(body.dark){background:#141a24}
section{padding-block:16px;scroll-margin-top:70px}h2{text-transform:capitalize}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(calc(var(--size)*1px + 32px),1fr));gap:16px}
.card{display:flex;flex-direction:column;align-items:center;gap:8px;text-align:center;text-decoration:none;min-width:0;font-size:11px}
.card img{width:calc(var(--size)*1px);height:calc(var(--size)*1px);display:block}
body.circle img{border-radius:50%}.card:focus-visible{outline:3px solid #155eef}
</style>
<h1>Soda robot v1</h1><p>Original procedural artwork · inspection preview, not an installed Forgejo screenshot. The samples use the production Go renderer. Select an image to download its SVG.</p>
<nav>{{range .Sections}}<a href="#{{.ID}}">{{.Title}}</a>{{end}}</nav>
<div class="controls"><label><input id="dark" type="checkbox"> Dark surface</label><label><input id="circle" type="checkbox"> Circular crop</label></div>
{{range .Sections}}<section id="{{.ID}}"><h2>{{.Title}}</h2><div class="grid" style="--size:{{.Size}}">{{range .Cards}}<a class="card" href="{{.File}}" download><img src="{{.File}}" alt="{{.Label}}" loading="eager"><span>{{.Label}}</span></a>{{end}}</div></section>{{end}}
<script>for(const id of ['dark','circle'])document.getElementById(id).addEventListener('change',e=>document.body.classList.toggle(id,e.target.checked));</script>
</html>`
