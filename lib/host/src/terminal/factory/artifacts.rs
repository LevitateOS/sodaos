use super::run::{valid_commit, valid_factory_role, valid_factory_run_id, valid_preparation_id};
use crate::domain;
use crate::project::Executor;
use crate::terminal::{self, Service};
use std::time::Instant;

/// Exported bundle bound (`4<<20`).
pub const MAX_FACTORY_EXPORT_BUNDLE: usize = 4 << 20;
/// Takeover checkout directory inside a member home.
pub const TAKEOVER_DIR_NAME: &str = "factory-takeover";

/// `project.ErrFactoryExportCandidate`.
pub const ERR_FACTORY_EXPORT_CANDIDATE: &str = "export candidate is not recorded";
/// `project.ErrFactoryExportBounds`.
pub const ERR_FACTORY_EXPORT_BOUNDS: &str = "candidate export exceeds bounds";

/// `TakeoverDestination`: member-owned checkout destination for one run.
pub fn takeover_destination(member: &str, run: &str) -> Option<String> {
    if !domain::valid_login(member) || member == "root" || !valid_factory_run_id(run) {
        return None;
    }
    Some(format!("/home/{member}/{TAKEOVER_DIR_NAME}/{run}"))
}

/// `TakeoverSource`: the exact fixed role-checkout layout, nothing else.
pub fn takeover_source(path: &str, role: &str, preparation: &str) -> bool {
    if !valid_factory_role(role) || !valid_preparation_id(preparation) {
        return false;
    }
    path == format!("/home/{role}/checkouts/{preparation}")
}

/// `factoryExportScript`: objects-only candidate export through a clean
/// bare repository. Golden-pinned against the Go output.
pub const FACTORY_EXPORT_SCRIPT: &str = r#"set -eu
src=$1
candidate=$2
limit=$3
if ! /usr/bin/test -d "$src/.git/objects"; then
  printf 'soda-export-missing\n'
  exit 0
fi
dir=$(/usr/bin/mktemp -d "$TMPDIR/.soda-export-XXXXXX")
trap '/usr/bin/rm -rf "$dir"' EXIT HUP INT TERM
/usr/bin/git -c core.hooksPath=/dev/null init --bare --template= "$dir/repo.git" >/dev/null 2>/dev/null
export GIT_OBJECT_DIRECTORY="$src/.git/objects"
git_export() {
  /usr/bin/git -c core.hooksPath=/dev/null --git-dir="$dir/repo.git" "$@"
}
if ! actual=$(git_export rev-parse --verify "$candidate^{commit}" 2>/dev/null); then
  printf 'soda-export-invalid\n'
  exit 0
fi
if [ "$actual" != "$candidate" ]; then
  printf 'soda-export-invalid\n'
  exit 0
fi
git_export update-ref HEAD "$candidate" 2>/dev/null
git_export bundle create "$dir/candidate.bundle" HEAD 2>/dev/null
/usr/bin/head -c "$limit" "$dir/candidate.bundle"
"#;

/// `FactoryTakeoverCopy` fixed copy steps.
pub fn takeover_steps(src: &str, dest: &str, member: &str) -> Vec<Vec<String>> {
    let partial = format!("{dest}.partial");
    let parent = format!("/home/{member}/{TAKEOVER_DIR_NAME}");
    vec![
        vec!["/usr/bin/mkdir".to_string(), "-p".to_string(), parent],
        vec![
            "/usr/bin/rm".to_string(),
            "-rf".to_string(),
            partial.clone(),
        ],
        vec!["/usr/bin/mkdir".to_string(), partial.clone()],
        vec![
            "/usr/bin/cp".to_string(),
            "-a".to_string(),
            format!("{src}/."),
            format!("{partial}/"),
        ],
        vec![
            "/usr/bin/rm".to_string(),
            "-rf".to_string(),
            format!("{partial}/.git"),
            format!("{partial}/.soda-home"),
        ],
        vec![
            "/usr/bin/git".to_string(),
            "-C".to_string(),
            partial.clone(),
            "init".to_string(),
            "-q".to_string(),
        ],
        vec![
            "/usr/bin/chown".to_string(),
            "-R".to_string(),
            format!("--reference=/home/{member}"),
            partial,
        ],
    ]
}

