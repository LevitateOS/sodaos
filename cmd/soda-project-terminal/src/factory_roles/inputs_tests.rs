use super::*;
use crate::testutil::{
    approve_default, approve_value, assert_fail, fixture_files, op_value, Scratch, COMMIT, PID,
    PID2,
};
use soda_json::JsonValue;
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::path::Path;

fn mode(path: &Path) -> u32 {
    std::fs::metadata(path).unwrap().permissions().mode() & 0o777
}

#[test]
fn git_recipes_are_exact() {
    let bundle = Path::new("/s/source.bundle");
    let repo = Path::new("/s/verify-tmp/repo");
    assert_eq!(
        git_clone_argv(crate::GIT, bundle, repo),
        vec![
            "/usr/bin/git",
            "clone",
            "-q",
            "--no-checkout",
            "/s/source.bundle",
            "/s/verify-tmp/repo"
        ]
        .into_iter()
        .map(str::to_string)
        .collect::<Vec<_>>()
    );
    assert_eq!(
        git_catfile_argv(crate::GIT, repo, &"c".repeat(40)),
        vec![
            "/usr/bin/git",
            "-C",
            "/s/verify-tmp/repo",
            "cat-file",
            "-e",
            &"c".repeat(40)
        ]
        .into_iter()
        .map(str::to_string)
        .collect::<Vec<_>>()
    );
}

#[test]
fn ensure_reports_fixed_roles() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let result = do_ensure(&ctx, &op_value("ensure", None)).unwrap();
    assert_eq!(
        crate::emit::dumps_default(&result),
        "{\"roles\": [\"soda-coder\", \"soda-reviewer\"]}"
    );
    assert_fail(
        do_ensure(&ctx, &op_value("ensure", Some(PID))),
        "unsupported ensure request",
    );
}

#[test]
fn approve_writes_protected_snapshot_and_verifies_bundle() {
    let scratch = Scratch::fresh();
    let (git, record) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let result = do_approve(&ctx, &approve_default(PID)).unwrap();
    assert_eq!(result.get("approved").and_then(|v| v.as_str()), Some(PID));
    assert_eq!(
        result.get("repeated").and_then(|v| v.as_bool()),
        Some(false)
    );
    let snapshot = ctx.preparations.join(PID).join("snapshot");
    assert_eq!(std::fs::read(snapshot.join("setup.sh")).unwrap(), b"true\n");
    assert_eq!(std::fs::read(snapshot.join("check.sh")).unwrap(), b"true\n");
    assert_eq!(mode(&snapshot.join("setup.sh")), 0o644);
    assert_eq!(mode(&snapshot.join("source.bundle")), 0o644);
    assert_eq!(
        std::fs::read(snapshot.join("source.bundle")).unwrap(),
        b"bundle"
    );
    assert!(!snapshot.join("verify-tmp").exists());
    // Exact git argv, in order.
    let log = scratch.record_text(&record);
    let bundle = snapshot
        .join("source.bundle")
        .to_string_lossy()
        .into_owned();
    let repo = snapshot
        .join("verify-tmp")
        .join("repo")
        .to_string_lossy()
        .into_owned();
    assert!(
        log.contains(&format!(
            "---\n<clone>\n<-q>\n<--no-checkout>\n<{bundle}>\n<{repo}>\n"
        )),
        "{log}"
    );
    assert!(
        log.contains(&format!(
            "---\n<-C>\n<{repo}>\n<cat-file>\n<-e>\n<{COMMIT}>\n"
        )),
        "{log}"
    );
    // The checkout is role-owned; the receipt pins the inputs.
    let checkout = Path::new(result.get("checkout").and_then(|v| v.as_str()).unwrap());
    assert!(checkout.is_dir());
    assert_eq!(mode(checkout), 0o755);
    assert_eq!(std::fs::read_dir(checkout).unwrap().count(), 0);
    assert!(!ctx.preparations.join(PID).join("checkout-tmp").exists());
    assert_eq!(
        result.get("credential_file").and_then(|v| v.as_str()),
        Some("")
    );
    let receipt = std::fs::read_to_string(ctx.preparations.join(PID).join("request.json")).unwrap();
    assert!(receipt.contains(&format!("\"id\": \"{PID}\"")));
    // Identical repeat resumes; different inputs conflict.
    let repeated = do_approve(&ctx, &approve_default(PID)).unwrap();
    assert_eq!(
        repeated.get("repeated").and_then(|v| v.as_bool()),
        Some(true)
    );
    let mut conflict = approve_default(PID);
    if let JsonValue::Object(entries) = &mut conflict {
        for (key, value) in entries.iter_mut() {
            if key == "setup_digest" {
                *value = JsonValue::Str("e".repeat(64));
            }
        }
    }
    assert_fail(
        do_approve(&ctx, &conflict),
        "approved inputs do not match their digest",
    );
}

