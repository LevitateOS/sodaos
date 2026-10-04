package installer

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestZZOracleNetip(t *testing.T) {
	addrs := []string{"192.168.1.01", "192.168.1.1", "0.0.0.0", "255.255.255.255", "1.2.3.4.5",
		"0x7f.0.0.1", "0177.0.0.1", "1.2.3", "::", "::1", "::ffff:1.2.3.4", "::FFFF:1.2.3.4",
		"1::2::3", "fe80::1%eth0", "1.2.3.4%eth0", "%eth0", "", "FD00::123", "fd00::123",
		"100.90.1.2", "169.254.1.2", "10.0.0.256", "1.2.3.4.", ".1.2.3.4", "1.2.3.4 ",
		" 1.2.3.4", "0:0:0:0:0:ffff:1.2.3.4", "64:ff9b::1.2.3.4", "2001:db8::1",
		"fe80::1", "ff02::1", "100::1", "010.0.0.1", "1.02.3.4"}
	for _, s := range addrs {
		a, err := netip.ParseAddr(s)
		if err != nil {
			fmt.Printf("ADDR %q ERR\n", s)
			continue
		}
		fmt.Printf("ADDR %q ok is4=%t is6=%t is4in6=%t priv=%t str=%q\n", s, a.Is4(), a.Is6(), a.Is4In6(), a.IsPrivate(), a.String())
	}
	prefixes := []string{"10.89.0.0/24", "10.89.0.1/24", "0.0.0.0/0", "10.0.0.0/33", "10.0.0.0/-1",
		"10.0.0.0/", "10.0.0.0", "fd00::/64", "fd00::1/64", "010.0.0.0/8", "10.0.0.0/08",
		"10.0.0.0/8 ", " 10.0.0.0/8", "10.0.0.0/0", "::/0", "1.2.3.4/32", "1.2.3.4/31",
		"10.0.0.0/+8", "10.0.0.0/8/8", "fe80::1%eth0/64"}
	for _, s := range prefixes {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			fmt.Printf("PREFIX %q ERR\n", s)
			continue
		}
		fmt.Printf("PREFIX %q ok masked=%t str=%q addr=%q bits=%d\n", s, p == p.Masked(), p.String(), p.Addr().String(), p.Bits())
	}
}

