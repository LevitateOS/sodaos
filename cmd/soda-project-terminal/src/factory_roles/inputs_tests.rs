use super::*;
use crate::testutil::{
    approve_default, approve_value, assert_fail, fixture_files, op_value, Scratch, COMMIT, PID,
    PID2,
};
use soda_json::JsonValue;
use std::os::unix::fs::PermissionsExt;
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
