package scripts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoSecondaryAuthPagesOnlyAddPresentationRoot(t *testing.T) {
	upstreamHashes := map[string]string{
		"signup.tmpl":         "794d0f90f6a389616f8d0a1bbbf4a0f6c60cbc1727da5e26ff2ac969506e44ba",
		"forgot_passwd.tmpl":  "17a6eafe32e3c98d84c11ccb2a41dc3d1b504f7a5c51043ef217eea020e7c1fd",
		"reset_passwd.tmpl":   "bdd283d8dbb3923afeba119886ac9ee5a44339a9cfe304c9e0883cb41ed02e1e",
		"change_passwd.tmpl":  "3bacab8652ba773ce17bf2d89eb9e9135609940bfdc77b891ba3e8157dd10fc3",
		"twofa.tmpl":          "b7d8ba06a6ea7e9befd28fb2cc3794b6aef30a8f2b6197938cc905c437ddb1e0",
		"twofa_scratch.tmpl":  "cce435f8cba329d3e739f2e3e856138b50780183c1d9540399b141480341e9ee",
		"webauthn.tmpl":       "469c1cb9ef178590b0dcdc6240e3e6ead85a1b89efe76cfd454242578236bea9",
		"activate.tmpl":       "b639300f2bd1ff6bbeb04aa8967dd718f0236d0b23c3facc64eb3eb2d055d3bd",
		"prohibit_login.tmpl": "4b95c0a51eddf76bfc15c42823dbc566188f0c8ff9b6f436cc83b06cc6621e5a",
	}
	const presentationClasses = "class=\"page-content soda-page soda-auth soda-native-forms "
	const stockClasses = "class=\"page-content "
	const signedAttribute = ` data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`

	for name, upstreamHash := range upstreamHashes {
		t.Run(name, func(t *testing.T) {
			page := readForgejoTemplate(t, "user", "auth", name)
			if strings.Count(page, presentationClasses) != 1 {
				t.Fatalf("%s must have exactly one Soda auth root", name)
			}
			if strings.Count(page, signedAttribute) != 1 {
				t.Fatalf("%s must derive signed state exactly once", name)
			}
			stock := strings.Replace(page, presentationClasses, stockClasses, 1)
			stock = strings.Replace(stock, signedAttribute, "", 1)
			if name == "prohibit_login.tmpl" {
				stock = strings.Replace(stock, `
	{{if not .IsSigned}}{{template "custom/soda/theme_toggle" dict}}{{end}}`, "", 1)
			}
			if name == "signup.tmpl" {
				stock = strings.Replace(stock, `			{{if and (not .DisableRegistration) (not .LinkAccountMode)}}<img class="tw-mx-auto" src="{{AssetUrlPrefix}}/soda/forgejo/signup-papercraft.png" width="180" height="120" alt="">{{end}}`+"\n", "", 1)
			}
			actualHash := fmt.Sprintf("%x", sha256.Sum256([]byte(stock)))
			if actualHash != upstreamHash {
				t.Errorf("%s diverges from pristine Forgejo 15.0.7 after removing presentation attributes: got %s, want %s", name, actualHash, upstreamHash)
			}
		})
	}
}

func TestForgejoSecondaryAuthStylesRemainPageScoped(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "auth.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	if !strings.Contains(css, ".soda-auth") {
		t.Fatal("secondary auth stylesheet is missing its page root")
	}
	for _, forbidden := range []string{"body ", "#navbar", ".soda-page {", ".soda-form"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("secondary auth stylesheet reaches outside its page contract with %q", forbidden)
		}
	}
}
