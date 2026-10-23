//! Provisional app-owned owner claim. No ordinary application exists until the
//! exact stopped setup generation has produced its bound receipt.
use super::runtime::{data_directory, Runtime};
use super::wire::{hex, id, Observation, Spec};
use crate::project::Executor;
use serde::{Deserialize, Serialize};
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::time::{Instant, SystemTime, UNIX_EPOCH};

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SetupSpec {
    pub id: String,
    pub name: String,
    pub artifact_sha256: String,
    pub bundle_sha256: String,
    pub tools_sha256: String,
    pub runtime_image: String,
    pub image: String,
    pub data: String,
    pub origin: String,
    pub port: u16,
}
impl SetupSpec {
    fn bind(&self, owner: &str) -> Spec {
        Spec {
            id: self.id.clone(),
            revision: 1,
            name: self.name.clone(),
            artifact_sha256: self.artifact_sha256.clone(),
            bundle_sha256: self.bundle_sha256.clone(),
            tools_sha256: self.tools_sha256.clone(),
            runtime_image: self.runtime_image.clone(),
            image: self.image.clone(),
            data: self.data.clone(),
            owner_team: owner.into(),
            origin: self.origin.clone(),
            port: self.port,
        }
    }
    // Only shared byte/address admission uses this fixed value. It never becomes
    // an owner label, runtime argument, persisted Spec or administration grant.
    fn references(&self) -> Spec {
        self.bind("00000000-0000-0000-0000-000000000000")
    }
}
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SetupRecord {
    pub id: String,
    pub application: SetupSpec,
    pub administrator_id: i64,
    pub expires_at: i64,
    pub state: String,
    pub observation: Option<SetupObservation>,
}
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SetupObservation {
    pub container: String,
    pub running: bool,
    pub ready: bool,
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SetupRequest {
    pub setup: SetupRecord,
    pub action: String,
    pub expected_container: String,
}
#[derive(Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
#[allow(non_snake_case)]
pub struct Receipt {
    schema: String,
    operation: String,
    app: String,
    artifactDigest: String,
    bundleDigest: String,
    storageFingerprint: String,
    ownerTeam: String,
    completedAt: i64,
}
#[derive(Serialize)]
pub struct SetupResult {
    operation: String,
    observation: SetupObservation,
    #[serde(skip_serializing_if = "Option::is_none")]
    setup_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    receipt: Option<Receipt>,
    #[serde(skip_serializing_if = "Option::is_none")]
    application: Option<Observation>,
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Token {
    operation: String,
    token: String,
    expires_at: i64,
}
#[derive(Deserialize)]
struct Container {
    container: String,
    image: String,
    running: bool,
    labels: std::collections::BTreeMap<String, String>,
}
fn now() -> Result<i64, String> {
    Ok(SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|_| "setup clock unavailable")?
        .as_secs() as i64)
}
fn private(path: &Path, uid: u32) -> Result<(), String> {
    let m = fs::symlink_metadata(path).map_err(|_| "setup directory unavailable")?;
    if !m.is_dir()
        || m.file_type().is_symlink()
        || m.uid() != uid
        || m.mode() & 0o077 != 0
        || fs::canonicalize(path).map_err(|_| "setup directory unavailable")? != path
    {
        return Err("setup directory is unsafe".into());
    };
    Ok(())
}
fn read<T: serde::de::DeserializeOwned>(path: &Path, uid: u32) -> Result<T, String> {
    let f = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(path)
        .map_err(|_| "setup record unavailable")?;
    let m = f.metadata().map_err(|_| "setup record unavailable")?;
    if !m.is_file() || m.uid() != uid || m.mode() & 0o077 != 0 || m.nlink() != 1 || m.len() > 8192 {
        return Err("setup record is unsafe".into());
    };
    serde_json::from_reader(f.take(8193)).map_err(|_| "setup record invalid".into())
}
fn save<T: Serialize>(path: &Path, value: &T, uid: u32) -> Result<(), String> {
    let mut nonce = [0u8; 12];
    getrandom::fill(&mut nonce).map_err(|_| "setup randomness unavailable")?;
    let temporary = path.with_extension(format!("{}.pending", crate::sha256::hex_lower(&nonce)));
    let result = (|| {
        let mut f = OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW)
            .open(&temporary)
            .map_err(|_| "setup record cannot be staged")?;
        f.write_all(&serde_json::to_vec(value).map_err(|_| "setup record encoding failed")?)
            .map_err(|_| "setup record write failed")?;
        if unsafe { libc::fchown(std::os::fd::AsRawFd::as_raw_fd(&f), uid, uid) } != 0 {
            return Err("setup record ownership failed".into());
        }
        f.sync_all().map_err(|_| "setup record sync failed")?;
        let from = std::ffi::CString::new(temporary.as_os_str().as_encoded_bytes())
            .map_err(|_| "setup staging path invalid")?;
        let to = std::ffi::CString::new(path.as_os_str().as_encoded_bytes())
            .map_err(|_| "setup record path invalid")?;
        // Atomic nonoverwriting publication also preserves a single link if
        // the host exits between publishing and directory synchronization.
        if unsafe {
            libc::renameat2(
                libc::AT_FDCWD,
                from.as_ptr(),
                libc::AT_FDCWD,
                to.as_ptr(),
                libc::RENAME_NOREPLACE,
            )
        } != 0
        {
            return Err("setup record already exists or publication failed".into());
        }
        File::open(path.parent().ok_or("setup record has no parent")?)
            .and_then(|f| f.sync_all())
            .map_err(|_| "setup directory sync failed".into())
    })();
    let _ = fs::remove_file(&temporary);
    result
}

