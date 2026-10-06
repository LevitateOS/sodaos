use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::fixtures::*;

use crate::secrets::{provision_postgres_secrets, reuse_postgres_secrets};
use crate::setup::setup;

#[test]
fn provisions_all_four_postgres_files() {
    let root = test_root();
    let dir = root.join("soda");
    fs::create_dir(&dir).expect("soda dir");
    let token_path = write_token(&root);
    let stub = stub_server(true, false, "synthetic-bootstrap-token-not-for-retention");
    let pg_dir = root.join("postgres");
    let mut stdout: Vec<u8> = Vec::new();
    setup(
        "https://forgejo.test/",
        &stub.url,
        &token_path.to_string_lossy(),
        &dir.join("dashboard.json"),
        &pg_dir,
        &mut stdout,
    )
    .expect("setup");
    let mut passwords = Vec::new();
    for role in ["super", "forgejo", "soda"] {
        let path = pg_dir.join(format!("{role}.passwd"));
        let mode = fs::metadata(&path).expect("passwd").permissions().mode() & 0o777;
        assert_eq!(mode, 0o600, "{role} mode");
        let pw = fs::read_to_string(&path)
            .expect("passwd")
            .trim()
            .to_string();
        assert_eq!(pw.len(), 64, "{role} length");
        assert!(
            pw.bytes()
                .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()),
            "{role} hex"
        );
        passwords.push(pw);
    }
    assert_ne!(passwords[0], passwords[1]);
    assert_ne!(passwords[1], passwords[2]);
    assert_ne!(passwords[0], passwords[2]);
    let dsn_path = pg_dir.join("soda.dsn");
    let mode = fs::metadata(&dsn_path).expect("dsn").permissions().mode() & 0o777;
    assert_eq!(mode, 0o640);
    let dsn = fs::read_to_string(&dsn_path).expect("dsn");
    assert_eq!(
        dsn.trim(),
        format!(
            "postgres://soda:{}@/soda?host=/run/soda/postgres&sslmode=disable",
            passwords[2]
        )
    );
}

#[test]
fn reuses_complete_pre_existing_secrets() {
    let root = test_root();
    let dir = root.join("soda");
    fs::create_dir(&dir).expect("soda dir");
    let token_path = write_token(&root);
    let stub = stub_server(true, false, "synthetic-bootstrap-token-not-for-retention");
    let pg_dir = root.join("postgres");
    fs::create_dir_all(&pg_dir).expect("pg dir");
    let soda_pw = "b".repeat(64);
    let seed = [
        ("super.passwd", "a".repeat(64) + "\n"),
        ("forgejo.passwd", "c".repeat(64) + "\n"),
        ("soda.passwd", soda_pw.clone() + "\n"),
        (
            "soda.dsn",
            format!("postgres://soda:{soda_pw}@/soda?host=/run/soda/postgres&sslmode=disable\n"),
        ),
    ];
    for (name, value) in &seed {
        fs::write(pg_dir.join(name), value).expect("seed");
    }
    let mut stdout: Vec<u8> = Vec::new();
    setup(
        "https://forgejo.test/",
        &stub.url,
        &token_path.to_string_lossy(),
        &dir.join("dashboard.json"),
        &pg_dir,
        &mut stdout,
    )
    .expect("setup with complete secrets");
    for (name, value) in &seed {
        assert_eq!(
            &fs::read_to_string(pg_dir.join(name)).expect("seed"),
            value,
            "{name} regenerated"
        );
    }
    let config = fs::read_to_string(dir.join("dashboard.json")).expect("config");
    assert!(config.contains(&pg_dir.join("soda.dsn").to_string_lossy().into_owned()));
}

#[test]
fn disagreeing_secrets_are_reported() {
    let root = test_root();
    let pg_dir = root.join("postgres");
    fs::create_dir_all(&pg_dir).expect("pg dir");
    fs::write(pg_dir.join("super.passwd"), "a\n").expect("seed");
    fs::write(pg_dir.join("forgejo.passwd"), "c\n").expect("seed");
    fs::write(pg_dir.join("soda.passwd"), "b\n").expect("seed");
    fs::write(pg_dir.join("soda.dsn"), "postgres://soda:other@/soda\n").expect("seed");
    let err = reuse_postgres_secrets(&pg_dir).expect_err("disagree");
    assert!(err.contains("exist but disagree"), "{err}");
    // Partial sets report None so the caller fails O_EXCL instead.
    fs::remove_file(pg_dir.join("soda.dsn")).expect("remove");
    assert!(reuse_postgres_secrets(&pg_dir).expect("partial").is_none());
}

#[test]
fn provision_only_reports_dsn_path() {
    let root = test_root();
    let pg_dir = root.join("postgres");
    let (_, dsn) = provision_postgres_secrets(&pg_dir).expect("provision");
    assert_eq!(dsn, pg_dir.join("soda.dsn"));
    // Second run reuses without regenerating.
    let before = fs::read_to_string(&dsn).expect("dsn");
    let (created, again) = provision_postgres_secrets(&pg_dir).expect("reuse");
    assert!(created.is_empty());
    assert_eq!(again, dsn);
    assert_eq!(fs::read_to_string(&dsn).expect("dsn"), before);
}
