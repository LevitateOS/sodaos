use crate::domain;
use crate::json;
use crate::preparation;
use serde::de::{self, MapAccess, SeqAccess, Visitor};
use serde::Deserialize;
use std::fmt;

use super::deadline::{deadline_is_zero, parse_deadline, NANOS_PER_SEC};

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
/// Reassess/repair prompt exception: a reassess or repair prompt carries
/// the exact complete inventory inline plus bounded instructions. Every
/// other phase keeps `MAX_FACTORY_PROMPT`, and every phase result keeps
/// `MAX_FACTORY_OUTPUT`. Prompts use stdin/file delivery, never argv.
pub const MAX_REASSESS_PROMPT: usize = 320 * 1024;
/// `MaxExecutionManifest`: one staged execution manifest (D05).
pub const MAX_EXECUTION_MANIFEST: usize = 64 * 1024;
/// Admitted execution phases (`project.Execution*`). The empty phase is a
/// pre-phase record admitted during migration; admission always records an
/// explicit phase.
pub const EXECUTION_DECOMPOSE: &str = "decompose";
pub const EXECUTION_INVESTIGATE: &str = "investigate";
pub const EXECUTION_IMPLEMENT: &str = "implement";
pub const EXECUTION_REVIEW: &str = "review";
pub const EXECUTION_CORRECT: &str = "correct";
pub const EXECUTION_REASSESS: &str = "reassess";
/// D15/D16 stall and delivery repair investigation (coder, single-agent,
/// no publication, no candidate).
pub const EXECUTION_REPAIR: &str = "repair";
/// D16 integrated delivery verification (reviewer, read-only, no
/// publication or children, no candidate).
pub const EXECUTION_VERIFY: &str = "verify";
pub const EXECUTION_MAINTAIN_LIBRARY: &str = "maintain_library";

/// `ValidExecutionPhase`: admitted execution phases. Admission always
/// records an explicit phase; the empty phase is invalid. Phase is never
/// inferred from role or PR state.
pub fn valid_execution_phase(phase: &str) -> bool {
    matches!(
        phase,
        "decompose"
            | "investigate"
            | "implement"
            | "review"
            | "correct"
            | "reassess"
            | "repair"
            | "verify"
            | "maintain_library"
    )
}

/// `ExecutionManifestRequired`: implement runs are multi-child planned
/// execution and correct runs are singleton re-execution, both with a staged
/// manifest; every other phase stays single-agent.
pub fn execution_manifest_required(phase: &str) -> bool {
    phase == EXECUTION_IMPLEMENT || phase == EXECUTION_CORRECT
}

/// `MaxPromptForPhase`: the admitted prompt bound for one phase: the
/// 320 KiB inventory exception for reassess and repair, the standard
/// bound otherwise.
pub fn max_prompt_for_phase(phase: &str) -> usize {
    if phase == EXECUTION_REASSESS || phase == EXECUTION_REPAIR {
        MAX_REASSESS_PROMPT
    } else {
        MAX_FACTORY_PROMPT
    }
}

/// Task child-concurrency bounds for one admitted implement run. The
/// attempt policy mirrors this range; the manifest carries the admitted
/// snapshot. The parent run is separate; nested child spawning is refused.
pub const MIN_TASK_CHILDREN: i64 = 1;
pub const MAX_TASK_CHILDREN: i64 = 4;
pub const DEFAULT_TASK_CHILDREN: i64 = 3;
pub const MAX_FACTORY_OUTPUT: usize = 64 * 1024;
pub const MAX_FACTORY_OUTPUT_READ: i64 = 24 * 1024 - 256;
pub const MAX_FACTORY_OUTPUT_WINDOW: i64 = 256 * 1024;
pub const MAX_FACTORY_OUTPUT_OFFSET: i64 = 256 << 20;
pub const MAX_FACTORY_EXPORT_BUNDLE: usize = 4 << 20;

/// Broker kind for supervised factory executions.
pub const IDENTITY_FACTORY: &str = "factory";
/// Terminal broker execution state meaning custody settled elsewhere.
pub const EXECUTION_TERMINAL: &str = "terminal";

/// Client-side unknown-run error (`host.ErrRunNotFound`).
pub const ERR_RUN_NOT_FOUND: &str = "factory run not found";
/// Client-side incarnation-changed error (`host.ErrRunStale`).
pub const ERR_RUN_STALE: &str = "factory run incarnation changed";

/// Retry bound for post-expiry retirement, mirroring
/// `factoryCleanupTimeout`: stop, credential custody and receipt work run
/// under a fresh 60s deadline detached from the expired operation parent.
pub(in crate::factory) const FACTORY_CLEANUP_SECS: u64 = 60;
/// Bound for one liveness probe, mirroring the 10s live contexts.
pub(in crate::factory) const FACTORY_LIVE_SECS: u64 = 10;
/// Supervised run bound: deadlines must fall within three hours.
pub(in crate::factory) const FACTORY_DEADLINE_BOUND_NANOS: i128 = 3 * 3600 * NANOS_PER_SEC;

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

    pub(in crate::factory) fn msg(text: impl Into<String>) -> Self {
        FactoryError::Msg(text.into())
    }
}

