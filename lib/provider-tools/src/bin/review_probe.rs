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
//! - R3b: a missing digest inventory never disables verification (no legacy
//!   bypass): inventory-less bundles are denied even with intact bytes, and
//!   tamper-then-remove-inventory stays denied.
//! - R2b: competing successful publications race deterministically; the late
//!   finisher aborts onto the winner instead of overwriting it, and the race
//!   advances afterwards.
//! - R5 (UP-REVIEW-05 P1.1): a stale same-digest refresh converges onto a
//!   concurrent newer winner (no same-second ABA overwrite, no stamp
//!   regression); a stale-block keeps newer same-digest freshness knowledge
//!   while still blocking new work.
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
fn make_bundle(
    root: &Path,
    provider: Provider,
    version: &str,
    companion: bool,
) -> (ReleaseIdentity, Vec<u8>) {
    let content = format!("#!/bin/sh\necho review-{provider:?}-{version}\n").into_bytes();
    let digest = sha256_hex(&content);
    let bundle = root.join("bundles").join(provider.as_str()).join(&digest);
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

fn write_selection(
    root: &Path,
    release: &ReleaseIdentity,
    checked: u64,
    admitted: u64,
    freshness: Freshness,
) {
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
    expect(
        refreshed.checked_at_secs == 0,
        "failed refresh leaves checked_at untouched",
    );
    let on_disk = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("selection present");
    expect(
        matches!(&on_disk.freshness, Freshness::Stale { .. }) && on_disk.checked_at_secs == 0,
        "disk selection still Stale with original timestamps",
    );
    let launched = resolve::resolve_new(Provider::Muse).is_ok();
    println!("known_stale_launch_allowed_after_upstream_failure={launched}");
    expect(
        !launched,
        "resolve_new blocks known-stale after failed check",
    );
    // Direct refresh error class: must not block-or-fail, must preserve.
    let blocked = resolve::resolve_new(Provider::Muse).unwrap_err();
    expect(
        blocked.blocks_new_work(),
        "resolve_new error blocks new work",
    );
}

/// R1 for Codex: same stale-preservation rules through the codex provider
/// (companion-bearing) path.
///
/// History note: this scenario once drove a live codex refresh expecting an
/// admission failure, and it passed only because admission itself was broken
/// (the record was written into the staging dir, whose name can never match
/// the digest). Now that real admission works, a live refresh would genuinely
/// admit current upstream bytes, so the driver pins the stale-block contract
/// directly at the compare-and-swap layer instead of downloading live bytes.
/// Live admission is proven separately against an isolated store.
fn scenario_r1_codex() {
    println!("--- R1 stale preservation (codex) ---");
    let root = fresh_root("r1-codex");
    let now = now_secs();
    let (release, _) = make_bundle(&root, Provider::Codex, "review-old", true);
    write_selection(
        &root,
        &release,
        0,
        now.saturating_sub(1000),
        Freshness::Stale {
            latest: "review-known-newer".to_string(),
        },
    );
    let base = store::read_selection(&root, Provider::Codex)
        .expect("read")
        .expect("base present");
    // A known-newer release that cannot be admitted: the discovery is
    // persisted as Stale and new work stays blocked, while the old bytes are
    // never overwritten.
    let known_newer = ReleaseIdentity {
        provider: Provider::Codex,
        version: "review-known-newer".to_string(),
        digest: sha256_hex(b"known-newer-artifact"),
        size: 1,
        artifact_url: "https://example.invalid/review-newer".to_string(),
    };
    let blocked = refresh::publish_stale(
        &root,
        &base,
        &known_newer,
        now,
        "admission failed (deterministic fixture)".to_string(),
    );
    expect(
        blocked.blocks_new_work(),
        "codex admission failure blocks new work",
    );
    let on_disk = store::read_selection(&root, Provider::Codex)
        .expect("read")
        .expect("selection present");
    expect(
        matches!(&on_disk.freshness, Freshness::Stale { latest } if latest == "review-known-newer")
            && on_disk.release.digest == release.digest
            && on_disk.checked_at_secs == now,
        "disk codex selection still Stale on the old bytes (never overwritten)",
    );
    let launched = resolve::resolve_new(Provider::Codex).is_ok();
    println!("codex_known_stale_launch_allowed_after_upstream_failure={launched}");
    expect(
        !launched,
        "resolve_new blocks known-stale codex after failed check",
    );
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
    let exe = root.join("bundles/muse").join(&digest).join("bin/muse");
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
    std::fs::write(
        fake_bin.join("muse"),
        b"#!/bin/sh\necho unadmitted-fixture\n",
    )
    .expect("fake exe");
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
    store::record_inventory(&root.join("bundles/codex").join(&codex.digest), &codex)
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
    std::fs::write(
        root.join("refs/active-old.json"),
        b"{truncated-active-reference",
    )
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
    expect(
        preserved,
        "possibly-active bundle preserved while knowledge corrupt",
    );
    // Sound knowledge again: cleanup succeeds and collects the unreferenced dir.
    std::fs::remove_file(root.join("refs/active-old.json")).expect("drop corrupt ref");
    expect(
        store::cleanup(&root).is_ok(),
        "cleanup succeeds with sound knowledge",
    );
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
        let (root, digest, acquired, cleaned, refreshed) = (
            Arc::clone(&root),
            Arc::clone(&digest),
            Arc::clone(&acquired),
            Arc::clone(&cleaned),
            Arc::clone(&refreshed),
        );
        handles.push(std::thread::spawn(move || {
            for _ in 0..25 {
                if i < 2 {
                    // Failed-check refresh preserves Current on disk.
                    if refresh::refresh(Provider::Muse).is_ok() {
                        refreshed.fetch_add(1, Ordering::SeqCst);
                    }
                } else if i < 5 {
                    // Locked resolve+acquire, then release.
                    let (_exe, tok) =
                        resolve::resolve_pinned(Provider::Muse, &digest).expect("pinned");
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

/// R3b: no legacy bypass — a missing digest inventory never disables content
/// verification, even when the bytes are subsequently restored.
fn scenario_r3b() {
    println!("--- R3b missing inventory denied ---");
    let root = fresh_root("r3b");
    // Legacy-style cache: admission record present, inventory never recorded.
    let legacy_bytes = b"#!/bin/sh\necho legacy-cache\n".to_vec();
    let legacy_digest = sha256_hex(&legacy_bytes);
    let legacy_bin = root.join("bundles/muse").join(&legacy_digest).join("bin");
    std::fs::create_dir_all(&legacy_bin).expect("legacy bin");
    std::fs::write(legacy_bin.join("muse"), &legacy_bytes).expect("legacy exe");
    chmod_exec(&legacy_bin.join("muse"));
    let legacy = ReleaseIdentity {
        provider: Provider::Muse,
        version: "legacy".to_string(),
        digest: legacy_digest.clone(),
        size: legacy_bytes.len() as u64,
        artifact_url: "https://example.invalid/review".to_string(),
    };
    store::write_release_record(&root.join("bundles/muse").join(&legacy_digest), &legacy)
        .expect("legacy record");
    let accepted = resolve::resolve_existing(Provider::Muse, &legacy_digest).is_ok();
    println!("missing_inventory_still_accepted={accepted}");
    expect(!accepted, "legacy cache without digest inventory denied");
    // Fully admitted bundle: remove the inventory -> denied; tamper plus
    // remove -> denied; restore bytes without inventory -> still denied.
    let (release, exe_bytes) = make_bundle(&root, Provider::Muse, "review-1", false);
    let digest = release.digest.clone();
    let bundle = root.join("bundles/muse").join(&digest);
    expect(
        resolve::resolve_existing(Provider::Muse, &digest).is_ok(),
        "admitted bundle with inventory resolves",
    );
    std::fs::remove_file(bundle.join("inventory.json")).expect("remove inventory");
    let accepted = resolve::resolve_existing(Provider::Muse, &digest).is_ok();
    println!("missing_inventory_still_accepted={accepted}");
    expect(!accepted, "removed inventory denies verification");
    let exe = bundle.join("bin/muse");
    let mut tampered = exe_bytes.clone();
    tampered.extend_from_slice(b"# tampered\n");
    std::fs::write(&exe, &tampered).expect("tamper");
    let accepted = resolve::resolve_existing(Provider::Muse, &digest).is_ok();
    println!("tampered_after_inventory_removed_accepted={accepted}");
    expect(!accepted, "tampered bytes with removed inventory denied");
    std::fs::write(&exe, &exe_bytes).expect("restore");
    chmod_exec(&exe);
    expect(
        resolve::resolve_existing(Provider::Muse, &digest).is_err(),
        "restored bytes without inventory still denied (no re-hash)",
    );
}

/// R2b: competing successful publications race; the late finisher must abort
/// onto the winner instead of overwriting it (lost update/downgrade).
fn scenario_publish_race() {
    println!("--- R2b publication race ---");
    let root = fresh_root("rrace");
    let now = now_secs();
    let (v0, _) = make_bundle(&root, Provider::Muse, "race-v0", false);
    write_selection(&root, &v0, now, now, Freshness::Current);
    let base = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("base present");
    // Two competing discoveries, both fully admitted BEFORE either publishes:
    // real release records plus admission-time digest inventories.
    let (rel_b, _) = make_bundle(&root, Provider::Muse, "race-B", false);
    let (rel_c, _) = make_bundle(&root, Provider::Muse, "race-C", false);
    let admitted_b = Selection {
        provider: Provider::Muse,
        release: rel_b.clone(),
        checked_at_secs: now,
        admitted_at_secs: now,
        freshness: Freshness::Current,
    };
    let admitted_c = Selection {
        provider: Provider::Muse,
        release: rel_c.clone(),
        checked_at_secs: now,
        admitted_at_secs: now,
        freshness: Freshness::Current,
    };
    // Deterministic late finisher: B spins until C has published and only
    // then publishes, so B's publication strictly races late. Both threads go
    // through the real locked publication path, and the wait is bounded.
    let c_won = std::sync::atomic::AtomicBool::new(false);
    let (winner_c, winner_b) = std::thread::scope(|s| {
        let c_handle = s.spawn(|| {
            let winner = refresh::publish(&root, Some(&base), admitted_c).expect("C publishes");
            c_won.store(true, std::sync::atomic::Ordering::SeqCst);
            winner
        });
        let b_handle = s.spawn(|| {
            let start = std::time::Instant::now();
            while !c_won.load(std::sync::atomic::Ordering::SeqCst) {
                if start.elapsed() > std::time::Duration::from_secs(60) {
                    panic!("C never published within 60s");
                }
                std::thread::yield_now();
            }
            refresh::publish(&root, Some(&base), admitted_b).expect("late B converges")
        });
        (
            c_handle.join().expect("C thread survived"),
            b_handle.join().expect("B thread survived"),
        )
    });
    expect(
        winner_c.release.digest == rel_c.digest,
        "first publisher C wins",
    );
    expect(
        winner_b.release.digest == rel_c.digest,
        "late B aborts onto winner C",
    );
    let on_disk = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("selection present");
    let overwrote = on_disk.release.digest == rel_b.digest;
    println!("late_publish_overwrote_winner={overwrote}");
    expect(!overwrote, "late publication does not overwrite the winner");
    expect(
        on_disk.release.digest == rel_c.digest,
        "stored current is C",
    );
    expect(
        on_disk.checked_at_secs == now && on_disk.admitted_at_secs == now,
        "stored current carries the winner's stamps",
    );
    // The replaced base is retained as previous; the loser's stale snapshot
    // never clobbered it.
    let prev_bytes = std::fs::read(root.join("previous-muse.json")).expect("previous file");
    let prev: Selection = serde_json::from_slice(&prev_bytes).expect("previous parses");
    expect(
        prev.release.digest == v0.digest,
        "previous retains the raced base V0",
    );
    // Both admitted payloads are untouched: winner resolves, loser verifies.
    expect(
        resolve::resolve_existing(Provider::Muse, &rel_c.digest).is_ok(),
        "winner C resolves",
    );
    expect(
        store::verify_bundle(&root, Provider::Muse, &rel_b.digest).is_ok(),
        "loser B bytes unharmed and reusable",
    );
    // Forward progress after the race: a fresh base publishes D.
    let (rel_d, _) = make_bundle(&root, Provider::Muse, "race-D", false);
    let base_c = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("C current");
    let admitted_d = Selection {
        provider: Provider::Muse,
        release: rel_d.clone(),
        checked_at_secs: now,
        admitted_at_secs: now,
        freshness: Freshness::Current,
    };
    let winner_d = refresh::publish(&root, Some(&base_c), admitted_d).expect("D publishes");
    expect(
        winner_d.release.digest == rel_d.digest,
        "post-race publication advances to D",
    );
    let on_disk = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("selection present");
    expect(
        on_disk.release.digest == rel_d.digest,
        "stored current is D",
    );
    let prev_bytes = std::fs::read(root.join("previous-muse.json")).expect("previous file");
    let prev: Selection = serde_json::from_slice(&prev_bytes).expect("previous parses");
    expect(
        prev.release.digest == rel_c.digest,
        "previous advances to C",
    );
}

/// UP-REVIEW-05 P1.1: a same-digest refresh that read its base before a
/// concurrent newer winner published must converge onto the winner, never
/// overwrite it (even within the same second); same-digest stamp advances
/// never regress; and a stale-block against newer same-digest freshness
/// knowledge must keep that knowledge while still blocking new work.
fn scenario_unchanged_vs_newer_winner() {
    println!("--- unchanged vs newer winner ---");
    let root = fresh_root("r5-unchanged");
    let now = now_secs();
    let (rel_x, _) = make_bundle(&root, Provider::Muse, "race-X", false);
    write_selection(&root, &rel_x, now, now, Freshness::Current);
    let base = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("base present");
    // B wins with newer digest Y while A is still "checking upstream".
    let (rel_y, _) = make_bundle(&root, Provider::Muse, "race-Y", false);
    let admitted_y = Selection {
        provider: Provider::Muse,
        release: rel_y.clone(),
        checked_at_secs: now,
        admitted_at_secs: now,
        freshness: Freshness::Current,
    };
    let winner = refresh::publish(&root, Some(&base), admitted_y).expect("Y publishes");
    expect(winner.release.digest == rel_y.digest, "newer Y wins");
    // A's stale same-digest refresh (X, advanced check stamp) must converge.
    let stale_unchanged = Selection {
        provider: Provider::Muse,
        release: rel_x.clone(),
        checked_at_secs: now.saturating_add(5),
        admitted_at_secs: now,
        freshness: Freshness::Current,
    };
    let converged = refresh::publish_checked(&root, &base, &stale_unchanged).expect("converges");
    expect(
        converged.release.digest == rel_y.digest,
        "stale unchanged refresh converges onto newer Y",
    );
    let on_disk = store::read_selection(&root, Provider::Muse)
        .expect("read")
        .expect("selection present");
    expect(
        on_disk.release.digest == rel_y.digest && on_disk.checked_at_secs == now,
        "disk still Y with winner stamps (no overwrite, no same-second ABA)",
    );
    // Same-digest stamp race: B advanced the check stamp on X; A's older
    // advance from the same base must not regress it.
    let root2 = fresh_root("r5-unchanged-stamps");
    let (rel_x2, _) = make_bundle(&root2, Provider::Muse, "race-X2", false);
    write_selection(&root2, &rel_x2, now, now, Freshness::Current);
    let base2 = store::read_selection(&root2, Provider::Muse)
        .expect("read")
        .expect("base present");
    write_selection(
        &root2,
        &rel_x2,
        now.saturating_add(9),
        now,
        Freshness::Current,
    );
    let late = Selection {
        provider: Provider::Muse,
        release: rel_x2.clone(),
        checked_at_secs: now.saturating_add(5),
        admitted_at_secs: now,
        freshness: Freshness::Current,
    };
    let converged2 = refresh::publish_checked(&root2, &base2, &late).expect("converges");
    expect(
        converged2.checked_at_secs == now.saturating_add(9),
        "same-digest late write converges, no stamp regression",
    );
    // Fresh base still advances: disk == base writes the new stamps.
    let base3 = store::read_selection(&root2, Provider::Muse)
        .expect("read")
        .expect("fresh base present");
    let advance = Selection {
        checked_at_secs: now.saturating_add(12),
        ..base3.clone()
    };
    let written = refresh::publish_checked(&root2, &base3, &advance).expect("advances");
    expect(
        written.checked_at_secs == now.saturating_add(12),
        "fresh-base unchanged refresh advances the check stamp",
    );
    // Stale-block against newer same-digest knowledge keeps that knowledge.
    let root3 = fresh_root("r5-stale-knowledge");
    let (rel_x3, _) = make_bundle(&root3, Provider::Muse, "race-X3", false);
    write_selection(&root3, &rel_x3, now, now, Freshness::Current);
    let base4 = store::read_selection(&root3, Provider::Muse)
        .expect("read")
        .expect("base present");
    write_selection(
        &root3,
        &rel_x3,
        now.saturating_add(3),
        now,
        Freshness::Stale {
            latest: "race-Z".to_string(),
        },
    );
    let blocked = refresh::publish_stale(
        &root3,
        &base4,
        &rel_y,
        now.saturating_add(7),
        "race-Y admission failed".to_string(),
    );
    expect(
        blocked.blocks_new_work(),
        "stale-block still blocks new work",
    );
    let on_disk3 = store::read_selection(&root3, Provider::Muse)
        .expect("read")
        .expect("selection present");
    expect(
        matches!(&on_disk3.freshness, Freshness::Stale { latest } if latest == "race-Z")
            && on_disk3.checked_at_secs == now.saturating_add(3),
        "newer same-digest freshness knowledge retained, no regression",
    );
    // Plain stale-block still records when the base is fresh.
    let root4 = fresh_root("r5-stale-plain");
    let (rel_x4, _) = make_bundle(&root4, Provider::Muse, "race-X4", false);
    write_selection(&root4, &rel_x4, now, now, Freshness::Current);
    let base5 = store::read_selection(&root4, Provider::Muse)
        .expect("read")
        .expect("base present");
    let blocked2 = refresh::publish_stale(
        &root4,
        &base5,
        &rel_y,
        now.saturating_add(7),
        "race-Y admission failed".to_string(),
    );
    expect(
        blocked2.blocks_new_work(),
        "plain stale-block blocks new work",
    );
    let on_disk4 = store::read_selection(&root4, Provider::Muse)
        .expect("read")
        .expect("selection present");
    expect(
        matches!(&on_disk4.freshness, Freshness::Stale { latest } if *latest == rel_y.version)
            && on_disk4.checked_at_secs == now.saturating_add(7),
        "fresh-base stale-block records the known-newer discovery",
    );
}

fn main() {
    let _ = std::fs::remove_dir_all(scratch_base());
    scenario_r1();
    scenario_r1_codex();
    scenario_r3();
    scenario_r3b();
    scenario_r2();
    scenario_concurrency();
    scenario_publish_race();
    scenario_unchanged_vs_newer_winner();
    let failures = FAILURES.load(Ordering::SeqCst);
    if failures == 0 {
        println!("review_probe_repair=PASS");
    } else {
        println!("review_probe_repair=FAIL ({failures} mismatches)");
        std::process::exit(1);
    }
}
