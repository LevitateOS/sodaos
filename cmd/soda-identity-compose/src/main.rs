// soda-identity-compose registers one explicitly opted-in Compose service.
#[cfg(test)]
use std::fs;

mod compose;
mod launch_wire;
mod options;
mod registration;

use compose::launch_compose;
use launch_wire::NestedRegistration;
use options::{parse_options, Options};
use registration::{account, register, registration_root};

const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";

fn main() {
    if let Err(e) = run() {
        eprintln!("soda-identity-compose: {e}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), String> {
    let o = load_options()?;
    let actor = account(&o.login)?;
    let (root, registration) = registration_root()?;
    let child = launch_compose(&o, &root)?;
    register(NestedRegistration {
        child_id: child,
        actor_id: actor.to_string(),
        registration_id: registration,
        muse: o.muse,
    })
}

fn load_options() -> Result<Options, String> {
    // Never panic on non-UTF-8 argv; Go replaces invalid bytes with U+FFFD.
    let args: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    parse_options(&args)
}

#[cfg(test)]
#[path = "compose_tests.rs"]
mod compose_tests;
