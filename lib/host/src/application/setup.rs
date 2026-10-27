//! Provisional app-owned owner claim. No ordinary application exists until the
//! exact stopped setup generation has produced its bound receipt.
use super::runtime::{data_directory, Runtime};
use super::wire::{hex, id, Observation, Spec};
use crate::project::Executor;
use serde::{Deserialize, Serialize};
use std::fs::{self};
use std::io::Read;
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
pub struct Token {
    pub operation: String,
    pub token: String,
    pub expires_at: i64,
}
#[derive(Deserialize)]
struct Container {
    container: String,
    image: String,
    running: bool,
    labels: std::collections::BTreeMap<String, String>,
}
// Renewal re-arms an expired reservation for one fresh creation window
// (the store grants 30 minutes); the host admits only a forward-bounded
// expiry advance on the otherwise identical retained reservation. The store
// remains the policy authority on grace window and renewal count.
pub const SETUP_RENEWAL_WINDOW_SECS: i64 = 35 * 60;

/// FJ-874: running-holder fence predicate for the post-reconcile re-check.
/// The re-check runs default `podman ps` (running containers only), so a
/// correctly stopped retained container is simply absent and must NOT
/// refuse an unclaimed rotation; any listed ID is a genuinely running
/// holder and still fences. Pure so the running-vs-stopped boundary is
/// testable without a container runtime.
pub fn running_holder_listed(ps_output: &[u8]) -> bool {
    !ps_output.iter().all(u8::is_ascii_whitespace)
}

/// FJ-862: typed outcome of the packaged private Canlang reconcile for one
/// exact setup generation/operation/app context. Only `Unclaimed` —
/// proven-unclaimed, no committed owner binding and no competing durable
/// holder — permits the caller to rotate or replace the owner credential.
/// `Committed` (a committed binding exists; expiry never uncommits),
/// `Conflict` (another durable holder claims the namespace), and
/// `Uncertain` (anything the host cannot prove, including every reconcile
/// invocation failure) keep the old token and fail closed. The original
/// token is never destroyed on unknown/conflict.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ReconcileOutcome {
    Unclaimed,
    Committed,
    Conflict,
    Uncertain,
}

/// FJ-862: what a genuine reconcile success must bind. The stdout verdict
/// is only trusted when the reconstructed receipt names this exact
/// generation/operation/app context: `operation` pins the setup id,
/// `app`/`artifact_digest`/`bundle_digest` pin the admitted application
/// references, and `storage_fingerprint` pins the admitted runtime image's
/// storage label. A success that names anything else is a confused output
/// and refuses exactly like an unparsable one.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ReconcileExpectation {
    pub operation: String,
    pub app: String,
    pub artifact_digest: String,
    pub bundle_digest: String,
    pub storage_fingerprint: String,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ReconcileSuccess {
    ok: bool,
    command: String,
    schema: String,
    operation: String,
    app: String,
    #[serde(rename = "artifactDigest")]
    artifact_digest: String,
    #[serde(rename = "bundleDigest")]
    bundle_digest: String,
    #[serde(rename = "storageFingerprint")]
    storage_fingerprint: String,
    #[serde(rename = "ownerTeam")]
    owner_team: String,
    #[serde(rename = "completedAt")]
    completed_at: i64,
}

/// FJ-862: the packaged CLI's two sanitized nonzero outcomes, as they
/// appear at the END of the executor's failure diagnostic (`podman failed:
/// exit status 1: can-soda: …`). The CLI emits these two metadata-only
/// lines and nothing else on this path; podman wrapper text may precede
/// them, but free-form text after them (or anywhere else) never matches.
pub const RECONCILE_UNCLAIMED_LINE: &str =
    "can-soda: Setup reconciliation found no committed claim.";
pub const RECONCILE_REFUSAL_LINE: &str = "can-soda: Setup reconciliation refused.";

/// FJ-862: classification of a failed reconcile invocation. Only the two
/// exact CLI literals classify; every podman failure, timeout, capture
/// limit, or truncated pipe is `Unknown` and fails closed downstream.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ReconcileRefusal {
    Unclaimed,
    Refused,
    Unknown,
}

