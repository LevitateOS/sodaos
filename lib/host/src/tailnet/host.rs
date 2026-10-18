//! Exact-owned host Tailscale container. Stop never consults desired intent,
//! a current image cache, or an operational admission grant.

use std::fs;
use std::os::unix::fs::FileTypeExt;
use std::path::PathBuf;
use std::time::{Duration, Instant};

use serde::{Deserialize, Serialize};

use crate::project::Executor;
use crate::tailnet_admission::{Grant, StartGuard};
use crate::tailnet_domain::{valid_container_id, valid_image_id};
use crate::tailnet_files::OwnedState;

pub const HOST_NAME: &str = "soda-tailnet-host";
pub const HOST_SOCKET: &str = "/run/soda-tailnet/host/control/tailscaled.sock";
const INSPECT: &str = "{\"id\":{{json .ID}},\"image\":{{json .Image}},\"command\":{{json .Config.CreateCommand}},\"running\":{{json .State.Running}},\"pid\":{{json .State.Pid}},\"role\":{{json (index .Config.Labels \"org.soda.tailnet.role\")}},\"installation\":{{json (index .Config.Labels \"org.soda.tailnet.installation\")}}}";

fn unconfirmed() -> String {
    "unavailable: host Tailnet runtime is unconfirmed".to_string()
}
fn conflict() -> String {
    "conflict: host Tailnet ownership changed".to_string()
}

pub(crate) fn command(
    exec: &dyn Executor,
    binary: &str,
    args: &[String],
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    if Instant::now() >= deadline {
        return Err(unconfirmed());
    }
    let refs: Vec<&str> = args.iter().map(String::as_str).collect();
    match exec.run_bounded(&[], binary, &refs, deadline, 65536, 0) {
        Ok(data) if data.len() <= 65536 => Ok(data),
        _ => Err(unconfirmed()),
    }
}

