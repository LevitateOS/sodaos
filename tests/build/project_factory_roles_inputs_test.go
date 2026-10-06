package build

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRolesApproveWritesProtectedSnapshotAndVerifiesBundle(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	decoded := rolesOK(t, fenv, rolesFixture(t, rolesPID))
	Check(t, decoded["approved"] == rolesPID, "approved = %v", decoded["approved"])
	Check(t, decoded["repeated"] == false, "repeated = %v", decoded["repeated"])
	snapshot := filepath.Join(fenv.factory, "preparations", rolesPID, "snapshot")
	setup, err := os.ReadFile(filepath.Join(snapshot, "setup.sh"))
	Require(t, err == nil, "read setup: %v", err)
	Check(t, string(setup) == "true\n", "setup = %q", setup)
	Check(t, rolesMode(t, filepath.Join(snapshot, "setup.sh")) == 0o644, "setup mode")
	Check(t, rolesMode(t, filepath.Join(snapshot, "source.bundle")) == 0o644, "bundle mode")
	// Exact verification argv, in order.
	record, err := os.ReadFile(fenv.record)
	Require(t, err == nil, "read git record: %v", err)
	bundle := filepath.Join(snapshot, "source.bundle")
	repo := filepath.Join(snapshot, "verify-tmp", "repo")
	Check(t, strings.Contains(string(record),
		"---\n<clone>\n<-q>\n<--no-checkout>\n<"+bundle+">\n<"+repo+">\n"), "clone argv:\n%s", record)
	Check(t, strings.Contains(string(record),
		"---\n<-C>\n<"+repo+">\n<cat-file>\n<-e>\n<"+rolesCommit+">\n"), "cat-file argv:\n%s", record)
	_, err = os.Stat(filepath.Join(snapshot, "verify-tmp"))
	Check(t, os.IsNotExist(err), "verify-tmp remains")
	repeated := rolesOK(t, fenv, rolesFixture(t, rolesPID))
	Check(t, repeated["repeated"] == true, "repeat not reported")
	conflict := map[string]any{}
	Require(t, json.Unmarshal(rolesFixture(t, rolesPID), &conflict) == nil, "decode fixture")
	conflict["setup_digest"] = strings.Repeat("e", 64)
	rolesRefused(t, fenv, rolesMarshal(t, conflict))
}

func TestRolesApproveRejectsUntrustedInputsBeforeEffects(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	var base map[string]any
	Require(t, json.Unmarshal(rolesFixture(t, rolesPID), &base) == nil, "decode fixture")
	clone := func() map[string]any {
		out := map[string]any{}
		for key, value := range base {
			out[key] = value
		}
		return out
	}
	single := clone()
	single["files"] = map[string]any{"setup.sh": base64.StdEncoding.EncodeToString([]byte("x"))}
	cases := []struct {
		name  string
		value map[string]any
	}{
		{"role", func() map[string]any { v := clone(); v["role"] = "root"; return v }()},
		{"id", func() map[string]any { v := clone(); v["id"] = "../escape"; return v }()},
		{"digest", func() map[string]any { v := clone(); v["setup_digest"] = "zz"; return v }()},
		{"commit", func() map[string]any { v := clone(); v["source_commit"] = "short"; return v }()},
		{"credential", func() map[string]any { v := clone(); v["credential"] = "../x"; return v }()},
		{"files", single},
		{"bundle", func() map[string]any { v := clone(); v["bundle"] = "!!!"; return v }()},
		{"extra", func() map[string]any { v := clone(); v["extra"] = 1; return v }()},
	}
	Require(t, len(cases) == 8, "cases = %d", len(cases))
	for _, tc := range cases {
		rolesRefused(t, fenv, rolesMarshal(t, tc.value))
	}
	_, err := os.Stat(fenv.record)
	Check(t, os.IsNotExist(err), "git ran before refusal")
	_, err = os.Stat(filepath.Join(fenv.factory, "preparations", rolesPID))
	Check(t, os.IsNotExist(err), "effects before refusal")
}
