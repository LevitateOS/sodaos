package qualify

import (
	"bytes"
	"context"
	"crypto/sha256"
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

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/levitateos/sodaos/internal/store"
)

const (
	fixtureState   = "/var/lib/soda-qualification"
	fixtureLogin   = "soda-tester"
	fixtureProject = "p000000000000000000000001"
)

func checkStateSchema(ctx context.Context, path string) error {
	s, err := store.OpenObserve(ctx, path)
	if err != nil {
		return errors.New("existing schema differs; fixture migration refused")
	}
	return s.Close()
}

func validGuestAction(action string) bool {
	return action == "seed" || action == "later" || action == "snapshot" || action == "content"
}

func verifyGuestIdentity(payloadID, expectedPayload string) error {
	hostname, err := os.Hostname()
	if err != nil || hostname != fixtureLogin || payloadID != expectedPayload {
		return errors.New("qualification guest identity mismatch")
	}
	return nil
}

func admitGuestState(action, expectedPayload string) (deliver.Payload, error) {
	if !validGuestAction(action) {
		return deliver.Payload{}, errors.New("fixed state action required")
	}
	if os.Geteuid() != 0 {
		return deliver.Payload{}, errors.New("guest root required")
	}
	if _, err := os.Stat("/run/ostree-booted"); err != nil {
		return deliver.Payload{}, errors.New("native OSTree guest required")
	}
	payload, err := deliver.Load("/usr/share/soda/release.json")
	if err != nil {
		return payload, err
	}
	return payload, verifyGuestIdentity(payload.ID, expectedPayload)
}

func createFixtureAccount(ctx context.Context, credential string) error {
	// Native random-password creation keeps the password out of argv. Do not
	// log this output or attach it to errors; retain only the restricted input.
	cmd := exec.CommandContext(ctx, "podman", "exec", "--user", "git", "soda-forgejo", "forgejo", "admin", "user", "create", "--username", fixtureLogin, "--email", "soda-tester@example.invalid", "--admin", "--random-password", "--must-change-password=false")
	output, err := cmd.Output()
	if err != nil {
		return errors.New("native fixture account creation failed")
	}
	match := regexp.MustCompile(`(?m)^generated random password is '(.*)'$`).FindSubmatch(output)
	if len(match) != 2 || len(match[1]) < 12 {
		return errors.New("native random password output unavailable; do not replay account creation")
	}
	return build.WriteNew(credential, match[1], 0o600)
}

func bootstrapFixtureState(ctx context.Context, credential string) error {
	if err := os.Mkdir(fixtureState, 0o700); err != nil {
		return err
	}
	if err := configureFixtureForgejo(ctx); err != nil {
		return err
	}
	if err := createFixtureAccount(ctx, credential); err != nil {
		return err
	}
	return configureFixtureSoda(ctx)
}

type guestConfig struct {
	Database     string `json:"database"`
	GrantKeyFile string `json:"grant_key_file"`
}

