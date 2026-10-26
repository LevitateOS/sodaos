//! Operator command for provider CLI freshness (UP-20261026-v1, lane A).
//!
//! ```text
//! soda-provider-update check [--provider codex|muse] [--json]
//! soda-provider-update refresh [--provider codex|muse] [--json]
//! soda-provider-update status [--json]
//! ```
//!
//! Exit codes: 0 success/current, 1 operational error, 2 known-stale or failed
//! update blocks new provider work (existing work keeps exact bytes).

use soda_provider_tools::store::{Freshness, ReleaseIdentity, Selection, Status};
use soda_provider_tools::{config, discover, refresh, resolve, store::Provider};

fn usage() -> ! {
    eprintln!("usage: soda-provider-update check|refresh|status [--provider codex|muse] [--json]");
    std::process::exit(1);
}

fn parse_args(args: &[String]) -> (String, Option<Provider>, bool) {
    let mut command = String::new();
    let mut provider: Option<Provider> = None;
    let mut provider_seen = false;
    let mut json = false;
    let mut iter = args.iter().peekable();
    while let Some(arg) = iter.next() {
        if arg == "--provider" {
            let name = iter.next().unwrap_or_else(|| usage());
            provider_seen = true;
            match Provider::parse(name) {
                Some(p) => provider = Some(p),
                None => {
                    eprintln!("unknown provider: {name} (want codex|muse)");
                    std::process::exit(1);
                }
            }
        } else if arg == "--json" {
            json = true;
        } else if arg.starts_with("--") {
            eprintln!("unknown flag: {arg}");
            usage();
        } else if command.is_empty() {
            command = arg.clone();
        } else {
            eprintln!("unexpected argument: {arg}");
            usage();
        }
    }
    if command.is_empty() {
        usage();
    }
    let _ = provider_seen;
    (command, provider, json)
}

#[derive(serde::Serialize)]
struct CheckReport {
    provider: String,
    version: Option<String>,
    digest: Option<String>,
    size: Option<u64>,
    artifact_url: Option<String>,
    checked_at_secs: Option<u64>,
    admitted_at_secs: Option<u64>,
    freshness: Freshness,
    detail: String,
}

#[derive(serde::Serialize)]
struct RefreshReport {
    provider: String,
    selection: Option<Selection>,
    outcome: String,
    error: Option<String>,
}

fn providers_only(one: Option<Provider>) -> Vec<Provider> {
    match one {
        Some(p) => vec![p],
        None => vec![Provider::Codex, Provider::Muse],
    }
}

/// Exit mapping: 2 when a known-newer/unadmitted update blocks new work,
/// 1 for operational failures or unresolved freshness, 0 when current.
fn exit_for_blocked(blocked: bool, failed: bool) -> ! {
    if blocked {
        std::process::exit(2);
    }
    if failed {
        std::process::exit(1);
    }
    std::process::exit(0);
}

fn check_reports(only: Option<Provider>) -> (Vec<CheckReport>, bool, bool) {
    let root = config::store_root();
    let ttl = config::check_ttl();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    let mut blocked = false;
    let mut failed = false;
    let mut reports = Vec::new();
    for provider in providers_only(only) {
        let selection = store_read(&root, provider, &mut failed);
        let latest: Option<ReleaseIdentity> = match discover::resolve_upstream(provider) {
            Ok(latest) => Some(latest),
            Err(e) => {
                failed = true;
                reports.push(CheckReport {
                    provider: provider.as_str().to_string(),
                    version: selection
                        .as_ref()
                        .map(|s: &Selection| s.release.version.clone()),
                    digest: selection.as_ref().map(|s| s.release.digest.clone()),
                    size: selection.as_ref().map(|s| s.release.size),
                    artifact_url: selection.as_ref().map(|s| s.release.artifact_url.clone()),
                    checked_at_secs: selection.as_ref().map(|s| s.checked_at_secs),
                    admitted_at_secs: selection.as_ref().map(|s| s.admitted_at_secs),
                    freshness: soda_provider_tools::store::evaluate(
                        selection.as_ref(),
                        None,
                        now,
                        ttl.as_secs(),
                    ),
                    detail: format!("upstream check failed: {e}"),
                });
                continue;
            }
        };
        let freshness = soda_provider_tools::store::evaluate(
            selection.as_ref(),
            latest.as_ref(),
            now,
            ttl.as_secs(),
        );
        if matches!(freshness, Freshness::Stale { .. }) {
            blocked = true;
        }
        if matches!(
            freshness,
            Freshness::Unknown { .. } | Freshness::UpdateFailed { .. }
        ) {
            failed = true;
        }
        let latest = latest.expect("checked Ok above");
        reports.push(CheckReport {
            provider: provider.as_str().to_string(),
            detail: match &freshness {
                Freshness::Current => format!("current: {}", latest.version),
                Freshness::Stale { latest } => format!("stale: upstream is {latest}"),
                Freshness::Unknown { reason } => format!("unknown: {reason}"),
                Freshness::UpdateFailed { reason } => format!("update failed: {reason}"),
            },
            version: Some(latest.version.clone()),
            digest: Some(latest.digest.clone()),
            size: Some(latest.size),
            artifact_url: Some(latest.artifact_url.clone()),
            checked_at_secs: selection.as_ref().map(|s| s.checked_at_secs),
            admitted_at_secs: selection.as_ref().map(|s| s.admitted_at_secs),
            freshness,
        });
    }
    (reports, blocked, failed)
}

