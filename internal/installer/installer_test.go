package installer

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func fixtureKey(t *testing.T) string {
	t.Helper()
	key, err := ssh.NewPublicKey(ed25519.PublicKey(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
}

func TestInputValidation(t *testing.T) {
	for _, name := range []string{"soda", "soda.example.test", "appliance-12", strings.Repeat("a", 63)} {
		if !Hostname(name) {
			t.Errorf("valid hostname refused: %q", name)
		}
	}
	for _, name := range []string{"", "-soda", "soda-", "soda..test", "Soda", "soda\nreboot", "soda/test", strings.Repeat("a", 64), strings.Repeat("a.", 127) + "a"} {
		if Hostname(name) {
			t.Errorf("invalid hostname accepted: %q", name)
		}
	}
	key := fixtureKey(t)
	if got, err := PublicKey(key + " fixture-comment"); err != nil || got != key {
		t.Fatal("key normalization", err)
	}
	for _, bad := range []string{key + "\n" + key, "command=\"sudo reboot\" " + key, "ssh-ed25519 AAAA", "-----BEGIN OPENSSH PRIVATE KEY-----", key + "\x00"} {
		if _, err := PublicKey(bad); err == nil {
			t.Error("invalid key accepted")
		}
	}
	for _, value := range []string{"10.89.0.0/24", "172.16.0.0/16", "192.168.100.0/28"} {
		if err := ProjectSubnet(value, []string{"default", "192.0.2.0/24"}); err != nil {
			t.Error(err)
		}
	}
	for _, value := range []string{"0.0.0.0/0", "8.8.8.0/24", "172.0.0.0/8", "192.168.1.1/24", "fd00::/64"} {
		if ProjectSubnet(value, nil) == nil {
			t.Errorf("invalid project subnet accepted: %s", value)
		}
	}
	for _, route := range []string{"10.89.0.0/16", "10.89.0.4/32", "10.89.0.1", "unparseable"} {
		if ProjectSubnet("10.89.0.0/24", []string{route}) == nil {
			t.Errorf("overlapping/unknown route accepted: %s", route)
		}
	}
}

func TestDestinationUsesConvertedPublicBootstrap(t *testing.T) {
	template := []byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[{"path":"/etc/example","mode":420,"contents":{"source":"data:,public"}}]},"systemd":{"units":[{"name":"soda-extensions.service","enabled":true,"contents":"public-unit"}]}}`)
	hash := "$6$synthetic$" + strings.Repeat("a", 86)
	data, err := Destination(template, "soda.example", fixtureKey(t), hash, "10.89.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	users := config["passwd"].(map[string]interface{})["users"].([]interface{})
	if len(users) != 1 || users[0].(map[string]interface{})["name"] != "root" {
		t.Fatal("unexpected host identities")
	}
	files := config["storage"].(map[string]interface{})["files"].([]interface{})
	paths := map[string]interface{}{}
	for _, file := range files {
		entry := file.(map[string]interface{})
		paths[entry["path"].(string)] = entry
	}
	if len(files) != 3 || paths["/etc/example"] == nil || paths["/etc/hostname"] == nil || paths["/etc/soda-installer/project-subnet"] == nil {
		t.Fatal("public/private file merge failed")
	}
	if !strings.Contains(string(data), "public-unit") || strings.Contains(string(template), hash) {
		t.Fatal("bootstrap mutation")
	}
	for _, bad := range []string{"plaintext", "$6$short", hash + "\n"} {
		if _, err := Destination(template, "soda", fixtureKey(t), bad, "10.0.0.0/24"); err == nil {
			t.Error("invalid password hash accepted")
		}
	}
	for _, bad := range []string{`{"passwd":{},"storage":{"files":[]}}`, `{"storage":{"files":[{"path":"/etc/hostname"}]}}`} {
		if _, err := Destination([]byte(bad), "soda", fixtureKey(t), hash, "10.0.0.0/24"); err == nil {
			t.Error("template collision accepted")
		}
	}
	withBinary, err := addContinuation(data, []byte("fixture-not-an-executable"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(withBinary), "/var/usrlocal/libexec/soda/soda-install") || strings.Contains(string(withBinary), "ExecStart=") {
		t.Fatal("continuation must be explicit, not an automatic install service")
	}
	if _, err := addContinuation(withBinary, []byte("duplicate")); err == nil {
		t.Fatal("continuation collision accepted")
	}
	if _, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(paths["/etc/hostname"].(map[string]interface{})["contents"].(map[string]interface{})["source"].(string), "data:;base64,")); err != nil {
		t.Fatal(err)
	}
}

func fixtureDisk() Disk {
	return Disk{Sequence: "17", Device: BlockDevice{Name: "/dev/sda", KName: "/dev/sda", Type: "disk", Size: 64 << 30, MajorMinor: "8:0", Serial: "fixture", Mountpoints: []*string{nil}, Children: []BlockDevice{{Name: "/dev/sda1", KName: "/dev/sda1", Type: "part", Size: 32 << 30, MajorMinor: "8:1", FSType: "ext4", Mountpoints: []*string{nil}}}}}
}

func TestDiskUseAndIdentity(t *testing.T) {
	disk := fixtureDisk()
	if reason := unused(disk.Device); reason != "" {
		t.Fatal(reason)
	}
	mount := "/run/media/iso"
	for name, change := range map[string]func(*BlockDevice){
		"mounted":        func(d *BlockDevice) { d.Children[0].Mountpoints = []*string{&mount} },
		"read-only":      func(d *BlockDevice) { d.ReadOnly = true },
		"optical":        func(d *BlockDevice) { d.Children[0].FSType = "iso9660" },
		"btrfs":          func(d *BlockDevice) { d.Children[0].FSType = "btrfs" },
		"raid":           func(d *BlockDevice) { d.Children[0].FSType = "linux_raid_member" },
		"mapped":         func(d *BlockDevice) { d.Children[0].Type = "crypt" },
		"missing mounts": func(d *BlockDevice) { d.Mountpoints = nil },
		"invalid name":   func(d *BlockDevice) { d.Name = "/dev/../../secret" },
	} {
		t.Run(name, func(t *testing.T) {
			d := fixtureDisk().Device
			change(&d)
			if unused(d) == "" {
				t.Fatal("busy/unknown disk accepted")
			}
		})
	}
	if err := sameDisk(disk, []Disk{disk}); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Disk){
		"sequence":  func(d *Disk) { d.Sequence = "18" },
		"serial":    func(d *Disk) { d.Device.Serial = "replacement" },
		"partition": func(d *Disk) { d.Device.Children[0].UUID = "new-uuid" },
		"holder":    func(d *Disk) { d.Blocked = "holders" },
	} {
		t.Run(name, func(t *testing.T) {
			d := fixtureDisk()
			change(&d)
			if sameDisk(disk, []Disk{d}) == nil {
				t.Fatal("changed disk accepted")
			}
		})
	}
	if sameDisk(disk, nil) == nil {
		t.Fatal("disappeared disk accepted")
	}
}

func TestDiskExecutionBoundary(t *testing.T) {
	for _, test := range []struct {
		name                                            string
		changed, cancelled, markFailure, installFailure bool
	}{
		{name: "success"}, {name: "hotplug", changed: true}, {name: "cancel", cancelled: true}, {name: "marker failure", markFailure: true}, {name: "partial failure", installFailure: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if test.cancelled {
				cancel()
			}
			calls := 0
			marks := 0
			run := func(ctx context.Context, name string, args []string, input io.Reader) ([]byte, error) {
				calls++
				if marks != 1 || name != "coreos-installer" || !reflect.DeepEqual(args, []string{"install", "--offline", "--ignition-file", "/private/destination.ign", "--copy-network", "/dev/sda"}) || input != nil {
					t.Fatal("unexpected destructive command")
				}
				if test.installFailure {
					return nil, errors.New("synthetic credential from native diagnostics")
				}
				return nil, nil
			}
			err := executeDisk(ctx, fixtureDisk(), "/private/destination.ign", func() ([]Disk, error) {
				d := fixtureDisk()
				if test.changed {
					d.Sequence = "18"
				}
				return []Disk{d}, nil
			}, func() error {
				marks++
				if test.markFailure {
					return errors.New("failed")
				}
				return nil
			}, run)
			early := test.changed || test.cancelled || test.markFailure
			if early && calls != 0 {
				t.Fatal("disk write before safe preflight")
			}
			if !early && calls != 1 {
				t.Fatal("missing or repeated disk install")
			}
			if (early || test.installFailure) != (err != nil) {
				t.Fatalf("unexpected result %v", err)
			}
			if err != nil && strings.Contains(err.Error(), "synthetic credential") {
				t.Fatal("native diagnostics leaked")
			}
		})
	}
}

func TestExtensionActivationRequired(t *testing.T) {
	for _, raw := range []string{`{}`, `{"deployments":[{"booted":false},{"booted":true}]}`, `{"deployments":[{"booted":true,"staged":true}]}`, `{"deployments":[{"booted":true},{"booted":true}]}`, `invalid`} {
		if extensionsBooted([]byte(raw)) == nil {
			t.Fatal("unconfirmed activation accepted")
		}
	}
	if err := extensionsBooted([]byte(`{"deployments":[{"booted":true},{"booted":false}]}`)); err != nil {
		t.Fatal(err)
	}
}

func TestPrivateInputAndOutputBounds(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "input")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegular(file, 6); err == nil {
		t.Fatal("oversize accepted")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegular(link, 100); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := readRegular(dir, 100); err == nil {
		t.Fatal("directory accepted")
	}
	b := boundedOutput{}
	if _, err := b.Write(make([]byte, (8<<20)+1)); err == nil {
		t.Fatal("unbounded command output")
	}
	if _, err := verifyTrustedBundle(dir, "not-a-digest", "x86_64"); err == nil {
		t.Fatal("untrusted bundle accepted")
	}
}

func TestOSReleaseParsing(t *testing.T) {
	values := osRelease([]byte("ID=fedora\nVARIANT_ID=coreos\nVERSION_ID=44\nIMAGE_VERSION='44.20260817.3.2'\n"))
	if values["VERSION_ID"] != "44" || values["IMAGE_VERSION"] != "44.20260817.3.2" || values["VARIANT_ID"] != "coreos" {
		t.Fatal(values)
	}
	media := mediaIdentity{Architecture: "x86_64", Release: "44.20260817.3.2", Revision: strings.Repeat("a", 40)}
	if err := media.validate(values["IMAGE_VERSION"], "x86_64"); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"", values["VERSION_ID"], "44.20260817.3.3"} {
		if err := media.validate(version, "x86_64"); err == nil {
			t.Fatalf("wrong/missing image version accepted: %q", version)
		}
	}
	if err := media.validate(values["IMAGE_VERSION"], "aarch64"); err == nil {
		t.Fatal("wrong native architecture accepted")
	}
	media.Revision = "unreviewed"
	if err := media.validate(values["IMAGE_VERSION"], "x86_64"); err == nil {
		t.Fatal("invalid source revision accepted")
	}
	media.Revision = strings.Repeat("a", 40)
	media.Release = ""
	if err := media.validate("", "x86_64"); err == nil {
		t.Fatal("empty image identity accepted")
	}
}