func TestZZOracleSSH(t *testing.T) {
	edkey, _, _ := ed25519.GenerateKey(rand.Reader)
	edpub, _ := ssh.NewPublicKey(edkey)
	ed := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(edpub)))
	vectors := []string{ed, ed + " comment here", ed + " comment=with=equals", "  " + ed + "  ",
		"\t" + ed, ed + "\tcomment", "command=\"x\" " + ed, "no-pty " + ed, "AAA", "ssh-ed25519",
		"ssh-ed25519 AAAA", "ssh-ed25519 " + base64.StdEncoding.EncodeToString([]byte("short")),
		"ssh-ed25519-cert-v01@openssh.com AAAA", "sk-ssh-ed25519@openssh.com AAAA", "", "   ",
		"# just a comment", ed + "\n" + ed, "ssh-ed25519 " + strings.Repeat("A", 100) + " trailing"}
	for _, v := range vectors {
		key, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(v))
		typ := ""
		marshal := ""
		if err == nil {
			typ = key.Type()
			marshal = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
		}
		fields := strings.Fields(v)
		want := ""
		if len(fields) >= 2 {
			want = fields[0] + " " + fields[1]
		}
		fmt.Printf("SSH %q err=%v type=%q comment=%q options=%q rest=%q marshal_eq=%t\n", v, err != nil, typ, comment, options, rest, marshal == want)
	}
	// RSA non-minimal mpint: pad e and n with leading zeros, see if marshal canonicalizes.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("rsa gen: %v", err)
	}
	rsaPub := rsaKey.Public().(*rsa.PublicKey)
	wire := []byte{}
	putStr := func(s string) { var l [4]byte; binary.BigEndian.PutUint32(l[:], uint32(len(s))); wire = append(wire, l[:]...); wire = append(wire, s...) }
	putMP := func(b []byte) { var l [4]byte; binary.BigEndian.PutUint32(l[:], uint32(len(b))); wire = append(wire, l[:]...); wire = append(wire, b...) }
	putStr("ssh-rsa")
	e := big.NewInt(int64(rsaPub.E)).Bytes()
	putMP(append([]byte{0, 0, 0}, e...))
	putMP(append([]byte{0, 0}, rsaPub.N.Bytes()...))
	padded := "ssh-rsa " + base64.StdEncoding.EncodeToString(wire)
	k2, _, _, _, err2 := ssh.ParseAuthorizedKey([]byte(padded))
	fmt.Printf("RSA-PADDED err=%v\n", err2)
	if err2 == nil {
		m := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k2)))
		fmt.Printf("RSA-PADDED marshal_same=%t\n", m == padded)
		clean, _ := ssh.NewPublicKey(rsaPub)
		fmt.Printf("RSA-PADDED marshal_clean=%t\n", m == strings.TrimSpace(string(ssh.MarshalAuthorizedKey(clean))))
	}
	// SK key: hand-built wire.
	skwire := []byte{}
	putStr2 := func(s []byte) { var l [4]byte; binary.BigEndian.PutUint32(l[:], uint32(len(s))); skwire = append(skwire, l[:]...); skwire = append(skwire, s...) }
	putStr2([]byte("sk-ssh-ed25519@openssh.com"))
	putStr2(make([]byte, 32))
	putStr2([]byte("ssh:"))
	skwire = append(skwire, 0x01)
	putStr2([]byte{})
	putStr2([]byte{})
	sk := "sk-ssh-ed25519@openssh.com " + base64.StdEncoding.EncodeToString(skwire)
	k3, _, _, _, skErr := ssh.ParseAuthorizedKey([]byte(sk))
	skType := ""
	if skErr == nil {
		skType = k3.Type()
	}
	fmt.Printf("SK err=%v type=%v\n", skErr, skType)
	// ecdsa key marshal
	ecKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ecPub, _ := ssh.NewPublicKey(&ecKey.PublicKey)
	ecline := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(ecPub)))
	k4, _, _, _, err := ssh.ParseAuthorizedKey([]byte(ecline))
	fmt.Printf("ECDSA err=%v type=%v fp=%v\n", err, func() string { if err2 == nil { return k4.Type() }; return "" }(), func() string { if err == nil { return ssh.FingerprintSHA256(k4) }; return "" }())
	fmt.Printf("ED fp=%s\n", ssh.FingerprintSHA256(edpub))
	fmt.Printf("ED fp_bytes=%x\n", sha256.Sum256(edpub.Marshal()))
}

func TestZZOracleURL(t *testing.T) {
	urls := []string{"https://192.168.1.5", "https://192.168.1.5/", "HTTPS://192.168.1.5",
		"https://192.168.1.2:444", "https://192.168.1.2/path", "https://root@192.168.1.2",
		"https://192.168.1.2?argument", "https://192.168.1.2#frag", "https://[fd00::123]",
		"https://[FD00::123]/", "https://soda.example.test", "http://192.168.1.5",
		"https://192.168.1.5:443", "https://user:pass@192.168.1.5", "https://192.168.1.5//",
		" https://192.168.1.5", "https://192.168.1.5 ", "https://", "https://?x", "//192.168.1.5",
		"https:///path", "https://192.168.1.5:99999", "https://[::1]:8080/x?y#z"}
	for _, s := range urls {
		u, err := url.Parse(s)
		if err != nil {
			fmt.Printf("URL %q ERR %v\n", s, err)
			continue
		}
		fmt.Printf("URL %q scheme=%q user=%v host=%q hostname=%q port=%q path=%q rawq=%q frag=%q str=%q\n",
			s, u.Scheme, u.User != nil, u.Host, u.Hostname(), u.Port(), u.Path, u.RawQuery, u.Fragment, u.String())
	}
	// privateSetupOrigin equivalent
	for _, a := range []string{"192.168.2.100", "100.90.1.2", "fd00::123", "FD00::123", "::ffff:192.168.1.2"} {
		o, err := privateSetupOrigin(a)
		fmt.Printf("ORIGIN %q -> %q err=%v\n", a, o, err != nil)
	}
}

