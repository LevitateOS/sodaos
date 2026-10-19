use super::wire::{hex, Observation, Request, Spec};
use crate::project::Executor;
use serde::Deserialize;
use std::collections::BTreeMap;
use std::fs;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

fn canonical_image(value: &str) -> Result<String, String> {
    let digest = value.strip_prefix("sha256:").unwrap_or(value);
    if !hex(digest, 64) {
        return Err("invalid native application image identity".into());
    }
    Ok(format!("sha256:{digest}"))
}

const ROOT: &str = "/home/soda-applications";
const CONTAINER_FORMAT: &str = "{\"container\":{{json .ID}},\"image\":{{json .Image}},\"running\":{{json .State.Running}},\"labels\":{{json .Config.Labels}}}";
const IMAGE_FORMAT: &str =
    "{\"image\":{{json .ID}},\"user\":{{json .Config.User}},\"labels\":{{json .Config.Labels}}}";

#[derive(Deserialize)]
struct NativeContainer {
    container: String,
    image: String,
    running: bool,
    labels: BTreeMap<String, String>,
}

#[derive(Deserialize)]
struct NativeImage {
    image: String,
    user: String,
    labels: BTreeMap<String, String>,
}

fn same_storage(current: Option<&String>, replacement: Option<&String>) -> bool {
    current.is_some_and(|digest| hex(digest, 64)) && current == replacement
}

pub struct Runtime<E> {
    pub exec: E,
    pub release: Option<ReleaseBinding>,
}

/// Mechanical references from the immutable appliance catalog, or explicit fixtures.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ReleaseBinding {
    pub image: String,
    pub tools_sha256: String,
}

impl ReleaseBinding {
    pub fn validate(&self) -> Result<(), String> {
        if canonical_image(&self.image)? != self.image || !hex(&self.tools_sha256, 64) {
            return Err("invalid selected Canlang runtime references".into());
        }
        Ok(())
    }
}

impl<E: Executor> Runtime<E> {
    fn command(&self, cmd: &str, args: &[String], deadline: Instant) -> Result<Vec<u8>, String> {
        let args: Vec<&str> = args.iter().map(String::as_str).collect();
        self.exec
            .run_bounded(&[], cmd, &args, deadline, 65536, 4096)
            .map_err(|_| "native application operation failed; inspect operator journal".into())
    }

    fn systemd(&self, action: &str, id: &str, deadline: Instant) -> Result<(), String> {
        self.command(
            "systemctl",
            &[action.into(), format!("soda-canlang@{id}.service")],
            deadline,
        )
        .map(|_| ())
    }

    fn inspect(
        &self,
        spec: &Spec,
        expected: &str,
        deadline: Instant,
    ) -> Result<NativeContainer, String> {
        let bytes = self.command(
            "podman",
            &[
                "container".into(),
                "inspect".into(),
                "--format".into(),
                CONTAINER_FORMAT.into(),
                format!("soda-canlang-{}", spec.id),
            ],
            deadline,
        )?;
        let mut native: NativeContainer =
            serde_json::from_slice(&bytes).map_err(|_| "invalid application observation")?;
        native.image = canonical_image(&native.image)?;
        if !hex(&native.container, 64)
            || (!expected.is_empty() && native.container != expected)
            || native.labels.get("io.soda.canlang.id") != Some(&spec.id)
            || native.labels.get("io.soda.canlang.data") != Some(&spec.data)
            || native.labels.get("io.soda.canlang.owner") != Some(&spec.owner_team)
            || native.labels.get("io.soda.canlang.origin") != Some(&spec.origin)
            || native.labels.get("io.soda.canlang.port") != Some(&spec.port.to_string())
        {
            return Err("application container generation or identity differs".into());
        }
        Ok(native)
    }