/// `FactoryExportBundle` guest env (fixed, clean) plus argv tail.
pub fn export_argv(current: &str, role: &str, src: &str, candidate: &str) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--user".to_string(),
        role.to_string(),
        current.to_string(),
        "/usr/bin/env".to_string(),
        "-i".to_string(),
        "PATH=/usr/bin:/bin".to_string(),
        format!("HOME=/home/{role}"),
        "LC_ALL=C".to_string(),
        format!("TMPDIR=/home/{role}/checkouts"),
        "GIT_CONFIG_NOSYSTEM=1".to_string(),
        "GIT_CONFIG_GLOBAL=/dev/null".to_string(),
        "GIT_NO_REPLACE_OBJECTS=1".to_string(),
        "GIT_TERMINAL_PROMPT=0".to_string(),
        "/usr/bin/sh".to_string(),
        "-c".to_string(),
        FACTORY_EXPORT_SCRIPT.to_string(),
        "soda-export".to_string(),
        src.to_string(),
        candidate.to_string(),
        (MAX_FACTORY_EXPORT_BUNDLE + 1).to_string(),
    ]
}

impl<E: Executor> Service<E> {
    /// `Service.FactoryExportBundle`: exact-candidate bundle export.
    pub fn factory_export_bundle(
        &self,
        project_id: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let src = format!("/home/{role}/checkouts/{preparation}");
        if !takeover_source(&src, role, preparation)
            || !valid_commit(candidate)
            || !domain::valid_container_id(recorded)
        {
            return Err(terminal::err_denied());
        }
        let current = self.factory_project_container(project_id, true, deadline)?;
        if current != recorded {
            return Err(terminal::err_stale());
        }
        let argv = export_argv(&current, role, &src, candidate);
        let bundle = match self.run_podman(&[], &argv, deadline) {
            Ok(bundle) => bundle,
            Err(_) => {
                if Instant::now() >= deadline {
                    return Err("context deadline exceeded".to_string());
                }
                return Err("candidate export execution unconfirmed".to_string());
            }
        };
        if bundle == b"soda-export-missing\n" {
            return Err(terminal::ERR_NOT_FOUND.to_string());
        }
        if bundle == b"soda-export-invalid\n" {
            return Err(ERR_FACTORY_EXPORT_CANDIDATE.to_string());
        }
        if bundle.is_empty() || bundle.len() > MAX_FACTORY_EXPORT_BUNDLE {
            return Err(ERR_FACTORY_EXPORT_BOUNDS.to_string());
        }
        Ok(bundle)
    }

    /// `Service.FactoryTakeoverCopy`: copy retained role-checkout work
    /// into the admitted member's derived destination.
    #[allow(clippy::too_many_arguments)] // one parameter per Go argument, in order
    pub fn factory_takeover_copy(
        &self,
        project_id: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        member: &str,
        run: &str,
        deadline: Instant,
    ) -> Result<(String, bool), String> {
        let dest = takeover_destination(member, run).ok_or_else(terminal::err_denied)?;
        let src = format!("/home/{role}/checkouts/{preparation}");
        if !takeover_source(&src, role, preparation) || !domain::valid_container_id(recorded) {
            return Err(terminal::err_denied());
        }
        let current = self.factory_project_container(project_id, true, deadline)?;
        if current != recorded {
            return Err(terminal::err_stale());
        }
        let probe = |args: &[String]| self.run_podman(&[], args, deadline);
        let test_dir = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            current.clone(),
            "/usr/bin/test".to_string(),
            "-d".to_string(),
            dest.clone(),
        ];
        if probe(&test_dir).is_ok() {
            return Ok((dest, true));
        }
        let partial = format!("{dest}.partial");
        let test_src = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            current.clone(),
            "/usr/bin/test".to_string(),
            "-d".to_string(),
            src.clone(),
        ];
        if probe(&test_src).is_err() {
            return Err("takeover found no retained work".to_string());
        }
        for step in takeover_steps(&src, &dest, member) {
            let mut argv = vec![
                "--remote=false".to_string(),
                "exec".to_string(),
                current.clone(),
            ];
            argv.extend(step);
            match probe(&argv) {
                Ok(out) if out.len() <= 65536 => {}
                Ok(_) => return Err("takeover step returned excessive output".to_string()),
                Err(err) => return Err(err),
            }
        }
        let test_dest = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            current.clone(),
            "/usr/bin/test".to_string(),
            "-e".to_string(),
            dest.clone(),
        ];
        if probe(&test_dest).is_ok() {
            let cleanup = vec![
                "--remote=false".to_string(),
                "exec".to_string(),
                current,
                "/usr/bin/rm".to_string(),
                "-rf".to_string(),
                partial,
            ];
            let _ = probe(&cleanup);
            return Ok((dest, true));
        }
        let mv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            current.clone(),
            "/usr/bin/mv".to_string(),
            "-T".to_string(),
            partial.clone(),
            dest.clone(),
        ];
        if probe(&mv).is_err() {
            let cleanup = vec![
                "--remote=false".to_string(),
                "exec".to_string(),
                current,
                "/usr/bin/rm".to_string(),
                "-rf".to_string(),
                partial,
            ];
            let _ = probe(&cleanup);
            return Err("takeover destination was not confirmed".to_string());
        }
        Ok((dest, false))
    }
}
