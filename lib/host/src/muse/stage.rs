use std::time::Instant;

use super::{decode_config_view, MuseCaller, MuseHooks, MuseRuntime, ProviderSnapshot};
use crate::project::Executor;
use crate::terminal;

/// Guest provider installation paths, fixed by the installed project
/// interface (same destinations the maintain staging pipe installs).
const GUEST_NATIVE: &str = "/usr/local/libexec/soda/muse";
const GUEST_RECEIPT: &str = "/usr/local/libexec/soda/muse-selection.json";

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
        caller: &MuseCaller,
        snapshot: &ProviderSnapshot,
        deadline: Instant,
    ) -> Result<String, String> {
        let digest = snapshot.selection.release.digest.clone();
        let receipt =
            serde_json::to_vec_pretty(&snapshot.selection).map_err(|_| terminal::err_denied())?;
        if receipt.len() > 65536 {
            return Err(terminal::err_denied());
        }
        if self.guest_provider_current(caller, &digest, &receipt, deadline) {
            return Ok(digest);
        }
        // Staged under the snapshot's active store reference: cleanup
        // cannot collect the bundle mid-copy.
        let bytes = std::fs::read(&snapshot.executable).map_err(|_| terminal::err_denied())?;
        if bytes.is_empty() || bytes.len() as u64 != snapshot.selection.release.size {
            return Err(terminal::err_denied());
        }
        self.stage_guest_file(caller, GUEST_RECEIPT, &receipt, "0644", deadline)?;
        self.stage_guest_file(caller, GUEST_NATIVE, &bytes, "0755", deadline)?;
        // Re-verify exactly what the execution will run: digest, ELF
        // header, command availability. The receipt must corroborate the
        // staged bytes, matching the in-project enforcement.
        let verified =
            self.verify_guest_digest(&caller.container, &caller.child, &digest, deadline)?;
        if !self.guest_receipt_current(caller, &receipt, deadline) {
            return Err(terminal::err_denied());
        }
        Ok(verified)
    }

    /// True when the guest binary hashes to the snapshot digest AND the
    /// guest receipt byte-matches the snapshot receipt.
    fn guest_provider_current(
        &self,
        caller: &MuseCaller,
        digest: &str,
        receipt: &[u8],
        deadline: Instant,
    ) -> bool {
        let prefix = self.guest_binary_prefix(&caller.child);
        let mut hash_argv = prefix;
        hash_argv.extend(["/usr/bin/sha256sum".to_string(), GUEST_NATIVE.to_string()]);
        let hash = match self.guest(&caller.container, &[], &hash_argv, deadline) {
            Ok(hash) => hash,
            Err(_) => return false,
        };
        let text = String::from_utf8_lossy(&hash);
        let fields: Vec<&str> = text.split_whitespace().collect();
        if fields.len() != 2 || fields[0] != digest {
            return false;
        }
        self.guest_receipt_current(caller, receipt, deadline)
    }

    /// True when the guest receipt byte-matches the snapshot receipt.
    fn guest_receipt_current(
        &self,
        caller: &MuseCaller,
        receipt: &[u8],
        deadline: Instant,
    ) -> bool {
        let mut cat_argv = self.guest_binary_prefix(&caller.child);
        cat_argv.extend(["/usr/bin/cat".to_string(), GUEST_RECEIPT.to_string()]);
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
                    "--options=mode=0700,size=4M",
                    "tmpfs",
                    path,
                ],
                deadline,
            )?;
        }
        self.stage_files(caller, path, deadline)?;
        self.stage_config(caller, path, config_home, deadline)
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