    fn admit_image(&self, spec: &Spec, deadline: Instant) -> Result<NativeImage, String> {
        let bytes = self.command(
            "podman",
            &[
                "image".into(),
                "inspect".into(),
                "--format".into(),
                IMAGE_FORMAT.into(),
                spec.image.clone(),
            ],
            deadline,
        )?;
        let mut image: NativeImage =
            serde_json::from_slice(&bytes).map_err(|_| "invalid application image observation")?;
        image.image = canonical_image(&image.image)?;
        let expected = [
            ("io.canlang.app", &spec.name),
            ("io.canlang.artifact", &spec.artifact_sha256),
            ("io.canlang.bundle", &spec.bundle_sha256),
            ("io.canlang.tools", &spec.tools_sha256),
            ("io.soda.canlang.runtime", &spec.runtime_image),
            ("io.canlang.origin", &spec.origin),
        ];
        if image.image != spec.image
            || image.user != "10000:10000"
            || expected
                .iter()
                .any(|(key, value)| image.labels.get(*key) != Some(*value))
        {
            return Err("application image differs from admitted Canlang references".into());
        }
        Ok(image)
    }

    fn create(&self, spec: &Spec, deadline: Instant) -> Result<String, String> {
        self.admit_image(spec, deadline)?;
        let data = data_directory(&spec.data)?;
        let args = vec![
            "create".into(),
            "--pull=never".into(),
            "--read-only".into(),
            "--cap-drop=all".into(),
            "--security-opt=no-new-privileges".into(),
            "--pids-limit=128".into(),
            "--memory=1g".into(),
            "--cpus=2".into(),
            "--network=systemd-soda".into(),
            format!("--name=soda-canlang-{}", spec.id),
            format!("--publish=127.0.0.1:{}:8080", spec.port),
            format!("--volume={}:/home/app-state:rw,Z", data.display()),
            format!("--label=io.soda.canlang.id={}", spec.id),
            format!("--label=io.soda.canlang.data={}", spec.data),
            format!("--label=io.soda.canlang.owner={}", spec.owner_team),
            format!("--label=io.soda.canlang.origin={}", spec.origin),
            format!("--label=io.soda.canlang.port={}", spec.port),
            format!("--label=io.soda.canlang.revision={}", spec.revision),
            spec.image.clone(),
            "--owner".into(),
            spec.owner_team.clone(),
        ];
        let output = self.command("podman", &args, deadline)?;
        let container = std::str::from_utf8(&output)
            .map_err(|_| "invalid application creation")?
            .trim();
        if !hex(container, 64) {
            return Err("application creation was not confirmed".into());
        }
        Ok(container.into())
    }

    fn ready(&self, spec: &Spec, deadline: Instant) -> bool {
        let probe_deadline = deadline.min(Instant::now() + Duration::from_secs(2));
        self.command(
            "curl",
            &[
                "--silent".into(),
                "--output".into(),
                "/dev/null".into(),
                "--write-out".into(),
                "%{http_code}".into(),
                "--noproxy".into(),
                "*".into(),
                "--max-time".into(),
                "1".into(),
                format!("http://127.0.0.1:{}/auth/login", spec.port),
            ],
            probe_deadline,
        )
        .is_ok_and(|bytes| bytes == b"200")
    }

    pub(super) fn activate_ingress(&self, deadline: Instant) -> Result<(), String> {
        self.command(
            "podman",
            &[
                "exec".into(),
                "soda-proxy".into(),
                "caddy".into(),
                "validate".into(),
                "--config".into(),
                "/etc/caddy/Caddyfile".into(),
                "--adapter".into(),
                "caddyfile".into(),
            ],
            deadline,
        )?;
        // Caddy's admin API is disabled. Only explicit installation restarts
        // shared ingress; application start/stop/replacement never does.
        self.command(
            "systemctl",
            &["restart".into(), "soda-proxy.service".into()],
            deadline,
        )
        .map(|_| ())
    }

