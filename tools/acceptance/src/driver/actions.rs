use crate::command::{self, CommandSpec, Remote, StdinSpec};
use crate::error::Error;
use crate::evidence::Evidence;
use crate::jsonio;
use crate::process::Phase;
use crate::remote;
use crate::report::Observation;
use crate::vm;

use super::RunOptions;

/// Prebuilt target payload: the `soda-acceptance-remote` binary shipped
/// beside the driver. The Go owner embeds Python source instead, so a
/// missing sibling is a new deployment-shape failure with no Go text.
fn payload_bytes() -> Result<Vec<u8>, Error> {
    let driver = std::env::current_exe().map_err(Error::from)?;
    let payload = driver
        .parent()
        .unwrap_or(std::path::Path::new("."))
        .join("soda-acceptance-remote");
    std::fs::read(&payload).map_err(Error::from)
}

#[allow(clippy::too_many_arguments)]
fn execute_exec_or_native(
    phase: &Phase,
    evidence: &Evidence,
    action: &str,
    args: &[String],
    remote_file: &str,
    request: &str,
    revision: &str,
    arch: &str,
    target: &str,
    remote: &Remote,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    let spec = if action == "exec" {
        observation.invocation = Some(args.to_vec());
        let mut spec = CommandSpec {
            name: args[0].clone(),
            args: args[1..].to_vec(),
            dir: None,
            stdin: StdinSpec::Inherit,
            env: Vec::new(),
        };
        if !remote_file.is_empty() {
            spec = match remote.command(args, StdinSpec::Inherit) {
                Ok(spec) => spec,
                Err(err) => return (Some(err), None),
            };
        }
        spec
    } else {
        let payload = match payload_bytes() {
            Ok(payload) => payload,
            Err(err) => return (Some(err), None),
        };
        let spec = match remote::native_phase(remote, request, revision, arch, target, &payload) {
            Ok(spec) => spec,
            Err(err) => return (Some(err), None),
        };
        let raw = match jsonio::read_json_file(request) {
            Ok((value, _)) => value,
            Err(err) => return (Some(err), None),
        };
        let request = match remote::decode_remote_request(&raw) {
            Ok(request) => request,
            Err(err) => return (Some(err), None),
        };
        observation.invocation = Some(vec![
            "embedded native executor".to_string(),
            request.phase.clone(),
            request.revision,
            request.architecture,
            request.target,
            request.work,
        ]);
        observation.action = format!("native-{}", request.phase);
        spec
    };
    let (result, evidence_err) = command::execute(phase, evidence, "check", &spec);
    observation.exit_code = result.exit_code.map(i64::from);
    if result.started {
        observation.execution = "completed".to_string();
        if result.err.is_some() {
            observation.execution = "failed".to_string();
        }
    }
    (result.err, evidence_err)
}

fn execute_probe_ssh(
    phase: &Phase,
    evidence: &Evidence,
    remote: &Remote,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    observation.topology = format!(
        "direct client TCP to {}:{}; no proxy, forwarding inference or Git authentication",
        remote.host, remote.port
    );
    observation.invocation = Some(vec![
        "SSH key exchange only".to_string(),
        remote.host.clone(),
        remote.port.to_string(),
        remote.known_hosts.clone(),
    ]);
    observation.execution = "failed".to_string();
    observation.cleanup = "connection closed".to_string();
    let fingerprint = match crate::probe::probe_ssh_key(phase, remote) {
        Ok(fingerprint) => fingerprint,
        Err(err) => return (Some(err), None),
    };
    observation.execution = "completed".to_string();
    let evidence_err = evidence
        .write("ssh-key.txt", format!("{fingerprint}\nPinned SSH endpoint observed, not Git authentication or project routing proof.\n").as_bytes())
        .err();
    (None, evidence_err)
}

fn run_vm_phase(
    phase: &Phase,
    evidence: &Evidence,
    vm: &mut vm::Vm,
    work: &str,
    restart: bool,
    hold: bool,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    observation.execution = "completed".to_string();
    let mut evidence_err = evidence
        .write("boot-ready.txt", b"Fresh guest reached pinned SSH readiness. This is fixture evidence, not a product check.\n")
        .err();
    let mut operation_err = None;
    if restart {
        operation_err = vm.restart(phase).err();
        if operation_err.is_none() {
            evidence_err = Error::join(vec![
                evidence_err,
                evidence
                    .write(
                        "restart-ready.txt",
                        b"Same owned disk/NVRAM reached pinned SSH after restart. No project persistence assertion was performed.\n",
                    )
                    .err(),
            ]);
        }
    }
    if operation_err.is_none() && hold {
        println!("Fresh guest ready. Invoke owned checks separately. Interrupt shuts down and retains disks; cancellation is not a pass.");
        operation_err = vm.wait(phase).err();
    }
    if operation_err.is_some() {
        observation.execution = "failed".to_string();
    }
    let cleanup_err = vm.close().err();
    observation.cleanup = format!("completed; private disk/NVRAM retained at {work}");
    if cleanup_err.is_some() {
        observation.cleanup = "failed; private work retained".to_string();
    }
    (Error::join(vec![operation_err, cleanup_err]), evidence_err)
}

fn execute_vm(
    phase: &Phase,
    evidence: &Evidence,
    vm_config: vm::VmConfig,
    restart: bool,
    hold: bool,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    observation.topology =
        "matching-native KVM; loopback management SSH only, no routed-client claim".to_string();
    let mut invocation = vec![
        "fresh VM".to_string(),
        vm_config.name.clone(),
        vm_config.work.clone(),
        vm_config.base_receipt.clone(),
        "private single fw_cfg Ignition (not retained)".to_string(),
    ];
    if restart {
        invocation.push("--restart".to_string());
    }
    if hold {
        invocation.push("--hold".to_string());
    }
    observation.invocation = Some(invocation);
    observation.cleanup = "not-reached or failed launch; work retained".to_string();
    let work = vm_config.work.clone();
    let mut guest = match vm::launch_vm(phase, vm_config, evidence) {
        Ok(guest) => guest,
        Err(failure) => {
            let failure = *failure;
            if let Some(mut started) = failure.vm {
                observation.execution = "failed".to_string();
                observation.cleanup =
                    "completed after failed launch; private work retained".to_string();
                if started.close().is_err() {
                    observation.cleanup =
                        "failed after failed launch; private work retained".to_string();
                }
            }
            return (Some(failure.err), None);
        }
    };
    run_vm_phase(
        phase,
        evidence,
        &mut guest,
        &work,
        restart,
        hold,
        observation,
    )
}

pub(super) fn execute_action(
    phase: &Phase,
    evidence: &Evidence,
    opts: &RunOptions,
    vm_config: vm::VmConfig,
    remote: &Remote,
    observation: &mut Observation,
) -> (Option<Error>, Option<Error>) {
    match opts.action.as_str() {
        "exec" | "native" => execute_exec_or_native(
            phase,
            evidence,
            &opts.action,
            &opts.cmd_args,
            &opts.remote_file,
            &opts.request,
            &opts.revision,
            &opts.arch,
            &opts.target,
            remote,
            observation,
        ),
        "probe-ssh" => execute_probe_ssh(phase, evidence, remote, observation),
        "vm" => execute_vm(
            phase,
            evidence,
            vm_config,
            opts.restart,
            opts.hold,
            observation,
        ),
        _ => (None, None),
    }
}
