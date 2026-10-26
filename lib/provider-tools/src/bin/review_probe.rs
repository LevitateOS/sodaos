//! UP-REVIEW-01 regression driver (normal bin, direct execution, real output).
//!
//! Replays the review probe against the fixed library in isolated scratch
//! stores under `$HOME/.provider-review-scratch` (outside the repo):
//! - R1: a stored `Stale` selection survives a failed refresh untouched and
//!   `resolve_new` blocks new work instead of launching the known-old bundle.
//! - R2: `cleanup` fails `Uncertain` before any deletion when refs or
//!   selections are corrupt/unreadable, preserving possibly-active bundles;
//!   concurrent refresh/acquire/cleanup stay bounded and lossless.
//! - R3: `resolve_existing` (and new-work resolution) demand the admission
//!   record plus per-file digests/permissions/paths, refusing bare
//!   hash-shaped directories, symlinks, tampered bytes and missing companions.
//! No unit tests, no framework, no model calls, no credentials.

use soda_provider_tools::{
    refresh, resolve,
    store::{self, Freshness, Provider, ReleaseIdentity, Selection},
};
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};

static FAILURES: AtomicU64 = AtomicU64::new(0);

fn expect(cond: bool, what: &str) {
    println!("{}: {what}", if cond { "ok" } else { "MISMATCH" });
    if !cond {
        FAILURES.fetch_add(1, Ordering::SeqCst);
    }
}

fn scratch_base() -> PathBuf {
    let home = std::env::var("HOME").unwrap_or_else(|_| "/home/vince".to_string());
    PathBuf::from(home).join(".provider-review-scratch")
}

fn fresh_root(name: &str) -> PathBuf {
    let root = scratch_base().join(name);
    let _ = std::fs::remove_dir_all(&root);
    std::fs::create_dir_all(&root).expect("scratch root");
    std::env::set_var("SODA_PROVIDERS_ROOT", &root);
    root
}

fn sha256_hex(bytes: &[u8]) -> String {
    use sha2::Digest as _;
    let mut h = sha2::Sha256::new();
    h.update(bytes);
    h.finalize().iter().map(|b| format!("{b:02x}")).collect()
}

fn chmod_exec(path: &Path) {
    use std::os::unix::fs::PermissionsExt as _;
    let mut perms = std::fs::metadata(path).expect("stat").permissions();
    perms.set_mode(0o755);
    std::fs::set_permissions(path, perms).expect("chmod");
}

/// Build a fully admitted bundle WITHOUT network: real files, real admission
/// record, real digest inventory. Returns the identity plus exe bytes (so the
/// caller can tamper and restore).
fn make_bundle(root: &Path, provider: Provider, version: &str, companion: bool) -> (ReleaseIdentity, Vec<u8>) {
    let content = format!("#!/bin/sh\necho review-{provider:?}-{version}\n").into_bytes();
    let digest = sha256_hex(&content);
    let bundle = root
        .join("bundles")
        .join(provider.as_str())
        .join(&digest);
    let bin = bundle.join("bin");
    std::fs::create_dir_all(&bin).expect("bin dir");
    let exe_name = provider.executable_relpath().rsplit('/').next().unwrap();
    let exe = bin.join(exe_name);
    std::fs::write(&exe, &content).expect("exe");
    chmod_exec(&exe);
    if companion && provider == Provider::Codex {
        let comp = bin.join("codex-code-mode-host");
        std::fs::write(&comp, b"#!/bin/sh\necho companion\n").expect("companion");
        chmod_exec(&comp);
    }
    let release = ReleaseIdentity {
        provider,
        version: version.to_string(),
        digest,
        size: content.len() as u64,
        artifact_url: "https://example.invalid/review".to_string(),
    };
    store::write_release_record(&bundle, &release).expect("release record");
    store::record_inventory(&bundle, &release).expect("inventory");
    (release, content)
}

fn write_selection(root: &Path, release: &ReleaseIdentity, checked: u64, admitted: u64, freshness: Freshness) {
    store::write_selection(
        root,
        &Selection {
            provider: release.provider,
            release: release.clone(),
            checked_at_secs: checked,
            admitted_at_secs: admitted,
            freshness,
        },
    )
    .expect("write selection");
}

fn now_secs() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

