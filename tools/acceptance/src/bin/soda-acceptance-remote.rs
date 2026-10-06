// One explicitly requested target-side payload phase; piped over SSH,
// never installed. Dispatches the compiled payloads: the exact-source
// native phase plus the two root-run snapshot probes.
use soda_acceptance::{cockpit, native_phase, project_state};

fn native_phase(request: &str) -> i32 {
    native_phase::restrict_umask();
    let platform = match native_phase::native_platform() {
        Ok(platform) => platform,
        Err(failure) => {
            eprintln!(
                "Native phase failed ({}); run retained.",
                failure.kind_name()
            );
            return 1;
        }
    };
    match native_phase::run_phase(request, &platform, &native_phase::SystemRunner) {
        Ok(()) => 0,
        Err(failure) => {
            eprintln!(
                "Native phase failed ({}); run retained.",
                failure.kind_name()
            );
            1
        }
    }
}

fn cockpit_account() -> i32 {
    let platform = match native_phase::native_platform() {
        Ok(platform) => platform,
        Err(_) => {
            eprintln!("Cockpit account failed: AssertionError PAM initialization failed; this is not access denial");
            return 1;
        }
    };
    let validate = std::env::var("SODA_NATIVE_VALIDATE").ok();
    match cockpit::run_cockpit(
        unsafe { libc::getuid() },
        validate.as_deref(),
        &platform.hostname,
    ) {
        Ok(line) => {
            println!("{line}");
            0
        }
        Err(failure) => {
            eprintln!("{failure}");
            1
        }
    }
}

fn project_state() -> i32 {
    match project_state::run_snapshot() {
        Ok(data) => {
            println!("{}", project_state::dumps_sorted(&data));
            0
        }
        Err(failure) => {
            eprintln!("{failure}");
            1
        }
    }
}

fn main() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let code = match args
        .iter()
        .map(|arg| arg.as_str())
        .collect::<Vec<_>>()
        .as_slice()
    {
        ["native-phase", request] => native_phase(request),
        ["cockpit-account"] => cockpit_account(),
        ["project-state"] => project_state(),
        _ => {
            eprintln!("usage: soda-acceptance-remote native-phase|cockpit-account|project-state");
            1
        }
    };
    std::process::exit(code);
}
