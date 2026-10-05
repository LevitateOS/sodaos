//! Supervised factory-run execution (PR26).
//!
//! Port of `internal/host/project/{factory,factory_export,factory_output}.go`
//! and the `Factory` half of `factory_candidate.go` (`InspectCandidate`),
//! plus the factory/prepare/candidate slice of the top-level facades in
//! `internal/host/{factory_client,factory_candidate,prepare}.go`.
//!
//! The daemon side is a [`Factory`] over three seams: an [`Executor`] for
//! the podman observation in candidate inspection, a [`FactoryTerminal`]
//! for the supervised native boundary (`internal/host/terminal`, ported
//! separately) and a [`FactoryBroker`] for the identity broker client.
//! Receipt bytes, state mapping, phase machine, reason strings and error
//! strings match the Go implementation; only transport failures (I/O error
//! text, never wire-visible: the daemon maps them to a generic 500) follow
//! Rust formatting.
//!
//! Deadline strings are RFC3339Nano as emitted by Go's `time.Time` JSON
//! encoding. They are stored and re-emitted verbatim, so receipt bytes stay
//! exact; a parsed epoch-nanos copy drives the deadline comparisons. The
//! validator accepts the canonical shape Go emits (`YYYY-MM-DDTHH:MM:SS`
//! with an optional 1-9 digit fraction and a `Z` or numeric offset); exotic
//! inputs Go's own parser might accept or reject beyond that shape are out
//! of scope.
//!
//! Later route layers map [`FactoryError`] variants to statuses exactly
//! like `daemon.go`: `NotFound` to 404, `Stale` to 409, `ExportCandidate`
//! to 422, `ExportBounds` to 413, everything else to a generic 500.

use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use crate::domain;
use crate::json::{self, BoundMap, Kind, Spec, Value};
use crate::preparation::{
    self, HoldState, Preparation, PrepareHold, PrepareInspect, PrepareState, PrepareStop,
};
use crate::project::{Config, Executor, Runtime};

/// Daemon receipt directory for supervised factory runs.
pub const FACTORY_STATE_ROOT: &str = "/var/lib/soda/host/factory";

/// First harness family.
pub const FACTORY_HARNESS_CODEX: &str = "codex";
/// Muse Code CLI harness family.
pub const FACTORY_HARNESS_MUSE: &str = "muse";

/// Supported supervised CLI families.
pub fn valid_harness_family(family: &str) -> bool {
    family == FACTORY_HARNESS_CODEX || family == FACTORY_HARNESS_MUSE
}

pub const FACTORY_APPROVED: &str = "approved";
pub const FACTORY_RUNNING: &str = "running";
pub const FACTORY_COMPLETED: &str = "completed";
pub const FACTORY_FAILED: &str = "failed";
pub const FACTORY_STOPPED: &str = "stopped";
pub const FACTORY_UNCERTAIN: &str = "uncertain";

pub const MAX_FACTORY_PROMPT: usize = 64 * 1024;
pub const MAX_FACTORY_OUTPUT: usize = 64 * 1024;
pub const MAX_FACTORY_OUTPUT_READ: i64 = 24 * 1024 - 256;
pub const MAX_FACTORY_OUTPUT_WINDOW: i64 = 256 * 1024;
pub const MAX_FACTORY_OUTPUT_OFFSET: i64 = 256 << 20;
pub const MAX_FACTORY_EXPORT_BUNDLE: usize = 4 << 20;

/// Broker kind for supervised factory executions.
pub const IDENTITY_FACTORY: &str = "factory";
/// Broker provider for supervised factory executions.
pub const IDENTITY_CODEX: &str = "codex";
/// Terminal broker execution state meaning custody settled elsewhere.
pub const EXECUTION_TERMINAL: &str = "terminal";

/// Client-side unknown-run error (`host.ErrRunNotFound`).
pub const ERR_RUN_NOT_FOUND: &str = "factory run not found";
/// Client-side incarnation-changed error (`host.ErrRunStale`).
pub const ERR_RUN_STALE: &str = "factory run incarnation changed";

/// Retry bound for post-expiry retirement, mirroring
/// `factoryCleanupTimeout`: stop, credential custody and receipt work run
/// under a fresh 60s deadline detached from the expired operation parent.
const FACTORY_CLEANUP_SECS: u64 = 60;
/// Bound for one liveness probe, mirroring the 10s live contexts.
const FACTORY_LIVE_SECS: u64 = 10;
/// Supervised run bound: deadlines must fall within three hours.
const FACTORY_DEADLINE_BOUND_NANOS: i128 = 3 * 3600 * NANOS_PER_SEC;

pub fn valid_factory_phase(phase: &str) -> bool {
    matches!(
        phase,
        "approved" | "running" | "completed" | "failed" | "stopped" | "uncertain"
    )
}

/// `^[a-f0-9]{32}$`
pub fn valid_factory_run_id(id: &str) -> bool {
    id.len() == 32
        && id
            .bytes()
            .all(|b| b.is_ascii_digit() || (b.is_ascii_hexdigit() && b.is_ascii_lowercase()))
}

/// `^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`
pub fn valid_harness_version(version: &str) -> bool {
    let b = version.as_bytes();
    !b.is_empty()
        && b.len() <= 32
        && b[0].is_ascii_alphanumeric()
        && b[1..]
            .iter()
            .all(|c| c.is_ascii_alphanumeric() || matches!(*c, b'.' | b'_' | b'-'))
}

/// `FactoryUnitName`: the transient host unit supervising one run.
pub fn factory_unit_name(run: &str) -> String {
    if !valid_factory_run_id(run) {
        return String::new();
    }
    format!("soda-factory-{run}.service")
}

/// `FactoryRunPaths`: fixed container paths for one run. Empty strings
/// report an invalid identity.
pub fn factory_run_paths(
    role: &str,
    preparation: &str,
    run: &str,
) -> (String, String, String, String) {
    if !preparation::valid_factory_role(role)
        || !preparation::valid_preparation_id(preparation)
        || !valid_factory_run_id(run)
    {
        return (String::new(), String::new(), String::new(), String::new());
    }
    let checkout = format!("/home/{role}/checkouts/{preparation}");
    let run_dir = format!("{checkout}/.soda-home/runs/{run}");
    let home = format!("{run_dir}/home");
    let codex_home = format!("{home}/.codex");
    (checkout, run_dir, home, codex_home)
}

/// `TakeoverDestination`: member-owned checkout destination for one run.
pub fn takeover_destination(member: &str, run: &str) -> String {
    if !domain::valid_login(member) || member == "root" || !valid_factory_run_id(run) {
        return String::new();
    }
    format!("/home/{member}/factory-takeover/{run}")
}

/// `TakeoverSource`: only the exact fixed role-checkout layout validates.
pub fn takeover_source(path: &str, role: &str, preparation: &str) -> bool {
    if !preparation::valid_factory_role(role) || !preparation::valid_preparation_id(preparation) {
        return false;
    }
    path == format!("/home/{role}/checkouts/{preparation}")
}

// ---------- errors ----------

/// Daemon-side factory error. Sentinel variants mirror the `identity`
/// package sentinels plus the export verdicts so the later route layer can
/// map them to statuses; `Msg` carries exact Go error text otherwise.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FactoryError {
    NotFound,
    Stale,
    Busy,
    Denied,
    Uncertain,
    DeadlineExceeded,
    ExportCandidate,
    ExportBounds,
    Msg(String),
}

impl FactoryError {
    /// Exact Go error text for this failure.
    pub fn message(&self) -> String {
        match self {
            FactoryError::NotFound => "identity execution missing".to_string(),
            FactoryError::Stale => "identity generation changed".to_string(),
            FactoryError::Busy => "subscription is in use".to_string(),
            FactoryError::Denied => "identity authority denied".to_string(),
            FactoryError::Uncertain => "subscription requires reconnection".to_string(),
            FactoryError::DeadlineExceeded => "context deadline exceeded".to_string(),
            FactoryError::ExportCandidate => "export candidate is not recorded".to_string(),
            FactoryError::ExportBounds => "candidate export exceeds bounds".to_string(),
            FactoryError::Msg(s) => s.clone(),
        }
    }

    fn msg(text: impl Into<String>) -> Self {
        FactoryError::Msg(text.into())
    }
}

impl std::fmt::Display for FactoryError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.message())
    }
}

impl std::error::Error for FactoryError {}

// ---------- RFC3339Nano deadlines ----------

const NANOS_PER_SEC: i128 = 1_000_000_000;

fn is_leap_year(y: i64) -> bool {
    y % 4 == 0 && (y % 100 != 0 || y % 400 == 0)
}

fn days_in_month(y: i64, m: i64) -> i64 {
    match m {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if is_leap_year(y) => 29,
        2 => 28,
        _ => 0,
    }
}

/// Days from civil date to Unix epoch (Howard Hinnant's algorithm).
fn days_from_civil(y: i64, m: i64, d: i64) -> i64 {
    let y = if m <= 2 { y - 1 } else { y };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = y - era * 400;
    let mp = (m + 9) % 12;
    let doy = (153 * mp + 2) / 5 + d - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468
}

fn two_digits(b: &[u8]) -> Option<i64> {
    if b.len() == 2 && b[0].is_ascii_digit() && b[1].is_ascii_digit() {
        Some(((b[0] - b'0') * 10 + (b[1] - b'0')) as i64)
    } else {
        None
    }
}

/// Parse a Go `time.Time` JSON value to epoch nanos. Accepts the canonical
/// shape Go emits: `YYYY-MM-DDTHH:MM:SS` with an optional 1-9 digit
/// fraction and a `Z` or numeric offset.
fn parse_deadline(s: &str) -> Option<i128> {
    let b = s.as_bytes();
    if b.len() < 20
        || b[4] != b'-'
        || b[7] != b'-'
        || b[10] != b'T'
        || b[13] != b':'
        || b[16] != b':'
    {
        return None;
    }
    if !b[0..4].iter().all(|c| c.is_ascii_digit()) {
        return None;
    }
    let year: i64 = s[0..4].parse().ok()?;
    if !(1..=9999).contains(&year) {
        return None;
    }
    let month = two_digits(&b[5..7])?;
    let day = two_digits(&b[8..10])?;
    let hour = two_digits(&b[11..13])?;
    let minute = two_digits(&b[14..16])?;
    let second = two_digits(&b[17..19])?;
    if !(1..=12).contains(&month) || !(1..=days_in_month(year, month)).contains(&day) {
        return None;
    }
    if hour > 23 || minute > 59 || second > 59 {
        return None;
    }
    let mut rest = &b[19..];
    let mut frac_nanos: i128 = 0;
    if rest.first() == Some(&b'.') {
        rest = &rest[1..];
        let mut digits = 0;
        let mut scale: i128 = 100_000_000;
        while rest.first().is_some_and(|c| c.is_ascii_digit()) {
            if digits >= 9 {
                return None;
            }
            frac_nanos += ((rest[0] - b'0') as i128) * scale;
            scale /= 10;
            digits += 1;
            rest = &rest[1..];
        }
        if digits == 0 {
            return None;
        }
    }
    let offset_secs: i128 = if rest == b"Z" {
        0
    } else if rest.len() == 6 && (rest[0] == b'+' || rest[0] == b'-') && rest[3] == b':' {
        let oh = two_digits(&rest[1..3])?;
        let om = two_digits(&rest[4..6])?;
        if oh > 23 || om > 59 {
            return None;
        }
        let sign = if rest[0] == b'-' { -1 } else { 1 };
        sign * (oh as i128 * 3600 + om as i128 * 60)
    } else {
        return None;
    };
    let days = days_from_civil(year, month, day) as i128;
    Some(
        (days * 86400 + hour as i128 * 3600 + minute as i128 * 60 + second as i128) * NANOS_PER_SEC
            + frac_nanos
            - offset_secs * NANOS_PER_SEC,
    )
}

/// Current wall-clock time as Unix epoch nanos.
fn system_nanos_now() -> i128 {
    match SystemTime::now().duration_since(UNIX_EPOCH) {
        Ok(d) => d.as_nanos() as i128,
        Err(e) => -(e.duration().as_nanos() as i128),
    }
}

/// Go zero time (`0001-01-01T00:00:00Z`) has Unix epoch nanos of the
/// proleptic civil date; anything else, malformed included, is nonzero or
/// unusable. Callers treat `None` (malformed) like a missing deadline: the
/// route decoder owns malformed-deadline reports, the factory never sees
/// them over the wire.
fn deadline_is_zero(text: &str) -> bool {
    match parse_deadline(text) {
        Some(nanos) => nanos == days_from_civil(1, 1, 1) as i128 * 86400 * NANOS_PER_SEC,
        None => false,
    }
}

/// Truncated seconds from now until the deadline, like
/// `int64(time.Until(deadline).Seconds())`.
fn seconds_until(deadline_nanos: i128) -> i64 {
    ((deadline_nanos - system_nanos_now()) / NANOS_PER_SEC)
        .clamp(i64::MIN as i128, i64::MAX as i128) as i64
}

/// Operation deadline capped by the run deadline, like
/// `context.WithDeadline(ctx, run.Deadline)`.
fn drive_deadline(op: Instant, deadline_nanos: i128) -> Instant {
    let delta = deadline_nanos - system_nanos_now();
    if delta <= 0 {
        return Instant::now();
    }
    let wait = Duration::from_nanos(delta.min(u64::MAX as i128) as u64);
    Instant::now().checked_add(wait).unwrap_or(op).min(op)
}

fn min_instant(a: Instant, b: Instant) -> Instant {
    if a < b {
        a
    } else {
        b
    }
}

fn cleanup_deadline() -> Instant {
    Instant::now() + Duration::from_secs(FACTORY_CLEANUP_SECS)
}

fn live_deadline(op: Instant) -> Instant {
    min_instant(op, Instant::now() + Duration::from_secs(FACTORY_LIVE_SECS))
}

// ---------- run, lease and binding records ----------

/// `project.FactoryRun`: one supervised CLI execution identity. The
/// deadline keeps its verbatim Go `time.Time` JSON text so receipt bytes
/// round-trip exactly.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryRun {
    pub deadline: String,
    pub actor: i64,
    pub id: String,
    pub project: String,
    pub role: String,
    pub preparation: String,
    pub harness: String,
    pub harness_vers: String,
    pub model: String,
    pub assignment: String,
    pub source_commit: String,
    pub connection: String,
}

impl FactoryRun {
    pub fn validate(&self) -> Result<(), String> {
        if !valid_factory_run_id(&self.id)
            || !domain::valid_id(&self.project)
            || !preparation::valid_factory_role(&self.role)
        {
            return Err("invalid factory run identity".to_string());
        }
        if !preparation::valid_preparation_id(&self.preparation) {
            return Err("invalid run preparation reference".to_string());
        }
        if !valid_harness_family(&self.harness) || !valid_harness_version(&self.harness_vers) {
            return Err("unsupported factory harness".to_string());
        }
        if !self.model.is_empty() {
            if self.model.len() > 128 {
                return Err("invalid run model selection".to_string());
            }
            for b in self.model.bytes() {
                if b < 0x20 || b == 0x7f {
                    return Err("invalid run model selection".to_string());
                }
            }
        }
        if !preparation::valid_digest(&self.assignment)
            || !preparation::valid_commit(&self.source_commit)
        {
            return Err("invalid run assignment or source identity".to_string());
        }
        if self.connection.is_empty() || self.connection.len() > 128 || self.actor <= 0 {
            return Err("invalid run sponsorship".to_string());
        }
        if self.deadline.is_empty()
            || parse_deadline(&self.deadline).is_none()
            || deadline_is_zero(&self.deadline)
        {
            return Err("run deadline is required".to_string());
        }
        Ok(())
    }

    /// `encoding/json` field order with `model,omitempty` honored.
    pub fn encode_into(&self, out: &mut String) {
        out.push_str("{\"deadline\":");
        out.push_str(&json::quote(&self.deadline));
        out.push_str(",\"actor\":");
        out.push_str(&self.actor.to_string());
        out.push_str(",\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"role\":");
        out.push_str(&json::quote(&self.role));
        out.push_str(",\"preparation\":");
        out.push_str(&json::quote(&self.preparation));
        out.push_str(",\"harness\":");
        out.push_str(&json::quote(&self.harness));
        out.push_str(",\"harness_version\":");
        out.push_str(&json::quote(&self.harness_vers));
        if !self.model.is_empty() {
            out.push_str(",\"model\":");
            out.push_str(&json::quote(&self.model));
        }
        out.push_str(",\"assignment\":");
        out.push_str(&json::quote(&self.assignment));
        out.push_str(",\"source_commit\":");
        out.push_str(&json::quote(&self.source_commit));
        out.push_str(",\"connection\":");
        out.push_str(&json::quote(&self.connection));
        out.push('}');
    }

    pub fn from_map(m: &BoundMap) -> Self {
        FactoryRun {
            deadline: m.take_string("deadline"),
            actor: m.take_i64("actor"),
            id: m.take_string("id"),
            project: m.take_string("project"),
            role: m.take_string("role"),
            preparation: m.take_string("preparation"),
            harness: m.take_string("harness"),
            harness_vers: m.take_string("harness_version"),
            model: m.take_string("model"),
            assignment: m.take_string("assignment"),
            source_commit: m.take_string("source_commit"),
            connection: m.take_string("connection"),
        }
    }
}

const FACTORY_RUN_SPECS: &[Spec] = &[
    Spec {
        name: "deadline",
        kind: Kind::Str,
    },
    Spec {
        name: "actor",
        kind: Kind::I64,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "preparation",
        kind: Kind::Str,
    },
    Spec {
        name: "harness",
        kind: Kind::Str,
    },
    Spec {
        name: "harness_version",
        kind: Kind::Str,
    },
    Spec {
        name: "model",
        kind: Kind::Str,
    },
    Spec {
        name: "assignment",
        kind: Kind::Str,
    },
    Spec {
        name: "source_commit",
        kind: Kind::Str,
    },
    Spec {
        name: "connection",
        kind: Kind::Str,
    },
];

/// `project.FactoryLaunch`: the host run request.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryLaunch {
    pub run: FactoryRun,
    pub prompt: Vec<u8>,
    pub harness_sha256: String,
}

const FACTORY_LAUNCH_SPECS: &[Spec] = &[
    Spec {
        name: "run",
        kind: Kind::Object {
            go_type: "project.FactoryRun",
            struct_name: "FactoryRun",
            specs: FACTORY_RUN_SPECS,
        },
    },
    Spec {
        name: "prompt",
        kind: Kind::Bytes,
    },
    Spec {
        name: "harness_sha256",
        kind: Kind::Str,
    },
];

impl FactoryLaunch {
    /// Strict decode of one launch request (`strictjson.Decode` parity).
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m =
            json::bind_root(v, "FactoryLaunch", FACTORY_LAUNCH_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        FactoryLaunch {
            run: FactoryRun::from_map(&m.take_map("run")),
            prompt: m.take_bytes("prompt"),
            harness_sha256: m.take_string("harness_sha256"),
        }
    }
}

impl FactoryLaunch {
    pub fn validate(&self) -> Result<(), String> {
        self.run.validate()?;
        if self.prompt.is_empty() || self.prompt.len() > MAX_FACTORY_PROMPT {
            return Err("invalid run prompt size".to_string());
        }
        if crate::sha256::hex_lower(&crate::sha256::digest(&self.prompt)) != self.run.assignment {
            return Err("prompt bytes do not match their digest".to_string());
        }
        if !preparation::valid_digest(&self.harness_sha256) {
            return Err("invalid harness pin".to_string());
        }
        Ok(())
    }
}

/// `identity.Binding`: the recorded native process boundary.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Binding {
    pub child_id: String,
    pub uid: i64,
    pub gid: i64,
    pub scope: String,
    pub credential_root: String,
    pub invocation_id: String,
    pub kind: String,
    pub id: String,
    pub project: String,
    pub login: String,
    pub generation: i64,
}

impl Binding {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        let mut field = |out: &mut String, name: &str, value: &str| {
            if !first {
                out.push(',');
            }
            first = false;
            out.push_str(&json::quote(name));
            out.push(':');
            out.push_str(value);
        };
        if !self.child_id.is_empty() {
            field(out, "child_id", &json::quote(&self.child_id));
        }
        if self.uid != 0 {
            field(out, "uid", &self.uid.to_string());
        }
        if self.gid != 0 {
            field(out, "gid", &self.gid.to_string());
        }
        if !self.scope.is_empty() {
            field(out, "scope", &json::quote(&self.scope));
        }
        if !self.credential_root.is_empty() {
            field(out, "credential_root", &json::quote(&self.credential_root));
        }
        if !self.invocation_id.is_empty() {
            field(out, "invocation_id", &json::quote(&self.invocation_id));
        }
        field(out, "kind", &json::quote(&self.kind));
        field(out, "id", &json::quote(&self.id));
        field(out, "project", &json::quote(&self.project));
        field(out, "login", &json::quote(&self.login));
        field(out, "generation", &self.generation.to_string());
        out.push('}');
    }

    pub fn from_map(m: &BoundMap) -> Self {
        Binding {
            child_id: m.take_string("child_id"),
            uid: m.take_i64("uid"),
            gid: m.take_i64("gid"),
            scope: m.take_string("scope"),
            credential_root: m.take_string("credential_root"),
            invocation_id: m.take_string("invocation_id"),
            kind: m.take_string("kind"),
            id: m.take_string("id"),
            project: m.take_string("project"),
            login: m.take_string("login"),
            generation: m.take_i64("generation"),
        }
    }
}

const BINDING_SPECS: &[Spec] = &[
    Spec {
        name: "child_id",
        kind: Kind::Str,
    },
    Spec {
        name: "uid",
        kind: Kind::Int,
    },
    Spec {
        name: "gid",
        kind: Kind::Int,
    },
    Spec {
        name: "scope",
        kind: Kind::Str,
    },
    Spec {
        name: "credential_root",
        kind: Kind::Str,
    },
    Spec {
        name: "invocation_id",
        kind: Kind::Str,
    },
    Spec {
        name: "kind",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "generation",
        kind: Kind::I64,
    },
];

/// `identity.Lease`: the broker execution identity for one run.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Lease {
    pub repository_id: i64,
    pub provider_id: String,
    pub id: String,
    pub connection_id: String,
    pub generation: i64,
    pub actor_id: i64,
    pub project_id: String,
    pub execution_id: String,
    pub kind: String,
    pub role: String,
    pub deadline: String,
    pub grant_id: String,
    pub grant_revision: i64,
    pub binding: Option<Binding>,
}