/// Fixed allowlisted mapping of a failed reconcile invocation's executor
/// diagnostic to a typed refusal. The match is suffix-anchored on the
/// exact CLI literals: free-form stderr text never matches, never
/// classifies, and never surfaces — callers only see the typed refusal.
pub fn classify_reconcile_failure(diagnostic: &str) -> ReconcileRefusal {
    let trimmed = diagnostic.trim_end();
    if trimmed.ends_with(RECONCILE_UNCLAIMED_LINE) {
        ReconcileRefusal::Unclaimed
    } else if trimmed.ends_with(RECONCILE_REFUSAL_LINE) {
        ReconcileRefusal::Refused
    } else {
        ReconcileRefusal::Unknown
    }
}

/// Fixed allowlisted mapping of the packaged reconcile's stdout to a typed
/// outcome. Only the exact success shape — `ok:true` for the reconcile
/// command with every receipt field present and bound to this exact
/// generation/operation/app context — yields `Committed`; every other byte
/// sequence — garbage, empty, wrong shapes, unbound receipts, extra
/// fields, trailing non-whitespace bytes — is `Uncertain`. Stderr and all
/// other host text are never consulted: this function takes stdout only,
/// so there is structurally nothing else to trust. A nonzero exit never
/// reaches this function (see `classify_reconcile_failure`); stdout on a
/// failed invocation is discarded, never parsed.
pub fn parse_reconcile_outcome(stdout: &[u8], expected: &ReconcileExpectation) -> ReconcileOutcome {
    // The packaged CLI prints a trailing newline; only ASCII whitespace is
    // trimmed. Trailing non-whitespace bytes still refuse below, and leading
    // whitespace is trimmed for the same reason.
    let mut bytes = stdout;
    while bytes.first().is_some_and(u8::is_ascii_whitespace) {
        bytes = &bytes[1..];
    }
    while bytes.last().is_some_and(u8::is_ascii_whitespace) {
        bytes = &bytes[..bytes.len() - 1];
    }
    let report: ReconcileSuccess = match serde_json::from_slice(bytes) {
        Ok(report) => report,
        Err(_) => return ReconcileOutcome::Uncertain,
    };
    if !report.ok
        || report.command != "reconcile"
        || report.schema != "canlang.soda.setup.v1"
        || report.operation != expected.operation
        || report.app != expected.app
        || report.artifact_digest != expected.artifact_digest
        || report.bundle_digest != expected.bundle_digest
        || report.storage_fingerprint != expected.storage_fingerprint
        || report.owner_team.is_empty()
        || report.completed_at <= 0
    {
        return ReconcileOutcome::Uncertain;
    }
    ReconcileOutcome::Committed
}

/// Rotation gate: only the proven-unclaimed outcome permits replacement.
/// Pure so the fail-closed boundary is testable without a container
/// runtime or a live Canlang server.
pub fn reconcile_permits_rotation(outcome: ReconcileOutcome) -> bool {
    outcome == ReconcileOutcome::Unclaimed
}

/// Authorized-renewal predicate for the retained reservation journal: only
/// `expires_at` advances, forward and within one fresh creation window.
/// Anything else is still tampering. Pure so the rotation boundary is
/// testable without a container runtime.
pub fn setup_renewal_advance(
    previous: &SetupRecord,
    identity: &SetupRecord,
    now_secs: i64,
) -> bool {
    previous.id == identity.id
        && previous.application == identity.application
        && previous.administrator_id == identity.administrator_id
        && previous.state == identity.state
        && previous.observation == identity.observation
        && previous.expires_at < identity.expires_at
        && identity.expires_at <= now_secs + SETUP_RENEWAL_WINDOW_SECS
}

fn now() -> Result<i64, String> {
    Ok(SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|_| "setup clock unavailable")?
        .as_secs() as i64)
}
fn parse_record<T: serde::de::DeserializeOwned>(
    file: &std::fs::File,
    uid: u32,
) -> Result<T, String> {
    // FJ-847: owner, mode, link count, and length are asserted on the opened
    // file descriptor, never re-read from the path.
    let stat = super::home::stat_opened(file, "setup record unavailable")?;
    if (stat.st_mode as u32 & libc::S_IFMT) != libc::S_IFREG
        || stat.st_uid != uid
        || (stat.st_mode as u32 & 0o077) != 0
        || stat.st_nlink != 1
        || stat.st_size < 0
        || stat.st_size as u64 > 8192
    {
        return Err("setup record is unsafe".into());
    };
    serde_json::from_reader(file.take(8193)).map_err(|_| "setup record invalid".into())
}
fn read_at<T: serde::de::DeserializeOwned>(
    dir: &std::os::fd::OwnedFd,
    name: &str,
    uid: u32,
) -> Result<T, String> {
    let file = super::home::open_child_opt(dir, name, false, "setup record unavailable")?
        .ok_or("setup record unavailable")?;
    parse_record(&file, uid)
}
fn staged_name(name: &str) -> Result<String, String> {
    let mut nonce = [0u8; 12];
    getrandom::fill(&mut nonce).map_err(|_| "setup randomness unavailable")?;
    Ok(format!(
        ".{}.{}.pending",
        name,
        crate::sha256::hex_lower(&nonce)
    ))
}

