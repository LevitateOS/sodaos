use super::*;
use crate::sha;
use crate::term::term_binding_tests::{parse, sample};
use crate::term_attach::subscription_lifetime;
use crate::term_collect::collect_finished;
use crate::term_create::{file_sha256_hex, program_stat_ok};
use crate::term_paths::TMUX_CONFIG;
use crate::term_status::parse_ready;

#[test]
fn program_hash_streams_past_64k() {
    // Proves the deliberate no-cap delta: 100KB hashes whole.
    let dir = std::env::temp_dir().join(format!("soda-pt-term-{}-hash", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let big: Vec<u8> = (0..100_000).map(|i| (i * 31 + 7) as u8).collect();
    std::fs::write(dir.join("big"), &big).unwrap();
    let file = std::fs::File::open(dir.join("big")).unwrap();
    assert_eq!(file_sha256_hex(&file).unwrap(), sha::hex_digest(&big));
    assert_eq!(
        sha::hex(&sha::digest(b"abc")),
        "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
    );
    assert_ne!(
        file_sha256_hex(&file).unwrap(),
        sha::hex_digest(&big[..65536])
    );
    assert!(program_stat_ok(0, 0o100644, true));
    assert!(program_stat_ok(0, 0o100755, true));
    assert!(!program_stat_ok(0, 0o100664, true)); // group write
    assert!(!program_stat_ok(1000, 0o100644, true)); // owner
    assert!(!program_stat_ok(0, 0o100644, false)); // not regular
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn ready_matrix() {
    assert_eq!(
        parse_ready(&parse(r#"{"pid":7,"socket":[8,9]}"#)).unwrap(),
        (7, 8, 9)
    );
    // Extra keys tolerated (direct indexing in the `.py`).
    assert_eq!(
        parse_ready(&parse(r#"{"pid":7,"socket":[8,9],"x":1}"#)).unwrap(),
        (7, 8, 9)
    );
    assert_eq!(
        parse_ready(&parse(r#"{"pid":-0,"socket":[-0,9]}"#)).unwrap(),
        (0, 0, 9)
    );
    assert_eq!(
        parse_ready(&parse(r#"{"pid":1e400,"pid":7,"socket":[8,9]}"#)).unwrap(),
        (7, 8, 9)
    );
    assert!(parse_ready(&parse(r#"{"pid":7,"pid":1e400,"socket":[8,9]}"#)).is_err());
    for bad in [
        r#"{"socket":[8,9]}"#,
        r#"{"pid":7}"#,
        r#"{"pid":"7","socket":[8,9]}"#,
        r#"{"pid":7,"socket":[8]}"#,
        r#"{"pid":7,"socket":[8,-1]}"#,
        r#"{"pid":7.0,"socket":[8,9]}"#,
        r#"[]"#,
    ] {
        assert!(parse_ready(&parse(bad)).is_err(), "{bad}");
    }
}

#[test]
fn entry_point_smoke() {
    let account = sample();
    let id = "e".repeat(32);
    // Deterministic failures (bad shape, absent paths, no privilege).
    // `prepare`/`attach` failures write `closed/launch_failed` lines to
    // the harness-captured stdout.
    assert_eq!(prepare("not-an-id"), 1);
    assert_eq!(prepare(&id), 1);
    assert_eq!(attach_terminal("not-an-id", &account, 1, 80, 24, 60), 1);
    assert_eq!(attach_terminal(&id, &account, 1, 80, 24, 60), 1);
    assert!(control_terminal("bogus", &id, &account, 1, 80, 24, "", "", "").is_err());
    assert!(reserve_terminal(&id, &account, 1, 80, 24, "nm", "x", &"c".repeat(64)).is_err());
    assert!(create_terminal("not-an-id", &account, 1, 80, 24, "nm", &"c".repeat(64)).is_err());
    assert!(subscription_lifetime("/definitely/not/here-9f3c").is_err());
    // Environment-dependent outcomes (systemd host vs container):
    // exercised, not asserted.
    let dir = std::env::temp_dir().join(format!("soda-pt-term-{}-smoke", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let fd = std::fs::File::open(&dir).unwrap();
    let _result = binding_record(&fd, None, 0);
    let _result = terminal_status(&id, &account, 1);
    let _result = collect_finished();
    let _result = create_terminal(&id, &account, 1, 80, 24, "nm", &"c".repeat(64));
    let _result = control_terminal("list", "", &account, 1, 80, 24, "", "", "");
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn tmux_config_exact() {
    assert_eq!(
        TMUX_CONFIG,
        "set -g status off\nset -g history-limit 10000\nset -s buffer-limit 10\nset -s set-clipboard off\nset -s escape-time 10\nset -g default-terminal screen-256color\nset -g update-environment \"\"\nset -s exit-unattached off\n"
    );
}
