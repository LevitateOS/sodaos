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
