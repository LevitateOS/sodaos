package qualify

import (
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
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/image"
)

type fixture struct {
	mu                                   sync.Mutex
	blob                                 string
	hits, graphRequests, payloadRequests int
	blocked, offer                       bool
	registry, graph, rootfs              *http.Server
	cid, work                            string
	e                                    *acceptance.Evidence
	seq                                  int
	a, b                                 Artifact
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

// One demonstrated content fault; the upstream registry still owns OCI serving.
func (f *fixture) registryHandler(upstream http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		if strings.Contains(r.URL.Path, "/blobs/") {
			f.payloadRequests++
		}
		blocked := f.blocked && f.blob != "" && strings.HasSuffix(r.URL.Path, "/blobs/"+f.blob)
		if blocked {
			f.hits++
		}
		f.mu.Unlock()
		if blocked {
			http.Error(w, "qualification fixture: required content unavailable", http.StatusServiceUnavailable)
			return
		}
		upstream.ServeHTTP(w, r)
	})
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
	if err = build.WriteNew(filepath.Join(dir, "ca.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0o600); err != nil {
		return err
	}
	return build.WriteNew(filepath.Join(dir, "tls.key"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: priv}), 0o600)
}

func (f *fixture) serve(port int, handler http.Handler, tls bool) (*http.Server, error) {
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}
	s := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if tls {
			_ = s.ServeTLS(listener, filepath.Join(f.work, "ca.crt"), filepath.Join(f.work, "tls.key"))
		} else {
			_ = s.Serve(listener)
		}
	}()
	return s, nil
}

func localRootfsURL(raw string) (*url.URL, error) {
	rootfsURL, err := url.Parse(raw)
	if err != nil || rootfsURL.Host != "10.0.2.2:19948" || rootfsURL.Scheme != "http" {
		return nil, errors.New("selected local rootfs route required")
	}
	return rootfsURL, nil
}

func graphNode(a Artifact, age string) any {
	return map[string]any{"version": a.Payload.ID, "payload": a.Candidate.HostReference, "metadata": map[string]string{"org.fedoraproject.coreos.scheme": "oci", "org.fedoraproject.coreos.releases.age_index": age}}
}

func (f *fixture) serveGraph(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/graph" {
		http.NotFound(w, r)
		return
	}
	f.mu.Lock()
	offer := f.offer
	f.graphRequests++
	request := f.graphRequests
	f.mu.Unlock()
	edges := [][2]int{}
	if offer {
		edges = append(edges, [2]int{0, 1})
	}
	graph := map[string]any{"nodes": []any{graphNode(f.a, "1"), graphNode(f.b, "2")}, "edges": edges}
	if err := f.e.WriteJSON(fmt.Sprintf("graph-%03d.json", request), map[string]any{"request": r.URL.RequestURI(), "response": graph}); err != nil {
		http.Error(w, "evidence unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(graph)
}

func (f *fixture) rootfsHandler(path, file string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, file)
	})
}

func prepareFixtureWork(work string) error {
	for _, dir := range []string{"runtime", ".config/containers"} {
		if err := os.MkdirAll(filepath.Join(work, dir), 0o700); err != nil {
			return err
		}
	}
	return build.WriteNew(filepath.Join(work, ".config/containers/containers.conf"), []byte("[engine]\ncgroup_manager = \"cgroupfs\"\nevents_logger = \"file\"\n"), 0o600)
}

func (f *fixture) startRegistry(ctx context.Context, c Config) error {
	if !pinnedRegistry(c.RegistryImage) {
		return errors.New("pinned upstream registry required")
	}
	if _, err := f.run(ctx, "registry-pull", "podman", "--remote=false", "pull", c.RegistryImage); err != nil {
		return err
	}
	cid, err := f.run(ctx, "registry-start", "podman", "--remote=false", "run", "--detach", "--pull=never", "--name", "soda-p9-registry-"+filepath.Base(c.Work), "--network=host", "--env", "REGISTRY_HTTP_ADDR=127.0.0.1:19500", c.RegistryImage)
	if err != nil {
		return err
	}
	f.cid = strings.TrimSpace(string(cid))
	endpoint, _ := url.Parse("http://127.0.0.1:19500")
	proxy := httputil.NewSingleHostReverseProxy(endpoint)
	f.registry, err = f.serve(19443, f.registryHandler(proxy), true)
	return err
}

