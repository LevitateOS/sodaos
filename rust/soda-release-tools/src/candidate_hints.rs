//! Failure hints (Go `tools/soda-candidate` `hints.go`).

struct Hint {
    signature: &'static str,
    fix: &'static str,
}

const HINT_CATALOG: &[Hint] = &[
    Hint {
        signature: "could not import",
        fix: "Go cannot map its build cache under the worker domain; rerun cargo run -p soda-candidate-setup from the repo root.",
    },
    Hint {
        signature: "go failed",
        fix: "Go cannot run in the worker sandbox (often cache mapping); rerun cargo run -p soda-candidate-setup from the repo root.",
    },
    Hint {
        signature: "permission denied",
        fix: "A provisioned file, label, or directory blocks the worker; rerun cargo run -p soda-candidate-setup from the repo root.",
    },
    Hint {
        signature: "goproxy",
        fix: "The worker builds offline from warmed caches; rerun cargo run -p soda-candidate-setup from the repo root.",
    },
    Hint {
        signature: "module lookup disabled",
        fix: "The worker builds offline from warmed caches; rerun cargo run -p soda-candidate-setup from the repo root.",
    },
    Hint {
        signature: "already present",
        fix: "A previous worker unit still exists; wait for it to finish or stop it, then retry.",
    },
    Hint {
        signature: "interactive authentication required",
        fix: "A worker container could not use systemd cgroups (no user session); rerun cargo run -p soda-candidate-setup from the repo root.",
    },
];

pub fn failure_hint(reason: &str) -> &'static str {
    let lowered = reason.to_lowercase();
    for hint in HINT_CATALOG {
        if lowered.contains(hint.signature) {
            return hint.fix;
        }
    }
    ""
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn known_signatures_map_to_fixes() {
        for (reason, want) in [
            (
                "open /run/go/src/a.go: permission denied",
                "soda-candidate-setup",
            ),
            (
                "internal/strictjson/decode.go:5:2: could not import bytes (permission denied)",
                "map its build cache",
            ),
            (
                "go failed; retain attempt and inspect build.log: exit status 1",
                "soda-candidate-setup",
            ),
            (
                "GOPROXY list is not the empty string",
                "soda-candidate-setup",
            ),
            (
                "go: module lookup disabled by GOPROXY=off",
                "soda-candidate-setup",
            ),
            (
                "worker unit is already present or could not be checked",
                "wait for it",
            ),
            (
                "sd-bus call: Interactive authentication required",
                "soda-candidate-setup",
            ),
        ] {
            let got = failure_hint(reason);
            assert!(got.contains(want), "{reason} -> {got}");
        }
        assert_eq!(failure_hint("some brand-new failure mode"), "");
        assert_eq!(failure_hint(""), "");
    }
}
