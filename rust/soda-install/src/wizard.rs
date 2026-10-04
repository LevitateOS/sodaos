//! Five-step disk installation wizard: network, disk, hostname, password,
//! subnet plus final review.

use crate::command::Runner;
use crate::console::Console;
use crate::disks::{disk_summary, Disk};
use crate::errors::Error;
use crate::fmtx::{go_lower, sprintf, Arg};
use crate::inputs;
use crate::signal::Ctx;

#[derive(Debug, Clone)]
pub struct DiskInstallChoices {
    pub disk: Disk,
    pub hostname: String,
    pub password_hash: String,
    pub subnet: String,
    /// Set only by the final review for removable targets; travels with
    /// the confirmed disk to the writer.
    pub removable_ok: bool,
}

impl Default for DiskInstallChoices {
    fn default() -> DiskInstallChoices {
        DiskInstallChoices {
            disk: Disk::default(),
            hostname: String::new(),
            password_hash: String::new(),
            subnet: String::new(),
            removable_ok: false,
        }
    }
}

fn check_nav(value: &str) -> Option<Error> {
    match go_lower(value).as_str() {
        "back" => Some(Error::Back),
        "restart" => Some(Error::Restart),
        "cancel" => Some(Error::Cancel),
        _ => None,
    }
}

fn ask_nav(console: &Console, ctx: &Ctx, prompt: &str) -> Result<String, Error> {
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

fn step_network(ctx: &Ctx, console: &Console, run: &dyn Runner) -> Result<(), Error> {
    match console.network_with(ctx, run) {
        Ok(()) => Ok(()),
        Err(Error::Back) | Err(Error::Cancel) => Err(Error::Cancel),
        Err(err) => Err(err),
    }
}

fn handle_disk_inspect_failure(console: &Console, ctx: &Ctx) -> Result<(), Error> {
    console.page("Step 2 of 5 — Installation disk");
    console.print("Could not inspect disks. No disk installation started.", &[]);
    // Any non-navigation word retries; only navigation errors return.
    let _ = ask_nav(console, ctx, "Type retry, back, restart, or cancel")?;
    Ok(())
}

fn print_disk_list(console: &Console, disks: &[Disk], feedback: &str) {
    console.page("Step 2 of 5 — Installation disk");
    if !feedback.is_empty() {
        console.print("%s", &[Arg::Str(feedback)]);
        console.print("", &[]);
    }
    for (i, disk) in disks.iter().enumerate() {
        console.print("%d. %s", &[Arg::Uint((i + 1) as u64), Arg::Str(&disk_summary(&disk.device))]);
        for child in &disk.device.children {
            console.print(
                "   %q: %.1f GiB, filesystem %q",
                &[
                    Arg::Str(&child.name),
                    Arg::Float(child.size as f64 / (1u64 << 30) as f64),
                    Arg::Str(&child.fstype),
                ],
            );
        }
        if !disk.blocked.is_empty() {
            console.print("   Unavailable: %s", &[Arg::Str(&disk.blocked)]);
        }
        if disk.removable && disk.blocked.is_empty() {
            console.print("   Removable device: erasing it needs the explicit removable confirmation below.", &[]);
        }
    }
}

fn select_disk(disks: &[Disk], selected: &str) -> Result<Disk, String> {
    let index: i64 = match selected.parse() {
        Ok(index) => index,
        Err(_) => return Err("Choose an available disk number.".to_string()),
    };
    if index < 1 || index as usize > disks.len() {
        return Err("Choose an available disk number.".to_string());
    }
    let disk = disks[index as usize - 1].clone();
    if !disk.blocked.is_empty() {
        return Err(sprintf(
            "Disk %d is unavailable: %s. Choose an available disk number.",
            &[Arg::Int(index), Arg::Str(&disk.blocked)],
        ));
    }
    Ok(disk)
}

fn step_disk(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    inspect: &dyn Fn(&Ctx, &dyn Runner) -> Result<Vec<Disk>, Error>,
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

fn step_hostname(console: &Console, ctx: &Ctx, selected: &Disk, current: &str) -> Result<String, Error> {
    let mut feedback = String::new();
    loop {
        console.page("Step 3 of 5 — Hostname");
        if !feedback.is_empty() {
            console.print("%s", &[Arg::Str(&feedback)]);
            console.print("", &[]);
            feedback.clear();
        }
        console.print("Selected disk: %s", &[Arg::Str(&disk_summary(&selected.device))]);
        let default = if current.is_empty() { "soda" } else { current };
        let mut value = ask_nav(console, ctx, &format!("Hostname [{default}], back, restart, or cancel"))?;
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

fn valid_password(password: &[u8], confirmation: &[u8]) -> bool {
    match std::str::from_utf8(password) {
        Ok(text) => text.chars().count() >= MIN_PASSWORD_RUNES && password == confirmation,
        Err(_) => false,
    }
}

fn password_feedback(password: &[u8], confirmation: &[u8]) -> String {
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
        .run(ctx, "openssl", &["passwd".to_string(), "-6".to_string(), "-stdin".to_string()], Some(&input))
        .map_err(|_| Error::msg("password hashing failed"))?;
    Ok(crate::fmtx::go_trim_space(&String::from_utf8_lossy(&hash)).to_string())
}

fn step_password(ctx: &Ctx, console: &Console, run: &dyn Runner) -> Result<String, Error> {
    let mut feedback = String::new();
    loop {
        console.page("Step 4 of 5 — Native operator password");
        if !feedback.is_empty() {
            console.print("%s", &[Arg::Str(&feedback)]);
            console.print("", &[]);
            feedback.clear();
        }
        console.print("This password is for root login after reboot, local console and SSH.", &[]);
        console.print("Use at least 12 characters; both entries must match.", &[]);
        console.print("Enroll a key later and disable password logins yourself to go key-only.", &[]);
        console.print("Type back, restart, or cancel in a password field to navigate.", &[]);
        let password = ask_secret_nav(console, ctx, "Password")?;
        let confirmation = ask_secret_nav(console, ctx, "Confirm password")?;
        if !valid_password(&password, &confirmation) {
            feedback = password_feedback(&password, &confirmation);
            continue;
        }
        return hash_password(ctx, run, &password);
    }
}

fn step_subnet(console: &Console, ctx: &Ctx, current: &str) -> Result<String, Error> {
    let mut feedback = String::new();
    loop {
        console.page("Step 5 of 5 — Project network and review");
        if !feedback.is_empty() {
            console.print("%s", &[Arg::Str(&feedback)]);
            console.print("", &[]);
            feedback.clear();
        }
        console.print(
            "Developer client routing is configured separately. Any canonical IPv4 range is accepted, including public or overlapping ranges.",
            &[],
        );
        let default = if current.is_empty() { "10.89.0.0/24" } else { current };
        let mut subnet = ask_nav(console, ctx, &format!("Project IPv4 subnet [{default}], back, restart, or cancel"))?;
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

fn print_final_review(console: &Console, choices: &DiskInstallChoices, payload_bytes: u64) {
    console.page("Final review");
    console.print("ERASE ALL DATA on:", &[]);
    console.print("  %s", &[Arg::Str(&disk_summary(&choices.disk.device))]);
    if !choices.disk.blocked.is_empty() {
        console.print("  Installer note: %s.", &[Arg::Str(&choices.disk.blocked)]);
    }
    if choices.disk.removable {
        console.print("  Removable device: this target writes only with the explicit removable confirmation.", &[]);
    }
    console.print("Hostname: %s", &[Arg::Str(&choices.hostname)]);
    console.print("Project subnet: %s", &[Arg::Str(&choices.subnet)]);
    console.print("Operator access: root password (local console and SSH)", &[]);
    console.print(
        "Included Soda payload: %.1f MiB verified",
        &[Arg::Float(payload_bytes as f64 / (1u64 << 20) as f64)],
    );
    console.print("Network settings will be copied to the installed system.", &[]);
    console.print("After writing, follow the completion screen for media removal and next steps.", &[]);
}

fn erase_phrase(disk: &Disk) -> String {
    if disk.removable {
        return format!("ERASE REMOVABLE {}", disk.device.name);
    }
    format!("ERASE {}", disk.device.name)
}

fn confirm_final_review(console: &Console, ctx: &Ctx, disk: &Disk) -> Result<(), Error> {
    let phrase = erase_phrase(disk);
    loop {
        let answer = ask_nav(console, ctx, &format!("Type exactly {phrase}, back, restart, or cancel"))?;
        if answer == phrase {
            return Ok(());
        }
        console.print("Confirmation did not match. No disk writing started.", &[]);
    }
}

fn step_subnet_and_review(
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

fn dispatch_install_step(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    inspect: &dyn Fn(&Ctx, &dyn Runner) -> Result<Vec<Disk>, Error>,
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
    inspect: &dyn Fn(&Ctx, &dyn Runner) -> Result<Vec<Disk>, Error>,
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::console::test_support::*;
    use std::io::Write;

    #[test]
    fn selections_and_phrases() {
        let disk = Disk {
            device: crate::disks::BlockDevice {
                name: "/dev/sda".to_string(),
                kname: "/dev/sda".to_string(),
                device_type: "disk".to_string(),
                tran: String::new(),
                size: 0,
                model: String::new(),
                serial: String::new(),
                wwn: String::new(),
                major_minor: "8:0".to_string(),
                read_only: false,
                mountpoints: Some(vec![]),
                fstype: String::new(),
                uuid: String::new(),
                partuuid: String::new(),
                children: Vec::new(),
            },
            sequence: "1".to_string(),
            blocked: String::new(),
            removable: false,
        };
        assert!(select_disk(&[disk.clone()], "1").is_ok());
        assert_eq!(select_disk(&[disk.clone()], "0").unwrap_err(), "Choose an available disk number.");
        assert_eq!(select_disk(&[disk.clone()], "2").unwrap_err(), "Choose an available disk number.");
        assert_eq!(select_disk(&[disk.clone()], "x").unwrap_err(), "Choose an available disk number.");
        assert_eq!(select_disk(&[disk.clone()], "+1").unwrap().device.name, "/dev/sda");
        let mut blocked = disk.clone();
        blocked.blocked = "busy".to_string();
        assert_eq!(
            select_disk(&[blocked], "1").unwrap_err(),
            "Disk 1 is unavailable: busy. Choose an available disk number."
        );
        assert_eq!(erase_phrase(&disk), "ERASE /dev/sda");
        let mut removable = disk.clone();
        removable.removable = true;
        assert_eq!(erase_phrase(&removable), "ERASE REMOVABLE /dev/sda");
        assert!(valid_password(b"twelve-chars!!", b"twelve-chars!!"));
        assert!(!valid_password(b"short", b"short"));
        assert!(!valid_password(b"twelve-chars!!", b"twelve-chars!?"));
        assert!(!valid_password(b"\xff\xfe-twelve!!", b"\xff\xfe-twelve!!"));
        assert_eq!(password_feedback(b"a", b"b"), "Passwords do not match.");
        assert_eq!(password_feedback(b"short", b"short"), "Use at least 12 characters.");
        assert_eq!(password_feedback(b"twelve-chars!!", b"twelve-chars!!"), "");
        assert_eq!(password_feedback(b"\xff\xfe-twelve!!", b"\xff\xfe-twelve!!"), "Use at least 12 characters.");
    }

    #[test]
    fn full_wizard_flow() {
        let pty = open_pty();
        let slave = pty.slave_path.clone();
        let mut master = pty.master;
        let worker = std::thread::spawn(move || {
            let console = Console::open(&slave).unwrap();
            let (ctx, _flag) = Ctx::test();
            let runner = crate::command::FnRunner::new(|_, name, args, _| {
                if name == "ip" {
                    return Ok(b"lo UP 127.0.0.1/8\n".to_vec());
                }
                if name == "openssl" {
                    assert_eq!(args, &["passwd".to_string(), "-6".to_string(), "-stdin".to_string()]);
                    return Ok(b"$6$salt$hash\n".to_vec());
                }
                panic!("unexpected command {name}");
            });
            let disk = Disk {
                device: crate::disks::BlockDevice {
                    name: "/dev/sda".to_string(),
                    kname: "/dev/sda".to_string(),
                    device_type: "disk".to_string(),
                    tran: String::new(),
                    size: 64 << 30,
                    model: "M".to_string(),
                    serial: "S".to_string(),
                    wwn: "W".to_string(),
                    major_minor: "8:0".to_string(),
                    read_only: false,
                    mountpoints: Some(vec![None]),
                    fstype: String::new(),
                    uuid: String::new(),
                    partuuid: String::new(),
                    children: Vec::new(),
                },
                sequence: "9".to_string(),
                blocked: String::new(),
                removable: false,
            };
            collect_disk_install_choices(&ctx, &console, &runner, &|_, _| Ok(vec![disk.clone()]), 10 << 20)
        });
        let mut output = read_until(&mut master, b"Type keep, edit, back, restart, or cancel: ", 3000);
        master.write_all(b"keep\n").unwrap();
        output.extend(read_until(&mut master, b"Type yes to use them, edit, back, restart, or cancel: ", 3000));
        master.write_all(b"yes\n").unwrap();
        output.extend(read_until(&mut master, b"Disk number, back, restart, or cancel: ", 3000));
        master.write_all(b"1\n").unwrap();
        output.extend(read_until(&mut master, b"Hostname [soda], back, restart, or cancel: ", 3000));
        master.write_all(b"\n").unwrap();
        output.extend(read_until(&mut master, b"Password: ", 3000));
        master.write_all(b"twelve-chars!!\n").unwrap();
        output.extend(read_until(&mut master, b"Confirm password: ", 3000));
        master.write_all(b"twelve-chars!!\n").unwrap();
        output.extend(read_until(&mut master, b"Project IPv4 subnet [10.89.0.0/24], back, restart, or cancel: ", 3000));
        master.write_all(b"\n").unwrap();
        output.extend(read_until(&mut master, b"Type exactly ERASE /dev/sda, back, restart, or cancel: ", 3000));
        master.write_all(b"ERASE /dev/sda\n").unwrap();
        let choices = worker.join().unwrap().unwrap();
        output.extend(drain_idle(&mut master, 200));
        assert_eq!(choices.hostname, "soda");
        assert_eq!(choices.subnet, "10.89.0.0/24");
        assert_eq!(choices.password_hash, "$6$salt$hash");
        assert!(!choices.removable_ok);
        let text = String::from_utf8_lossy(&output);
        assert!(text.contains("Final review"));
        assert!(text.contains("ERASE ALL DATA on:"));
        assert!(text.contains("10.0 MiB verified"));
    }
}
