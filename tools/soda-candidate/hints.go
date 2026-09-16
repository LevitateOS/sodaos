// hints maps failure signatures the wrapper has actually seen to the one
// operator action that fixes them. Unknown failures get no guess: the
// build log path in the failure panel is the whole advice.
package main

import "strings"

// failureHint returns the fix for a failure reason, or "" when the reason
// matches nothing known. Matching is case-insensitive substring on the
// already one-line reason.
func failureHint(reason string) string {
	lowered := strings.ToLower(reason)
	for _, h := range hintCatalog {
		if strings.Contains(lowered, h.signature) {
			return h.fix
		}
	}
	return ""
}

var hintCatalog = []struct {
	signature string
	fix       string
}{
	{
		"could not import",
		"Go cannot map its build cache under the worker domain; rerun bash scripts/setup-soda-candidate.sh from the repo root.",
	},
	{
		"go failed",
		"Go cannot run in the worker sandbox (often cache mapping); rerun bash scripts/setup-soda-candidate.sh from the repo root.",
	},
	{
		"permission denied",
		"A provisioned file, label, or directory blocks the worker; rerun bash scripts/setup-soda-candidate.sh from the repo root.",
	},
	{
		"goproxy",
		"The worker builds offline from warmed caches; rerun bash scripts/setup-soda-candidate.sh from the repo root.",
	},
	{
		"module lookup disabled",
		"The worker builds offline from warmed caches; rerun bash scripts/setup-soda-candidate.sh from the repo root.",
	},
	{
		"already present",
		"A previous worker unit still exists; wait for it to finish or stop it, then retry.",
	},
	{
		"interactive authentication required",
		"A worker container could not use systemd cgroups (no user session); rerun bash scripts/setup-soda-candidate.sh from the repo root.",
	},
}
