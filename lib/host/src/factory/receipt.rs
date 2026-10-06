use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use crate::json::{self, Kind, Spec};
use crate::project::Executor;

use super::identity::{BINDING_SPECS, LEASE_SPECS};
use super::run::FACTORY_RUN_SPECS;
use super::{
    factory_unit_name, valid_factory_phase, Binding, Factory, FactoryBroker, FactoryError,
    FactoryRun, FactoryState, FactoryTerminal, Lease, RunLock, FACTORY_STOPPED, FACTORY_UNCERTAIN,
    IDENTITY_FACTORY, MAX_FACTORY_OUTPUT,
};

// ---------- durable run receipts ----------

#[derive(Debug, Clone, Default)]
pub(in crate::factory) struct FactoryReceipt {
    pub(in crate::factory) run: FactoryRun,
    pub(in crate::factory) lease: Option<Lease>,
    pub(in crate::factory) binding: Option<Binding>,
    pub(in crate::factory) exit_code: Option<i64>,
    pub(in crate::factory) generation: i64,
    pub(in crate::factory) phase: String,
    pub(in crate::factory) started: bool,
    pub(in crate::factory) delivered: bool,
    pub(in crate::factory) credential_returned: bool,
    pub(in crate::factory) output: String,
    pub(in crate::factory) retirement: String,
    pub(in crate::factory) reason: String,
}

impl FactoryReceipt {
    pub(in crate::factory) fn validate(&self) -> Result<(), FactoryError> {
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
    pub(in crate::factory) fn encode(&self) -> String {
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

pub(in crate::factory) fn receipt_terminal(phase: &str) -> bool {
    matches!(phase, "completed" | "failed" | "stopped")
}

/// Stop-owned outcomes: a concurrent operator stop recorded the run's
/// fate (cleanly or fenced-uncertain). The launch path never advances
/// past them and never overwrites them; only the stop path itself may
/// repair an uncertain stop.
pub(in crate::factory) fn receipt_stop_owned(phase: &str) -> bool {
    phase == FACTORY_STOPPED || phase == FACTORY_UNCERTAIN
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

pub(in crate::factory) fn receipt_state(receipt: &FactoryReceipt, live: bool) -> FactoryState {
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

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    fn receipt_path(&self, project: &str, run: &str) -> String {
        format!("{}/{project}-{run}.json", self.state_dir)
    }

    fn lock_path(&self, project: &str, run: &str) -> String {
        format!("{}/{project}-{run}.lock", self.state_dir)
    }

    /// `lockRun`: per-run exclusive lock with deadline, mirroring
    /// `filelock.Acquire` (nonblocking flock retried every 25ms).
    pub(in crate::factory) fn lock_run(
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
    pub(in crate::factory) fn load_receipt(
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
    pub(in crate::factory) fn store_receipt(
        &self,
        receipt: &FactoryReceipt,
    ) -> Result<(), FactoryError> {
        receipt.validate()?;
        self.write_receipt_file(receipt)
    }

    /// `storeTombstone`: persist a stop marker for a never-admitted run
    /// without run validation.
    pub(in crate::factory) fn store_tombstone(
        &self,
        receipt: &FactoryReceipt,
    ) -> Result<(), FactoryError> {
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
}