fn stage_record<T: Serialize>(
    dir: &std::os::fd::OwnedFd,
    name: &str,
    value: &T,
    uid: u32,
) -> Result<String, String> {
    // FJ-847: staging resolves through the validated parent descriptor:
    // exclusive no-follow creation, ownership assigned on the open file.
    let bytes = serde_json::to_vec(value).map_err(|_| "setup record encoding failed")?;
    let temporary = staged_name(name)?;
    let file = super::home::open_staged_file(
        dir,
        &temporary,
        uid,
        "setup record cannot be staged",
        "setup record ownership failed",
    )?;
    super::home::write_full_and_sync(
        dir,
        &temporary,
        &file,
        &bytes,
        "setup record write failed",
        "setup record sync failed",
    )?;
    Ok(temporary)
}

/// Atomically publish a setup file: stage beside the target with the same
/// permission and sync discipline as admission, then rename over the
/// target. The previous bytes survive every fallible step until the final
/// atomic rename, so a failed rotation never strands the retained setup
/// without its old credential.
pub fn replace<T: Serialize>(path: &Path, value: &T, uid: u32) -> Result<(), String> {
    // Journal renewal publication: same permission and sync discipline as
    // admission staging, but an overwriting rename so the retained
    // reservation journal can advance its expiry. Used only for the
    // forward-bounded renewal the caller pinned above. FJ-847: every step
    // resolves through the validated parent descriptor, never a bare path.
    let parent = path.parent().ok_or("setup record has no parent")?;
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or("setup record has no parent")?;
    let anchor = super::home::open_parent_dir(parent, "setup record cannot be staged")?;
    let temporary = stage_record(&anchor, name, value, uid)?;
    super::home::publish_staged_replace(
        &anchor,
        &temporary,
        name,
        "setup record publication failed",
        "setup directory sync failed",
    )
}

fn save_at<T: Serialize>(
    dir: &std::os::fd::OwnedFd,
    name: &str,
    value: &T,
    uid: u32,
) -> Result<(), String> {
    let temporary = stage_record(dir, name, value, uid)?;
    super::home::publish_staged_noreplace(
        dir,
        &temporary,
        name,
        "setup record already exists or publication failed",
        "setup directory sync failed",
    )
}

fn replace_at<T: Serialize>(
    dir: &std::os::fd::OwnedFd,
    name: &str,
    value: &T,
    uid: u32,
) -> Result<(), String> {
    let temporary = stage_record(dir, name, value, uid)?;
    super::home::publish_staged_replace(
        dir,
        &temporary,
        name,
        "setup record publication failed",
        "setup directory sync failed",
    )
}

