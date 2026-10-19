// soda-project-maintain performs explicit retained Project tool and release maintenance.
use std::time::{Duration, Instant};

mod archive;
mod canlang;
mod command;
mod config;
mod config_validation;
mod config_wire;
mod factory_mcp;
mod filesystem;
mod importer_custody;
mod interface;
mod interface_admission;
mod network;
mod options;
mod project;
mod release;
mod release_extensions;
mod release_inventory;
mod release_operation;
mod release_stage;
mod release_target;
mod release_wire;
mod sha256;
mod stage;
mod support_root;
mod terminal_support;
#[cfg(test)]
mod test_support;

use config::{load_config, Config};
use filesystem::load_tools;
use interface::{attach_interface, prepare_interface};
use options::{parse, Options};
use project::wait_project;
use stage::{ensure_system_bus, stage_tools};

const RELEASE_PATH: &str = "/usr/share/soda/release.json";

fn main() {
    let argv: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|arg| arg.to_string_lossy().into_owned())
        .collect();
    if unsafe { libc::geteuid() } == 0 {
        let release = match argv.first().map(String::as_str) {
            Some("release") => Some(release_operation::run(&argv[1..])),
            Some("release-reconcile") => Some(release_operation::reconcile_boot(&argv[1..])),
            _ => None,
        };
        if let Some(result) = release {
            if let Err(error) = result {
                eprintln!("soda-project-maintain: {error}");
                std::process::exit(error.exit_code());
            }
            return;
        }
    }
    if let Err(e) = run(&argv) {
        eprintln!("soda-project-maintain: {e}");
        std::process::exit(1);
    }
}

fn run(argv: &[String]) -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("project maintenance requires root"));
    }
    match argv.first().map(String::as_str) {
        Some("terminal-support") => return terminal_support::run(&argv[1..]),
        Some("factory-mcp") => return factory_mcp::run(&argv[1..]),
        Some("canlang") => return canlang::run(&argv[1..]),
        Some("muse") => {}
        _ => {
            return Err(
                "select muse, canlang, factory-mcp, terminal-support or release maintenance"
                    .to_string(),
            )
        }
    }
    let o = parse(&argv[1..])?;
    let c = load_config(&o.config)?;
    if c.muse_sha256.is_empty() {
        if o.bind_only {
            return Ok(());
        }
        return Err(String::from("muse project runtime is disabled"));
    }
    maintain(&o, &c)
}

fn is_lower_hex(b: u8) -> bool {
    b.is_ascii_digit() || (b'a'..=b'f').contains(&b)
}

fn maintain(o: &Options, c: &Config) -> Result<(), String> {
    // Tool closes its own fd on drop, mirroring deferred closeTools.
    let sources = load_tools(&o.tools, &c.muse_sha256)?;
    let deadline = Instant::now() + Duration::from_secs(120);
    let target = wait_project(&o.project, deadline)?;
    if !o.bind_only {
        ensure_system_bus(&target, deadline)?;
        stage_tools(&target, &sources, deadline)?;
    }
    prepare_interface(&target, deadline)?;
    attach_interface(&target, &c.muse_socket, deadline)
}
