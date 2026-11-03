package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/execution"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/host"
)

// scriptedFactoryOutput serves one recorded log in bounded slices with a
// fixed terminal phase, like the native host output route.
type scriptedFactoryOutput struct {
	log      []byte
	terminal bool
	phase    string
	limit    int
	foreign  bool
	reads    atomic.Int64
	mutate   func(*project.FactoryOutputState)
}

func (s *scriptedFactoryOutput) RoundTrip(r *http.Request) (*http.Response, error) {
	s.reads.Add(1)
	var in project.FactoryOutput
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, err
	}
	if r.URL.Path != "/factory-output" {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
	}
	s.limit = in.Limit
	start := in.Offset
	if start < 0 {
		start = 0
	}
	if start > int64(len(s.log)) {
		start = int64(len(s.log))
	}
	end := start + int64(in.Limit)
	if end > int64(len(s.log)) {
		end = int64(len(s.log))
	}
	chunk := s.log[start:end]
	out := project.FactoryOutputState{
		ID: in.ID, Project: in.Project, Phase: s.phase, Terminal: s.terminal,
		Total: int64(len(s.log)), Offset: start, Next: start + int64(len(chunk)),
	}
	if s.foreign {
		out.ID, out.Project = strings.Repeat("b", 32), "another-workspace"
	}
	if len(chunk) > 0 {
		out.Data = base64.StdEncoding.EncodeToString(chunk)
	}
	if s.mutate != nil {
		s.mutate(&out)
	}
	buf, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(buf)), Header: make(http.Header), Request: r}, nil
}

func factoryOutputTestPair(t *testing.T) (serverConn, clientConn *websocket.Conn, cleanup func()) {
	t.Helper()
	connCh := make(chan *websocket.Conn, 1)
	errCh := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			errCh <- err
			return
		}
		connCh <- conn
	}))
	dial, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client, _, err := websocket.Dial(dial, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	cancel()
	if err != nil {
		server.Close()
		t.Fatalf("dial test peer: %v", err)
	}
	client.SetReadLimit(1 << 20)
	select {
	case conn := <-connCh:
		return conn, client, func() {
			client.CloseNow()
			conn.CloseNow()
			server.Close()
		}
	case err := <-errCh:
		server.Close()
		t.Fatalf("accept test peer: %v", err)
	case <-time.After(10 * time.Second):
		server.Close()
		t.Fatal("accept test peer: timeout")
	}
	return nil, nil, nil
}

func readFactoryTestFrame(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, body, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	var frame map[string]any
	if err := json.Unmarshal(body, &frame); err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	return frame
}

func factoryOutputTestBytes(t *testing.T, frame map[string]any) []byte {
	t.Helper()
	if frame["type"] != "output" {
		t.Fatalf("expected output frame, got %v", frame["type"])
	}
	raw, err := base64.StdEncoding.DecodeString(frame["data"].(string))
	if err != nil {
		t.Fatalf("decode output data: %v", err)
	}
	return raw
}

// Host channel admission is declared here; the real WebSocket and PostgreSQL
// attachment exercise the shared protocol. The host owns cancellation/draining.
func sharedFactoryOutputPeer(t *testing.T, transport factoryhttpapi.Output) (*websocket.Conn, <-chan struct{}, context.CancelFunc) {
	t.Helper()
	server, client, cleanup := factoryOutputTestPair(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		transport.Serve(ctx, server)
	}()
	t.Cleanup(func() {
		cancel()
		client.CloseNow()
		server.CloseNow()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("shared output reader did not drain")
		}
		cleanup()
	})
	return client, done, cancel
}

func startSharedFactoryOutput(t *testing.T, attachment *factoryservice.OutputAttachment) *websocket.Conn {
	t.Helper()
	transport := factoryhttpapi.Output{Attachment: attachment,
		Handshake: func(in factoryhttpapi.OutputHandshake) bool {
			return in.RepositoryID == "9223372036854775807" && in.SessionGeneration == "declared-profile"
		},
		Current: func(ctx context.Context) bool { return ctx.Err() == nil },
	}
	conn, _, _ := sharedFactoryOutputPeer(t, transport)
	encoded, _ := json.Marshal(factoryhttpapi.OutputHandshake{RunID: attachment.RunID(), RepositoryID: "9223372036854775807", SessionGeneration: "declared-profile"})
	if err := conn.Write(t.Context(), websocket.MessageText, encoded); err != nil {
		t.Fatal(err)
	}
	return conn
}

