use std::env;
use std::fs;

use crate::process::{capture, stripped_string, Captured};
use crate::{Exit, AUTHORITY, FAIL_PREFIX};

/// `current_pwd` mirrors the script's bare `$PWD` in the metadata glob: the
/// inherited value verbatim (even when stale or empty), or the `set -u`
/// crash when unset. Only the crash prefix differs (fixed here instead of
/// `$0`-and-line), since invocation paths differ inherently.
fn current_pwd() -> Result<String, Exit> {
    match env::var("PWD") {
        Ok(v) => Ok(v),
        Err(_) => {
            eprintln!("{FAIL_PREFIX}: PWD: unbound variable");
            Err(Exit::Propagate(1))
        }
    }
}

/// `note` mirrors `printf '%-7s %s\n'`: the tag padded to 7, then the text.
pub(crate) fn note(tag: &str, msg: &str) -> String {
    format!("{tag:<7} {msg}")
}

/// `stat_line` mirrors `stat_one`: mode/owner/size/mtime for a path, never
/// content. Unset `$USER` on a missing path crashes like the script's
/// `set -u` expansion; only the crash prefix differs (fixed here instead of
/// `$0`-and-line), since invocation paths differ inherently.
pub(crate) fn stat_line(path: &str, want: &str) -> Result<String, Exit> {
    if fs::metadata(path).is_err() {
        let user = match env::var("USER") {
            Ok(user) => user,
            Err(_) => {
                eprintln!("{FAIL_PREFIX}: USER: unbound variable");
                return Err(Exit::Propagate(1));
            }
        };
        return Ok(note(
            "SKIP",
            &format!("{path} absent or not visible to {user}"),
        ));
    }
    // The script suppresses this stat's stderr (`2>/dev/null`); the two
    // mode checks below inherit it, like the script's unredirected stats.
    let full = match capture("stat", &["-c", "%a %U:%G %s %y", path], true) {
        Captured::SpawnFailed => {
            return Ok(note(
                "SKIP",
                &format!("{path} unreadable (run with read access)"),
            ));
        }
        Captured::Done(code, out) => {
            if code != 0 {
                return Ok(note(
                    "SKIP",
                    &format!("{path} unreadable (run with read access)"),
                ));
            }
            stripped_string(&out)
        }
    };
    let mode = match capture("stat", &["-c", "%a", path], false) {
        Captured::SpawnFailed => String::new(),
        Captured::Done(_, out) => stripped_string(&out),
    };
    if mode == want {
        Ok(note("PASS", &format!("{path} [{full}]")))
    } else {
        let shown = match capture("stat", &["-c", "%a", path], false) {
            Captured::SpawnFailed => String::new(),
            Captured::Done(_, out) => stripped_string(&out),
        };
        Ok(note(
            "WARN",
            &format!("{path} mode is {shown}, want {want} [{full}]"),
        ))
    }
}

/// `first_live_inputs` mirrors the build-metadata glob: the first
/// `soda-live-inputs-*.json` under the isolated releases directory in byte
/// (LC_ALL=C collation) order, or nothing when the glob matches nothing.
/// Like the script's `[ -e ]` guard, entries that do not resolve (dangling
/// symlinks, unreadable paths) are skipped, not reported.
pub(crate) fn first_live_inputs(pwd: &str) -> Option<String> {
    let dir = format!("{pwd}/.artifacts/releases/isolated");
    let entries = fs::read_dir(&dir).ok()?;
    let mut names: Vec<String> = entries
        .filter_map(|entry| entry.ok())
        .map(|entry| entry.file_name().to_string_lossy().into_owned())
        .filter(|name| {
            name.starts_with("soda-live-inputs-")
                && name.ends_with(".json")
                && fs::metadata(format!("{dir}/{name}")).is_ok()
        })
        .collect();
    names.sort();
    names.into_iter().next().map(|name| format!("{dir}/{name}"))
}

pub(crate) fn inventory() -> Result<(), Exit> {
    println!("-- fixture-only media authority (dev scope; regenerable)");
    for name in [
        "artifact.private",
        "passphrase",
        "config.json",
        "trust.json",
    ] {
        println!("{}", stat_line(&format!("{AUTHORITY}/{name}"), "600")?);
    }
    println!("{}", stat_line(&format!("{AUTHORITY}/worker.json"), "600")?);
    println!("-- cloudflared tunnel credentials (dashboard-issued)");
    println!(
        "{}",
        stat_line("/etc/cloudflared/dimensionlab-forgejo-https.token", "640")?
    );
    println!(
        "{}",
        stat_line("/etc/cloudflared/dimensionlab-forgejo-https.id", "640")?
    );
    println!("-- forgejo runner registration (host service config)");
    let home = match env::var("HOME") {
        Ok(home) => home,
        Err(_) => {
            eprintln!("{FAIL_PREFIX}: HOME: unbound variable");
            return Err(Exit::Propagate(1));
        }
    };
    println!(
        "{}",
        stat_line(
            &format!("{home}/containers/forgejo-runner/data/config.yaml"),
            "600"
        )?
    );
    println!(
        "{}",
        stat_line(
            &format!("{home}/containers/forgejo-runner/data/.runner"),
            "600"
        )?
    );
    println!("-- build metadata (public pins only; informational)");
    if let Some(first) = first_live_inputs(&current_pwd()?) {
        println!(
            "{}",
            note(
                "INFO",
                &format!("{first} carries public URLs/hashes only; 0644 is expected")
            )
        );
    }
    println!("-- D3 exposure reminder");
    println!(
        "{}",
        note(
            "INFO",
            "served QCOW2/ISO/rootfs bytes are 0644-or-readable over HTTP;"
        )
    );
    println!(
        "{}",
        note(
            "INFO",
            "guest runtime secrets inside them are UNKNOWN until rotated."
        )
    );
    println!("inventory complete; no values printed, nothing mutated.");
    Ok(())
}
