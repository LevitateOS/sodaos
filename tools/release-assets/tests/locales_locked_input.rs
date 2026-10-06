#![cfg(unix)]

// Shared across suites; each suite uses a subset.
#[allow(dead_code)]
#[path = "locales_support/mod.rs"]
mod locales_support;

use locales_support::{run, status_of, TempDir, EXTRA};

#[test]
fn lock_with_unexpected_source_is_refused_without_fetch() {
    let dir = TempDir::new("source");
    let lock = dir.file(
        "lock.json",
        br#"{"url": "https://example.com/locale.ini", "sha256": "abc"}"#,
    );
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--lock",
            lock.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("unexpected native catalog source"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn lock_document_problems_fail_without_fetch() {
    let dir = TempDir::new("lockdoc");
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    for (tag, body) in [
        ("broken", b"{not json".as_slice()),
        (
            "keyless",
            br#"{"url": "https://codeberg.org/forgejo/forgejo/raw/tag/v1/x"}"#.as_slice(),
        ),
    ] {
        let lock = dir.file(&format!("{tag}.json"), body);
        let out = dir.path.join(format!("{tag}.ini"));
        let output = run(
            &dir.path,
            &[
                "--lock",
                lock.to_str().unwrap(),
                "--additions",
                additions.to_str().unwrap(),
                "--out",
                out.to_str().unwrap(),
            ],
        );
        assert_eq!(
            status_of(&output),
            1,
            "{tag}: {}",
            String::from_utf8_lossy(&output.stderr)
        );
        assert!(!out.exists());
    }
    let missing = dir.path.join("missing.json");
    let out = dir.path.join("missing.ini");
    let output = run(
        &dir.path,
        &[
            "--lock",
            missing.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 1);
    assert!(!out.exists());
}