    pub(super) fn wait_ready(
        &self,
        spec: &Spec,
        expected: &str,
        deadline: Instant,
    ) -> Result<Observation, String> {
        let deadline = deadline.min(Instant::now() + Duration::from_secs(30));
        loop {
            let native = self.inspect(spec, expected, deadline)?;
            if native.image != spec.image {
                return Err("application changed while waiting for readiness".into());
            }
            if native.running && self.ready(spec, deadline) {
                let revision = native
                    .labels
                    .get("io.soda.canlang.revision")
                    .and_then(|value| value.parse().ok())
                    .ok_or("invalid application generation label")?;
                return Ok(Observation {
                    id: spec.id.clone(),
                    revision,
                    container: native.container,
                    image: native.image,
                    running: true,
                    ready: true,
                    origin: spec.origin.clone(),
                });
            }
            let remaining = deadline.saturating_duration_since(Instant::now());
            if remaining.is_zero() {
                return Err("application backend did not become ready before the deadline".into());
            }
            std::thread::sleep(remaining.min(Duration::from_millis(250)));
        }
    }

    fn check_release(&self, request: &Request) -> Result<(), String> {
        if matches!(request.action.as_str(), "install" | "replace") {
            let selected = self
                .release
                .as_ref()
                .ok_or("selected Canlang runtime is unavailable")?;
            selected.validate()?;
            let spec = &request.application;
            if spec.runtime_image != selected.image || spec.tools_sha256 != selected.tools_sha256 {
                return Err(
                    "application runtime or tools differ from the selected appliance release"
                        .into(),
                );
            }
        }
        Ok(())
    }

    fn install(&self, spec: &Spec, deadline: Instant) -> Result<Observation, String> {
        let created = self.create(spec, deadline)?;
        let native = self.inspect(spec, &created, deadline)?;
        if native.running || native.image != spec.image {
            return Err("new application must be installed stopped with its admitted image".into());
        }
        super::ingress::install(spec)?;
        self.activate_ingress(deadline)?;
        self.systemd("disable", &spec.id, deadline)?;
        self.observe(spec, native, deadline)
    }

    fn run_start(
        &self,
        spec: &Spec,
        container: &str,
        deadline: Instant,
    ) -> Result<Observation, String> {
        super::ingress::verify(spec)?;
        self.systemd("enable", &spec.id, deadline)?;
        self.systemd("start", &spec.id, deadline)?;
        self.wait_ready(spec, container, deadline)
    }

    fn run_stop(
        &self,
        spec: &Spec,
        action: &str,
        deadline: Instant,
    ) -> Result<Observation, String> {
        self.systemd("disable", &spec.id, deadline)?;
        self.systemd("stop", &spec.id, deadline)?;
        self.finish(spec, action, deadline)
    }

    fn run_replace(
        &self,
        spec: &Spec,
        native: NativeContainer,
        action: &str,
        deadline: Instant,
    ) -> Result<Observation, String> {
        let changed = native.labels.get("io.canlang.artifact") != Some(&spec.artifact_sha256)
            || native.labels.get("io.canlang.bundle") != Some(&spec.bundle_sha256);
        let current_storage = native.labels.get("io.canlang.storage");
        if changed && !current_storage.is_some_and(|digest| hex(digest, 64)) {
            return Err("application migration is not qualified; replacement refused".into());
        }
        let image = self.admit_image(spec, deadline)?;
        if changed && !same_storage(current_storage, image.labels.get("io.canlang.storage")) {
            return Err("application storage schema changed; migration is not supported".into());
        }
        super::ingress::verify(spec)?;
        self.systemd("disable", &spec.id, deadline)?;
        self.systemd("stop", &spec.id, deadline)?;
        let stopped = self.inspect(spec, &native.container, deadline)?;
        if stopped.running {
            return Err("application writer did not stop".into());
        }
        // No deletion of app data; a failure leaves an honestly stopped
        // app. Retained old image identity is available in the Go ledger.
        self.command("podman", &["rm".into(), native.container], deadline)?;
        self.create(spec, deadline)?;
        self.finish(spec, action, deadline)
    }