impl Lease {
    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
        let mut first = true;
        let mut field = |out: &mut String, name: &str, value: &str| {
            if !first {
                out.push(',');
            }
            first = false;
            out.push_str(&json::quote(name));
            out.push(':');
            out.push_str(value);
        };
        if self.repository_id != 0 {
            field(
                out,
                "repository_id",
                &json::quote(&self.repository_id.to_string()),
            );
        }
        field(out, "provider_id", &json::quote(&self.provider_id));
        field(out, "id", &json::quote(&self.id));
        field(out, "connection_id", &json::quote(&self.connection_id));
        field(out, "generation", &self.generation.to_string());
        field(out, "actor_id", &json::quote(&self.actor_id.to_string()));
        field(out, "project_id", &json::quote(&self.project_id));
        field(out, "execution_id", &json::quote(&self.execution_id));
        field(out, "kind", &json::quote(&self.kind));
        if !self.role.is_empty() {
            field(out, "role", &json::quote(&self.role));
        }
        field(out, "deadline", &json::quote(&self.deadline));
        if !self.grant_id.is_empty() {
            field(out, "grant_id", &json::quote(&self.grant_id));
        }
        if self.grant_revision != 0 {
            field(out, "grant_revision", &self.grant_revision.to_string());
        }
        if let Some(b) = &self.binding {
            if !first {
                out.push(',');
            }
            out.push_str("\"binding\":");
            b.encode_into(out);
        }
        out.push('}');
    }

    pub fn from_map(m: &BoundMap) -> Result<Self, FactoryError> {
        const ERR: &str = "invalid run receipt";
        // `,string` decimals: missing and null bind zero like Go; a present
        // value must be a quoted decimal.
        let string_int = |name: &str| -> Result<i64, FactoryError> {
            if !m.contains(name) {
                return Ok(0);
            }
            json::parse_go_int64(&m.take_string(name)).ok_or_else(|| FactoryError::msg(ERR))
        };
        Ok(Lease {
            repository_id: string_int("repository_id")?,
            provider_id: m.take_string("provider_id"),
            id: m.take_string("id"),
            connection_id: m.take_string("connection_id"),
            generation: m.take_i64("generation"),
            actor_id: string_int("actor_id")?,
            project_id: m.take_string("project_id"),
            execution_id: m.take_string("execution_id"),
            kind: m.take_string("kind"),
            role: m.take_string("role"),
            deadline: m.take_string("deadline"),
            grant_id: m.take_string("grant_id"),
            grant_revision: m.take_i64("grant_revision"),
            binding: m.take_opt_map("binding").map(|b| Binding::from_map(&b)),
        })
    }

    /// `leaseWithBinding`: the lease carrying its recorded binding for one
    /// native call.
    pub fn with_binding(&self, binding: &Binding) -> Self {
        let mut leased = self.clone();
        leased.binding = Some(binding.clone());
        leased
    }
}

const LEASE_SPECS: &[Spec] = &[
    Spec {
        name: "repository_id",
        kind: Kind::Str,
    },
    Spec {
        name: "provider_id",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "connection_id",
        kind: Kind::Str,
    },
    Spec {
        name: "generation",
        kind: Kind::I64,
    },
    Spec {
        name: "actor_id",
        kind: Kind::Str,
    },
    Spec {
        name: "project_id",
        kind: Kind::Str,
    },
    Spec {
        name: "execution_id",
        kind: Kind::Str,
    },
    Spec {
        name: "kind",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "deadline",
        kind: Kind::Str,
    },
    Spec {
        name: "grant_id",
        kind: Kind::Str,
    },
    Spec {
        name: "grant_revision",
        kind: Kind::I64,
    },
    Spec {
        name: "binding",
        kind: Kind::OptObject {
            go_type: "*identity.Binding",
            struct_name: "Binding",
            specs: BINDING_SPECS,
        },
    },
];

/// `identity.AcquireRequest` as built by `acquireRunLease` (the repository
/// field stays unset on this path, so it is not mirrored).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AcquireRequest {
    pub provider_id: String,
    pub execution_id: String,
    pub actor_id: i64,
    pub connection_id: String,
    pub project_id: String,
    pub kind: String,
    pub deadline: String,
    pub role: String,
}

/// One observed output slice with its cursor placement, mirroring
/// `terminal.FactoryCodexOutputSlice`.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct OutputSlice {
    pub data: Vec<u8>,
    pub total: i64,
    pub offset: i64,
    pub truncated: bool,
    pub gap: bool,
}

// ---------- request and response records ----------

/// `project.FactoryState`: the observed run outcome for one identity.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryState {
    pub exit_code: Option<i64>,
    pub generation: i64,
    pub uid: i64,
    pub gid: i64,
    pub credential_returned: bool,
    pub live: bool,
    pub delivered: bool,
    pub id: String,
    pub project: String,
    pub role: String,
    pub phase: String,
    pub container: String,
    pub unit: String,
    pub invocation: String,
    pub login: String,
    pub lease_id: String,
    pub output: String,
    pub retirement: String,
    pub reason: String,
}

impl FactoryState {
    /// `encoding/json` field order with `omitempty` honored.
    pub fn encode(&self) -> String {
        let mut out = String::from("{");
        let mut first = true;
        let mut field = |out: &mut String, name: &str, value: &str| {
            if !first {
                out.push(',');
            }
            first = false;
            out.push_str(&json::quote(name));
            out.push(':');
            out.push_str(value);
        };
        if let Some(code) = self.exit_code {
            field(&mut out, "exit_code", &code.to_string());
        }
        field(&mut out, "generation", &self.generation.to_string());
        field(&mut out, "uid", &self.uid.to_string());
        field(&mut out, "gid", &self.gid.to_string());
        field(
            &mut out,
            "credential_returned",
            if self.credential_returned {
                "true"
            } else {
                "false"
            },
        );
        field(&mut out, "live", if self.live { "true" } else { "false" });
        field(
            &mut out,
            "delivered",
            if self.delivered { "true" } else { "false" },
        );
        field(&mut out, "id", &json::quote(&self.id));
        field(&mut out, "project", &json::quote(&self.project));
        field(&mut out, "role", &json::quote(&self.role));
        field(&mut out, "phase", &json::quote(&self.phase));
        field(&mut out, "container", &json::quote(&self.container));
        field(&mut out, "unit", &json::quote(&self.unit));
        field(&mut out, "invocation", &json::quote(&self.invocation));
        field(&mut out, "login", &json::quote(&self.login));
        if !self.lease_id.is_empty() {
            field(&mut out, "lease_id", &json::quote(&self.lease_id));
        }
        if !self.output.is_empty() {
            field(&mut out, "output", &json::quote(&self.output));
        }
        if !self.retirement.is_empty() {
            field(&mut out, "retirement", &json::quote(&self.retirement));
        }
        if !self.reason.is_empty() {
            field(&mut out, "reason", &json::quote(&self.reason));
        }
        out.push('}');
        out
    }
}

/// `project.FactoryHarnessPin`: the staged-harness identity. The daemon
/// route fills `image` from its configuration after calling `harness_pin`,
/// exactly like the Go dispatch.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryHarnessPin {
    pub harness: String,
    pub version: String,
    pub sha256: String,
    pub image: String,
}

impl FactoryHarnessPin {
    pub fn validate(&self) -> Result<(), String> {
        if !valid_harness_family(&self.harness) || !valid_harness_version(&self.version) {
            return Err("unsupported factory harness".to_string());
        }
        if !preparation::valid_digest(&self.sha256) {
            return Err("invalid harness pin".to_string());
        }
        match self.image.strip_prefix("sha256:") {
            Some(value) if preparation::valid_digest(value) => Ok(()),
            _ => Err("execution image is not pinned".to_string()),
        }
    }

    pub fn encode(&self) -> String {
        let mut out = String::from("{\"harness\":");
        out.push_str(&json::quote(&self.harness));
        out.push_str(",\"version\":");
        out.push_str(&json::quote(&self.version));
        out.push_str(",\"sha256\":");
        out.push_str(&json::quote(&self.sha256));
        out.push_str(",\"image\":");
        out.push_str(&json::quote(&self.image));
        out.push('}');
        out
    }
}

/// `project.FactoryInspect`: address a run before or after completion.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryInspect {
    pub project: String,
    pub id: String,
}

const FACTORY_ADDRESS_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
];

impl FactoryInspect {
    /// Strict decode of one inspect request (`strictjson.Decode` parity).
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m =
            json::bind_root(v, "FactoryInspect", FACTORY_ADDRESS_SPECS, false).map_err(|e| e.0)?;
        Ok(FactoryInspect {
            project: m.take_string("project"),
            id: m.take_string("id"),
        })
    }
}

impl FactoryInspect {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory run address".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryStop`: retire one run identity.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryStop {
    pub project: String,
    pub id: String,
}

impl FactoryStop {
    /// Strict decode of one stop request (`strictjson.Decode` parity).
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "FactoryStop", FACTORY_ADDRESS_SPECS, false).map_err(|e| e.0)?;
        Ok(FactoryStop {
            project: m.take_string("project"),
            id: m.take_string("id"),
        })
    }
}

impl FactoryStop {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory run address".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryTakeover`: address one reconciled run and the admitted
/// member receiving its retained work.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryTakeover {
    pub project: String,
    pub id: String,
    pub member: String,
}

const FACTORY_TAKEOVER_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "member",
        kind: Kind::Str,
    },
];

impl FactoryTakeover {
    /// Strict decode of one takeover request (`strictjson.Decode` parity).
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "FactoryTakeover", FACTORY_TAKEOVER_SPECS, false)
            .map_err(|e| e.0)?;
        Ok(FactoryTakeover {
            project: m.take_string("project"),
            id: m.take_string("id"),
            member: m.take_string("member"),
        })
    }
}

impl FactoryTakeover {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory takeover address".to_string());
        }
        if !domain::valid_login(&self.member) || self.member == "root" {
            return Err("invalid takeover member".to_string());
        }
        Ok(())
    }
}

/// `project.TakeoverResult`: the confirmed takeover outcome.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TakeoverResult {
    pub id: String,
    pub project: String,
    pub member: String,
    pub destination: String,
    pub reused: bool,
}

impl TakeoverResult {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid takeover result identity".to_string());
        }
        if takeover_destination(&self.member, &self.id) != self.destination {
            return Err("takeover destination does not match its identities".to_string());
        }
        Ok(())
    }

    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"member\":");
        out.push_str(&json::quote(&self.member));
        out.push_str(",\"destination\":");
        out.push_str(&json::quote(&self.destination));
        out.push_str(",\"reused\":");
        out.push_str(if self.reused { "true" } else { "false" });
        out.push('}');
        out
    }
}

/// `project.FactoryOutput`: address one bounded output slice.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryOutput {
    pub project: String,
    pub id: String,
    pub offset: i64,
    pub limit: i64,
}

const FACTORY_OUTPUT_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "offset",
        kind: Kind::I64,
    },
    Spec {
        name: "limit",
        kind: Kind::I64,
    },
];

impl FactoryOutput {
    /// Strict decode of one output request (`strictjson.Decode` parity).
    /// Go's `limit` is `int` (64-bit); the range check lives in `validate`.
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m =
            json::bind_root(v, "FactoryOutput", FACTORY_OUTPUT_SPECS, false).map_err(|e| e.0)?;
        Ok(FactoryOutput {
            project: m.take_string("project"),
            id: m.take_string("id"),
            offset: m.take_i64("offset"),
            limit: m.take_i64("limit"),
        })
    }
}

impl FactoryOutput {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory run address".to_string());
        }
        if self.offset < 0 || self.offset > MAX_FACTORY_OUTPUT_OFFSET {
            return Err("invalid output cursor".to_string());
        }
        if self.limit < 1 || self.limit > MAX_FACTORY_OUTPUT_READ {
            return Err("invalid output read bound".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryOutputState`: one observed output slice.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryOutputState {
    pub exit_code: Option<i64>,
    pub live: bool,
    pub terminal: bool,
    pub truncated: bool,
    pub gap: bool,
    pub id: String,
    pub project: String,
    pub phase: String,
    pub container: String,
    pub unit: String,
    pub invocation: String,
    pub total: i64,
    pub offset: i64,
    pub next: i64,
    pub data: String,
    pub reason: String,
}

impl FactoryOutputState {
    pub fn encode(&self) -> String {
        let mut out = String::from("{");
        let mut first = true;
        let mut field = |out: &mut String, name: &str, value: &str| {
            if !first {
                out.push(',');
            }
            first = false;
            out.push_str(&json::quote(name));
            out.push(':');
            out.push_str(value);
        };
        if let Some(code) = self.exit_code {
            field(&mut out, "exit_code", &code.to_string());
        }
        field(&mut out, "live", if self.live { "true" } else { "false" });
        field(
            &mut out,
            "terminal",
            if self.terminal { "true" } else { "false" },
        );
        field(
            &mut out,
            "truncated",
            if self.truncated { "true" } else { "false" },
        );
        field(&mut out, "gap", if self.gap { "true" } else { "false" });
        field(&mut out, "id", &json::quote(&self.id));
        field(&mut out, "project", &json::quote(&self.project));
        field(&mut out, "phase", &json::quote(&self.phase));
        field(&mut out, "container", &json::quote(&self.container));
        field(&mut out, "unit", &json::quote(&self.unit));
        field(&mut out, "invocation", &json::quote(&self.invocation));
        field(&mut out, "total", &self.total.to_string());
        field(&mut out, "offset", &self.offset.to_string());
        field(&mut out, "next", &self.next.to_string());
        field(&mut out, "data", &json::quote(&self.data));
        if !self.reason.is_empty() {
            field(&mut out, "reason", &json::quote(&self.reason));
        }
        out.push('}');
        out
    }
}

/// `project.FactoryExport`: address one settled run's exact candidate.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryExport {
    pub project: String,
    pub id: String,
    pub role: String,
    pub preparation: String,
    pub candidate: String,
}

const FACTORY_EXPORT_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "preparation",
        kind: Kind::Str,
    },
    Spec {
        name: "candidate",
        kind: Kind::Str,
    },
];

impl FactoryExport {
    /// Strict decode of one export request (`strictjson.Decode` parity).
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m =
            json::bind_root(v, "FactoryExport", FACTORY_EXPORT_SPECS, false).map_err(|e| e.0)?;
        Ok(FactoryExport {
            project: m.take_string("project"),
            id: m.take_string("id"),
            role: m.take_string("role"),
            preparation: m.take_string("preparation"),
            candidate: m.take_string("candidate"),
        })
    }
}

impl FactoryExport {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory export address".to_string());
        }
        if !preparation::valid_factory_role(&self.role) {
            return Err("invalid factory export role".to_string());
        }
        if !preparation::valid_preparation_id(&self.preparation) {
            return Err("invalid factory export preparation".to_string());
        }
        if !preparation::valid_commit(&self.candidate) {
            return Err("invalid factory export candidate".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryExportState`: one observed candidate export.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryExportState {
    pub id: String,
    pub project: String,
    pub phase: String,
    pub container: String,
    pub candidate: String,
    pub bundle: String,
}

impl FactoryExportState {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"phase\":");
        out.push_str(&json::quote(&self.phase));
        out.push_str(",\"container\":");
        out.push_str(&json::quote(&self.container));
        out.push_str(",\"candidate\":");
        out.push_str(&json::quote(&self.candidate));
        out.push_str(",\"bundle\":");
        out.push_str(&json::quote(&self.bundle));
        out.push('}');
        out
    }
}

/// `project.FactoryCandidateInspect`: address a checkout through a settled
/// recorded run.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryCandidateInspect {
    pub project: String,
    pub id: String,
}

impl FactoryCandidateInspect {
    /// Strict decode of one candidate-inspect request (`strictjson.Decode` parity).
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "FactoryCandidateInspect", FACTORY_ADDRESS_SPECS, false)
            .map_err(|e| e.0)?;
        Ok(FactoryCandidateInspect {
            project: m.take_string("project"),
            id: m.take_string("id"),
        })
    }
}

impl FactoryCandidateInspect {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid candidate inspection address".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryCandidateState`: the observed checkout HEAD and dirtiness.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryCandidateState {
    pub id: String,
    pub project: String,
    pub container: String,
    pub candidate: String,
    pub dirty: bool,
}

impl FactoryCandidateState {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"container\":");
        out.push_str(&json::quote(&self.container));
        out.push_str(",\"candidate\":");
        out.push_str(&json::quote(&self.candidate));
        out.push_str(",\"dirty\":");
        out.push_str(if self.dirty { "true" } else { "false" });
        out.push('}');
        out
    }
}

// ---------- supervised-boundary seams ----------

