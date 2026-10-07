// soda-muse-maintain installs public Muse tools and restores a live project interface.
use std::time::{Duration, Instant};

mod archive;
mod command;
mod config;
mod config_validation;
mod config_wire;
mod filesystem;
mod interface;
mod interface_admission;
mod network;
mod options;
mod project;
mod release;
mod release_validation;
mod release_wire;
mod sha256;
mod stage;
#[cfg(test)]
mod test_support;

use config::{load_config, Config};
use filesystem::load_tools;
use interface::{attach_interface, prepare_interface};
use options::{parse, Options};
use project::wait_project;
use stage::{ensure_system_bus, stage_tools};

const MUSE_VERSION: &str = "1.4.0-R4161.1";
const RELEASE_PATH: &str = "/usr/share/soda/release.json";

fn main() {
    if let Err(e) = run() {
        eprintln!("soda-muse-maintain: {e}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("project maintenance requires root"));
    }
    let argv: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    let o = parse(&argv)?;
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
    let sources = load_tools(&o.tools, &c.muse_sha256, &c.muse_version)?;
    let deadline = Instant::now() + Duration::from_secs(120);
    let target = wait_project(&o.project, deadline)?;
    if !o.bind_only {
        ensure_system_bus(&target, deadline)?;
        stage_tools(&target, &sources, deadline)?;
    }
    prepare_interface(&target, deadline)?;
    attach_interface(&target, &c.muse_socket, deadline)
}