impl std::fmt::Display for FactoryError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.message())
    }
}

impl std::error::Error for FactoryError {}

// ---------- run, lease and binding records ----------

/// `project.FactoryRun`: one supervised CLI execution identity. The
/// deadline keeps its verbatim Go `time.Time` JSON text so receipt bytes
/// round-trip exactly. `execution_phase` names the admitted factory phase;
/// `manifest_digest` binds the staged execution manifest for implement and
/// correct runs and stays empty otherwise.
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
    pub execution_phase: String,
    pub manifest_digest: String,
}

impl FactoryRun {
    fn validate_run_identity(&self) -> Result<(), String> {
        if !valid_factory_run_id(&self.id)
            || !domain::valid_id(&self.project)
            || !preparation::valid_factory_role(&self.role)
        {
            return Err("invalid factory run identity".to_string());
        }
        if !preparation::valid_preparation_id(&self.preparation) {
            return Err("invalid run preparation reference".to_string());
        }
        Ok(())
    }

    fn validate_run_harness_model(&self) -> Result<(), String> {
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
        Ok(())
    }

    fn validate_run_assignment_deadline(&self) -> Result<(), String> {
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

    fn validate_run_phase(&self) -> Result<(), String> {
        if !valid_execution_phase(&self.execution_phase) {
            return Err("invalid run execution phase".to_string());
        }
        if self.execution_phase == EXECUTION_REASSESS && self.role != preparation::ROLE_CODER {
            return Err("reassess run requires the coder role".to_string());
        }
        if self.execution_phase == EXECUTION_REPAIR && self.role != preparation::ROLE_CODER {
            return Err("repair run requires the coder role".to_string());
        }
        if self.execution_phase == EXECUTION_VERIFY && self.role != preparation::ROLE_REVIEWER {
            return Err("verify run requires the reviewer role".to_string());
        }
        if self.execution_phase == EXECUTION_MAINTAIN_LIBRARY
            && self.role != preparation::ROLE_REVIEWER
        {
            return Err("library maintenance run requires the reviewer role".to_string());
        }
        Ok(())
    }

    fn validate_run_manifest(&self) -> Result<(), String> {
        if execution_manifest_required(&self.execution_phase) {
            if !preparation::valid_digest(&self.manifest_digest) {
                if self.execution_phase == EXECUTION_CORRECT {
                    return Err("correct run lacks its manifest digest".to_string());
                }
                return Err("implement run lacks its manifest digest".to_string());
            }
        } else if !self.manifest_digest.is_empty() {
            return Err("non-implement run carries a manifest digest".to_string());
        }
        Ok(())
    }

    pub fn validate(&self) -> Result<(), String> {
        self.validate_run_identity()?;
        self.validate_run_harness_model()?;
        self.validate_run_assignment_deadline()?;
        self.validate_run_phase()?;
        self.validate_run_manifest()?;
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
        if !self.execution_phase.is_empty() {
            out.push_str(",\"execution_phase\":");
            out.push_str(&json::quote(&self.execution_phase));
        }
        if !self.manifest_digest.is_empty() {
            out.push_str(",\"manifest_digest\":");
            out.push_str(&json::quote(&self.manifest_digest));
        }
        out.push('}');
    }
}

macro_rules! typed_run_fields {
    ($ty:ty, $expect:literal, {$($field:ident => $name:literal : $value:ty),+ $(,)?}) => {
        impl<'de> Deserialize<'de> for $ty {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error> where D: serde::Deserializer<'de> {
                struct ObjectVisitor;
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str($expect) }
                    fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error> where A: MapAccess<'de> {
                        let mut out = <$ty>::default();
                        while let Some(key) = map.next_key::<String>()? {
                            $(if key.eq_ignore_ascii_case($name) { if let Some(v) = map.next_value::<Option<$value>>()? { out.$field = v.into(); } continue; })+
                            return Err(de::Error::unknown_field(&key, &[$($name),+]));
                        }
                        Ok(out)
                    }
                }
                deserializer.deserialize_map(ObjectVisitor)
            }
        }
    };
}

typed_run_fields!(FactoryRun, "a factory run object", {
    deadline => "deadline": String,
    actor => "actor": json::SignedInteger,
    id => "id": String,
    project => "project": String,
    role => "role": String,
    preparation => "preparation": String,
    harness => "harness": String,
    harness_vers => "harness_version": String,
    model => "model": String,
    assignment => "assignment": String,
    source_commit => "source_commit": String,
    connection => "connection": String,
    execution_phase => "execution_phase": String,
    manifest_digest => "manifest_digest": String,
});

