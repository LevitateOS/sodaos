use std::env;

use super::process::{capture, ls_nonempty, run, Captured};
use super::{command_v, fail, id_un, is_dir, Exit, LEGACY_HOME, WORKER_USER};

/// `units_have_active` mirrors the `awk` over `systemctl list-units`: any
/// unit whose third field is active, activating, or deactivating blocks.
pub(super) fn units_have_active(output: &[u8]) -> bool {
    let text = String::from_utf8_lossy(output);
    text.lines().any(|line| {
        matches!(
            line.split_whitespace().nth(2),
            Some("active" | "activating" | "deactivating")
        )
    })
}

pub(super) fn refuse_active_build() -> Result<(), Exit> {
    let args = [
        "list-units",
        "--all",
        "--type=service",
        "--no-legend",
        "--plain",
        "soda-build-*",
    ];
    match capture("systemctl", &args, true) {
        Captured::SpawnFailed(_) => {
            fail("cannot list worker units; refusing to touch shared build state")
        }
        Captured::Done(code, out) => {
            if code != 0 {
                return fail("cannot list worker units; refusing to touch shared build state");
            }
            if units_have_active(&out) {
                return fail("a candidate build is still active; finish it before rerunning setup");
            }
            Ok(())
        }
    }
}

pub(super) struct Storage {
    pub(super) root: String,
    pub(super) home: String,
    pub(super) run: String,
    pub(super) scratch: String,
}

pub(super) fn migrate_candidate_home(storage: &Storage) -> Result<(), Exit> {
    if !is_dir(LEGACY_HOME) {
        return Ok(());
    }
    if ls_nonempty(&storage.home) {
        println!("-- new worker home already populated; legacy {LEGACY_HOME} preserved untouched");
        return Ok(());
    }
    refuse_active_build()?;
    if command_v("pgrep").is_some() {
        // Both streams silenced like `>/dev/null 2>&1`; captured stdout is
        // discarded and only the exit status decides.
        let runs = match capture("pgrep", &["-u", WORKER_USER], true) {
            Captured::SpawnFailed(_) => false,
            Captured::Done(code, _) => code == 0,
        };
        if runs {
            return fail(
                "soda-build-worker still owns processes; finish them before migrating heavy state",
            );
        }
    }
    println!(
        "-- migrating legacy worker home to {} (legacy preserved)",
        storage.home
    );
    let from = format!("{LEGACY_HOME}/.");
    let dest = format!("{}/", storage.home);
    run("sudo", &["cp", "-a", &from, &dest])?;
    let owned = format!("{WORKER_USER}:{WORKER_USER}");
    run("sudo", &["chown", "-R", &owned, &storage.home])?;
    println!("-- legacy {LEGACY_HOME} preserved; retire it explicitly (D2) after the new home proves itself");
    Ok(())
}

/// Scratch setup for the candidate storage root.
pub(super) fn prepare_scratch(storage: &Storage) -> Result<(), Exit> {
    println!("-- candidate storage root ({})", storage.root);
    run("sudo", &["mkdir", "-p", &storage.scratch])?;
    let scratch_owner = match env::var("SUDO_USER") {
        Ok(owner) if !owner.is_empty() => owner,
        _ => String::from_utf8_lossy(&id_un()).into_owned(),
    };
    run("sudo", &["chown", &scratch_owner, &storage.scratch])?;
    Ok(())
}