/// Supervised native boundary: the `terminal.Service` methods the factory
/// orchestrator calls. Ported with the terminal lane; tests script fakes.
pub trait FactoryTerminal {
    fn harness_family(&self) -> String;
    fn harness_version(&self) -> String;
    fn harness_sha256(&self) -> String;
    fn reserve(
        &self,
        run: &FactoryRun,
        lease: &Lease,
        pin: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<Binding, FactoryError>;
    fn start(
        &self,
        lease: &Lease,
        credential: &[u8],
        prompt: &[u8],
        deadline: Instant,
    ) -> Result<(), FactoryError>;
    fn wait(&self, lease: &Lease, deadline: Instant) -> Result<(i64, String), FactoryError>;
    fn stop(&self, lease: &Lease, deadline: Instant) -> Result<(), FactoryError>;
    fn stop_unbound(&self, run: &FactoryRun, deadline: Instant) -> Result<(), FactoryError>;
    fn capture(&self, lease: &Lease, deadline: Instant) -> Result<Vec<u8>, FactoryError>;
    fn live(&self, binding: &Binding, deadline: Instant) -> bool;
    fn output(
        &self,
        project: &str,
        binding: &Binding,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<OutputSlice, FactoryError>;
    #[allow(clippy::too_many_arguments)]
    fn takeover_copy(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        member: &str,
        run: &str,
        deadline: Instant,
    ) -> Result<(String, bool), FactoryError>;
    fn export_bundle(
        &self,
        project: &str,
        recorded: &str,
        role: &str,
        preparation: &str,
        candidate: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, FactoryError>;
}

/// Identity broker: the `identityclient.Client` methods the factory
/// orchestrator calls. Reconciliation failures are ignored by every
/// caller, exactly like the Go `_ =` assignments, so reconcile reports
/// nothing.
pub trait FactoryBroker {
    fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, FactoryError>;
    fn register(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Vec<u8>, FactoryError>;
    fn return_lease(
        &self,
        lease_id: &str,
        binding: &Binding,
        credential: &[u8],
        deadline: Instant,
    ) -> Result<(), FactoryError>;
    fn reconcile_lease(&self, lease_id: &str, deadline: Instant);
    fn execution_is_terminal(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<bool, FactoryError>;
    fn close_execution(
        &self,
        kind: &str,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), FactoryError>;
}

/// Credential bytes that are wiped when they fall out of scope, mirroring
/// Go's `defer clear(...)`.
struct Secret(Vec<u8>);

impl Drop for Secret {
    fn drop(&mut self) {
        for b in self.0.iter_mut() {
            // Volatile so the wipe survives optimization.
            unsafe { std::ptr::write_volatile(b, 0) };
        }
    }
}

// ---------- durable run receipts ----------

#[derive(Debug, Clone, Default)]
struct FactoryReceipt {
    run: FactoryRun,
    lease: Option<Lease>,
    binding: Option<Binding>,
    exit_code: Option<i64>,
    generation: i64,
    phase: String,
    started: bool,
    delivered: bool,
    credential_returned: bool,
    output: String,
    retirement: String,
    reason: String,
}

impl FactoryReceipt {
    fn validate(&self) -> Result<(), FactoryError> {
        self.run.validate().map_err(FactoryError::msg)?;
        if !valid_factory_phase(&self.phase) {
            return Err(FactoryError::msg("invalid run phase"));
        }
        if let Some(lease) = &self.lease {
            if lease.execution_id != self.run.id || lease.kind != IDENTITY_FACTORY {
                return Err(FactoryError::msg("run lease identity mismatch"));
            }
        }
        if let Some(binding) = &self.binding {
            if binding.kind != IDENTITY_FACTORY || binding.id != self.run.id {
                return Err(FactoryError::msg("run binding identity mismatch"));
            }
        }
        if self.output.len() > MAX_FACTORY_OUTPUT + 1
            || self.reason.len() > 256
            || self.retirement.len() > 32
        {
            return Err(FactoryError::msg("run record exceeds bounds"));
        }
        Ok(())
    }

    fn lease_id(&self) -> &str {
        self.lease.as_ref().map(|l| l.id.as_str()).unwrap_or("")
    }

    /// `json.Marshal` field order with `omitempty` honored.
    fn encode(&self) -> String {
        let mut out = String::from("{\"run\":");
        self.run.encode_into(&mut out);
        if let Some(lease) = &self.lease {
            out.push_str(",\"lease\":");
            lease.encode_into(&mut out);
        }
        if let Some(binding) = &self.binding {
            out.push_str(",\"binding\":");
            binding.encode_into(&mut out);
        }
        if let Some(code) = self.exit_code {
            out.push_str(",\"exit_code\":");
            out.push_str(&code.to_string());
        }
        if self.generation != 0 {
            out.push_str(",\"generation\":");
            out.push_str(&self.generation.to_string());
        }
        out.push_str(",\"phase\":");
        out.push_str(&json::quote(&self.phase));
        out.push_str(",\"started\":");
        out.push_str(if self.started { "true" } else { "false" });
        out.push_str(",\"delivered\":");
        out.push_str(if self.delivered { "true" } else { "false" });
        out.push_str(",\"credential_returned\":");
        out.push_str(if self.credential_returned {
            "true"
        } else {
            "false"
        });
        if !self.output.is_empty() {
            out.push_str(",\"output\":");
            out.push_str(&json::quote(&self.output));
        }
        if !self.retirement.is_empty() {
            out.push_str(",\"retirement\":");
            out.push_str(&json::quote(&self.retirement));
        }
        if !self.reason.is_empty() {
            out.push_str(",\"reason\":");
            out.push_str(&json::quote(&self.reason));
        }
        out.push('}');
        out
    }
}

const RECEIPT_SPECS: &[Spec] = &[
    Spec {
        name: "run",
        kind: Kind::Object {
            go_type: "project.FactoryRun",
            struct_name: "FactoryRun",
            specs: FACTORY_RUN_SPECS,
        },
    },
    Spec {
        name: "lease",
        kind: Kind::OptObject {
            go_type: "*identity.Lease",
            struct_name: "Lease",
            specs: LEASE_SPECS,
        },
    },
    Spec {
        name: "binding",
        kind: Kind::OptObject {
            go_type: "*identity.Binding",
            struct_name: "Binding",
            specs: BINDING_SPECS,
        },
    },
    Spec {
        name: "exit_code",
        kind: Kind::OptInt,
    },
    Spec {
        name: "generation",
        kind: Kind::I64,
    },
    Spec {
        name: "phase",
        kind: Kind::Str,
    },
    Spec {
        name: "started",
        kind: Kind::Bool,
    },
    Spec {
        name: "delivered",
        kind: Kind::Bool,
    },
    Spec {
        name: "credential_returned",
        kind: Kind::Bool,
    },
    Spec {
        name: "output",
        kind: Kind::Str,
    },
    Spec {
        name: "retirement",
        kind: Kind::Str,
    },
    Spec {
        name: "reason",
        kind: Kind::Str,
    },
];

fn receipt_terminal(phase: &str) -> bool {
    matches!(phase, "completed" | "failed" | "stopped")
}

/// Retained last-message output is byte-truncated for the response. Go
/// slices bytes and lets `encoding/json` substitute U+FFFD per invalid
/// byte; Rust strings cannot hold the split sequence, so the lossy
/// conversion below may merge adjacent replacement characters at the cut.
/// Pure-ASCII output, the only shape tests use, is byte-identical.
fn truncate_output(output: &str) -> String {
    if output.len() <= MAX_FACTORY_OUTPUT {
        return output.to_string();
    }
    String::from_utf8_lossy(&output.as_bytes()[..MAX_FACTORY_OUTPUT]).into_owned()
}

fn receipt_state(receipt: &FactoryReceipt, live: bool) -> FactoryState {
    let mut state = FactoryState {
        generation: receipt.generation,
        credential_returned: receipt.credential_returned,
        live,
        delivered: receipt.delivered,
        id: receipt.run.id.clone(),
        project: receipt.run.project.clone(),
        role: receipt.run.role.clone(),
        phase: receipt.phase.clone(),
        container: String::new(),
        unit: factory_unit_name(&receipt.run.id),
        invocation: String::new(),
        login: String::new(),
        uid: 0,
        gid: 0,
        lease_id: receipt.lease_id().to_string(),
        output: truncate_output(&receipt.output),
        retirement: receipt.retirement.clone(),
        reason: receipt.reason.clone(),
        exit_code: receipt.exit_code,
    };
    if let Some(binding) = &receipt.binding {
        state.container = binding.project.clone();
        state.invocation = binding.invocation_id.clone();
        state.login = binding.login.clone();
        state.uid = binding.uid;
        state.gid = binding.gid;
    }
    state
}

/// Held per-run lock file. Closing releases the flock, exactly like Go.
struct RunLock {
    _file: std::fs::File,
}

// ---------- factory orchestrator ----------

/// Supervised factory-run orchestration over a private receipt directory.
/// Launch and stop never hold the per-run lock across broker or native
/// calls; takeover holds it across the copy to serialize duplicates.
pub struct Factory<E, T, B> {
    exec: E,
    terminal: T,
    broker: B,
    state_dir: String,
}

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    /// `OpenFactory`: wire run orchestration over a private receipt
    /// directory. The directory must already exist with private
    /// permissions. (Go also rejects nil dependencies; Rust's type
    /// system owns them, so there is no such error here.)
    pub fn open_factory(state_dir: &str, exec: E, terminal: T, broker: B) -> Result<Self, String> {
        if !state_dir.starts_with('/') {
            return Err("factory receipt directory must be absolute".to_string());
        }
        let info = std::fs::symlink_metadata(state_dir).map_err(|e| e.to_string())?;
        if !info.is_dir() || info.permissions().mode() & 0o777 & 0o077 != 0 {
            return Err("factory receipts must live in a private directory".to_string());
        }
        Ok(Factory {
            exec,
            terminal,
            broker,
            state_dir: state_dir.to_string(),
        })
    }

    /// `HarnessPin`: the staged-harness identity the executor admits.
    /// `image` stays empty here; the daemon route fills it from its
    /// configuration, exactly like the Go dispatch.
    pub fn harness_pin(&self) -> FactoryHarnessPin {
        FactoryHarnessPin {
            harness: self.terminal.harness_family(),
            version: self.terminal.harness_version(),
            sha256: self.terminal.harness_sha256(),
            image: String::new(),
        }
    }

    fn receipt_path(&self, project: &str, run: &str) -> String {
        format!("{}/{project}-{run}.json", self.state_dir)
    }

    fn lock_path(&self, project: &str, run: &str) -> String {
        format!("{}/{project}-{run}.lock", self.state_dir)
    }

    /// `lockRun`: per-run exclusive lock with deadline, mirroring
    /// `filelock.Acquire` (nonblocking flock retried every 25ms).
    fn lock_run(
        &self,
        project: &str,
        run: &str,
        deadline: Instant,
    ) -> Result<RunLock, FactoryError> {
        use std::os::unix::io::AsRawFd;
        if Instant::now() >= deadline {
            return Err(FactoryError::DeadlineExceeded);
        }
        let file = std::fs::OpenOptions::new()
            .create(true)
            .truncate(false)
            .read(true)
            .write(true)
            .mode(0o600)
            .open(self.lock_path(project, run))
            .map_err(|e| FactoryError::msg(e.to_string()))?;
        loop {
            // SAFETY: flock on an owned open fd with no timeout side effects.
            let rc = unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
            if rc == 0 {
                return Ok(RunLock { _file: file });
            }
            let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
            if errno != libc::EWOULDBLOCK && errno != libc::EINTR {
                return Err(FactoryError::msg(
                    std::io::Error::from_raw_os_error(errno).to_string(),
                ));
            }
            if Instant::now() >= deadline {
                return Err(FactoryError::DeadlineExceeded);
            }
            let remaining = deadline.saturating_duration_since(Instant::now());
            std::thread::sleep(remaining.min(Duration::from_millis(25)));
        }
    }

    /// `loadReceipt`: read and validate one durable run record. The `bool`
    /// reports existence; absent reads succeed with `false`. Decoding is
    /// strict like Go's `DisallowUnknownFields`; duplicate fields are
    /// rejected here while Go's decoder would take the last, but
    /// daemon-written receipts never contain any.
    fn load_receipt(
        &self,
        project: &str,
        run: &str,
    ) -> Result<(FactoryReceipt, bool), FactoryError> {
        let data = match std::fs::read(self.receipt_path(project, run)) {
            Ok(data) => data,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                return Ok((FactoryReceipt::default(), false));
            }
            Err(e) => return Err(FactoryError::msg(e.to_string())),
        };
        if data.len() > 128 << 10 {
            return Err(FactoryError::msg("run receipt exceeds bounds"));
        }
        let value =
            json::decode_strict(&data).map_err(|_| FactoryError::msg("invalid run receipt"))?;
        let bound = json::bind_root(&value, "factoryReceipt", RECEIPT_SPECS, false)
            .map_err(|_| FactoryError::msg("invalid run receipt"))?;
        let receipt = FactoryReceipt {
            run: FactoryRun::from_map(&bound.take_map("run")),
            lease: match bound.take_opt_map("lease") {
                Some(m) => Some(Lease::from_map(&m)?),
                None => None,
            },
            binding: bound.take_opt_map("binding").map(|m| Binding::from_map(&m)),
            exit_code: bound.take_opt_i64("exit_code"),
            generation: bound.take_i64("generation"),
            phase: bound.take_string("phase"),
            started: bound.take_bool("started"),
            delivered: bound.take_bool("delivered"),
            credential_returned: bound.take_bool("credential_returned"),
            output: bound.take_string("output"),
            retirement: bound.take_string("retirement"),
            reason: bound.take_string("reason"),
        };
        if receipt.run.id != run || receipt.run.project != project {
            return Err(FactoryError::msg("run receipt identity mismatch"));
        }
        // A stop tombstone for a never-admitted run carries no validated
        // run record; its only job is to refuse future work under that
        // identity.
        if receipt.phase == FACTORY_STOPPED && receipt.run.validate().is_err() {
            if receipt.lease.is_some()
                || receipt.binding.is_some()
                || receipt.started
                || receipt.delivered
            {
                return Err(FactoryError::msg("invalid run tombstone"));
            }
            return Ok((receipt, true));
        }
        receipt.validate()?;
        Ok((receipt, true))
    }

    /// `storeReceipt`: validate, then atomically persist via a 0600
    /// temporary file and rename.
    fn store_receipt(&self, receipt: &FactoryReceipt) -> Result<(), FactoryError> {
        receipt.validate()?;
        self.write_receipt_file(receipt)
    }

    /// `storeTombstone`: persist a stop marker for a never-admitted run
    /// without run validation.
    fn store_tombstone(&self, receipt: &FactoryReceipt) -> Result<(), FactoryError> {
        self.write_receipt_file(receipt)
    }

    fn write_receipt_file(&self, receipt: &FactoryReceipt) -> Result<(), FactoryError> {
        use std::io::Write;
        let data = receipt.encode();
        let path = self.receipt_path(&receipt.run.project, &receipt.run.id);
        // Unique temporary sibling, like `os.CreateTemp(stateDir, ".receipt-")`.
        let mut counter = 0u32;
        let tmp_path = loop {
            counter += 1;
            let candidate = format!(
                "{}/.receipt-{}-{}-{counter}",
                self.state_dir,
                std::process::id(),
                SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .map(|d| d.as_nanos())
                    .unwrap_or(0)
            );
            match std::fs::OpenOptions::new()
                .create_new(true)
                .write(true)
                .mode(0o600)
                .open(&candidate)
            {
                Ok(_) => break candidate,
                Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists && counter < 100 => {
                    continue
                }
                Err(e) => return Err(FactoryError::msg(e.to_string())),
            }
        };
        let write_result = (|| -> std::io::Result<()> {
            let mut tmp = std::fs::OpenOptions::new().write(true).open(&tmp_path)?;
            tmp.write_all(data.as_bytes())?;
            tmp.set_permissions(std::fs::Permissions::from_mode(0o600))?;
            drop(tmp);
            std::fs::rename(&tmp_path, &path)?;
            Ok(())
        })();
        if let Err(e) = write_result {
            let _ = std::fs::remove_file(&tmp_path);
            return Err(FactoryError::msg(e.to_string()));
        }
        Ok(())
    }

    /// `Launch`: admit one supervised run and drive it to completion. An
    /// existing receipt is authoritative: duplicates return its state.
    /// Only unconfirmed operations return an error.
    pub fn launch(
        &self,
        req: &FactoryLaunch,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        if req.harness_sha256.is_empty() || req.harness_sha256 != self.terminal.harness_sha256() {
            return Err(FactoryError::msg(
                "factory harness is unavailable until its own proof passes",
            ));
        }
        // Validated above: parseable and nonzero.
        let deadline_nanos = parse_deadline(&req.run.deadline).unwrap_or(0);
        let now = system_nanos_now();
        if !(deadline_nanos > now && deadline_nanos <= now + FACTORY_DEADLINE_BOUND_NANOS) {
            return Err(FactoryError::msg(
                "run deadline is outside the supervised bound",
            ));
        }
        let lock = self.lock_run(&req.run.project, &req.run.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.run.project, &req.run.id)?;
        if !exists {
            let receipt = FactoryReceipt {
                run: req.run.clone(),
                phase: FACTORY_APPROVED.to_string(),
                ..FactoryReceipt::default()
            };
            let stored = self.store_receipt(&receipt);
            drop(lock);
            stored?;
            return self.drive(req, &receipt, deadline_nanos, deadline);
        }
        drop(lock);
        Ok(receipt_state(&receipt, false))
    }

    /// `drive`: broker acquisition, native reservation, attested delivery,
    /// single-use start, bounded wait and credential return.
    fn drive(
        &self,
        req: &FactoryLaunch,
        receipt: &FactoryReceipt,
        deadline_nanos: i128,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let run_deadline = drive_deadline(deadline, deadline_nanos);
        let lease = match self.broker.acquire(
            &AcquireRequest {
                provider_id: IDENTITY_CODEX.to_string(),
                execution_id: req.run.id.clone(),
                actor_id: req.run.actor,
                connection_id: req.run.connection.clone(),
                project_id: req.run.project.clone(),
                kind: IDENTITY_FACTORY.to_string(),
                deadline: req.run.deadline.clone(),
                role: req.run.role.clone(),
            },
            run_deadline,
        ) {
            Ok(lease) => lease,
            Err(cause) => return self.fail_run(receipt, cause, deadline),
        };
        let mut receipt = receipt.clone();
        receipt.generation = lease.generation;
        receipt.lease = Some(lease.clone());
        if let Some(stopped) = self.refresh_stopped(&mut receipt, deadline)? {
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &req.run.id, deadline);
            return Ok(stopped);
        }
        let binding = match self.terminal.reserve(
            &req.run,
            &lease,
            &req.harness_sha256,
            seconds_until(deadline_nanos),
            run_deadline,
        ) {
            Ok(binding) => binding,
            Err(_) => return self.abandon_run(&receipt, "reserve-refused", deadline),
        };
        receipt.binding = Some(binding.clone());
        if let Some(stopped) = self.refresh_stopped(&mut receipt, deadline)? {
            let _ = self
                .terminal
                .stop(&lease.with_binding(&binding), run_deadline);
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &req.run.id, deadline);
            return Ok(stopped);
        }
        let credential = Secret(
            match self.broker.register(&lease.id, &binding, run_deadline) {
                Ok(credential) => credential,
                Err(_) => {
                    let _ = self
                        .terminal
                        .stop(&lease.with_binding(&binding), run_deadline);
                    let _ = self
                        .broker
                        .close_execution(IDENTITY_FACTORY, &req.run.id, deadline);
                    return self.abandon_run(&receipt, "register-refused", deadline);
                }
            },
        );
        if let Some(stopped) = self.consume_start(&mut receipt, deadline)? {
            let _ = self
                .terminal
                .stop(&lease.with_binding(&binding), run_deadline);
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &req.run.id, deadline);
            return Ok(stopped);
        }
        if self
            .terminal
            .start(
                &lease.with_binding(&binding),
                &credential.0,
                &req.prompt,
                run_deadline,
            )
            .is_err()
        {
            return self.stop_failed_start(&receipt, &lease, &binding, deadline);
        }
        receipt.delivered = true;
        self.update_receipt(&mut receipt, deadline)?;
        match self
            .terminal
            .wait(&lease.with_binding(&binding), run_deadline)
        {
            Ok((exit, output)) => {
                self.finish_run(&receipt, &lease, &binding, exit, output, deadline)
            }
            Err(_) => self.stop_timed_out(&receipt, &lease, &binding, deadline),
        }
    }

