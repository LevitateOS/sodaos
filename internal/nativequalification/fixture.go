package nativequalification

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

type fixture struct {
	mu                      sync.Mutex
	mode, blob              string
	hits, graphRequests     int
	offer                   bool
	registry, graph, rootfs *http.Server
	cid, work               string
	e                       *acceptance.Evidence
	seq                     int
	a, b                    Artifact
}

func (f *fixture) run(ctx context.Context, label, name string, args ...string) ([]byte, error) {
	f.seq++
	r, e := acceptance.Execute(ctx, f.e, fmt.Sprintf("fixture-%03d-%s", f.seq, label), acceptance.Command{Name: name, Args: args})
	return r.Stdout, errors.Join(e, r.Err)
}
func (f *fixture) close() error {
	var err error
	for _, s := range []*http.Server{f.registry, f.graph, f.rootfs} {
		if s != nil {
			err = errors.Join(err, s.Close())
		}
	}
	if f.cid != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, e := f.run(ctx, "stop-registry", "podman", "--remote=false", "stop", "--time=10", f.cid)
		err = errors.Join(err, e)
	}
	return err
}
func tlsFixture(dir string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "soda-p9-local"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("10.0.2.2"), net.ParseIP("127.0.0.1")}}
	cert, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}
	priv, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	if err = nativebuild.WriteNew(filepath.Join(dir, "ca.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0600); err != nil {
		return err
	}
	return nativebuild.WriteNew(filepath.Join(dir, "tls.key"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: priv}), 0600)
}
func (f *fixture) serve(port int, handler http.Handler, tls bool) (*http.Server, error) {
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}
	s := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if tls {
			s.ServeTLS(listener, filepath.Join(f.work, "ca.crt"), filepath.Join(f.work, "tls.key"))
		} else {
			s.Serve(listener)
		}
	}()
	return s, nil
}
func newFixture(ctx context.Context, c Config, a, b Artifact, media hostimage.Media, e *acceptance.Evidence) (f *fixture, err error) {
	f = &fixture{work: filepath.Join(c.Work, "fixture"), e: e, a: a, b: b}
	if err = os.Mkdir(f.work, 0700); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, f.close())
		}
	}()
	if err = tlsFixture(f.work); err != nil {
		return f, err
	}
	for _, dir := range []string{"runtime", ".config/containers"} {
		if err = os.MkdirAll(filepath.Join(c.Work, dir), 0700); err != nil {
			return f, err
		}
	}
	if err = nativebuild.WriteNew(filepath.Join(c.Work, ".config/containers/containers.conf"), []byte("[engine]\ncgroup_manager = \"cgroupfs\"\nevents_logger = \"file\"\n"), 0600); err != nil {
		return f, err
	}
	if !strings.HasPrefix(c.RegistryImage, "docker.io/library/registry@sha256:") || !nativebuild.Digest(strings.TrimPrefix(c.RegistryImage, "docker.io/library/registry@sha256:")) {
		return f, errors.New("pinned upstream registry required")
	}
	if _, err = f.run(ctx, "registry-pull", "podman", "--remote=false", "pull", c.RegistryImage); err != nil {
		return f, err
	}
	cid, err := f.run(ctx, "registry-start", "podman", "--remote=false", "run", "--detach", "--pull=never", "--name", "soda-p9-registry-"+filepath.Base(c.Work), "--publish", "127.0.0.1:19500:5000", c.RegistryImage)
	if err != nil {
		return f, err
	}
	f.cid = strings.TrimSpace(string(cid))
	endpoint, _ := url.Parse("http://127.0.0.1:19500")
	proxy := httputil.NewSingleHostReverseProxy(endpoint)
	proxy.ModifyResponse = func(r *http.Response) error {
		f.mu.Lock()
		mode, match := f.mode, f.blob != "" && strings.HasSuffix(r.Request.URL.Path, "/blobs/"+f.blob)
		if match && mode != "" {
			f.hits++
		}
		f.mu.Unlock()
		if !match || mode == "" || r.StatusCode != 200 {
			return nil
		}
		switch mode {
		case "missing":
			r.Body.Close()
			r.StatusCode = 404
			r.Body = io.NopCloser(strings.NewReader("missing fixture content"))
			r.ContentLength = int64(len("missing fixture content"))
			r.Header.Set("Content-Length", fmt.Sprint(r.ContentLength))
		case "interrupt":
			r.Body = struct {
				io.Reader
				io.Closer
			}{io.LimitReader(r.Body, 65536), r.Body}
		case "tamper":
			var one [1]byte
			if _, err := io.ReadFull(r.Body, one[:]); err != nil {
				return err
			}
			one[0] ^= 1
			r.Body = struct {
				io.Reader
				io.Closer
			}{io.MultiReader(bytes.NewReader(one[:]), r.Body), r.Body}
		}
		return nil
	}
	f.registry, err = f.serve(19443, proxy, true)
	if err != nil {
		return f, err
	}
	f.graph, err = f.serve(19444, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/graph" {
			http.NotFound(w, r)
			return
		}
		f.mu.Lock()
		offer := f.offer
		f.graphRequests++
		request := f.graphRequests
		f.mu.Unlock()
		node := func(a Artifact, age string) any {
			return map[string]any{"version": a.Payload.ID, "payload": a.Candidate.HostReference, "metadata": map[string]string{"org.fedoraproject.coreos.scheme": "oci", "org.fedoraproject.coreos.releases.age_index": age}}
		}
		edges := [][2]int{}
		if offer {
			edges = append(edges, [2]int{0, 1})
		}
		graph := map[string]any{"nodes": []any{node(a, "1"), node(b, "2")}, "edges": edges}
		if err := e.WriteJSON(fmt.Sprintf("graph-%03d.json", request), map[string]any{"request": r.URL.RequestURI(), "response": graph}); err != nil {
			http.Error(w, "evidence unavailable", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(graph)
	}), true)
	if err != nil {
		return f, err
	}
	rootfsURL, err := url.Parse(media.RootfsURL)
	if err != nil || rootfsURL.Host != "10.0.2.2:19948" || rootfsURL.Scheme != "http" {
		return f, errors.New("selected local rootfs route required")
	}
	f.rootfs, err = f.serve(19948, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != rootfsURL.Path {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join("/run/soda-p9-input/candidate/media", media.Rootfs.Path))
	}), false)
	if err != nil {
		return f, err
	}
	pass := make([]byte, 32)
	if _, err = rand.Read(pass); err != nil {
		return f, err
	}
	if err = nativebuild.WriteNew(filepath.Join(f.work, "passphrase"), []byte(fmt.Sprintf("%x", pass)), 0600); err != nil {
		return f, err
	}
	for _, name := range []string{"correct", "wrong"} {
		if _, err = f.run(ctx, "key-"+name, "skopeo", "generate-sigstore-key", "--output-prefix", filepath.Join(f.work, name), "--passphrase-file", filepath.Join(f.work, "passphrase")); err != nil {
			return f, err
		}
	}
	public, err := os.ReadFile(filepath.Join(f.work, "correct.pub"))
	if err != nil {
		return f, err
	}
	ca, err := os.ReadFile(filepath.Join(f.work, "ca.crt"))
	if err != nil {
		return f, err
	}
	if err = e.WriteJSON("fixture-trust.json", map[string]any{"repository": b.Candidate.HostReference, "registry_image": c.RegistryImage, "public_key": string(public), "ca": string(ca), "graph_url": "https://10.0.2.2:19444/v1/graph"}); err != nil {
		return f, err
	}
	// The publishing client uses the same native attachment mechanism as the guest.
	cfg := "docker:\n  127.0.0.1:19500:\n    use-sigstore-attachments: true\n"
	if err = os.Mkdir(filepath.Join(f.work, "registries.d"), 0700); err != nil {
		return f, err
	}
	if err = nativebuild.WriteNew(filepath.Join(f.work, "registries.d/fixture.yaml"), []byte(cfg), 0600); err != nil {
		return f, err
	}
	policy := map[string]any{"default": []any{map[string]string{"type": "reject"}}, "transports": map[string]any{"oci-archive": map[string]any{"/run/soda-p9-input/candidate/host.oci": []any{map[string]string{"type": "insecureAcceptAnything"}}}}}
	if err = writeNewJSON(filepath.Join(f.work, "source-policy.json"), policy); err != nil {
		return f, err
	}
	return f, nil
}
func (f *fixture) push(ctx context.Context, key string) error {
	_, err := f.run(ctx, "sign-push-"+key, "skopeo", "--registries.d", filepath.Join(f.work, "registries.d"), "--policy", filepath.Join(f.work, "source-policy.json"), "copy", "--preserve-digests", "--dest-tls-verify=false", "--sign-by-sigstore-private-key", filepath.Join(f.work, key+".private"), "--sign-passphrase-file", filepath.Join(f.work, "passphrase"), "--sign-identity", f.b.Candidate.HostReference, "oci-archive:/run/soda-p9-input/candidate/host.oci", "docker://127.0.0.1:19500/levitateos/sodaos-host:candidate")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:19500/v2/levitateos/sodaos-host/manifests/"+f.b.Candidate.Host.Manifest, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errors.New("registry candidate manifest unavailable")
	}
	var manifest struct {
		Layers []struct {
			Digest string
			Size   int64
		}
	}
	if err = json.NewDecoder(res.Body).Decode(&manifest); err != nil {
		return err
	}
	var size int64
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range manifest.Layers {
		if l.Size > size {
			size = l.Size
			f.blob = l.Digest
		}
	}
	if f.blob == "" {
		return errors.New("candidate has no testable layer")
	}
	return nil
}