// Real private PostgreSQL backs the attachment and cursor journeys. Only current
// runtime authority is declared; output still crosses the production host client
// and actual WebSocket connection. There is no agent launch or host process.
func admittedOutputFixture(t *testing.T, s *API, workspace, runID string) factoryViewerIdentity {
	t.Helper()
	if s.Store == nil {
		fx, _ := journeyMux(t, nil)
		s.Store = fx.db
	}
	if _, err := s.Store.FactoryRun(t.Context(), runID); errors.Is(err, store.ErrNotFound) {
		now := time.Now()
		if err := s.Store.RecordFactoryRun(t.Context(), factory.Run{ID: runID, ProjectID: workspace, Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("c", 64), Harness: "muse", Model: "declared-fixture"}); err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	identity := factoryViewerIdentity{project: store.Project{ID: workspace}, runID: runID}
	const actor collaboration.ActorID = 9007199254740993
	reader := s.factoryOutputReader(nil, identity, actor)
	reader.Scope = func(ctx context.Context, admitted collaboration.ActorID, run factory.Run) (collaboration.RepositoryID, error) {
		if admitted != actor || run.ID != runID || run.ProjectID != workspace {
			return 0, errors.New("declared output scope denied")
		}
		return collaboration.RepositoryID(9223372036854775807), nil
	}
	reader.Access = func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID, run factory.Run) error {
		if admitted != actor || repo != 9223372036854775807 {
			return errors.New("declared output authority denied")
		}
		return nil
	}
	var err error
	identity.output, err = reader.Open(t.Context(), actor, runID)
	if err != nil {
		t.Fatal(err)
	}
	return identity
}