func loadGuestConfig() (guestConfig, error) {
	var cfg guestConfig
	b, err := os.ReadFile("/etc/soda/dashboard.json")
	if err != nil {
		return cfg, err
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if !strings.HasPrefix(cfg.Database, "/var/lib/soda/") {
		return cfg, errors.New("native Soda database required")
	}
	return cfg, nil
}

type forgejoClient struct {
	ctx      context.Context
	password string
}

func (c forgejoClient) call(method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(c.ctx, method, "http://127.0.0.1:3000/api/v1"+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(fixtureLogin, c.password)
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
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

func initGuestClient(ctx context.Context, action, credential string) (guestConfig, forgejoClient, error) {
	if action == "seed" {
		if err := bootstrapFixtureState(ctx, credential); err != nil {
			return guestConfig{}, forgejoClient{}, err
		}
	}
	cfg, err := loadGuestConfig()
	if err != nil {
		return cfg, forgejoClient{}, err
	}
	password, err := os.ReadFile(credential)
	if err != nil {
		return cfg, forgejoClient{}, err
	}
	return cfg, forgejoClient{ctx: ctx, password: string(password)}, nil
}

type fixtureRepo struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type fixtureUser struct {
	ID int64 `json:"id"`
}

func setupFixtureRepo(client forgejoClient, action string) (fixtureUser, fixtureRepo, error) {
	var user fixtureUser
	if err := client.call("GET", "/user", nil, &user); err != nil {
		return user, fixtureRepo{}, err
	}
	var repo fixtureRepo
	var err error
	if action == "seed" {
		err = client.call("POST", "/user/repos", map[string]any{"name": "p9-repository", "auto_init": true, "default_branch": "main", "private": true}, &repo)
	} else {
		err = client.call("GET", "/repos/"+fixtureLogin+"/p9-repository", nil, &repo)
	}
	return user, repo, err
}

func openFixtureStore(ctx context.Context, cfg guestConfig, action string) (*store.Store, error) {
	if action == "later" {
		if err := checkStateSchema(ctx, cfg.Database); err != nil {
			return nil, err
		}
	}
	key, err := config.GrantKey(cfg.GrantKeyFile)
	if err != nil {
		return nil, err
	}
	return store.OpenEncrypted(cfg.Database, key)
}

func updateFixtureProfile(ctx context.Context, db *store.Store, userID int64, generation, action string) error {
	name := "Soda fixture generation " + generation
	if action == "seed" {
		return db.UpsertUser(ctx, store.User{ID: userID, Login: fixtureLogin, Name: name})
	}
	// Provider refresh deliberately preserves an existing profile name.
	return db.RenameProfile(ctx, userID, name)
}

func updateFixtureGeneration(ctx context.Context, client forgejoClient, db *store.Store, userID int64, generation, action string, content []byte) error {
	if err := client.call("POST", "/repos/"+fixtureLogin+"/p9-repository/contents/"+generation+".txt", map[string]any{"content": base64.StdEncoding.EncodeToString(content), "message": "Qualification generation " + generation, "branch": "main"}, nil); err != nil {
		return err
	}
	return updateFixtureProfile(ctx, db, userID, generation, action)
}

func seedFixtureProject(ctx context.Context, db *store.Store, payload deliver.Payload, user fixtureUser, repo fixtureRepo) error {
	im := payload.Images["project-os"]
	raw, err := exec.CommandContext(ctx, "podman", "image", "inspect", "--format", "{{json .Labels}}", im.Config).Output()
	if err != nil {
		return err
	}
	var labels map[string]string
	if err = json.Unmarshal(raw, &labels); err != nil {
		return err
	}
	profile := project.Profile{ID: labels["org.soda.profile"], Distribution: labels["org.soda.distribution"], Version: labels["org.soda.distribution.version"], Interface: labels["org.soda.interface"], Architecture: "amd64", Image: im.Config, Revision: payload.Revision}
	if err = db.CreateProject(ctx, store.Project{ID: fixtureProject, Name: "p9-project", RepositoryID: repo.ID, OwnerID: user.ID, Repository: repo.FullName, Profile: &profile}); err != nil {
		return err
	}
	request, _ := json.Marshal(project.Create{ID: fixtureProject, Owner: user.ID, Profile: &profile})
	cmd := exec.CommandContext(ctx, "runuser", "-u", "soda", "--", "curl", "--silent", "--show-error", "--fail", "--max-time", "240", "--unix-socket", "/run/soda/host.sock", "--header", "Content-Type: application/json", "--data-binary", "@-", "http://soda-host/create")
	cmd.Stdin = bytes.NewReader(request)
	result, err := cmd.Output()
	if err != nil {
		return errors.New("native fixture project creation failed; do not recreate")
	}
	var environment project.Environment
	if err = json.Unmarshal(result, &environment); err != nil || environment.ID != fixtureProject || !environment.Running {
		return errors.New("native project identity differs")
	}
	if err = db.MarkReady(ctx, fixtureProject, environment.IP); err != nil {
		return err
	}
	return exec.CommandContext(ctx, "podman", "exec", "soda-"+fixtureProject, "mkdir", "/var/lib/p9-data").Run()
}

func writeProjectFile(ctx context.Context, generation string, content []byte) error {
	cmd := exec.CommandContext(ctx, "podman", "exec", "-i", "soda-"+fixtureProject, "/bin/sh", "-ec", "set -C; cat > /var/lib/p9-data/"+generation+".txt")
	cmd.Stdin = bytes.NewReader(content)
	return cmd.Run()
}

func mutateFixtureState(ctx context.Context, client forgejoClient, cfg guestConfig, payload deliver.Payload, action string, user fixtureUser, repo fixtureRepo) error {
	db, err := openFixtureStore(ctx, cfg, action)
	if err != nil {
		return err
	}
	defer db.Close()
	generation := "a"
	if action == "later" {
		generation = "b"
	}
	content := []byte("committed on generation " + generation + "\n")
	if err = updateFixtureGeneration(ctx, client, db, user.ID, generation, action, content); err != nil {
		return err
	}
	if action == "seed" {
		if err = seedFixtureProject(ctx, db, payload, user, repo); err != nil {
			return err
		}
	}
	if err = writeProjectFile(ctx, generation, content); err != nil {
		return err
	}
	if err = db.Close(); err != nil {
		return err
	}
	if action == "seed" {
		return os.Chown(cfg.Database, 2000, 2000)
	}
	return nil
}

func observeDatabase(ctx context.Context, dbPath string, userID int64) (store.User, map[string]any, error) {
	observed, err := store.OpenObserve(ctx, dbPath)
	if err != nil {
		return store.User{}, nil, errors.New("schema differs; native downgrade/observation refused")
	}
	defer observed.Close()
	if err = observed.IntegrityCheck(ctx); err != nil {
		return store.User{}, nil, err
	}
	observedUser, err := observed.User(ctx, userID)
	if err != nil {
		return store.User{}, nil, err
	}
	p, err := observed.Project(ctx, fixtureProject)
	if err != nil {
		return store.User{}, nil, err
	}
	ready := 0
	if p.Ready {
		ready = 1
	}
	creationProfile := ""
	if p.Profile != nil {
		raw, err := json.Marshal(p.Profile)
		if err != nil {
			return store.User{}, nil, err
		}
		creationProfile = string(raw)
	}
	project := map[string]any{"id": fixtureProject, "name": p.Name, "repository_id": p.RepositoryID, "owner": p.OwnerID, "repository": p.Repository, "ip": p.IP, "ready": ready, "creation_profile": creationProfile}
	return observedUser, project, nil
}

func observeProjectState(ctx context.Context) (map[string]string, string, error) {
	files := map[string]string{}
	out, err := exec.CommandContext(ctx, "podman", "exec", "soda-"+fixtureProject, "/bin/sh", "-ec", "cd /var/lib/p9-data; sha256sum *.txt").Output()
	if err != nil {
		return nil, "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 || !build.Digest(parts[0]) {
			return nil, "", errors.New("project file observation malformed")
		}
		files[parts[1]] = parts[0]
	}
	container, err := exec.CommandContext(ctx, "podman", "inspect", "--format", "{{.Id}} {{.Image}} {{.Name}}", "soda-"+fixtureProject).Output()
	if err != nil {
		return nil, "", err
	}
	return files, strings.TrimSpace(string(container)), nil
}

func observeMachineIdentity() (map[string]string, error) {
	identity := map[string]string{}
	paths := []string{"/etc/machine-id", "/etc/hostname", "/etc/soda/dashboard.json", "/etc/subuid", "/etc/subgid"}
	public, err := filepath.Glob("/etc/ssh/ssh_host_*.pub")
	if err != nil {
		return nil, err
	}
	paths = append(paths, public...)
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		identity[p] = hex.EncodeToString(sum[:])
	}
	return identity, nil
}

type guestObservation struct {
	user     store.User
	project  map[string]any
	files    map[string]string
	ref      any
	identity map[string]string
}

func observeGuestState(ctx context.Context, client forgejoClient, dbPath string, userID int64) (guestObservation, error) {
	var obs guestObservation
	var err error
	obs.user, obs.project, err = observeDatabase(ctx, dbPath, userID)
	if err != nil {
		return obs, err
	}
	obs.files, obs.project["container"], err = observeProjectState(ctx)
	if err != nil {
		return obs, err
	}
	if err = client.call("GET", "/repos/"+fixtureLogin+"/p9-repository/git/refs/heads/main", nil, &obs.ref); err != nil {
		return obs, err
	}
	obs.identity, err = observeMachineIdentity()
	return obs, err
}

func observeContent(payload deliver.Payload) (map[string]any, error) {
	files, size, err := deliver.VerifyContent(payload, deliver.ImagesPath)
	if err != nil {
		return nil, err
	}
	return map[string]any{"files": files, "bytes": size}, nil
}

func observeFixtureState(ctx context.Context, client forgejoClient, dbPath string, user fixtureUser, repo fixtureRepo) (map[string]any, error) {
	obs, err := observeGuestState(ctx, client, dbPath, user.ID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"user":                         obs.user,
		"project":                      obs.project,
		"project_files":                obs.files,
		"forgejo_repository":           repo,
		"forgejo_ref":                  obs.ref,
		"machine_settings_public_keys": obs.identity,
		"schema":                       store.SchemaVersion(),
	}, nil
}

// GuestState is the fixed synthetic-state driver, invoked only through the
// qualifier's pinned private SSH connection. It uses Store and native Forgejo
// CLI/API operations, not schema recreation or database restoration.
func GuestState(ctx context.Context, action, expectedPayload string) (map[string]any, error) {
	payload, err := admitGuestState(action, expectedPayload)
	if err != nil {
		return nil, err
	}
	if action == "content" {
		return observeContent(payload)
	}
	credential := filepath.Join(fixtureState, "forgejo-password")
	cfg, client, err := initGuestClient(ctx, action, credential)
	if err != nil {
		return nil, err
	}
	user, repo, err := setupFixtureRepo(client, action)
	if err != nil {
		return nil, err
	}
	if action != "snapshot" {
		if err = mutateFixtureState(ctx, client, cfg, payload, action, user, repo); err != nil {
			return nil, err
		}
	}
	return observeFixtureState(ctx, client, cfg.Database, user, repo)
}