#[test]
fn approve_rejects_untrusted_inputs_before_effects() {
    let scratch = Scratch::fresh();
    let (git, record) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let files_b64 = |contents: &[u8]| crate::testutil::b64_encode(contents);
    type Mutate<'a> = dyn Fn(&mut Vec<(String, JsonValue)>) + 'a;
    let base = |mutate: &Mutate<'_>| {
        let mut value = approve_value(PID, "soda-coder", &fixture_files(), b"bundle", "");
        // Rebuild with the caller's mutation over canonical fields.
        let JsonValue::Object(entries) = &mut value else {
            panic!("object");
        };
        mutate(entries);
        value
    };
    let set = |entries: &mut Vec<(String, JsonValue)>, key: &str, value: JsonValue| match entries
        .iter_mut()
        .find(|(k, _)| k == key)
    {
        Some(slot) => slot.1 = value,
        None => entries.push((key.to_string(), value)),
    };
    let cases: Vec<(&str, JsonValue)> = vec![
        (
            "role",
            base(&|e| set(e, "role", JsonValue::Str("root".to_string()))),
        ),
        (
            "id",
            base(&|e| set(e, "id", JsonValue::Str("../escape".to_string()))),
        ),
        (
            "digest",
            base(&|e| set(e, "setup_digest", JsonValue::Str("zz".to_string()))),
        ),
        (
            "commit",
            base(&|e| set(e, "source_commit", JsonValue::Str("short".to_string()))),
        ),
        (
            "credential",
            base(&|e| set(e, "credential", JsonValue::Str("../x".to_string()))),
        ),
        (
            "files",
            base(&|e| {
                set(
                    e,
                    "files",
                    JsonValue::Object(vec![(
                        "setup.sh".to_string(),
                        JsonValue::Str(files_b64(b"x")),
                    )]),
                )
            }),
        ),
        (
            "bundle",
            base(&|e| set(e, "bundle", JsonValue::Str("!!!".to_string()))),
        ),
        (
            "extra",
            base(&|e| set(e, "extra", JsonValue::Number("1".to_string()))),
        ),
    ];
    for (name, bad) in &cases {
        assert!(do_approve(&ctx, bad).is_err(), "{name} accepted");
    }
    assert_eq!(scratch.record_text(&record), "", "git ran before refusal");
    assert!(
        !ctx.preparations.join(PID).exists(),
        "effects before refusal"
    );
}

#[test]
fn approve_enforces_all_bounds() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    // Nine files.
    let mut nine = fixture_files();
    for i in 0..7 {
        nine.push((format!("extra{i}.sh"), b"x".to_vec()));
    }
    assert_fail(
        do_approve(
            &ctx,
            &approve_value(PID, "soda-coder", &nine, b"bundle", ""),
        ),
        "unsupported approved file set",
    );
    // One oversized file.
    let big = vec![
        ("setup.sh".to_string(), vec![b'x'; 32 * 1024 + 1]),
        ("check.sh".to_string(), b"true\n".to_vec()),
    ];
    assert_fail(
        do_approve(&ctx, &approve_value(PID, "soda-coder", &big, b"bundle", "")),
        "unsupported approved file size",
    );
    // Total over the cap with each file legal.
    let wide = vec![
        ("setup.sh".to_string(), vec![b'x'; 32 * 1024]),
        ("check.sh".to_string(), vec![b'y'; 32 * 1024]),
        ("a.sh".to_string(), vec![b'z'; 32 * 1024]),
        ("b.sh".to_string(), vec![b'w'; 32 * 1024]),
        ("c.sh".to_string(), b"q".to_vec()),
    ];
    assert_fail(
        do_approve(
            &ctx,
            &approve_value(PID, "soda-coder", &wide, b"bundle", ""),
        ),
        "approved inputs exceed the bounded size",
    );
    // Oversized bundle.
    assert_fail(
        do_approve(
            &ctx,
            &approve_value(
                PID,
                "soda-coder",
                &fixture_files(),
                &vec![0u8; 512 * 1024 + 1],
                "",
            ),
        ),
        "unsupported source bundle size",
    );
    // Empty bundle and empty file.
    assert_fail(
        do_approve(
            &ctx,
            &approve_value(PID, "soda-coder", &fixture_files(), b"", ""),
        ),
        "unsupported source bundle size",
    );
    let empty = vec![
        ("setup.sh".to_string(), Vec::new()),
        ("check.sh".to_string(), b"true\n".to_vec()),
    ];
    assert_fail(
        do_approve(
            &ctx,
            &approve_value(PID, "soda-coder", &empty, b"bundle", ""),
        ),
        "unsupported approved file size",
    );
    assert!(!ctx.preparations.join(PID).exists());
}