    /// `failRun`: record a refusal that never acquired native or broker
    /// resources. A concurrent stop still wins.
    fn fail_run(
        &self,
        receipt: &FactoryReceipt,
        cause: FactoryError,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, cleanup)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(FactoryState::default());
        }
        if current.phase == FACTORY_STOPPED {
            return Ok(receipt_state(&current, false));
        }
        if cause == FactoryError::Busy {
            current.reason = "broker-busy".to_string();
            self.store_receipt(&current)?;
            return Ok(receipt_state(&current, false));
        }
        if matches!(
            cause,
            FactoryError::Denied | FactoryError::Uncertain | FactoryError::DeadlineExceeded
        ) {
            current.phase = FACTORY_FAILED.to_string();
            current.reason = run_reason(&cause).to_string();
            self.store_receipt(&current)?;
            return Ok(receipt_state(&current, false));
        }
        Err(cause)
    }

    /// `abandonRun`: close the broker execution and record failure after a
    /// refused local step. Transport failures stay errors.
    fn abandon_run(
        &self,
        receipt: &FactoryReceipt,
        reason: &str,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let _ = self
            .broker
            .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, cleanup)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(FactoryState::default());
        }
        if current.phase == FACTORY_STOPPED {
            return Ok(receipt_state(&current, false));
        }
        current.phase = FACTORY_FAILED.to_string();
        current.reason = reason.to_string();
        self.store_receipt(&current)?;
        Ok(receipt_state(&current, false))
    }

    /// `refreshStopped`: re-read the receipt and store progress. A stop
    /// tombstone aborts the launch.
    fn refresh_stopped(
        &self,
        receipt: &mut FactoryReceipt,
        deadline: Instant,
    ) -> Result<Option<FactoryState>, FactoryError> {
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(None);
        }
        if current.phase == FACTORY_STOPPED {
            return Ok(Some(receipt_state(&current, false)));
        }
        current.lease = receipt.lease.clone();
        current.binding = receipt.binding.clone();
        current.generation = receipt.generation;
        self.store_receipt(&current)?;
        *receipt = current;
        Ok(None)
    }

    /// `consumeStart`: consume the single-use start marker. A second start,
    /// or a start after the stop tombstone, never reaches the native
    /// boundary.
    fn consume_start(
        &self,
        receipt: &mut FactoryReceipt,
        deadline: Instant,
    ) -> Result<Option<FactoryState>, FactoryError> {
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(None);
        }
        if current.phase == FACTORY_STOPPED || current.started {
            return Ok(Some(receipt_state(&current, false)));
        }
        current.started = true;
        current.phase = FACTORY_RUNNING.to_string();
        self.store_receipt(&current)?;
        *receipt = current;
        Ok(None)
    }

    fn update_receipt(
        &self,
        receipt: &mut FactoryReceipt,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(());
        }
        current.delivered = receipt.delivered;
        current.output = receipt.output.clone();
        current.exit_code = receipt.exit_code;
        current.credential_returned = receipt.credential_returned;
        current.retirement = receipt.retirement.clone();
        current.reason = receipt.reason.clone();
        if !receipt.phase.is_empty() {
            current.phase = receipt.phase.clone();
        }
        self.store_receipt(&current)?;
        *receipt = current;
        Ok(())
    }

    /// `launchYielded`: whether a concurrent stop tombstoned the run.
    fn launch_yielded(&self, receipt: &FactoryReceipt, deadline: Instant) -> bool {
        let Ok(_lock) = self.lock_run(&receipt.run.project, &receipt.run.id, deadline) else {
            return false;
        };
        match self.load_receipt(&receipt.run.project, &receipt.run.id) {
            Ok((current, true)) => current.phase == FACTORY_STOPPED,
            _ => false,
        }
    }

    /// `stopFailedStart`: retire a run whose start never confirmed: native
    /// stop, credential capture attempt, broker return or reconcile, then
    /// close.
    fn stop_failed_start(
        &self,
        receipt: &FactoryReceipt,
        lease: &Lease,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let bound = lease.with_binding(binding);
        let _ = self.terminal.stop(&bound, cleanup);
        let mut receipt = receipt.clone();
        if self.launch_yielded(&receipt, cleanup) {
            return self.record_stop_outcome(&receipt, false, "stopped", false, cleanup);
        }
        let mut uncertain = false;
        match self.terminal.capture(&bound, cleanup) {
            Ok(auth) => {
                let auth = Secret(auth);
                if self
                    .broker
                    .return_lease(&lease.id, binding, &auth.0, cleanup)
                    .is_ok()
                {
                    receipt.credential_returned = true;
                } else {
                    uncertain = true;
                }
            }
            Err(_) => {
                self.broker.reconcile_lease(&lease.id, cleanup);
                uncertain = true;
            }
        }
        if self
            .broker
            .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup)
            .is_err()
        {
            uncertain = true;
        }
        self.record_stop_outcome(&receipt, uncertain, "start-unconfirmed", false, cleanup)
    }

    fn stop_timed_out(
        &self,
        receipt: &FactoryReceipt,
        lease: &Lease,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let bound = lease.with_binding(binding);
        let mut uncertain = false;
        if self.terminal.stop(&bound, cleanup).is_err() {
            uncertain = true;
        }
        let mut receipt = receipt.clone();
        if self.launch_yielded(&receipt, cleanup) {
            return self.record_stop_outcome(&receipt, uncertain, "stopped", false, cleanup);
        }
        match self.terminal.capture(&bound, cleanup) {
            Ok(auth) => {
                let auth = Secret(auth);
                if self
                    .broker
                    .return_lease(&lease.id, binding, &auth.0, cleanup)
                    .is_ok()
                {
                    receipt.credential_returned = true;
                } else {
                    uncertain = true;
                }
            }
            Err(_) => {
                self.broker.reconcile_lease(&lease.id, cleanup);
                uncertain = true;
            }
        }
        if self
            .broker
            .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup)
            .is_err()
        {
            uncertain = true;
        }
        self.record_stop_outcome(&receipt, uncertain, "deadline-exceeded", false, cleanup)
    }

    fn finish_run(
        &self,
        receipt: &FactoryReceipt,
        lease: &Lease,
        binding: &Binding,
        exit: i64,
        output: String,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _ = deadline;
        let cleanup = cleanup_deadline();
        let bound = lease.with_binding(binding);
        let mut receipt = receipt.clone();
        if exit >= 0 {
            receipt.exit_code = Some(exit);
        }
        receipt.output = output;
        // The CLI exited, but descendants may linger: retire the boundary
        // before capturing, so the captured bytes are final.
        if self.terminal.stop(&bound, cleanup).is_err() {
            self.broker.reconcile_lease(&lease.id, cleanup);
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
            return self.record_stop_outcome(
                &receipt,
                true,
                "retirement-unconfirmed",
                false,
                cleanup,
            );
        }
        if self.launch_yielded(&receipt, cleanup) {
            return self.record_stop_outcome(&receipt, false, "stopped", false, cleanup);
        }
        let auth = Secret(match self.terminal.capture(&bound, cleanup) {
            Ok(auth) => auth,
            Err(_) => {
                self.broker.reconcile_lease(&lease.id, cleanup);
                let _ = self
                    .broker
                    .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
                return self.record_stop_outcome(
                    &receipt,
                    true,
                    "credential-capture-unconfirmed",
                    false,
                    cleanup,
                );
            }
        });
        if self
            .broker
            .return_lease(&lease.id, binding, &auth.0, cleanup)
            .is_err()
        {
            let _ = self
                .broker
                .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
            return self.record_stop_outcome(
                &receipt,
                true,
                "credential-return-unconfirmed",
                false,
                cleanup,
            );
        }
        receipt.credential_returned = true;
        let _ = self
            .broker
            .close_execution(IDENTITY_FACTORY, &receipt.run.id, cleanup);
        receipt.phase = FACTORY_COMPLETED.to_string();
        receipt.retirement = "confirmed".to_string();
        receipt.reason.clear();
        if exit != 0 {
            receipt.phase = FACTORY_FAILED.to_string();
            receipt.reason = "execution-failed".to_string();
        }
        self.update_receipt(&mut receipt, cleanup)?;
        Ok(receipt_state(&receipt, false))
    }

    /// `Stop`: persist the stop tombstone for one run identity and retire
    /// its native boundary and broker execution. It works before the run
    /// is ever observed and refuses all subsequent work under that ID.
    pub fn stop(&self, req: &FactoryStop, deadline: Instant) -> Result<FactoryState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (mut receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        if !exists {
            receipt = FactoryReceipt {
                run: FactoryRun {
                    id: req.id.clone(),
                    project: req.project.clone(),
                    // Go zero-time encoding, byte-exact in the tombstone.
                    deadline: "0001-01-01T00:00:00Z".to_string(),
                    ..FactoryRun::default()
                },
                phase: FACTORY_STOPPED.to_string(),
                retirement: "confirmed".to_string(),
                reason: "stop-before-start".to_string(),
                ..FactoryReceipt::default()
            };
            // A tombstone for an unknown run carries no validated run
            // record; store it directly so later launches refuse without
            // a full identity.
            drop(lock);
            self.store_tombstone(&receipt)?;
            if self
                .broker
                .close_execution(IDENTITY_FACTORY, &req.id, deadline)
                .is_err()
            {
                receipt.reason = "broker-close-uncertain".to_string();
                let _ = self.store_tombstone(&receipt);
                return Ok(receipt_state(&receipt, false));
            }
            return Ok(receipt_state(&receipt, false));
        }
        if receipt_terminal(&receipt.phase) {
            // A tombstone whose broker fence never confirmed retries it;
            // every other terminal receipt is final.
            let retry_close = receipt.phase == FACTORY_STOPPED
                && receipt.reason == "broker-close-uncertain"
                && receipt.run.validate().is_err();
            drop(lock);
            if retry_close
                && self
                    .broker
                    .close_execution(IDENTITY_FACTORY, &req.id, deadline)
                    .is_ok()
            {
                receipt.reason = "stop-before-start".to_string();
                let _ = self.store_tombstone(&receipt);
            }
            return Ok(receipt_state(&receipt, false));
        }
        receipt.phase = FACTORY_STOPPED.to_string();
        self.store_receipt(&receipt)?;
        drop(lock);
        let mut uncertain = false;
        match (&receipt.binding, &receipt.lease) {
            (Some(binding), Some(lease)) => {
                if self
                    .terminal
                    .stop(&lease.with_binding(binding), deadline)
                    .is_err()
                {
                    uncertain = true;
                }
            }
            _ => {
                if receipt.run.validate().is_ok()
                    && self.terminal.stop_unbound(&receipt.run, deadline).is_err()
                {
                    uncertain = true;
                }
            }
        }
        if !self.reconcile_run_credential(&mut receipt, deadline) {
            uncertain = true;
        }
        if self
            .broker
            .close_execution(IDENTITY_FACTORY, &req.id, deadline)
            .is_err()
        {
            uncertain = true;
        }
        let reason = if uncertain {
            "stop-uncertain"
        } else {
            "stopped"
        };
        self.record_stop_outcome(&receipt, uncertain, reason, true, deadline)
    }

    /// `reconcileRunCredential`: return or reconcile the run's broker lease
    /// after native retirement. Reports whether custody is settled.
    fn reconcile_run_credential(&self, receipt: &mut FactoryReceipt, deadline: Instant) -> bool {
        let (lease, binding) = match (&receipt.lease, &receipt.binding) {
            (Some(lease), binding) if !receipt.credential_returned => {
                (lease.clone(), binding.clone())
            }
            _ => return true,
        };
        if receipt.delivered {
            if let Some(binding) = binding {
                let auth = match self
                    .terminal
                    .capture(&lease.with_binding(&binding), deadline)
                {
                    Ok(auth) => Secret(auth),
                    Err(_) => {
                        self.broker.reconcile_lease(&lease.id, deadline);
                        return false;
                    }
                };
                if self
                    .broker
                    .return_lease(&lease.id, &binding, &auth.0, deadline)
                    .is_err()
                {
                    // The racing launch may have returned first: a terminal
                    // execution means custody settled without this stop.
                    // CredentialReturned stays false: this stop did not return.
                    if let Ok(true) = self.broker.execution_is_terminal(
                        IDENTITY_FACTORY,
                        &receipt.run.id,
                        deadline,
                    ) {
                        return true;
                    }
                    return false;
                }
                receipt.credential_returned = true;
                return true;
            }
        }
        self.broker.reconcile_lease(&lease.id, deadline);
        true
    }

    /// `Inspect`: report the authoritative recorded state for one run
    /// identity and whether its unit is currently live.
    pub fn inspect(
        &self,
        req: &FactoryInspect,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        drop(lock);
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let mut live = false;
        if let Some(binding) = &receipt.binding {
            if receipt.phase == FACTORY_APPROVED || receipt.phase == FACTORY_RUNNING {
                live = self.terminal.live(binding, live_deadline(deadline));
            }
        }
        Ok(receipt_state(&receipt, live))
    }

    /// `Takeover`: copy one retired run's retained work into the admitted
    /// member's own derived checkout destination. Holds the lock across
    /// the copy to serialize duplicate takeovers.
    pub fn takeover(
        &self,
        req: &FactoryTakeover,
        deadline: Instant,
    ) -> Result<TakeoverResult, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let _lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = match &receipt.binding {
            Some(binding) if receipt_terminal(&receipt.phase) && receipt.run.validate().is_ok() => {
                binding.clone()
            }
            _ => return Err(FactoryError::msg("factory run is not retired for takeover")),
        };
        if !takeover_source(
            &format!(
                "/home/{}/checkouts/{}",
                receipt.run.role, receipt.run.preparation
            ),
            &receipt.run.role,
            &receipt.run.preparation,
        ) {
            return Err(FactoryError::msg("invalid takeover source"));
        }
        let (dest, reused) = self.terminal.takeover_copy(
            &req.project,
            &binding.project,
            &receipt.run.role,
            &receipt.run.preparation,
            &req.member,
            &req.id,
            deadline,
        )?;
        let result = TakeoverResult {
            id: req.id.clone(),
            project: req.project.clone(),
            member: req.member.clone(),
            destination: dest,
            reused,
        };
        result.validate().map_err(FactoryError::msg)?;
        Ok(result)
    }

    fn record_stop_outcome(
        &self,
        receipt: &FactoryReceipt,
        uncertain: bool,
        reason: &str,
        overwrite: bool,
        deadline: Instant,
    ) -> Result<FactoryState, FactoryError> {
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        let (mut current, exists) = self.load_receipt(&receipt.run.project, &receipt.run.id)?;
        if !exists {
            return Ok(FactoryState::default());
        }
        // A concurrent operator stop owns the outcome of a launch; only
        // the stop path itself overwrites the tombstone it just persisted.
        if current.phase == FACTORY_STOPPED && !overwrite {
            return Ok(receipt_state(&current, false));
        }
        current.credential_returned = receipt.credential_returned;
        current.output = receipt.output.clone();
        current.exit_code = receipt.exit_code;
        if uncertain {
            current.phase = FACTORY_UNCERTAIN.to_string();
            current.retirement = "uncertain".to_string();
            current.reason = reason.to_string();
        } else if overwrite {
            current.phase = FACTORY_STOPPED.to_string();
            current.retirement = "confirmed".to_string();
            current.reason = reason.to_string();
        } else {
            current.phase = FACTORY_FAILED.to_string();
            current.retirement = "confirmed".to_string();
            current.reason = reason.to_string();
        }
        self.store_receipt(&current)?;
        Ok(receipt_state(&current, false))
    }

    /// `Output`: report one bounded slice of a run's recorded CLI output
    /// with its run/process binding. A run whose recorded container
    /// incarnation no longer resolves refuses stale.
    pub fn output(
        &self,
        req: &FactoryOutput,
        deadline: Instant,
    ) -> Result<FactoryOutputState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        drop(lock);
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let mut state = FactoryOutputState {
            id: req.id.clone(),
            project: req.project.clone(),
            phase: receipt.phase.clone(),
            terminal: receipt_terminal(&receipt.phase),
            reason: receipt.reason.clone(),
            exit_code: receipt.exit_code,
            ..FactoryOutputState::default()
        };
        let binding = match &receipt.binding {
            Some(binding) => binding.clone(),
            None => return Ok(state),
        };
        // Container, unit and invocation describe the recorded process
        // binding together; before a binding exists all three stay empty.
        state.container = binding.project.clone();
        state.unit = factory_unit_name(&req.id);
        state.invocation = binding.invocation_id.clone();
        if receipt.phase == FACTORY_APPROVED || receipt.phase == FACTORY_RUNNING {
            state.live = self.terminal.live(&binding, live_deadline(deadline));
        }
        let slice = self.terminal.output(
            &receipt.run.project,
            &binding,
            req.offset,
            req.limit,
            deadline,
        )?;
        state.total = slice.total;
        state.offset = slice.offset;
        state.truncated = slice.truncated;
        state.gap = slice.gap;
        state.next = slice.offset + slice.data.len() as i64;
        if !slice.data.is_empty() {
            state.data = crate::ssh::b64_encode(&slice.data);
        }
        Ok(state)
    }

    /// `Export`: read one settled run's exact candidate as a bounded Git
    /// bundle from its recorded role checkout.
    pub fn export(
        &self,
        req: &FactoryExport,
        deadline: Instant,
    ) -> Result<FactoryExportState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        drop(lock);
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = match &receipt.binding {
            Some(binding) if receipt_terminal(&receipt.phase) && receipt.run.validate().is_ok() => {
                binding.clone()
            }
            _ => return Err(FactoryError::msg("factory run is not settled for export")),
        };
        if receipt.run.role != req.role || receipt.run.preparation != req.preparation {
            return Err(FactoryError::Denied);
        }
        let bundle = self.terminal.export_bundle(
            &req.project,
            &binding.project,
            &req.role,
            &req.preparation,
            &req.candidate,
            deadline,
        )?;
        if bundle.is_empty() || bundle.len() > MAX_FACTORY_EXPORT_BUNDLE {
            return Err(FactoryError::ExportBounds);
        }
        Ok(FactoryExportState {
            id: req.id.clone(),
            project: req.project.clone(),
            phase: receipt.phase.clone(),
            container: binding.project.clone(),
            candidate: req.candidate.clone(),
            bundle: crate::ssh::b64_encode(&bundle),
        })
    }

    /// `InspectCandidate`: resolve role and checkout from the settled host
    /// receipt. An old run never reads a replacement Project container: a
    /// container mismatch, like any observation failure, refuses stale.
    pub fn inspect_candidate(
        &self,
        req: &FactoryCandidateInspect,
        deadline: Instant,
    ) -> Result<FactoryCandidateState, FactoryError> {
        req.validate().map_err(FactoryError::msg)?;
        let lock = self.lock_run(&req.project, &req.id, deadline)?;
        let (receipt, exists) = self.load_receipt(&req.project, &req.id)?;
        drop(lock);
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = match &receipt.binding {
            Some(binding) if receipt_terminal(&receipt.phase) && receipt.run.validate().is_ok() => {
                binding.clone()
            }
            _ => {
                return Err(FactoryError::msg(
                    "candidate inspection requires a settled run",
                ))
            }
        };
        // Go builds a zero-config Runtime here; container binding needs no
        // configuration.
        let runtime = Runtime {
            exec: &self.exec,
            config: Config::default(),
        };
        let container = runtime.prepare_container(&req.project, true, deadline);
        let container = match container {
            Ok(container) if container == binding.project => container,
            _ => return Err(FactoryError::Stale),
        };
        let (checkout, _, _, _) =
            factory_run_paths(&receipt.run.role, &receipt.run.preparation, &receipt.run.id);
        let home = format!("/home/{}", receipt.run.role);
        let tmpdir = format!("/home/{}/checkouts", receipt.run.role);
        let path = "PATH=/usr/bin:/bin";
        let home_env = format!("HOME={home}");
        let tmpdir_env = format!("TMPDIR={tmpdir}");
        let args = [
            "exec",
            "--user",
            receipt.run.role.as_str(),
            container.as_str(),
            "/usr/bin/env",
            "-i",
            path,
            home_env.as_str(),
            "LC_ALL=C",
            tmpdir_env.as_str(),
            "GIT_CONFIG_NOSYSTEM=1",
            "GIT_CONFIG_GLOBAL=/dev/null",
            "GIT_NO_REPLACE_OBJECTS=1",
            "GIT_TERMINAL_PROMPT=0",
            "GIT_OPTIONAL_LOCKS=0",
            "/usr/bin/sh",
            "-c",
            CANDIDATE_INSPECT_SCRIPT,
            "soda-candidate",
            checkout.as_str(),
        ];
        let out = runtime
            .exec
            .run(&[], "/usr/bin/podman", &args, deadline)
            .map_err(|_| FactoryError::msg("candidate checkout inspection unconfirmed"))?;
        if out.len() > 64 {
            return Err(FactoryError::msg(
                "candidate checkout inspection unconfirmed",
            ));
        }
        let text = String::from_utf8_lossy(&out);
        let fields: Vec<&str> = text.split_whitespace().collect();
        if fields.len() != 2
            || !preparation::valid_commit(fields[0])
            || (fields[1] != "clean" && fields[1] != "dirty")
        {
            return Err(FactoryError::msg(
                "candidate checkout observation is invalid",
            ));
        }
        Ok(FactoryCandidateState {
            id: req.id.clone(),
            project: req.project.clone(),
            container,
            candidate: fields[0].to_string(),
            dirty: fields[1] == "dirty",
        })
    }
}

fn run_reason(cause: &FactoryError) -> &'static str {
    match cause {
        FactoryError::Denied => "broker-denied",
        FactoryError::Uncertain => "broker-unavailable",
        FactoryError::DeadlineExceeded => "deadline-exceeded",
        _ => "launch-refused",
    }
}

// Only rev-parse reads source metadata; it executes no filters or hooks.
// Every worktree/index comparison uses a clean repository, source objects
// and a disposable index. No agent configuration or optional external diff
// runs. Byte-exact with Go's `candidateInspectScript`, trailing newline
// included.
pub const CANDIDATE_INSPECT_SCRIPT: &str = r#"set -eu
umask 077
src=$1
head=$(/usr/bin/git --git-dir="$src/.git" rev-parse --verify 'HEAD^{commit}')
dir=$(/usr/bin/mktemp -d "$TMPDIR/.soda-candidate-XXXXXX")
trap '/usr/bin/rm -rf "$dir"' EXIT HUP INT TERM
/usr/bin/git -c core.hooksPath=/dev/null init --bare --template= "$dir/repo.git" >/dev/null 2>/dev/null
export GIT_OBJECT_DIRECTORY="$src/.git/objects"
git_candidate() {
  /usr/bin/git -c core.hooksPath=/dev/null -c core.fsmonitor=false --git-dir="$dir/repo.git" --work-tree="$src" "$@"
}
dirty=clean
export GIT_INDEX_FILE="$src/.git/index"
if git_candidate diff-index --cached --quiet --no-ext-diff --no-textconv "$head" --; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"; dirty=dirty
fi
export GIT_INDEX_FILE="$dir/index"
git_candidate read-tree "$head"
if git_candidate update-index --refresh >/dev/null; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"
fi
if git_candidate diff-files --quiet --no-ext-diff --no-textconv --; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"; dirty=dirty
fi
# Standard exclusions read worktree .gitignore with this clean repository's
# empty info/exclude and no global configuration. A new visible .gitignore must
# itself remain visible, even if its rules try to ignore that file. Already
# ignored build directories stay excluded without traversing their contents.
git_candidate ls-files --others --exclude-standard --exclude='!**/.gitignore' --exclude=.git/ --exclude=.soda-home/ >"$dir/untracked"
[ ! -s "$dir/untracked" ] || dirty=dirty
printf '%s %s\n' "$head" "$dirty"
"#;

// ---------- facade confirmations ----------
//
// Pure client-side logic from `internal/host/{factory_client,
// factory_candidate,prepare}.go`: transport (`c.call`) arrives with the
// route layer, but identity confirmations and status mappings are exact
// here so both sides pin the same wire contract.

/// `factoryStateConfirmed`: the launch returned the admitted run.
pub fn confirm_factory_launch(run: &FactoryRun, state: &FactoryState) -> Result<(), String> {
    if state.id == run.id
        && state.project == run.project
        && state.role == run.role
        && valid_factory_phase(&state.phase)
    {
        Ok(())
    } else {
        Err("native factory launch did not return the admitted run".to_string())
    }
}

/// `FactoryInspect` client confirmation.
pub fn confirm_factory_inspect(req: &FactoryInspect, state: &FactoryState) -> Result<(), String> {
    if state.id == req.id && state.project == req.project && valid_factory_phase(&state.phase) {
        Ok(())
    } else {
        Err("native factory observation does not match its identity".to_string())
    }
}

/// `FactoryHarness` client confirmation.
pub fn confirm_factory_harness(pin: &FactoryHarnessPin) -> Result<(), String> {
    if pin.validate().is_ok() {
        Ok(())
    } else {
        Err("native factory harness pin is not usable".to_string())
    }
}

/// `FactoryStop` client confirmation.
pub fn confirm_factory_stop(req: &FactoryStop, state: &FactoryState) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && valid_factory_phase(&state.phase)
        && (state.retirement.is_empty()
            || state.retirement == "confirmed"
            || state.retirement == "uncertain")
    {
        Ok(())
    } else {
        Err("native factory stop was not confirmed".to_string())
    }
}

/// `FactoryOutput` client identity confirmation (status mapping is
/// [`factory_output_status_error`]).
pub fn confirm_factory_output(
    req: &FactoryOutput,
    state: &FactoryOutputState,
) -> Result<(), String> {
    if state.id == req.id && state.project == req.project && valid_factory_phase(&state.phase) {
        Ok(())
    } else {
        Err("native factory output does not match its identity".to_string())
    }
}

/// `FactoryExport` client identity confirmation.
pub fn confirm_factory_export(
    req: &FactoryExport,
    state: &FactoryExportState,
) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && state.candidate == req.candidate
        && valid_factory_phase(&state.phase)
        && !state.bundle.is_empty()
        && state.bundle.len() <= 8 << 20
    {
        Ok(())
    } else {
        Err("native factory export does not match its identity".to_string())
    }
}

/// `FactoryTakeover` client confirmation.
pub fn confirm_factory_takeover(
    req: &FactoryTakeover,
    result: &TakeoverResult,
) -> Result<(), String> {
    if result.validate().is_ok()
        && result.id == req.id
        && result.project == req.project
        && result.member == req.member
    {
        Ok(())
    } else {
        Err("native factory takeover did not return the admitted destination".to_string())
    }
}

/// `FactoryInspectCandidate` client identity confirmation.
pub fn confirm_factory_candidate_inspect(
    req: &FactoryCandidateInspect,
    state: &FactoryCandidateState,
) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && domain::valid_container_id(&state.container)
        && preparation::valid_commit(&state.candidate)
    {
        Ok(())
    } else {
        Err("native candidate observation does not match its identity".to_string())
    }
}

/// `preparationIdentityConfirmed`, shared by the prepare and
/// prepare-candidate clients.
pub fn confirm_prepare(prep: &Preparation, state: &PrepareState) -> Result<(), String> {
    if state.id == prep.id
        && state.project == prep.project
        && state.role == prep.role
        && preparation::valid_prepare_phase(&state.phase)
        && domain::valid_container_id(&state.container)
        && state.source_commit == prep.source_commit
        && state.setup_digest == prep.setup_digest
    {
        Ok(())
    } else {
        Err("native preparation result does not match its identity".to_string())
    }
}

/// `PrepareCandidate` client confirmation.
pub fn confirm_prepare_candidate(prep: &Preparation, state: &PrepareState) -> Result<(), String> {
    if state.id == prep.id
        && state.project == prep.project
        && state.role == prep.role
        && preparation::valid_prepare_phase(&state.phase)
        && domain::valid_container_id(&state.container)
        && state.source_commit == prep.source_commit
        && state.setup_digest == prep.setup_digest
    {
        Ok(())
    } else {
        Err("native candidate preparation does not match its identity".to_string())
    }
}

/// `InspectPreparation` client confirmation.
pub fn confirm_inspect_preparation(
    req: &PrepareInspect,
    state: &PrepareState,
) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && preparation::valid_prepare_phase(&state.phase)
        && domain::valid_container_id(&state.container)
    {
        Ok(())
    } else {
        Err("native preparation observation does not match its identity".to_string())
    }
}

/// `StopPreparation` client confirmation.
pub fn confirm_stop_preparation(req: &PrepareStop, state: &PrepareState) -> Result<(), String> {
    if state.id == req.id
        && state.project == req.project
        && state.stopped
        && (state.retirement == "confirmed" || state.retirement == "uncertain")
    {
        Ok(())
    } else {
        Err("native preparation stop was not confirmed".to_string())
    }
}

/// `HoldPreparation` client confirmation.
pub fn confirm_hold_preparation(req: &PrepareHold, state: &HoldState) -> Result<(), String> {
    if state.active == req.hold {
        Ok(())
    } else {
        Err("native maintenance hold outcome not confirmed".to_string())
    }
}

/// `factoryNotFound`: inspect/takeover 404 mapping.
pub fn factory_not_found_status(status: u16) -> Option<String> {
    if status == 404 {
        Some(ERR_RUN_NOT_FOUND.to_string())
    } else {
        None
    }
}

/// `factoryOutputError` status mapping: 404 and 409.
pub fn factory_output_status_error(status: u16) -> Option<String> {
    match status {
        404 => Some(ERR_RUN_NOT_FOUND.to_string()),
        409 => Some(ERR_RUN_STALE.to_string()),
        _ => None,
    }
}

/// `factoryExportError` status mapping: 404, 409, 422 and 413.
pub fn factory_export_status_error(status: u16) -> Option<String> {
    match status {
        404 => Some(ERR_RUN_NOT_FOUND.to_string()),
        409 => Some(ERR_RUN_STALE.to_string()),
        422 => Some("export candidate is not recorded".to_string()),
        413 => Some("candidate export exceeds bounds".to_string()),
        _ => None,
    }
}