func TestZZOraclePEM(t *testing.T) {
	der := []byte{0x30, 0x03, 0x02, 0x01, 0x05}
	mk := func(typ string, b []byte) []byte { return pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: b}) }
	vectors := map[string][]byte{
		"clean":      mk("CERTIFICATE", der),
		"leading":    append([]byte("garbage\nline2\n"), mk("CERTIFICATE", der)...),
		"two":        append(mk("CERTIFICATE", der), mk("CERTIFICATE", der)...),
		"trailingws": append(mk("CERTIFICATE", der), []byte("  \n\t")...),
		"private":    mk("PRIVATE KEY", der),
		"lower":      []byte("-----begin certificate-----\n" + base64.StdEncoding.EncodeToString(der) + "\n-----end certificate-----\n"),
		"noend":      []byte("-----BEGIN CERTIFICATE-----\n" + base64.StdEncoding.EncodeToString(der) + "\n"),
		"empty":      []byte(""),
		"crlf":       []byte("-----BEGIN CERTIFICATE-----\r\n" + base64.StdEncoding.EncodeToString(der) + "\r\n-----END CERTIFICATE-----\r\n"),
	}
	for name, v := range vectors {
		block, rest := pem.Decode(v)
		if block == nil {
			fmt.Printf("PEM %s NIL rest=%q\n", name, string(rest))
			continue
		}
		fmt.Printf("PEM %s type=%q bytes_eq=%t rest=%q\n", name, block.Type, string(block.Bytes) == string(der), string(rest))
	}
}

func TestZZOracleFmt(t *testing.T) {
	strs := []string{"eth0", "a\"b\\c", "line\nfeed\ttab", "\x01\x02\x7f", "héllo", "日本語",
		"\u200bzero-width", "a\xffb", "\xff\xfe", "trailing space ", "UPPER", "sod\x00a",
		"\x00\x1f\x20\x7e", "𝄞musical", "e\xcc\x81combining"}
	for _, s := range strs {
		fmt.Printf("Q %q -> %%q=%q\n", s, fmt.Sprintf("%q", s))
	}
	for _, size := range []uint64{0, 1, 1 << 30, (1 << 30) + (1 << 29), 64 << 30, 1000000000, 1503238553, 42 << 20, 1536 << 20, 1073741824, 1610612736, 123456789, 999999999999, 1<<64 - 1, 5905580032, 5905580033, 3221225472, 3758096384} {
		fmt.Printf("F1 gib(%d)=%s mib(%d)=%s\n", size, fmt.Sprintf("%.1f", float64(size)/(1<<30)), size, fmt.Sprintf("%.1f", float64(size)/(1<<20)))
	}
	// sweep for .05-boundary divergences
	mismatch := 0
	for i := uint64(0); i < 2000000; i++ {
		_ = i
	}
	_ = mismatch
	fmt.Printf("TRIM %q\n", strings.TrimSpace("  \t\n\v\f\r \u0085\u00a0x\u0085\u00a0  "))
	fmt.Printf("FIELDS %q\n", strings.Fields("a\tb\nc\vd\fe\rf\u0085g\u00a0h  i"))
	fmt.Printf("LOWER %q %q %q\n", strings.ToLower("BACK"), strings.ToLower("İ"), strings.ToLower("Σς"))
	fmt.Printf("FOLD usb=%t back=%t k=%t sigma=%t\n", strings.EqualFold("USB", "usb"), strings.EqualFold("Back", "back"), strings.EqualFold("k", "K"), strings.EqualFold("ς", "Σ"))
	for _, s := range []string{"+1", "007", " 1", "1 ", "0x1", "99999999999999999999999", "-0", "", "+", "-", "１２３"} {
		n, err := strconv.Atoi(s)
		fmt.Printf("ATOI %q -> %d err=%v\n", s, n, err != nil)
	}
}