#[test]
fn approve_failure_removes_everything() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-fail", 1);
    let ctx = scratch.ctx(&git);
    assert_fail(
        do_approve(&ctx, &approve_default(PID)),
        "source bundle does not carry the approved commit",
    );
    assert!(!ctx.preparations.join(PID).exists());
    let checkouts = ctx
        .factory
        .join("test-homes")
        .join("soda-coder")
        .join("checkouts");
    assert!(!checkouts.join(PID).exists());
    // And the identity is reusable afterwards with working git.
    let (git_ok, _) = scratch.git_script("git-ok", 0);
    let ctx_ok = scratch.ctx(&git_ok);
    do_approve(&ctx_ok, &approve_default(PID)).unwrap();
}

/// Seed a foreign preexisting checkout holding operator bytes, resolving the
/// role home through production account lookup rather than a hardcoded layout.
fn preexisting_checkout(ctx: &crate::Ctx, marker: &[u8]) -> std::path::PathBuf {
    let account = crate::account::role_record(ctx, "soda-coder")
        .expect("role record")
        .expect("soda-coder account");
    let checkout = account.dir.join("checkouts").join(PID);
    std::fs::create_dir_all(&checkout).unwrap();
    // A realistic preexisting home is secured; only the checkout is foreign.
    std::fs::set_permissions(
        &account.dir,
        std::os::unix::fs::PermissionsExt::from_mode(0o700),
    )
    .unwrap();
    std::fs::write(checkout.join("marker.txt"), marker).unwrap();
    checkout
}

#[test]
fn approve_refusal_preserves_preexisting_checkout() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let checkout = preexisting_checkout(&ctx, b"operator data");
    assert_fail(
        do_approve(&ctx, &approve_default(PID)),
        "checkout path already exists",
    );
    assert_eq!(
        std::fs::read(checkout.join("marker.txt")).unwrap(),
        b"operator data",
        "preexisting checkout bytes preserved on refusal"
    );
    assert!(
        !ctx.preparations.join(PID).exists(),
        "owned preparation state still cleaned"
    );
}

#[test]
fn approve_bundle_failure_preserves_preexisting_checkout() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-fail", 1);
    let ctx = scratch.ctx(&git);
    let checkout = preexisting_checkout(&ctx, b"operator data");
    assert_fail(
        do_approve(&ctx, &approve_default(PID)),
        "source bundle does not carry the approved commit",
    );
    assert_eq!(
        std::fs::read(checkout.join("marker.txt")).unwrap(),
        b"operator data",
        "preexisting checkout bytes preserved on bundle failure"
    );
}

/// Resolve the checkout path an approve of `PID` as soda-coder would claim.
/// Pure production path math; touches nothing.
fn claimed_checkout(ctx: &crate::Ctx) -> std::path::PathBuf {
    let account = crate::account::role_record(ctx, "soda-coder")
        .expect("role record")
        .expect("soda-coder account");
    account.dir.join("checkouts").join(PID)
}

