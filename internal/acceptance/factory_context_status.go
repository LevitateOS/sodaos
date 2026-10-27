package acceptance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// FactoryContextToolRevision is pinned at build time; an unpinned probe refuses.
var FactoryContextToolRevision string

type factoryContextScope struct {
	InstalledSource     string `json:"installed_source_revision"`
	PolicyCommandID     string `json:"policy_command_id"`
	Owner               int64  `json:"owner,string"`
	Operation           string `json:"operation"`
	ReviewerPreparation string `json:"reviewer_preparation"`
	Format              int    `json:"format"`
	ToolSource          string `json:"tool_source_revision"`
	PayloadSHA          string `json:"payload_sha256"`
	Repository          int64  `json:"repository"`
	Issue               int64  `json:"issue"`
	Project             string `json:"project"`
	Acceptance          string `json:"acceptance"`
	SourceCommit        string `json:"source_commit"`
	Preparation         string `json:"preparation"`
	Connection          string `json:"connection"`
}

func readFactoryContextScope(scopePath string) (*factoryContextScope, string, error) {
	if !filepath.IsAbs(scopePath) || filepath.Clean(scopePath) != scopePath || !strings.HasPrefix(scopePath, "/home/") || strings.ContainsAny(scopePath, "\x00\r\n") {
		return nil, "", errors.New("scope unavailable")
	}
	directory, err := os.Lstat(filepath.Dir(scopePath))
	if err != nil || !directory.IsDir() || directory.Mode().Perm() != 0700 {
		return nil, "", errors.New("scope unavailable")
	}
	s, err := os.Lstat(scopePath)
	if err != nil {
		return nil, "", errors.New("scope unavailable")
	}
	stat, ok := s.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || stat.Gid != 0 || stat.Nlink != 1 || s.Mode().Perm() != 0600 {
		return nil, "", errors.New("scope unavailable")
	}
	raw, err := readFactoryContextPrivateFile(scopePath, 4096)
	if err != nil {
		return nil, "", err
	}
	var out factoryContextScope
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil || decoder.Decode(new(any)) != io.EOF || out.Format != 1 || out.ToolSource != FactoryContextToolRevision || !factory.ValidCommit(FactoryContextToolRevision) || !factory.ValidCommit(out.InstalledSource) || !factory.ValidID(out.PolicyCommandID) || !factory.ValidDigest(out.PayloadSHA) || out.Repository <= 0 || out.Issue <= 0 || !factory.ValidProjectID(out.Project) || !project.ValidDecisionID(out.Acceptance) || !factory.ValidCommit(out.SourceCommit) || !factory.ValidPreparationRef(out.Preparation) || !factory.ValidID(out.Connection) || out.Owner <= 0 || !factory.ValidID(out.Operation) || !factory.ValidPreparationRef(out.ReviewerPreparation) {
		return nil, "", errors.New("scope unavailable")
	}
	digest := sha256.Sum256(raw)
	return &out, hex.EncodeToString(digest[:]), nil
}

type factoryContextRequest struct {
	RunID        string `json:"run_id"`
	AssignmentID string `json:"assignment_id"`
}

type factoryContextResponse struct {
	Phase               string                        `json:"phase"`
	Role                string                        `json:"role"`
	SourceCommit        string                        `json:"source_commit"`
	Preparation         string                        `json:"preparation"`
	Publication         string                        `json:"publication,omitempty"`
	Candidate           string                        `json:"candidate,omitempty"`
	CurrentGeneration   int64                         `json:"current_connection_generation"`
	ScopeSHA            string                        `json:"scope_sha256"`
	Format              int                           `json:"format"`
	ToolSource          string                        `json:"tool_source_revision"`
	RunID               string                        `json:"run_id"`
	AssignmentID        string                        `json:"assignment_id"`
	Repository          int64                         `json:"repository"`
	Issue               int64                         `json:"issue"`
	Project             string                        `json:"project"`
	Root                string                        `json:"attempt_root"`
	ContextPending      bool                          `json:"context_pending"`
	Reconciled          bool                          `json:"reconciled"`
	CredentialDelegated bool                          `json:"credential_delegated"`
	NativeBinding       bool                          `json:"native_binding_present"`
	ContextSelection    *control.ContextSelectionView `json:"context_selection"`
}

