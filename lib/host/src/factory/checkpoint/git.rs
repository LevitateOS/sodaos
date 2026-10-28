use std::time::Instant;

use serde::{Deserialize, Serialize};

use crate::project::Executor;
use crate::terminal::factory::native::shell_quote;

use super::super::{factory_run_paths, FactoryRun};
use super::state::CheckpointOperation;
use super::wire::{CheckpointTask, FactoryTaskCheckpoint};

#[derive(Clone, Debug, Deserialize, Serialize)]
pub(super) struct GitContext {
    pub container: String,
    pub role: String,
    pub uid: i64,
    pub gid: i64,
    pub checkout: String,
    pub home: String,
    pub work: String,
    pub binding: String,
}

impl GitContext {
    pub(super) fn new(
        run: &FactoryRun,
        req: &FactoryTaskCheckpoint,
        uid: i64,
        gid: i64,
        key: &str,
    ) -> Self {
        let (checkout, _, home) = factory_run_paths(&run.role, &run.preparation, &run.id);
        Self {
            container: req.container.clone(),
            role: run.role.clone(),
            uid,
            gid,
            checkout,
            home,
            work: format!("/home/.soda-factory-checkpoints/{}/{key}", run.id),
            binding: format!("{}:{}:{}", run.id, req.manifest_digest, req.invocation),
        }
    }

    pub(super) fn exec<E: Executor>(
        &self,
        exec: &E,
        script: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let command = format!(
            "set -eu\numask 077\nsrc={}\nwork={}\nbinding={}\n{}\n{}",
            shell_quote(&self.checkout),
            shell_quote(&self.work),
            shell_quote(&self.binding),
            CLEAN_ENV,
            script
        );
        exec.run_bounded(
            &[],
            "/usr/bin/podman",
            &[
                "--remote=false",
                "exec",
                "--user",
                "0",
                &self.container,
                "/usr/bin/sh",
                "-c",
                &command,
            ],
            deadline,
            1024 * 1024,
            64 * 1024,
        )
    }

    pub(super) fn guard<E: Executor>(
        &self,
        exec: &E,
        source: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let script = format!(
            r#"
[ "$(/usr/bin/realpath -e "$src")" = "$src" ]
[ -d "$src/.git" ] && [ ! -L "$src/.git" ]
[ -z "$(/usr/bin/find "$src/.git" -type l -print -quit)" ]
head=$(/usr/bin/git -c core.hooksPath=/dev/null --git-dir="$src/.git" rev-parse --verify 'HEAD^{{commit}}')
[ "$head" = {source} ]
index=$(/usr/bin/git -c core.hooksPath=/dev/null --git-dir="$src/.git" write-tree)
tree=$(/usr/bin/git -c core.hooksPath=/dev/null --git-dir="$src/.git" rev-parse 'HEAD^{{tree}}')
[ "$index" = "$tree" ]
/usr/bin/chown 0:{gid} "$src"
/usr/bin/chmod 1770 "$src"
/usr/bin/chown -R 0:{gid} "$src/.git"
/usr/bin/find "$src/.git" -type d -exec /usr/bin/chmod 750 {{}} +
/usr/bin/find "$src/.git" -type f -exec /usr/bin/chmod 640 {{}} +
/usr/bin/printf '%s\n' "$binding" >"$src/.git/soda-factory-binding"
/usr/bin/chmod 640 "$src/.git/soda-factory-binding"
/usr/bin/sync -f "$src/.git/soda-factory-binding"
/usr/bin/sync -f "$src/.git"
"#,
            source = shell_quote(source),
            gid = self.gid
        );
        self.exec(exec, &script, deadline).map(|_| ())
    }

