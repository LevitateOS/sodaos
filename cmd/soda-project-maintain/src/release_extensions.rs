//! Frozen Fountain intent and cache-only native package preparation.
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use soda_build_tools::payload::{FountainCatalog, FountainPackage, PayloadV4 as Payload};
use soda_host::tailnet_admission::Entry;
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt};
use std::path::Path;
use std::time::{Duration, Instant};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Snapshot {
    pub catalog_sha256: String,
    pub entries: Vec<Entry>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Frozen {
    pub source: Payload,
    pub target: Payload,
    pub snapshot: Snapshot,
}

fn rpc(path: &str, body: &[u8], cap: u64, deadline: Instant) -> Result<Vec<u8>, String> {
    if body.len() > 2 << 20 {
        return Err("extension request exceeds limit".into());
    }
    let remaining = deadline
        .checked_duration_since(Instant::now())
        .ok_or("extension deadline expired")?;
    let response = soda_unix_http::request(
        Path::new(soda_host::tailnet_admission::OPERATOR_SOCKET),
        "POST",
        path,
        "soda-operator",
        body,
        remaining.min(Duration::from_secs(cap)),
        soda_unix_http::Limits {
            header_bytes: 8192,
            body_bytes: 128 << 10,
        },
    )
    .map_err(|_| "Fountain extension operation unavailable")?;
    if response.status != 200 {
        return Err(format!(
            "Fountain extension operation refused ({})",
            response.status
        ));
    }
    Ok(response.body)
}

pub(crate) fn snapshot(deadline: Instant) -> Result<Snapshot, String> {
    soda_host::json::decode_strict_as(&rpc("/operator/extensions/snapshot", b"{}", 10, deadline)?)
        .map_err(|_| "invalid Fountain snapshot".into())
}

fn check_row_binding(row: &Entry, package: &FountainPackage) -> Result<(), String> {
    if row.package_id != package.id
        || row.required != package.required
        || row.pending_operation.is_some()
        || !soda_host::tcontrol_wire::valid_lifecycle_generation(&row.generation)
    {
        return Err("extension snapshot is invalid or has pending work".into());
    }
    if let Some(id) = &row.installation_id {
        if !soda_host::tcontrol_wire::valid_lifecycle_uuid(id) {
            return Err("invalid extension installation".into());
        }
    }
    Ok(())
}

fn check_row_intent(row: &Entry) -> Result<(), String> {
    if row.required
        && (!row.desired_enabled || row.generation != "0" || row.installation_id.is_none())
        || !row.required
            && ((row.generation == "0") != row.installation_id.is_none()
                || row.desired_enabled && row.generation == "0")
    {
        return Err("invalid extension intent identity".into());
    }
    Ok(())
}

fn check_row_receipt(row: &Entry, package: &FountainPackage) -> Result<(), String> {
    if row.required || row.desired_enabled {
        if row.installed_version.as_deref() != Some(&package.version)
            || row.package_sha256.as_deref() != Some(&package.sha256)
        {
            return Err("source package receipt differs from trusted catalog".into());
        }
    }
    Ok(())
}

impl Frozen {
    pub fn capture(target: &Payload, deadline: Instant) -> Result<Self, String> {
        let source: Payload = soda_host::json::decode_strict_as(
            &super::terminal_support::trusted_read(Path::new(crate::RELEASE_PATH), 4 << 20)?,
        )
        .map_err(|_| "source payload unavailable")?;
        let frozen = Self {
            source,
            target: target.clone(),
            snapshot: snapshot(deadline)?,
        };
        frozen.validate()?;
        Ok(frozen)
    }
    pub fn validate(&self) -> Result<(), String> {
        self.source
            .validate()
            .map_err(|_| "invalid source extension release")?;
        self.target
            .validate()
            .map_err(|_| "invalid target extension release")?;
        let bytes = self
            .source
            .projected_catalog_bytes()
            .map_err(|_| "invalid source catalog")?;
        if self.snapshot.catalog_sha256 != format!("{:x}", Sha256::digest(bytes))
            || self.snapshot.entries.len() != self.source.extension_catalog.packages.len()
        {
            return Err("snapshot does not bind the installed source catalog".into());
        }
        for (row, package) in self
            .snapshot
            .entries
            .iter()
            .zip(&self.source.extension_catalog.packages)
        {
            check_row_binding(row, package)?;
            check_row_intent(row)?;
            check_row_receipt(row, package)?;
        }
        Ok(())
    }
    pub fn selected(&self) -> Vec<String> {
        self.snapshot
            .entries
            .iter()
            .filter(|r| r.required || r.desired_enabled)
            .map(|r| r.package_id.clone())
            .collect()
    }
    pub fn same_intent(&self, current: &Snapshot) -> Result<(), String> {
        if current.entries.len() != self.snapshot.entries.len()
            || self
                .snapshot
                .entries
                .iter()
                .zip(&current.entries)
                .any(|(a, b)| {
                    a.package_id != b.package_id
                        || a.installation_id != b.installation_id
                        || a.generation != b.generation
                        || a.desired_enabled != b.desired_enabled
                        || a.required != b.required
                        || b.pending_operation.is_some()
                })
        {
            return Err(
                "extension intent changed since release snapshot; refusing stale release operation"
                    .into(),
            );
        }
        Ok(())
    }
    pub fn verify_packages(&self, rollback: bool, deadline: Instant) -> Result<(), String> {
        let current = snapshot(deadline)?;
        self.same_intent(&current)?;
        let payload = if rollback { &self.source } else { &self.target };
        let bytes = payload
            .projected_catalog_bytes()
            .map_err(|_| "invalid selected catalog")?;
        if current.catalog_sha256 != format!("{:x}", Sha256::digest(bytes)) {
            return Err("active extension catalog differs from release".into());
        }
        for (row, pkg) in current
            .entries
            .iter()
            .zip(&payload.extension_catalog.packages)
        {
            if (row.required || row.desired_enabled)
                && (row.installed_version.as_deref() != Some(&pkg.version)
                    || row.package_sha256.as_deref() != Some(&pkg.sha256))
            {
                return Err("active extension package differs from release".into());
            }
        }
        Ok(())
    }
    pub fn prefetch(&self, deadline: Instant) -> Result<(), String> {
        self.validate()?;
        for payload in [&self.source, &self.target] {
            ensure_images(
                payload,
                self.snapshot
                    .entries
                    .iter()
                    .any(|e| e.package_id == "tailnet" && e.desired_enabled),
                deadline,
            )?;
            let body=serde_json::to_vec(&serde_json::json!({"catalog":payload.extension_catalog,"package_ids":self.selected(),"source_snapshot":self.snapshot})).map_err(|_|"extension request invalid")?;
            verify_cache(
                &rpc("/operator/extensions/cache", &body, 600, deadline)?,
                &payload.extension_catalog,
                &self.selected(),
            )?;
        }
        Ok(())
    }
    pub fn prepare(&self, rollback: bool, deadline: Instant) -> Result<(), String> {
        let payload = if rollback { &self.source } else { &self.target };
        let inputs = inputs(payload, Some(&self.snapshot), deadline)?;
        crate::command::run("/usr/bin/systemctl", &["stop", "forgejo.service"], deadline)?;
        let image = payload.images.forgejo.config.as_str();
        let volume = format!("{}:/release:ro", inputs.display());
        let remaining = deadline.saturating_duration_since(Instant::now()).as_secs();
        if remaining <= 15 {
            return Err("offline preparation has insufficient remaining deadline".into());
        }
        let timeout = format!("--timeout={}", remaining - 15);
        crate::command::podman(
            &[
                "run",
                &timeout,
                "--stop-timeout=5",
                "--rm",
                "--pull=never",
                "--network=none",
                "--user=1000:1000",
                "--cap-drop=all",
                "--security-opt=no-new-privileges",
                "--volume=/var/lib/soda/forgejo:/data:Z",
                "--volume",
                &volume,
                "--env=FORGEJO__server__APP_DATA_PATH=/data/gitea",
                "--entrypoint=/usr/local/bin/gitea",
                image,
                "--config",
                "/data/gitea/conf/app.ini",
                "extensions",
                "prepare",
                "--offline",
                "--catalog",
                "/release/catalog.json",
                "--snapshot",
                "/release/snapshot.json",
            ],
            deadline,
        )
        .map_err(|_| {
            "offline extension preparation failed; recover using the saved release operation"
        })?;
        Ok(())
    }
}

fn verify_cache(body: &[u8], catalog: &FountainCatalog, selected: &[String]) -> Result<(), String> {
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Cached {
        id: String,
        version: String,
        sha256: String,
        size: u64,
    }
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Response {
        cached: Vec<Cached>,
    }
    let response: Response =
        soda_host::json::decode_strict_as(body).map_err(|_| "invalid extension cache receipt")?;
    let packages: Vec<_> = catalog
        .packages
        .iter()
        .filter(|p| selected.contains(&p.id))
        .collect();
    if response.cached.len() != packages.len()
        || response.cached.iter().zip(packages).any(|(a, b)| {
            a.id != b.id || a.version != b.version || a.sha256 != b.sha256 || a.size != b.size
        })
    {
        return Err("extension cache receipt differs from selected catalog".into());
    }
    Ok(())
}

fn prepare_input_dir() -> Result<std::path::PathBuf, String> {
    let dir = Path::new(soda_host::project::admission::DIRECTORY).join("extension-inputs");
    match fs::create_dir(&dir) {
        Ok(()) => fs::set_permissions(&dir, fs::Permissions::from_mode(0o755))
            .map_err(|_| "extension input permissions")?,
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
        Err(_) => return Err("extension inputs unavailable".into()),
    }
    let info = fs::symlink_metadata(&dir).map_err(|_| "extension inputs unavailable")?;
    if !info.is_dir() || info.uid() != 0 || info.mode() & 0o7777 != 0o755 {
        return Err("unsafe extension input directory".into());
    }
    Ok(dir)
}

fn collect_documents(
    payload: &Payload,
    snapshot: Option<&Snapshot>,
) -> Result<Vec<(&'static str, Vec<u8>)>, String> {
    let mut documents = vec![
        (
            "catalog.json",
            payload
                .projected_catalog_bytes()
                .map_err(|_| "invalid extension catalog")?,
        ),
        (
            "payload.json",
            serde_json::to_vec(payload).map_err(|_| "invalid payload")?,
        ),
    ];
    if let Some(snapshot) = snapshot {
        documents.push((
            "snapshot.json",
            serde_json::to_vec(snapshot).map_err(|_| "invalid snapshot")?,
        ));
    }
    Ok(documents)
}

fn retire_scratch(temporary: &Path) -> Result<(), String> {
    match fs::symlink_metadata(temporary) {
        Ok(info)
            if info.is_file()
                && info.uid() == 0
                && info.nlink() == 1
                && info.mode() & 0o022 == 0
                && info.len() <= 4 << 20 =>
        {
            fs::remove_file(temporary).map_err(|_| "extension scratch retirement failed")?
        }
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
        _ => return Err("unsafe extension input scratch".into()),
    }
    Ok(())
}

fn write_document(dir: &Path, name: &str, bytes: &[u8]) -> Result<(), String> {
    let temporary = dir.join(format!("{name}.next"));
    retire_scratch(&temporary)?;
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o644)
        .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(&temporary)
        .map_err(|_| "extension input scratch already exists or is unavailable")?;
    file.set_permissions(fs::Permissions::from_mode(0o644))
        .map_err(|_| "extension input permissions")?;
    file.write_all(bytes)
        .and_then(|_| file.sync_all())
        .map_err(|_| "extension input write failed")?;
    fs::rename(temporary, dir.join(name)).map_err(|_| "extension input replacement failed")?;
    Ok(())
}

