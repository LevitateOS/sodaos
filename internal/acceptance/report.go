package acceptance

import (
	"encoding/json"
	"errors"
	"fmt"
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
	err := filepath.WalkDir(e.Path(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("unexpected evidence entry")
		}
		rel, err := filepath.Rel(e.Path(), path)
		if err != nil {
			return err
		}
		sum, err := nativebuild.HashFile(path)
		if err != nil {
			return err
		}
		files[rel] = sum
		return nil
	})
	return files, err
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
	fmt.Fprintf(&text, "# Native support handoff\n\nCandidate: `%s` / `%s`.\n\nProduct readiness: **not assessed here; U20 owns it**. No ISO/QCOW2 delivery selected by this report. Sibling architecture is independent.\n\n", revision, arch)
	seen := map[string]bool{}
	for _, file := range records {
		var o Observation
		if err := nativebuild.ReadJSON(file, &o); err != nil {
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
		root, err := os.OpenRoot(filepath.Dir(file))
		if err != nil {
			return err
		}
		for name, sum := range o.Files {
			if !filepath.IsLocal(name) {
				root.Close()
				return errors.New("unsafe observation reference")
			}
			st, err := root.Lstat(name)
			if err != nil || !st.Mode().IsRegular() {
				root.Close()
				return errors.New("non-regular observation reference")
			}
			actual, err := nativebuild.HashFile(filepath.Join(filepath.Dir(file), name))
			if err != nil || actual != sum {
				root.Close()
				return errors.New("observation bytes changed")
			}
		}
		if err = root.Close(); err != nil {
			return err
		}
		digest, err := nativebuild.HashFile(file)
		if err != nil {
			return err
		}
		// JSON quoting avoids Markdown/control injection from an external log index.
		description, _ := json.Marshal(struct{ Owner, Target, Outcome, Execution, Evidence string }{o.Owner, o.Target, o.Outcome, o.Execution, o.Evidence})
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