pub(crate) fn podman(
    exec: &dyn Executor,
    args: &[String],
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    let mut full = vec!["--remote=false".into()];
    full.extend_from_slice(args);
    command(exec, "/usr/bin/podman", &full, deadline)
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HostRecord {
    pub id: String,
    pub image: String,
    pub installation_id: String,
    pub generation: String,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct Inspect {
    id: String,
    image: String,
    command: Vec<String>,
    running: bool,
    pid: i64,
    role: String,
    installation: String,
}

pub struct HostProcess {
    pub record: HostRecord,
    pub running: bool,
    pub pid: i64,
}

pub struct HostRuntime<E> {
    pub exec: E,
    state: OwnedState,
    control: PathBuf,
}

impl<E: Executor> HostRuntime<E> {
    pub fn new(exec: E) -> Self {
        Self {
            exec,
            state: OwnedState {
                root: PathBuf::from("/var/lib/soda-tailnet"),
                owner: 0,
            },
            control: PathBuf::from("/run/soda-tailnet/host/control"),
        }
    }

    fn recipe(&self, record: &HostRecord) -> Result<Vec<String>, String> {
        // Reuse the lifecycle UUID validator; unlike a name/label match, all
        // immutable daemon arguments and the locally recorded image must agree.
        if !valid_image_id(&record.image)
            || !crate::tcontrol_wire::valid_lifecycle_uuid(&record.installation_id)
            || record.generation == "0"
            || !crate::tcontrol_wire::valid_lifecycle_generation(&record.generation)
        {
            return Err(conflict());
        }
        Ok(vec![
            "--remote=false".into(),
            "create".into(),
            "--name".into(),
            HOST_NAME.into(),
            "--label".into(),
            "org.soda.tailnet.role=host".into(),
            "--label".into(),
            format!("org.soda.tailnet.installation={}", record.installation_id),
            "--userns=host".into(),
            "--network=host".into(),
            "--pid=private".into(),
            "--ipc=private".into(),
            "--uts=private".into(),
            "--cgroupns=private".into(),
            "--user=0:0".into(),
            "--cap-drop=ALL".into(),
            "--cap-add=NET_ADMIN".into(),
            "--device=/dev/net/tun".into(),
            "--security-opt=label=disable".into(),
            "--no-hosts".into(),
            "--log-driver=none".into(),
            "--pull=never".into(),
            "--volume".into(),
            format!(
                "{}:/var/lib/tailscale:rw",
                self.state.root.join("host").display()
            ),
            "--volume".into(),
            format!("{}:/run/tailscale:rw", self.control.display()),
            "--entrypoint=/usr/local/bin/tailscaled".into(),
            record.image.clone(),
            "--state=/var/lib/tailscale/tailscaled.state".into(),
            "--socket=/run/tailscale/tailscaled.sock".into(),
            "--tun=tailscale0".into(),
            "--no-logs-no-support".into(),
        ])
    }

    fn ids(&self, filter: &str, deadline: Instant) -> Result<Vec<String>, String> {
        let data = podman(
            &self.exec,
            &[
                "ps".into(),
                "-a".into(),
                "--no-trunc".into(),
                "--format={{.ID}}".into(),
                "--filter".into(),
                filter.into(),
            ],
            deadline,
        )?;
        let text = std::str::from_utf8(&data).map_err(|_| unconfirmed())?;
        let ids: Vec<String> = text.split_whitespace().map(String::from).collect();
        if ids.len() > 1 || ids.iter().any(|id| !valid_container_id(id)) {
            return Err(conflict());
        }
        Ok(ids)
    }

    fn admitted_record(&self, named: &[String]) -> Result<Option<HostRecord>, String> {
        if !self.state.check()? {
            return if named.is_empty() {
                Ok(None)
            } else {
                Err(conflict())
            };
        }
        match self.state.read("host-runtime.json") {
            Ok(record) => Ok(Some(record)),
            Err(error) => match fs::symlink_metadata(self.state.root.join("host-runtime.json")) {
                Err(e) if e.kind() == std::io::ErrorKind::NotFound && named.is_empty() => Ok(None),
                _ => Err(error),
            },
        }
    }

    fn inspected_actual(
        &self,
        record: &HostRecord,
        expected: &[String],
        deadline: Instant,
    ) -> Result<Inspect, String> {
        let data = podman(
            &self.exec,
            &[
                "inspect".into(),
                "--format".into(),
                INSPECT.into(),
                record.id.clone(),
            ],
            deadline,
        )?;
        let actual: Inspect = crate::json::decode_strict_as(&data).map_err(|_| unconfirmed())?;
        if actual.id != record.id
            || actual.image.trim_start_matches("sha256:")
                != record.image.trim_start_matches("sha256:")
            || actual.role != "host"
            || actual.installation != record.installation_id
            || !crate::tailnet_companion::companion_command_matches(&actual.command, expected)
        {
            return Err(conflict());
        }
        Ok(actual)
    }

    fn confirmed_namespaces(actual: &Inspect) -> Result<(), String> {
        if actual.running {
            if actual.pid <= 1 {
                return Err(conflict());
            }
            for namespace in ["net", "user"] {
                let host = fs::read_link(format!("/proc/self/ns/{namespace}"))
                    .map_err(|_| unconfirmed())?;
                let daemon = fs::read_link(format!("/proc/{}/ns/{namespace}", actual.pid))
                    .map_err(|_| unconfirmed())?;
                if host != daemon {
                    return Err(conflict());
                }
            }
        }
        Ok(())
    }

    pub fn inspect(
        &self,
        installation: Option<&str>,
        deadline: Instant,
    ) -> Result<Option<HostProcess>, String> {
        let named = self.ids("name=^soda-tailnet-host$", deadline)?;
        let Some(record) = self.admitted_record(&named)? else {
            return Ok(None);
        };
        if !valid_container_id(&record.id)
            || installation.is_some_and(|id| id != record.installation_id)
        {
            return Err(conflict());
        }
        let expected = self.recipe(&record)?;
        let ids = self.ids(&format!("id={}", record.id), deadline)?;
        if ids.is_empty() {
            if !named.is_empty() {
                return Err(conflict());
            }
            return Ok(None);
        }
        if named.as_slice() != [record.id.clone()] {
            return Err(conflict());
        }
        let actual = self.inspected_actual(&record, &expected, deadline)?;
        Self::confirmed_namespaces(&actual)?;
        Ok(Some(HostProcess {
            record,
            running: actual.running,
            pid: actual.pid,
        }))
    }

    fn prepare_dirs(&self) -> Result<(), String> {
        self.state.ensure()?;
        for path in [
            self.state.root.join("host"),
            self.control.parent().ok_or_else(unconfirmed)?.to_path_buf(),
            self.control.clone(),
        ] {
            OwnedState {
                root: path,
                owner: 0,
            }
            .ensure()?;
        }
        Ok(())
    }

    fn reuse_running(&self, grant: &Grant, deadline: Instant) -> Result<Option<String>, String> {
        let Some(process) = self.inspect(Some(&grant.installation_id), deadline)? else {
            return Ok(None);
        };
        if process.running && process.record.image == grant.image {
            self.ready(deadline)?;
            return Ok(Some(process.record.id));
        }
        self.stop(Some(&grant.installation_id), deadline)?;
        Ok(None)
    }

    fn launch_record(&self, grant: &Grant, deadline: Instant) -> Result<HostRecord, String> {
        if fs::symlink_metadata("/sys/class/net/tailscale0").is_ok() {
            return Err(conflict());
        }
        self.prepare_dirs()?;
        let mut record = HostRecord {
            id: String::new(),
            image: grant.image.clone(),
            installation_id: grant.installation_id.clone(),
            generation: grant.generation.clone(),
        };
        let args = self.recipe(&record)?;
        let data = command(&self.exec, "/usr/bin/podman", &args, deadline)?;
        record.id = std::str::from_utf8(&data)
            .map_err(|_| unconfirmed())?
            .trim()
            .to_string();
        if !valid_container_id(&record.id) {
            return Err(unconfirmed());
        }
        self.state.write("host-runtime.json", &record)?;
        Ok(record)
    }

    fn confirm_launched(
        &self,
        grant: &Grant,
        record: HostRecord,
        deadline: Instant,
    ) -> Result<String, String> {
        let created = self
            .inspect(Some(&grant.installation_id), deadline)?
            .ok_or_else(unconfirmed)?;
        if created.record.id != record.id {
            return Err(conflict());
        }
        podman(&self.exec, &["start".into(), record.id.clone()], deadline)?;
        self.ready(deadline)?;
        self.inspect(Some(&grant.installation_id), deadline)?
            .filter(|p| p.running)
            .ok_or_else(unconfirmed)?;
        Ok(record.id)
    }

    pub fn ensure(
        &self,
        grant: &Grant,
        _starts: &StartGuard,
        deadline: Instant,
    ) -> Result<String, String> {
        if let Some(id) = self.reuse_running(grant, deadline)? {
            return Ok(id);
        }
        let record = self.launch_record(grant, deadline)?;
        self.confirm_launched(grant, record, deadline)
    }

    fn ready(&self, deadline: Instant) -> Result<(), String> {
        let end = deadline.min(Instant::now() + Duration::from_secs(10));
        loop {
            if crate::tcontrol_native::local_request(
                &self.control.join("tailscaled.sock"),
                "GET",
                "status",
                None,
                end,
            )
            .is_ok_and(|(status, data)| {
                status == 200
                    && serde_json::from_slice::<serde_json::Value>(&data)
                        .is_ok_and(|v| v.get("BackendState").and_then(|s| s.as_str()).is_some())
            }) {
                return Ok(());
            }
            if Instant::now() >= end {
                return Err(unconfirmed());
            }
            std::thread::sleep(
                Duration::from_millis(50).min(end.saturating_duration_since(Instant::now())),
            );
        }
    }

    fn stop_running(&self, installation: Option<&str>, deadline: Instant) -> Result<(), String> {
        if let Some(process) = self.inspect(installation, deadline)? {
            if process.running {
                podman(
                    &self.exec,
                    &["stop".into(), "--time=10".into(), process.record.id.clone()],
                    deadline,
                )?;
            }
            if self
                .inspect(installation, deadline)?
                .is_some_and(|p| p.running)
            {
                return Err(unconfirmed());
            }
            if podman(&self.exec, &["rm".into(), process.record.id], deadline).is_err()
                && self.inspect(installation, deadline)?.is_some()
            {
                return Err(unconfirmed());
            }
        }
        if self.inspect(installation, deadline)?.is_some() {
            return Err(unconfirmed());
        }
        Ok(())
    }

    fn cleanup_control(&self) -> Result<(), String> {
        // A missing record never authorizes unlinking through unsafe parents.
        for path in [
            self.control.parent().ok_or_else(unconfirmed)?.to_path_buf(),
            self.control.clone(),
        ] {
            if !(OwnedState {
                root: path,
                owner: 0,
            })
            .check()?
            {
                return Ok(());
            }
        }
        let socket = self.control.join("tailscaled.sock");
        match fs::symlink_metadata(&socket) {
            Ok(info) if info.file_type().is_socket() => {
                fs::remove_file(socket).map_err(|_| unconfirmed())?
            }
            Ok(_) => return Err(conflict()),
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
            Err(_) => return Err(unconfirmed()),
        }
        Ok(()) // Preserve host credentials, policy, ownership receipt and image cache.
    }

    pub fn stop(&self, installation: Option<&str>, deadline: Instant) -> Result<(), String> {
        self.stop_running(installation, deadline)?;
        self.cleanup_control()
    }

    pub fn wait(&self, cid: &str, deadline: Instant) -> Result<(), String> {
        if !valid_container_id(cid) {
            return Err(conflict());
        }
        let output = podman(&self.exec, &["wait".into(), cid.into()], deadline)?;
        if output.strip_suffix(b"\n").unwrap_or(&output) != b"0" {
            return Err(unconfirmed());
        }
        Ok(())
    }

    pub fn cli(&self, args: &[String], deadline: Instant) -> Result<Vec<u8>, String> {
        let process = self
            .inspect(None, deadline)?
            .filter(|p| p.running)
            .ok_or_else(unconfirmed)?;
        let mut full = vec![
            "exec".into(),
            "--user=0:0".into(),
            process.record.id,
            "/usr/local/bin/tailscale".into(),
            "--socket=/run/tailscale/tailscaled.sock".into(),
        ];
        full.extend_from_slice(args);
        podman(&self.exec, &full, deadline)
    }
}

#[cfg(test)]
#[path = "host/tests.rs"]
mod tests;
