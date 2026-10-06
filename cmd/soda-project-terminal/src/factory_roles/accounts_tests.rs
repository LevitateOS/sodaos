use super::*;
use crate::error::Error;
use crate::testutil::Scratch;
use std::os::unix::fs::PermissionsExt;
use std::path::Path;

fn mode(path: &Path) -> u32 {
    std::fs::metadata(path).unwrap().permissions().mode() & 0o777
}

#[test]
fn useradd_recipe_is_exact() {
    assert_eq!(
        useradd_argv("soda-coder"),
        vec![
            "useradd",
            "--create-home",
            "--shell",
            "/usr/sbin/nologin",
            "--password",
            "!",
            "soda-coder",
        ]
        .into_iter()
        .map(str::to_string)
        .collect::<Vec<_>>()
    );
}

#[test]
fn ensure_provisions_locked_roles() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    crate::fsx::ensure_layout(&ctx).unwrap();
    for login in ["soda-coder", "soda-reviewer"] {
        let account = ensure_role(&ctx, login).unwrap();
        assert_eq!(account.name, login);
        assert_eq!(account.shell, NOLOGIN);
        assert_eq!(mode(&account.dir), 0o700);
        assert_eq!(mode(&account.dir.join("checkouts")), 0o755);
        assert_eq!(mode(&ctx.credentials.join(login)), 0o755);
    }
    // Second pass re-verifies without effects.
    for login in ["soda-coder", "soda-reviewer"] {
        ensure_role(&ctx, login).unwrap();
    }
    assert!(role_record(&ctx, "root").unwrap().is_none());
}

#[test]
fn ensure_refuses_interactive_accounts() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    crate::fsx::ensure_layout(&ctx).unwrap();
    let overlay = ctx.factory.join("test-accounts");
    std::fs::create_dir_all(&overlay).unwrap();
    std::fs::write(
        overlay.join("soda-coder.json"),
        "{\"shell\": \"/bin/bash\"}",
    )
    .unwrap();
    match ensure_role(&ctx, "soda-coder") {
        Err(Error::Fail(text)) => assert_eq!(
            text,
            "existing factory account has an interactive shell: soda-coder"
        ),
        other => panic!("unexpected {other:?}"),
    }
}

#[test]
fn ensure_refuses_external_keys_and_links() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    crate::fsx::ensure_layout(&ctx).unwrap();
    let keys = ctx.factory.join("test-ssh-keys");
    std::fs::create_dir_all(&keys).unwrap();
    std::fs::write(keys.join("soda-coder"), "ssh-ed25519 AAAA").unwrap();
    match ensure_role(&ctx, "soda-coder") {
        Err(Error::Fail(text)) => assert_eq!(
            text,
            "factory account must not hold external SSH keys: soda-coder"
        ),
        other => panic!("unexpected {other:?}"),
    }
    std::fs::remove_file(keys.join("soda-coder")).unwrap();
    ensure_role(&ctx, "soda-coder").unwrap();
    // A symlinked checkout root is refused even though the home exists.
    let home = ctx.factory.join("test-homes").join("soda-coder");
    std::fs::remove_dir(home.join("checkouts")).unwrap();
    std::os::unix::fs::symlink("/tmp", home.join("checkouts")).unwrap();
    match ensure_role(&ctx, "soda-coder") {
        Err(Error::Fail(text)) => {
            assert_eq!(text, "unsafe factory checkout root: soda-coder")
        }
        other => panic!("unexpected {other:?}"),
    }
}

#[test]
fn ensure_refuses_bad_roles() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    match ensure_role(&ctx, "root") {
        Err(Error::Fail(text)) => assert_eq!(text, "unsupported factory role"),
        other => panic!("unexpected {other:?}"),
    }
}