func (f *fixture) startServers(rootfs *url.URL, media image.Media) error {
	var err error
	f.graph, err = f.serve(19444, http.HandlerFunc(f.serveGraph), true)
	if err != nil {
		return err
	}
	f.rootfs, err = f.serve(19948, f.rootfsHandler(rootfs.Path, filepath.Join("/run/soda-p9-input/candidate/media", media.Rootfs.Path)), false)
	return err
}

func (f *fixture) startLocalServices(ctx context.Context, c Config, media image.Media) error {
	if err := f.startRegistry(ctx, c); err != nil {
		return err
	}
	rootfsURL, err := localRootfsURL(media.RootfsURL)
	if err != nil {
		return err
	}
	return f.startServers(rootfsURL, media)
}

func writeFixturePassphrase(dir string) error {
	pass := make([]byte, 32)
	if _, err := rand.Read(pass); err != nil {
		return err
	}
	return build.WriteNew(filepath.Join(dir, "passphrase"), []byte(fmt.Sprintf("%x", pass)), 0o600)
}

func (f *fixture) writeTrust(ctx context.Context, c Config) error {
	for _, name := range []string{"correct", "wrong"} {
		if _, err := f.run(ctx, "key-"+name, "skopeo", "generate-sigstore-key", "--output-prefix", filepath.Join(f.work, name), "--passphrase-file", filepath.Join(f.work, "passphrase")); err != nil {
			return err
		}
	}
	public, err := os.ReadFile(filepath.Join(f.work, "correct.pub"))
	if err != nil {
		return err
	}
	ca, err := os.ReadFile(filepath.Join(f.work, "ca.crt"))
	if err != nil {
		return err
	}
	return f.e.WriteJSON("fixture-trust.json", map[string]any{"repository": f.b.Candidate.HostReference, "registry_image": c.RegistryImage, "public_key": string(public), "ca": string(ca), "graph_url": "https://10.0.2.2:19444/v1/graph"})
}

func (f *fixture) writePublishPolicy() error {
	// The publishing client uses the same native attachment mechanism as the guest.
	cfg := "docker:\n  127.0.0.1:19500:\n    use-sigstore-attachments: true\n"
	if err := os.Mkdir(filepath.Join(f.work, "registries.d"), 0o700); err != nil {
		return err
	}
	if err := build.WriteNew(filepath.Join(f.work, "registries.d/fixture.yaml"), []byte(cfg), 0o600); err != nil {
		return err
	}
	policy := map[string]any{"default": []any{map[string]string{"type": "reject"}}, "transports": map[string]any{"oci-archive": map[string]any{"/run/soda-p9-input/candidate/host.oci": []any{map[string]string{"type": "insecureAcceptAnything"}}}}}
	return writeNewJSON(filepath.Join(f.work, "source-policy.json"), policy)
}

func newFixture(ctx context.Context, c Config, a, b Artifact, media image.Media, e *acceptance.Evidence) (f *fixture, err error) {
	f = &fixture{work: filepath.Join(c.Work, "fixture"), e: e, a: a, b: b}
	if err = os.Mkdir(f.work, 0o700); err != nil {
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
	if err = prepareFixtureWork(c.Work); err != nil {
		return f, err
	}
	if err = f.startLocalServices(ctx, c, media); err != nil {
		return f, err
	}
	if err = writeFixturePassphrase(f.work); err != nil {
		return f, err
	}
	if err = f.writeTrust(ctx, c); err != nil {
		return f, err
	}
	return f, f.writePublishPolicy()
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