    fn prefix(&self) -> String {
        format!(
            r#"
[ "$(/usr/bin/realpath -e "$src")" = "$src" ]
[ -d "$src/.git" ] && [ ! -L "$src/.git" ]
[ "$(/usr/bin/stat -c '%u' "$src/.git")" = 0 ]
[ "$(/usr/bin/cat "$src/.git/soda-factory-binding")" = "$binding" ]
for dir in /home/.soda-factory-checkpoints /home/.soda-factory-checkpoints/{run} "$work"; do
  if [ ! -e "$dir" ]; then /usr/bin/mkdir -m 711 "$dir"; /usr/bin/chown 0:0 "$dir"; fi
  [ ! -L "$dir" ] && [ "$(/usr/bin/stat -c '%u:%g:%a' "$dir")" = 0:0:711 ]
done
if [ ! -d "$work/repo.git" ]; then
  /usr/bin/git -c core.hooksPath=/dev/null init --bare --template= "$work/repo.git" >/dev/null 2>/dev/null
fi
[ ! -L "$work/repo.git" ] && [ "$(/usr/bin/stat -c '%u' "$work/repo.git")" = 0 ]
export GIT_OBJECT_DIRECTORY="$src/.git/objects"
git_private() {{ /usr/bin/git -c core.hooksPath=/dev/null -c core.fsmonitor=false --git-dir="$work/repo.git" --work-tree="$src" "$@"; }}
head=$(/usr/bin/git -c core.hooksPath=/dev/null --git-dir="$src/.git" rev-parse --verify 'HEAD^{{commit}}')
parent_tree=$(git_private rev-parse "$head^{{tree}}")
index_tree=$(GIT_INDEX_FILE="$src/.git/index" git_private write-tree)
"#,
            run = self.work.split('/').nth(3).unwrap_or("")
        )
    }

    pub(super) fn observe<E: Executor>(
        &self,
        exec: &E,
        task: &CheckpointTask,
        expected: &str,
        deadline: Instant,
    ) -> Result<(String, String), String> {
        let mut script = self.prefix();
        script.push_str(&format!("[ \"$head\" = {} ]\n[ \"$index_tree\" = \"$parent_tree\" ]\nexport GIT_INDEX_FILE=\"$work/output.index\"\ngit_private read-tree --empty\n", shell_quote(expected)));
        for path in &task.owned_paths {
            let path = path.trim_end_matches('/');
            script.push_str(&format!("path={}\nif [ -e \"$src/$path\" ] || [ -L \"$src/$path\" ]; then\n [ \"$(/usr/bin/realpath -m \"$src/$path\")\" = \"$src/$path\" ]\n [ -z \"$(/usr/bin/find \"$src/$path\" -type l -print -quit)\" ]\n git_private --literal-pathspecs add --all -- {}\nfi\n", shell_quote(path), shell_quote(path)));
        }
        script.push_str(
            "tree=$(git_private write-tree)\nprintf '%s %s\\n' \"$parent_tree\" \"$tree\"\n",
        );
        let bytes = self.exec(exec, &script, deadline)?;
        let text = std::str::from_utf8(&bytes).map_err(|_| "invalid output tree observation")?;
        let fields: Vec<&str> = text.split_whitespace().collect();
        if fields.len() != 2 || fields.iter().any(|s| !crate::preparation::valid_commit(s)) {
            return Err("invalid output tree observation".into());
        }
        Ok((fields[0].into(), fields[1].into()))
    }

    pub(super) fn integrate<E: Executor>(
        &self,
        exec: &E,
        task: &CheckpointTask,
        expected: &str,
        released_tree: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        let mut script = self.prefix();
        script.push_str(&format!("[ \"$head\" = {} ]\n[ \"$index_tree\" = \"$parent_tree\" ]\nexport GIT_INDEX_FILE=\"$work/integration.index\"\ngit_private read-tree \"$head\"\n", shell_quote(expected)));
        for path in &task.owned_paths {
            script.push_str(&format!("git_private --literal-pathspecs ls-files -z -- {} | git_private update-index --force-remove -z --stdin\n", shell_quote(path.trim_end_matches('/'))));
        }
        script.push_str(&format!("git_private ls-tree -r -z {} | git_private update-index -z --index-info\ngit_private write-tree\n", shell_quote(released_tree)));
        let bytes = self.exec(exec, &script, deadline)?;
        let tree = std::str::from_utf8(&bytes)
            .map_err(|_| "invalid integrated task tree")?
            .trim();
        if !crate::preparation::valid_commit(tree) {
            return Err("invalid integrated task tree".into());
        }
        Ok(tree.into())
    }