    fn finish(&self, spec: &Spec, action: &str, deadline: Instant) -> Result<Observation, String> {
        let observed = self.observe(spec, self.inspect(spec, "", deadline)?, deadline)?;
        if action == "stop" && observed.running {
            return Err("application writer did not stop".into());
        }
        Ok(observed)
    }

    pub fn operate(&self, request: &Request, deadline: Instant) -> Result<Observation, String> {
        request.validate()?;
        let spec = &request.application;
        self.check_release(request)?;
        if request.action == "install" {
            return self.install(spec, deadline);
        }
        let native = self.inspect(spec, &request.expected_container, deadline)?;
        if request.action == "start" && native.image != spec.image {
            return Err("start requires the admitted application image generation".into());
        }
        match request.action.as_str() {
            "start" => self.run_start(spec, &native.container, deadline),
            "stop" => self.run_stop(spec, &request.action, deadline),
            "replace" => self.run_replace(spec, native, &request.action, deadline),
            "inspect" => self.finish(spec, &request.action, deadline),
            _ => unreachable!(),
        }
    }

    fn observe(
        &self,
        spec: &Spec,
        native: NativeContainer,
        deadline: Instant,
    ) -> Result<Observation, String> {
        let revision = native
            .labels
            .get("io.soda.canlang.revision")
            .and_then(|s| s.parse().ok())
            .ok_or("invalid application generation label")?;
        Ok(Observation {
            id: spec.id.clone(),
            revision,
            container: native.container,
            image: native.image,
            running: native.running,
            ready: native.running && self.ready(spec, deadline),
            origin: spec.origin.clone(),
        })
    }
}

fn private_root(path: &Path) -> Result<(), String> {
    let metadata = fs::symlink_metadata(path).map_err(|_| "application storage is unavailable")?;
    if !metadata.is_dir()
        || metadata.file_type().is_symlink()
        || metadata.mode() & 0o077 != 0
        || fs::canonicalize(path).map_err(|_| "application storage is unavailable")? != path
    {
        return Err("application storage must be an existing private canonical directory".into());
    }
    Ok(())
}

pub(super) fn data_directory(data: &str) -> Result<PathBuf, String> {
    if !super::wire::id(data) {
        return Err("invalid application data identity".into());
    }
    let root = super::home_path(Path::new(ROOT))?;
    let root_meta =
        fs::symlink_metadata(&root).map_err(|_| "application storage is unavailable")?;
    if !root_meta.is_dir() || root_meta.uid() != 0 || root_meta.mode() & 0o022 != 0 {
        return Err("application storage root must belong to the appliance operator".into());
    }
    let base = root.join("data");
    private_root(&base)?;
    if fs::symlink_metadata(&base)
        .map_err(|_| "application storage is unavailable")?
        .uid()
        != 0
    {
        return Err("application data parent must belong to the appliance operator".into());
    }
    let path = base.join(data);
    private_root(&path)?;
    if fs::symlink_metadata(&path)
        .map_err(|_| "application storage is unavailable")?
        .uid()
        != 10000
    {
        return Err("application data must belong to the runtime account".into());
    }
    // Open without following links; the Canlang launcher takes the same lock
    // before provisioning or runtime startup.
    let lock = fs::OpenOptions::new()
        .read(true)
        .write(true)
        .custom_flags(libc::O_NOFOLLOW)
        .open(path.join(".writer-lock"))
        .map_err(|_| "application data lock is unavailable")?;
    let metadata = lock
        .metadata()
        .map_err(|_| "application data lock is unavailable")?;
    if !metadata.is_file() || metadata.uid() != 10000 || metadata.mode() & 0o077 != 0 {
        return Err("application data lock is not a regular file".into());
    }
    Ok(path)
}
