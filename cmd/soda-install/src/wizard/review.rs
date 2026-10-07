use super::steps::{ask_nav, step_subnet};
use super::DiskInstallChoices;

use crate::command::Runner;
use crate::console::Console;
use crate::disks::{disk_summary, Disk};
use crate::errors::Error;
use crate::signal::Ctx;

fn print_final_review(console: &Console, choices: &DiskInstallChoices, payload_bytes: u64) {
    console.page("Final review");
    console.print("ERASE ALL DATA on:");
    console.print(format_args!("  {}", disk_summary(&choices.disk.device)));
    if !choices.disk.blocked.is_empty() {
        console.print(format_args!("  Installer note: {}.", choices.disk.blocked));
    }
    if choices.disk.removable {
        console.print(
            "  Removable device: this target writes only with the explicit removable confirmation.",
        );
    }
    console.print(format_args!("Hostname: {}", choices.hostname));
    console.print(format_args!("Project subnet: {}", choices.subnet));
    console.print("Operator access: root password (local console and SSH)");
    console.print(format_args!(
        "Included Soda payload: {:.1} MiB verified",
        payload_bytes as f64 / (1u64 << 20) as f64
    ));
    console.print("Network settings will be copied to the installed system.");
    console.print("After writing, follow the completion screen for media removal and next steps.");
}

pub(super) fn erase_phrase(disk: &Disk) -> String {
    if disk.removable {
        return format!("ERASE REMOVABLE {}", disk.device.name);
    }
    format!("ERASE {}", disk.device.name)
}

fn confirm_final_review(console: &Console, ctx: &Ctx, disk: &Disk) -> Result<(), Error> {
    let phrase = erase_phrase(disk);
    loop {
        let answer = ask_nav(
            console,
            ctx,
            &format!("Type exactly {phrase}, back, restart, or cancel"),
        )?;
        if answer == phrase {
            return Ok(());
        }
        console.print("Confirmation did not match. No disk writing started.");
    }
}

pub(super) fn step_subnet_and_review(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    choices: &mut DiskInstallChoices,
    payload_bytes: u64,
) -> Result<(), Error> {
    let _ = run;
    loop {
        let subnet = step_subnet(console, ctx, &choices.subnet.clone())?;
        choices.subnet = subnet;
        print_final_review(console, choices, payload_bytes);
        match confirm_final_review(console, ctx, &choices.disk) {
            Err(Error::Back) => {
                choices.removable_ok = false;
            }
            Err(err) => {
                choices.removable_ok = false;
                return Err(err);
            }
            Ok(()) => {
                choices.removable_ok = choices.disk.removable;
                return Ok(());
            }
        }
    }
}