func TestSharedOutputAdmissionAndBounds(t *testing.T) {
	fx, _ := journeyMux(t, nil)
	workspace, runID := "p0123456789abcdef01234567", strings.Repeat("a", 32)
	script := &scriptedFactoryOutput{log: []byte{0, 255, 1, 2}, terminal: true, phase: project.FactoryCompleted}
	fx.api.Host = &host.Client{HTTP: &http.Client{Transport: script}}
	identity := admittedOutputFixture(t, fx.api, workspace, runID)
	const actor collaboration.ActorID = 9007199254740993
	const repository collaboration.RepositoryID = 9223372036854775807
	reader := fx.api.factoryOutputReader(nil, identity, actor)
	allowed, repo := true, repository
	denied := errors.New("declared current output authority denied")
	reader.Scope = func(ctx context.Context, admitted collaboration.ActorID, run factory.Run) (collaboration.RepositoryID, error) {
		if admitted != actor || run.ID != runID || run.ProjectID != workspace {
			return 0, denied
		}
		return repo, nil
	}
	reader.Access = func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID, run factory.Run) error {
		if admitted != actor || repo != repository || !allowed {
			return denied
		}
		return nil
	}
	attachment, err := reader.Open(t.Context(), actor, runID)
	if err != nil {
		t.Fatal(err)
	}
	view, err := attachment.Read(t.Context(), 0)
	if err != nil || !view.Done || view.Next != 4 || view.Output == nil {
		t.Fatal("initial bounded output", view, err)
	}
	decoded, _ := base64.StdEncoding.DecodeString(view.Output.Data)
	if !bytes.Equal(decoded, script.log) || script.limit != project.MaxFactoryOutputRead {
		t.Fatal("output bytes/read bound changed")
	}
	// A declared native source uses the same output record without a host
	// client or container. This checks the seam, not an installed process.
	native, err := fx.db.FactoryRun(t.Context(), runID)
	if err != nil {
		t.Fatal(err)
	}
	native.ID, native.Image, native.Harness = strings.Repeat("b", 32), "", "muse-1.0.0"
	native.Native = &execution.NativeRecord{Tool: execution.NativeTool{Harness: "muse", Version: "1.0.0", SHA256: strings.Repeat("e", 64), Executable: "/home/vince/declared-tools/muse"}, Retirement: execution.Running,
		Provenance: execution.NativeProvenance{TrustMode: "same_user_trusted_host", Profile: collaboration.ProfileID(factory.NewID()), InstanceID: factory.NewID(), WorktreeID: "declared-worktree", Custody: execution.NativeCustody{State: execution.NotAcquired},
			Process: &execution.NativeProcess{InvocationID: strings.Repeat("d", 32), SystemInstance: "declared-boot", StartIdentity: "declared-start", PID: 123, StartedAt: native.Started, Executable: "/home/vince/declared-tools/muse", ExecutableSHA256: strings.Repeat("e", 64)}}}
	if err := fx.db.RecordFactoryRun(t.Context(), native); err != nil {
		t.Fatal(err)
	}
	nativeReader := reader
	nativeReader.Scope = func(ctx context.Context, admitted collaboration.ActorID, run factory.Run) (collaboration.RepositoryID, error) {
		if admitted != actor || run.ID != native.ID || run.ProjectID != workspace {
			return 0, denied
		}
		return repository, nil
	}
	nativeContainer := ""
	nativeReader.Source = func(ctx context.Context, admitted collaboration.ActorID, run factory.Run, cursor int64, limit int) (project.FactoryOutputState, error) {
		if admitted != actor || run.Native == nil || limit != project.MaxFactoryOutputRead {
			return project.FactoryOutputState{}, denied
		}
		return project.FactoryOutputState{ID: run.ID, Project: run.ProjectID, Invocation: run.InvocationID(), Phase: project.FactoryRunning, Live: true, Container: nativeContainer, Total: 4, Next: 4, Data: base64.StdEncoding.EncodeToString(script.log)}, nil
	}
	nativeAttachment, err := nativeReader.Open(t.Context(), actor, native.ID)
	if err != nil {
		t.Fatal(err)
	}
	nativeView, err := nativeAttachment.Read(t.Context(), 0)
	if err != nil || nativeView.Done || nativeView.Status.Invocation != strings.Repeat("d", 32) || nativeView.Status.Container != "" || nativeView.Status.Unit != "" {
		t.Fatal("native output inherited appliance facts", nativeView, err)
	}
	encodedNative, _ := json.Marshal(nativeView)
	if bytes.Contains(encodedNative, []byte("declared-tools")) || bytes.Contains(encodedNative, []byte("declared-start")) {
		t.Fatal("native private process metadata entered output")
	}
	nativeContainer = "invented-container"
	if view, err := nativeAttachment.Read(t.Context(), 0); view != nil || !errors.Is(err, factoryservice.ErrRunOutputChanged) {
		t.Fatal("native output accepted container identity", view, err)
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := fx.raw.QueryRowContext(t.Context(), `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_runs r),
		 (SELECT count(*) FROM factory_commands),(SELECT count(*) FROM factory_usage),(SELECT count(*) FROM projects))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := snapshot()
	script.mutate = func(out *project.FactoryOutputState) { out.Data += strings.Repeat("\n", 20000) }
	view, err = attachment.Read(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(view.Output)
	if len(encoded) > 32768 || view.Output.Data != base64.StdEncoding.EncodeToString(script.log) {
		t.Fatal("noncanonical producer base64 broke wire bound", len(encoded))
	}
	for _, invalid := range []func(*project.FactoryOutputState){
		func(out *project.FactoryOutputState) { out.Data = "!" },
		func(out *project.FactoryOutputState) { out.Next++; out.Total++ },
		func(out *project.FactoryOutputState) { out.Offset = 1 },
		func(out *project.FactoryOutputState) { out.Total = -1 },
		func(out *project.FactoryOutputState) { out.Live = true },
		func(out *project.FactoryOutputState) {
			out.Data = base64.StdEncoding.EncodeToString(make([]byte, project.MaxFactoryOutputRead+1))
		},
	} {
		script.mutate = invalid
		if view, err := attachment.Read(t.Context(), 0); view != nil || !errors.Is(err, factoryservice.ErrRunOutputInvalid) {
			t.Fatal("invalid source slice served", view, err)
		}
	}
	// Both truncation on attach and a cursor beyond the file preserve the
	// source's exact offsets. Neither an empty gap nor a terminal partial read
	// can invent bytes or end before the final byte.
	script.mutate = func(out *project.FactoryOutputState) { out.Offset, out.Next, out.Total, out.Truncated = 5, 9, 9, true }
	if view, err := attachment.Read(t.Context(), 0); err != nil || !view.Done || !view.Output.Truncated || view.Output.Cursor != 5 || view.Next != 9 {
		t.Fatal("truncated attach cursor", view, err)
	}
	script.mutate = func(out *project.FactoryOutputState) { out.Gap = true }
	if view, err := attachment.Read(t.Context(), 99); err != nil || !view.Done || !view.Output.Gap || view.Next != 4 {
		t.Fatal("gap cursor", view, err)
	}
	script.mutate = func(out *project.FactoryOutputState) { allowed = false }
	if view, err := attachment.Read(t.Context(), 0); view != nil || !errors.Is(err, denied) {
		t.Fatal("source read outlived authority", view, err)
	}
	script.mutate = nil
	reads := script.reads.Load()
	if view, err := attachment.Read(t.Context(), 0); view != nil || !errors.Is(err, denied) || script.reads.Load() != reads {
		t.Fatal("denied output reached source", view, err)
	}
	allowed = true
	repo--
	if view, err := attachment.Read(t.Context(), 0); view != nil || !errors.Is(err, factoryservice.ErrRunOutputChanged) {
		t.Fatal("attachment followed another repository", view, err)
	}
	repo = repository
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if view, err := attachment.Read(cancelled, 0); view != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled output served", view, err)
	}
	if view, err := attachment.Read(t.Context(), -1); view != nil || !errors.Is(err, factoryservice.ErrRunOutputCursor) {
		t.Fatal("invalid cursor served", view, err)
	}
	missing := reader
	missing.Access = nil
	if view, err := missing.Open(t.Context(), actor, runID); view != nil || !errors.Is(err, factoryservice.ErrRunOutputUnavailable) {
		t.Fatal("missing authority opened output", view, err)
	}
	script.mutate = func(out *project.FactoryOutputState) { out.Invocation = strings.Repeat("c", 32) }
	if _, err := attachment.Read(t.Context(), 0); err != nil {
		t.Fatal(err)
	}
	script.mutate = func(out *project.FactoryOutputState) { out.Invocation = strings.Repeat("d", 32) }
	if view, err := attachment.Read(t.Context(), 0); view != nil || !errors.Is(err, factoryservice.ErrRunOutputChanged) {
		t.Fatal("changed process incarnation served", view, err)
	}
	if snapshot() != before {
		t.Fatal("output reads launched, retired, charged or changed the run")
	}
	// Reattachment against the retained store resumes bytes without any worker
	// start/resume API or native Project. Runtime authority remains required.
	if err := fx.db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	reader.Store = reopened
	script.mutate = func(out *project.FactoryOutputState) { out.Invocation = strings.Repeat("c", 32) }
	reattached, err := reader.Open(t.Context(), actor, runID)
	if err != nil {
		t.Fatal(err)
	}
	if view, err := reattached.Read(t.Context(), 3); err != nil || view.Next != 4 || view.Output.Cursor != 3 {
		t.Fatal("retained output resume", view, err)
	}
	allowed = false
	if view, err := reattached.Read(t.Context(), 3); view != nil || !errors.Is(err, denied) {
		t.Fatal("retained output bypassed admission", view, err)
	}
	if snapshot() != before {
		t.Fatal("reopen/reattachment changed recorded work")
	}
	// A declared native output source uses the reopened ledger and shared wire
	// protocol with its own profile-generation label. No native session, host
	// client, Project, provider process or installed executor is involved.
	allowed = true
	var admitted, withdrawDuringRead, blockRead atomic.Bool
	admitted.Store(true)
	var nativeReads atomic.Int64
	reading := make(chan struct{}, 1)
	standalone := nativeReader
	standalone.Store = reopened
	standalone.Source = func(ctx context.Context, selected collaboration.ActorID, run factory.Run, cursor int64, limit int) (project.FactoryOutputState, error) {
		if selected != actor || run.ID != native.ID || run.Native == nil || limit != project.MaxFactoryOutputRead {
			return project.FactoryOutputState{}, denied
		}
		nativeReads.Add(1)
		if blockRead.Load() {
			reading <- struct{}{}
			<-ctx.Done()
			return project.FactoryOutputState{}, ctx.Err()
		}
		start := min(cursor, int64(len(script.log)))
		out := project.FactoryOutputState{ID: run.ID, Project: run.ProjectID, Invocation: run.InvocationID(), Phase: project.FactoryCompleted, Terminal: true,
			Total: int64(len(script.log)), Offset: start, Next: int64(len(script.log)), Gap: cursor > start,
			Data: base64.StdEncoding.EncodeToString(script.log[start:])}
		if withdrawDuringRead.Load() {
			admitted.Store(false)
		}
		return out, nil
	}
	bound, err := standalone.Open(t.Context(), actor, native.ID)
	if err != nil {
		t.Fatal(err)
	}
	transport := factoryhttpapi.Output{Attachment: bound,
		Handshake: func(in factoryhttpapi.OutputHandshake) bool {
			return in.RepositoryID == "9223372036854775807" && in.SessionGeneration == "declared-profile"
		},
		Current: func(ctx context.Context) bool { return ctx.Err() == nil && admitted.Load() },
	}
	good := factoryhttpapi.OutputHandshake{RunID: native.ID, RepositoryID: "9223372036854775807", SessionGeneration: "declared-profile", Cursor: 3}
	encode := func(in factoryhttpapi.OutputHandshake) []byte {
		body, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	waitDone := func(done <-chan struct{}) {
		t.Helper()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("shared output transport did not drain")
		}
	}
	conn, done, _ := sharedFactoryOutputPeer(t, transport)
	if err := conn.Write(t.Context(), websocket.MessageText, encode(good)); err != nil {
		t.Fatal(err)
	}
	status := readFactoryTestFrame(t, conn)
	if status["type"] != "status" || status["run_id"] != native.ID || status["container"] != "" || status["unit"] != "" {
		t.Fatal("native output wire invented appliance facts", status)
	}
	frame := readFactoryTestFrame(t, conn)
	if frame["cursor"] != float64(3) || frame["next"] != float64(4) || !bytes.Equal(factoryOutputTestBytes(t, frame), script.log[3:]) {
		t.Fatal("shared retained native cursor changed bytes", frame)
	}
	if closed := readFactoryTestFrame(t, conn); closed["reason"] != "eof" {
		t.Fatal("native output did not finish its retained bytes", closed)
	}
	waitDone(done)
	readsBeforeRefusals := nativeReads.Load()
	type handshakeCase struct {
		kind websocket.MessageType
		body []byte
	}
	cases := []handshakeCase{
		{websocket.MessageBinary, encode(good)},
		{websocket.MessageText, append(encode(good), encode(good)...)},
		{websocket.MessageText, []byte(`{"run_id":"` + native.ID + `","run_id":"` + native.ID + `"}`)},
		{websocket.MessageText, []byte(`{"invented":true}`)},
	}
	for _, alter := range []func(*factoryhttpapi.OutputHandshake){
		func(in *factoryhttpapi.OutputHandshake) { in.RunID = runID },
		func(in *factoryhttpapi.OutputHandshake) { in.RepositoryID = "7" },
		func(in *factoryhttpapi.OutputHandshake) { in.SessionGeneration = "another-account" },
		func(in *factoryhttpapi.OutputHandshake) { in.Cursor = -1 },
		func(in *factoryhttpapi.OutputHandshake) { in.Cursor = project.MaxFactoryOutputOffset + 1 },
		func(in *factoryhttpapi.OutputHandshake) { in.SessionGeneration = strings.Repeat("x", 4096) },
	} {
		bad := good
		alter(&bad)
		cases = append(cases, handshakeCase{websocket.MessageText, encode(bad)})
	}
	for _, invalid := range cases {
		conn, done, _ := sharedFactoryOutputPeer(t, transport)
		if err := conn.Write(t.Context(), invalid.kind, invalid.body); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		_, _, err := conn.Read(ctx)
		cancel()
		if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
			t.Fatal("invalid output handshake reached the stream", string(invalid.body), err)
		}
		waitDone(done)
	}
	for _, missing := range []factoryhttpapi.Output{
		{Attachment: bound, Handshake: transport.Handshake},
		{Attachment: bound, Current: transport.Current},
		{Handshake: transport.Handshake, Current: transport.Current},
	} {
		conn, done, _ := sharedFactoryOutputPeer(t, missing)
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		_, _, err := conn.Read(ctx)
		cancel()
		if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
			t.Fatal("missing output port granted a stream", err)
		}
		waitDone(done)
	}
	if nativeReads.Load() != readsBeforeRefusals {
		t.Fatal("refused handshake or missing authority reached native output")
	}
	withdrawDuringRead.Store(true)
	conn, done, _ = sharedFactoryOutputPeer(t, transport)
	if err := conn.Write(t.Context(), websocket.MessageText, encode(good)); err != nil {
		t.Fatal(err)
	}
	if closed := readFactoryTestFrame(t, conn); closed["type"] != "closed" || closed["reason"] != "authority_lost" {
		t.Fatal("account loss during read exposed private output", closed)
	}
	waitDone(done)
	withdrawDuringRead.Store(false)
	admitted.Store(true)
	blockRead.Store(true)
	conn, done, cancelStream := sharedFactoryOutputPeer(t, transport)
	if err := conn.Write(t.Context(), websocket.MessageText, encode(good)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-reading:
	case <-time.After(5 * time.Second):
		t.Fatal("native output source did not enter the bounded read")
	}
	cancelStream()
	waitDone(done)
	if snapshot() != before {
		t.Fatal("shared native WebSocket output changed retained work or provenance")
	}
}

func TestFactoryOutputNativeStreamRevocation(t *testing.T) {
	fx, _ := journeyMux(t, nil)
	if err := fx.db.UpsertUser(t.Context(), store.User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	const workspace = "p0123456789abcdef01234567"
	if err := fx.db.CreateProject(t.Context(), store.Project{ID: workspace, RepositoryID: 7, OwnerID: 1, Name: "output", Repository: "alice/Alpha"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	run := factory.Run{ID: factory.NewID(), ProjectID: workspace, Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("c", 64), Harness: "muse", Model: "declared-fixture"}
	if err := fx.db.RecordFactoryRun(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	script := &scriptedFactoryOutput{log: []byte{0, 1, 255, 2}, phase: project.FactoryRunning}
	fx.api.Host = &host.Client{HTTP: &http.Client{Transport: script}}
	server := httptest.NewServer(fx.api.ExtensionHandler())
	t.Cleanup(server.Close)
	authority, _ := json.Marshal(extensions.Authority{ExtensionID: "soda", InstanceID: "output-test", SessionGeneration: "generation-1",
		Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: "get"}, Actor: extensions.Actor{ID: "1", Username: "alice"}})
	headers := http.Header{}
	headers.Set(extensions.ContextHeader, string(authority))
	headers.Set(extensions.AdmissionHeader, "test-admission")
	headers.Set(extensions.SessionGenerationHeader, "generation-1")
	headers.Set("Origin", "https://soda.test")
	dial := func(cursor int64) *websocket.Conn {
		t.Helper()
		conn, response, err := websocket.Dial(t.Context(), "ws"+strings.TrimPrefix(server.URL, "http")+"/api/factory/runs/"+run.ID+"/output", &websocket.DialOptions{HTTPHeader: headers})
		if err != nil {
			if response != nil {
				t.Fatal("native output upgrade", response.StatusCode, err)
			}
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.CloseNow() })
		handshake, _ := json.Marshal(factoryhttpapi.OutputHandshake{RunID: run.ID, RepositoryID: "7", SessionGeneration: "generation-1", Cursor: cursor})
		if err := conn.Write(t.Context(), websocket.MessageText, handshake); err != nil {
			t.Fatal(err)
		}
		return conn
	}
	conn := dial(0)
	if status := readFactoryTestFrame(t, conn); status["type"] != "status" {
		t.Fatal(status)
	}
	if output := factoryOutputTestBytes(t, readFactoryTestFrame(t, conn)); !bytes.Equal(output, script.log) {
		t.Fatal("native binary output changed", output)
	}
	reads := script.reads.Load()
	fx.permission.Store("read")
	if closed := readFactoryTestFrame(t, conn); closed["type"] != "closed" || closed["reason"] != "authority_lost" {
		t.Fatal("revoked native output remained open", closed)
	}
	if script.reads.Load() != reads {
		t.Fatal("revocation reached output source")
	}
	fx.permission.Store("admin")
	reconnected := dial(3)
	if status := readFactoryTestFrame(t, reconnected); status["type"] != "status" {
		t.Fatal(status)
	}
	frame := readFactoryTestFrame(t, reconnected)
	if frame["cursor"] != float64(3) || frame["next"] != float64(4) || !bytes.Equal(factoryOutputTestBytes(t, frame), script.log[3:]) {
		t.Fatal("reattachment restarted or changed output", frame)
	}
	reads = script.reads.Load()
	if err := reconnected.Write(t.Context(), websocket.MessageText, []byte(`{"type":"end"}`)); err != nil {
		t.Fatal(err)
	}
	check, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if _, _, err := reconnected.Read(check); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatal("read-only output accepted End/input", err)
	}
	if script.reads.Load() != reads {
		t.Fatal("viewer input reached output source")
	}
	stored, err := fx.db.FactoryRun(t.Context(), run.ID)
	beforeJSON, _ := json.Marshal(run)
	afterJSON, _ := json.Marshal(stored)
	if err != nil || string(beforeJSON) != string(afterJSON) {
		t.Fatal("output attachment changed or retired its run", err)
	}
}

func TestFactoryOutputRejectsForeignSlice(t *testing.T) {
	fx, _ := journeyMux(t, nil)
	now := time.Now()
	run := factory.Run{ID: strings.Repeat("a", 32), ProjectID: "p0123456789abcdef01234567", Role: "coder", InputSHA: strings.Repeat("a", 40),
		Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("c", 64), Harness: "muse", Model: "declared-fixture"}
	if err := fx.db.RecordFactoryRun(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	script := &scriptedFactoryOutput{log: []byte("foreign private output"), terminal: true, phase: project.FactoryCompleted, foreign: true}
	fx.api.Host = &host.Client{HTTP: &http.Client{Transport: script}}
	identity := admittedOutputFixture(t, fx.api, run.ProjectID, run.ID)
	clientConn := startSharedFactoryOutput(t, identity.output)
	frame := readFactoryTestFrame(t, clientConn)
	if frame["type"] != "closed" || frame["reason"] != "incarnation_changed" {
		t.Fatal("foreign output slice reached the viewer", frame)
	}
}

func TestFactoryOutputFrameFitsProxyWireLimit(t *testing.T) {
	raw := make([]byte, project.MaxFactoryOutputRead)
	for i := range raw {
		raw[i] = byte(i)
	}
	frame := factoryservice.RunOutputFrame{
		Type: "output", Data: base64.StdEncoding.EncodeToString(raw),
		Cursor: project.MaxFactoryOutputOffset, Next: project.MaxFactoryOutputOffset, Truncated: true,
	}
	body, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) > 32768 {
		t.Fatalf("max output frame is %d wire bytes, above the 32768 proxy limit", len(body))
	}
}

func TestPumpFactorySliceDrainsTerminalOutputBeforeEOF(t *testing.T) {
	log := make([]byte, 2*project.MaxFactoryOutputRead)
	for i := range log {
		log[i] = byte(i % 251)
	}
	script := &scriptedFactoryOutput{log: log, terminal: true, phase: project.FactoryCompleted}
	s := &API{Host: &host.Client{HTTP: &http.Client{Transport: script}}}
	identity := admittedOutputFixture(t, s, "p0123456789abcdef01234567", strings.Repeat("a", 32))
	clientConn := startSharedFactoryOutput(t, identity.output)
	if status := readFactoryTestFrame(t, clientConn); status["type"] != "status" {
		t.Fatalf("first frame: %v", status["type"])
	}
	var got []byte
	first := readFactoryTestFrame(t, clientConn)
	if first["cursor"] != float64(0) || first["next"] != float64(project.MaxFactoryOutputRead) {
		t.Fatal("first terminal slice cursor", first)
	}
	got = append(got, factoryOutputTestBytes(t, first)...)
	final := readFactoryTestFrame(t, clientConn)
	if final["cursor"] != float64(project.MaxFactoryOutputRead) || final["next"] != float64(len(log)) {
		t.Fatal("final terminal slice cursor", final)
	}
	got = append(got, factoryOutputTestBytes(t, final)...)
	if !bytes.Equal(got, log) {
		t.Fatalf("delivered %d bytes, want %d exact log bytes", len(got), len(log))
	}
	closed := readFactoryTestFrame(t, clientConn)
	if closed["type"] != "closed" || closed["reason"] != "eof" {
		t.Fatalf("terminal close: %+v", closed)
	}
}

func TestPumpFactorySliceEndsEmptyTerminalRun(t *testing.T) {
	script := &scriptedFactoryOutput{terminal: true, phase: project.FactoryCompleted}
	s := &API{Host: &host.Client{HTTP: &http.Client{Transport: script}}}
	identity := admittedOutputFixture(t, s, "p0123456789abcdef01234567", strings.Repeat("a", 32))
	clientConn := startSharedFactoryOutput(t, identity.output)
	if status := readFactoryTestFrame(t, clientConn); status["type"] != "status" {
		t.Fatalf("first frame: %v", status["type"])
	}
	closed := readFactoryTestFrame(t, clientConn)
	if closed["type"] != "closed" || closed["reason"] != "eof" {
		t.Fatalf("terminal close: %+v", closed)
	}
}
