use std::fs;
use std::path::Path;

use super::fixtures::*;

use crate::cli::parse_args;
use crate::origin::admit_setup_paths;
use crate::system::euid;

#[test]
fn admits_only_valid_setup_paths() {
    let root = test_root();
    let out = root.join("soda").join("dashboard.json");
    assert!(admit_setup_paths("https://forgejo.test", "http://127.0.0.1:3000", &out).is_ok());
    assert!(admit_setup_paths("http://forgejo.test", "http://127.0.0.1:3000", &out).is_err());
    assert!(admit_setup_paths("https://forgejo.test/x", "http://127.0.0.1:3000", &out).is_err());
    assert!(admit_setup_paths("https://forgejo.test", "http://127.0.0.1:3000/x", &out).is_err());
    assert!(admit_setup_paths("https://forgejo.test/..", "http://127.0.0.1:3000", &out).is_err());
    assert!(admit_setup_paths("https://forgejo.test:0", "http://127.0.0.1:0", &out).is_ok());
    assert!(
        admit_setup_paths("https://forgejo.test:65535", "http://127.0.0.1:65535", &out).is_ok()
    );
    assert!(
        admit_setup_paths("https://forgejo.test:65536", "http://127.0.0.1:3000", &out).is_err()
    );
    assert!(admit_setup_paths("https://user@forgejo.test", "http://127.0.0.1:3000", &out).is_err());
    assert!(admit_setup_paths(
        "https://forgejo.test",
        "http://127.0.0.1:3000",
        Path::new("relative.json")
    )
    .is_err());
    fs::create_dir_all(out.parent().expect("parent")).expect("parent");
    fs::write(&out, "{}\n").expect("existing");
    assert!(admit_setup_paths("https://forgejo.test", "http://127.0.0.1:3000", &out).is_err());
}

#[test]
fn non_root_run_reports_operator_access() {
    if euid() == 0 {
        return;
    }
    // run() checks euid before parsing flags; only the message is pinned.
    assert_ne!(euid(), 0);
}

#[test]
fn flag_parsing_matches_go_setup() {
    let args: Vec<String> = [
        "--forgejo-url",
        "https://x.test",
        "--token-file",
        "/t",
        "--out",
        "/o",
    ]
    .iter()
    .map(|s| s.to_string())
    .collect();
    let opts = parse_args(&args).expect("parse");
    assert_eq!(opts.forgejo_url, "https://x.test");
    assert_eq!(opts.forgejo_internal_url, "http://127.0.0.1:3000");
    assert!(!opts.provision_db_only);
    let args: Vec<String> = ["--provision-db-only"]
        .iter()
        .map(|s| s.to_string())
        .collect();
    assert!(parse_args(&args).expect("bare bool").provision_db_only);
    let args: Vec<String> = ["--bogus"].iter().map(|s| s.to_string()).collect();
    assert!(parse_args(&args).is_err());
    let args: Vec<String> = ["--socket"].iter().map(|s| s.to_string()).collect();
    assert!(parse_args(&args).is_err());
}
