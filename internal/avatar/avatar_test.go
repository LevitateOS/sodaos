package avatar

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

type snapshot struct {
	Hash   string
	Size   int
	SHA256 string
}

func snapshots(t *testing.T) []snapshot {
	t.Helper()
	b, err := os.ReadFile("testdata/v1-snapshots.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []snapshot
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

func TestVersionedSnapshots(t *testing.T) {
	for _, c := range snapshots(t) {
		svg, err := Render(c.Hash, c.Size)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(svg))); got != c.SHA256 {
			t.Fatalf("v1 changed for %s: %s; inspect artwork/version before changing snapshots", c.Hash, got)
		}
		upper, err := Render(strings.ToUpper(c.Hash), c.Size)
		if err != nil || upper != svg {
			t.Fatal("hash normalization changed the avatar", err)
		}
	}
}

func TestConcurrentRendering(t *testing.T) {
	cases := snapshots(t)
	var wg sync.WaitGroup
	for i := range 48 {
		wg.Add(1)
		go func(c snapshot) {
			defer wg.Done()
			svg, err := Render(c.Hash, c.Size)
			if err != nil {
				t.Error(err)
				return
			}
			if fmt.Sprintf("%x", sha256.Sum256([]byte(svg))) != c.SHA256 {
				t.Error("concurrent rendering changed output")
			}
		}(cases[i%len(cases)])
	}
	wg.Wait()
}

func TestRenderProcess(t *testing.T) {
	if os.Getenv("SODA_AVATAR_PROCESS") != "1" {
		return
	}
	svg, err := Render("0123456789abcdef0123456789abcdef", 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%x", sha256.Sum256([]byte(svg)))
	os.Exit(0)
}

func TestStableAcrossProcesses(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var previous string
	for range 2 {
		cmd := exec.Command(exe, "-test.run=^TestRenderProcess$")
		cmd.Env = append(os.Environ(), "SODA_AVATAR_PROCESS=1")
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 64 {
			t.Fatalf("unexpected child output: %q", out)
		}
		if previous != "" && previous != string(out) {
			t.Fatal("avatar changed on process restart")
		}
		previous = string(out)
	}
}

func TestDefinitionInventory(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	var d struct {
		Components map[string]struct{ Variants map[string]json.RawMessage }
		Colors     map[string]struct{ Values []string }
	}
	if err := json.Unmarshal([]byte(Definition()), &d); err != nil {
		t.Fatal(err)
	}
	for name, count := range map[string]int{"head": 6, "face": 6, "eyes": 10, "mouth": 8, "earcaps": 6, "accessory": 8} {
		if got := len(d.Components[name].Variants); got != count {
			t.Errorf("%s: got %d variants, want %d", name, got, count)
		}
	}
	if len(d.Colors["background"].Values) != 8 || len(d.Colors["accent"].Values) != 4 {
		t.Fatal("v1 palette changed")
	}
}

type rejectNetwork struct{ t *testing.T }

func (n rejectNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	n.t.Error("renderer attempted an outbound HTTP request")
	return nil, fmt.Errorf("network disabled")
}

func TestOfflineSVGContent(t *testing.T) {
	old := http.DefaultTransport
	http.DefaultTransport = rejectNetwork{t}
	t.Cleanup(func() { http.DefaultTransport = old })
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for i := range 100 {
		hash := fmt.Sprintf("%032x", i)
		svg, err := Render(hash, 32)
		if err != nil {
			t.Fatal(err)
		}
		seen[svg] = true
		if strings.Contains(svg, hash) {
			t.Fatal("raw seed exposed in SVG")
		}
		d := xml.NewDecoder(strings.NewReader(svg))
		for {
			token, err := d.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			e, ok := token.(xml.StartElement)
			if !ok {
				continue
			}
			switch e.Name.Local {
			case "script", "foreignObject", "image", "style", "text", "animate", "set":
				t.Fatalf("unexpected SVG element %s", e.Name.Local)
			}
			for _, a := range e.Attr {
				if strings.HasPrefix(strings.ToLower(a.Name.Local), "on") || (a.Name.Local == "href" && !strings.HasPrefix(a.Value, "#")) {
					t.Fatal("active SVG attribute", a)
				}
			}
		}
	}
	if len(seen) < 95 {
		t.Fatalf("unexpectedly low sample diversity: %d/100", len(seen))
	}
}

func TestInputBounds(t *testing.T) {
	for _, hash := range []string{"", "alice@example.test", strings.Repeat("g", 32), strings.Repeat("0", 31), strings.Repeat("0", 33)} {
		if _, err := Render(hash, 128); err == nil {
			t.Errorf("accepted hash %q", hash)
		}
	}
	for _, size := range []int{-1, 0, 1025} {
		if _, err := Render(strings.Repeat("a", 32), size); err == nil {
			t.Errorf("accepted size %d", size)
		}
	}
	for _, size := range []int{1, 24, 128, 1024} {
		svg, err := Render(strings.Repeat("a", 32), size)
		if err != nil {
			t.Fatal(err)
		}
		for _, attr := range []string{"width", "height"} {
			if !strings.Contains(svg, fmt.Sprintf(`%s="%d"`, attr, size)) {
				t.Errorf("missing %s=%d", attr, size)
			}
		}
	}
}