/// CODEX-P07-003: a checkout appearing DURING bundle verification (after any
/// pre-probe) is refused and must survive: the failure arm only removes
/// state this invocation created. The git fixture plants it synchronously
/// inside verification, so the race is deterministic. Pre-fix the planted
/// directory is deleted.
#[test]
fn approve_refusal_preserves_checkout_planted_during_verification() {
    let scratch = Scratch::fresh();
    let (git_probe, _) = scratch.git_script("git-ok", 0);
    let checkout = claimed_checkout(&scratch.ctx(&git_probe));
    let record = scratch.root.join("git-plant.record");
    let script = scratch.root.join("git-plant");
    let body = format!(
        "#!/bin/sh\n{{ echo '---'; printf '<%s>\\\\n' \"$@\"; }} >> '{}'\nmkdir -p '{}' && echo 'foreign data' > '{}/marker.txt'\nexit 0\n",
        record.to_string_lossy(),
        checkout.to_string_lossy(),
        checkout.to_string_lossy(),
    );
    std::fs::write(&script, body).expect("git script");
    std::fs::set_permissions(&script, std::fs::Permissions::from_mode(0o755)).expect("chmod");
    let ctx = scratch.ctx(&script);
    assert_fail(
        do_approve(&ctx, &approve_default(PID)),
        "checkout path already exists",
    );
    assert_eq!(
        std::fs::read(checkout.join("marker.txt")).unwrap(),
        b"foreign data\n",
        "checkout planted during verification survives refusal"
    );
    assert!(
        !ctx.preparations.join(PID).exists(),
        "owned preparation state still cleaned"
    );
}

/// CODEX-P07-003: a dangling symlink at the checkout path is refused and
/// preserved (metadata identity, not existence, decides). Pre-fix the
/// failure arm deletes the link itself.
#[test]
fn approve_refusal_preserves_dangling_checkout_symlink() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let checkout = claimed_checkout(&ctx);
    std::fs::create_dir_all(checkout.parent().unwrap()).unwrap();
    let account = crate::account::role_record(&ctx, "soda-coder")
        .expect("role record")
        .expect("soda-coder account");
    // A realistic preexisting home is secured; only the link is foreign.
    std::fs::set_permissions(
        &account.dir,
        std::os::unix::fs::PermissionsExt::from_mode(0o700),
    )
    .unwrap();
    std::os::unix::fs::symlink("/nonexistent-soda-target", &checkout).unwrap();
    assert_fail(
        do_approve(&ctx, &approve_default(PID)),
        "checkout path already exists",
    );
    assert_eq!(
        std::fs::read_link(&checkout).unwrap(),
        Path::new("/nonexistent-soda-target"),
        "dangling checkout link survives refusal"
    );
}

/// CODEX-P07-003 guard: a regular file at the checkout path is refused and
/// preserved. Green before and after; pins the identity branch.
#[test]
fn approve_refusal_preserves_checkout_path_file() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let checkout = claimed_checkout(&ctx);
    std::fs::create_dir_all(checkout.parent().unwrap()).unwrap();
    let account = crate::account::role_record(&ctx, "soda-coder")
        .expect("role record")
        .expect("soda-coder account");
    std::fs::set_permissions(
        &account.dir,
        std::os::unix::fs::PermissionsExt::from_mode(0o700),
    )
    .unwrap();
    std::fs::write(&checkout, b"operator bytes").unwrap();
    assert_fail(
        do_approve(&ctx, &approve_default(PID)),
        "checkout path already exists",
    );
    assert_eq!(
        std::fs::read(&checkout).unwrap(),
        b"operator bytes",
        "file at checkout path survives refusal"
    );
}

/// CODEX-P07-003 guard: a failure AFTER the checkout was created (valid
/// name, missing credential file) still removes the invocation-owned
/// checkout and preparation. Green before and after; proves the provenance
/// restructure preserves owned cleanup.
#[test]
fn approve_credential_failure_removes_owned_checkout() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let request = approve_value(PID, "soda-coder", &fixture_files(), b"bundle", "nokey");
    assert_fail(
        do_approve(&ctx, &request),
        "assigned service credential is missing",
    );
    let checkout = claimed_checkout(&ctx);
    assert!(!checkout.exists(), "owned checkout removed on late failure");
    assert!(
        !ctx.preparations.join(PID).exists(),
        "owned preparation removed on late failure"
    );
}