fn directory(
    parent: &std::os::fd::OwnedFd,
    name: &str,
    uid: u32,
) -> Result<std::os::fd::OwnedFd, String> {
    // FJ-847: creation (or re-validation) goes through the validated parent
    // descriptor. Only a directory this call just created is assigned its
    // owner; an existing directory is re-validated and never chowned.
    super::home::make_child_dir(parent, name, uid, "setup directory is unsafe")
}
/// FJ-855: anchored application-data home shared by creation and the
/// pre-rotation reconcile. The operator-owned data parent is validated
/// through the storage-root anchor; the runtime-owned leaf is created
/// through that anchor (never by privileged path strings) with its writer
/// lock, or re-validated when it already exists; the podman path is rebound
/// to the new descriptor before hand-off. Preparing the empty home mints no
/// credential — token and journal decisions stay with the caller.
fn ensure_data_dir(dirs: &SetupDirs, data: &str) -> Result<PathBuf, String> {
    let data_parent = super::home::validate_child_dir(
        &dirs.root_anchor,
        "data",
        0,
        0o077,
        "application storage is unavailable",
    )?;
    let path = dirs.root.join("data").join(data);
    match super::home::open_child_dir_opt(&data_parent, data, "setup data unavailable")? {
        Some(app) => {
            super::home::validate_open_dir(&app, 10000, 0o077, "setup directory is unsafe")?;
            super::home::fd_matches_path(&path, &app)
                .map_err(|_| "setup directory is unsafe".to_string())?;
        }
        None => {
            let app = directory(&data_parent, data, 10000)?;
            let lock = super::home::open_staged_file(
                &app,
                ".writer-lock",
                10000,
                "setup record cannot be staged",
                "setup record ownership failed",
            )?;
            super::home::write_full_and_sync(
                &app,
                ".writer-lock",
                &lock,
                b"{}",
                "setup record write failed",
                "setup record sync failed",
            )?;
            drop(lock);
            super::home::fsync_fd(&app, "setup directory sync failed")?;
            super::home::fd_matches_path(&path, &app)
                .map_err(|_| "setup directory is unsafe".to_string())?;
        }
    }
    Ok(path)
}

/// FJ-862: retained application state beyond the writer lock. Used only
/// to recover the conflict signal the CLI redacts: the binary refuses
/// identically for foreign/incomplete/mismatched claims, and a populated
/// store without a matching claim is exactly the conflict case. An empty
/// data home (writer lock only) carries no such signal.
fn data_has_claim_evidence(data: &Path) -> bool {
    fs::read_dir(data)
        .map(|mut entries| {
            entries.any(|entry| entry.is_ok_and(|e| e.file_name() != ".writer-lock"))
        })
        .unwrap_or(false)
}

fn application_root() -> Result<PathBuf, String> {
    // Single trusted seam shared with runtime data and snapshots (see
    // `home::storage_root`): production keeps the fixed path; the fixture
    // redirect applies to all three together, never a split root. Either way
    // the value passes the identical operator-owned /home tree checks plus
    // the root ownership checks below, so an outside root is refused before
    // any mutation.
    super::storage_root()
}
/// Validated setup directories. `root` and `control` are path strings only
/// for podman arguments and directory scans; every privileged mutation uses
/// `data_parent` / `control_dir`, and the paths are rebound to those
/// descriptors by dev/ino before hand-off.
struct SetupDirs {
    root: PathBuf,
    root_anchor: std::os::fd::OwnedFd,
    control: PathBuf,
    control_dir: std::os::fd::OwnedFd,
}

