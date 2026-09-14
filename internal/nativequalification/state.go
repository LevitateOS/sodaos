package nativequalification

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/projectos"
	"github.com/levitateos/sodaos/internal/store"
)

const fixtureState = "/var/lib/soda-qualification"
const fixtureLogin = "soda-tester"
const fixtureProject = "p000000000000000000000001"

func checkStateSchema(ctx context.Context, path string) error {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	var count, minimum, maximum int
	if err = db.QueryRowContext(ctx, "SELECT count(*),min(version),max(version) FROM schema_version").Scan(&count, &minimum, &maximum); err != nil || count != 1 || minimum != store.SchemaVersion() || maximum != minimum {
		return errors.New("existing schema differs; fixture migration refused")
	}
	return nil
}

// GuestState is the fixed synthetic-state driver, invoked only through the
// qualifier's pinned private SSH connection. It uses Store and native Forgejo
// CLI/API operations, not schema recreation or database restoration.
func GuestState(ctx context.Context, action, expectedPayload string) (map[string]any, error) {
	if os.Geteuid() != 0 {
		return nil, errors.New("guest root required")
	}
	if _, err := os.Stat("/run/ostree-booted"); err != nil {
		return nil, errors.New("native OSTree guest required")
	}
	payload, err := appliancerelease.Load("/usr/share/soda/release.json")
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil || hostname != fixtureLogin || payload.ID != expectedPayload {
		return nil, errors.New("qualification guest identity mismatch")
	}
	if action != "seed" && action != "later" && action != "snapshot" {
		return nil, errors.New("fixed state action required")
	}
	credential := filepath.Join(fixtureState, "forgejo-password")
	if action == "seed" {
		if err = os.Mkdir(fixtureState, 0700); err != nil {
			return nil, err
		}
		if err = configureFixtureForgejo(ctx); err != nil {
			return nil, err
		}
		// Native random-password creation keeps the password out of argv. Do not
		// log this output or attach it to errors; retain only the restricted input.
		cmd := exec.CommandContext(ctx, "podman", "exec", "--user", "git", "soda-forgejo", "forgejo", "admin", "user", "create", "--username", fixtureLogin, "--email", "soda-tester@example.invalid", "--admin", "--random-password", "--must-change-password=false")
		output, e := cmd.Output()
		if e != nil {
			return nil, errors.New("native fixture account creation failed")
		}
		match := regexp.MustCompile(`(?m)^generated random password is '(.*)'$`).FindSubmatch(output)
		if len(match) != 2 || len(match[1]) < 12 {
			return nil, errors.New("native random password output unavailable; do not replay account creation")
		}
		if err = nativebuild.WriteNew(credential, match[1], 0600); err != nil {
			return nil, err
		}
		if err = configureFixtureSoda(ctx); err != nil {
			return nil, err
		}
	}
	var cfg struct {
		Database     string `json:"database"`
		GrantKeyFile string `json:"grant_key_file"`
	}
	b, err := os.ReadFile("/etc/soda/dashboard.json")
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(cfg.Database, "/var/lib/soda/") {
		return nil, errors.New("native Soda database required")
	}
	password, err := os.ReadFile(credential)
	if err != nil {
		return nil, err
	}
	api := func(method, path string, input, output any) error {
		var body io.Reader
		if input != nil {
			b, e := json.Marshal(input)
			if e != nil {
				return e
			}
			body = bytes.NewReader(b)
		}
		req, e := http.NewRequestWithContext(ctx, method, "http://127.0.0.1:3000/api/v1"+path, body)
		if e != nil {
			return e
		}
		req.SetBasicAuth(fixtureLogin, string(password))
		req.Header.Set("Content-Type", "application/json")
		res, e := (&http.Client{Timeout: 30 * time.Second}).Do(req)
		if e != nil {
			return errors.New("local Forgejo API unavailable")
		}
		defer res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("local Forgejo fixture API refused: %d", res.StatusCode)
		}
		if output != nil {
			return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(output)
		}
		return nil
	}
	var user struct {
		ID int64 `json:"id"`
	}
	if err = api("GET", "/user", nil, &user); err != nil {
		return nil, err
	}
	var repo struct {
		ID       int64  `json:"id"`
		FullName string `json:"full_name"`
	}
	if action == "seed" {
		err = api("POST", "/user/repos", map[string]any{"name": "p9-repository", "auto_init": true, "default_branch": "main", "private": true}, &repo)
	} else {
		err = api("GET", "/repos/"+fixtureLogin+"/p9-repository", nil, &repo)
	}
	if err != nil {
		return nil, err
	}
	if action != "snapshot" {
		if action == "later" {
			if err = checkStateSchema(ctx, cfg.Database); err != nil {
				return nil, err
			}
		}
		key, err := config.GrantKey(cfg.GrantKeyFile)
		if err != nil {
			return nil, err
		}
		db, err := store.OpenEncrypted(cfg.Database, key)
		if err != nil {
			return nil, err
		}
		defer db.Close()
		generation := "a"
		if action == "later" {
			generation = "b"
		}
		content := []byte("committed on generation " + generation + "\n")
		if err = api("POST", "/repos/"+fixtureLogin+"/p9-repository/contents/"+generation+".txt", map[string]any{"content": base64.StdEncoding.EncodeToString(content), "message": "Qualification generation " + generation, "branch": "main"}, nil); err != nil {
			return nil, err
		}
		if err = db.UpsertUser(ctx, store.User{ID: user.ID, Login: fixtureLogin, Name: "Soda fixture generation " + generation}); err != nil {
			return nil, err
		}
		if action == "seed" {
			im := payload.Images["project-os"]
			raw, e := exec.CommandContext(ctx, "podman", "image", "inspect", "--format", "{{json .Labels}}", im.Config).Output()
			if e != nil {
				return nil, e
			}
			var labels map[string]string
			if e = json.Unmarshal(raw, &labels); e != nil {
				return nil, e
			}
			profile := projectos.Profile{ID: labels["org.soda.profile"], Distribution: labels["org.soda.distribution"], Version: labels["org.soda.distribution.version"], Interface: labels["org.soda.interface"], Architecture: "amd64", Image: im.Config, Revision: payload.Revision}
			if err = db.CreateProject(ctx, store.Project{ID: fixtureProject, Name: "p9-project", RepositoryID: repo.ID, OwnerID: user.ID, Repository: repo.FullName, Profile: &profile}); err != nil {
				return nil, err
			}
			request, _ := json.Marshal(host.Create{ID: fixtureProject, Owner: user.ID, Profile: &profile})
			cmd := exec.CommandContext(ctx, "runuser", "-u", "soda", "--", "curl", "--silent", "--show-error", "--fail", "--max-time", "240", "--unix-socket", "/run/soda/host.sock", "--header", "Content-Type: application/json", "--data-binary", "@-", "http://soda-host/create")
			cmd.Stdin = bytes.NewReader(request)
			result, e := cmd.Output()
			if e != nil {
				return nil, errors.New("native fixture project creation failed; do not recreate")
			}
			var environment host.Environment
			if e = json.Unmarshal(result, &environment); e != nil || environment.ID != fixtureProject || !environment.Running {
				return nil, errors.New("native project identity differs")
			}
			if e = db.MarkReady(ctx, fixtureProject, environment.IP); e != nil {
				return nil, e
			}
			if e = exec.CommandContext(ctx, "podman", "exec", "soda-"+fixtureProject, "mkdir", "/var/lib/p9-data").Run(); e != nil {
				return nil, e
			}
		}
		cmd := exec.CommandContext(ctx, "podman", "exec", "-i", "soda-"+fixtureProject, "/bin/sh", "-ec", "set -C; cat > /var/lib/p9-data/"+generation+".txt")
		cmd.Stdin = bytes.NewReader(content)
		if err = cmd.Run(); err != nil {
			return nil, err
		}
		if err = db.Close(); err != nil {
			return nil, err
		}
		if action == "seed" {
			if err = os.Chown(cfg.Database, 2000, 2000); err != nil {
				return nil, err
			}
		}
	}
	// Observers open the existing database read-only, and never run migrations.
	observedDB, err := sql.Open("sqlite", "file:"+cfg.Database+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer observedDB.Close()
	var schema int
	var integrity string
	if err = observedDB.QueryRowContext(ctx, "SELECT version FROM schema_version").Scan(&schema); err != nil || schema != store.SchemaVersion() {
		return nil, errors.New("schema differs; native downgrade/observation refused")
	}
	if err = observedDB.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		return nil, errors.New("Soda database integrity failed")
	}
	var observedUser store.User
	if err = observedDB.QueryRowContext(ctx, "SELECT id,login,name FROM users WHERE id=?", user.ID).Scan(&observedUser.ID, &observedUser.Login, &observedUser.Name); err != nil {
		return nil, err
	}
	var name, repository, ip, creationProfile string
	var owner, repositoryID, ready int64
	if err = observedDB.QueryRowContext(ctx, "SELECT name,repository_id,owner_id,repository,ip,ready,creation_profile FROM projects WHERE id=?", fixtureProject).Scan(&name, &repositoryID, &owner, &repository, &ip, &ready, &creationProfile); err != nil {
		return nil, err
	}
	project := map[string]any{"id": fixtureProject, "name": name, "repository_id": repositoryID, "owner": owner, "repository": repository, "ip": ip, "ready": ready, "creation_profile": creationProfile}
	files := map[string]string{}
	out, err := exec.CommandContext(ctx, "podman", "exec", "soda-"+fixtureProject, "/bin/sh", "-ec", "cd /var/lib/p9-data; sha256sum *.txt").Output()
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 || !nativebuild.Digest(parts[0]) {
			return nil, errors.New("project file observation malformed")
		}
		files[parts[1]] = parts[0]
	}
	container, err := exec.CommandContext(ctx, "podman", "inspect", "--format", "{{.Id}} {{.Image}} {{.Name}}", "soda-"+fixtureProject).Output()
	if err != nil {
		return nil, err
	}
	project["container"] = strings.TrimSpace(string(container))
	var ref any
	if err = api("GET", "/repos/"+fixtureLogin+"/p9-repository/git/refs/heads/main", nil, &ref); err != nil {
		return nil, err
	}
	identity := map[string]string{}
	paths := []string{"/etc/machine-id", "/etc/hostname", "/etc/soda/dashboard.json", "/etc/subuid", "/etc/subgid"}
	public, e := filepath.Glob("/etc/ssh/ssh_host_*.pub")
	if e != nil {
		return nil, e
	}
	paths = append(paths, public...)
	for _, p := range paths {
		b, e := os.ReadFile(p)
		if e != nil {
			return nil, e
		}
		sum := sha256.Sum256(b)
		identity[p] = hex.EncodeToString(sum[:])
	}
	return map[string]any{"user": observedUser, "project": project, "project_files": files, "forgejo_repository": repo, "forgejo_ref": ref, "machine_settings_public_keys": identity, "schema": store.SchemaVersion()}, nil
}