/// CODEX-P07-003 guard: when the checkout cannot even be metadata-probed
/// (unwritable parent), approve refuses without deleting anything. Green
/// before and after; pins metadata-error refusal through the restructure.
#[test]
fn approve_refuses_when_checkout_parent_unwritable() {
    if unsafe { libc::geteuid() } == 0 {
        eprintln!("SKIP unwritable-parent case: root writes through 555");
        return;
    }
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let checkout = claimed_checkout(&ctx);
    std::fs::create_dir_all(checkout.parent().unwrap()).unwrap();
    let account = crate::account::role_record(&ctx, "soda-coder")
        .expect("role record")
        .expect("soda-coder account");
    std::fs::set_permissions(
        &account.dir,
        std::os::unix::fs::PermissionsExt::from_mode(0o700),
    )
    .unwrap();
    std::fs::set_permissions(
        checkout.parent().unwrap(),
        std::fs::Permissions::from_mode(0o555),
    )
    .unwrap();
    let result = do_approve(&ctx, &approve_default(PID));
    std::fs::set_permissions(
        checkout.parent().unwrap(),
        std::fs::Permissions::from_mode(0o755),
    )
    .unwrap();
    assert!(
        result.is_err(),
        "unwritable parent must refuse, got {result:?}"
    );
    assert!(
        !ctx.preparations.join(PID).exists(),
        "owned preparation removed on metadata refusal"
    );
    assert!(
        !checkout.exists(),
        "no checkout created on metadata refusal"
    );
}

/// CODEX-P07-003b: the checkouts parent is role-owned, so the claimed name
/// can be swapped between creation and cleanup independently of the helper
/// lock. A racing rename loop must never cost foreign bytes: every round
/// still fails, and the foreign marker always survives. Pre-fix the
/// path-based cleanup deletes the marker whenever the race hits
/// (overwhelming at 100 forced-failure rounds); post-fix the fd-bound
/// cleanup preserves it structurally. Red-pre is probabilistic by nature
/// of races; green-post is deterministic.
#[test]
fn approve_failure_preserves_checkout_swapped_during_cleanup() {
    use std::sync::atomic::{AtomicBool, Ordering};
    use std::sync::Arc;
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let checkout = claimed_checkout(&ctx);
    let parent = checkout.parent().unwrap().to_path_buf();
    std::fs::create_dir_all(&parent).unwrap();
    let account = crate::account::role_record(&ctx, "soda-coder")
        .expect("role record")
        .expect("soda-coder account");
    std::fs::set_permissions(
        &account.dir,
        std::os::unix::fs::PermissionsExt::from_mode(0o700),
    )
    .unwrap();
    // Foreign decoy holding the marker; the loop swaps it with the claim.
    let decoy = parent.join("soda-decoy");
    std::fs::create_dir_all(&decoy).unwrap();
    std::fs::write(decoy.join("marker.txt"), b"foreign data").unwrap();
    let stop = Arc::new(AtomicBool::new(false));
    let swap = {
        let stop = stop.clone();
        let checkout = checkout.clone();
        let decoy = decoy.clone();
        std::thread::spawn(move || {
            while !stop.load(Ordering::Relaxed) {
                let _ = std::fs::rename(&decoy, &checkout);
                let _ = std::fs::rename(&checkout, &decoy);
            }
        })
    };
    // Forced late failure (missing credential) runs creation+cleanup
    // every round the create wins the race.
    let request = approve_value(PID, "soda-coder", &fixture_files(), b"bundle", "nokey");
    for _ in 0..100 {
        let result = do_approve(&ctx, &request);
        assert!(result.is_err(), "swap round must fail, got {result:?}");
    }
    stop.store(true, Ordering::Relaxed);
    swap.join().unwrap();
    // The marker survives at one side or the other; nothing deleted it.
    let survived = [&decoy, &checkout].iter().any(|dir| {
        std::fs::read(dir.join("marker.txt"))
            .map(|body| body == b"foreign data")
            .unwrap_or(false)
    });
    assert!(survived, "racing swap cost foreign bytes");
}

