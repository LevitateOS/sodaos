use std::time::Instant;

use super::{
    decode_config_view, LaunchProvider, MuseCaller, MuseHooks, MuseRuntime, ProviderSnapshot,
};
use crate::project::Executor;
use crate::terminal;

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    /// Ensure the RUNNING Project serves this invocation's admitted
    /// snapshot: when the guest binary already matches the snapshot digest
    /// with a corroborating receipt, touch nothing. Otherwise stage the
    /// snapshot's bundle bytes plus its admitted-identity receipt into the
    /// live container (normal and nested paths) and re-verify. Each file
    /// lands via temp file plus atomic rename, so already-running processes
    /// keep their old inode and never observe a truncation: in-flight work
    /// stays bound to its exact old bytes with no restart, while the next
    /// exec sees the admitted bytes. Fails closed on any mismatch.
    pub fn ensure_guest_provider(
        &self,
        provider: LaunchProvider,
        caller: &MuseCaller,
        snapshot: &ProviderSnapshot,
        deadline: Instant,
    ) -> Result<String, String> {
        let release = snapshot.selection.release.digest.clone();
        let digest = super::provider::snapshot_executable_digest(provider, snapshot)?;
        let receipt =
            serde_json::to_vec_pretty(&snapshot.selection).map_err(|_| terminal::err_denied())?;
        if receipt.len() > 65536 {
            return Err(terminal::err_denied());
        }
        if self.guest_provider_current(provider, caller, &digest, &receipt, deadline) {
            return Ok(release);
        }
        if provider == LaunchProvider::Codex {
            // Codex is a bundle directory: never nested, staged whole.
            if !caller.child.is_empty() {
                return Err(terminal::err_denied());
            }
            let bundle = super::provider::snapshot_bundle(snapshot)?;
            self.stage_guest_bundle(caller, provider.guest_native(), &bundle, deadline)?;
            self.stage_guest_file(caller, provider.guest_receipt(), &receipt, "0644", deadline)?;
            self.verify_guest_digest(provider, &caller.container, "", &digest, deadline)?;
            if !self.guest_receipt_current(provider, caller, &receipt, deadline) {
                return Err(terminal::err_denied());
            }
            return Ok(release);
        }
        // Staged under the snapshot's active store reference: cleanup
        // cannot collect the bundle mid-copy.
        let bytes = std::fs::read(&snapshot.executable).map_err(|_| terminal::err_denied())?;
        if bytes.is_empty() || bytes.len() as u64 != snapshot.selection.release.size {
            return Err(terminal::err_denied());
        }
        self.stage_guest_file(caller, provider.guest_receipt(), &receipt, "0644", deadline)?;
        self.stage_guest_file(caller, provider.guest_native(), &bytes, "0755", deadline)?;
        // Re-verify exactly what the execution will run: digest, ELF
        // header, command availability. The receipt must corroborate the
        // staged bytes, matching the in-project enforcement.
        self.verify_guest_digest(
            provider,
            &caller.container,
            &caller.child,
            &digest,
            deadline,
        )?;
        if !self.guest_receipt_current(provider, caller, &receipt, deadline) {
            return Err(terminal::err_denied());
        }
        Ok(release)
    }

    /// Copy one host bundle directory into the running container at `path`:
    /// extract beside it, then swap directories. Running processes keep
    /// their open inodes; the next exec sees the new bundle.
    fn stage_guest_bundle(
        &self,
        caller: &MuseCaller,
        path: &str,
        bundle: &std::path::Path,
        deadline: Instant,
    ) -> Result<(), String> {
        let id = terminal::rand_id().map_err(|_| terminal::err_denied())?;
        let tmp = format!("{path}.soda-restage-{id}");
        let retired = format!("{path}.soda-retired-{id}");
        let bundle = bundle.to_str().ok_or_else(terminal::err_denied)?;
        let staged = self
            .guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/install", "--directory", "--mode=0755", &tmp],
                deadline,
            )
            .and_then(|_| {
                crate::terminal::run_harness_pipeline(
                    &self.exec,
                    "/usr/bin/tar",
                    "/usr/bin/podman",
                    bundle,
                    &caller.container,
                    &tmp,
                    deadline,
                )
            })
            .and_then(|_| {
                // A previous bundle moves aside first; absence is fine.
                let _ = self.guest_refs(
                    &caller.container,
                    &[],
                    &["/usr/bin/mv", "-T", "--", path, &retired],
                    deadline,
                );
                self.guest_refs(
                    &caller.container,
                    &[],
                    &["/usr/bin/mv", "-T", "--", &tmp, path],
                    deadline,
                )
            });
        let _ = self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/rm", "-rf", "--", &tmp, &retired],
            deadline,
        );
        staged.map(|_| ()).map_err(|_| terminal::err_denied())
    }

    /// True when the guest binary hashes to the snapshot digest AND the
    /// guest receipt byte-matches the snapshot receipt.
    fn guest_provider_current(
        &self,
        provider: LaunchProvider,
        caller: &MuseCaller,
        digest: &str,
        receipt: &[u8],
        deadline: Instant,
    ) -> bool {
        let prefix = self.guest_binary_prefix(&caller.child);
        let mut hash_argv = prefix;
        hash_argv.extend([
            "/usr/bin/sha256sum".to_string(),
            provider.guest_executable().to_string(),
        ]);
        let hash = match self.guest(&caller.container, &[], &hash_argv, deadline) {
            Ok(hash) => hash,
            Err(_) => return false,
        };
        let text = String::from_utf8_lossy(&hash);
        let fields: Vec<&str> = text.split_whitespace().collect();
        if fields.len() != 2 || fields[0] != digest {
            return false;
        }
        self.guest_receipt_current(provider, caller, receipt, deadline)
    }

    /// True when the guest receipt byte-matches the snapshot receipt.
    fn guest_receipt_current(
        &self,
        provider: LaunchProvider,
        caller: &MuseCaller,
        receipt: &[u8],
        deadline: Instant,
    ) -> bool {
        let mut cat_argv = self.guest_binary_prefix(&caller.child);
        cat_argv.extend([
            "/usr/bin/cat".to_string(),
            provider.guest_receipt().to_string(),
        ]);
        match self.guest(&caller.container, &[], &cat_argv, deadline) {
            Ok(body) => body == receipt,
            Err(_) => false,
        }
    }

    /// Copy one blob into the running container at `path` via temp file
    /// plus atomic rename. Nested children are reached through the same
    /// parent `podman exec` prefix the verifier uses.
    fn stage_guest_file(
        &self,
        caller: &MuseCaller,
        path: &str,
        bytes: &[u8],
        mode: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let id = terminal::rand_id().map_err(|_| terminal::err_denied())?;
        let tmp = format!("{path}.soda-restage-{id}");
        let prefix = self.inner_exec_prefix(caller);
        let mut dd_argv = prefix.clone();
        dd_argv.extend([
            "/usr/bin/dd".to_string(),
            format!("of={tmp}"),
            "status=none".to_string(),
        ]);
        let staged = self
            .guest(&caller.container, bytes, &dd_argv, deadline)
            .and_then(|_| {
                let mut chmod_argv = prefix.clone();
                chmod_argv.extend(["/usr/bin/chmod".to_string(), mode.to_string(), tmp.clone()]);
                self.guest(&caller.container, &[], &chmod_argv, deadline)
            })
            .and_then(|_| {
                let mut mv_argv = prefix.clone();
                mv_argv.extend([
                    "/usr/bin/mv".to_string(),
                    "-T".to_string(),
                    "--".to_string(),
                    tmp.clone(),
                    path.to_string(),
                ]);
                self.guest(&caller.container, &[], &mv_argv, deadline)
            });
        if staged.is_err() {
            let mut rm_argv = prefix;
            rm_argv.extend([
                "/usr/bin/rm".to_string(),
                "-f".to_string(),
                "--".to_string(),
                tmp,
            ]);
            let _ = self.guest(&caller.container, &[], &rm_argv, deadline);
            return Err(terminal::err_denied());
        }
        Ok(())
    }

    /// Command prefix that executes inside the target namespace: empty for
    /// the normal project-namespace path, parent `podman exec` (with stdin
    /// forwarding) for nested children.
    fn inner_exec_prefix(&self, caller: &MuseCaller) -> Vec<String> {
        let mut prefix = Vec::new();
        if !caller.child.is_empty() {
            prefix.extend([
                "/usr/bin/podman".to_string(),
                "--remote=false".to_string(),
                "exec".to_string(),
                "--interactive".to_string(),
                caller.child.clone(),
            ]);
        }
        prefix
    }
    /// `MuseRuntime.stage`: credential directory plus config population.
    pub fn stage(
        &self,
        provider: LaunchProvider,
        caller: &MuseCaller,
        path: &str,
        config_home: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/install", "--directory", "--mode=0700", path],
            deadline,
        )?;
        if caller.child.is_empty() {
            self.guest_refs(
                &caller.container,
                &[],
                &[
                    "/usr/bin/mount",
                    "--types=tmpfs",
                    provider.root_tmpfs(),
                    "tmpfs",
                    path,
                ],
                deadline,
            )?;
        }
        if provider == LaunchProvider::Codex {
            // The launcher links the account's ~/.codex entries in itself;
            // only the private credential lives here.
            return self.stage_codex_home(caller, path, deadline);
        }
        self.stage_files(caller, path, deadline)?;
        self.stage_config(caller, path, config_home, deadline)
    }

    /// Writable private `CODEX_HOME` holding a placeholder credential the
    /// delivery overwrites in place (keeping account ownership).
    fn stage_codex_home(
        &self,
        caller: &MuseCaller,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let dir = format!("{path}/codex");
        self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/install",
                "--directory",
                "--mode=0700",
                &format!("--owner={}", caller.uid),
                &format!("--group={}", caller.gid),
                &dir,
            ],
            deadline,
        )?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0711", path],
            deadline,
        )?;
        let auth = LaunchProvider::Codex.credential_file(path);
        self.guest(
            &caller.container,
            b"{}",
            &[
                "/usr/bin/dd".to_string(),
                format!("of={auth}"),
                "status=none".to_string(),
            ],
            deadline,
        )?;
        let ownership = format!("{}:{}", caller.uid, caller.gid);
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chown", &ownership, &auth],
            deadline,
        )?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0600", &auth],
            deadline,
        )
        .map(|_| ())
    }

    fn stage_files(
        &self,
        caller: &MuseCaller,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let dir = format!("{path}/config/muse");
        self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/install",
                "--directory",
                "--mode=0700",
                &format!("--owner={}", caller.uid),
                &format!("--group={}", caller.gid),
                &dir,
            ],
            deadline,
        )?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0711", path],
            deadline,
        )?;
        let auth = format!("{path}/auth.json");
        self.guest(
            &caller.container,
            b"{}",
            &[
                "/usr/bin/dd".to_string(),
                format!("of={auth}"),
                "status=none".to_string(),
            ],
            deadline,
        )?;
        let ownership = format!("{}:{}", caller.uid, caller.gid);
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chown", &ownership, &auth],
            deadline,
        )?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0600", &auth],
            deadline,
        )
        .map(|_| ())
    }

    fn stage_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        config_home: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let source = if config_home.is_empty() {
            format!("{}/.config", caller.home)
        } else {
            config_home.to_string()
        };
        self.populate_config(caller, path, &source, deadline)?;
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/chmod", "0711", &format!("{path}/config")],
            deadline,
        )?;
        self.auth_mount_target(caller, path, deadline)
    }

    fn populate_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        source: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if !caller.child.is_empty() {
            return self.copy_nested_config(caller, path, source, deadline);
        }
        self.podman(
            &[],
            &[
                "exec".to_string(),
                format!("--user={}:{}", caller.uid, caller.gid),
                caller.container.clone(),
                "/usr/local/bin/muse".to_string(),
                "--soda-copy-config".to_string(),
                format!("{source}/muse"),
                format!("{path}/config/muse"),
            ],
            deadline,
        )
        .map(|_| ())
    }

    fn auth_mount_target(
        &self,
        caller: &MuseCaller,
        path: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        if !caller.child.is_empty() {
            return self
                .guest_refs(
                    &caller.container,
                    &[],
                    &[
                        "/usr/bin/ln",
                        "--symbolic",
                        "../../auth.json",
                        &format!("{path}/config/muse/auth.json"),
                    ],
                    deadline,
                )
                .map(|_| ());
        }
        self.guest_refs(
            &caller.container,
            &[],
            &["/usr/bin/touch", &format!("{path}/config/muse/auth.json")],
            deadline,
        )
        .map(|_| ())
    }

    fn copy_nested_config(
        &self,
        caller: &MuseCaller,
        path: &str,
        source: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let body = self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/nsenter",
                &format!("--target={}", caller.nested_pid),
                "--mount",
                "--pid",
                "--uts",
                "--ipc",
                "--net",
                "--root",
                &format!("--setuid={}", caller.uid),
                &format!("--setgid={}", caller.gid),
                "--",
                "/usr/local/bin/muse",
                "--soda-read-config",
                &format!("{source}/muse"),
            ],
            deadline,
        )?;
        if body.len() > 3 << 20 {
            return Err(terminal::err_denied());
        }
        let mut view = decode_config_view(&body)?;
        for name in ["settings.json", "trust.json"] {
            let Some(value) = view.remove(name) else {
                continue;
            };
            let target = format!("{path}/config/muse/{name}");
            let dd = [
                "/usr/bin/dd".to_string(),
                format!("of={target}"),
                "status=none".to_string(),
            ];
            self.guest(&caller.container, &value, &dd, deadline)?;
            let mut value = value;
            for byte in value.iter_mut() {
                *byte = 0;
            }
            let ownership = format!("{}:{}", caller.uid, caller.gid);
            self.guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/chown", &ownership, &target],
                deadline,
            )?;
            self.guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/chmod", "0600", &target],
                deadline,
            )?;
        }
        Ok(())
    }
}