fn inputs(
    payload: &Payload,
    snapshot: Option<&Snapshot>,
    deadline: Instant,
) -> Result<std::path::PathBuf, String> {
    let dir = prepare_input_dir()?;
    for (name, bytes) in collect_documents(payload, snapshot)? {
        write_document(&dir, name, &bytes)?;
    }
    fs::File::open(&dir)
        .and_then(|f| f.sync_all())
        .map_err(|_| "extension input sync failed")?;
    // The declared public-input label is separate from private journals.
    // Never relabel the shared maintenance inode or grant journal reads.
    crate::command::run(
        "/usr/sbin/restorecon",
        &["-R", dir.to_str().ok_or("invalid input path")?],
        deadline,
    )
    .map_err(|_| "extension input labels unavailable")?;
    Ok(dir)
}

fn ensure_images(payload: &Payload, tailnet: bool, deadline: Instant) -> Result<(), String> {
    // Same importer and native Podman store as boot; no embedded OCI reader.
    // Every acquisition runs under durable custody: the exact payload and
    // argv bind before launch, the group identity binds at launch, and
    // retirement is recorded after the wait loop verifies the empty group.
    // A failed acquisition keeps its launched custody so recovery either
    // retires the group or fences; it never silently releases descendants.
    let dir = inputs(payload, None, deadline)?;
    let path = dir.join("payload.json");
    let path = path.to_str().ok_or("invalid release input path")?;
    let bytes = fs::read(path).map_err(|_| "release payload unreadable")?;
    let digest = format!("{:x}", Sha256::digest(&bytes));
    let custody = Path::new(soda_host::project::admission::DIRECTORY);
    for set in std::iter::once("required").chain(tailnet.then_some("tailnet")) {
        let argv = vec![
            "ensure".to_string(),
            "--payload".to_string(),
            path.to_string(),
            "--set".to_string(),
            set.to_string(),
        ];
        let args: Vec<&str> = argv.iter().map(String::as_str).collect();
        crate::importer_custody::acquire(custody, &digest, &argv, 0)?;
        let launched = crate::command::run_group_observed(
            "/usr/libexec/soda/soda-image-import",
            &args,
            deadline,
            |pgid| crate::importer_custody::launched(custody, &digest, &argv, pgid, 0),
        );
        match launched {
            Ok(_) => crate::importer_custody::retired(custody, &digest, 0)?,
            Err(_) => return Err("selected release image acquisition failed".into()),
        }
    }
    Ok(())
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ReconciledRow {
    package_id: String,
    #[serde(deserialize_with = "Option::deserialize")]
    installation_id: Option<String>,
    generation: String,
    desired_enabled: bool,
    #[serde(deserialize_with = "Option::deserialize")]
    pending_operation: Option<String>,
    #[serde(deserialize_with = "Option::deserialize")]
    installed_version: Option<String>,
    #[serde(deserialize_with = "Option::deserialize")]
    package_sha256: Option<String>,
    required: bool,
    backend: String,
    effects: String,
    state: String,
    admission_open: bool,
    #[serde(deserialize_with = "Option::deserialize")]
    error: Option<soda_host::tcontrol_wire::LifecycleError>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Reconciled {
    catalog_sha256: String,
    entries: Vec<ReconciledRow>,
}

pub(crate) fn reconcile(frozen: &Frozen, rollback: bool, deadline: Instant) -> Result<(), String> {
    let body = rpc("/operator/extensions/reconcile", b"{}", 300, deadline)?;
    let current = validate_reconciled(frozen, rollback, &body)?;
    // A later administrator decision wins; never overwrite it to satisfy this release.
    let state = soda_host::tailnet_admission::manager_state(
        Path::new(soda_host::tailnet_admission::OPERATOR_SOCKET),
        deadline,
    )?;
    let expected = frozen
        .snapshot
        .entries
        .iter()
        .find(|r| r.package_id == "tailnet")
        .ok_or("missing Tailnet snapshot")?;
    if state.controller_id != "soda"
        || state.catalog_sha256 != current.catalog_sha256
        || state.entry.package_id != expected.package_id
        || state.entry.installation_id != expected.installation_id
        || state.entry.generation != expected.generation
        || state.entry.desired_enabled != expected.desired_enabled
        || state.entry.pending_operation.is_some()
        || state.admission_open != expected.desired_enabled
        || !state.controller_ready
    {
        return Err("extension state changed or remains unconfirmed after core commit".into());
    }
    Ok(())
}

fn validate_reconciled(frozen: &Frozen, rollback: bool, body: &[u8]) -> Result<Snapshot, String> {
    let result: Reconciled = soda_host::json::decode_strict_as(body)
        .map_err(|_| "invalid extension reconciliation response")?;
    let current = Snapshot {
        catalog_sha256: result.catalog_sha256,
        entries: result
            .entries
            .iter()
            .map(|r| Entry {
                package_id: r.package_id.clone(),
                installation_id: r.installation_id.clone(),
                generation: r.generation.clone(),
                desired_enabled: r.desired_enabled,
                pending_operation: r.pending_operation.clone(),
                installed_version: r.installed_version.clone(),
                package_sha256: r.package_sha256.clone(),
                required: r.required,
            })
            .collect(),
    };
    frozen.same_intent(&current)?;
    let payload = if rollback {
        &frozen.source
    } else {
        &frozen.target
    };
    if current.catalog_sha256
        != format!(
            "{:x}",
            Sha256::digest(
                payload
                    .projected_catalog_bytes()
                    .map_err(|_| "invalid catalog")?
            )
        )
    {
        return Err("post-commit catalog differs from release".into());
    }
    for (r, pkg) in result
        .entries
        .iter()
        .zip(&payload.extension_catalog.packages)
    {
        let ready = if r.required || r.desired_enabled {
            r.state == "enabled"
                && r.backend == "running"
                && r.effects == "running"
                && r.admission_open
                && r.installed_version.as_ref() == Some(&pkg.version)
                && r.package_sha256.as_ref() == Some(&pkg.sha256)
        } else {
            (r.state == "disabled" || r.state == "available")
                && r.backend == "stopped"
                && r.effects == "stopped"
                && !r.admission_open
        };
        if !ready || r.error.is_some() {
            return Err("extension effects are failed or unknown after core commit".into());
        }
    }
    Ok(current)
}

impl Frozen {
    /// Read-only custody checks before rollback/recovery mutates host or packages.
    /// Fountain remains the only archive interpreter and package installer.
    pub fn retained(&self, rollback: bool, deadline: Instant) -> Result<(), String> {
        use std::io::Read;
        let payload = if rollback { &self.source } else { &self.target };
        let selected = self.selected();
        for package in payload
            .extension_catalog
            .packages
            .iter()
            .filter(|p| selected.contains(&p.id))
        {
            let path = Path::new("/var/lib/soda/forgejo/gitea/extensions/.cache/packages")
                .join(format!("{}.tar.gz", package.sha256));
            let mut file = OpenOptions::new()
                .read(true)
                .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
                .open(path)
                .map_err(|_| {
                    "retained extension archive missing; refusing maintenance before mutation"
                })?;
            let info = file
                .metadata()
                .map_err(|_| "retained archive unavailable")?;
            if !info.is_file() || info.len() != package.size {
                return Err("retained extension archive size changed".into());
            }
            let mut hash = Sha256::new();
            let mut buf = [0; 65536];
            loop {
                if Instant::now() >= deadline {
                    return Err("retained archive validation deadline expired".into());
                }
                let n = file
                    .read(&mut buf)
                    .map_err(|_| "retained archive unreadable")?;
                if n == 0 {
                    break;
                }
                hash.update(&buf[..n]);
            }
            if format!("{:x}", hash.finalize()) != package.sha256 {
                return Err("retained extension archive digest changed".into());
            }
        }
        let mut images: Vec<&str> = soda_build_tools::payload::Role::ALL
            .iter()
            .map(|role| role.binding(&payload.images).config.as_str())
            .collect();
        if self
            .snapshot
            .entries
            .iter()
            .any(|r| r.package_id == "tailnet" && r.desired_enabled)
        {
            images.push(&payload.optional_extensions.tailnet.images.tailscale.config);
        }
        for image in images {
            let body = crate::command::podman(
                &[
                    "image",
                    "inspect",
                    "--format",
                    "{{.ID}} {{.Os}} {{.Architecture}}",
                    image,
                ],
                deadline,
            )
            .map_err(|_| "retained release image missing")?;
            let text = std::str::from_utf8(&body).map_err(|_| "retained image identity invalid")?;
            let fields = text.split_whitespace().collect::<Vec<_>>();
            if fields.len() != 3
                || fields[0].trim_start_matches("sha256:") != image.trim_start_matches("sha256:")
                || fields[1..] != ["linux", "amd64"]
            {
                return Err("retained release image identity changed".into());
            }
        }
        Ok(())
    }
}
