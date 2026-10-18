//! R07 bounded native overlap trials, one per harness.
//!
//! Ignored by default: they spawn REAL provider children and spend real
//! budget. Run explicitly once per trial with a finite authorized
//! reservation, e.g.
//! `cargo test -p soda-host --lib trial_codex_overlap -- --ignored --nocapture`.
//!
//! Each trial stages a read-only copy of the operator's harness auth into a
//! disposable custody root (0700, removed afterwards) and drives two
//! trivial children concurrently through [`ChildTable`]: it proves
//! simultaneous liveness (overlap), terminal join with exact output,
//! cancellation of a third child and parent-stop retirement with nothing
//! uncertain. Spend is bounded by trivial one-word prompts. The Muse side
//! resolves its binary as newest `muse-bin-*` per the owner latest direction.

use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use super::super::children::{ChildProfile, ChildSpec, ChildTable};
use super::super::run::{FACTORY_HARNESS_CODEX, FACTORY_HARNESS_MUSE, ROLE_CODER};
use crate::factory::{
    TASK_EVENT_CHILD_REPORTED, TASK_EVENT_CHILD_START_INTENT, TASK_EVENT_TASK_COMMITTED,
    TaskRunner,
};
use crate::sha256;

/// Pinned Codex binary for the trial; override with `CODEX_TRIAL_BIN`.
const DEFAULT_CODEX_BIN: &str =
    "/home/vince/.codex/packages/standalone/releases/0.161.0-x86_64-unknown-linux-musl/bin/codex";

struct TrialHarness {
    /// Evidence tag and scratch tag.
    name: &'static str,
    family: &'static str,
    bin_env: &'static str,
    auth_rel: &'static str,
    effort: &'static str,
    contributor: bool,
}

fn resolve_bin(h: &TrialHarness) -> Result<String, String> {
    if let Ok(bin) = std::env::var(h.bin_env) {
        if PathBuf::from(&bin).is_file() {
            return Ok(bin);
        }
        return Err(format!("trial binary override is absent: {bin}"));
    }
    if h.family == FACTORY_HARNESS_CODEX {
        if PathBuf::from(DEFAULT_CODEX_BIN).is_file() {
            return Ok(DEFAULT_CODEX_BIN.to_string());
        }
        return Err("trial precondition: pinned codex binary is absent".to_string());
    }
    let home = std::env::var("HOME").map_err(|_| "HOME is not set".to_string())?;
    let dir = PathBuf::from(format!("{home}/.local/bin"));
    let mut newest: Option<String> = None;
    let entries = std::fs::read_dir(&dir).map_err(|e| e.to_string())?;
    for entry in entries {
        let entry = entry.map_err(|e| e.to_string())?;
        let name = entry.file_name().to_string_lossy().into_owned();
        if name.starts_with("muse-bin-") && entry.path().is_file() {
            if newest.as_ref().is_none_or(|top: &String| name > *top) {
                newest = Some(entry.path().to_string_lossy().into_owned());
            }
        }
    }
    newest.ok_or_else(|| "trial precondition: no muse-bin-* binary is installed".to_string())
}

fn nanos_now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_nanos() as u64)
        .unwrap_or(0)
}

