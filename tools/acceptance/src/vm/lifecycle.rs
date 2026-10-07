use std::sync::{Arc, Mutex};
use std::time::Duration;

use crate::structured::Value as JsonValue;

use super::base::{publish_launch_fixture, verify_launch_base_image};
use super::config::{preflight, VmConfig};
use super::launch::{
    prepare_vm_work_directory, verify_base_image_format, verify_qemu_commands, LaunchFailure,
};
use crate::command::{CommandSpec, StdinSpec};
use crate::error::Error;
use crate::evidence::{Evidence, RedactingWriter};
use crate::process::{self, Phase, Process, SharedWriter};
use crate::qmp::QmpClient;

/// One owned fixture VM.
pub struct Vm<'a> {
    pub(super) config: VmConfig,
    pub(super) process: Option<Arc<Process>>,
    pub(super) boot_args: Option<Vec<String>>,
    pub(super) wait_ssh: bool,
    pub(super) qmp: QmpClient,
    pub(super) outputs: Vec<SharedWriter>,
    pub(super) evidence: &'a Evidence,
    pub(super) attempt: u32,
    pub(super) closed: Option<(String, bool)>,
}

/// Launch a fresh fixture VM: preflight, verified base, fixture
/// record, QEMU checks, work directory, then boot to SSH readiness.
pub fn launch_vm<'a>(
    phase: &Phase,
    mut config: VmConfig,
    evidence: &'a Evidence,
) -> Result<Vm<'a>, Box<LaunchFailure<'a>>> {
    if config.disk_gib == 0 {
        config.disk_gib = 64;
    }
    if let Err(err) = preflight(&config, evidence) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    let base = match verify_launch_base_image(phase, &config) {
        Ok(base) => base,
        Err(err) => return Err(Box::new(LaunchFailure { vm: None, err })),
    };
    if let Err(err) = publish_launch_fixture(&config, evidence, &base) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    if let Err(err) = verify_qemu_commands(phase, evidence, &config.qemu) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    if let Err(err) = verify_base_image_format(phase, evidence, &base.path, config.disk_gib) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    if let Err(err) = prepare_vm_work_directory(phase, evidence, &config, &base.path) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    let mut vm = Vm {
        config,
        process: None,
        boot_args: None,
        wait_ssh: true,
        qmp: QmpClient {
            socket: String::new(),
            #[cfg(test)]
            dial: None,
        },
        outputs: Vec::new(),
        evidence,
        attempt: 0,
        closed: None,
    };
    if let Err(err) = vm.start(phase) {
        let joined = Error::join(vec![Some(err), vm.close().err()]).unwrap();
        return Err(Box::new(LaunchFailure {
            vm: Some(vm),
            err: joined,
        }));
    }
    Ok(vm)
}

