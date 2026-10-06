use super::Executor;
use crate::domain;
use std::time::{Duration, Instant};

/// `path/filepath.Dir` for socket paths.
fn go_dir(path: &str) -> &str {
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() {
        return "/";
    }
    match trimmed.rfind('/') {
        None => ".",
        Some(0) => "/",
        Some(i) => {
            let dir = trimmed[..i].trim_end_matches('/');
            if dir.is_empty() {
                "/"
            } else {
                dir
            }
        }
    }
}

impl<E: Executor> super::Runtime<E> {
    /// `Create`: provision and start a project environment. Callers hold
    /// the mutation gate before invoking this method.
    pub fn create(
        &self,
        input: &domain::Create,
        deadline: Instant,
    ) -> Result<domain::Environment, String> {
        input.validate()?;
        self.create_container(input, deadline)?;
        self.start_created(input, deadline)
    }

    fn create_container(&self, input: &domain::Create, deadline: Instant) -> Result<(), String> {
        let profile = self.resolve_profile(deadline)?;
        let want = input
            .profile
            .as_ref()
            .ok_or_else(|| "invalid creation identity".to_string())?;
        if profile != *want {
            return Err("installed profile changed; reservation retained".to_string());
        }
        let encoded = profile.encode();
        if self
            .podman(&[], &["network", "exists", &self.config.network], deadline)
            .is_err()
        {
            self.podman(
                &[],
                &[
                    "network",
                    "create",
                    "--driver",
                    "bridge",
                    "--subnet",
                    &self.config.subnet,
                    "--interface-name",
                    &self.config.bridge,
                    &self.config.network,
                ],
                deadline,
            )?;
        }
        let name = format!("soda-{}", input.id);
        let mut args: Vec<String> = vec![
            "create".to_string(),
            "--name".to_string(),
            name,
            "--label".to_string(),
            format!("org.soda.project={}", input.id),
            "--label".to_string(),
            format!("org.soda.owner={}", input.owner),
            "--network".to_string(),
            self.config.network.clone(),
            "--userns=auto:size=262144".to_string(),
            "--systemd=always".to_string(),
            "--cgroupns=private".to_string(),
            "--cap-add=SYS_ADMIN,MKNOD,NET_ADMIN,SYS_PTRACE".to_string(),
            "--device=/dev/fuse".to_string(),
            "--security-opt=label=disable".to_string(),
            "--label".to_string(),
            format!("org.soda.profile={}", profile.id),
            "--label".to_string(),
            format!("org.soda.creation-profile={encoded}"),
            "--pull=never".to_string(),
        ];
        if !self.config.muse_socket.is_empty() {
            args.push("--volume".to_string());
            args.push(format!(
                "{}:/run/soda-muse-interface:ro",
                go_dir(&self.config.muse_socket)
            ));
        }
        args.push(profile.image.clone());
        let refs: Vec<&str> = args.iter().map(String::as_str).collect();
        self.podman(&[], &refs, deadline)?;
        Ok(())
    }

    fn start_created(
        &self,
        input: &domain::Create,
        deadline: Instant,
    ) -> Result<domain::Environment, String> {
        let name = format!("soda-{}", input.id);
        let unit = format!("soda-project@{}.service", input.id);
        self.exec.run(
            &[],
            "/usr/bin/systemctl",
            &["enable", "--now", &unit],
            deadline,
        )?;
        self.wait_project_ready(&name, deadline)?;
        let (env, _) = self.inspect(&input.id, deadline)?;
        let want = input
            .profile
            .as_ref()
            .ok_or_else(|| "invalid creation identity".to_string())?;
        match &env.profile {
            Some(got) if env.running && !env.ip.is_empty() && got == want => Ok(env),
            _ => {
                Err("project did not report the expected profile and running endpoint".to_string())
            }
        }
    }

    fn wait_project_ready(&self, name: &str, deadline: Instant) -> Result<(), String> {
        loop {
            if self
                .podman(
                    &[],
                    &[
                        "exec",
                        name,
                        "/usr/bin/test",
                        "-f",
                        "/run/soda-project-ready",
                    ],
                    deadline,
                )
                .is_ok()
            {
                return Ok(());
            }
            if Instant::now() >= deadline {
                return Err("deadline exceeded".to_string());
            }
            std::thread::sleep(Duration::from_secs(1));
            if Instant::now() >= deadline {
                return Err("deadline exceeded".to_string());
            }
        }
    }
}