/// R1: failed refresh preserves stored Stale; resolve_new blocks.
fn scenario_r1() {
    println!("--- R1 stale preservation ---");
    let root = fresh_root("r1");
    std::env::set_var("SODA_PROVIDERS_CHANNEL", "review-invalid-channel");
    std::env::set_var("SODA_PROVIDERS_CHECK_TTL_SECS", "300");
    let now = now_secs();
    let (release, _) = make_bundle(&root, Provider::Muse, "review-old", false);
    write_selection(
        &root,
        &release,
        0,
        now.saturating_sub(1000),
        Freshness::Stale {
            latest: "review-known-newer".to_string(),
        },
    );
    let refreshed = refresh::refresh(Provider::Muse).expect("failed check returns previous");
    expect(
        matches!(&refreshed.freshness, Freshness::Stale { latest } if latest == "review-known-newer"),
        "failed refresh preserves Stale (not downgraded to UpdateFailed)",
    );
    expect(refreshed.checked_at_secs == 0, "failed refresh leaves checked_at untouched");
    let on_disk = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("selection present");
    expect(
        matches!(&on_disk.freshness, Freshness::Stale { .. }) && on_disk.checked_at_secs == 0,
        "disk selection still Stale with original timestamps",
    );
    let launched = resolve::resolve_new(Provider::Muse).is_ok();
    println!("known_stale_launch_allowed_after_upstream_failure={launched}");
    expect(!launched, "resolve_new blocks known-stale after failed check");
    // Direct refresh error class: must not block-or-fail, must preserve.
    let blocked = resolve::resolve_new(Provider::Muse).unwrap_err();
    expect(blocked.blocks_new_work(), "resolve_new error blocks new work");
}

/// R3: admission record + digests enforced; hash-shaped dirs refused.
fn scenario_r3() {
    println!("--- R3 admission verification ---");
    let root = fresh_root("r3");
    let (release, exe_bytes) = make_bundle(&root, Provider::Muse, "review-1", false);
    let digest = release.digest.clone();
    expect(
        resolve::resolve_existing(Provider::Muse, &digest).is_ok(),
        "admitted bundle resolves",
    );
    // Tamper with one byte: verification must fail; restore: resolves again.
    let exe = root
        .join("bundles/muse")
        .join(&digest)
        .join("bin/muse");
    let mut tampered = exe_bytes.clone();
    tampered[0] = b'#';
    tampered[1] = b'!';
    tampered.push(b'\n');
    std::fs::write(&exe, &tampered).expect("tamper");
    expect(
        resolve::resolve_existing(Provider::Muse, &digest).is_err(),
        "tampered bundle denied",
    );
    std::fs::write(&exe, &exe_bytes).expect("restore");
    chmod_exec(&exe);
    expect(
        resolve::resolve_existing(Provider::Muse, &digest).is_ok(),
        "restored bundle resolves",
    );
    // Bare hash-shaped directory with fabricated executable, no record.
    let other = sha256_hex(b"unadmitted-fixture");
    let fake_bin = root.join("bundles/muse").join(&other).join("bin");
    std::fs::create_dir_all(&fake_bin).expect("fake bin");
    std::fs::write(fake_bin.join("muse"), b"#!/bin/sh\necho unadmitted-fixture\n").expect("fake exe");
    chmod_exec(&fake_bin.join("muse"));
    let accepted = resolve::resolve_existing(Provider::Muse, &other).is_ok();
    println!("bundle_without_admission_record_accepted={accepted}");
    expect(!accepted, "hash-shaped dir without admission record denied");
    // Symlink authority refused.
    let link_digest = sha256_hex(b"link-fixture");
    std::os::unix::fs::symlink(
        root.join("bundles/muse").join(&digest),
        root.join("bundles/muse").join(&link_digest),
    )
    .expect("symlink");
    expect(
        resolve::resolve_existing(Provider::Muse, &link_digest).is_err(),
        "symlinked bundle dir denied",
    );
    std::fs::remove_file(root.join("bundles/muse").join(&link_digest)).expect("unlink");
    // Codex companion required.
    let (codex, _) = make_bundle(&root, Provider::Codex, "review-1", false);
    expect(
        resolve::resolve_existing(Provider::Codex, &codex.digest).is_err(),
        "codex bundle without companion denied",
    );
    let comp = root
        .join("bundles/codex")
        .join(&codex.digest)
        .join("bin/codex-code-mode-host");
    std::fs::write(&comp, b"#!/bin/sh\necho companion\n").expect("companion");
    chmod_exec(&comp);
    store::record_inventory(
        &root.join("bundles/codex").join(&codex.digest),
        &codex,
    )
    .expect("re-inventory");
    expect(
        resolve::resolve_existing(Provider::Codex, &codex.digest).is_ok(),
        "codex bundle with companion resolves",
    );
}

