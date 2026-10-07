use super::DiskInspector;

use crate::command::Runner;
use crate::console::Console;
use crate::disks::{disk_summary, Disk};
use crate::errors::Error;
use crate::inputs;
use crate::signal::Ctx;

fn check_nav(value: &str) -> Option<Error> {
    match value.to_ascii_lowercase().as_str() {
        "back" => Some(Error::Back),
        "restart" => Some(Error::Restart),
        "cancel" => Some(Error::Cancel),
        _ => None,
    }
}

pub(super) fn ask_nav(console: &Console, ctx: &Ctx, prompt: &str) -> Result<String, Error> {
    let value = console.ask(ctx, prompt)?;
    if let Some(nav) = check_nav(&value) {
        return Err(nav);
    }
    Ok(value)
}

fn ask_secret_nav(console: &Console, ctx: &Ctx, prompt: &str) -> Result<Vec<u8>, Error> {
    let value = console.secret(ctx, prompt)?;
    // Navigation words are ASCII; lossy decoding detects them exactly like
    // Go comparing the raw bytes.
    if let Some(nav) = check_nav(&String::from_utf8_lossy(&value)) {
        return Err(nav);
    }
    Ok(value)
}

pub(super) fn step_network(ctx: &Ctx, console: &Console, run: &dyn Runner) -> Result<(), Error> {
    match console.network_with(ctx, run) {
        Ok(()) => Ok(()),
        Err(Error::Back) | Err(Error::Cancel) => Err(Error::Cancel),
        Err(err) => Err(err),
    }
}

fn handle_disk_inspect_failure(console: &Console, ctx: &Ctx) -> Result<(), Error> {
    console.page("Step 2 of 5 — Installation disk");
    console.print(
        "Could not inspect disks. No disk installation started.");
    // Any non-navigation word retries; only navigation errors return.
    let _ = ask_nav(console, ctx, "Type retry, back, restart, or cancel")?;
    Ok(())
}

fn print_disk_list(console: &Console, disks: &[Disk], feedback: &str) {
    console.page("Step 2 of 5 — Installation disk");
    if !feedback.is_empty() {
        console.print(feedback);
        console.print("");
    }
    for (i, disk) in disks.iter().enumerate() {
        console.print(
            format_args!("{}. {}", i + 1, disk_summary(&disk.device)),
        );
        for child in &disk.device.children {
            console.print(
                format_args!(
                    "   {:?}: {:.1} GiB, filesystem {:?}",
                    child.name,
                    child.size as f64 / (1u64 << 30) as f64,
                    child.fstype
                ),
            );
        }
        if !disk.blocked.is_empty() {
            console.print(format_args!("   Unavailable: {}", disk.blocked));
        }
        if disk.removable && disk.blocked.is_empty() {
            console.print(
                "   Removable device: erasing it needs the explicit removable confirmation below.");
        }
    }
}

pub(super) fn select_disk(disks: &[Disk], selected: &str) -> Result<Disk, String> {
    let index: i64 = match selected.parse() {
        Ok(index) => index,
        Err(_) => return Err("Choose an available disk number.".to_string()),
    };
    if index < 1 || index as usize > disks.len() {
        return Err("Choose an available disk number.".to_string());
    }
    let disk = disks[index as usize - 1].clone();
    if !disk.blocked.is_empty() {
        return Err(format!(
            "Disk {index} is unavailable: {}. Choose an available disk number.",
            disk.blocked
        ));
    }
    Ok(disk)
}

pub(super) fn step_disk(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    inspect: DiskInspector<'_>,
) -> Result<Disk, Error> {
    let mut feedback = String::new();
    loop {
        match inspect(ctx, run) {
            Err(_) => {
                handle_disk_inspect_failure(console, ctx)?;
                continue;
            }
            Ok(disks) => {
                print_disk_list(console, &disks, &feedback);
                feedback.clear();
                let selected = ask_nav(console, ctx, "Disk number, back, restart, or cancel")?;
                match select_disk(&disks, &selected) {
                    Ok(disk) => return Ok(disk),
                    Err(retry) => feedback = retry,
                }
            }
        }
    }
}

