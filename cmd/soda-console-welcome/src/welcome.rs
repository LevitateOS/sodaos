use std::env;
use std::fs;
use std::process::{Command, Stdio};

use crate::config::render_config;

pub(crate) fn run() -> i32 {
    let argv: Vec<String> = env::args().collect();
    if argv.len() > 2 {
        let prog = &argv[0];
        eprintln!("usage: {prog} [DASHBOARD_CONFIG]");
        return 2;
    }
    // `[ "$(id -u)" = 0 ] || exit 0`: a real subprocess, so PATH doubles
    // (and a missing id) behave exactly as in the shell.
    if capture("id", &["-u"], &[]).trim_end_matches('\n') != "0" {
        return 0;
    }
    let config = argv
        .get(1)
        .map(String::as_str)
        .unwrap_or("/etc/soda/dashboard.json");

    println!("\nWelcome to Soda OS — host operator console");
    println!("The operating system for software factories.");
    println!(
        "Run your coding agents on your infrastructure, with controlled access and reviewable results."
    );
    println!("Hostname: {}", hostname());

    // Main-table lookup: an exit node may install routes in a separate
    // table. Connected Ethernet/Wi-Fi without a default route counts too.
    let mut devices: Vec<String> = Vec::new();
    for line in capture(
        "ip",
        &["-o", "-4", "route", "show", "default", "table", "main"],
        &[],
    )
    .lines()
    {
        let fields: Vec<&str> = line.split_whitespace().collect();
        for i in 0..fields.len().saturating_sub(1) {
            if fields[i] == "dev" {
                devices.push(fields[i + 1].to_string());
            }
        }
    }
    for line in capture(
        "nmcli",
        &["-t", "-f", "DEVICE,TYPE,STATE", "device", "status"],
        &[("LC_ALL", "C")],
    )
    .lines()
    {
        let fields: Vec<&str> = line.split(':').collect();
        if fields.len() >= 3
            && fields[2] == "connected"
            && (fields[1] == "ethernet" || fields[1] == "wifi")
        {
            devices.push(fields[0].to_string());
        }
    }
    // `awk '!seen[$0]++'`: exact-string dedupe, first occurrence wins.
    let mut seen = std::collections::HashSet::new();
    devices.retain(|d| seen.insert(d.clone()));

    let mut found = false;
    let mut tunnel_host = String::new();
    for device in &devices {
        // `case "$device" in lo|tailscale*) continue`.
        if device == "lo" || device.starts_with("tailscale") {
            continue;
        }
        let out = capture(
            "ip",
            &["-o", "-4", "addr", "show", "dev", device, "scope", "global"],
            &[],
        );
        for line in out.lines() {
            // `awk '{sub(/\/.*/, "", $4); print $4}'`, empties dropped by
            // the shell word split.
            let field = line.split_whitespace().nth(3).unwrap_or("");
            let address = field.split('/').next().unwrap_or("");
            if address.is_empty() {
                continue;
            }
            found = true;
            if tunnel_host.is_empty() {
                tunnel_host = address.to_string();
            }
            println!("Observed local IPv4 ({device}): {address}");
        }
    }
    if !found {
        println!("No local uplink IPv4 address is currently available.");
    }
    if !tunnel_host.is_empty() {
        println!("\nCockpit: https://{tunnel_host}:9090");
        println!(
            "Self-signed certificate; expect a browser warning. Root password login is enabled."
        );
    } else {
        println!(
            "\nCockpit: unavailable until a local uplink IPv4 address exists; configure networking first."
        );
    }

    render_config(config, &tunnel_host);

    // `command -v soda-tailnet`, run with inherited stdio; failures ignored.
    if have_command("soda-tailnet") {
        let _ = Command::new("soda-tailnet").status();
    } else {
        println!("\nTailnet status helper is unavailable; inspect native tailscaled.service.");
    }

    println!("\nProject developers use their project IPs, not host Linux accounts.");
    println!("People authorize factory work and merge the final verified commit in Forgejo.");
    println!(
        "Persistent human projects support development and intervention; run cleanup leaves them intact."
    );
    println!("Host Tailnet enrollment does not establish project-subnet routing.");
    println!("No firewall, listener, enrollment or service configuration was changed.");
    0
}

/// `$(...)`: stdout captured, stderr discarded, "" on any failure.
fn capture(program: &str, args: &[&str], extra_env: &[(&str, &str)]) -> String {
    let mut command = Command::new(program);
    command
        .args(args)
        .stdout(Stdio::piped())
        .stderr(Stdio::null());
    for (key, value) in extra_env {
        command.env(key, value);
    }
    command
        .output()
        .ok()
        .filter(|out| out.status.success())
        .map(|out| String::from_utf8_lossy(&out.stdout).into_owned())
        .unwrap_or_default()
}

/// `command -v`: an executable file on PATH.
fn have_command(program: &str) -> bool {
    env::var_os("PATH").is_some_and(|paths| {
        env::split_paths(&paths).any(|dir| {
            let candidate = dir.join(program);
            fs::metadata(&candidate).is_ok_and(|info| {
                use std::os::unix::fs::PermissionsExt;
                info.is_file() && info.permissions().mode() & 0o111 != 0
            })
        })
    })
}

fn hostname() -> String {
    // `$(...)` strips trailing newlines only, never other whitespace.
    let host = capture("hostnamectl", &["--static"], &[]);
    let host = host.trim_end_matches('\n');
    if !host.is_empty() {
        return host.to_string();
    }
    capture("uname", &["-n"], &[])
        .trim_end_matches('\n')
        .to_string()
}