fn paths(s: &SetupRecord, begin: bool) -> Result<SetupDirs, String> {
    // FJ-847: the storage root anchor is validated BEFORE any creation, the
    // setups journal and the runtime control dir are created through it, and
    // every journal/token mutation below resolves through those descriptors.
    let root = application_root()?;
    let root_anchor = super::home::open_validated_dir(
        &root,
        0,
        0o022,
        "application storage root must belong to the operator",
    )?;

    let setups = directory(&root_anchor, "setups", 0)?;
    let journal = format!("{}.json", s.id);
    let mut identity = s.clone();
    identity.observation = None;
    match super::home::open_child_opt(&setups, &journal, false, "setup record unavailable")? {
        Some(_) => {
            let previous: SetupRecord = read_at(&setups, &journal, 0)?;
            if previous != identity {
                // An authorized renewal advances only the reservation expiry on
                // the retained id, references, administrator and state. The
                // advance must be forward and within one fresh creation window;
                // any other admission change is still tampering.
                if !setup_renewal_advance(&previous, &identity, now()?) {
                    return Err("setup immutable admission changed".into());
                }
                replace_at(&setups, &journal, &identity, 0)?
            }
        }
        None => {
            if !begin {
                return Err("setup has not been prepared".into());
            };
            save_at(&setups, &journal, &identity, 0)?
        }
    }
    let control_dir = directory(&setups, &s.id, 10000)?;
    let control = root.join("setups").join(&s.id);
    super::home::fd_matches_path(&control, &control_dir)
        .map_err(|_| "setup directory is unsafe".to_string())?;
    Ok(SetupDirs {
        root,
        root_anchor,
        control,
        control_dir,
    })
}
impl<E: Executor> Runtime<E> {
    /// Stop the retained setup generation's live holder before a renewal
    /// rotates its credential. A stopped or absent generation needs no
    /// stop; a holder that will not stop refuses the rotation with the old
    /// token file untouched, so a later retry converges instead of
    /// stranding the retained setup without any credential.
    fn stop_retained_setup(&self, s: &SetupRecord, deadline: Instant) -> Result<(), String> {
        let live = self.command(
            "podman",
            &[
                "ps".into(),
                "--no-trunc".into(),
                format!("--filter=name=^soda-canlang-setup-{}$", s.id),
                "--format={{.ID}}".into(),
            ],
            deadline,
        )?;
        if live.iter().all(u8::is_ascii_whitespace) {
            return Ok(());
        }
        let c = self
            .setup_container(s, "", deadline)
            .map_err(|_| "setup generation differs; inspect or cancel the retained setup")?;
        if !c.running {
            return Ok(());
        }
        self.command(
            "podman",
            &["stop".into(), "--time=20".into(), c.container.clone()],
            deadline,
        )?;
        let c = self.setup_container(s, &c.container, deadline)?;
        if c.running {
            return Err("setup process is still running; retry renewal after it stops".into());
        }
        Ok(())
    }
    /// FJ-862: run the packaged private Canlang reconcile for the exact
    /// generation/operation/app context and return its TYPED outcome.
    ///
    /// The caller stops/fences the exact setup holder first
    /// (`stop_retained_setup`, whose pinned re-inspect is the inspection
    /// under real writer custody — holder death is never assumed from
    /// store state); this method then wires the invocation without
    /// reimplementing the reconcile: the ADMITTED image's packaged
    /// `/opt/canlang/bin/can-soda reconcile` entrypoint runs once,
    /// ephemeral (`--rm`, unnamed), read-only image rootfs, unprivileged,
    /// without a network, against the exact data and setup volumes rebound
    /// to their validated descriptors. Only the admitted image ever runs:
    /// admission failure, volume rebind failure, invocation failure, or an
    /// unclassifiable verdict all yield a non-permitting outcome — never
    /// a permit.
    ///
    /// Argv contract (defined here; verified against the packaged CLI in
    /// `packages/cloudflare/src/cli/soda-runtime.ts` and proven against
    /// the packaged binary in the 862 release logs): `reconcile --bundle
    /// /opt/application/bundle.json --data /home/app-state
    /// --setup-token-file /home/app-setup/token.json --setup-receipt-file
    /// /home/app-setup/receipt.json`. There is deliberately NO
    /// `--operation` flag — the packaged CLI rejects unknown flags, so
    /// generation/operation/app binding rides host admission, the
    /// container labels, and the stdout receipt binding instead.
    ///
    /// Mounts: both volumes are read-write — the data mount because the
    /// reconcile takes the Canlang O_RDWR writer lock and reads the claim
    /// from the provisioned store inside it, and the setup mount because
    /// the reconcile RECONSTRUCTS and saves the receipt file there (a
    /// read-only receipt path cannot work; proven in the 862 logs).
    /// Least privilege otherwise: read-only image rootfs, no network, no
    /// new privileges, no capabilities, bounded pids/memory/cpus,
    /// ephemeral, unnamed, unprivileged account-owned leaves.
    ///
    /// Verdicts: exit 0 with the bound `ok:true` stdout shape is
    /// `Committed`; the sanitized nonzero unclaimed line is `Unclaimed`
    /// (the only permit); the sanitized nonzero refusal line is
    /// `Conflict` when retained application data is present (the CLI
    /// redacts every foreign/incomplete/mismatched claim to that one
    /// line, and a populated store without a matching claim is exactly
    /// the conflict case) and `Uncertain` otherwise. The executor's
    /// failure diagnostic never leaves this function: callers only see
    /// the typed outcome, never host or stderr text.
    fn reconcile_setup_claim(
        &self,
        s: &SetupRecord,
        dirs: &SetupDirs,
        deadline: Instant,
    ) -> ReconcileOutcome {
        // Pin the binary: only the admitted image's packaged entrypoint.
        // Its storage label also binds the success shape below.
        let image = match self.admit_image(&s.application.references(), deadline) {
            Ok(image) => image,
            Err(_) => return ReconcileOutcome::Uncertain,
        };
        // Rebind both volume paths to their validated descriptors at the
        // moment of use; preparing the empty data home mints no credential.
        let data = match ensure_data_dir(dirs, &s.application.data) {
            Ok(data) => data,
            Err(_) => return ReconcileOutcome::Uncertain,
        };
        if super::home::fd_matches_path(&dirs.control, &dirs.control_dir).is_err() {
            return ReconcileOutcome::Uncertain;
        }
        // FJ-862: the executor surfaces the binary's sanitized stderr line
        // inside its failure diagnostic, so the nonzero outcomes stay
        // mappable WITHOUT trusting free text: only the two exact literals
        // classify, via `classify_reconcile_failure`. `self.command` would
        // redact the diagnostic, so this call goes to the executor
        // directly with the same bounds.
        let args = [
            "run".to_string(),
            "--rm".into(),
            "--pull=never".into(),
            "--read-only".into(),
            "--cap-drop=all".into(),
            "--security-opt=no-new-privileges".into(),
            "--pids-limit=128".into(),
            "--memory=1g".into(),
            "--cpus=2".into(),
            "--network=none".into(),
            format!("--volume={}:/home/app-state:rw,Z", data.display()),
            format!("--volume={}:/home/app-setup:rw,Z", dirs.control.display()),
            format!("--label=io.soda.canlang.setup={}", s.id),
            format!("--label=io.soda.canlang.data={}", s.application.data),
            format!("--label=io.soda.canlang.origin={}", s.application.origin),
            format!("--label=io.soda.canlang.port={}", s.application.port),
            "--entrypoint=/opt/canlang/bin/can-soda".into(),
            s.application.image.clone(),
            "reconcile".into(),
            "--bundle".into(),
            "/opt/application/bundle.json".into(),
            "--data".into(),
            "/home/app-state".into(),
            "--setup-token-file".into(),
            "/home/app-setup/token.json".into(),
            "--setup-receipt-file".into(),
            "/home/app-setup/receipt.json".into(),
        ];
        let argv: Vec<&str> = args.iter().map(String::as_str).collect();
        match self
            .exec
            .run_bounded(&[], "podman", &argv, deadline, 65536, 4096)
        {
            Ok(stdout) => {
                let storage = image
                    .labels
                    .get("io.canlang.storage")
                    .cloned()
                    .unwrap_or_default();
                parse_reconcile_outcome(
                    &stdout,
                    &ReconcileExpectation {
                        operation: s.id.clone(),
                        app: s.application.name.clone(),
                        artifact_digest: s.application.artifact_sha256.clone(),
                        bundle_digest: s.application.bundle_sha256.clone(),
                        storage_fingerprint: storage,
                    },
                )
            }
            Err(diagnostic) => match classify_reconcile_failure(&diagnostic) {
                ReconcileRefusal::Unclaimed => ReconcileOutcome::Unclaimed,
                ReconcileRefusal::Refused if data_has_claim_evidence(&data) => {
                    ReconcileOutcome::Conflict
                }
                ReconcileRefusal::Refused | ReconcileRefusal::Unknown => {
                    ReconcileOutcome::Uncertain
                }
            },
        }
    }
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
        dirs: &SetupDirs,
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
        // FJ-847/FJ-855: the application-data home is anchored from the
        // validated storage root and rebound before use; a non-empty leaf
        // still refuses a NEW setup below.
        let data = ensure_data_dir(dirs, &a.data)?;
        if fs::read_dir(&data)
            .map_err(|_| "setup data unavailable")?
            .any(|entry| entry.is_ok_and(|e| e.file_name() != ".writer-lock"))
        {
            return Err("new setup requires fresh empty application data".into());
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
                format!("--volume={}:/home/app-setup:rw,Z", dirs.control.display()),
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
        // Identity before mutation: when the caller pins a generation, bind
        // it against the retained setup BEFORE any journal, holder, or
        // credential mutation. A stale caller is refused here with zero
        // mutation instead of stopping or swapping the generation it
        // cannot name.
        if r.action == "begin" && !r.expected_container.is_empty() {
            self.setup_container(s, &r.expected_container, deadline)
                .map_err(|_| "setup generation differs; inspect or cancel the retained setup")?;
        }
        // FJ-847: `dirs` carries the validated anchors; the receipt, token,
        // and journal below are all reached through the control-dir
        // descriptor, never re-resolved from bare paths.
        let dirs = paths(s, true)?;
        let token_present = super::home::open_child_opt(
            &dirs.control_dir,
            "token.json",
            false,
            "setup record unavailable",
        )?
        .is_some();
        if r.action == "begin"
            && super::home::open_child_opt(
                &dirs.control_dir,
                "receipt.json",
                false,
                "setup record unavailable",
            )?
            .is_some()
        {
            return Err("owner receipt already exists; inspect and complete this setup".into());
        }
        // An authorized renewal rotates the expired same-operation token so
        // the retained reservation mints a fresh credential. The claim
        // stays certain: operation matches this setup, the record expiry
        // advanced past the token under the journal-pinned renewal above,
        // and the old token itself is expired. Anything else keeps the
        // existing mismatch refusal below.
        //
        // A live setup process still holds the old token+expiry in memory
        // (captured at process start), so the retained generation stops
        // first and the replacement process captures the fresh token at
        // start. The swap is atomic: the old token survives every fallible
        // step until the final rename, and a holder that will not stop
        // refuses with the old credential untouched.
        let mut rotated = false;
        if r.action == "begin" && token_present {
            if let Ok(token) = read_at::<Token>(&dirs.control_dir, "token.json", 10000) {
                if token.operation == s.id
                    && s.expires_at > token.expires_at
                    && now()? >= token.expires_at
                {
                    self.stop_retained_setup(s, deadline)?;
                    // FJ-862: fence → reconcile → typed-outcomes gate. The
                    // holder above is stopped and re-inspected pinned to its
                    // exact container id (never assumed dead from store
                    // state); rotation proceeds only on the packaged
                    // reconcile's proven-unclaimed outcome. Committed (expiry
                    // never uncommits), conflict, and uncertain keep the old
                    // token: the original credential survives every refusal
                    // below, so a later retry converges instead of stranding
                    // the retained setup without any credential.
                    //
                    // Reservation honesty: the native journal advance in
                    // `paths` above and this fence are NOT the counted
                    // resource — the bounded renewal count lives in the
                    // Go-side renewal audit, which this path never writes
                    // (renewal stays store-staged; see the 862 release).
                    // A refused rotation may leave the native journal ahead
                    // of the database record; the advance is forward-only
                    // and reconverges on retry. Only the rotation below
                    // mints, and only after the custody re-check.
                    match self.reconcile_setup_claim(s, &dirs, deadline) {
                        ReconcileOutcome::Unclaimed => {}
                        ReconcileOutcome::Committed => {
                            return Err(
                                "setup claim is committed; renewal refused with the retained credential kept"
                                    .into(),
                            )
                        }
                        ReconcileOutcome::Conflict => {
                            return Err(
                                "setup claim conflicts; renewal refused with the retained credential kept"
                                    .into(),
                            )
                        }
                        ReconcileOutcome::Uncertain => {
                            return Err(
                                "setup claim is unconfirmed; renewal refused with the retained credential kept"
                                    .into(),
                            )
                        }
                    }
                    // FJ-862: condition the rotation on the still-valid
                    // fence. The reconcile container held the Canlang
                    // writer lock while it ran, but that custody ends when
                    // the container is disposed — before this rotation. So
                    // the rotation re-proves the fence instead of assuming
                    // the verdict is still fresh: the token file must still
                    // hold the exact fenced bytes, no holder may have
                    // reappeared under the setup name, and the control dir
                    // must still be the validated descriptor. Anything else
                    // refuses with the retained credential kept — the
                    // outcome is unconfirmable, never a permit.
                    let fence_intact: bool =
                        read_at::<Token>(&dirs.control_dir, "token.json", 10000).is_ok_and(
                            |current| {
                                current.operation == token.operation
                                    && current.token == token.token
                                    && current.expires_at == token.expires_at
                            },
                        );
                    if !fence_intact
                        || super::home::fd_matches_path(&dirs.control, &dirs.control_dir).is_err()
                    {
                        return Err(
                            "setup claim is unconfirmed; renewal refused with the retained credential kept"
                                .into(),
                        );
                    }
                    // FJ-874: running-holder fence. Default `ps` lists
                    // running containers only: a correctly stopped retained
                    // container is absent here and must NOT refuse an
                    // unclaimed rotation, while a genuinely running holder
                    // still fences with the retained credential kept.
                    let live = self.command(
                        "podman",
                        &[
                            "ps".into(),
                            "--no-trunc".into(),
                            format!("--filter=name=^soda-canlang-setup-{}$", s.id),
                            "--format={{.ID}}".into(),
                        ],
                        deadline,
                    )?;
                    if running_holder_listed(&live) {
                        return Err(
                            "setup claim is unconfirmed; renewal refused with the retained credential kept"
                                .into(),
                        );
                    }
                    let mut bytes = [0u8; 32];
                    getrandom::fill(&mut bytes).map_err(|_| "setup randomness unavailable")?;
                    replace_at(
                        &dirs.control_dir,
                        "token.json",
                        &Token {
                            operation: s.id.clone(),
                            token: crate::sha256::hex_lower(&bytes),
                            expires_at: s.expires_at,
                        },
                        10000,
                    )?;
                    rotated = true;
                }
            }
        }
        if r.action == "begin" && !token_present {
            // FJ-862: ordinary first setup has no token yet and must
            // progress WITHOUT a pre-existing claim — so this path never
            // invokes the packaged reconcile. The binary REQUIRES the
            // token file (it refuses a missing token), so calling it here
            // could only refuse; first use mints directly when the
            // namespace is genuinely empty. Missing credentials on an
            // existing generation/data tree stay an uncertain prior claim,
            // never authority to mint a replacement. A just-rotated
            // renewal token is the exception: its prior claim was certain,
            // so the retained generation is reused below.
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
            // FJ-847: the data tree is inspected through the anchored data
            // parent and rebound by dev/ino — a swapped path fails closed.
            let data = dirs.root.join("data").join(&s.application.data);
            let data_nonempty = match super::home::open_child_dir_opt(
                &dirs.root_anchor,
                "data",
                "setup data unavailable",
            )? {
                Some(data_parent) => {
                    match super::home::open_child_dir_opt(
                        &data_parent,
                        &s.application.data,
                        "setup data unavailable",
                    )? {
                        Some(app) => {
                            super::home::fd_matches_path(&data, &app).map_err(|_| {
                                "setup credentials are unavailable; inspect or cancel the retained setup"
                                    .to_string()
                            })?;
                            fs::read_dir(&data)
                                .map_err(|_| "setup data unavailable")?
                                .any(|entry| entry.is_ok_and(|e| e.file_name() != ".writer-lock"))
                        }
                        None => false,
                    }
                }
                None => false,
            };
            if !rotated && (!existing.iter().all(u8::is_ascii_whitespace) || data_nonempty) {
                return Err(
                    "setup credentials are unavailable; inspect or cancel the retained setup"
                        .into(),
                );
            }
            // FJ-862: a stillborn credential is never minted — an
            // already-expired reservation cannot begin, with or without a
            // holder. The mint below (non-replacing publish) only runs on a
            // live reservation with no holder and no retained data.
            if now()? >= s.expires_at {
                return Err("setup token expired".into());
            };
            let mut bytes = [0u8; 32];
            getrandom::fill(&mut bytes).map_err(|_| "setup randomness unavailable")?;
            save_at(
                &dirs.control_dir,
                "token.json",
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
                    if r.action == "abort"
                        && super::home::open_child_opt(
                            &dirs.control_dir,
                            "token.json",
                            false,
                            "setup record unavailable",
                        )?
                        .is_some()
                    {
                        super::home::unlink_child(
                            &dirs.control_dir,
                            "token.json",
                            "setup token cannot be retired",
                        )?;
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
                let created = self.setup_create(s, &dirs, deadline)?;
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
            let token: Token = read_at(&dirs.control_dir, "token.json", 10000)?;
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
                let receipt: Receipt = read_at(&dirs.control_dir, "receipt.json", 10000)?;
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
            // FJ-847: guarded cleanup through the trusted descriptor — a
            // swapped token path refuses instead of removing foreign bytes.
            if super::home::open_child_opt(
                &dirs.control_dir,
                "token.json",
                false,
                "setup record unavailable",
            )?
            .is_some()
            {
                super::home::unlink_child(
                    &dirs.control_dir,
                    "token.json",
                    "setup token cannot be retired",
                )?;
                super::home::fsync_fd(&dirs.control_dir, "setup token retirement failed")?;
            };
            return Ok(result);
        }
        result.observation.ready = c.running && self.setup_ready(s, deadline);
        Ok(result)
    }
}