    pub(super) fn prepare<E: Executor>(
        &self,
        exec: &E,
        operation: &CheckpointOperation,
        deadline: Instant,
    ) -> Result<String, String> {
        let mut script = self.prefix();
        script.push_str(&format!(
            "[ \"$head\" = {} ]\n[ \"$index_tree\" = {} ]\n",
            shell_quote(&operation.request.expected_head),
            shell_quote(&operation.parent_tree)
        ));
        if operation.final_candidate || operation.tree == operation.parent_tree {
            script.push_str("candidate=$head\n");
        } else {
            let message = format!(
                "Factory task {}\n\nRun: {}\nOutput: {}",
                operation.request.task_key, operation.request.id, operation.request.output_digest
            );
            script.push_str(&format!("export GIT_AUTHOR_NAME={} GIT_COMMITTER_NAME={} GIT_AUTHOR_EMAIL={} GIT_COMMITTER_EMAIL={} GIT_AUTHOR_DATE={} GIT_COMMITTER_DATE={}\ncandidate=$(git_private commit-tree {} -p \"$head\" -m {})\n", shell_quote(&self.role), shell_quote(&self.role), shell_quote(&format!("{}@localhost", self.role)), shell_quote(&format!("{}@localhost", self.role)), shell_quote(&format!("{} +0000", operation.created_unix)), shell_quote(&format!("{} +0000", operation.created_unix)), shell_quote(&operation.tree), shell_quote(&message)));
        }
        script.push_str(&format!(r#"
/usr/bin/chown -R 0:{gid} "$src/.git/objects"
/usr/bin/find "$src/.git/objects" -type d -exec /usr/bin/chmod 750 {{}} +
/usr/bin/find "$src/.git/objects" -type f -exec /usr/bin/chmod 640 {{}} +
if [ ! -d "$work/export" ]; then
  /usr/bin/mkdir -m 700 "$work/export"
  (unset GIT_OBJECT_DIRECTORY; /usr/bin/git -c core.hooksPath=/dev/null init --template= "$work/export" >/dev/null 2>/dev/null)
  /usr/bin/printf '%s\n' "$src/.git/objects" >"$work/export/.git/objects/info/alternates"
  /usr/bin/git -c core.hooksPath=/dev/null --git-dir="$work/export/.git" update-ref --no-deref HEAD "$candidate"
  GIT_INDEX_FILE="$work/export/.git/index" git_private read-tree "$candidate"
  GIT_INDEX_FILE="$work/export/.git/index" git_private checkout-index --all --prefix="$work/export/"
  /usr/bin/chown -R {uid}:{gid} "$work/export"
fi
printf '%s\n' "$candidate"
"#, uid=self.uid, gid=self.gid));
        let bytes = self.exec(exec, &script, deadline)?;
        let candidate = std::str::from_utf8(&bytes)
            .map_err(|_| "invalid checkpoint candidate")?
            .trim();
        if !crate::preparation::valid_commit(candidate) {
            return Err("invalid checkpoint candidate".into());
        }
        Ok(candidate.into())
    }

    pub(super) fn verify_export<E: Executor>(
        &self,
        exec: &E,
        operation: &CheckpointOperation,
        deadline: Instant,
    ) -> Result<(), String> {
        let mut script = self.prefix();
        script.push_str(&format!(r#"
/usr/bin/chown -R 0:{gid} "$work/export"
/usr/bin/chmod -R a-w "$work/export"
actual=$(/usr/bin/git -c core.hooksPath=/dev/null --git-dir="$work/export/.git" rev-parse --verify 'HEAD^{{commit}}')
[ "$actual" = {candidate} ]
export GIT_INDEX_FILE="$work/export/.git/index"
[ "$(git_private write-tree)" = {tree} ]
git_export() {{ /usr/bin/git -c core.hooksPath=/dev/null -c core.fsmonitor=false --git-dir="$work/repo.git" --work-tree="$work/export" "$@"; }}
git_export update-index --refresh >/dev/null
git_export diff-files --quiet --no-ext-diff --no-textconv --
"#, gid=self.gid, candidate=shell_quote(&operation.candidate), tree=shell_quote(&operation.tree)));
        self.exec(exec, &script, deadline).map(|_| ())
    }

    pub(super) fn commit<E: Executor>(
        &self,
        exec: &E,
        operation: &CheckpointOperation,
        deadline: Instant,
    ) -> Result<(), String> {
        let mut script = self.prefix();
        script.push_str(&format!(
            r#"
candidate={candidate}
old={old}
if [ "$head" = "$candidate" ]; then
  [ "$index_tree" = {parent_tree} ] || [ "$index_tree" = {tree} ]
else
  [ "$head" = "$old" ] && [ "$index_tree" = {parent_tree} ]
  /usr/bin/git -c core.hooksPath=/dev/null --git-dir="$src/.git" update-ref HEAD "$candidate" "$old"
fi
GIT_INDEX_FILE="$src/.git/index" git_private read-tree "$candidate"
/usr/bin/chown 0:{gid} "$src/.git/index"
/usr/bin/chmod 640 "$src/.git/index"
/usr/bin/sync -f "$src/.git/index"
/usr/bin/sync -f "$src/.git"
"#,
            candidate = shell_quote(&operation.candidate),
            old = shell_quote(&operation.request.expected_head),
            parent_tree = shell_quote(&operation.parent_tree),
            tree = shell_quote(&operation.tree),
            gid = self.gid
        ));
        self.exec(exec, &script, deadline).map(|_| ())
    }

    pub(super) fn commit_binding_current<E: Executor>(
        &self,
        exec: &E,
        operation: &CheckpointOperation,
        deadline: Instant,
    ) -> Result<bool, String> {
        let mut script = self.prefix();
        script.push_str(&format!(r#"
if [ "$head" = {candidate} ]; then
  if [ "$index_tree" = {parent_tree} ] || [ "$index_tree" = {tree} ]; then printf 'current\n'; else printf 'conflict\n'; fi
elif [ "$head" = {old} ] && [ "$index_tree" = {parent_tree} ]; then printf 'current\n'; else printf 'conflict\n'; fi
"#, candidate=shell_quote(&operation.candidate), old=shell_quote(&operation.request.expected_head), parent_tree=shell_quote(&operation.parent_tree), tree=shell_quote(&operation.tree)));
        let bytes = self.exec(exec, &script, deadline)?;
        match bytes.as_slice() {
            b"current\n" => Ok(true),
            b"conflict\n" => Ok(false),
            _ => Err("invalid checkpoint compare-and-swap observation".into()),
        }
    }

    pub(super) fn inspect_final<E: Executor>(
        &self,
        exec: &E,
        expected: &str,
        commits: &[String],
        deadline: Instant,
    ) -> Result<String, String> {
        let mut script = self.prefix();
        script.push_str(&format!(
            "[ \"$head\" = {} ]\n[ \"$index_tree\" = \"$parent_tree\" ]\n",
            shell_quote(expected)
        ));
        for commit in commits {
            script.push_str(&format!(
                "git_private merge-base --is-ancestor {} \"$head\"\n",
                shell_quote(commit)
            ));
        }
        script.push_str("export GIT_INDEX_FILE=\"$work/final.index\"\ngit_private read-tree \"$head\"\ngit_private update-index --refresh >/dev/null\ngit_private diff-files --quiet --no-ext-diff --no-textconv --\n[ -z \"$(git_private ls-files --others --exclude-standard -- ':!/.soda-home' ':!/.soda-home/**')\" ]\nprintf '%s\\n' \"$parent_tree\"\n");
        let bytes = self.exec(exec, &script, deadline)?;
        let tree = std::str::from_utf8(&bytes)
            .map_err(|_| "invalid final candidate tree")?
            .trim();
        if !crate::preparation::valid_commit(tree) {
            return Err("invalid final candidate tree".into());
        }
        Ok(tree.into())
    }
}

const CLEAN_ENV: &str = "export PATH=/usr/bin:/bin LC_ALL=C GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_NO_REPLACE_OBJECTS=1 GIT_TERMINAL_PROMPT=0 GIT_OPTIONAL_LOCKS=0\nunset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_CONFIG_COUNT\n";
