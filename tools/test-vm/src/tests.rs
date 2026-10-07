use super::state::{pid_alive, pid_display, ssh_args};
use super::stripped;

#[test]
fn pid_display_keeps_spaces_drops_newlines() {
    assert_eq!(pid_display(b"  3512  \n\n"), "  3512  ");
    assert_eq!(pid_display(b"42"), "42");
    assert_eq!(pid_display(b""), "");
}

#[test]
fn pid_alive_matches_kill_zero_semantics() {
    let mine = std::process::id().to_string();
    assert!(pid_alive(mine.as_bytes()));
    assert!(pid_alive(format!("  {mine}  \n").as_bytes()));
    assert!(pid_alive(format!("0{mine}\n").as_bytes()));
    assert!(pid_alive(b"0"));
    assert!(!pid_alive(b""));
    assert!(!pid_alive(b"\n\n"));
    assert!(!pid_alive(b"abc\n"));
    assert!(!pid_alive(b"12 3\n"));
    assert!(!pid_alive(b"-5\n"));
    assert!(!pid_alive(b"2147483647\n"));
    assert!(!pid_alive(b"9999999999\n"));
}

#[test]
fn ssh_args_pin_key_paths_and_options() {
    assert_eq!(
        ssh_args("/repo/.artifacts/test-vm"),
        vec![
            "-p",
            "22220",
            "-i",
            "/repo/.artifacts/test-vm/operator",
            "-o",
            "IdentitiesOnly=yes",
            "-o",
            "StrictHostKeyChecking=yes",
            "-o",
            "UserKnownHostsFile=/repo/.artifacts/test-vm/known_hosts",
        ]
        .into_iter()
        .map(|part| part.to_string())
        .collect::<Vec<String>>(),
    );
}

#[test]
fn stripped_drops_all_trailing_newlines() {
    assert_eq!(stripped(b"Linux\n"), b"Linux");
    assert_eq!(stripped(b"x\n\n\n"), b"x");
    assert_eq!(stripped(b""), b"");
}
