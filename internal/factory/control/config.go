// Package control admits and executes the fixed factory loop. Forgejo owns
// collaboration, store owns the ledger, and host executors own infrastructure.
package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host/publish"
	"github.com/levitateos/sodaos/internal/host/workspace"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
)

type Config struct {
	IdentitySocket          string           `json:"identity_socket"`
	ConnectionID            string           `json:"connection_id"`
	ProjectID               string           `json:"project_id"`
	Root                    string           `json:"root"`
	ForgejoURL              string           `json:"forgejo_url"`
	RepositoryID            int64            `json:"repository_id"`
	HumanTokenFile          string           `json:"human_token_file"`
	ImplementationTokenFile string           `json:"implementation_token_file"`
	ReviewTokenFile         string           `json:"review_token_file"`
	ImplementationActor     string           `json:"implementation_actor"`
	ReviewActor             string           `json:"review_actor"`
	Workflow                string           `json:"workflow"`
	Workspace               workspace.Config `json:"workspace"`
}

type Controller struct {
	Identity                                     *identityclient.Client
	Config                                       Config
	PolicySHA                                    string
	Store                                        *store.Store
	Forgejo                                      *forgejo.Client
	Workspace                                    *workspace.Runtime
	Publisher                                    publish.Config
	Repository                                   forgejo.Repository
	Human                                        forgejo.User
	Implementer                                  forgejo.User
	Reviewer                                     forgejo.User
	humanToken, implementationToken, reviewToken string
}

func Load(path string) (Config, string, error) {
	var config Config
	data, err := privateFile(path)
	if err != nil {
		return config, "", err
	}
	if err = strictjson.Decode(strings.NewReader(string(data)), &config); err != nil {
		return config, "", err
	}
	digest := sha256.Sum256(data)
	return config, hex.EncodeToString(digest[:]), config.Validate()
}

func (c Config) validateIdentity() error {
	if !filepath.IsAbs(c.IdentitySocket) || strings.ContainsAny(c.IdentitySocket, "\x00\r\n") || c.ConnectionID == "" || c.ProjectID == "" {
		return errors.New("explicit identity socket, connection and project are required")
	}
	return nil
}

func (c Config) Validate() error {
	if err := c.validateIdentity(); err != nil {
		return err
	}
	if !filepath.IsAbs(c.Root) || c.RepositoryID <= 0 || c.Workflow == "" || strings.ContainsAny(c.Workflow, "/\\\r\n") {
		return errors.New("explicit factory root, repository and workflow are required")
	}
	if c.Workspace.Root != filepath.Join(c.Root, "workspaces") {
		return errors.New("workspace root must belong to factory state")
	}
	if err := c.Workspace.Validate(); err != nil {
		return err
	}
	remote := publish.Config{Root: c.Root, Remote: c.ForgejoURL, Username: "validation", TokenFile: c.HumanTokenFile}
	return remote.Validate()
}

func privateFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > 256<<10 {
		return nil, errors.New("factory inputs must be bounded private regular files")
	}
	return os.ReadFile(path)
}

func token(path string) (string, error) {
	data, err := privateFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(data))
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", errors.New("invalid credential input")
	}
	return value, nil
}

func OpenLocal(config Config, policy string) (*Controller, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if err := privateDirectory(config.Root); err != nil {
		return nil, err
	}
	s, err := store.Open(filepath.Join(config.Root, "execution.db"))
	if err != nil {
		return nil, err
	}
	return &Controller{Identity: identityclient.New(config.IdentitySocket), Config: config, PolicySHA: policy, Store: s, Workspace: &workspace.Runtime{Config: config.Workspace, Exec: workspace.Native{}}}, nil
}

func Open(ctx context.Context, config Config, policy string) (*Controller, error) {
	c, err := OpenLocal(config, policy)
	if err != nil {
		return nil, err
	}
	c.Forgejo = forgejo.New(config.ForgejoURL)
	if err = c.initialize(ctx); err != nil {
		_ = c.Store.Close()
		return nil, err
	}
	return c, nil
}

func (c *Controller) initialize(ctx context.Context) error {
	if err := privateDirectory(c.Config.Root); err != nil {
		return err
	}
	var err error
	c.humanToken, err = token(c.Config.HumanTokenFile)
	if err != nil {
		return err
	}
	c.implementationToken, err = token(c.Config.ImplementationTokenFile)
	if err != nil {
		return err
	}
	c.reviewToken, err = token(c.Config.ReviewTokenFile)
	if err != nil {
		return err
	}
	if err = c.identities(ctx); err != nil {
		return err
	}
	if err = c.repository(ctx); err != nil {
		return err
	}
	c.Workspace, err = workspace.Open(c.Config.Workspace)
	if err == nil && c.Config.Workspace.GitSocket != "" {
		c.Workspace.GitRemote = c.Publisher.Remote
	}
	return err
}

func privateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return errors.New("factory state must be an existing private directory")
	}
	return nil
}

func (c *Controller) identities(ctx context.Context) error {
	var err error
	c.Human, err = c.Forgejo.Current(ctx, c.humanToken)
	if err != nil {
		return err
	}
	c.Implementer, err = c.Forgejo.WorkActor(ctx, c.humanToken, c.Config.ImplementationActor)
	if err != nil {
		return err
	}
	c.Reviewer, err = c.Forgejo.WorkActor(ctx, c.humanToken, c.Config.ReviewActor)
	if err != nil {
		return err
	}
	return c.validateIdentities()
}

func (c *Controller) validateIdentities() error {
	if c.Human.ID <= 0 || c.Implementer.ID <= 0 || c.Reviewer.ID <= 0 {
		return errors.New("invalid factory actors")
	}
	if c.Human.ID == c.Implementer.ID || c.Human.ID == c.Reviewer.ID || c.Implementer.ID == c.Reviewer.ID {
		return errors.New("factory roles require separate actors")
	}
	if c.Implementer.Admin || c.Reviewer.Admin {
		return errors.New("factory workers cannot be Forgejo administrators")
	}
	return nil
}

func (c *Controller) repository(ctx context.Context) error {
	repo, err := c.Forgejo.RepositoryByID(ctx, c.humanToken, c.Config.RepositoryID)
	if err != nil {
		return err
	}
	if !repo.Private || repo.Permissions == nil || !repo.Permissions.Push {
		return errors.New("human must retain write access to a private repository")
	}
	c.Repository = repo
	remote := strings.TrimRight(c.Config.ForgejoURL, "/") + "/" + url.PathEscape(repo.Owner.Login) + "/" + url.PathEscape(repo.Name) + ".git"
	c.Publisher = publish.Config{Root: filepath.Join(c.Config.Root, "publications"), Remote: remote, Username: c.Implementer.Login, TokenFile: c.Config.ImplementationTokenFile}
	return privateDirectory(c.Publisher.Root)
}
