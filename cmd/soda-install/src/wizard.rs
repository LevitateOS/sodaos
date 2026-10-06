//! Five-step disk installation wizard: network, disk, hostname, password,
//! subnet plus final review.

use crate::command::Runner;
use crate::console::Console;
use crate::disks::Disk;
use crate::errors::Error;
use crate::signal::Ctx;

use self::review::{erase_phrase, step_subnet_and_review};
use self::steps::{
    password_feedback, select_disk, step_disk, step_hostname, step_network, step_password,
    valid_password,
};

type DiskInspector<'a> = &'a dyn Fn(&Ctx, &dyn Runner) -> Result<Vec<Disk>, Error>;

#[derive(Debug, Clone, Default)]
pub struct DiskInstallChoices {
    pub disk: Disk,
    pub hostname: String,
    pub password_hash: String,
    pub subnet: String,
    /// Set only by the final review for removable targets; travels with
    /// the confirmed disk to the writer.
    pub removable_ok: bool,
}

fn dispatch_install_step(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    inspect: DiskInspector<'_>,
    payload_bytes: u64,
    step: i32,
    result: &mut DiskInstallChoices,
) -> Result<(), Error> {
    match step {
        0 => step_network(ctx, console, run),
        // Go assigns the step's zero value on error, so back-navigation
        // defaults reset exactly: a failed disk step clears the disk and
        // the removable confirmation, a failed hostname step clears the
        // hostname default, a failed password step clears the hash.
        1 => match step_disk(ctx, console, run, inspect) {
            Ok(disk) => {
                result.disk = disk;
                result.removable_ok = false;
                Ok(())
            }
            Err(err) => {
                result.disk = Disk::default();
                result.removable_ok = false;
                Err(err)
            }
        },
        2 => {
            let disk = result.disk.clone();
            let current = result.hostname.clone();
            match step_hostname(console, ctx, &disk, &current) {
                Ok(hostname) => {
                    result.hostname = hostname;
                    Ok(())
                }
                Err(err) => {
                    result.hostname.clear();
                    Err(err)
                }
            }
        }
        3 => match step_password(ctx, console, run) {
            Ok(hash) => {
                result.password_hash = hash;
                Ok(())
            }
            Err(err) => {
                result.password_hash.clear();
                Err(err)
            }
        },
        4 => {
            let err = step_subnet_and_review(ctx, console, run, result, payload_bytes);
            if err.is_err() {
                result.password_hash.clear();
            }
            err
        }
        _ => Ok(()),
    }
}

pub fn collect_disk_install_choices(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    inspect: DiskInspector<'_>,
    payload_bytes: u64,
) -> Result<DiskInstallChoices, Error> {
    let mut result = DiskInstallChoices::default();
    let mut step = 0;
    loop {
        match dispatch_install_step(ctx, console, run, inspect, payload_bytes, step, &mut result) {
            Err(Error::Back) => step -= 1,
            Err(err) => return Err(err),
            Ok(()) if step == 4 => return Ok(result),
            Ok(()) => step += 1,
        }
    }
}

mod review;
mod steps;

#[cfg(test)]
mod tests;
