package acceptance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

// ValidOwner labels an observation; it never grants execution permission.
func ValidOwner(owner string) bool {
	switch owner {
	case "P02", "P03", "P04", "P05", "P06", "P11":
		return true
	}
	return regexp.MustCompile(`^U(?:0[1-9]|1[0-9]|20)$`).MatchString(owner)
}

// Observation is an ordinary log index, not a scenario/qualification registry.
type Observation struct {
	Owner, RequestedRevision, ToolRevision, RequestedArchitecture, Target, ClientPlatform, Action, Outcome, Execution, Evidence, Cleanup, Topology string
	ExitCode                                                                                                                                       *int
	ToolDirty                                                                                                                                      bool
	Invocation                                                                                                                                     []string
	Files                                                                                                                                          map[string]string
	Artifacts                                                                                                                                      map[string]string
	Started, Finished                                                                                                                              time.Time
}

func (e *Evidence) Hashes() (map[string]string, error) {
	files := map[string]string{}
	err := fs.WalkDir(e.root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("unexpected evidence entry")
		}
		sum, err := nativebuild.HashAt(e.root, path)
		if err != nil {
			return err
		}
		files[path] = sum
		return nil
	})
	return files, err
}

// readObservation binds decoded metadata and all retained-file hashes through
// one open directory, even if its pathname is renamed during the read.
func readObservation(file string) (Observation, string, error) {
	var o Observation
	root, err := os.OpenRoot(filepath.Dir(file))
	if err != nil {
		return o, "", err
	}
	defer root.Close()
	digest, err := nativebuild.ReadJSONAt(root, filepath.Base(file), &o)
	if err != nil {
		return o, "", err
	}
	for name, sum := range o.Artifacts {
		if name == "" || !nativebuild.Digest(sum) {
			return o, "", errors.New("invalid public artifact reference")
		}
	}
	for name, sum := range o.Files {
		if !filepath.IsLocal(name) || filepath.Clean(name) != name || name == "." {
			return o, "", errors.New("unsafe observation reference")
		}
		actual, err := nativebuild.HashAt(root, name)
		if err != nil || actual != sum {
			return o, "", errors.New("observation bytes changed")
		}
	}
	return o, digest, nil
}

// Handoff cites exact observations and explicitly leaves missing scopes open.
// It can complete without core product evidence and never certifies readiness.
func Handoff(out, arch, revision string, records []string) error {
	if err := nativebuild.PrivateDestination(out); err != nil {
		return err
	}
	if _, err := nativebuild.OCIArchitecture(arch); err != nil {
		return err
	}
	if !nativebuild.Revision(revision) {
		return errors.New("full candidate revision required")
	}
	var text strings.Builder
	fmt.Fprintf(&text, "# Native support handoff\n\nCandidate: `%s` / `%s`.\n\nProduct readiness: **not assessed here; U20 owns it**. No ISO/QCOW2 delivery selected by this report. Sibling architecture is independent.\n\nCandidate/target fields are requested identities, not independently discovered facts for arbitrary exec commands. Read each owner's invoked check and retained artifact references for actual binding. Artifact references are historical digests, not a fresh verification of files at their former locations.\n\n", revision, arch)
	seen := map[string]bool{}
	for _, file := range records {
		if filepath.Base(file) != "observation.json" {
			return errors.New("finalized observation.json required")
		}
		o, digest, err := readObservation(file)
		if err != nil {
			return err
		}
		if !ValidOwner(o.Owner) {
			return errors.New("unknown observation owner")
		}
		switch o.Outcome {
		case "completed":
			if o.Execution != "completed" || o.Evidence != "completed" || len(o.Files) == 0 {
				return errors.New("completed observation lacks execution/evidence")
			}
		case "failed", "cancelled":
		default:
			return errors.New("unknown observation outcome")
		}
		if o.RequestedRevision != revision || o.RequestedArchitecture != arch {
			return errors.New("observation belongs to a different candidate/platform")
		}
		// JSON quoting avoids Markdown/control injection from an external log index.
		description, _ := json.Marshal(struct {
			Owner, Target, Outcome, Execution, Evidence, Cleanup, Topology string
			ExitCode                                                       *int
			Invocation                                                     []string
			Artifacts                                                      map[string]string
		}{o.Owner, o.Target, o.Outcome, o.Execution, o.Evidence, o.Cleanup, o.Topology, o.ExitCode, o.Invocation, o.Artifacts})
		location, _ := json.Marshal(file)
		fmt.Fprintf(&text, "    %s\n    record-sha256: %s\n    path: %s\n\n", description, digest, location)
		seen[o.Owner] = true
	}
	text.WriteString("\n## Not reached / not supplied\n\n")
	owners := []string{"P02", "P03", "P04", "P05", "P06", "P11", "U08", "U20"}
	sort.Strings(owners)
	for _, owner := range owners {
		if !seen[owner] {
			fmt.Fprintf(&text, "- %s: no observation supplied (not a pass).\n", owner)
		}
	}
	text.WriteString("\nP07/P08 redirect to core U08/U20. P09/P10 remain not selected. A command completing, a hash matching or a version printing proves only that observation. Retained paths and provider cleanup must be reviewed alongside the cited logs; missing cleanup is not inferred successful. Failed/cancelled/evidence-failed observations remain failures. This report performs no tests, retries, provider mutations or publication.\n")
	return nativebuild.WriteNew(out, []byte(text.String()), 0600)
}