fn trial_dir(home: &str, tag: &str) -> PathBuf {
    PathBuf::from(format!("{home}/.local/state/r07-trial-{}-{tag}", nanos_now()))
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

fn run_trial(h: &TrialHarness, scratch: &std::path::Path) -> Result<String, String> {
    let home = std::env::var("HOME").map_err(|_| "HOME is not set".to_string())?;
    let auth_src = PathBuf::from(format!("{home}/{}", h.auth_rel));
    if !auth_src.is_file() {
        return Err(format!("trial precondition: ~/{0} is absent", h.auth_rel));
    }
    let bin = resolve_bin(h)?;
    std::fs::create_dir_all(scratch).map_err(|e| e.to_string())?;
    std::fs::set_permissions(scratch, std::fs::Permissions::from_mode(0o700))
        .map_err(|e| e.to_string())?;
    let custody = scratch.join("custody");
    let auth_bytes = std::fs::read(&auth_src).map_err(|e| e.to_string())?;
    write_private(&custody.join(&h.auth_rel), &auth_bytes)?;
    let jobs = scratch.join("jobs");
    std::fs::create_dir_all(&jobs).map_err(|e| e.to_string())?;

    let run = format!("{:032x}", nanos_now());
    let mut table = ChildTable::open(&run, "r07trial", 2, scratch)?;
    let profile = ChildProfile {
        effort: h.effort.to_string(),
        contributor: h.contributor,
        role: ROLE_CODER.to_string(),
        account_home: custody.to_string_lossy().into_owned(),
    };
    let deadline = Instant::now() + Duration::from_secs(600);
    let jobs_str = jobs.to_string_lossy().into_owned();
    let child_argv = |word: &str| {
        if h.family == FACTORY_HARNESS_CODEX {
            vec![
                bin.clone(),
                "exec".to_string(),
                "-s".to_string(),
                "read-only".to_string(),
                "--ephemeral".to_string(),
                "-C".to_string(),
                jobs_str.clone(),
                "--skip-git-repo-check".to_string(),
                format!("Reply with exactly the single word {word} and nothing else."),
            ]
        } else {
            vec![
                bin.clone(),
                "exec".to_string(),
                "--reasoning-effort".to_string(),
                "minimal".to_string(),
                format!("Reply with exactly the single word {word} and nothing else."),
            ]
        }
    };
    let stem = if h.family == FACTORY_HARNESS_CODEX { "BEACON" } else { "MBEACON" };
    let spec = |task: &str, word: &str| ChildSpec {
        task_key: task.to_string(),
        argv: child_argv(word),
        workdir: jobs.clone(),
    };

    let id_a = table.start(
        &spec("overlap-a", &format!("{stem}A")),
        &profile,
        h.family,
        deadline,
    )?;
    let id_b = table.start(
        &spec("overlap-b", &format!("{stem}B")),
        &profile,
        h.family,
        deadline,
    )?;
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
    if code_a != 0 || !text_a.contains(&format!("{stem}A")) {
        return Err(format!("child A failed: exit={code_a} tail={text_a:?}"));
    }
    if code_b != 0 || !text_b.contains(&format!("{stem}B")) {
        return Err(format!("child B failed: exit={code_b} tail={text_b:?}"));
    }

    let id_c = table.start(
        &spec("cancel-c", &format!("{stem}C")),
        &profile,
        h.family,
        deadline,
    )?;
    let limit = Instant::now() + Duration::from_secs(60);
    loop {
        let observation = table.observe("cancel-c")?;
        if observation.alive || Instant::now() >= limit {
            break;
        }
        std::thread::sleep(Duration::from_millis(50));
    }
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

fn drive(h: &TrialHarness, tag: &str) {
    let home = std::env::var("HOME").expect("HOME");
    let scratch = trial_dir(&home, tag);
    let result = run_trial(h, &scratch);
    let _ = std::fs::remove_dir_all(&scratch);
    let evidence = result.expect("trial failed");
    println!("R07-{}-TRIAL {evidence}", h.name.to_uppercase());
}

const CODEX: TrialHarness = TrialHarness {
    name: "codex",
    family: FACTORY_HARNESS_CODEX,
    bin_env: "CODEX_TRIAL_BIN",
    auth_rel: ".codex/auth.json",
    effort: "low",
    contributor: false,
};

const MUSE: TrialHarness = TrialHarness {
    name: "muse",
    family: FACTORY_HARNESS_MUSE,
    bin_env: "MUSE_TRIAL_BIN",
    auth_rel: ".config/muse/auth.json",
    effort: "low",
    contributor: false,
};

#[test]
#[ignore = "spends real provider budget; run explicitly for a bounded R07 trial"]
fn trial_codex_overlap_and_cancel() {
    drive(&CODEX, "codex");
}

#[test]
#[ignore = "spends real provider budget; run explicitly for a bounded R07 trial"]
fn trial_muse_overlap_and_cancel() {
    drive(&MUSE, "muse");
}

fn output_digest(tail: &[u8]) -> String {
    sha256::hex_lower(&sha256::digest(tail))
}

/// Dependent successor through the real [`TaskRunner`]: the second child
/// starts only after the first reaches its terminal receipt, and the
/// journal records intent/started/reported/committed in order for both.
fn run_successor_trial(scratch: &std::path::Path) -> Result<String, String> {
    let home = std::env::var("HOME").map_err(|_| "HOME is not set".to_string())?;
    let auth_src = PathBuf::from(format!("{home}/{}", CODEX.auth_rel));
    if !auth_src.is_file() {
        return Err(format!("trial precondition: ~/{0} is absent", CODEX.auth_rel));
    }
    let bin = resolve_bin(&CODEX)?;
    std::fs::create_dir_all(scratch).map_err(|e| e.to_string())?;
    std::fs::set_permissions(scratch, std::fs::Permissions::from_mode(0o700))
        .map_err(|e| e.to_string())?;
    let custody = scratch.join("custody");
    let auth_bytes = std::fs::read(&auth_src).map_err(|e| e.to_string())?;
    write_private(&custody.join(CODEX.auth_rel), &auth_bytes)?;
    let jobs = scratch.join("jobs");
    std::fs::create_dir_all(&jobs).map_err(|e| e.to_string())?;
    let jobs_str = jobs.to_string_lossy().into_owned();

    let run = format!("{:032x}", nanos_now());
    let plan = format!("{:064x}", nanos_now());
    let profile = ChildProfile {
        effort: "low".to_string(),
        contributor: false,
        role: ROLE_CODER.to_string(),
        account_home: custody.to_string_lossy().into_owned(),
    };
    let mut runner =
        TaskRunner::open(&run, "r07trial", &plan, FACTORY_HARNESS_CODEX, &profile, 2, scratch)?;
    let deadline = Instant::now() + Duration::from_secs(600);
    let spec = |task: &str, word: &str| ChildSpec {
        task_key: task.to_string(),
        argv: vec![
            bin.clone(),
            "exec".to_string(),
            "-s".to_string(),
            "read-only".to_string(),
            "--ephemeral".to_string(),
            "-C".to_string(),
            jobs_str.clone(),
            "--skip-git-repo-check".to_string(),
            format!("Reply with exactly the single word {word} and nothing else."),
        ],
        workdir: jobs.clone(),
    };
    let settle = |runner: &mut TaskRunner,
                  task: &str,
                  word: &str|
     -> Result<String, String> {
        let code = runner.join_task(task, Duration::from_secs(300))?;
        let tail = runner.observe_task(task)?.output_tail;
        let text = String::from_utf8_lossy(&tail).into_owned();
        if code != 0 || !text.contains(word) {
            return Err(format!("{task} failed: exit={code} tail={text:?}"));
        }
        let digest = output_digest(&tail);
        let reported = runner.report_task(task, &digest)?;
        if reported != 0 {
            return Err(format!("{task} reported nonzero: {reported}"));
        }
        runner.complete_task(task, None, None)?;
        Ok(digest)
    };

    let first_id = runner.start_task(&spec("dep-first", "SBEACONA"), deadline)?;
    if runner.active_tasks() != vec!["dep-first".to_string()] {
        return Err(format!("active = {:?}", runner.active_tasks()));
    }
    settle(&mut runner, "dep-first", "SBEACONA")?;
    let second_id = runner.start_task(&spec("dep-second", "SBEACONB"), deadline)?;
    let journal = runner.journal_snapshot();
    let committed = journal
        .iter()
        .find(|e| e.event_type == TASK_EVENT_TASK_COMMITTED && e.task_key == "dep-first")
        .map(|e| e.sequence)
        .ok_or("missing dep-first committed receipt")?;
    let intent = journal
        .iter()
        .find(|e| e.event_type == TASK_EVENT_CHILD_START_INTENT && e.task_key == "dep-second")
        .map(|e| e.sequence)
        .ok_or("missing dep-second start intent")?;
    if !(committed < intent) {
        return Err(format!("successor started before predecessor receipt: {committed} !< {intent}"));
    }
    settle(&mut runner, "dep-second", "SBEACONB")?;
    if !runner.active_tasks().is_empty() {
        return Err(format!("tasks remain active: {:?}", runner.active_tasks()));
    }
    let reported = journal
        .iter()
        .filter(|e| e.event_type == TASK_EVENT_CHILD_REPORTED)
        .count();
    let report = runner.stop("dep-second", "trial complete")?;
    if !report.uncertain.is_empty() {
        return Err(format!("uncertain remainder: {:?}", report.uncertain));
    }
    Ok(format!(
        "first={} second={} committed_seq={committed} intent_seq={intent} reported={reported} retired={}",
        &first_id[..8],
        &second_id[..8],
        report.retired.len(),
    ))
}

#[test]
#[ignore = "spends real provider budget; run explicitly for a bounded R07 trial"]
fn trial_codex_dependent_successor() {
    let home = std::env::var("HOME").expect("HOME");
    let scratch = trial_dir(&home, "successor");
    let result = run_successor_trial(&scratch);
    let _ = std::fs::remove_dir_all(&scratch);
    let evidence = result.expect("trial failed");
    println!("R07-CODEX-SUCCESSOR {evidence}");
}

/// Abandoned-child run: the table is forgotten while the child is alive —
/// no stop, cancel or join is ever issued, which is exactly what the child
/// observes when its parent dies (own process group, null stdin, owned log
/// handles). The trial proves exact output still lands in the child's log.
/// Retirement after death is therefore observed downstream (see the R11
/// crash-replay proof), never imposed by kill: no prctl, death pipe or
/// Drop guard exists in this layer.
fn run_abandon_trial(scratch: &std::path::Path) -> Result<String, String> {
    let home = std::env::var("HOME").map_err(|_| "HOME is not set".to_string())?;
    let auth_src = PathBuf::from(format!("{home}/{}", CODEX.auth_rel));
    if !auth_src.is_file() {
        return Err(format!("trial precondition: ~/{0} is absent", CODEX.auth_rel));
    }
    let bin = resolve_bin(&CODEX)?;
    std::fs::create_dir_all(scratch).map_err(|e| e.to_string())?;
    std::fs::set_permissions(scratch, std::fs::Permissions::from_mode(0o700))
        .map_err(|e| e.to_string())?;
    let custody = scratch.join("custody");
    let auth_bytes = std::fs::read(&auth_src).map_err(|e| e.to_string())?;
    write_private(&custody.join(CODEX.auth_rel), &auth_bytes)?;
    let jobs = scratch.join("jobs");
    std::fs::create_dir_all(&jobs).map_err(|e| e.to_string())?;
    let jobs_str = jobs.to_string_lossy().into_owned();

    let run = format!("{:032x}", nanos_now());
    let mut table = ChildTable::open(&run, "r07trial", 2, scratch)?;
    let profile = ChildProfile {
        effort: "low".to_string(),
        contributor: false,
        role: ROLE_CODER.to_string(),
        account_home: custody.to_string_lossy().into_owned(),
    };
    let deadline = Instant::now() + Duration::from_secs(600);
    let id = table.start(
        &ChildSpec {
            task_key: "abandon-a".to_string(),
            argv: vec![
                bin.clone(),
                "exec".to_string(),
                "-s".to_string(),
                "read-only".to_string(),
                "--ephemeral".to_string(),
                "-C".to_string(),
                jobs_str,
                "--skip-git-repo-check".to_string(),
                "Reply with exactly the single word ABEACONA and nothing else.".to_string(),
            ],
            workdir: jobs.clone(),
        },
        &profile,
        FACTORY_HARNESS_CODEX,
        deadline,
    )?;
    if !table.observe("abandon-a")?.alive {
        return Err("child exited before abandon".to_string());
    }
    std::mem::forget(table);
    let log = scratch.join(format!("{id}.log"));
    let limit = Instant::now() + Duration::from_secs(300);
    loop {
        let bytes =
            std::fs::read(&log).map_err(|e| format!("orphan log unreadable: {e}"))?;
        if String::from_utf8_lossy(&bytes).contains("ABEACONA") {
            break;
        }
        if Instant::now() >= limit {
            return Err("abandoned child produced no exact output".to_string());
        }
        std::thread::sleep(Duration::from_millis(200));
    }
    Ok(format!("bin={bin} abandoned={} output=ABEACONA retire=never-issued", &id[..8]))
}

#[test]
#[ignore = "spends real provider budget; run explicitly for a bounded R07 trial"]
fn trial_codex_abandoned_child_completes() {
    let home = std::env::var("HOME").expect("HOME");
    let scratch = trial_dir(&home, "abandon");
    let result = run_abandon_trial(&scratch);
    let _ = std::fs::remove_dir_all(&scratch);
    let evidence = result.expect("trial failed");
    println!("R07-CODEX-ABANDON {evidence}");
}
