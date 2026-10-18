//! R07 bounded native overlap trial, Codex side.
//!
//! Ignored by default: it spawns REAL provider children and spends real
//! budget. Run explicitly once per trial with a finite authorized
//! reservation, e.g.
//! `cargo test -p soda-host --lib trial_codex_overlap -- --ignored --nocapture`.
//!
//! The trial stages a read-only copy of the operator's Codex auth into a
//! disposable custody root (0700, removed afterwards) and drives two
//! trivial children concurrently through [`ChildTable`]: it proves
//! simultaneous liveness (overlap), terminal join with exact output,
//! cancellation of a third child and parent-stop retirement with nothing
//! uncertain. Spend is bounded by trivial one-word prompts.

use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use super::super::children::{ChildProfile, ChildSpec, ChildTable};
use super::super::run::{FACTORY_HARNESS_CODEX, ROLE_CODER};

/// Pinned Codex binary for the trial; override with `CODEX_TRIAL_BIN`.
const DEFAULT_CODEX_BIN: &str =
    "/home/vince/.codex/packages/standalone/releases/0.161.0-x86_64-unknown-linux-musl/bin/codex";

fn nanos_now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_nanos() as u64)
        .unwrap_or(0)
}

fn trial_dir(tag: &str) -> Result<PathBuf, String> {
    let home = std::env::var("HOME").map_err(|_| "HOME is not set".to_string())?;
    Ok(PathBuf::from(format!(
        "{home}/.local/state/r07-trial-{}-{tag}",
        nanos_now()
    )))
}

fn write_private(path: &std::path::Path, bytes: &[u8]) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).map_err(|e| e.to_string())?;
        std::fs::set_permissions(parent, std::fs::Permissions::from_mode(0o700))
            .map_err(|e| e.to_string())?;
    }
    std::fs::write(path, bytes).map_err(|e| e.to_string())?;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600))
        .map_err(|e| e.to_string())?;
    Ok(())
}

fn observe_alive(
    table: &mut ChildTable,
    task: &str,
    timeout: Duration,
) -> Result<(), String> {
    let limit = Instant::now() + timeout;
    loop {
        let observation = table.observe(task)?;
        if observation.alive {
            return Ok(());
        }
        if Instant::now() >= limit {
            return Err(format!("{task} never observed alive"));
        }
        std::thread::sleep(Duration::from_millis(50));
    }
}

fn run_trial(scratch: &std::path::Path) -> Result<String, String> {
    let home = std::env::var("HOME").map_err(|_| "HOME is not set".to_string())?;
    let auth_src = PathBuf::from(format!("{home}/.codex/auth.json"));
    if !auth_src.is_file() {
        return Err("trial precondition: ~/.codex/auth.json is absent".to_string());
    }
    let bin = std::env::var("CODEX_TRIAL_BIN").unwrap_or_else(|_| DEFAULT_CODEX_BIN.to_string());
    if !PathBuf::from(&bin).is_file() {
        return Err(format!("trial precondition: codex binary is absent: {bin}"));
    }
    std::fs::create_dir_all(scratch).map_err(|e| e.to_string())?;
    std::fs::set_permissions(scratch, std::fs::Permissions::from_mode(0o700))
        .map_err(|e| e.to_string())?;
    let custody = scratch.join("custody");
    let auth_bytes = std::fs::read(&auth_src).map_err(|e| e.to_string())?;
    write_private(&custody.join(".codex/auth.json"), &auth_bytes)?;
    let jobs = scratch.join("jobs");
    std::fs::create_dir_all(&jobs).map_err(|e| e.to_string())?;

    let run = format!("{:032x}", nanos_now());
    let mut table = ChildTable::open(&run, "r07trial", 2, scratch)?;
    let profile = ChildProfile {
        effort: "low".to_string(),
        contributor: false,
        role: ROLE_CODER.to_string(),
        account_home: custody.to_string_lossy().into_owned(),
    };
    let deadline = Instant::now() + Duration::from_secs(600);
    let child_argv = |word: &str| {
        vec![
            bin.clone(),
            "exec".to_string(),
            "-s".to_string(),
            "read-only".to_string(),
            "--ephemeral".to_string(),
            "-C".to_string(),
            jobs.to_string_lossy().into_owned(),
            "--skip-git-repo-check".to_string(),
            format!("Reply with exactly the single word {word} and nothing else."),
        ]
    };
    let spec = |task: &str, word: &str| ChildSpec {
        task_key: task.to_string(),
        argv: child_argv(word),
        workdir: jobs.clone(),
    };

    let id_a = table.start(&spec("overlap-a", "BEACONA"), &profile, FACTORY_HARNESS_CODEX, deadline)?;
    let id_b = table.start(&spec("overlap-b", "BEACONB"), &profile, FACTORY_HARNESS_CODEX, deadline)?;
    if id_a == id_b {
        return Err("child identities are not distinct".to_string());
    }
    let wait_start = Instant::now();
    let overlap_limit = Instant::now() + Duration::from_secs(60);
    let overlapped_ms = loop {
        let a = table.observe("overlap-a")?;
        let b = table.observe("overlap-b")?;
        if a.alive && b.alive {
            break wait_start.elapsed().as_millis();
        }
        if Instant::now() >= overlap_limit {
            return Err("children never observed alive together".to_string());
        }
        std::thread::sleep(Duration::from_millis(50));
    };

    let code_a = table.join("overlap-a", Duration::from_secs(300))?;
    let code_b = table.join("overlap-b", Duration::from_secs(300))?;
    let tail_a = table.observe("overlap-a")?.output_tail;
    let tail_b = table.observe("overlap-b")?.output_tail;
    let text_a = String::from_utf8_lossy(&tail_a).into_owned();
    let text_b = String::from_utf8_lossy(&tail_b).into_owned();
    if code_a != 0 || !text_a.contains("BEACONA") {
        return Err(format!("child A failed: exit={code_a} tail={text_a:?}"));
    }
    if code_b != 0 || !text_b.contains("BEACONB") {
        return Err(format!("child B failed: exit={code_b} tail={text_b:?}"));
    }

    let id_c = table.start(&spec("cancel-c", "BEACONC"), &profile, FACTORY_HARNESS_CODEX, deadline)?;
    let _ = observe_alive(&mut table, "cancel-c", Duration::from_secs(60));
    table.cancel("cancel-c")?;
    let after = table.observe("cancel-c")?;
    if after.alive {
        return Err("cancelled child is still alive".to_string());
    }
    let report = table.stop_all();
    if !report.uncertain.is_empty() {
        return Err(format!("uncertain remainder: {:?}", report.uncertain));
    }
    if table.active_count() != 0 {
        return Err("children remain active after parent stop".to_string());
    }
    Ok(format!(
        "bin={bin} overlap_ms={overlapped_ms} exit_a={code_a} exit_b={code_b} cancelled={} retired={}",
        &id_c[..8],
        report.retired.len(),
    ))
}

#[test]
#[ignore = "spends real provider budget; run explicitly for a bounded R07 trial"]
fn trial_codex_overlap_and_cancel() {
    let scratch = trial_dir("codex").expect("scratch path");
    let result = run_trial(&scratch);
    let _ = std::fs::remove_dir_all(&scratch);
    let evidence = result.expect("trial failed");
    println!("R07-CODEX-TRIAL {evidence}");
}