/// CODEX-P07-CUSTODY-17 characterization: the bind primitives refuse
/// non-directories and symlinks (never following a link, even to a real
/// directory); staging publishes atomically with inode continuity, and the
/// carried descriptor mutates the published inode after the move.
/// Green-post (the custody surface is new).
#[test]
fn checkout_identity_primitives_refuse_and_preserve() {
    let root = std::env::temp_dir().join(format!("soda-p3b-{}", std::process::id()));
    std::fs::create_dir_all(root.join("parent")).unwrap();
    let parent = root.join("parent");
    assert!(open_dir_no_follow(&parent.join("absent")).is_err());
    std::fs::write(parent.join("file"), b"x").unwrap();
    assert!(open_dir_no_follow(&parent.join("file")).is_err());
    std::os::unix::fs::symlink("file", parent.join("link")).unwrap();
    assert!(open_dir_no_follow(&parent.join("link")).is_err());
    std::os::unix::fs::symlink("/nonexistent-soda-target", parent.join("dangle")).unwrap();
    assert!(open_dir_no_follow(&parent.join("dangle")).is_err());
    // A symlink to a real directory must NOT bind through the link.
    std::fs::create_dir(parent.join("realdir")).unwrap();
    std::os::unix::fs::symlink("realdir", parent.join("dirlink")).unwrap();
    assert!(open_dir_no_follow(&parent.join("dirlink")).is_err());
    let parent_fd = open_dir_no_follow(&parent).unwrap();
    // Descriptor-relative bind refuses links and non-directories too.
    assert!(openat_dir_no_follow(&parent_fd, "file").is_err());
    assert!(openat_dir_no_follow(&parent_fd, "dirlink").is_err());
    assert!(openat_dir_no_follow(&parent_fd, "absent").is_err());
    // Exclusive staging; restaging over any identity is mechanical Exists.
    mkdirat_exclusive(&parent_fd, "staged", 0o700).unwrap();
    assert!(matches!(
        mkdirat_exclusive(&parent_fd, "staged", 0o700),
        Err(crate::error::Error::Exists)
    ));
    // Publish, then mutate through the carried descriptor: the published
    // path shows this invocation's inode, mode, and ownership.
    let staging = openat_dir_no_follow(&parent_fd, "staged").unwrap();
    let before = staging.metadata().unwrap();
    publish_checkout(&parent_fd, "staged", &parent_fd, "published").unwrap();
    assert!(!parent.join("staged").exists());
    fchmod(&staging, 0o755).unwrap();
    let uid = unsafe { libc::geteuid() };
    let gid = unsafe { libc::getegid() };
    fchown(&staging, uid, gid).unwrap();
    let after = std::fs::metadata(parent.join("published")).unwrap();
    assert_eq!((after.dev(), after.ino()), (before.dev(), before.ino()));
    assert!(after.is_dir());
    assert_eq!(mode(&parent.join("published")), 0o755);
    assert_eq!((after.uid(), after.gid()), (uid, gid));
    std::fs::remove_dir_all(&root).ok();
}

/// CUSTODY-17 publication: the public name refuses every preexisting
/// identity without replacement — including an empty directory, which an
/// ordinary rename would silently swap — and a refused publication leaves
/// both the staging scope and the public name untouched. Quiet fixtures;
/// same-UID test mappings cannot qualify production custody, only the
/// refusal mechanics.
#[test]
fn checkout_publication_refuses_without_replacement() {
    let root = std::env::temp_dir().join(format!("soda-pub-{}", std::process::id()));
    std::fs::create_dir_all(root.join("staging")).unwrap();
    std::fs::create_dir_all(root.join("public")).unwrap();
    let staging_dir = root.join("staging");
    let public = root.join("public");
    let staging_parent = open_dir_no_follow(&staging_dir).unwrap();
    let public_parent = open_dir_no_follow(&public).unwrap();
    let assert_refused = |result: Result<(), crate::error::Error>| match result {
        Err(crate::error::Error::Fail(text)) => assert_eq!(text, "checkout path already exists"),
        other => panic!("expected Fail(checkout path already exists), got {other:?}"),
    };
    mkdirat_exclusive(&staging_parent, "staged", 0o700).unwrap();
    // Non-empty dir with foreign bytes.
    std::fs::create_dir(public.join("full")).unwrap();
    std::fs::write(public.join("full").join("marker.txt"), b"foreign").unwrap();
    assert_refused(publish_checkout(
        &staging_parent,
        "staged",
        &public_parent,
        "full",
    ));
    // Empty dir: ordinary rename would replace; noreplace refuses.
    std::fs::create_dir(public.join("empty")).unwrap();
    assert_refused(publish_checkout(
        &staging_parent,
        "staged",
        &public_parent,
        "empty",
    ));
    // Regular file.
    std::fs::write(public.join("file"), b"operator bytes").unwrap();
    assert_refused(publish_checkout(
        &staging_parent,
        "staged",
        &public_parent,
        "file",
    ));
    // Dangling symlink.
    std::os::unix::fs::symlink("/nonexistent-soda-target", public.join("dangle")).unwrap();
    assert_refused(publish_checkout(
        &staging_parent,
        "staged",
        &public_parent,
        "dangle",
    ));
    // Nothing replaced, nothing lost: staging and every target survive.
    assert!(staging_dir.join("staged").is_dir());
    assert_eq!(
        std::fs::read(public.join("full").join("marker.txt")).unwrap(),
        b"foreign"
    );
    assert!(public.join("empty").is_dir());
    assert_eq!(
        std::fs::read(public.join("file")).unwrap(),
        b"operator bytes"
    );
    assert_eq!(
        std::fs::read_link(public.join("dangle")).unwrap(),
        Path::new("/nonexistent-soda-target")
    );
    // And the surviving staging still publishes onto a free name.
    publish_checkout(&staging_parent, "staged", &public_parent, "claimed").unwrap();
    assert!(public.join("claimed").is_dir());
    assert!(!staging_dir.join("staged").exists());
    std::fs::remove_dir_all(&root).ok();
}

