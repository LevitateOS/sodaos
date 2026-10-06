use std::time::Instant;

use crate::preparation;
use crate::project::{Config, Executor, Runtime};

use super::receipt::receipt_terminal;
use super::{
    factory_run_paths, Factory, FactoryBroker, FactoryCandidateInspect, FactoryCandidateState,
    FactoryError, FactoryTerminal,
};

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// `InspectCandidate`: resolve role and checkout from the settled host
    /// receipt. An old run never reads a replacement Project container: a
    /// container mismatch, like any observation failure, refuses stale.
    pub fn inspect_candidate(
        &self,
        req: &FactoryCandidateInspect,
        deadline: Instant,
    ) -> Result<FactoryCandidateState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        drop(lock);
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = match &receipt.binding {
            Some(binding) if receipt_terminal(&receipt.phase) && receipt.run.validate().is_ok() => {
                binding.clone()
            }
            _ => {
                return Err(FactoryError::msg(
                    "candidate inspection requires a settled run",
                ))
            }
        };
        // Go builds a zero-config Runtime here; container binding needs no
        // configuration.
        let runtime = Runtime {
            exec: &self.exec,
            config: Config::default(),
        };
        let container = runtime.prepare_container(&req.project, true, deadline);
        let container = match container {
            Ok(container) if container == binding.project => container,
            _ => return Err(FactoryError::Stale),
        };
        let (checkout, _, _, _) =
            factory_run_paths(&receipt.run.role, &receipt.run.preparation, &receipt.run.id);
        let home = format!("/home/{}", receipt.run.role);
        let tmpdir = format!("/home/{}/checkouts", receipt.run.role);
        let path = "PATH=/usr/bin:/bin";
        let home_env = format!("HOME={home}");
        let tmpdir_env = format!("TMPDIR={tmpdir}");
        let args = [
            "exec",
            "--user",
            receipt.run.role.as_str(),
            container.as_str(),
            "/usr/bin/env",
            "-i",
            path,
            home_env.as_str(),
            "LC_ALL=C",
            tmpdir_env.as_str(),
            "GIT_CONFIG_NOSYSTEM=1",
            "GIT_CONFIG_GLOBAL=/dev/null",
            "GIT_NO_REPLACE_OBJECTS=1",
            "GIT_TERMINAL_PROMPT=0",
            "GIT_OPTIONAL_LOCKS=0",
            "/usr/bin/sh",
            "-c",
            CANDIDATE_INSPECT_SCRIPT,
            "soda-candidate",
            checkout.as_str(),
        ];
        let out = runtime
            .exec
            .run(&[], "/usr/bin/podman", &args, deadline)
            .map_err(|_| FactoryError::msg("candidate checkout inspection unconfirmed"))?;
        if out.len() > 64 {
            return Err(FactoryError::msg(
                "candidate checkout inspection unconfirmed",
            ));
        }
        let text = String::from_utf8_lossy(&out);
        let fields: Vec<&str> = text.split_whitespace().collect();
        if fields.len() != 2
            || !preparation::valid_commit(fields[0])
            || (fields[1] != "clean" && fields[1] != "dirty")
        {
            return Err(FactoryError::msg(
                "candidate checkout observation is invalid",
            ));
        }
        Ok(FactoryCandidateState {
            id: req.id.clone(),
            project: req.project.clone(),
            container,
            candidate: fields[0].to_string(),
            dirty: fields[1] == "dirty",
        })
    }
}

pub(in crate::factory) fn run_reason(cause: &FactoryError) -> &'static str {
    match cause {
        FactoryError::Denied => "broker-denied",
        FactoryError::Uncertain => "broker-unavailable",
        FactoryError::DeadlineExceeded => "deadline-exceeded",
        _ => "launch-refused",
    }
}

// Only rev-parse reads source metadata; it executes no filters or hooks.
// Every worktree/index comparison uses a clean repository, source objects
// and a disposable index. No agent configuration or optional external diff
// runs. Byte-exact with Go's `candidateInspectScript`, trailing newline
// included.
pub const CANDIDATE_INSPECT_SCRIPT: &str = r#"set -eu
umask 077
src=$1
head=$(/usr/bin/git --git-dir="$src/.git" rev-parse --verify 'HEAD^{commit}')
dir=$(/usr/bin/mktemp -d "$TMPDIR/.soda-candidate-XXXXXX")
trap '/usr/bin/rm -rf "$dir"' EXIT HUP INT TERM
/usr/bin/git -c core.hooksPath=/dev/null init --bare --template= "$dir/repo.git" >/dev/null 2>/dev/null
export GIT_OBJECT_DIRECTORY="$src/.git/objects"
git_candidate() {
  /usr/bin/git -c core.hooksPath=/dev/null -c core.fsmonitor=false --git-dir="$dir/repo.git" --work-tree="$src" "$@"
}
dirty=clean
export GIT_INDEX_FILE="$src/.git/index"
if git_candidate diff-index --cached --quiet --no-ext-diff --no-textconv "$head" --; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"; dirty=dirty
fi
export GIT_INDEX_FILE="$dir/index"
git_candidate read-tree "$head"
if git_candidate update-index --refresh >/dev/null; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"
fi
if git_candidate diff-files --quiet --no-ext-diff --no-textconv --; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"; dirty=dirty
fi
# Standard exclusions read worktree .gitignore with this clean repository's
# empty info/exclude and no global configuration. A new visible .gitignore must
# itself remain visible, even if its rules try to ignore that file. Already
# ignored build directories stay excluded without traversing their contents.
git_candidate ls-files --others --exclude-standard --exclude='!**/.gitignore' --exclude=.git/ --exclude=.soda-home/ >"$dir/untracked"
[ ! -s "$dir/untracked" ] || dirty=dirty
printf '%s %s\n' "$head" "$dirty"
"#;