/// R2: corrupt refs/selections fail cleanup Uncertain with nothing deleted.
fn scenario_r2() {
    println!("--- R2 cleanup fail-uncertain ---");
    let root = fresh_root("r2");
    let now = now_secs();
    let (release, _) = make_bundle(&root, Provider::Muse, "review-1", false);
    write_selection(&root, &release, now, now, Freshness::Current);
    // Possibly-active old bundle: unadmitted hash dir, like the review store.
    let old_digest = sha256_hex(b"possibly-active-old");
    let old_bin = root.join("bundles/muse").join(&old_digest).join("bin");
    std::fs::create_dir_all(&old_bin).expect("old bin");
    std::fs::write(old_bin.join("muse"), b"#!/bin/sh\necho old\n").expect("old exe");
    std::fs::create_dir_all(root.join("refs")).expect("refs");
    std::fs::write(root.join("refs/active-old.json"), b"{truncated-active-reference")
        .expect("corrupt ref");
    let cleaned = store::cleanup(&root);
    let succeeded = cleaned.is_ok();
    println!("cleanup_succeeded_with_corrupt_reference={succeeded}");
    expect(!succeeded, "cleanup with corrupt ref must not succeed");
    expect(
        matches!(cleaned, Err(ref e) if matches!(e, soda_provider_tools::Error::Uncertain(_))),
        "cleanup with corrupt ref is Uncertain",
    );
    let preserved = root.join("bundles/muse").join(&old_digest).exists();
    println!("possibly_active_old_bundle_preserved={preserved}");
    expect(preserved, "possibly-active bundle preserved while knowledge corrupt");
    // Sound knowledge again: cleanup succeeds and collects the unreferenced dir.
    std::fs::remove_file(root.join("refs/active-old.json")).expect("drop corrupt ref");
    expect(store::cleanup(&root).is_ok(), "cleanup succeeds with sound knowledge");
    expect(
        !root.join("bundles/muse").join(&old_digest).exists(),
        "unreferenced bundle collected once knowledge is sound",
    );
    // Corrupt selection is equally fatal, before any deletion.
    let root2 = fresh_root("r2sel");
    make_bundle(&root2, Provider::Muse, "review-1", false);
    std::fs::write(root2.join("current-muse.json"), b"{truncated-selection").expect("corrupt sel");
    let cleaned2 = store::cleanup(&root2);
    expect(
        matches!(cleaned2, Err(ref e) if matches!(e, soda_provider_tools::Error::Uncertain(_))),
        "cleanup with corrupt selection is Uncertain",
    );
}

/// R2 concurrency: bounded refresh/acquire/cleanup with no loss.
fn scenario_concurrency() {
    println!("--- concurrency refresh/acquire/cleanup ---");
    use std::sync::Arc;
    let root = fresh_root("rcon");
    std::env::set_var("SODA_PROVIDERS_CHANNEL", "review-invalid-channel");
    std::env::set_var("SODA_PROVIDERS_CHECK_TTL_SECS", "1000000");
    let now = now_secs();
    let (release, _) = make_bundle(&root, Provider::Muse, "review-1", false);
    write_selection(&root, &release, now, now, Freshness::Current);
    let digest = release.digest.clone();
    // Unprotected junk the collector should reclaim exactly once.
    let junk = sha256_hex(b"junk-payload");
    std::fs::create_dir_all(root.join("bundles/muse").join(&junk)).expect("junk");
    std::fs::write(root.join("bundles/muse").join(&junk).join("x"), b"junk").expect("junk file");
    let root = Arc::new(root);
    let digest = Arc::new(digest);
    let acquired = Arc::new(AtomicU64::new(0));
    let cleaned = Arc::new(AtomicU64::new(0));
    let refreshed = Arc::new(AtomicU64::new(0));
    let mut handles = Vec::new();
    for i in 0..8 {
        let (root, digest, acquired, cleaned, refreshed) =
            (Arc::clone(&root), Arc::clone(&digest), Arc::clone(&acquired), Arc::clone(&cleaned), Arc::clone(&refreshed));
        handles.push(std::thread::spawn(move || {
            for _ in 0..25 {
                if i < 2 {
                    // Failed-check refresh preserves Current on disk.
                    if refresh::refresh(Provider::Muse).is_ok() {
                        refreshed.fetch_add(1, Ordering::SeqCst);
                    }
                } else if i < 5 {
                    // Locked resolve+acquire, then release.
                    let (_exe, tok) = resolve::resolve_pinned(Provider::Muse, &digest).expect("pinned");
                    acquired.fetch_add(1, Ordering::SeqCst);
                    resolve::release(&tok).expect("release");
                } else {
                    // Collector under lock; never deletes live bundle.
                    if store::cleanup(&root).is_ok() {
                        cleaned.fetch_add(1, Ordering::SeqCst);
                    }
                }
            }
        }));
    }
    for h in handles {
        h.join().expect("worker survived");
    }
    println!(
        "concurrent refresh_ok={} pinned_acquires={} cleanups_ok={}",
        refreshed.load(Ordering::SeqCst),
        acquired.load(Ordering::SeqCst),
        cleaned.load(Ordering::SeqCst)
    );
    expect(
        resolve::resolve_existing(Provider::Muse, &digest).is_ok(),
        "live bundle survives concurrent refresh/acquire/cleanup",
    );
    let sel = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("selection present");
    expect(
        matches!(sel.freshness, Freshness::Current) && sel.checked_at_secs == now,
        "selection untouched by failed concurrent checks",
    );
    expect(
        !root.join("bundles/muse").join(&junk).exists(),
        "junk collected exactly once under sound knowledge",
    );
    expect(store::cleanup(&root).is_ok(), "final cleanup ok");
}

fn main() {
    let _ = std::fs::remove_dir_all(scratch_base());
    scenario_r1();
    scenario_r3();
    scenario_r2();
    scenario_concurrency();
    let failures = FAILURES.load(Ordering::SeqCst);
    if failures == 0 {
        println!("review_probe_repair=PASS");
    } else {
        println!("review_probe_repair=FAIL ({failures} mismatches)");
        std::process::exit(1);
    }
}