fn store_read(root: &std::path::Path, provider: Provider, failed: &mut bool) -> Option<Selection> {
    match soda_provider_tools::store::read_selection(root, provider) {
        Ok(sel) => sel,
        Err(e) => {
            *failed = true;
            eprintln!("{}: cannot read selection: {e}", provider.as_str());
            None
        }
    }
}

fn print_check(reports: &[CheckReport], json: bool) {
    if json {
        println!(
            "{}",
            serde_json::to_string_pretty(&reports).unwrap_or_else(|_| "[]".to_string())
        );
        return;
    }
    for report in reports {
        let state = match &report.freshness {
            Freshness::Current => "current",
            Freshness::Stale { .. } => "STALE",
            Freshness::Unknown { .. } => "unknown",
            Freshness::UpdateFailed { .. } => "update-failed",
        };
        println!(
            "{}: {state} version={} digest={} {}",
            report.provider,
            report.version.as_deref().unwrap_or("-"),
            report
                .digest
                .as_deref()
                .map(|d| d.get(..12).unwrap_or(d))
                .unwrap_or("-"),
            report.detail
        );
    }
}

fn run_refresh(only: Option<Provider>, json: bool) -> ! {
    let mut blocked = false;
    let mut failed = false;
    let mut reports = Vec::new();
    for provider in providers_only(only) {
        match refresh::refresh(provider) {
            Ok(selection) => {
                let outcome = match &selection.freshness {
                    Freshness::Current => "current".to_string(),
                    Freshness::Stale { latest } => {
                        blocked = true;
                        format!("stale: upstream is {latest}")
                    }
                    Freshness::Unknown { reason } => {
                        failed = true;
                        format!("unknown: {reason}")
                    }
                    Freshness::UpdateFailed { reason } => {
                        failed = true;
                        format!("update-failed: {reason}")
                    }
                };
                reports.push(RefreshReport {
                    provider: provider.as_str().to_string(),
                    selection: Some(selection),
                    outcome,
                    error: None,
                });
            }
            Err(e) => {
                if e.blocks_new_work() {
                    blocked = true;
                } else {
                    failed = true;
                }
                reports.push(RefreshReport {
                    provider: provider.as_str().to_string(),
                    selection: None,
                    outcome: "error".to_string(),
                    error: Some(e.to_string()),
                });
            }
        }
    }
    if json {
        println!(
            "{}",
            serde_json::to_string_pretty(&reports).unwrap_or_else(|_| "[]".to_string())
        );
    } else {
        for report in &reports {
            match (&report.selection, &report.error) {
                (Some(sel), _) => println!(
                    "{}: {} version={} digest={}",
                    report.provider,
                    report.outcome,
                    sel.release.version,
                    sel.release.digest.get(..12).unwrap_or(&sel.release.digest)
                ),
                (None, Some(err)) => println!("{}: error: {err}", report.provider),
                (None, None) => println!("{}: {}", report.provider, report.outcome),
            }
        }
    }
    exit_for_blocked(blocked, failed);
}

fn run_status(only: Option<Provider>, json: bool) -> ! {
    let root = config::store_root();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    let mut entries = Vec::new();
    for provider in providers_only(only) {
        match soda_provider_tools::store::read_selection(&root, provider) {
            Ok(Some(sel)) => entries.push(sel),
            Ok(None) => {}
            Err(e) => {
                eprintln!("{}: cannot read selection: {e}", provider.as_str());
                std::process::exit(1);
            }
        }
    }
    let status = Status {
        root: root.clone(),
        now_secs: now,
        entries,
    };
    if json {
        println!(
            "{}",
            serde_json::to_string_pretty(&status).unwrap_or_else(|_| "{}".to_string())
        );
    } else {
        println!("root: {}", status.root.display());
        if status.entries.is_empty() {
            println!("no provider bundles admitted yet");
        }
        for sel in &status.entries {
            let exe = resolve::resolve_existing(sel.provider, &sel.release.digest)
                .map(|p| p.display().to_string())
                .unwrap_or_else(|_| "(bundle damaged)".to_string());
            let freshness = match &sel.freshness {
                Freshness::Current => "current".to_string(),
                Freshness::Stale { latest } => format!("STALE upstream={latest}"),
                Freshness::Unknown { reason } => format!("unknown: {reason}"),
                Freshness::UpdateFailed { reason } => format!("update-failed: {reason}"),
            };
            println!(
                "{}: {} version={} digest={} checked={} admitted={} exe={}",
                sel.provider.as_str(),
                freshness,
                sel.release.version,
                sel.release.digest.get(..12).unwrap_or(&sel.release.digest),
                sel.checked_at_secs,
                sel.admitted_at_secs,
                exe
            );
        }
    }
    std::process::exit(0);
}

fn main() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let (command, only, json) = parse_args(&args);
    match command.as_str() {
        "check" => {
            let (reports, blocked, failed) = check_reports(only);
            print_check(&reports, json);
            exit_for_blocked(blocked, failed);
        }
        "refresh" => run_refresh(only, json),
        "status" => run_status(only, json),
        _ => usage(),
    }
}