func readFactoryContextPrivateFile(path string, limit int64) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || limit <= 0 || limit > 1<<20 {
		return nil, errors.New("private input unavailable")
	}
	for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
		s, err := os.Lstat(parent)
		if err != nil {
			return nil, errors.New("private input unavailable")
		}
		stat, ok := s.Sys().(*syscall.Stat_t)
		if !ok || !s.IsDir() || stat.Uid != 0 || stat.Gid != 0 || s.Mode().Perm()&0022 != 0 {
			return nil, errors.New("private input unavailable")
		}
		if parent == "/" {
			break
		}
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	stat, ok := s.Sys().(*syscall.Stat_t)
	if !ok || !s.Mode().IsRegular() || stat.Uid != 0 || stat.Gid != 0 || stat.Nlink != 1 || s.Size() <= 0 || s.Size() > limit || s.Mode().Perm()&0022 != 0 {
		return nil, errors.New("private input unavailable")
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, errors.New("private input unavailable")
	}
	return raw, nil
}

func observeFactoryContext(ctx context.Context, scopePath string, in factoryContextRequest) (*factoryContextResponse, error) {
	bound, scopeHash, err := readFactoryContextScope(scopePath)
	if err != nil {
		return nil, err
	}
	release, err := readFactoryContextPrivateFile("/usr/share/soda/release.json", 1<<20)
	if err != nil {
		return nil, err
	}
	var installed struct {
		Revision string `json:"Revision"`
	}
	payloadDigest := sha256.Sum256(release)
	if json.Unmarshal(release, &installed) != nil || installed.Revision != bound.InstalledSource || hex.EncodeToString(payloadDigest[:]) != bound.PayloadSHA {
		return nil, errors.New("installed source unavailable")
	}
	configBytes, err := readFactoryContextPrivateFile("/etc/soda/dashboard.json", 1<<20)
	if err != nil {
		return nil, err
	}
	var server struct {
		DSNFile string `json:"database_dsn_file"`
	}
	if json.Unmarshal(configBytes, &server) != nil || server.DSNFile != "/etc/soda/postgres/soda.dsn" {
		return nil, errors.New("private input unavailable")
	}
	dsn, err := readFactoryContextPrivateFile(server.DSNFile, 16384)
	if err != nil {
		return nil, err
	}
	config, err := pgx.ParseConfig(string(bytes.TrimSpace(dsn)))
	if err != nil {
		return nil, err
	}
	if config.RuntimeParams == nil {
		config.RuntimeParams = make(map[string]string)
	}
	config.RuntimeParams["default_transaction_read_only"] = "on"
	config.RuntimeParams["statement_timeout"] = "5000"
	config.RuntimeParams["lock_timeout"] = "5000"
	config.ConnectTimeout = 5 * time.Second
	registered := stdlib.RegisterConnConfig(config)
	defer stdlib.UnregisterConnConfig(registered)
	st, err := store.OpenObserve(ctx, registered)
	if err != nil {
		return nil, err
	}
	defer st.Close()
	run, err := st.FactoryRun(ctx, in.RunID)
	if err != nil {
		return nil, err
	}
	a, err := st.Assignment(ctx, in.AssignmentID)
	if err != nil {
		return nil, err
	}
	generation, err := observeFactoryContextOwnerOperation(ctx, st, registered, bound, a)
	if err != nil {
		return nil, err
	}
	if a.Phase == project.ExecutionReview && a.Role == project.RoleReviewer {
		return observeFactoryReviewContext(ctx, st, bound, scopeHash, in, run, a, generation)
	}
	if a.ID != in.AssignmentID || run.ID != in.RunID || run.Validate() != nil || !time.Now().Before(run.Deadline) || a.Run != in.RunID || a.Repository != bound.Repository || a.Issue != bound.Issue || a.ProjectID != bound.Project || run.ProjectID != bound.Project || !factory.ValidID(a.AttemptRoot) || a.ID == a.AttemptRoot || a.ParentAssignment != a.AttemptRoot || a.PublicationAssignment != a.AttemptRoot || a.Acceptance != bound.Acceptance || a.SourceCommit != bound.SourceCommit || run.InputSHA != bound.SourceCommit || a.Preparation != bound.Preparation || a.Connection != bound.Connection || a.Phase != project.ExecutionImplement || a.Role != project.RoleCoder || run.Role != project.RoleCoder || a.Stage != factory.AssignmentAssigned || !run.ContextPending || run.Reconciled || run.Outcome != "" || run.CredentialDelegated || run.IdentityBinding != nil {
		return nil, errors.New("scope unavailable")
	}
	parent, err := st.Assignment(ctx, a.AttemptRoot)
	if err != nil || parent.Validate() != nil || parent.ID != a.AttemptRoot || parent.ActorID != bound.Owner || parent.PublicationAssignment != parent.ID || parent.AttemptRoot != parent.ID || parent.Repository != bound.Repository || parent.Issue != bound.Issue || parent.ProjectID != bound.Project || parent.SourceCommit != bound.SourceCommit || parent.Acceptance != bound.Acceptance || parent.Preparation != bound.Preparation || parent.Connection != bound.Connection || parent.Phase != project.ExecutionInvestigate || parent.Role != project.RoleCoder || parent.Stage != factory.AssignmentFinished || parent.Outcome != "succeeded" || parent.Result == nil || !parent.Result.Reported {
		return nil, errors.New("scope unavailable")
	}
	parentRun, err := st.FactoryRun(ctx, parent.Run)
	if err != nil || parentRun.Validate() != nil || parentRun.ID != parent.Run || parentRun.ProjectID != parent.ProjectID || parentRun.Role != parent.Role || parentRun.InputSHA != parent.SourceCommit || !parentRun.Reconciled || parentRun.Outcome != "succeeded" || !parentRun.CredentialReturned {
		return nil, errors.New("scope unavailable")
	}
	view, err := st.FactoryRunView(ctx, in.RunID)
	if err != nil || view.Repository != bound.Repository || view.Issue != bound.Issue || view.Attempt != a.ID {
		return nil, errors.New("scope unavailable")
	}
	selection, err := control.ContextSelectionStatus(ctx, st, run)
	if err != nil {
		return nil, err
	}
	return &factoryContextResponse{Phase: a.Phase, Role: a.Role, SourceCommit: a.SourceCommit, Preparation: a.Preparation, CurrentGeneration: generation, ScopeSHA: scopeHash, Format: 1, ToolSource: FactoryContextToolRevision, RunID: run.ID, AssignmentID: a.ID, Repository: bound.Repository, Issue: bound.Issue, Project: bound.Project, Root: a.AttemptRoot, ContextPending: run.ContextPending, Reconciled: run.Reconciled, CredentialDelegated: run.CredentialDelegated, NativeBinding: run.IdentityBinding != nil, ContextSelection: selection}, nil
}

