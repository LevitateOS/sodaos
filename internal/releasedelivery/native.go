package releasedelivery

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

//go:embed tools.json
var toolLock []byte

type Runner interface {
	Run(context.Context, ...string) ([]byte, error)
}
type Native struct{ Home string }

func (n Native) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/skopeo", args...)
	// No ambient registry credentials, proxy secrets, signing-password environment
	// or user configuration. Auth/passphrases are explicit restricted file inputs.
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + n.Home, "XDG_RUNTIME_DIR=" + n.Home, "LANG=C.UTF-8"}
	var output boundedOutput
	cmd.Stdout = &output
	if e := cmd.Run(); e != nil {
		return nil, ErrUnavailable
	}
	return output.Bytes(), nil
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 2<<20 {
		return 0, ErrRefused
	}
	return b.Buffer.Write(p)
}

func CheckNative(ctx context.Context, r Runner) error {
	var versions map[string]string
	if e := json.Unmarshal(toolLock, &versions); e != nil {
		return e
	}
	b, e := r.Run(ctx, "--version")
	if e != nil || !strings.HasPrefix(string(b), "skopeo version "+versions["skopeo"]+" ") {
		return errors.New("locked native skopeo required")
	}
	return nil
}

func PrivateFile(path string) error {
	if !filepath.IsAbs(path) {
		return ErrRefused
	}
	resolved, e := filepath.EvalSymlinks(path)
	if e != nil || resolved != path {
		return ErrRefused
	}
	st, e := os.Lstat(path)
	if e != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o077 != 0 || st.Size() > 1<<20 {
		return ErrRefused
	}
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return ErrRefused
	}
	parent, e := os.Stat(filepath.Dir(path))
	if e != nil || parent.Mode().Perm()&0o077 != 0 {
		return ErrRefused
	}
	return nil
}

func writeJSON(path string, v any) error {
	b, e := marshal(v)
	if e != nil {
		return e
	}
	return nativebuild.WriteNew(path, b, 0o600)
}

type requirement struct {
	Type           string            `json:"type"`
	KeyDatas       []string          `json:"keyDatas,omitempty"`
	SignedIdentity map[string]string `json:"signedIdentity,omitempty"`
}

func (t Trust) requirement(repo string) (requirement, error) {
	role, e := t.Role(repo)
	if e != nil {
		return requirement{}, e
	}
	keys := []string{}
	for _, key := range t.Keys[role] {
		keys = append(keys, base64.StdEncoding.EncodeToString([]byte(key)))
	}
	return requirement{Type: "sigstoreSigned", KeyDatas: keys, SignedIdentity: map[string]string{"type": "exactRepository", "dockerRepository": repo}}, nil
}

func policyFor(t Trust, repo, transport, scope string) (any, error) {
	req, e := t.requirement(repo)
	if e != nil {
		return nil, e
	}
	return map[string]any{"default": []requirement{{Type: "reject"}}, "transports": map[string]any{transport: map[string]any{scope: []requirement{req}}}}, nil
}

func localPolicy(transport, path string) any {
	return map[string]any{"default": []requirement{{Type: "reject"}}, "transports": map[string]any{transport: map[string]any{path: []requirement{{Type: "insecureAcceptAnything"}}}}}
}

// MergePolicy emits a proposed policy, never installs it. Unrelated Fedora/vendor
// scopes are preserved. Existing more-specific Soda overrides must be reviewed,
// not silently allowed to bypass the new rule.
func MergePolicy(t Trust, original []byte) ([]byte, error) {
	if e := t.Validate(); e != nil {
		return nil, e
	}
	var p struct {
		Default    json.RawMessage                       `json:"default"`
		Transports map[string]map[string]json.RawMessage `json:"transports"`
	}
	if e := decode(original, &p); e != nil || len(p.Default) == 0 {
		return nil, ErrRefused
	}
	if p.Transports == nil {
		p.Transports = map[string]map[string]json.RawMessage{}
	}
	if p.Transports["docker"] == nil {
		p.Transports["docker"] = map[string]json.RawMessage{}
	}
	repos := []string{t.Prefix + "-host", t.Prefix + "-release"}
	for _, n := range appliancerelease.Names {
		repos = append(repos, t.Prefix+"-"+n)
	}
	for _, c := range []string{"candidate", "preview", "stable"} {
		repos = append(repos, t.Prefix+"-channel-"+c)
	}
	for _, repo := range repos {
		for existing := range p.Transports["docker"] {
			if existing == repo || strings.HasPrefix(existing, repo+":") || strings.HasPrefix(existing, repo+"@") || strings.HasPrefix(existing, repo+"/") {
				return nil, errors.New("existing Soda trust override requires explicit review")
			}
		}
		req, _ := t.requirement(repo)
		raw, _ := json.Marshal([]requirement{req})
		p.Transports["docker"][repo] = raw
	}
	return marshal(p)
}

func WriteRegistryConfig(out string, t Trust) error {
	if e := t.Validate(); e != nil {
		return e
	}
	_, e := registryConfig(out, t)
	return e
}

func registryConfig(out string, t Trust) (string, error) {
	dir := filepath.Join(out, "registries.d")
	if e := os.Mkdir(dir, 0o700); e != nil {
		return "", e
	}
	// JSON is valid YAML; quote exact repository scopes rather than registry-wide
	// attachment/trust changes. No global host configuration is read or written.
	repos := map[string]any{}
	for _, n := range append([]string{"host", "release", "channel-candidate", "channel-preview", "channel-stable"}, appliancerelease.Names...) {
		repos[t.Prefix+"-"+n] = map[string]bool{"use-sigstore-attachments": true}
	}
	if e := writeJSON(filepath.Join(dir, "soda.yaml"), map[string]any{"docker": repos}); e != nil {
		return "", e
	}
	return dir, nil
}

