use std::time::{Duration, Instant};

use serde::{Deserialize, Serialize};

use super::deadline::{parse_deadline, system_nanos_now};
use super::receipt::{receipt_stop_owned, receipt_terminal, FactoryReceipt};
use super::{
    Factory, FactoryBroker, FactoryError, FactoryRun, FactoryTerminal, FactoryVerificationChecks,
    FactoryVerificationChecksState, PrivateState, VerificationManifest,
};
use crate::{domain, json, preparation, project::Executor};

#[derive(Serialize)]
struct HelperRequest<'a> {
    op: &'a str,
    id: &'a str,
    run_id: &'a str,
    invocation_id: &'a str,
    manifest_digest: &'a str,
    check_digest: &'a str,
    source_commit: &'a str,
    deadline: &'a str,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct HelperState {
    id: String,
    run_id: String,
    invocation_id: String,
    manifest_digest: String,
    check_digest: String,
    source_commit: String,
    deadline: String,
    phase: String,
    setup_exit: Option<i64>,
    check_exit: Option<i64>,
    output_digest: String,
    output: String,
    truncated: bool,
    retirement: String,
}

fn manifest_name(run: &FactoryRun) -> String {
    format!("verification-manifest-{}-{}.json", run.project, run.id)
}
fn state_name(run: &FactoryRun) -> String {
    format!("verification-checks-{}-{}.json", run.project, run.id)
}

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    pub(in crate::factory) fn store_verification_manifest(
        &self,
        run: &FactoryRun,
        raw: &[u8],
    ) -> Result<(), FactoryError> {
        if raw.is_empty() {
            return Ok(());
        }
        VerificationManifest::decode(raw)
            .and_then(|m| m.verify_for_run(run, raw))
            .map_err(FactoryError::msg)?;
        let state = PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        if let Some(old) = state
            .read_bytes(&manifest_name(run))
            .map_err(FactoryError::msg)?
        {
            if old != raw {
                return Err(FactoryError::Stale);
            }
            return Ok(());
        }
        state
            .write_bytes(&manifest_name(run), raw)
            .map_err(FactoryError::msg)
    }

    fn load_verification_manifest(
        &self,
        run: &FactoryRun,
    ) -> Result<VerificationManifest, FactoryError> {
        let state = PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        let raw = state
            .read_bytes(&manifest_name(run))
            .map_err(FactoryError::msg)?
            .ok_or(FactoryError::NotFound)?;
        let manifest = VerificationManifest::decode(&raw).map_err(FactoryError::msg)?;
        manifest
            .verify_for_run(run, &raw)
            .map_err(FactoryError::msg)?;
        Ok(manifest)
    }

    pub(in crate::factory) fn saved_verification_checks(
        &self,
        receipt: &FactoryReceipt,
    ) -> Result<Option<FactoryVerificationChecksState>, FactoryError> {
        if receipt.run.verification_manifest_digest.is_empty() {
            return Ok(None);
        }
        let state = PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        let saved: Option<FactoryVerificationChecksState> = state
            .read(&state_name(&receipt.run))
            .map_err(FactoryError::msg)?;
        if let Some(saved) = &saved {
            let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
            let req = FactoryVerificationChecks {
                project: receipt.run.project.clone(),
                run: receipt.run.id.clone(),
                incarnation: binding.invocation_id.clone(),
                manifest_digest: receipt.run.verification_manifest_digest.clone(),
            };
            saved.validate_for(&req).map_err(FactoryError::msg)?;
            let manifest = self.load_verification_manifest(&receipt.run)?;
            if saved.source_commit != manifest.source_commit
                || saved.preparation != manifest.preparation
                || saved.check_digest != manifest.check_digest
            {
                return Err(FactoryError::Stale);
            }
        }
        Ok(saved)
    }

    fn verification_helper(
        &self,
        receipt: &FactoryReceipt,
        manifest: &VerificationManifest,
        op: &str,
        deadline: Instant,
    ) -> Result<FactoryVerificationChecksState, FactoryError> {
        let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
        if !domain::valid_container_id(&binding.project) || binding.id != receipt.run.id {
            return Err(FactoryError::Stale);
        }
        let request = HelperRequest {
            op,
            id: &manifest.preparation,
            run_id: &manifest.run,
            invocation_id: &binding.invocation_id,
            manifest_digest: &receipt.run.verification_manifest_digest,
            check_digest: &manifest.check_digest,
            source_commit: &manifest.source_commit,
            deadline: &manifest.deadline,
        };
        let body = serde_json::to_vec(&request)
            .map_err(|_| FactoryError::msg("verification request encoding failed"))?;
        let raw = self
            .exec
            .run_bounded(
                &body,
                "podman",
                &[
                    "--remote=false",
                    "exec",
                    "--interactive",
                    &binding.project,
                    preparation::FACTORY_HELPER,
                ],
                deadline.min(Instant::now() + Duration::from_secs(20)),
                6 * super::MAX_FACTORY_OUTPUT + 8192,
                4096,
            )
            .map_err(FactoryError::msg)?;
        let helper: HelperState = json::decode_strict_as(&raw)
            .map_err(|_| FactoryError::msg("verification helper receipt is invalid"))?;
        if helper.id != manifest.preparation
            || helper.run_id != manifest.run
            || helper.invocation_id != binding.invocation_id
            || helper.manifest_digest != receipt.run.verification_manifest_digest
            || helper.check_digest != manifest.check_digest
            || helper.source_commit != manifest.source_commit
            || parse_deadline(&helper.deadline) != parse_deadline(&manifest.deadline)
            || helper.output.len() > super::MAX_FACTORY_OUTPUT
        {
            return Err(FactoryError::Stale);
        }
        let _ = helper.truncated; // Display truncation never determines the native output digest.
        let mut state = FactoryVerificationChecksState {
            project: receipt.run.project.clone(),
            run: receipt.run.id.clone(),
            incarnation: binding.invocation_id.clone(),
            manifest_digest: receipt.run.verification_manifest_digest.clone(),
            source_commit: manifest.source_commit.clone(),
            preparation: manifest.preparation.clone(),
            check_digest: manifest.check_digest.clone(),
            ..Default::default()
        };
        match helper.phase.as_str() {
            "pending" => state.phase = "preparing".into(),
            "running" => state.phase = "running".into(),
            "completed"
                if helper.setup_exit == Some(0)
                    && helper.check_exit == Some(0)
                    && helper.retirement == "confirmed" =>
            {
                state.phase = "passed".into();
                state.exit_code = Some(0);
                state.output_digest = helper.output_digest;
                state.retirement = "confirmed".into();
            }
            "completed" | "failed" | "stopped" | "uncertain" => {
                state.phase = if helper.phase == "stopped" {
                    "stopped"
                } else {
                    "failed"
                }
                .into();
                state.exit_code = helper.check_exit.or(helper.setup_exit);
                state.output_digest = helper.output_digest;
                state.retirement = if helper.retirement == "confirmed" {
                    "confirmed"
                } else {
                    "uncertain"
                }
                .into();
                state.reason = "approved-verification-check-did-not-pass".into();
            }
            _ => return Err(FactoryError::msg("unknown verification helper phase")),
        }
        state
            .validate_for(&FactoryVerificationChecks {
                project: state.project.clone(),
                run: state.run.clone(),
                incarnation: state.incarnation.clone(),
                manifest_digest: state.manifest_digest.clone(),
            })
            .map_err(FactoryError::msg)?;
        Ok(state)
    }

    // A first drive records pending before contacting the helper. Every later
    // drive observes: a lost reply or host restart never starts another check.
    pub fn verification_checks(
        &self,
        request: &FactoryVerificationChecks,
        deadline: Instant,
    ) -> Result<FactoryVerificationChecksState, FactoryError> {
        request.validate().map_err(FactoryError::msg)?;
        let _lock = self.lock_run(&request.project, &request.run, deadline)?;
        let (receipt, exists) = self.load_receipt(&request.project, &request.run)?;
        if !exists {
            return Err(FactoryError::NotFound);
        }
        let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
        if binding.invocation_id != request.incarnation
            || receipt.run.verification_manifest_digest != request.manifest_digest
        {
            return Err(FactoryError::Stale);
        }
        let previous = self.saved_verification_checks(&receipt)?;
        if let Some(saved) = &previous {
            if matches!(saved.phase.as_str(), "passed" | "failed" | "stopped") {
                return Ok(saved.clone());
            }
        }
        if receipt.started || receipt_terminal(&receipt.phase) || receipt_stop_owned(&receipt.phase)
        {
            return Err(FactoryError::Stale);
        }
        let manifest = self.load_verification_manifest(&receipt.run)?;
        let store = PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        let op = if previous.is_none() {
            if parse_deadline(&manifest.deadline).unwrap_or(0) <= system_nanos_now() {
                return Err(FactoryError::DeadlineExceeded);
            }
            let pending = FactoryVerificationChecksState {
                project: request.project.clone(),
                run: request.run.clone(),
                incarnation: request.incarnation.clone(),
                manifest_digest: request.manifest_digest.clone(),
                phase: "preparing".into(),
                source_commit: manifest.source_commit.clone(),
                preparation: manifest.preparation.clone(),
                check_digest: manifest.check_digest.clone(),
                ..Default::default()
            };
            store
                .write(&state_name(&receipt.run), &pending)
                .map_err(FactoryError::msg)?;
            "verify"
        } else {
            "verify_inspect"
        };
        let state = self.verification_helper(&receipt, &manifest, op, deadline)?;
        store
            .write(&state_name(&receipt.run), &state)
            .map_err(FactoryError::msg)?;
        Ok(state)
    }

    pub(in crate::factory) fn await_verification_checks(
        &self,
        receipt: &FactoryReceipt,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        if receipt.run.verification_manifest_digest.is_empty() {
            return Ok(());
        }
        let request = FactoryVerificationChecks {
            project: receipt.run.project.clone(),
            run: receipt.run.id.clone(),
            incarnation: receipt
                .binding
                .as_ref()
                .ok_or(FactoryError::Stale)?
                .invocation_id
                .clone(),
            manifest_digest: receipt.run.verification_manifest_digest.clone(),
        };
        loop {
            if Instant::now() >= deadline {
                return Err(FactoryError::DeadlineExceeded);
            }
            let state = self.verification_checks(&request, deadline)?;
            match state.phase.as_str() {
                "passed" => return Ok(()),
                "preparing" | "running" => std::thread::sleep(Duration::from_millis(100)),
                _ => {
                    return Err(FactoryError::msg(
                        "approved verification checks did not pass",
                    ))
                }
            }
        }
    }

    pub(in crate::factory) fn retire_verification_checks(
        &self,
        receipt: &FactoryReceipt,
        deadline: Instant,
    ) -> Result<(), FactoryError> {
        if receipt.run.verification_manifest_digest.is_empty() || receipt.binding.is_none() {
            return Ok(());
        }
        let _lock = self.lock_run(&receipt.run.project, &receipt.run.id, deadline)?;
        // The pending record precedes every helper launch. An absent record
        // therefore needs no native stop; inspect and stop cannot race writes.
        let saved = self.saved_verification_checks(receipt)?;
        if saved.is_none() || saved.is_some_and(|s| s.retirement == "confirmed") {
            return Ok(());
        }
        let manifest = self.load_verification_manifest(&receipt.run)?;
        let state = self.verification_helper(receipt, &manifest, "verify_stop", deadline)?;
        let confirmed = state.retirement == "confirmed";
        PrivateState::open(&self.state_dir)
            .and_then(|s| s.write(&state_name(&receipt.run), &state))
            .map_err(FactoryError::msg)?;
        if confirmed {
            Ok(())
        } else {
            Err(FactoryError::Uncertain)
        }
    }

    pub(in crate::factory) fn verification_retirement_known(
        &self,
        receipt: &FactoryReceipt,
    ) -> bool {
        if receipt.run.verification_manifest_digest.is_empty() {
            return true;
        }
        match self.saved_verification_checks(receipt) {
            Ok(None) => true, // No helper may start before its durable pending record.
            Ok(Some(state)) => state.retirement == "confirmed",
            Err(_) => false,
        }
    }
}