fn directory(path: &Path, uid: u32) -> Result<(), String> {
    match fs::create_dir(path) {
        Ok(()) => {
            fs::set_permissions(path, fs::Permissions::from_mode(0o700))
                .map_err(|_| "setup directory permission failed")?;
            let p = std::ffi::CString::new(path.as_os_str().as_encoded_bytes())
                .map_err(|_| "setup directory invalid")?;
            if unsafe { libc::chown(p.as_ptr(), uid, uid) } != 0 {
                return Err("setup directory ownership failed".into());
            }
        }
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => (),
        Err(_) => return Err("setup directory unavailable".into()),
    };
    private(path, uid)
}
fn paths(s: &SetupRecord, begin: bool) -> Result<(PathBuf, PathBuf), String> {
    let root = super::home_path(Path::new("/home/soda-applications"))?;
    let metadata = fs::symlink_metadata(&root).map_err(|_| "application storage unavailable")?;
    if !metadata.is_dir() || metadata.uid() != 0 || metadata.mode() & 0o022 != 0 {
        return Err("application storage root must belong to the operator".into());
    }

    let setups = root.join("setups");
    directory(&setups, 0)?;
    let journal = setups.join(format!("{}.json", s.id));
    let mut identity = s.clone();
    identity.observation = None;
    if journal.exists() {
        let previous: SetupRecord = read(&journal, 0)?;
        if previous != identity {
            return Err("setup immutable admission changed".into());
        }
    } else {
        if !begin {
            return Err("setup has not been prepared".into());
        };
        save(&journal, &identity, 0)?
    }
    let control = setups.join(&s.id);
    directory(&control, 10000)?;
    Ok((root, control))
}
impl<E: Executor> Runtime<E> {
    fn setup_container(
        &self,
        s: &SetupRecord,
        expected: &str,
        deadline: Instant,
    ) -> Result<Container, String> {
        let bytes=self.command("podman",&["container".into(),"inspect".into(),"--format".into(),"{\"container\":{{json .ID}},\"image\":{{json .Image}},\"running\":{{json .State.Running}},\"labels\":{{json .Config.Labels}}}".into(),format!("soda-canlang-setup-{}",s.id)],deadline)?;
        let mut c: Container =
            serde_json::from_slice(&bytes).map_err(|_| "invalid setup container")?;
        if !c.image.starts_with("sha256:") {
            c.image = format!("sha256:{}", c.image)
        };
        if !hex(&c.container, 64)
            || (!expected.is_empty() && c.container != expected)
            || c.image != s.application.image
            || c.labels.get("io.soda.canlang.setup") != Some(&s.id)
            || c.labels.get("io.soda.canlang.data") != Some(&s.application.data)
            || c.labels.get("io.soda.canlang.origin") != Some(&s.application.origin)
            || c.labels.get("io.soda.canlang.port") != Some(&s.application.port.to_string())
        {
            return Err("setup container generation differs".into());
        };
        Ok(c)
    }
    fn setup_create(
        &self,
        s: &SetupRecord,
        root: &Path,
        control: &Path,
        deadline: Instant,
    ) -> Result<String, String> {
        let a = &s.application;
        let refs = a.references();
        self.admit_image(&refs, deadline)?;
        let selected = self
            .release
            .as_ref()
            .ok_or("selected Canlang runtime unavailable")?;
        selected.validate()?;
        if a.runtime_image != selected.image || a.tools_sha256 != selected.tools_sha256 {
            return Err("setup runtime differs from selected release".into());
        }
        private(&root.join("data"), 0)?;
        let data = root.join("data").join(&a.data);
        if !data.exists() {
            directory(&data, 10000)?;
            let lock = data.join(".writer-lock");
            save(&lock, &serde_json::json!({}), 10000)?
        } else {
            private(&data, 10000)?;
            if fs::read_dir(&data)
                .map_err(|_| "setup data unavailable")?
                .any(|entry| entry.is_ok_and(|e| e.file_name() != ".writer-lock"))
            {
                return Err("new setup requires fresh empty application data".into());
            }
        }
        data_directory(&a.data)?;
        let output = self.command(
            "podman",
            &[
                "create".into(),
                "--pull=never".into(),
                "--read-only".into(),
                "--cap-drop=all".into(),
                "--security-opt=no-new-privileges".into(),
                "--pids-limit=128".into(),
                "--memory=1g".into(),
                "--cpus=2".into(),
                "--network=systemd-soda".into(),
                format!("--name=soda-canlang-setup-{}", s.id),
                format!("--publish=127.0.0.1:{}:8080", a.port),
                format!("--volume={}:/home/app-state:rw,Z", data.display()),
                format!("--volume={}:/home/app-setup:rw,Z", control.display()),
                format!("--label=io.soda.canlang.setup={}", s.id),
                format!("--label=io.soda.canlang.data={}", a.data),
                format!("--label=io.soda.canlang.origin={}", a.origin),
                format!("--label=io.soda.canlang.port={}", a.port),
                "--entrypoint=/opt/canlang/bin/can-soda".into(),
                a.image.clone(),
                "setup".into(),
                "--host".into(),
                "0.0.0.0".into(),
                "--port".into(),
                "8080".into(),
                "--data".into(),
                "/home/app-state".into(),
                "--bundle".into(),
                "/opt/application/bundle.json".into(),
                "--setup-token-file".into(),
                "/home/app-setup/token.json".into(),
                "--setup-receipt-file".into(),
                "/home/app-setup/receipt.json".into(),
            ],
            deadline,
        )?;
        let container = std::str::from_utf8(&output)
            .map_err(|_| "invalid setup creation")?
            .trim()
            .to_string();
        if !hex(&container, 64) {
            return Err("setup creation unconfirmed".into());
        };
        Ok(container)
    }
    fn setup_ready(&self, s: &SetupRecord, deadline: Instant) -> bool {
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
                format!("http://127.0.0.1:{}/setup", s.application.port),
            ],
            deadline.min(Instant::now() + std::time::Duration::from_secs(2)),
        )
        .is_ok_and(|b| b == b"200")
    }
    pub fn setup(&self, r: &SetupRequest, deadline: Instant) -> Result<SetupResult, String> {
        let s = &r.setup;
        s.application.references().validate()?;
        if !id(&s.id)
            || s.administrator_id <= 0
            || s.state != "pending"
            || s.expires_at <= 0
            || !["begin", "inspect", "complete", "abort"].contains(&r.action.as_str())
            || (!r.expected_container.is_empty() && !hex(&r.expected_container, 64))
            || (r.action == "complete" && r.expected_container.is_empty())
        {
            return Err("invalid provisional setup request".into());
        }
        let (root, control) = paths(s, true)?;
        let token_path = control.join("token.json");
        if r.action == "begin" && control.join("receipt.json").exists() {
            return Err("owner receipt already exists; inspect and complete this setup".into());
        }
        if r.action == "begin" && !token_path.exists() {
            // Missing credentials on an existing generation/data tree are an
            // uncertain prior claim, never authority to mint a replacement.
            let existing = self.command(
                "podman",
                &[
                    "ps".into(),
                    "--all".into(),
                    "--no-trunc".into(),
                    format!("--filter=name=^soda-canlang-setup-{}$", s.id),
                    "--format={{.ID}}".into(),
                ],
                deadline,
            )?;
            let data = root.join("data").join(&s.application.data);
            if !existing.iter().all(u8::is_ascii_whitespace)
                || (data.exists()
                    && fs::read_dir(&data)
                        .map_err(|_| "setup data unavailable")?
                        .any(|entry| entry.is_ok_and(|e| e.file_name() != ".writer-lock")))
            {
                return Err(
                    "setup credentials are unavailable; inspect or cancel the retained setup"
                        .into(),
                );
            }

            if now()? >= s.expires_at {
                return Err("setup token expired".into());
            };
            let mut bytes = [0u8; 32];
            getrandom::fill(&mut bytes).map_err(|_| "setup randomness unavailable")?;
            save(
                &token_path,
                &Token {
                    operation: s.id.clone(),
                    token: crate::sha256::hex_lower(&bytes),
                    expires_at: s.expires_at,
                },
                10000,
            )?
        }
        let mut c = match self.setup_container(s, &r.expected_container, deadline) {
            Ok(c) => c,
            Err(error) => {
                let existing = self.command(
                    "podman",
                    &[
                        "ps".into(),
                        "--all".into(),
                        "--no-trunc".into(),
                        format!("--filter=name=^soda-canlang-setup-{}$", s.id),
                        "--format={{.ID}}".into(),
                    ],
                    deadline,
                )?;
                if !existing.iter().all(u8::is_ascii_whitespace) {
                    return Err(error);
                };
                if r.action != "begin" {
                    if !r.expected_container.is_empty() {
                        return Err("setup generation is absent".into());
                    }
                    if r.action == "abort" && token_path.exists() {
                        fs::remove_file(&token_path)
                            .map_err(|_| "setup token cannot be retired")?;
                    }
                    return Ok(SetupResult {
                        operation: s.id.clone(),
                        observation: SetupObservation {
                            container: String::new(),
                            running: false,
                            ready: false,
                        },
                        setup_url: None,
                        receipt: None,
                        application: None,
                    });
                }
                let created = self.setup_create(s, &root, &control, deadline)?;
                self.setup_container(s, &created, deadline)?
            }
        };
        let mut result = SetupResult {
            operation: s.id.clone(),
            observation: SetupObservation {
                container: c.container.clone(),
                running: c.running,
                ready: false,
            },
            setup_url: None,
            receipt: None,
            application: None,
        };
        if r.action == "begin" {
            let token: Token = read(&token_path, 10000)?;
            if token.operation != s.id
                || token.expires_at != s.expires_at
                || !hex(&token.token, 64)
                || now()? >= token.expires_at
            {
                return Err("setup token expired or differs".into());
            };
            let refs = s.application.references();
            let activated =
                super::ingress::verify(&refs).is_ok() && c.running && self.setup_ready(s, deadline);
            super::ingress::install(&refs)?;
            // First activation can close the calling browser connection.
            // Reconciliation of that exact ready generation reuses its
            // capability without repeatedly restarting shared ingress.
            if !activated {
                self.activate_ingress(deadline)?;
            }
            if !c.running {
                self.command("podman", &["start".into(), c.container.clone()], deadline)?;
                c = self.setup_container(s, &c.container, deadline)?
            }
            let ready_deadline = deadline.min(Instant::now() + std::time::Duration::from_secs(30));
            while !self.setup_ready(s, ready_deadline) {
                if Instant::now() >= ready_deadline {
                    return Err("setup did not become ready".into());
                };
                std::thread::sleep(std::time::Duration::from_millis(200));
                c = self.setup_container(s, &c.container, ready_deadline)?;
                if !c.running {
                    return Err("setup process exited".into());
                }
            }
            result.observation.running = true;
            result.observation.ready = true;
            result.setup_url = Some(format!(
                "{}/setup#operation={}&token={}",
                s.application.origin, s.id, token.token
            ));
            return Ok(result);
        }
        if ["complete", "abort"].contains(&r.action.as_str()) {
            self.command(
                "podman",
                &["stop".into(), "--time=20".into(), c.container.clone()],
                deadline,
            )?;
            c = self.setup_container(s, &c.container, deadline)?;
            if c.running {
                return Err("setup writer did not stop".into());
            };
            result.observation.running = false;
            if r.action == "complete" {
                let receipt: Receipt = read(&control.join("receipt.json"), 10000)?;
                let image = self.admit_image(&s.application.references(), deadline)?;
                if receipt.schema != "canlang.soda.setup.v1"
                    || receipt.operation != s.id
                    || receipt.app != s.application.name
                    || receipt.artifactDigest != s.application.artifact_sha256
                    || receipt.bundleDigest != s.application.bundle_sha256
                    || receipt.completedAt <= 0
                    || receipt.completedAt > s.expires_at
                    || image.labels.get("io.canlang.storage") != Some(&receipt.storageFingerprint)
                    || !hex(&receipt.storageFingerprint, 64)
                {
                    return Err("setup receipt differs from admitted image and operation".into());
                }
                let spec = s.application.bind(&receipt.ownerTeam);
                spec.validate()?;
                super::ingress::verify(&spec)?;
                self.systemd("disable", &spec.id, deadline)?;
                let existing = self.container_id(&spec.id, deadline)?;
                let normal = if existing.is_empty() {
                    let created = self.create(&spec, deadline)?;
                    self.inspect(&spec, &created, deadline)?
                } else {
                    self.inspect(&spec, &existing, deadline)?
                };
                if normal.running || normal.image != spec.image {
                    return Err("completed application must be stopped at admitted image".into());
                };
                result.application = Some(self.observe(&spec, normal, deadline)?);
                result.receipt = Some(receipt)
            }
            if token_path.exists() {
                fs::remove_file(&token_path).map_err(|_| "setup token cannot be retired")?;
                File::open(&control)
                    .and_then(|f| f.sync_all())
                    .map_err(|_| "setup token retirement failed")?
            };
            return Ok(result);
        }
        result.observation.ready = c.running && self.setup_ready(s, deadline);
        Ok(result)
    }
}