// VerifyCopy is deliberately a fresh native copy, never a Podman-cache existence
// assertion or an inspect-only request (neither proves signature enforcement).
func VerifyCopy(ctx context.Context, r Runner, t Trust, ref, source, out string) error {
	repo, _, e := t.Reference(ref)
	if e != nil {
		return e
	}
	_, digest, _ := strings.Cut(ref, "@")
	if e = nativebuild.FreshDirectory(out); e != nil {
		return e
	}
	transport, scope, ok := strings.Cut(source, ":")
	if !ok {
		return ErrRefused
	}
	switch transport {
	case "docker":
		if source != "docker://"+ref {
			return ErrRefused
		}
		scope = repo
	case "dir":
		if !filepath.IsAbs(scope) {
			return ErrRefused
		}
	default:
		return ErrRefused
	}
	p, e := policyFor(t, repo, transport, scope)
	if e != nil {
		return e
	}
	pp := filepath.Join(out, "policy.json")
	if e = writeJSON(pp, p); e != nil {
		return e
	}
	registry, e := registryConfig(out, t)
	if e != nil {
		return e
	}
	if _, e = r.Run(ctx, "--command-timeout=10m", "--policy", pp, "--registries.d", registry, "copy", "--preserve-digests", "--src-no-creds", source, "dir:"+filepath.Join(out, "image")); e != nil {
		return e
	}
	b, e := ReadFile(filepath.Join(out, "image/manifest.json"), 1<<20)
	if e != nil || Hash(b) != digest {
		return ErrRefused
	}
	return nil
}

type SecretFiles struct{ Key, Passphrase string }

// Sign admits a protected exact permit and signs a private snapshot only after
// checking its manifest. No candidate script runs with signing credentials.
func admitSignInputs(t Trust, p Permit, input string, key SecretFiles) error {
	if t.Validate() != nil || p.Validate(t, nowUTC()) != nil || PrivateFile(key.Key) != nil || PrivateFile(key.Passphrase) != nil || !filepath.IsAbs(input) {
		return ErrRefused
	}
	return nil
}

func snapshotSignSource(ctx context.Context, r Runner, transport, input, out string) (string, error) {
	if transport != "oci" && transport != "oci-archive" && transport != "dir" {
		return "", ErrRefused
	}
	if e := nativebuild.FreshDirectory(out); e != nil {
		return "", e
	}
	policy := filepath.Join(out, "snapshot-policy.json")
	if e := writeJSON(policy, localPolicy(transport, input)); e != nil {
		return "", e
	}
	snapshot := filepath.Join(out, "snapshot")
	if _, e := r.Run(ctx, "--command-timeout=10m", "--policy", policy, "copy", "--preserve-digests", "--remove-signatures", transport+":"+input, "dir:"+snapshot); e != nil {
		return "", e
	}
	return snapshot, nil
}

func admitSignedPayload(t Trust, p Permit, snapshot string) error {
	mb, e := ReadFile(filepath.Join(snapshot, "manifest.json"), 1<<20)
	if e != nil || Hash(mb) != p.Digest {
		return ErrRefused
	}
	role, _ := t.Role(p.Repository)
	if channel(role) {
		var c Channel
		if e = ReadDocument(snapshot, p.Digest, &c); e != nil {
			return e
		}
		if _, e = AdmitChannel(t, EmptyState(), c, p.Digest, role, nowUTC()); e != nil {
			return e
		}
	}
	if p.Repository == t.Prefix+"-release" {
		var release Release
		if e = ReadDocument(snapshot, p.Digest, &release); e != nil {
			return e
		}
		if _, _, e = release.Validate(t); e != nil {
			return e
		}
	}
	return nil
}

func emitSignedDirectory(ctx context.Context, r Runner, t Trust, p Permit, snapshot, out string, key SecretFiles) error {
	policy := filepath.Join(out, "sign-policy.json")
	if e := writeJSON(policy, localPolicy("dir", snapshot)); e != nil {
		return e
	}
	signed := filepath.Join(out, "signed")
	ref := p.Repository + "@" + p.Digest
	if _, e := r.Run(ctx, "--command-timeout=10m", "--policy", policy, "copy", "--preserve-digests", "--sign-by-sigstore-private-key", key.Key, "--sign-passphrase-file", key.Passphrase, "--sign-identity", ref, "dir:"+snapshot, "dir:"+signed); e != nil {
		return e
	}
	if e := VerifyCopy(ctx, r, t, ref, "dir:"+signed, filepath.Join(out, "check")); e != nil {
		return e
	}
	return writeJSON(filepath.Join(out, "receipt.json"), map[string]string{"Reference": ref, "Scope": "native-signed local directory; not published or boot-qualified"})
}

func Sign(ctx context.Context, r Runner, t Trust, p Permit, transport, input, out string, key SecretFiles) error {
	if e := admitSignInputs(t, p, input, key); e != nil {
		return e
	}
	snapshot, e := snapshotSignSource(ctx, r, transport, input, out)
	if e != nil {
		return e
	}
	if e = admitSignedPayload(t, p, snapshot); e != nil {
		return e
	}
	return emitSignedDirectory(ctx, r, t, p, snapshot, out, key)
}