/// CUSTODY-17 publication across filesystems fails explicitly with both
/// sides preserved and no copy fallback. Skips where no second writable
/// filesystem exists: the refusal string is pinned only where runnable.
#[test]
fn checkout_publication_across_filesystems_fails_explicitly() {
    let root = std::env::temp_dir().join(format!("soda-xdev-{}", std::process::id()));
    std::fs::create_dir_all(root.join("staging")).unwrap();
    let shm =
        std::path::PathBuf::from("/dev/shm").join(format!("soda-xdev-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&shm);
    if std::fs::create_dir_all(shm.join("public")).is_err() {
        eprintln!("SKIP cross-filesystem publication: /dev/shm unavailable");
        return;
    }
    let staging_dev = std::fs::metadata(root.join("staging")).unwrap().dev();
    let public_dev = std::fs::metadata(shm.join("public")).unwrap().dev();
    if staging_dev == public_dev {
        eprintln!("SKIP cross-filesystem publication: single filesystem");
        let _ = std::fs::remove_dir_all(&shm);
        return;
    }
    let staging_parent = open_dir_no_follow(&root.join("staging")).unwrap();
    let public_parent = open_dir_no_follow(&shm.join("public")).unwrap();
    mkdirat_exclusive(&staging_parent, "staged", 0o700).unwrap();
    match publish_checkout(&staging_parent, "staged", &public_parent, "claimed") {
        Err(crate::error::Error::Fail(text)) => {
            assert_eq!(text, "checkout publication crosses filesystems")
        }
        other => panic!("expected Fail(crosses filesystems), got {other:?}"),
    }
    assert!(root.join("staging").join("staged").is_dir());
    assert!(!shm.join("public").join("claimed").exists());
    let _ = std::fs::remove_dir_all(&shm);
    std::fs::remove_dir_all(&root).ok();
}

#[test]
fn approve_binds_role_private_credentials() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    // Missing credential file.
    assert_fail(
        do_approve(
            &ctx,
            &approve_value(PID, "soda-coder", &fixture_files(), b"bundle", "db.env"),
        ),
        "assigned service credential is missing",
    );
    assert!(!ctx.preparations.join(PID).exists());
    // Plant a proper credential: role-owned, 0600.
    do_approve(&ctx, &approve_default(PID2)).unwrap();
    let cred = ctx.credentials.join("soda-coder").join("db.env");
    std::fs::write(&cred, b"secret").unwrap();
    std::fs::set_permissions(&cred, std::fs::Permissions::from_mode(0o600)).unwrap();
    let result = do_approve(
        &ctx,
        &approve_value(PID, "soda-coder", &fixture_files(), b"bundle", "db.env"),
    )
    .unwrap();
    assert_eq!(
        result.get("credential_file").and_then(|v| v.as_str()),
        Some(cred.to_string_lossy().as_ref())
    );
    // World-readable credentials are refused.
    std::fs::set_permissions(&cred, std::fs::Permissions::from_mode(0o644)).unwrap();
    assert_fail(
        do_approve(
            &ctx,
            &approve_value(
                "f999999999999999999999999",
                "soda-coder",
                &fixture_files(),
                b"bundle",
                "db.env",
            ),
        ),
        "assigned service credential is not role-private",
    );
}