func observeFactoryContextOwnerOperation(ctx context.Context, st *store.Store, registered string, bound *factoryContextScope, a factory.Assignment) (int64, error) {
	if a.ActorID != bound.Owner || a.Repository != bound.Repository || a.Issue != bound.Issue || a.ProjectID != bound.Project || a.Connection != bound.Connection || a.Acceptance != bound.Acceptance || a.Validate() != nil {
		return 0, errors.New("owner scope unavailable")
	}
	db, err := sql.Open("pgx", registered)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var owner, generation int64
	var state string
	if db.QueryRowContext(ctx, "SELECT owner_id,generation,state FROM identity_connections WHERE id=$1", bound.Connection).Scan(&owner, &generation, &state) != nil || owner != bound.Owner || generation < 1 || state != "ready" {
		return 0, errors.New("current connection unavailable")
	}
	commandID := bound.PolicyCommandID
	command, err := st.FactoryCommand(ctx, commandID)
	if err != nil || command.ID != commandID || command.Type != factory.CommandPolicy || command.Target != "repository/"+strconv.FormatInt(bound.Repository, 10)+"/policy" || command.Principal != "native:"+strconv.FormatInt(bound.Owner, 10) || command.Finished == "" || command.Validate() != nil {
		return 0, errors.New("collector operation unavailable")
	}
	return generation, nil
}