pub(super) fn step_hostname(
    console: &Console,
    ctx: &Ctx,
    selected: &Disk,
    current: &str,
) -> Result<String, Error> {
    let mut feedback = String::new();
    loop {
        console.page("Step 3 of 5 — Hostname");
        if !feedback.is_empty() {
            console.print(&feedback);
            console.print("");
            feedback.clear();
        }
        console.print(
            format_args!("Selected disk: {}", disk_summary(&selected.device)),
        );
        let default = if current.is_empty() { "soda" } else { current };
        let mut value = ask_nav(
            console,
            ctx,
            &format!("Hostname [{default}], back, restart, or cancel"),
        )?;
        if value.is_empty() {
            value = default.to_string();
        }
        if !inputs::hostname(&value) {
            feedback = "Use lowercase letters, digits, dots, and interior hyphens.".to_string();
            continue;
        }
        return Ok(value);
    }
}

const MIN_PASSWORD_RUNES: usize = 12;

pub(super) fn valid_password(password: &[u8], confirmation: &[u8]) -> bool {
    match std::str::from_utf8(password) {
        Ok(text) => text.chars().count() >= MIN_PASSWORD_RUNES && password == confirmation,
        Err(_) => false,
    }
}

pub(super) fn password_feedback(password: &[u8], confirmation: &[u8]) -> String {
    if password != confirmation {
        return "Passwords do not match.".to_string();
    }
    match std::str::from_utf8(password) {
        Ok(text) if text.chars().count() >= MIN_PASSWORD_RUNES => String::new(),
        _ => "Use at least 12 characters.".to_string(),
    }
}

fn hash_password(ctx: &Ctx, run: &dyn Runner, password: &[u8]) -> Result<String, Error> {
    let mut input = password.to_vec();
    input.push(b'\n');
    let hash = run
        .run(
            ctx,
            "openssl",
            &["passwd".to_string(), "-6".to_string(), "-stdin".to_string()],
            Some(&input),
        )
        .map_err(|_| Error::msg("password hashing failed"))?;
    Ok(String::from_utf8_lossy(&hash).trim().to_string())
}

pub(super) fn step_password(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
) -> Result<String, Error> {
    let mut feedback = String::new();
    loop {
        console.page("Step 4 of 5 — Native operator password");
        if !feedback.is_empty() {
            console.print(&feedback);
            console.print("");
            feedback.clear();
        }
        console.print(
            "This password is for root login after reboot, local console and SSH.");
        console.print("Use at least 12 characters; both entries must match.");
        console.print(
            "Enroll a key later and disable password logins yourself to go key-only.");
        console.print(
            "Type back, restart, or cancel in a password field to navigate.");
        let password = ask_secret_nav(console, ctx, "Password")?;
        let confirmation = ask_secret_nav(console, ctx, "Confirm password")?;
        if !valid_password(&password, &confirmation) {
            feedback = password_feedback(&password, &confirmation);
            continue;
        }
        return hash_password(ctx, run, &password);
    }
}

pub(super) fn step_subnet(console: &Console, ctx: &Ctx, current: &str) -> Result<String, Error> {
    let mut feedback = String::new();
    loop {
        console.page("Step 5 of 5 — Project network and review");
        if !feedback.is_empty() {
            console.print(&feedback);
            console.print("");
            feedback.clear();
        }
        console.print(
            "Developer client routing is configured separately. Any canonical IPv4 range is accepted, including public or overlapping ranges.");
        let default = if current.is_empty() {
            "10.89.0.0/24"
        } else {
            current
        };
        let mut subnet = ask_nav(
            console,
            ctx,
            &format!("Project IPv4 subnet [{default}], back, restart, or cancel"),
        )?;
        if subnet.is_empty() {
            subnet = default.to_string();
        }
        if let Err(err) = inputs::project_subnet(&subnet) {
            feedback = format!("Invalid subnet: {err}");
            continue;
        }
        return Ok(subnet);
    }
}
