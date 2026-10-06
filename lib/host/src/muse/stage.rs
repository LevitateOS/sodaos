use std::time::Instant;

use super::{decode_config_view, MuseCaller, MuseHooks, MuseRuntime};
use crate::project::Executor;
use crate::terminal;

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
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