/// `project.FactoryLaunch`: the host run request. `execution_manifest`
/// carries the immutable staged plan for implement and correct runs and stays
/// absent for every other phase; the run's manifest digest must match its
/// exact bytes.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryLaunch {
    pub run: FactoryRun,
    pub prompt: Vec<u8>,
    pub harness_sha256: String,
    pub execution_manifest: Vec<u8>,
    pub mcp: Option<FactoryMCPBinding>,
}

/// A run-scoped transport credential. It is staged privately before launch and
/// never included in the user prompt, saved run receipt or diagnostic output.
#[derive(Clone, Deserialize, serde::Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct FactoryMCPBinding {
    pub capability: String,
    pub secret: String,
}

impl fmt::Debug for FactoryMCPBinding {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("FactoryMCPBinding")
            .field("capability", &self.capability)
            .field("secret", &"[redacted]")
            .finish()
    }
}

impl FactoryMCPBinding {
    pub fn validate(&self) -> Result<(), String> {
        if !valid_factory_run_id(&self.capability) || !preparation::valid_digest(&self.secret) {
            return Err("invalid Factory MCP binding".to_string());
        }
        Ok(())
    }
}

struct PromptBytes(Vec<u8>);

impl<'de> Deserialize<'de> for PromptBytes {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct BytesVisitor;
        impl<'de> Visitor<'de> for BytesVisitor {
            type Value = PromptBytes;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("base64 text or a byte array")
            }
            fn visit_str<E>(self, value: &str) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                crate::ssh::b64_decode_go(value.as_bytes())
                    .map(PromptBytes)
                    .map_err(|_| E::custom("invalid base64 bytes"))
            }
            fn visit_string<E>(self, value: String) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                self.visit_str(&value)
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut bytes = Vec::new();
                while let Some(value) = seq.next_element::<Option<u64>>()? {
                    let byte = value.unwrap_or(0);
                    if byte > u8::MAX as u64 {
                        return Err(de::Error::custom("byte out of range"));
                    }
                    bytes.push(byte as u8);
                }
                Ok(PromptBytes(bytes))
            }
        }
        deserializer.deserialize_any(BytesVisitor)
    }
}

impl<'de> Deserialize<'de> for FactoryLaunch {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct LaunchVisitor;
        impl<'de> Visitor<'de> for LaunchVisitor {
            type Value = FactoryLaunch;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a factory launch object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = FactoryLaunch::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("run") {
                        if let Some(v) = map.next_value::<Option<FactoryRun>>()? {
                            out.run = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("prompt") {
                        if let Some(v) = map.next_value::<Option<PromptBytes>>()? {
                            out.prompt = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("harness_sha256") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.harness_sha256 = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("execution_manifest") {
                        if let Some(v) = map.next_value::<Option<PromptBytes>>()? {
                            out.execution_manifest = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("mcp") {
                        out.mcp = map.next_value::<Option<FactoryMCPBinding>>()?;
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "run",
                            "prompt",
                            "harness_sha256",
                            "execution_manifest",
                            "mcp",
                        ],
                    ));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(LaunchVisitor)
    }
}

impl FactoryLaunch {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

impl FactoryLaunch {
    pub fn validate(&self) -> Result<(), String> {
        self.run.validate()?;
        if let Some(binding) = &self.mcp {
            binding.validate()?;
        }
        if self.prompt.is_empty()
            || self.prompt.len() > max_prompt_for_phase(&self.run.execution_phase)
        {
            return Err("invalid run prompt size".to_string());
        }
        if crate::sha256::hex_lower(&crate::sha256::digest(&self.prompt)) != self.run.assignment {
            return Err("prompt bytes do not match their digest".to_string());
        }
        if !preparation::valid_digest(&self.harness_sha256) {
            return Err("invalid harness pin".to_string());
        }
        if self.execution_manifest.len() > MAX_EXECUTION_MANIFEST {
            return Err("execution manifest exceeds its bound".to_string());
        }
        if execution_manifest_required(&self.run.execution_phase) {
            if self.execution_manifest.is_empty() {
                if self.run.execution_phase == EXECUTION_CORRECT {
                    return Err("correct run lacks its execution manifest".to_string());
                }
                return Err("implement run lacks its execution manifest".to_string());
            }
            if crate::sha256::hex_lower(&crate::sha256::digest(&self.execution_manifest))
                != self.run.manifest_digest
            {
                return Err("execution manifest does not match its digest".to_string());
            }
            // FW14: the staged bytes must decode as the immutable
            // execution envelope and bind this launching run. Raw plan
            // bytes staged by older dispatch code refuse here.
            let manifest = super::manifest::ExecutionManifest::decode(&self.execution_manifest)?;
            manifest.verify_for_run(&self.run, &self.execution_manifest)?;
        } else if !self.execution_manifest.is_empty() {
            return Err("non-implement run stages an execution manifest".to_string());
        }
        Ok(())
    }
}
