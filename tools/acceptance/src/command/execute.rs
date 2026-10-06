use std::sync::{Arc, Mutex};

use crate::error::Error;
use crate::evidence::{Evidence, RedactingWriter};
use crate::process::{self, Phase, SharedWriter};

use super::{CommandResult, CommandSpec};

fn valid_command_label(label: &str) -> bool {
    !label.is_empty()
        && label
            .chars()
            .all(|c| matches!(c, 'a'..='z' | 'A'..='Z' | '0'..='9' | '_' | '-'))
}

fn redact_opt(evidence: &Evidence, err: Option<Error>) -> Option<Error> {
    err.map(|e| evidence.redact_error(e))
}

fn close_shared(writer: &SharedWriter) -> Option<Error> {
    writer
        .lock()
        .unwrap_or_else(|e| e.into_inner())
        .close()
        .err()
}

/// Execute one labeled command with redacted capture. The result carries
/// execution failure; the return value carries evidence retention failure.
/// Both must be checked, including expected denials.
pub fn execute(
    phase: &Phase,
    evidence: &Evidence,
    label: &str,
    spec: &CommandSpec,
) -> (CommandResult, Option<Error>) {
    if !valid_command_label(label) {
        return (
            CommandResult::default(),
            Some(Error::msg("invalid command label")),
        );
    }
    let out_file = match evidence.open_file(&format!("{label}.stdout")) {
        Ok(file) => file,
        Err(e) => return (CommandResult::default(), Some(e)),
    };
    let err_file = match evidence.open_file(&format!("{label}.stderr")) {
        Ok(file) => file,
        Err(e) => {
            drop(out_file);
            return (CommandResult::default(), Some(e));
        }
    };
    let out: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(
        out_file,
        evidence.secrets().to_vec(),
    )));
    let err_writer: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(
        err_file,
        evidence.secrets().to_vec(),
    )));
    let process = match process::start_process(phase, spec, out.clone(), err_writer.clone()) {
        Ok(process) => process,
        Err(run_err) => {
            let write_err = Error::join(vec![close_shared(&out), close_shared(&err_writer)]);
            let result = CommandResult {
                err: Some(evidence.redact_error(run_err)),
                ..Default::default()
            };
            return (result, redact_opt(evidence, write_err));
        }
    };
    let mut run_err = process.wait(phase).err();
    if phase.check().is_err() {
        run_err = Error::join(vec![run_err, phase.check().err(), process.stop().err()]);
    }
    if !process.is_done() {
        let out_close = out.clone();
        let err_close = err_writer.clone();
        std::thread::spawn(move || {
            let _ = process.wait(&Phase::background());
            let _ = close_shared(&out_close);
            let _ = close_shared(&err_close);
        });
        let result = CommandResult {
            started: true,
            err: redact_opt(evidence, run_err),
            ..Default::default()
        };
        return (
            result,
            Some(Error::msg(
                "evidence still owned by incomplete process cleanup",
            )),
        );
    }
    let pump_err = process.join_pumps();
    let close_err = Error::join(vec![close_shared(&out), close_shared(&err_writer)]);
    let outcome = process.outcome();
    let exit_code = outcome.as_ref().and_then(|o| o.exit_code);
    let (run_err, write_err) = match (&outcome, exit_code) {
        (Some(o), Some(0)) if pump_err.is_some() && phase.check().is_ok() => {
            // Copy failures after exit zero belong to retention, not a
            // native denial. Group cleanup failures remain execution failures.
            (
                o.cleanup_message.clone().map(Error::msg),
                Error::join(vec![close_err, pump_err.map(Error::msg)]),
            )
        }
        _ => (run_err, close_err),
    };
    let stdout = out
        .lock()
        .unwrap_or_else(|e| e.into_inner())
        .buffer()
        .to_vec();
    let stderr = err_writer
        .lock()
        .unwrap_or_else(|e| e.into_inner())
        .buffer()
        .to_vec();
    let mut result = CommandResult {
        stdout,
        stderr,
        err: None,
        started: true,
        exit_code,
    };
    if let Some(e) = run_err {
        result.err = Some(evidence.redact_error(Error::wrap("command execution", e)));
    }
    (result, redact_opt(evidence, write_err))
}