/// `FactoryInspectCandidate` status mapping: 409, then 404.
pub fn factory_candidate_status_error(status: u16) -> Option<String> {
    match status {
        409 => Some(ERR_RUN_STALE.to_string()),
        404 => Some(ERR_RUN_NOT_FOUND.to_string()),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::RefCell;
    use std::collections::VecDeque;
    use std::sync::atomic::{AtomicU64, Ordering};

    static TAG_COUNTER: AtomicU64 = AtomicU64::new(0);

    fn test_state_dir(tag: &str) -> std::path::PathBuf {
        let dir = std::env::temp_dir().join(format!(
            "soda-pf26-{}-{}-{}",
            tag,
            std::process::id(),
            TAG_COUNTER.fetch_add(1, Ordering::SeqCst)
        ));
        std::fs::create_dir_all(&dir).unwrap();
        std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700)).unwrap();
        dir
    }

    fn deadline() -> Instant {
        Instant::now() + Duration::from_secs(30)
    }

    /// Inverse of `days_from_civil` for test deadlines.
    fn civil_from_days(z: i64) -> (i64, u32, u32) {
        let z = z + 719468;
        let era = if z >= 0 { z } else { z - 146096 } / 146097;
        let doe = z - era * 146097;
        let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
        let y = yoe + era * 400;
        let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
        let mp = (5 * doy + 2) / 153;
        let d = (doy - (153 * mp + 2) / 5 + 1) as u32;
        let m = if mp < 10 { mp + 3 } else { mp - 9 } as u32;
        (if m <= 2 { y + 1 } else { y }, m, d)
    }

    /// RFC3339 `Z` deadline `offset_secs` in the future.
    fn deadline_text(offset_secs: i64) -> String {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64
            + offset_secs;
        let (y, m, d) = civil_from_days(now.div_euclid(86400));
        let secs = now.rem_euclid(86400);
        format!(
            "{y:04}-{m:02}-{d:02}T{:02}:{:02}:{:02}Z",
            secs / 3600,
            secs % 3600 / 60,
            secs % 60
        )
    }

    fn run_id() -> String {
        "a".repeat(32)
    }

    fn project_id() -> String {
        format!("p{}", "b".repeat(24))
    }

    fn prep_id() -> String {
        format!("f{}", "c".repeat(24))
    }

    fn container_id() -> String {
        "d".repeat(64)
    }

    fn sample_run() -> FactoryRun {
        let prompt = b"do the thing";
        FactoryRun {
            deadline: deadline_text(3600),
            actor: 7,
            id: run_id(),
            project: project_id(),
            role: "soda-coder".to_string(),
            preparation: prep_id(),
            harness: "codex".to_string(),
            harness_vers: "v1".to_string(),
            model: String::new(),
            assignment: crate::sha256::hex_lower(&crate::sha256::digest(prompt)),
            source_commit: "e".repeat(40),
            connection: "conn-1".to_string(),
        }
    }

    fn sample_launch() -> FactoryLaunch {
        FactoryLaunch {
            run: sample_run(),
            prompt: b"do the thing".to_vec(),
            harness_sha256: "f".repeat(64),
        }
    }

    fn sample_lease(run: &FactoryRun) -> Lease {
        Lease {
            provider_id: "codex".to_string(),
            id: "lease-1".to_string(),
            connection_id: run.connection.clone(),
            generation: 3,
            actor_id: run.actor,
            project_id: run.project.clone(),
            execution_id: run.id.clone(),
            kind: "factory".to_string(),
            role: run.role.clone(),
            deadline: run.deadline.clone(),
            ..Lease::default()
        }
    }

    fn sample_binding(run: &FactoryRun) -> Binding {
        Binding {
            scope: "factory-codex".to_string(),
            invocation_id: "09".repeat(16),
            kind: "factory".to_string(),
            id: run.id.clone(),
            project: container_id(),
            login: run.role.clone(),
            generation: 3,
            uid: 1001,
            gid: 1001,
            ..Binding::default()
        }
    }

    type ExecCall = (Vec<u8>, String, Vec<String>);

    struct FakeExec {
        calls: RefCell<Vec<ExecCall>>,
        script: RefCell<VecDeque<Result<Vec<u8>, String>>>,
    }

    impl FakeExec {
        fn new(responses: Vec<Result<Vec<u8>, String>>) -> Self {
            FakeExec {
                calls: RefCell::new(Vec::new()),
                script: RefCell::new(responses.into()),
            }
        }
    }

    impl Executor for FakeExec {
        fn run(
            &self,
            stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            self.calls.borrow_mut().push((
                stdin.to_vec(),
                cmd.to_string(),
                args.iter().map(|s| s.to_string()).collect(),
            ));
            self.script
                .borrow_mut()
                .pop_front()
                .unwrap_or(Err("no scripted exec response".to_string()))
        }
    }

    type TakeoverCall = (String, String, String, String, String, String);
    type ExportCall = (String, String, String, String, String);

    struct FakeTerminal {
        version: String,
        sha256: String,
        reserve: RefCell<VecDeque<Result<Binding, FactoryError>>>,
        start: RefCell<VecDeque<Result<(), FactoryError>>>,
        wait: RefCell<VecDeque<Result<(i64, String), FactoryError>>>,
        stop: RefCell<VecDeque<Result<(), FactoryError>>>,
        stop_unbound: RefCell<VecDeque<Result<(), FactoryError>>>,
        capture: RefCell<VecDeque<Result<Vec<u8>, FactoryError>>>,
        live: RefCell<VecDeque<bool>>,
        output: RefCell<VecDeque<Result<OutputSlice, FactoryError>>>,
        takeover: RefCell<VecDeque<Result<(String, bool), FactoryError>>>,
        export: RefCell<VecDeque<Result<Vec<u8>, FactoryError>>>,
        reserve_calls: RefCell<Vec<(String, i64)>>,
        takeover_calls: RefCell<Vec<TakeoverCall>>,
        export_calls: RefCell<Vec<ExportCall>>,
        output_calls: RefCell<Vec<(String, i64, i64)>>,
    }

    impl FakeTerminal {
        fn new() -> Self {
            FakeTerminal {
                version: "v1".to_string(),
                sha256: "f".repeat(64),
                reserve: RefCell::new(VecDeque::new()),
                start: RefCell::new(VecDeque::new()),
                wait: RefCell::new(VecDeque::new()),
                stop: RefCell::new(VecDeque::new()),
                stop_unbound: RefCell::new(VecDeque::new()),
                capture: RefCell::new(VecDeque::new()),
                live: RefCell::new(VecDeque::new()),
                output: RefCell::new(VecDeque::new()),
                takeover: RefCell::new(VecDeque::new()),
                export: RefCell::new(VecDeque::new()),
                reserve_calls: RefCell::new(Vec::new()),
                takeover_calls: RefCell::new(Vec::new()),
                export_calls: RefCell::new(Vec::new()),
                output_calls: RefCell::new(Vec::new()),
            }
        }

        fn pop<T>(queue: &RefCell<VecDeque<T>>, what: &str) -> T {
            queue
                .borrow_mut()
                .pop_front()
                .unwrap_or_else(|| panic!("no scripted {what}"))
        }
    }

    impl FactoryTerminal for FakeTerminal {
        fn harness_family(&self) -> String {
            FACTORY_HARNESS_CODEX.to_string()
        }
        fn harness_version(&self) -> String {
            self.version.clone()
        }
        fn harness_sha256(&self) -> String {
            self.sha256.clone()
        }
        fn reserve(
            &self,
            _run: &FactoryRun,
            _lease: &Lease,
            pin: &str,
            max_secs: i64,
            _deadline: Instant,
        ) -> Result<Binding, FactoryError> {
            self.reserve_calls
                .borrow_mut()
                .push((pin.to_string(), max_secs));
            Self::pop(&self.reserve, "reserve")
        }
        fn start(
            &self,
            _lease: &Lease,
            _credential: &[u8],
            _prompt: &[u8],
            _deadline: Instant,
        ) -> Result<(), FactoryError> {
            Self::pop(&self.start, "start")
        }
        fn wait(&self, _lease: &Lease, _deadline: Instant) -> Result<(i64, String), FactoryError> {
            Self::pop(&self.wait, "wait")
        }
        fn stop(&self, _lease: &Lease, _deadline: Instant) -> Result<(), FactoryError> {
            Self::pop(&self.stop, "stop")
        }
        fn stop_unbound(&self, _run: &FactoryRun, _deadline: Instant) -> Result<(), FactoryError> {
            Self::pop(&self.stop_unbound, "stop_unbound")
        }
        fn capture(&self, _lease: &Lease, _deadline: Instant) -> Result<Vec<u8>, FactoryError> {
            Self::pop(&self.capture, "capture")
        }
        fn live(&self, _binding: &Binding, _deadline: Instant) -> bool {
            Self::pop(&self.live, "live")
        }
        fn output(
            &self,
            project: &str,
            _binding: &Binding,
            offset: i64,
            limit: i64,
            _deadline: Instant,
        ) -> Result<OutputSlice, FactoryError> {
            self.output_calls
                .borrow_mut()
                .push((project.to_string(), offset, limit));
            Self::pop(&self.output, "output")
        }
        fn takeover_copy(
            &self,
            project: &str,
            recorded: &str,
            role: &str,
            preparation: &str,
            member: &str,
            run: &str,
            _deadline: Instant,
        ) -> Result<(String, bool), FactoryError> {
            self.takeover_calls.borrow_mut().push((
                project.to_string(),
                recorded.to_string(),
                role.to_string(),
                preparation.to_string(),
                member.to_string(),
                run.to_string(),
            ));
            Self::pop(&self.takeover, "takeover")
        }
        fn export_bundle(
            &self,
            project: &str,
            recorded: &str,
            role: &str,
            preparation: &str,
            candidate: &str,
            _deadline: Instant,
        ) -> Result<Vec<u8>, FactoryError> {
            self.export_calls.borrow_mut().push((
                project.to_string(),
                recorded.to_string(),
                role.to_string(),
                preparation.to_string(),
                candidate.to_string(),
            ));
            Self::pop(&self.export, "export")
        }
    }

    struct FakeBroker {
        acquire: RefCell<VecDeque<Result<Lease, FactoryError>>>,
        register: RefCell<VecDeque<Result<Vec<u8>, FactoryError>>>,
        returns: RefCell<VecDeque<Result<(), FactoryError>>>,
        terminal: RefCell<VecDeque<Result<bool, FactoryError>>>,
        close: RefCell<VecDeque<Result<(), FactoryError>>>,
        acquire_calls: RefCell<Vec<AcquireRequest>>,
        reconcile_calls: RefCell<Vec<String>>,
        close_calls: RefCell<Vec<(String, String)>>,
    }

    impl FakeBroker {
        fn new() -> Self {
            FakeBroker {
                acquire: RefCell::new(VecDeque::new()),
                register: RefCell::new(VecDeque::new()),
                returns: RefCell::new(VecDeque::new()),
                terminal: RefCell::new(VecDeque::new()),
                close: RefCell::new(VecDeque::new()),
                acquire_calls: RefCell::new(Vec::new()),
                reconcile_calls: RefCell::new(Vec::new()),
                close_calls: RefCell::new(Vec::new()),
            }
        }

        fn pop<T>(queue: &RefCell<VecDeque<T>>, what: &str) -> T {
            queue
                .borrow_mut()
                .pop_front()
                .unwrap_or_else(|| panic!("no scripted {what}"))
        }
    }

    impl FactoryBroker for FakeBroker {
        fn acquire(&self, req: &AcquireRequest, _deadline: Instant) -> Result<Lease, FactoryError> {
            self.acquire_calls.borrow_mut().push(req.clone());
            Self::pop(&self.acquire, "acquire")
        }
        fn register(
            &self,
            _lease_id: &str,
            _binding: &Binding,
            _deadline: Instant,
        ) -> Result<Vec<u8>, FactoryError> {
            Self::pop(&self.register, "register")
        }
        fn return_lease(
            &self,
            _lease_id: &str,
            _binding: &Binding,
            _credential: &[u8],
            _deadline: Instant,
        ) -> Result<(), FactoryError> {
            Self::pop(&self.returns, "return")
        }
        fn reconcile_lease(&self, lease_id: &str, _deadline: Instant) {
            self.reconcile_calls.borrow_mut().push(lease_id.to_string());
        }
        fn execution_is_terminal(
            &self,
            _kind: &str,
            _execution_id: &str,
            _deadline: Instant,
        ) -> Result<bool, FactoryError> {
            Self::pop(&self.terminal, "get-execution")
        }
        fn close_execution(
            &self,
            kind: &str,
            execution_id: &str,
            _deadline: Instant,
        ) -> Result<(), FactoryError> {
            self.close_calls
                .borrow_mut()
                .push((kind.to_string(), execution_id.to_string()));
            Self::pop(&self.close, "close")
        }
    }

    /// Script a full successful launch: acquire, reserve, register, start,
    /// wait, stop (retire), capture, return, close.
    fn script_success(
        term: &FakeTerminal,
        broker: &FakeBroker,
        run: &FactoryRun,
        exit: i64,
        output: &str,
    ) {
        broker.acquire.borrow_mut().push_back(Ok(sample_lease(run)));
        term.reserve.borrow_mut().push_back(Ok(sample_binding(run)));
        broker
            .register
            .borrow_mut()
            .push_back(Ok(b"{\"auth\":1}".to_vec()));
        term.start.borrow_mut().push_back(Ok(()));
        term.wait
            .borrow_mut()
            .push_back(Ok((exit, output.to_string())));
        term.stop.borrow_mut().push_back(Ok(()));
        term.capture
            .borrow_mut()
            .push_back(Ok(b"{\"auth\":2}".to_vec()));
        broker.returns.borrow_mut().push_back(Ok(()));
        broker.close.borrow_mut().push_back(Ok(()));
    }

    // Shared handles: the factory owns its seams, tests keep a clone.
    impl Executor for std::rc::Rc<FakeExec> {
        fn run(
            &self,
            stdin: &[u8],
            cmd: &str,
            args: &[&str],
            deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            (**self).run(stdin, cmd, args, deadline)
        }
    }

    impl FactoryTerminal for std::rc::Rc<FakeTerminal> {
        fn harness_family(&self) -> String {
            (**self).harness_family()
        }
        fn harness_version(&self) -> String {
            (**self).harness_version()
        }
        fn harness_sha256(&self) -> String {
            (**self).harness_sha256()
        }
        fn reserve(
            &self,
            run: &FactoryRun,
            lease: &Lease,
            pin: &str,
            max_secs: i64,
            deadline: Instant,
        ) -> Result<Binding, FactoryError> {
            (**self).reserve(run, lease, pin, max_secs, deadline)
        }
        fn start(
            &self,
            lease: &Lease,
            credential: &[u8],
            prompt: &[u8],
            deadline: Instant,
        ) -> Result<(), FactoryError> {
            (**self).start(lease, credential, prompt, deadline)
        }
        fn wait(&self, lease: &Lease, deadline: Instant) -> Result<(i64, String), FactoryError> {
            (**self).wait(lease, deadline)
        }
        fn stop(&self, lease: &Lease, deadline: Instant) -> Result<(), FactoryError> {
            (**self).stop(lease, deadline)
        }
        fn stop_unbound(&self, run: &FactoryRun, deadline: Instant) -> Result<(), FactoryError> {
            (**self).stop_unbound(run, deadline)
        }
        fn capture(&self, lease: &Lease, deadline: Instant) -> Result<Vec<u8>, FactoryError> {
            (**self).capture(lease, deadline)
        }
        fn live(&self, binding: &Binding, deadline: Instant) -> bool {
            (**self).live(binding, deadline)
        }
        fn output(
            &self,
            project: &str,
            binding: &Binding,
            offset: i64,
            limit: i64,
            deadline: Instant,
        ) -> Result<OutputSlice, FactoryError> {
            (**self).output(project, binding, offset, limit, deadline)
        }
        fn takeover_copy(
            &self,
            project: &str,
            recorded: &str,
            role: &str,
            preparation: &str,
            member: &str,
            run: &str,
            deadline: Instant,
        ) -> Result<(String, bool), FactoryError> {
            (**self).takeover_copy(project, recorded, role, preparation, member, run, deadline)
        }
        fn export_bundle(
            &self,
            project: &str,
            recorded: &str,
            role: &str,
            preparation: &str,
            candidate: &str,
            deadline: Instant,
        ) -> Result<Vec<u8>, FactoryError> {
            (**self).export_bundle(project, recorded, role, preparation, candidate, deadline)
        }
    }

    impl FactoryBroker for std::rc::Rc<FakeBroker> {
        fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, FactoryError> {
            (**self).acquire(req, deadline)
        }
        fn register(
            &self,
            lease_id: &str,
            binding: &Binding,
            deadline: Instant,
        ) -> Result<Vec<u8>, FactoryError> {
            (**self).register(lease_id, binding, deadline)
        }
        fn return_lease(
            &self,
            lease_id: &str,
            binding: &Binding,
            credential: &[u8],
            deadline: Instant,
        ) -> Result<(), FactoryError> {
            (**self).return_lease(lease_id, binding, credential, deadline)
        }
        fn reconcile_lease(&self, lease_id: &str, deadline: Instant) {
            (**self).reconcile_lease(lease_id, deadline);
        }
        fn execution_is_terminal(
            &self,
            kind: &str,
            execution_id: &str,
            deadline: Instant,
        ) -> Result<bool, FactoryError> {
            (**self).execution_is_terminal(kind, execution_id, deadline)
        }
        fn close_execution(
            &self,
            kind: &str,
            execution_id: &str,
            deadline: Instant,
        ) -> Result<(), FactoryError> {
            (**self).close_execution(kind, execution_id, deadline)
        }
    }

    type TestFactory =
        Factory<std::rc::Rc<FakeExec>, std::rc::Rc<FakeTerminal>, std::rc::Rc<FakeBroker>>;

    fn test_factory(
        dir: &std::path::Path,
        exec: std::rc::Rc<FakeExec>,
        term: std::rc::Rc<FakeTerminal>,
        broker: std::rc::Rc<FakeBroker>,
    ) -> TestFactory {
        Factory::open_factory(dir.to_str().unwrap(), exec, term, broker).unwrap()
    }

    fn receipt_bytes(dir: &std::path::Path, project: &str, run: &str) -> Vec<u8> {
        std::fs::read(dir.join(format!("{project}-{run}.json"))).unwrap()
    }

    #[test]
    fn validators_match_go() {
        assert!(valid_factory_phase("approved"));
        assert!(valid_factory_phase("uncertain"));
        assert!(!valid_factory_phase("waiting"));
        assert!(!valid_factory_phase(""));
        assert!(valid_factory_run_id(&"a".repeat(32)));
        assert!(valid_factory_run_id(&"09af".repeat(8)));
        assert!(!valid_factory_run_id(&"A".repeat(32)));
        assert!(!valid_factory_run_id(&"a".repeat(31)));
        assert!(!valid_factory_run_id(""));
        assert!(valid_harness_version("v1"));
        assert!(valid_harness_version(&"a".repeat(32)));
        assert!(valid_harness_version("1.2-3_4"));
        assert!(!valid_harness_version(""));
        assert!(!valid_harness_version(&"a".repeat(33)));
        assert!(!valid_harness_version("-v1"));
        assert!(!valid_harness_version("v 1"));
        assert_eq!(
            factory_unit_name(&"a".repeat(32)),
            format!("soda-factory-{}.service", "a".repeat(32))
        );
        assert_eq!(factory_unit_name("bogus"), "");
        let (checkout, run_dir, home, codex) = factory_run_paths(
            "soda-coder",
            &format!("f{}", "c".repeat(24)),
            &"a".repeat(32),
        );
        assert_eq!(
            checkout,
            format!("/home/soda-coder/checkouts/f{}", "c".repeat(24))
        );
        assert!(run_dir.ends_with(&format!("/.soda-home/runs/{}", "a".repeat(32))));
        assert!(home.ends_with("/home"));
        assert!(codex.ends_with("/home/.codex"));
        assert_eq!(
            factory_run_paths("root", "f", "g"),
            (String::new(), String::new(), String::new(), String::new())
        );
        assert_eq!(
            takeover_destination("alice", &"a".repeat(32)),
            format!("/home/alice/factory-takeover/{}", "a".repeat(32))
        );
        assert_eq!(takeover_destination("root", &"a".repeat(32)), "");
        assert!(takeover_source(
            &format!("/home/soda-coder/checkouts/f{}", "c".repeat(24)),
            "soda-coder",
            &format!("f{}", "c".repeat(24))
        ));
        assert!(!takeover_source(
            "/home/soda-coder/checkouts/f",
            "soda-coder",
            "f"
        ));
        assert!(!takeover_source(
            "/home/soda-coder/checkouts/x/",
            "soda-coder",
            &format!("f{}", "c".repeat(24))
        ));
    }

    #[test]
    fn run_and_launch_validation_pins_every_error() {
        let good = sample_run();
        assert!(good.validate().is_ok());
        let mut bad = good.clone();
        bad.id = "short".to_string();
        assert_eq!(bad.validate().unwrap_err(), "invalid factory run identity");
        bad = good.clone();
        bad.role = "root".to_string();
        assert_eq!(bad.validate().unwrap_err(), "invalid factory run identity");
        bad = good.clone();
        bad.preparation = "bogus".to_string();
        assert_eq!(
            bad.validate().unwrap_err(),
            "invalid run preparation reference"
        );
        bad = good.clone();
        bad.harness = "Muse".to_string();
        assert_eq!(bad.validate().unwrap_err(), "unsupported factory harness");
        bad = good.clone();
        bad.harness_vers = String::new();
        assert_eq!(bad.validate().unwrap_err(), "unsupported factory harness");
        bad = good.clone();
        bad.model = "x".repeat(129);
        assert_eq!(bad.validate().unwrap_err(), "invalid run model selection");
        bad = good.clone();
        bad.model = "gpt-\u{7f}".to_string();
        assert_eq!(bad.validate().unwrap_err(), "invalid run model selection");
        bad = good.clone();
        bad.model = "line\nbreak".to_string();
        assert_eq!(bad.validate().unwrap_err(), "invalid run model selection");
        bad = good.clone();
        bad.model = "gpt-5".to_string();
        assert!(bad.validate().is_ok());
        bad = good.clone();
        bad.assignment = "zz".to_string();
        assert_eq!(
            bad.validate().unwrap_err(),
            "invalid run assignment or source identity"
        );
        bad = good.clone();
        bad.connection = String::new();
        assert_eq!(bad.validate().unwrap_err(), "invalid run sponsorship");
        bad = good.clone();
        bad.actor = 0;
        assert_eq!(bad.validate().unwrap_err(), "invalid run sponsorship");
        bad = good.clone();
        bad.deadline = String::new();
        assert_eq!(bad.validate().unwrap_err(), "run deadline is required");
        bad = good.clone();
        bad.deadline = "0001-01-01T00:00:00Z".to_string();
        assert_eq!(bad.validate().unwrap_err(), "run deadline is required");
        bad = good.clone();
        bad.deadline = "not-a-time".to_string();
        assert_eq!(bad.validate().unwrap_err(), "run deadline is required");

        let launch = sample_launch();
        assert!(launch.validate().is_ok());
        let mut bad_launch = launch.clone();
        bad_launch.prompt = Vec::new();
        assert_eq!(
            bad_launch.validate().unwrap_err(),
            "invalid run prompt size"
        );
        bad_launch = launch.clone();
        bad_launch.prompt = vec![b'x'; MAX_FACTORY_PROMPT + 1];
        assert_eq!(
            bad_launch.validate().unwrap_err(),
            "invalid run prompt size"
        );
        bad_launch = launch.clone();
        bad_launch.prompt = b"something else".to_vec();
        assert_eq!(
            bad_launch.validate().unwrap_err(),
            "prompt bytes do not match their digest"
        );
        bad_launch = launch.clone();
        bad_launch.harness_sha256 = "zz".to_string();
        assert_eq!(bad_launch.validate().unwrap_err(), "invalid harness pin");
    }

    #[test]
    fn deadline_parser_matches_go_shapes() {
        assert!(parse_deadline("2030-01-02T03:04:05Z").is_some());
        assert!(parse_deadline("2030-01-02T03:04:05.123456789Z").is_some());
        assert!(parse_deadline("2030-01-02T03:04:05.1+02:00").is_some());
        assert!(parse_deadline("2030-01-02T03:04:05-05:30").is_some());
        // Same instant, different offsets.
        assert_eq!(
            parse_deadline("2030-01-02T03:04:05Z"),
            parse_deadline("2030-01-02T05:04:05+02:00")
        );
        // Fraction precision.
        let whole = parse_deadline("2030-01-02T03:04:05Z").unwrap();
        let frac = parse_deadline("2030-01-02T03:04:05.000000001Z").unwrap();
        assert_eq!(frac - whole, 1);
        // Rejections.
        for bad in [
            "",
            "2030-01-02",
            "2030-01-02T03:04:05",
            "2030-13-02T03:04:05Z",
            "2030-01-32T03:04:05Z",
            "2029-02-29T03:04:05Z",
            "2030-01-02T24:04:05Z",
            "2030-01-02T03:60:05Z",
            "2030-01-02T03:04:60Z",
            "2030-01-02T03:04:05.Z",
            "2030-01-02T03:04:05.1234567890Z",
            "2030-01-02T03:04:05+24:00",
            "2030-01-02T03:04:05+02:60",
            "2030-01-02T03:04:05+0200",
            "2030-01-02 03:04:05Z",
        ] {
            assert!(parse_deadline(bad).is_none(), "accepted {bad:?}");
        }
        assert!(deadline_is_zero("0001-01-01T00:00:00Z"));
        assert!(!deadline_is_zero("0001-01-01T00:00:01Z"));
        assert!(!deadline_is_zero("garbage"));
    }

    #[test]
    fn error_messages_are_exact() {
        assert_eq!(
            FactoryError::NotFound.message(),
            "identity execution missing"
        );
        assert_eq!(FactoryError::Stale.message(), "identity generation changed");
        assert_eq!(FactoryError::Busy.message(), "subscription is in use");
        assert_eq!(FactoryError::Denied.message(), "identity authority denied");
        assert_eq!(
            FactoryError::Uncertain.message(),
            "subscription requires reconnection"
        );
        assert_eq!(
            FactoryError::DeadlineExceeded.message(),
            "context deadline exceeded"
        );
        assert_eq!(
            FactoryError::ExportCandidate.message(),
            "export candidate is not recorded"
        );
        assert_eq!(
            FactoryError::ExportBounds.message(),
            "candidate export exceeds bounds"
        );
        assert_eq!(ERR_RUN_NOT_FOUND, "factory run not found");
        assert_eq!(ERR_RUN_STALE, "factory run incarnation changed");
        assert_eq!(FACTORY_STATE_ROOT, "/var/lib/soda/host/factory");
    }

    fn fixed_run() -> FactoryRun {
        FactoryRun {
            deadline: "2030-06-07T08:09:10Z".to_string(),
            actor: 7,
            id: "a".repeat(32),
            project: format!("p{}", "b".repeat(24)),
            role: "soda-coder".to_string(),
            preparation: format!("f{}", "c".repeat(24)),
            harness: "codex".to_string(),
            harness_vers: "v1".to_string(),
            model: String::new(),
            assignment: "0".repeat(64),
            source_commit: "e".repeat(40),
            connection: "conn-1".to_string(),
        }
    }

    fn fixed_run_json() -> String {
        format!(
            "{{\"deadline\":\"2030-06-07T08:09:10Z\",\"actor\":7,\"id\":\"{}\",\"project\":\"p{}\",\"role\":\"soda-coder\",\"preparation\":\"f{}\",\"harness\":\"codex\",\"harness_version\":\"v1\",\"assignment\":\"{}\",\"source_commit\":\"{}\",\"connection\":\"conn-1\"}}",
            "a".repeat(32),
            "b".repeat(24),
            "c".repeat(24),
            "0".repeat(64),
            "e".repeat(40)
        )
    }

    fn fixed_binding() -> Binding {
        Binding {
            scope: "factory-codex".to_string(),
            invocation_id: "09".repeat(16),
            kind: "factory".to_string(),
            id: "a".repeat(32),
            project: "d".repeat(64),
            login: "soda-coder".to_string(),
            generation: 3,
            uid: 1001,
            gid: 1001,
            ..Binding::default()
        }
    }

    fn fixed_binding_json() -> String {
        format!(
            "{{\"uid\":1001,\"gid\":1001,\"scope\":\"factory-codex\",\"invocation_id\":\"{}\",\"kind\":\"factory\",\"id\":\"{}\",\"project\":\"{}\",\"login\":\"soda-coder\",\"generation\":3}}",
            "09".repeat(16),
            "a".repeat(32),
            "d".repeat(64)
        )
    }

    fn fixed_lease() -> Lease {
        Lease {
            provider_id: "codex".to_string(),
            id: "lease-1".to_string(),
            connection_id: "conn-1".to_string(),
            generation: 3,
            actor_id: 7,
            project_id: format!("p{}", "b".repeat(24)),
            execution_id: "a".repeat(32),
            kind: "factory".to_string(),
            role: "soda-coder".to_string(),
            deadline: "2030-06-07T08:09:10Z".to_string(),
            ..Lease::default()
        }
    }

    fn fixed_lease_json() -> String {
        format!(
            "{{\"provider_id\":\"codex\",\"id\":\"lease-1\",\"connection_id\":\"conn-1\",\"generation\":3,\"actor_id\":\"7\",\"project_id\":\"p{}\",\"execution_id\":\"{}\",\"kind\":\"factory\",\"role\":\"soda-coder\",\"deadline\":\"2030-06-07T08:09:10Z\"}}",
            "b".repeat(24),
            "a".repeat(32)
        )
    }

    fn dummy_factory(dir: &std::path::Path) -> TestFactory {
        use std::rc::Rc;
        test_factory(
            dir,
            Rc::new(FakeExec::new(Vec::new())),
            Rc::new(FakeTerminal::new()),
            Rc::new(FakeBroker::new()),
        )
    }

    #[test]
    fn receipt_bytes_match_go_marshal() {
        let dir = test_state_dir("receipt-bytes");
        let factory = dummy_factory(&dir);
        // Approved receipt: everything optional omitted.
        let approved = FactoryReceipt {
            run: fixed_run(),
            phase: "approved".to_string(),
            ..FactoryReceipt::default()
        };
        factory.store_receipt(&approved).unwrap();
        let path = dir.join(format!("p{}-{}.json", "b".repeat(24), "a".repeat(32)));
        let data = std::fs::read(&path).unwrap();
        let expected = format!(
            "{{\"run\":{},\"phase\":\"approved\",\"started\":false,\"delivered\":false,\"credential_returned\":false}}",
            fixed_run_json()
        );
        assert_eq!(data, expected.as_bytes());
        // Mode 0600 on the stored receipt.
        use std::os::unix::fs::PermissionsExt;
        assert_eq!(
            std::fs::metadata(&path).unwrap().permissions().mode() & 0o777,
            0o600
        );
        // No temporary siblings left behind.
        let leftovers: Vec<_> = std::fs::read_dir(&dir)
            .unwrap()
            .filter_map(|e| e.ok())
            .map(|e| e.file_name().into_string().unwrap())
            .filter(|n| n.starts_with(".receipt-"))
            .collect();
        assert!(leftovers.is_empty());

        // Completed receipt with lease, binding, exit and output.
        let completed = FactoryReceipt {
            run: fixed_run(),
            lease: Some(fixed_lease()),
            binding: Some(fixed_binding()),
            exit_code: Some(0),
            generation: 3,
            phase: "completed".to_string(),
            started: true,
            delivered: true,
            credential_returned: true,
            output: "done\n".to_string(),
            retirement: "confirmed".to_string(),
            ..FactoryReceipt::default()
        };
        factory.store_receipt(&completed).unwrap();
        let data = std::fs::read(&path).unwrap();
        let expected = format!(
            "{{\"run\":{},\"lease\":{},\"binding\":{},\"exit_code\":0,\"generation\":3,\"phase\":\"completed\",\"started\":true,\"delivered\":true,\"credential_returned\":true,\"output\":\"done\\n\",\"retirement\":\"confirmed\"}}",
            fixed_run_json(),
            fixed_lease_json(),
            fixed_binding_json()
        );
        assert_eq!(data, expected.as_bytes());

        // Tombstones skip validation but keep the exact shape.
        let tombstone = FactoryReceipt {
            run: FactoryRun {
                id: "a".repeat(32),
                project: format!("p{}", "b".repeat(24)),
                deadline: "0001-01-01T00:00:00Z".to_string(),
                ..FactoryRun::default()
            },
            phase: "stopped".to_string(),
            retirement: "confirmed".to_string(),
            reason: "stop-before-start".to_string(),
            ..FactoryReceipt::default()
        };
        factory.store_tombstone(&tombstone).unwrap();
        let data = std::fs::read(&path).unwrap();
        assert!(data.starts_with(
            format!(
                "{{\"run\":{{\"deadline\":\"0001-01-01T00:00:00Z\",\"actor\":0,\"id\":\"{}\"",
                "a".repeat(32)
            )
            .as_bytes()
        ));
        assert!(data.ends_with(
            b"\"phase\":\"stopped\",\"started\":false,\"delivered\":false,\"credential_returned\":false,\"retirement\":\"confirmed\",\"reason\":\"stop-before-start\"}"
        ));
    }

    #[test]
    fn receipt_decode_matrix() {
        let dir = test_state_dir("receipt-decode");
        let factory = dummy_factory(&dir);
        let project = format!("p{}", "b".repeat(24));
        let run = "a".repeat(32);
        let path = dir.join(format!("{project}-{run}.json"));

        // Round trip of a full receipt, including a nested lease binding.
        let mut lease = fixed_lease();
        lease.binding = Some(fixed_binding());
        let full = FactoryReceipt {
            run: fixed_run(),
            lease: Some(lease),
            binding: Some(fixed_binding()),
            exit_code: Some(3),
            generation: 9,
            phase: "failed".to_string(),
            started: true,
            delivered: true,
            credential_returned: true,
            output: "x".to_string(),
            retirement: "confirmed".to_string(),
            reason: "execution-failed".to_string(),
        };
        factory.store_receipt(&full).unwrap();
        let (loaded, exists) = factory.load_receipt(&project, &run).unwrap();
        assert!(exists);
        assert_eq!(loaded.run, full.run);
        assert_eq!(loaded.exit_code, Some(3));
        assert_eq!(loaded.generation, 9);
        assert_eq!(loaded.reason, "execution-failed");
        assert_eq!(loaded.lease.as_ref().unwrap().actor_id, 7);
        assert!(loaded.lease.as_ref().unwrap().binding.is_some());

        // Missing receipt reports absence.
        let (_, exists) = factory.load_receipt(&project, &"b".repeat(32)).unwrap();
        assert!(!exists);

        // Oversized.
        std::fs::write(&path, vec![b'x'; (128 << 10) + 1]).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "run receipt exceeds bounds"
        );
        // Truncated JSON.
        std::fs::write(&path, b"{\"run\":").unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "invalid run receipt"
        );
        // Unknown field (DisallowUnknownFields).
        let mut raw = full.encode();
        raw.pop();
        raw.push_str(",\"extra\":1}");
        std::fs::write(&path, raw.as_bytes()).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "invalid run receipt"
        );
        // Identity mismatch.
        let mismatch = full.encode().replacen(&run, &"b".repeat(32), 1);
        std::fs::write(&path, mismatch.as_bytes()).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "run receipt identity mismatch"
        );
        // Tombstone with a lease is corrupt.
        let bad_tomb = format!(
            "{{\"run\":{{\"deadline\":\"0001-01-01T00:00:00Z\",\"actor\":0,\"id\":\"{run}\",\"project\":\"{project}\",\"role\":\"\",\"preparation\":\"\",\"harness\":\"\",\"harness_version\":\"\",\"assignment\":\"\",\"source_commit\":\"\",\"connection\":\"\"}},\"lease\":{},\"phase\":\"stopped\",\"started\":false,\"delivered\":false,\"credential_returned\":false}}",
            fixed_lease_json()
        );
        std::fs::write(&path, bad_tomb.as_bytes()).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "invalid run tombstone"
        );
        // Bad phase.
        let bad_phase = full
            .encode()
            .replace("\"phase\":\"failed\"", "\"phase\":\"waiting\"");
        std::fs::write(&path, bad_phase.as_bytes()).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "invalid run phase"
        );
        // Lease identity mismatch.
        let bad_lease = full.encode().replace(
            &format!("\"execution_id\":\"{run}\""),
            &format!("\"execution_id\":\"{}\"", "b".repeat(32)),
        );
        std::fs::write(&path, bad_lease.as_bytes()).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "run lease identity mismatch"
        );
        // Record exceeds bounds.
        let bad_bounds = full.encode().replace(
            "\"reason\":\"execution-failed\"",
            &format!("\"reason\":\"{}\"", "r".repeat(257)),
        );
        std::fs::write(&path, bad_bounds.as_bytes()).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "run record exceeds bounds"
        );
        // Malformed `,string` lease integer.
        let bad_actor = full
            .encode()
            .replace("\"actor_id\":\"7\"", "\"actor_id\":\"zz\"");
        std::fs::write(&path, bad_actor.as_bytes()).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "invalid run receipt"
        );
        // Bare (unquoted) `,string` integer is rejected like Go.
        let bare_actor = full
            .encode()
            .replace("\"actor_id\":\"7\"", "\"actor_id\":7");
        std::fs::write(&path, bare_actor.as_bytes()).unwrap();
        assert_eq!(
            factory.load_receipt(&project, &run).unwrap_err().message(),
            "invalid run receipt"
        );
    }

    #[test]
    fn open_factory_matrix() {
        use std::rc::Rc;
        let term = Rc::new(FakeTerminal::new());
        let broker = Rc::new(FakeBroker::new());
        let exec = Rc::new(FakeExec::new(Vec::new()));
        assert_eq!(
            Factory::open_factory("relative/path", exec.clone(), term.clone(), broker.clone())
                .err()
                .unwrap(),
            "factory receipt directory must be absolute"
        );
        assert!(Factory::open_factory(
            "/tmp/soda-pf26-definitely-missing",
            exec.clone(),
            term.clone(),
            broker.clone()
        )
        .is_err());
        // A regular file is not a directory.
        let file = test_state_dir("open-file").join("f");
        std::fs::write(&file, b"x").unwrap();
        assert_eq!(
            Factory::open_factory(
                file.to_str().unwrap(),
                exec.clone(),
                term.clone(),
                broker.clone()
            )
            .err()
            .unwrap(),
            "factory receipts must live in a private directory"
        );
        // Group/world-readable directories are refused.
        let open = test_state_dir("open-perms");
        std::fs::set_permissions(&open, std::fs::Permissions::from_mode(0o755)).unwrap();
        assert_eq!(
            Factory::open_factory(
                open.to_str().unwrap(),
                exec.clone(),
                term.clone(),
                broker.clone()
            )
            .err()
            .unwrap(),
            "factory receipts must live in a private directory"
        );
        // A symlink is not a directory (Lstat semantics).
        let target = test_state_dir("open-target");
        let link = std::env::temp_dir().join(format!(
            "soda-pf26-open-link-{}-{}",
            std::process::id(),
            TAG_COUNTER.fetch_add(1, Ordering::SeqCst)
        ));
        std::os::unix::fs::symlink(&target, &link).unwrap();
        assert_eq!(
            Factory::open_factory(
                link.to_str().unwrap(),
                exec.clone(),
                term.clone(),
                broker.clone()
            )
            .err()
            .unwrap(),
            "factory receipts must live in a private directory"
        );
        std::fs::remove_file(&link).unwrap();
        // Private directory opens.
        let dir = test_state_dir("open-ok");
        assert!(Factory::open_factory(dir.to_str().unwrap(), exec, term, broker).is_ok());
    }

    #[test]
    fn harness_pin_shape_matches_go() {
        let dir = test_state_dir("harness");
        let factory = dummy_factory(&dir);
        let pin = factory.harness_pin();
        assert_eq!(pin.harness, "codex");
        assert_eq!(pin.version, "v1");
        assert_eq!(pin.sha256, "f".repeat(64));
        // The daemon route fills the image from its own configuration.
        assert_eq!(pin.image, "");
        assert_eq!(
            pin.encode(),
            format!(
                "{{\"harness\":\"codex\",\"version\":\"v1\",\"sha256\":\"{}\",\"image\":\"\"}}",
                "f".repeat(64)
            )
        );
        // Unpinned harness is unusable, like the Go client check.
        assert_eq!(
            confirm_factory_harness(&pin).unwrap_err(),
            "native factory harness pin is not usable"
        );
        let mut pinned = pin;
        pinned.image = format!("sha256:{}", "a".repeat(64));
        assert!(confirm_factory_harness(&pinned).is_ok());
    }

    fn wired_factory(
        tag: &str,
    ) -> (
        std::path::PathBuf,
        TestFactory,
        std::rc::Rc<FakeExec>,
        std::rc::Rc<FakeTerminal>,
        std::rc::Rc<FakeBroker>,
    ) {
        use std::rc::Rc;
        let dir = test_state_dir(tag);
        let exec = Rc::new(FakeExec::new(Vec::new()));
        let term = Rc::new(FakeTerminal::new());
        let broker = Rc::new(FakeBroker::new());
        let factory = test_factory(&dir, exec.clone(), term.clone(), broker.clone());
        (dir, factory, exec, term, broker)
    }

    #[test]
    fn launch_success_drives_to_completed() {
        let (dir, factory, _exec, term, broker) = wired_factory("launch-ok");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "all done");
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.id, req.run.id);
        assert_eq!(state.project, req.run.project);
        assert_eq!(state.role, "soda-coder");
        assert_eq!(state.phase, "completed");
        assert_eq!(state.exit_code, Some(0));
        assert_eq!(state.output, "all done");
        assert_eq!(state.retirement, "confirmed");
        assert_eq!(state.reason, "");
        assert!(state.credential_returned);
        assert!(state.delivered);
        assert!(!state.live);
        assert_eq!(state.generation, 3);
        assert_eq!(state.lease_id, "lease-1");
        assert_eq!(state.container, container_id());
        assert_eq!(state.unit, factory_unit_name(&req.run.id));
        assert_eq!(state.invocation, "09".repeat(16));
        assert_eq!(state.login, "soda-coder");
        assert_eq!(state.uid, 1001);
        assert_eq!(state.gid, 1001);
        // The broker saw the exact acquisition identity.
        let calls = broker.acquire_calls.borrow();
        assert_eq!(calls.len(), 1);
        assert_eq!(
            calls[0],
            AcquireRequest {
                provider_id: "codex".to_string(),
                execution_id: req.run.id.clone(),
                actor_id: 7,
                connection_id: "conn-1".to_string(),
                project_id: req.run.project.clone(),
                kind: "factory".to_string(),
                deadline: req.run.deadline.clone(),
                role: "soda-coder".to_string(),
            }
        );
        // Reserve got the caller pin and a sane second bound.
        let reserves = term.reserve_calls.borrow();
        assert_eq!(reserves.len(), 1);
        assert_eq!(reserves[0].0, "f".repeat(64));
        assert!(reserves[0].1 > 3500 && reserves[0].1 <= 3600);
        // The execution fence closed exactly once.
        assert_eq!(
            broker.close_calls.borrow().as_slice(),
            &[(String::from("factory"), req.run.id.clone())]
        );
        // The durable receipt carries the outcome.
        let data = receipt_bytes(&dir, &req.run.project, &req.run.id);
        let text = String::from_utf8(data).unwrap();
        assert!(text.contains("\"phase\":\"completed\""), "{text}");
        assert!(text.contains("\"exit_code\":0"), "{text}");
        assert!(text.contains("\"credential_returned\":true"), "{text}");
        // Exact state encoding.
        assert!(state.encode().starts_with(
            "{\"exit_code\":0,\"generation\":3,\"uid\":1001,\"gid\":1001,\"credential_returned\":true,\"live\":false,\"delivered\":true"
        ));
    }

    #[test]
    fn launch_duplicate_returns_recorded_state() {
        let (_dir, factory, _exec, term, broker) = wired_factory("launch-dup");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        let first = factory.launch(&req, deadline()).unwrap();
        assert_eq!(first.phase, "completed");
        // No further scripts: a duplicate must not touch any seam.
        let second = factory.launch(&req, deadline()).unwrap();
        assert_eq!(second, first);
        assert_eq!(broker.acquire_calls.borrow().len(), 1);
    }

    #[test]
    fn launch_prechecks_reject_before_any_seam() {
        let (_dir, factory, _exec, term, broker) = wired_factory("launch-pre");
        // Invalid request.
        let mut bad = sample_launch();
        bad.run.id = "short".to_string();
        assert_eq!(
            factory.launch(&bad, deadline()).unwrap_err().message(),
            "invalid factory run identity"
        );
        // Harness mismatch.
        let mut bad = sample_launch();
        bad.harness_sha256 = "0".repeat(64);
        assert_eq!(
            factory.launch(&bad, deadline()).unwrap_err().message(),
            "factory harness is unavailable until its own proof passes"
        );
        // Past deadline.
        let mut bad = sample_launch();
        bad.run.deadline = deadline_text(-10);
        assert_eq!(
            factory.launch(&bad, deadline()).unwrap_err().message(),
            "run deadline is outside the supervised bound"
        );
        // Beyond the three-hour supervised bound.
        let mut bad = sample_launch();
        bad.run.deadline = deadline_text(3 * 3600 + 60);
        assert_eq!(
            factory.launch(&bad, deadline()).unwrap_err().message(),
            "run deadline is outside the supervised bound"
        );
        // Nothing reached the broker or the terminal.
        assert!(broker.acquire_calls.borrow().is_empty());
        assert!(term.reserve_calls.borrow().is_empty());
    }

    #[test]
    fn launch_acquire_refusal_matrix() {
        // (cause, expected phase, expected reason, stores receipt, propagates)
        let cases: Vec<(FactoryError, &str, &str, bool)> = vec![
            (FactoryError::Busy, "approved", "broker-busy", true),
            (FactoryError::Denied, "failed", "broker-denied", true),
            (
                FactoryError::Uncertain,
                "failed",
                "broker-unavailable",
                true,
            ),
            (
                FactoryError::DeadlineExceeded,
                "failed",
                "deadline-exceeded",
                true,
            ),
        ];
        for (cause, phase, reason, _) in cases {
            let (dir, factory, _exec, _term, broker) = wired_factory("launch-acq");
            let req = sample_launch();
            broker.acquire.borrow_mut().push_back(Err(cause));
            let state = factory.launch(&req, deadline()).unwrap();
            assert_eq!(state.phase, phase);
            assert_eq!(state.reason, reason);
            assert_eq!(state.id, req.run.id);
            let text =
                String::from_utf8(receipt_bytes(&dir, &req.run.project, &req.run.id)).unwrap();
            assert!(text.contains(&format!("\"phase\":\"{phase}\"")), "{text}");
            assert!(text.contains(&format!("\"reason\":\"{reason}\"")), "{text}");
        }
        // Transport failures stay errors and store nothing new.
        let (dir, factory, _exec, _term, broker) = wired_factory("launch-acq-err");
        let req = sample_launch();
        broker
            .acquire
            .borrow_mut()
            .push_back(Err(FactoryError::msg("identity broker unavailable")));
        assert_eq!(
            factory.launch(&req, deadline()).unwrap_err().message(),
            "identity broker unavailable"
        );
        let text = String::from_utf8(receipt_bytes(&dir, &req.run.project, &req.run.id)).unwrap();
        assert!(text.contains("\"phase\":\"approved\""), "{text}");
        assert!(!text.contains("\"reason\""), "{text}");
    }

    #[test]
    fn launch_abandon_paths_record_refusals() {
        for (refusal, queue) in [("reserve-refused", 0), ("register-refused", 1)] {
            let (dir, factory, _exec, term, broker) = wired_factory("launch-abandon");
            let req = sample_launch();
            broker
                .acquire
                .borrow_mut()
                .push_back(Ok(sample_lease(&req.run)));
            if queue == 0 {
                term.reserve
                    .borrow_mut()
                    .push_back(Err(FactoryError::msg("reserve blew up")));
            } else {
                term.reserve
                    .borrow_mut()
                    .push_back(Ok(sample_binding(&req.run)));
                broker
                    .register
                    .borrow_mut()
                    .push_back(Err(FactoryError::msg("register blew up")));
                term.stop.borrow_mut().push_back(Ok(()));
            }
            // Register refusal closes twice (drive, then abandonRun), like Go.
            broker.close.borrow_mut().push_back(Ok(()));
            if queue == 1 {
                broker.close.borrow_mut().push_back(Ok(()));
            }
            let state = factory.launch(&req, deadline()).unwrap();
            assert_eq!(state.phase, "failed");
            assert_eq!(state.reason, refusal);
            // Like Go's abandonRun: phase and reason only, no retirement.
            assert_eq!(state.retirement, "");
            assert_eq!(broker.close_calls.borrow().len(), queue + 1);
            let text =
                String::from_utf8(receipt_bytes(&dir, &req.run.project, &req.run.id)).unwrap();
            assert!(
                text.contains(&format!("\"reason\":\"{refusal}\"")),
                "{text}"
            );
        }
    }

    fn drive_to_start(
        term: &std::rc::Rc<FakeTerminal>,
        broker: &std::rc::Rc<FakeBroker>,
        run: &FactoryRun,
    ) {
        broker.acquire.borrow_mut().push_back(Ok(sample_lease(run)));
        term.reserve.borrow_mut().push_back(Ok(sample_binding(run)));
        broker
            .register
            .borrow_mut()
            .push_back(Ok(b"{\"auth\":1}".to_vec()));
    }

    #[test]
    fn launch_failed_start_retires_cleanly() {
        let (dir, factory, _exec, term, broker) = wired_factory("launch-startfail");
        let req = sample_launch();
        drive_to_start(&term, &broker, &req.run);
        term.start
            .borrow_mut()
            .push_back(Err(FactoryError::msg("start blew up")));
        term.stop.borrow_mut().push_back(Ok(()));
        term.capture
            .borrow_mut()
            .push_back(Ok(b"{\"auth\":2}".to_vec()));
        broker.returns.borrow_mut().push_back(Ok(()));
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, "failed");
        assert_eq!(state.reason, "start-unconfirmed");
        assert_eq!(state.retirement, "confirmed");
        assert!(state.credential_returned);
        let text = String::from_utf8(receipt_bytes(&dir, &req.run.project, &req.run.id)).unwrap();
        assert!(text.contains("\"phase\":\"failed\""), "{text}");
    }

    #[test]
    fn launch_failed_start_uncertainty_matrix() {
        // Each custody failure flips the run uncertain.
        for case in ["return", "capture", "close"] {
            let (_dir, factory, _exec, term, broker) = wired_factory("launch-startunc");
            let req = sample_launch();
            drive_to_start(&term, &broker, &req.run);
            term.start
                .borrow_mut()
                .push_back(Err(FactoryError::msg("nope")));
            term.stop.borrow_mut().push_back(Ok(()));
            match case {
                "return" => {
                    term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
                    broker
                        .returns
                        .borrow_mut()
                        .push_back(Err(FactoryError::msg("return blew up")));
                }
                "capture" => {
                    term.capture
                        .borrow_mut()
                        .push_back(Err(FactoryError::msg("capture blew up")));
                }
                _ => {
                    term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
                    broker.returns.borrow_mut().push_back(Ok(()));
                }
            }
            broker.close.borrow_mut().push_back(if case == "close" {
                Err(FactoryError::msg("close blew up"))
            } else {
                Ok(())
            });
            let state = factory.launch(&req, deadline()).unwrap();
            assert_eq!(state.phase, "uncertain", "case {case}");
            assert_eq!(state.reason, "start-unconfirmed", "case {case}");
            assert_eq!(state.retirement, "uncertain", "case {case}");
        }
    }

    #[test]
    fn launch_timed_out_wait_records_deadline() {
        let (_dir, factory, _exec, term, broker) = wired_factory("launch-timeout");
        let req = sample_launch();
        drive_to_start(&term, &broker, &req.run);
        term.start.borrow_mut().push_back(Ok(()));
        term.wait
            .borrow_mut()
            .push_back(Err(FactoryError::DeadlineExceeded));
        term.stop.borrow_mut().push_back(Ok(()));
        term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
        broker.returns.borrow_mut().push_back(Ok(()));
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, "failed");
        assert_eq!(state.reason, "deadline-exceeded");
        assert_eq!(state.retirement, "confirmed");
        assert!(state.credential_returned);
    }

    #[test]
    fn launch_timed_out_stop_failure_is_uncertain() {
        let (_dir, factory, _exec, term, broker) = wired_factory("launch-timeout-stop");
        let req = sample_launch();
        drive_to_start(&term, &broker, &req.run);
        term.start.borrow_mut().push_back(Ok(()));
        term.wait
            .borrow_mut()
            .push_back(Err(FactoryError::DeadlineExceeded));
        term.stop
            .borrow_mut()
            .push_back(Err(FactoryError::msg("stop blew up")));
        term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
        broker.returns.borrow_mut().push_back(Ok(()));
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, "uncertain");
        assert_eq!(state.reason, "deadline-exceeded");
        assert_eq!(state.retirement, "uncertain");
    }

    #[test]
    fn launch_finish_maps_exit_codes() {
        // Nonzero exit fails the run but keeps the receipt confirmed.
        let (_dir, factory, _exec, term, broker) = wired_factory("launch-exit3");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 3, "boom");
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, "failed");
        assert_eq!(state.reason, "execution-failed");
        assert_eq!(state.retirement, "confirmed");
        assert_eq!(state.exit_code, Some(3));
        assert_eq!(state.output, "boom");
        assert!(state.credential_returned);

        // Negative exits record no code (Go's `exit >= 0` guard).
        let (_dir, factory, _exec, term, broker) = wired_factory("launch-exitneg");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, -1, "huh");
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, "failed");
        assert_eq!(state.exit_code, None);
    }

    #[test]
    fn launch_finish_uncertainty_matrix() {
        for (case, reason) in [
            ("stop", "retirement-unconfirmed"),
            ("capture", "credential-capture-unconfirmed"),
            ("return", "credential-return-unconfirmed"),
        ] {
            let (_dir, factory, _exec, term, broker) = wired_factory("launch-finunc");
            let req = sample_launch();
            drive_to_start(&term, &broker, &req.run);
            term.start.borrow_mut().push_back(Ok(()));
            term.wait.borrow_mut().push_back(Ok((0, "out".to_string())));
            term.stop.borrow_mut().push_back(if case == "stop" {
                Err(FactoryError::msg("stop blew up"))
            } else {
                Ok(())
            });
            if case != "stop" {
                term.capture.borrow_mut().push_back(if case == "capture" {
                    Err(FactoryError::msg("capture blew up"))
                } else {
                    Ok(b"{}".to_vec())
                });
            }
            if case == "return" {
                broker
                    .returns
                    .borrow_mut()
                    .push_back(Err(FactoryError::msg("return blew up")));
            }
            broker.close.borrow_mut().push_back(Ok(()));
            let state = factory.launch(&req, deadline()).unwrap();
            assert_eq!(state.phase, "uncertain", "case {case}");
            assert_eq!(state.reason, reason, "case {case}");
            assert_eq!(state.retirement, "uncertain", "case {case}");
        }
    }

    #[test]
    fn launch_on_tombstoned_id_returns_stop() {
        let (_dir, factory, _exec, _term, broker) = wired_factory("launch-tomb");
        let req = sample_launch();
        broker.close.borrow_mut().push_back(Ok(()));
        let stopped = factory
            .stop(
                &FactoryStop {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(stopped.phase, "stopped");
        // No launch scripts: the tombstone refuses without driving.
        let relaunched = factory.launch(&req, deadline()).unwrap();
        assert_eq!(relaunched.phase, "stopped");
        assert_eq!(relaunched.reason, "stop-before-start");
    }

    fn stop_req(run: &FactoryRun) -> FactoryStop {
        FactoryStop {
            project: run.project.clone(),
            id: run.id.clone(),
        }
    }

    #[test]
    fn stop_before_start_writes_exact_tombstone() {
        let (dir, factory, _exec, _term, broker) = wired_factory("stop-tomb");
        let run = sample_run();
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.stop(&stop_req(&run), deadline()).unwrap();
        assert_eq!(state.phase, "stopped");
        assert_eq!(state.retirement, "confirmed");
        assert_eq!(state.reason, "stop-before-start");
        assert_eq!(state.id, run.id);
        assert_eq!(state.unit, factory_unit_name(&run.id));
        let data = receipt_bytes(&dir, &run.project, &run.id);
        let text = String::from_utf8(data).unwrap();
        assert!(
            text.contains("\"deadline\":\"0001-01-01T00:00:00Z\""),
            "{text}"
        );
        assert!(
            text.ends_with("\"retirement\":\"confirmed\",\"reason\":\"stop-before-start\"}"),
            "{text}"
        );
        // A relaunch refuses against the tombstone.
        assert_eq!(broker.close_calls.borrow().len(), 1);
    }

    #[test]
    fn stop_before_start_close_failure_retries() {
        let (_dir, factory, _exec, _term, broker) = wired_factory("stop-retry");
        let run = sample_run();
        broker
            .close
            .borrow_mut()
            .push_back(Err(FactoryError::msg("close blew up")));
        let state = factory.stop(&stop_req(&run), deadline()).unwrap();
        assert_eq!(state.phase, "stopped");
        assert_eq!(state.reason, "broker-close-uncertain");
        // A second stop retries the fence and converges.
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.stop(&stop_req(&run), deadline()).unwrap();
        assert_eq!(state.reason, "stop-before-start");
        assert_eq!(broker.close_calls.borrow().len(), 2);
    }

    #[test]
    fn stop_approved_run_uses_unbound_stop() {
        let (_dir, factory, _exec, term, broker) = wired_factory("stop-approved");
        let req = sample_launch();
        broker
            .acquire
            .borrow_mut()
            .push_back(Err(FactoryError::Busy));
        let launched = factory.launch(&req, deadline()).unwrap();
        assert_eq!(launched.phase, "approved");
        // No lease, no binding: unbound stop, reconcile, close.
        term.stop_unbound.borrow_mut().push_back(Ok(()));
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.stop(&stop_req(&req.run), deadline()).unwrap();
        assert_eq!(state.phase, "stopped");
        assert_eq!(state.reason, "stopped");
        assert_eq!(state.retirement, "confirmed");
        assert_eq!(broker.reconcile_calls.borrow().len(), 0);
    }

    /// Hand-write a running receipt with lease and binding, as if a launch
    /// were mid-flight.
    fn write_running(dir: &std::path::Path, run: &FactoryRun, delivered: bool) {
        let receipt = FactoryReceipt {
            run: run.clone(),
            lease: Some(sample_lease(run)),
            binding: Some(sample_binding(run)),
            generation: 3,
            phase: "running".to_string(),
            started: true,
            delivered,
            ..FactoryReceipt::default()
        };
        receipt.validate().unwrap();
        std::fs::write(
            dir.join(format!("{}-{}.json", run.project, run.id)),
            receipt.encode().as_bytes(),
        )
        .unwrap();
    }

    #[test]
    fn stop_running_run_reconciles_custody() {
        let (dir, factory, _exec, term, broker) = wired_factory("stop-running");
        let run = sample_run();
        write_running(&dir, &run, true);
        term.stop.borrow_mut().push_back(Ok(()));
        term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
        broker.returns.borrow_mut().push_back(Ok(()));
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.stop(&stop_req(&run), deadline()).unwrap();
        assert_eq!(state.phase, "stopped");
        assert_eq!(state.reason, "stopped");
        assert!(state.credential_returned);
    }

    #[test]
    fn stop_converges_when_launch_returned_first() {
        let (dir, factory, _exec, term, broker) = wired_factory("stop-converge");
        let run = sample_run();
        write_running(&dir, &run, true);
        term.stop.borrow_mut().push_back(Ok(()));
        term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
        broker
            .returns
            .borrow_mut()
            .push_back(Err(FactoryError::msg("already returned")));
        // Terminal execution: custody settled without this stop.
        broker.terminal.borrow_mut().push_back(Ok(true));
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.stop(&stop_req(&run), deadline()).unwrap();
        assert_eq!(state.phase, "stopped");
        assert_eq!(state.reason, "stopped");
        // This stop did not return: the flag stays false.
        assert!(!state.credential_returned);
    }

    #[test]
    fn stop_uncertain_when_custody_unsettled() {
        let (dir, factory, _exec, term, broker) = wired_factory("stop-uncertain");
        let run = sample_run();
        write_running(&dir, &run, true);
        term.stop
            .borrow_mut()
            .push_back(Err(FactoryError::msg("stop blew up")));
        term.capture
            .borrow_mut()
            .push_back(Err(FactoryError::msg("capture blew up")));
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.stop(&stop_req(&run), deadline()).unwrap();
        assert_eq!(state.phase, "uncertain");
        assert_eq!(state.reason, "stop-uncertain");
        assert_eq!(state.retirement, "uncertain");
        assert_eq!(
            broker.reconcile_calls.borrow().as_slice(),
            &["lease-1".to_string()]
        );
    }

    #[test]
    fn stop_terminal_receipt_is_final() {
        let (_dir, factory, _exec, term, broker) = wired_factory("stop-final");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        // No scripts left: terminal receipts never touch a seam.
        let state = factory.stop(&stop_req(&req.run), deadline()).unwrap();
        assert_eq!(state.phase, "completed");
        // Invalid addresses reject.
        assert_eq!(
            factory
                .stop(
                    &FactoryStop {
                        project: "bogus".to_string(),
                        id: run_id()
                    },
                    deadline()
                )
                .unwrap_err()
                .message(),
            "invalid factory run address"
        );
    }

    #[test]
    fn inspect_matrix() {
        let (_dir, factory, _exec, _term, broker) = wired_factory("inspect");
        let req = sample_launch();
        // Unknown run.
        assert_eq!(
            factory
                .inspect(
                    &FactoryInspect {
                        project: req.run.project.clone(),
                        id: req.run.id.clone()
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::NotFound
        );
        // Approved run without binding: no liveness probe.
        broker
            .acquire
            .borrow_mut()
            .push_back(Err(FactoryError::Busy));
        factory.launch(&req, deadline()).unwrap();
        let state = factory
            .inspect(
                &FactoryInspect {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "approved");
        assert!(!state.live);
        // Bad address.
        assert_eq!(
            factory
                .inspect(
                    &FactoryInspect {
                        project: "x".to_string(),
                        id: "y".to_string()
                    },
                    deadline()
                )
                .unwrap_err()
                .message(),
            "invalid factory run address"
        );
    }

    #[test]
    fn inspect_running_probes_liveness() {
        let (dir, factory, _exec, term, _broker) = wired_factory("inspect-live");
        let run = sample_run();
        write_running(&dir, &run, true);
        term.live.borrow_mut().push_back(true);
        let state = factory
            .inspect(
                &FactoryInspect {
                    project: run.project.clone(),
                    id: run.id.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "running");
        assert!(state.live);
        assert_eq!(state.container, container_id());
    }

    #[test]
    fn takeover_success_and_refusals() {
        let (_dir, factory, _exec, term, broker) = wired_factory("takeover-ok");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        let dest = format!("/home/alice/factory-takeover/{}", req.run.id);
        term.takeover
            .borrow_mut()
            .push_back(Ok((dest.clone(), false)));
        let result = factory
            .takeover(
                &FactoryTakeover {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    member: "alice".to_string(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(result.destination, dest);
        assert!(!result.reused);
        assert_eq!(
            result.encode(),
            format!(
                "{{\"id\":\"{}\",\"project\":\"{}\",\"member\":\"alice\",\"destination\":\"{dest}\",\"reused\":false}}",
                req.run.id, req.run.project
            )
        );
        let calls = term.takeover_calls.borrow();
        assert_eq!(calls.len(), 1);
        assert_eq!(
            calls[0],
            (
                req.run.project.clone(),
                container_id(),
                "soda-coder".to_string(),
                req.run.preparation.clone(),
                "alice".to_string(),
                req.run.id.clone()
            )
        );

        // A run that never retired refuses.
        let (_dir, factory, _exec, _term, broker) = wired_factory("takeover-busy");
        let req = sample_launch();
        broker
            .acquire
            .borrow_mut()
            .push_back(Err(FactoryError::Busy));
        factory.launch(&req, deadline()).unwrap();
        assert_eq!(
            factory
                .takeover(
                    &FactoryTakeover {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                        member: "alice".to_string(),
                    },
                    deadline()
                )
                .unwrap_err()
                .message(),
            "factory run is not retired for takeover"
        );
        // Unknown runs are not found.
        assert_eq!(
            factory
                .takeover(
                    &FactoryTakeover {
                        project: req.run.project.clone(),
                        id: "b".repeat(32),
                        member: "alice".to_string(),
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::NotFound
        );
        // A wrong destination from the boundary fails validation.
        let (_dir, factory, _exec, term, broker) = wired_factory("takeover-baddest");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        term.takeover
            .borrow_mut()
            .push_back(Ok(("/elsewhere".to_string(), false)));
        assert_eq!(
            factory
                .takeover(
                    &FactoryTakeover {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                        member: "alice".to_string(),
                    },
                    deadline()
                )
                .unwrap_err()
                .message(),
            "takeover destination does not match its identities"
        );
        // Bad member.
        assert_eq!(
            FactoryTakeover {
                project: project_id(),
                id: run_id(),
                member: "root".to_string(),
            }
            .validate()
            .unwrap_err(),
            "invalid takeover member"
        );
    }

    #[test]
    fn output_without_binding_reports_phase_only() {
        let (_dir, factory, _exec, _term, broker) = wired_factory("output-nobind");
        let req = sample_launch();
        broker
            .acquire
            .borrow_mut()
            .push_back(Err(FactoryError::Busy));
        factory.launch(&req, deadline()).unwrap();
        // No output script: unbound runs never reach the boundary.
        let state = factory
            .output(
                &FactoryOutput {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    offset: 0,
                    limit: 100,
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "approved");
        assert!(!state.terminal);
        assert!(state.container.is_empty());
        assert!(state.data.is_empty());
        assert_eq!(state.encode(), format!(
            "{{\"live\":false,\"terminal\":false,\"truncated\":false,\"gap\":false,\"id\":\"{}\",\"project\":\"{}\",\"phase\":\"approved\",\"container\":\"\",\"unit\":\"\",\"invocation\":\"\",\"total\":0,\"offset\":0,\"next\":0,\"data\":\"\",\"reason\":\"broker-busy\"}}",
            req.run.id, req.run.project
        ));
    }

    #[test]
    fn output_slice_reports_binding_and_cursors() {
        let (_dir, factory, _exec, term, broker) = wired_factory("output-slice");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        term.output.borrow_mut().push_back(Ok(OutputSlice {
            data: b"hello".to_vec(),
            total: 100,
            offset: 40,
            truncated: true,
            gap: false,
        }));
        let state = factory
            .output(
                &FactoryOutput {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    offset: 40,
                    limit: 100,
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "completed");
        assert!(state.terminal);
        assert!(!state.live);
        assert_eq!(state.container, container_id());
        assert_eq!(state.unit, factory_unit_name(&req.run.id));
        assert_eq!(state.invocation, "09".repeat(16));
        assert_eq!(state.total, 100);
        assert_eq!(state.offset, 40);
        assert_eq!(state.next, 45);
        assert!(state.truncated);
        assert_eq!(state.data, "aGVsbG8=");
        assert_eq!(state.exit_code, Some(0));
        {
            let calls = term.output_calls.borrow();
            assert_eq!(calls.as_slice(), &[(req.run.project.clone(), 40, 100)]);
        }
        // Unknown runs are not found; stale incarnations refuse.
        assert_eq!(
            factory
                .output(
                    &FactoryOutput {
                        project: req.run.project.clone(),
                        id: "b".repeat(32),
                        offset: 0,
                        limit: 1,
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::NotFound
        );
        term.output.borrow_mut().push_back(Err(FactoryError::Stale));
        assert_eq!(
            factory
                .output(
                    &FactoryOutput {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                        offset: 0,
                        limit: 1,
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::Stale
        );
        // Cursor validation.
        assert_eq!(
            FactoryOutput {
                project: project_id(),
                id: run_id(),
                offset: -1,
                limit: 1,
            }
            .validate()
            .unwrap_err(),
            "invalid output cursor"
        );
        assert_eq!(
            FactoryOutput {
                project: project_id(),
                id: run_id(),
                offset: 0,
                limit: 0,
            }
            .validate()
            .unwrap_err(),
            "invalid output read bound"
        );
    }

    #[test]
    fn export_success_denied_and_bounds() {
        let (_dir, factory, _exec, term, broker) = wired_factory("export-ok");
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        let candidate = "1".repeat(40);
        term.export
            .borrow_mut()
            .push_back(Ok(b"bundle-bytes".to_vec()));
        let state = factory
            .export(
                &FactoryExport {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                    role: "soda-coder".to_string(),
                    preparation: req.run.preparation.clone(),
                    candidate: candidate.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "completed");
        assert_eq!(state.container, container_id());
        assert_eq!(state.candidate, candidate);
        assert_eq!(state.bundle, crate::ssh::b64_encode(b"bundle-bytes"));
        {
            let calls = term.export_calls.borrow();
            assert_eq!(
                calls.as_slice(),
                &[(
                    req.run.project.clone(),
                    container_id(),
                    "soda-coder".to_string(),
                    req.run.preparation.clone(),
                    candidate.clone()
                )]
            );
        }
        // Role mismatch is denied (the daemon maps it to a generic 500).
        assert_eq!(
            factory
                .export(
                    &FactoryExport {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                        role: "soda-reviewer".to_string(),
                        preparation: req.run.preparation.clone(),
                        candidate: candidate.clone(),
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::Denied
        );
        // Empty and oversized bundles exceed bounds.
        term.export.borrow_mut().push_back(Ok(Vec::new()));
        let export_req = FactoryExport {
            project: req.run.project.clone(),
            id: req.run.id.clone(),
            role: "soda-coder".to_string(),
            preparation: req.run.preparation.clone(),
            candidate: candidate.clone(),
        };
        assert_eq!(
            factory.export(&export_req, deadline()).unwrap_err(),
            FactoryError::ExportBounds
        );
        term.export
            .borrow_mut()
            .push_back(Ok(vec![0u8; MAX_FACTORY_EXPORT_BUNDLE + 1]));
        assert_eq!(
            factory.export(&export_req, deadline()).unwrap_err(),
            FactoryError::ExportBounds
        );
        // Terminal verdicts propagate for the route layer to map.
        term.export
            .borrow_mut()
            .push_back(Err(FactoryError::ExportCandidate));
        assert_eq!(
            factory.export(&export_req, deadline()).unwrap_err(),
            FactoryError::ExportCandidate
        );
        // Unsettled runs refuse.
        let (_dir, factory, _exec, _term, broker) = wired_factory("export-busy");
        let req = sample_launch();
        broker
            .acquire
            .borrow_mut()
            .push_back(Err(FactoryError::Busy));
        factory.launch(&req, deadline()).unwrap();
        assert_eq!(
            factory
                .export(
                    &FactoryExport {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                        role: "soda-coder".to_string(),
                        preparation: req.run.preparation.clone(),
                        candidate: "1".repeat(40),
                    },
                    deadline()
                )
                .unwrap_err()
                .message(),
            "factory run is not settled for export"
        );
        // Request validation.
        assert_eq!(
            FactoryExport {
                project: "x".to_string(),
                id: run_id(),
                role: "soda-coder".to_string(),
                preparation: prep_id(),
                candidate: "1".repeat(40),
            }
            .validate()
            .unwrap_err(),
            "invalid factory export address"
        );
        assert_eq!(
            FactoryExport {
                project: project_id(),
                id: run_id(),
                role: "root".to_string(),
                preparation: prep_id(),
                candidate: "1".repeat(40),
            }
            .validate()
            .unwrap_err(),
            "invalid factory export role"
        );
    }

    fn wired_factory_exec(
        tag: &str,
        responses: Vec<Result<Vec<u8>, String>>,
    ) -> (
        std::path::PathBuf,
        TestFactory,
        std::rc::Rc<FakeExec>,
        std::rc::Rc<FakeTerminal>,
        std::rc::Rc<FakeBroker>,
    ) {
        use std::rc::Rc;
        let dir = test_state_dir(tag);
        let exec = Rc::new(FakeExec::new(responses));
        let term = Rc::new(FakeTerminal::new());
        let broker = Rc::new(FakeBroker::new());
        let factory = test_factory(&dir, exec.clone(), term.clone(), broker.clone());
        (dir, factory, exec, term, broker)
    }

    fn preparation_target(project: &str, cid: &str) -> Vec<u8> {
        format!(
            "{{\"id\":{cid:?},\"running\":true,\"project\":{project:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
        )
        .into_bytes()
    }

    #[test]
    fn candidate_script_pins_cutover_shape() {
        // The Go oracle is retired at executor cutover: the embedded
        // script is canonical now, pinned by its security-critical shape
        // (behavior stays covered by inspect_candidate_reports_clean_and_dirty).
        for marker in [
            "set -eu\numask 077",
            "core.hooksPath=/dev/null",
            "GIT_OBJECT_DIRECTORY=\"$src/.git/objects\"",
            "trap '/usr/bin/rm -rf \"$dir\"' EXIT HUP INT TERM",
            "diff-index --cached --quiet --no-ext-diff --no-textconv",
            "diff-files --quiet --no-ext-diff --no-textconv",
            "printf '%s %s\\n' \"$head\" \"$dirty\"",
        ] {
            assert!(
                CANDIDATE_INSPECT_SCRIPT.contains(marker),
                "candidate script lost {marker:?}"
            );
        }
    }

    #[test]
    fn inspect_candidate_reports_clean_and_dirty() {
        for (word, dirty) in [("clean", false), ("dirty", true)] {
            let candidate = "2".repeat(40);
            let (_dir, factory, exec, term, broker) = wired_factory_exec(
                "candidate-ok",
                vec![
                    Ok(preparation_target(&project_id(), &container_id())),
                    Ok(format!("{candidate} {word}\n").into_bytes()),
                ],
            );
            let req = sample_launch();
            script_success(&term, &broker, &req.run, 0, "done");
            factory.launch(&req, deadline()).unwrap();
            let state = factory
                .inspect_candidate(
                    &FactoryCandidateInspect {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                    },
                    deadline(),
                )
                .unwrap();
            assert_eq!(state.id, req.run.id);
            assert_eq!(state.project, req.run.project);
            assert_eq!(state.container, container_id());
            assert_eq!(state.candidate, candidate);
            assert_eq!(state.dirty, dirty);
            assert_eq!(
                state.encode(),
                format!(
                    "{{\"id\":\"{}\",\"project\":\"{}\",\"container\":\"{}\",\"candidate\":\"{candidate}\",\"dirty\":{dirty}}}",
                    req.run.id,
                    req.run.project,
                    container_id()
                )
            );
            // Full argv vectors, byte-exact.
            let calls = exec.calls.borrow();
            assert_eq!(calls.len(), 2);
            assert_eq!(calls[0].1, "/usr/bin/podman");
            assert_eq!(
                calls[0].2,
                vec![
                    "--remote=false".to_string(),
                    "inspect".to_string(),
                    "--format".to_string(),
                    crate::project::PROJECT_INSPECT_FORMAT.to_string(),
                    format!("soda-{}", req.run.project),
                ]
            );
            assert_eq!(calls[1].1, "/usr/bin/podman");
            assert_eq!(
                calls[1].2,
                vec![
                    "exec".to_string(),
                    "--user".to_string(),
                    "soda-coder".to_string(),
                    container_id(),
                    "/usr/bin/env".to_string(),
                    "-i".to_string(),
                    "PATH=/usr/bin:/bin".to_string(),
                    "HOME=/home/soda-coder".to_string(),
                    "LC_ALL=C".to_string(),
                    "TMPDIR=/home/soda-coder/checkouts".to_string(),
                    "GIT_CONFIG_NOSYSTEM=1".to_string(),
                    "GIT_CONFIG_GLOBAL=/dev/null".to_string(),
                    "GIT_NO_REPLACE_OBJECTS=1".to_string(),
                    "GIT_TERMINAL_PROMPT=0".to_string(),
                    "GIT_OPTIONAL_LOCKS=0".to_string(),
                    "/usr/bin/sh".to_string(),
                    "-c".to_string(),
                    CANDIDATE_INSPECT_SCRIPT.to_string(),
                    "soda-candidate".to_string(),
                    format!("/home/soda-coder/checkouts/{}", req.run.preparation),
                ]
            );
            assert!(calls[0].0.is_empty() && calls[1].0.is_empty());
        }
    }

    #[test]
    fn inspect_candidate_incarnation_matrix_is_stale() {
        // A replacement container refuses.
        let (_dir, factory, _exec, term, broker) = wired_factory_exec(
            "candidate-replaced",
            vec![Ok(preparation_target(&project_id(), &"e".repeat(64)))],
        );
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        assert_eq!(
            factory
                .inspect_candidate(
                    &FactoryCandidateInspect {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::Stale
        );
        // Any observation failure refuses stale too.
        let (_dir, factory, _exec, term, broker) =
            wired_factory_exec("candidate-unobs", vec![Err("podman blew up".to_string())]);
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        assert_eq!(
            factory
                .inspect_candidate(
                    &FactoryCandidateInspect {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::Stale
        );
        // Unsettled runs refuse before any exec.
        let (_dir, factory, _exec, _term, broker) = wired_factory("candidate-busy");
        let req = sample_launch();
        broker
            .acquire
            .borrow_mut()
            .push_back(Err(FactoryError::Busy));
        factory.launch(&req, deadline()).unwrap();
        assert_eq!(
            factory
                .inspect_candidate(
                    &FactoryCandidateInspect {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                    },
                    deadline()
                )
                .unwrap_err()
                .message(),
            "candidate inspection requires a settled run"
        );
        // Unknown runs are not found.
        assert_eq!(
            factory
                .inspect_candidate(
                    &FactoryCandidateInspect {
                        project: req.run.project.clone(),
                        id: "b".repeat(32),
                    },
                    deadline()
                )
                .unwrap_err(),
            FactoryError::NotFound
        );
    }

    #[test]
    fn inspect_candidate_output_shapes() {
        // (raw bytes, expected error or dirty flag)
        let commit = "2".repeat(40);
        let cases: Vec<(Vec<u8>, Result<bool, &str>)> = vec![
            (format!("{commit} clean\n").into_bytes(), Ok(false)),
            (format!("  {commit}   dirty  \n").into_bytes(), Ok(true)),
            (
                vec![b'x'; 65],
                Err("candidate checkout inspection unconfirmed"),
            ),
            (
                b"nope\n".to_vec(),
                Err("candidate checkout observation is invalid"),
            ),
            (
                format!("{} bogus\n", "2".repeat(40)).into_bytes(),
                Err("candidate checkout observation is invalid"),
            ),
            (
                format!("{} clean extra\n", "2".repeat(40)).into_bytes(),
                Err("candidate checkout observation is invalid"),
            ),
            (
                b"\n".to_vec(),
                Err("candidate checkout observation is invalid"),
            ),
        ];
        for (raw, expected) in cases {
            let (_dir, factory, _exec, term, broker) = wired_factory_exec(
                "candidate-shape",
                vec![
                    Ok(preparation_target(&project_id(), &container_id())),
                    Ok(raw),
                ],
            );
            let req = sample_launch();
            script_success(&term, &broker, &req.run, 0, "done");
            factory.launch(&req, deadline()).unwrap();
            let out = factory.inspect_candidate(
                &FactoryCandidateInspect {
                    project: req.run.project.clone(),
                    id: req.run.id.clone(),
                },
                deadline(),
            );
            match expected {
                Ok(dirty) => assert_eq!(out.unwrap().dirty, dirty),
                Err(msg) => assert_eq!(out.unwrap_err().message(), msg),
            }
        }
        // Podman failure on the script exec is unconfirmed.
        let (_dir, factory, _exec, term, broker) = wired_factory_exec(
            "candidate-execfail",
            vec![
                Ok(preparation_target(&project_id(), &container_id())),
                Err("podman blew up".to_string()),
            ],
        );
        let req = sample_launch();
        script_success(&term, &broker, &req.run, 0, "done");
        factory.launch(&req, deadline()).unwrap();
        assert_eq!(
            factory
                .inspect_candidate(
                    &FactoryCandidateInspect {
                        project: req.run.project.clone(),
                        id: req.run.id.clone(),
                    },
                    deadline()
                )
                .unwrap_err()
                .message(),
            "candidate checkout inspection unconfirmed"
        );
    }

    fn sample_state(run: &FactoryRun) -> FactoryState {
        FactoryState {
            id: run.id.clone(),
            project: run.project.clone(),
            role: run.role.clone(),
            phase: "completed".to_string(),
            ..FactoryState::default()
        }
    }

    #[test]
    fn factory_facade_confirmations() {
        let run = sample_run();
        let state = sample_state(&run);
        assert!(confirm_factory_launch(&run, &state).is_ok());
        let mut bad = state.clone();
        bad.role = "soda-reviewer".to_string();
        assert_eq!(
            confirm_factory_launch(&run, &bad).unwrap_err(),
            "native factory launch did not return the admitted run"
        );

        let inspect = FactoryInspect {
            project: run.project.clone(),
            id: run.id.clone(),
        };
        assert!(confirm_factory_inspect(&inspect, &state).is_ok());
        assert_eq!(
            confirm_factory_inspect(&inspect, &FactoryState::default()).unwrap_err(),
            "native factory observation does not match its identity"
        );

        let stop = FactoryStop {
            project: run.project.clone(),
            id: run.id.clone(),
        };
        let mut stopped = state.clone();
        stopped.phase = "stopped".to_string();
        stopped.retirement = "confirmed".to_string();
        assert!(confirm_factory_stop(&stop, &stopped).is_ok());
        stopped.retirement = "bogus".to_string();
        assert_eq!(
            confirm_factory_stop(&stop, &stopped).unwrap_err(),
            "native factory stop was not confirmed"
        );

        let output_req = FactoryOutput {
            project: run.project.clone(),
            id: run.id.clone(),
            offset: 0,
            limit: 1,
        };
        let output_state = FactoryOutputState {
            id: run.id.clone(),
            project: run.project.clone(),
            phase: "completed".to_string(),
            ..FactoryOutputState::default()
        };
        assert!(confirm_factory_output(&output_req, &output_state).is_ok());
        let mut bad_output = output_state.clone();
        bad_output.phase = "bogus".to_string();
        assert_eq!(
            confirm_factory_output(&output_req, &bad_output).unwrap_err(),
            "native factory output does not match its identity"
        );

        let export_req = FactoryExport {
            project: run.project.clone(),
            id: run.id.clone(),
            role: run.role.clone(),
            preparation: run.preparation.clone(),
            candidate: "1".repeat(40),
        };
        let export_state = FactoryExportState {
            id: run.id.clone(),
            project: run.project.clone(),
            phase: "completed".to_string(),
            container: container_id(),
            candidate: "1".repeat(40),
            bundle: "eA==".to_string(),
        };
        assert!(confirm_factory_export(&export_req, &export_state).is_ok());
        let mut bad_export = export_state.clone();
        bad_export.bundle = String::new();
        assert_eq!(
            confirm_factory_export(&export_req, &bad_export).unwrap_err(),
            "native factory export does not match its identity"
        );

        let takeover_req = FactoryTakeover {
            project: run.project.clone(),
            id: run.id.clone(),
            member: "alice".to_string(),
        };
        let takeover_result = TakeoverResult {
            id: run.id.clone(),
            project: run.project.clone(),
            member: "alice".to_string(),
            destination: takeover_destination("alice", &run.id),
            reused: false,
        };
        assert!(confirm_factory_takeover(&takeover_req, &takeover_result).is_ok());
        assert_eq!(
            confirm_factory_takeover(
                &takeover_req,
                &TakeoverResult {
                    member: "bob".to_string(),
                    ..takeover_result.clone()
                }
            )
            .unwrap_err(),
            "native factory takeover did not return the admitted destination"
        );

        let cand_req = FactoryCandidateInspect {
            project: run.project.clone(),
            id: run.id.clone(),
        };
        let cand_state = FactoryCandidateState {
            id: run.id.clone(),
            project: run.project.clone(),
            container: container_id(),
            candidate: "1".repeat(40),
            dirty: false,
        };
        assert!(confirm_factory_candidate_inspect(&cand_req, &cand_state).is_ok());
        let mut bad_cand = cand_state.clone();
        bad_cand.container = "short".to_string();
        assert_eq!(
            confirm_factory_candidate_inspect(&cand_req, &bad_cand).unwrap_err(),
            "native candidate observation does not match its identity"
        );
    }

    #[test]
    fn factory_status_mappings() {
        assert_eq!(
            factory_not_found_status(404).as_deref(),
            Some(ERR_RUN_NOT_FOUND)
        );
        assert_eq!(factory_not_found_status(500), None);
        assert_eq!(
            factory_output_status_error(404).as_deref(),
            Some(ERR_RUN_NOT_FOUND)
        );
        assert_eq!(
            factory_output_status_error(409).as_deref(),
            Some(ERR_RUN_STALE)
        );
        assert_eq!(factory_output_status_error(200), None);
        assert_eq!(
            factory_export_status_error(404).as_deref(),
            Some(ERR_RUN_NOT_FOUND)
        );
        assert_eq!(
            factory_export_status_error(409).as_deref(),
            Some(ERR_RUN_STALE)
        );
        assert_eq!(
            factory_export_status_error(422).as_deref(),
            Some("export candidate is not recorded")
        );
        assert_eq!(
            factory_export_status_error(413).as_deref(),
            Some("candidate export exceeds bounds")
        );
        assert_eq!(factory_export_status_error(500), None);
        assert_eq!(
            factory_candidate_status_error(409).as_deref(),
            Some(ERR_RUN_STALE)
        );
        assert_eq!(
            factory_candidate_status_error(404).as_deref(),
            Some(ERR_RUN_NOT_FOUND)
        );
        assert_eq!(factory_candidate_status_error(200), None);
    }

    fn sample_preparation() -> Preparation {
        Preparation {
            id: format!("f{}", "c".repeat(24)),
            project: project_id(),
            role: "soda-coder".to_string(),
            revision: 1,
            requirements: preparation::RequirementAcceptance {
                id: format!("d{}", "c".repeat(24)),
                revision: 1,
                approver: 7,
                source_commit: "e".repeat(40),
                digest: "0".repeat(64),
            },
            approval: preparation::AdminApproval {
                id: format!("d{}", "c".repeat(24)),
                revision: 1,
                approver: 7,
                effects_digest: "0".repeat(64),
            },
            source_commit: "e".repeat(40),
            setup_digest: "0".repeat(64),
            tools: Vec::new(),
            credential: String::new(),
        }
    }

    fn sample_prepare_state(prep: &Preparation) -> PrepareState {
        PrepareState {
            id: prep.id.clone(),
            project: prep.project.clone(),
            role: prep.role.clone(),
            phase: "ready".to_string(),
            container: container_id(),
            source_commit: prep.source_commit.clone(),
            setup_digest: prep.setup_digest.clone(),
            ..PrepareState::default()
        }
    }

    #[test]
    fn prepare_facade_confirmations() {
        let prep = sample_preparation();
        let state = sample_prepare_state(&prep);
        assert!(confirm_prepare(&prep, &state).is_ok());
        assert!(confirm_prepare_candidate(&prep, &state).is_ok());
        let mut bad = state.clone();
        bad.setup_digest = "1".repeat(64);
        assert_eq!(
            confirm_prepare(&prep, &bad).unwrap_err(),
            "native preparation result does not match its identity"
        );
        assert_eq!(
            confirm_prepare_candidate(&prep, &bad).unwrap_err(),
            "native candidate preparation does not match its identity"
        );

        let inspect = PrepareInspect {
            project: prep.project.clone(),
            id: prep.id.clone(),
        };
        assert!(confirm_inspect_preparation(&inspect, &state).is_ok());
        assert_eq!(
            confirm_inspect_preparation(&inspect, &PrepareState::default()).unwrap_err(),
            "native preparation observation does not match its identity"
        );

        let stop = PrepareStop {
            project: prep.project.clone(),
            id: prep.id.clone(),
        };
        let mut stopped = state.clone();
        stopped.stopped = true;
        stopped.retirement = "uncertain".to_string();
        assert!(confirm_stop_preparation(&stop, &stopped).is_ok());
        assert_eq!(
            confirm_stop_preparation(&stop, &state).unwrap_err(),
            "native preparation stop was not confirmed"
        );

        let hold = PrepareHold {
            project: prep.project.clone(),
            hold: true,
            revision: 2,
        };
        assert!(confirm_hold_preparation(
            &hold,
            &HoldState {
                active: true,
                revision: 2
            }
        )
        .is_ok());
        assert_eq!(
            confirm_hold_preparation(&hold, &HoldState::default()).unwrap_err(),
            "native maintenance hold outcome not confirmed"
        );
    }
}