func TestZZOracleX509Algs(t *testing.T) {
	mk := func(name string, tmpl, parent *x509.Certificate, pub, priv any) {
		der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, priv)
		if err != nil {
			fmt.Printf("X509 %s CREATE-FAIL %v\n", name, err)
			return
		}
		// Save one fixture for the Rust port tests.
		if name == "rsa-ca" || name == "ec-ca" {
			os.WriteFile("/tmp/oracle-"+name+".pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644)
		}
		fp, err := localCAFingerprint(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
		fmt.Printf("X509 %s err=%v fplen=%d\n", name, err != nil, len(fp))
	}
	base := func(alg x509.SignatureAlgorithm, ca bool) *x509.Certificate {
		return &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "x"}, IsCA: ca, BasicConstraintsValid: true, SignatureAlgorithm: alg}
	}
	rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	for _, alg := range []x509.SignatureAlgorithm{x509.SHA256WithRSA, x509.SHA384WithRSA, x509.SHA512WithRSA, x509.SHA1WithRSA, x509.MD5WithRSA, x509.SHA256WithRSAPSS, x509.ECDSAWithSHA256} {
		tmpl := base(alg, true)
		mk(fmt.Sprintf("rsa-%d", int(alg)), tmpl, tmpl, &rsaKey.PublicKey, rsaKey)
	}
	for _, curve := range []elliptic.Curve{elliptic.P224(), elliptic.P256(), elliptic.P384(), elliptic.P521()} {
		key, _ := ecdsa.GenerateKey(curve, rand.Reader)
		for _, alg := range []x509.SignatureAlgorithm{x509.ECDSAWithSHA1, x509.ECDSAWithSHA256, x509.ECDSAWithSHA384, x509.ECDSAWithSHA512} {
			tmpl := base(alg, true)
			mk(fmt.Sprintf("ec-%d-%d", curve.Params().BitSize, int(alg)), tmpl, tmpl, &key.PublicKey, key)
		}
	}
	{
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		tmpl := base(x509.PureEd25519, true)
		mk("ed25519-ca", tmpl, tmpl, pub, priv)
		tmpl2 := base(x509.PureEd25519, false)
		mk("ed25519-nonca", tmpl2, tmpl2, pub, priv)
	}
	{
		tmpl := base(x509.SHA256WithRSA, true)
		mk("rsa-ca", tmpl, tmpl, &rsaKey.PublicKey, rsaKey)
		tmpl2 := base(x509.SHA256WithRSA, false)
		mk("rsa-nonca", tmpl2, tmpl2, &rsaKey.PublicKey, rsaKey)
		ecKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		tmpl3 := base(x509.ECDSAWithSHA256, true)
		mk("ec-ca", tmpl3, tmpl3, &ecKey.PublicKey, ecKey)
		// v1: no BasicConstraintsValid, no extensions
		v1 := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "x"}, SignatureAlgorithm: x509.SHA256WithRSA}
		mk("rsa-v1", v1, v1, &rsaKey.PublicKey, rsaKey)
		// tampered signature
		der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &rsaKey.PublicKey, rsaKey)
		der[len(der)-1] ^= 1
		_, err := localCAFingerprint(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
		fmt.Printf("X509 rsa-tampered err=%v\n", err != nil)
		der2, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &rsaKey.PublicKey, rsaKey)
		der2[10] ^= 0x40
		_, err = localCAFingerprint(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der2}))
		fmt.Printf("X509 rsa-tampered-tbs err=%v\n", err != nil)
		// critical unknown extension
		tmplX := base(x509.SHA256WithRSA, true)
		tmplX.ExtraExtensions = []pkix.Extension{{Id: []int{1, 2, 3, 4, 5}, Critical: true, Value: []byte{5, 0}}}
		mk("rsa-critical-unknown", tmplX, tmplX, &rsaKey.PublicKey, rsaKey)
		tmplY := base(x509.SHA256WithRSA, true)
		tmplY.ExtraExtensions = []pkix.Extension{{Id: []int{1, 2, 3, 4, 5}, Critical: false, Value: []byte{5, 0}}}
		mk("rsa-noncritical-unknown", tmplY, tmplY, &rsaKey.PublicKey, rsaKey)
		// cross-signed: same key, different issuer name (not self)
		issuer := base(x509.SHA256WithRSA, true)
		issuer.Subject = pkix.Name{CommonName: "other"}
		leaf := base(x509.SHA256WithRSA, true)
		leaf.Subject = pkix.Name{CommonName: "x"}
		mk("rsa-cross", leaf, issuer, &rsaKey.PublicKey, rsaKey)
	}
	// Go-generated ECDSA cert: inspect alg OID + key type path
	{
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "x"}, IsCA: true, BasicConstraintsValid: true}
		der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
		cert, _ := x509.ParseCertificate(der)
		fmt.Printf("X509 go-ec alg=%v sigalg=%v\n", cert.PublicKeyAlgorithm, cert.SignatureAlgorithm)
		_ = crypto.SHA256
	}
}