func observeFactoryReviewContext(ctx context.Context, st *store.Store, bound *factoryContextScope, scopeHash string, in factoryContextRequest, run factory.Run, a factory.Assignment, generation int64) (*factoryContextResponse, error) {
	if run.Validate() != nil || a.Stage != factory.AssignmentAssigned || !run.ContextPending || run.Reconciled || run.Outcome != "" || run.CredentialDelegated || run.IdentityBinding != nil || !time.Now().Before(run.Deadline) {
		return nil, errors.New("review not staged")
	}
	publicationBinding, err := st.PublicationByAssignment(ctx, a.AttemptRoot)
	if err != nil {
		return nil, err
	}
	implementationBinding, err := st.AssignmentByRun(ctx, publicationBinding.Run)
	if err != nil {
		return nil, err
	}
	b := struct{ Run, Assignment, Root, Publication, Candidate, Preparation, Implementation, ImplementationRun string }{in.RunID, in.AssignmentID, a.AttemptRoot, publicationBinding.ID, publicationBinding.Candidate, a.Preparation, implementationBinding.ID, publicationBinding.Run}
	if !factory.ValidID(b.Run) || !factory.ValidID(b.Assignment) || !factory.ValidID(b.Root) || !factory.ValidID(b.Publication) || !factory.ValidCommit(b.Candidate) || !factory.ValidPreparationRef(b.Preparation) || !factory.ValidID(b.Implementation) || !factory.ValidID(b.ImplementationRun) {
		return nil, errors.New("review scope unavailable")
	}
	if in.RunID != b.Run || in.AssignmentID != b.Assignment || run.ID != b.Run || a.ID != b.Assignment || a.Run != b.Run || a.Repository != bound.Repository || a.Issue != bound.Issue || a.ProjectID != bound.Project || run.ProjectID != bound.Project || a.AttemptRoot != b.Root || a.ParentAssignment != b.Root || a.PublicationAssignment != b.Root || a.Acceptance != bound.Acceptance || a.SourceCommit != b.Candidate || run.InputSHA != b.Candidate || a.Preparation != b.Preparation || a.Connection != bound.Connection || a.Phase != project.ExecutionReview || a.Role != project.RoleReviewer || run.Role != project.RoleReviewer || a.Validate() != nil {
		return nil, errors.New("review scope unavailable")
	}
	root, err := st.Assignment(ctx, b.Root)
	if err != nil || root.Validate() != nil || root.ID != b.Root || root.PublicationAssignment != root.ID || root.ActorID != bound.Owner || root.AttemptRoot != b.Root || root.Repository != bound.Repository || root.Issue != bound.Issue || root.ProjectID != bound.Project || root.SourceCommit != bound.SourceCommit || root.Acceptance != bound.Acceptance || root.Preparation != bound.Preparation || root.Connection != bound.Connection || root.Phase != project.ExecutionInvestigate || root.Role != project.RoleCoder || root.Stage != factory.AssignmentFinished || root.Outcome != factory.Succeeded || root.Result == nil || !root.Result.Reported {
		return nil, errors.New("review root unavailable")
	}
	rootRun, err := st.FactoryRun(ctx, root.Run)
	if err != nil || rootRun.Validate() != nil || rootRun.ID != root.Run || rootRun.ProjectID != root.ProjectID || rootRun.Role != root.Role || rootRun.InputSHA != root.SourceCommit || !rootRun.Reconciled || rootRun.Outcome != factory.Succeeded || !rootRun.CredentialReturned {
		return nil, errors.New("review root unavailable")
	}
	publication, err := st.PublicationByAssignment(ctx, b.Root)
	if err != nil || publication.ID != b.Publication || publication.AssignmentID != b.Root || publication.Run != b.ImplementationRun || publication.ProjectID != bound.Project || publication.Repository != bound.Repository || publication.Issue != bound.Issue || publication.Acceptance != bound.Acceptance || publication.Candidate != b.Candidate || publication.Stage != factory.PublicationPublished || publication.Validate() != nil {
		return nil, errors.New("review publication unavailable")
	}
	implementation, err := st.Assignment(ctx, b.Implementation)
	if err != nil || implementation.Validate() != nil || implementation.ID != b.Implementation || implementation.Run != b.ImplementationRun || implementation.ActorID != bound.Owner || implementation.AttemptRoot != b.Root || implementation.ParentAssignment != b.Root || implementation.PublicationAssignment != b.Root || implementation.Repository != bound.Repository || implementation.Issue != bound.Issue || implementation.ProjectID != bound.Project || implementation.SourceCommit != bound.SourceCommit || implementation.Acceptance != bound.Acceptance || implementation.Preparation != bound.Preparation || implementation.Connection != bound.Connection || implementation.Phase != project.ExecutionImplement || implementation.Role != project.RoleCoder || implementation.Stage != factory.AssignmentFinished || implementation.Outcome != factory.Succeeded || implementation.Result == nil || !implementation.Result.Reported || implementation.Result.Status != "completed" || implementation.Result.Candidate != b.Candidate {
		return nil, errors.New("review implementation unavailable")
	}
	implementationRun, err := st.FactoryRun(ctx, b.ImplementationRun)
	if err != nil || implementationRun.Validate() != nil || implementationRun.ID != implementation.Run || implementationRun.ProjectID != implementation.ProjectID || implementationRun.Role != implementation.Role || implementationRun.InputSHA != implementation.SourceCommit || !implementationRun.Reconciled || implementationRun.Outcome != factory.Succeeded || !implementationRun.CredentialReturned {
		return nil, errors.New("review implementation unavailable")
	}
	prep, err := st.CandidatePreparation(ctx, b.Preparation)
	if err != nil || prep.Preparation.Preparation.ID != b.Preparation || prep.Preparation.Preparation.Project != bound.Project || prep.Preparation.Preparation.Role != project.RoleReviewer || prep.Preparation.Preparation.SourceCommit != b.Candidate || prep.Admission.ActorID != bound.Owner || prep.Admission.Repository != bound.Repository || prep.Admission.Issue != bound.Issue || prep.Admission.Project != bound.Project || prep.Admission.OwnerAssignment != b.Root || prep.Admission.ChildAssignment != b.Assignment || prep.Admission.AttemptRoot != b.Root || prep.Admission.PublicationID != b.Publication || prep.Admission.Connection != bound.Connection {
		return nil, errors.New("review preparation unavailable")
	}
	view, err := st.FactoryRunView(ctx, in.RunID)
	if err != nil || view.Repository != bound.Repository || view.Issue != bound.Issue || view.Attempt != a.ID {
		return nil, errors.New("review view unavailable")
	}

	standing, err := st.Preparation(ctx, bound.ReviewerPreparation)
	if err != nil || standing.Preparation.Project != bound.Project || standing.Preparation.Role != project.RoleReviewer || standing.Preparation.SourceCommit != bound.SourceCommit || !standing.State.Ready || standing.State.Stopped || !prep.Preparation.Preparation.Requirements.Equal(standing.Preparation.Requirements) || prep.Preparation.Preparation.Approval != standing.Preparation.Approval || prep.Preparation.Preparation.SetupDigest != standing.Preparation.SetupDigest || prep.Stopped || !prep.Preparation.State.Ready || prep.Preparation.State.Stopped || prep.Preparation.State.SetupExit == nil || *prep.Preparation.State.SetupExit != 0 || prep.Preparation.State.CheckExit == nil || *prep.Preparation.State.CheckExit != 0 || time.Now().Unix() >= prep.NotAfter {
		return nil, errors.New("review preparation not eligible")
	}
	selection, err := control.ContextSelectionStatus(ctx, st, run)
	if err != nil {
		return nil, err
	}
	return &factoryContextResponse{Phase: a.Phase, Role: a.Role, SourceCommit: a.SourceCommit, Preparation: a.Preparation, Publication: publication.ID, Candidate: publication.Candidate, CurrentGeneration: generation, ScopeSHA: scopeHash, Format: 1, ToolSource: FactoryContextToolRevision, RunID: run.ID, AssignmentID: a.ID, Repository: bound.Repository, Issue: bound.Issue, Project: bound.Project, Root: a.AttemptRoot, ContextPending: run.ContextPending, Reconciled: run.Reconciled, CredentialDelegated: run.CredentialDelegated, NativeBinding: run.IdentityBinding != nil, ContextSelection: selection}, nil
}

// RunFactoryContextStatus reads one protected scope and emits canonical selected context metadata.
func RunFactoryContextStatus(args []string, stdin io.Reader, stdout io.Writer) (resultErr error) {
	defer func() {
		if recover() != nil {
			resultErr = errors.New("canonical_context_status_unavailable")
		}
		if resultErr != nil {
			resultErr = errors.New("canonical_context_status_unavailable")
		}
	}()
	if os.Geteuid() != 0 || len(args) != 1 || !factory.ValidCommit(FactoryContextToolRevision) {
		return errors.New("canonical_context_status_unavailable")
	}
	input, err := io.ReadAll(io.LimitReader(stdin, 1025))
	if err != nil || len(input) > 1024 {
		return errors.New("canonical_context_status_unavailable")
	}
	var in factoryContextRequest
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&in) != nil || decoder.Decode(new(any)) != io.EOF || !factory.ValidID(in.RunID) || !factory.ValidID(in.AssignmentID) {
		return errors.New("canonical_context_status_unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := observeFactoryContext(ctx, args[0], in)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(out)
	if err != nil || len(encoded) > 8192 {
		return errors.New("canonical_context_status_unavailable")
	}
	_, err = stdout.Write(append(encoded, '\n'))
	return err
}