impl<'a> Vm<'a> {
    fn open_boot_logs(&mut self) -> Result<(SharedWriter, SharedWriter), Error> {
        let label = format!("boot-{}", self.attempt);
        let out_file = self.evidence.open_file(&format!("{label}.serial"))?;
        let out: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(
            out_file,
            self.evidence.secrets().to_vec(),
        )));
        self.outputs.push(out.clone());
        let err_file = self.evidence.open_file(&format!("{label}.stderr"))?;
        let err_writer: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(
            err_file,
            self.evidence.secrets().to_vec(),
        )));
        self.outputs.push(err_writer.clone());
        Ok((out, err_writer))
    }

    fn wait_qemu_ready(&self, phase: &Phase) -> Result<(), Error> {
        let ready = phase.child(Duration::from_secs(30));
        let process = self.process.as_ref().unwrap();
        loop {
            let mut status = JsonValue::Null;
            // Any successful response is readiness, like the Go owner;
            // the payload content is not consulted.
            if self
                .qmp
                .execute("query-status", "status", None, Some(&mut status), &ready)
                .is_ok()
            {
                return Ok(());
            }
            if process.is_done() {
                return Err(Error::join(vec![
                    Some(Error::msg("QEMU exited before readiness")),
                    process.wait(phase).err(),
                ])
                .unwrap());
            }
            ready.check()?;
            std::thread::sleep(Duration::from_millis(200));
        }
    }

    fn wait_guest_ssh(&self, phase: &Phase) -> Result<(), Error> {
        if !self.wait_ssh {
            return Ok(());
        }
        let ssh_phase = phase.child(Duration::from_secs(10 * 60));
        let watcher_phase = ssh_phase.clone();
        let watcher_process = self.process.as_ref().unwrap().clone();
        std::thread::spawn(move || loop {
            if watcher_process.is_done() {
                watcher_phase.cancel();
                break;
            }
            if watcher_phase.is_cancelled() || watcher_phase.expired() {
                break;
            }
            std::thread::sleep(Duration::from_millis(50));
        });
        let process = self.process.as_ref().unwrap();
        match self.config.ssh.remote().wait_ready(&ssh_phase) {
            Ok(()) => Ok(()),
            Err(err) => {
                if process.is_done() {
                    return Err(Error::join(vec![
                        Some(Error::msg("QEMU exited during SSH readiness")),
                        process.wait(phase).err(),
                        Some(err),
                    ])
                    .unwrap());
                }
                Err(err)
            }
        }
    }

    fn start(&mut self, phase: &Phase) -> Result<(), Error> {
        self.attempt += 1;
        let (out, err_writer) = self.open_boot_logs()?;
        let args = self.boot_args.clone().unwrap_or_else(|| self.config.args());
        let spec = CommandSpec {
            name: self.config.qemu.clone(),
            args,
            dir: None,
            stdin: StdinSpec::Null,
            env: Vec::new(),
        };
        self.process = Some(process::start_process(phase, &spec, out, err_writer)?);
        self.qmp = QmpClient {
            socket: format!("{}/qmp.sock", self.config.work),
            #[cfg(test)]
            dial: None,
        };
        self.wait_qemu_ready(phase)?;
        self.wait_guest_ssh(phase)
    }

    /// Restart reusing exactly this instance's disk and NVRAM. No data
    /// is removed.
    pub fn restart(&mut self, phase: &Phase) -> Result<(), Error> {
        self.power_down(phase)?;
        self.start(phase)
    }

    fn power_down(&mut self, phase: &Phase) -> Result<(), Error> {
        let Some(process) = self.process.clone() else {
            return Ok(());
        };
        let shutdown = phase.child(Duration::from_secs(2 * 60));
        let mut err = self
            .qmp
            .execute("system_powerdown", "powerdown", None, None, &shutdown)
            .err();
        // A failed shutdown request leaves a live guest for close_once to
        // stop; waiting out the graceful allowance cannot help that failure.
        if err.is_some() && !process.is_done() {
            return Err(err.expect("QMP shutdown failed"));
        }
        err = Error::join(vec![err, process.wait(&shutdown).err()]);
        if !process.is_done() {
            return err.map(Err).unwrap_or(Ok(()));
        }
        err = Error::join(vec![err, process.join_pumps().map(Error::msg)]);
        self.process = None;
        match std::fs::remove_file(&self.qmp.socket) {
            Ok(()) => {}
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => {}
            Err(remove_err) => err = Error::join(vec![err, Some(Error::from(remove_err))]),
        }
        err = Error::join(vec![err, self.close_outputs()]);
        err.map(Err).unwrap_or(Ok(()))
    }

    fn close_outputs(&mut self) -> Option<Error> {
        let mut err = None;
        for writer in self.outputs.drain(..) {
            let mut slot = writer.lock().unwrap_or_else(|e| e.into_inner());
            err = Error::join(vec![err, slot.close().err()]);
        }
        err
    }

    /// Wait for guest exit or phase end. The Go owner panics on a
    /// closed VM; this port reports the guest as unexpectedly gone, a
    /// path the driver never takes.
    pub fn wait(&self, phase: &Phase) -> Result<(), Error> {
        let Some(process) = self.process.as_ref() else {
            return Err(Error::msg("guest exited unexpectedly"));
        };
        loop {
            phase.check()?;
            if process.is_done() {
                return Err(Error::join(vec![
                    Some(Error::msg("guest exited unexpectedly")),
                    process.wait(phase).err(),
                ])
                .unwrap());
            }
            std::thread::sleep(Duration::from_millis(50));
        }
    }

    fn close_once(&mut self) -> Result<(), Error> {
        let shutdown = Phase::background().child(Duration::from_secs(2 * 60));
        let mut err = self.power_down(&shutdown).err();
        if let Some(process) = self.process.clone() {
            err = Error::join(vec![err, process.stop().err()]);
            if process.is_done() {
                err = Error::join(vec![err, process.join_pumps().map(Error::msg)]);
                self.process = None;
            } else {
                // A kernel-stuck child still owns the capture writers. Do not
                // close them concurrently or describe their retention as complete.
                let outputs = std::mem::take(&mut self.outputs);
                std::thread::spawn(move || {
                    let _ = process.wait(&Phase::background());
                    let _ = process.join_pumps();
                    for writer in outputs {
                        let mut slot = writer.lock().unwrap_or_else(|e| e.into_inner());
                        let _ = slot.close();
                    }
                });
                err = Error::join(vec![
                    err,
                    Some(Error::msg("VM capture still owned by incomplete cleanup")),
                ]);
                return err.map(Err).unwrap_or(Ok(()));
            }
        }
        err = Error::join(vec![err, self.close_outputs()]);
        err.map(Err).unwrap_or(Ok(()))
    }

    /// Keep disks/provisioning for inspection, even on failure. No
    /// enrollment is inferred or revoked. The first outcome replays on
    /// later calls, like the Go owner's `sync.Once`.
    pub fn close(&mut self) -> Result<(), Error> {
        if let Some((message, cancelled)) = &self.closed {
            if message.is_empty() && !cancelled {
                return Ok(());
            }
            let cause = if *cancelled {
                Error::Cancelled
            } else {
                Error::msg(String::new())
            };
            return Err(Error::redacted(message.clone(), cause));
        }
        let result = self.close_once();
        let (message, cancelled) = match &result {
            Ok(()) => (String::new(), false),
            Err(err) => (err.to_string(), err.is_cancelled()),
        };
        self.closed = Some((message, cancelled));
        result
    }
}
