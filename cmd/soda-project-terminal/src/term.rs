//! Terminal lifecycle (port of the `TERMINALS` half of
//! `project_terminal.py`: reserve/create/attach/inspect/list/end/rename plus
//! `prepare`).
//!
//! Merge-safe: uses only the brief-pinned `sys`/`fs`/`timex` signatures plus
//! the scaffold (`proto`/`pyemit`/`b64`/`sha`), `libc`, and `soda_json`.
//!
//! Two deliberate deltas from the `.py`, both documented at their sites:
//! 1. `reserve` streams the whole `project-terminal` binary for the
//!    `source_hash` check (no 64KB cap — the binary replaces the script).
//! 2. `String` errors cannot separate `NotFound` from other I/O failures, so
//!    absence checks use an explicit existence probe first. Every such site
//!    runs under the TERMINALS parent lock, where concurrent mutation (the
//!    only way the probe could go stale) is impossible.

use std::io;

use soda_json::JsonValue;

use crate::account::Account;
use crate::svc;
use crate::sys;
pub(crate) use crate::term_attach::{attach_terminal, subscription_command};
pub(crate) use crate::term_binding::binding_record;
use crate::term_binding::{binding_matches_account, read_reservation, write_name};
pub(crate) use crate::term_collect::remove_owned_files;
use crate::term_collect::terminal_directories;
pub(crate) use crate::term_create::{create_terminal, file_sha256_hex, reserve_terminal};
use crate::term_paths::list_dir_names;
pub(crate) use crate::term_paths::{checked_chain, record_exists, terminal_path, TERMINALS};
pub(crate) use crate::term_prepare::prepare;
use crate::term_status::{status_state, terminal_status};

// ---------------------------------------------------------------------------
// Binding + reservation records.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Status classification.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Inventory + retirement.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Reserve.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// List / mutate / control.
// ---------------------------------------------------------------------------

fn list_owned_terminals(account: &Account, identity: i64) -> Result<Vec<JsonValue>, String> {
    let mut values = Vec::new();
    for identifier in terminal_directories()? {
        let path = format!("{TERMINALS}/{identifier}");
        let directory = checked_chain(&path)?;
        let record = binding_record(&directory, None, 0)?;
        let pending = read_reservation(&directory)?;
        drop(directory);
        if pending.is_none()
            && record.identity == identity
            && binding_matches_account(&record, account)
        {
            if let Some(value) = terminal_status(&identifier, account, identity)? {
                if status_state(&value)? != "ended" {
                    values.push(value);
                }
            }
        }
    }
    Ok(values)
}

fn mutate_terminal(
    action: &str,
    identifier: &str,
    account: &Account,
    identity: i64,
    name: &str,
) -> Result<Vec<JsonValue>, String> {
    let path = terminal_path(identifier)?;
    // Absence probe first (always under the parent lock, like above).
    if let Err(err) = std::fs::symlink_metadata(&path) {
        if err.kind() == io::ErrorKind::NotFound {
            if action == "end" && terminal_status(identifier, account, identity)?.is_none() {
                return Ok(Vec::new());
            }
            return Err("terminal missing".to_string());
        }
        return Err(err.to_string());
    }
    let directory = checked_chain(&path)?;
    binding_record(&directory, Some(account), identity)?;
    if action == "end" {
        if list_dir_names(&directory)?
            .iter()
            .any(|n| n == "subscription")
        {
            return Err("subscription requires broker End".to_string());
        }
        svc::stop_service(identifier, account)?;
        remove_owned_files(&path, &directory, account)?;
        return Ok(Vec::new());
    }
    if action != "rename" {
        return Err("terminal action".to_string());
    }
    write_name(&directory, name)?;
    drop(directory);
    match terminal_status(identifier, account, identity)? {
        None => Ok(Vec::new()),
        Some(value) => Ok(vec![value]),
    }
}

/// Dispatch a control action under the TERMINALS parent lock (shared for
/// reads, exclusive for mutations).
#[allow(clippy::too_many_arguments)] // 9-arg shape mandated by the PR25 brief
pub fn control_terminal(
    action: &str,
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    name: &str,
    source_hash: &str,
    scope: &str,
) -> Result<Vec<JsonValue>, String> {
    let parent = checked_chain(TERMINALS)?;
    if action == "list" || action == "inspect" {
        sys::flock_shared(&parent).map_err(|e| e.to_string())?;
    } else {
        sys::flock_exclusive(&parent).map_err(|e| e.to_string())?;
    }
    let result = match action {
        "reserve" => reserve_terminal(
            identifier,
            account,
            identity,
            cols,
            rows,
            name,
            source_hash,
            scope,
        )
        .map(|v| vec![v]),
        "create" => {
            create_terminal(identifier, account, identity, cols, rows, name, scope).map(|v| vec![v])
        }
        "list" => list_owned_terminals(account, identity),
        "inspect" => {
            terminal_status(identifier, account, identity).map(|v| v.into_iter().collect())
        }
        _ => mutate_terminal(action, identifier, account, identity, name),
    };
    drop(parent);
    result
}

// ---------------------------------------------------------------------------
// Prepare (systemd ExecStartPost).
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;
    use crate::pyemit;
    use crate::sha;
    use crate::term_attach::{lifetime_argv, subscription_lifetime};
    use crate::term_binding::{
        binding_object, reservation_object, validate_binding, validate_reservation,
    };
    use crate::term_collect::collect_finished;
    use crate::term_create::{file_sha256_hex, program_stat_ok, systemd_run_argv};
    use crate::term_paths::{PROGRAM, TMUX_CONFIG};
    use crate::term_prepare::tmux_new_session_args;
    use crate::term_status::{
        classify_bare_state, parse_ready, ready_object, status_object, UnitClass,
    };

    fn sample() -> Account {
        Account {
            pw_name: "op".to_string(),
            pw_uid: 1001,
            pw_gid: 1002,
            pw_dir: "/home/op".to_string(),
            pw_shell: "/bin/bash".to_string(),
        }
    }

    fn binding_doc(account: &str, identity: &str, cols: &str, rows: &str, created: &str) -> String {
        format!(
            "{{\"account\":{account},\"identity\":{identity},\"cols\":{cols},\"rows\":{rows},\"created_at\":{created}}}"
        )
    }

    fn good_account() -> String {
        r#"["op",1001,1002,"/home/op","/bin/bash"]"#.to_string()
    }

    fn parse(text: &str) -> JsonValue {
        JsonValue::parse(text).unwrap()
    }

    #[test]
    fn path_matrix() {
        let id = "a".repeat(32);
        assert_eq!(
            terminal_path(&id).unwrap(),
            format!("/run/soda-terminals/{id}")
        );
        assert!(terminal_path("").is_err());
        assert!(terminal_path(&"A".repeat(32)).is_err());
        assert!(terminal_path("short").is_err());
        assert!(terminal_path("../escape").is_err());
    }

    #[test]
    fn binding_matrix() {
        let good = binding_doc(&good_account(), "7", "80", "24", "1700000000");
        let record = validate_binding(&parse(&good)).unwrap();
        assert_eq!(record.login, "op");
        assert_eq!((record.uid, record.gid), (1001, 1002));
        assert_eq!(
            (record.identity, record.cols, record.rows, record.created_at),
            (7, 80, 24, 1700000000)
        );
        // Shape violations.
        assert!(validate_binding(&parse("[1,2]")).is_err());
        assert!(
            validate_binding(&parse(r#"{"account":[],"identity":1,"cols":80,"rows":24}"#)).is_err()
        );
        assert!(validate_binding(&parse(
            &binding_doc(&good_account(), "7", "80", "24", "1700000000").replace("cols", "colz")
        ))
        .is_err());
        // Duplicate keys tolerated (json.loads last-wins), unlike control frames.
        let dup = binding_doc(&good_account(), "7", "80", "24", "1");
        let dup = dup.replace("\"created_at\":1", "\"created_at\":1,\"created_at\":9");
        assert_eq!(validate_binding(&parse(&dup)).unwrap().created_at, 9);
        // Account vector violations.
        for bad in [
            r#"["Root",1001,1002,"/home/op","/bin/bash"]"#, // login case
            r#"["root",1001,1002,"/home/op","/bin/bash"]"#, // root
            r#"["op",0,1002,"/home/op","/bin/bash"]"#,      // uid 0
            r#"["op",-1,1002,"/home/op","/bin/bash"]"#,     // uid negative
            r#"["op",1001,-1,"/home/op","/bin/bash"]"#,     // gid negative
            r#"["op",1001,1002,"home/op","/bin/bash"]"#,    // relative dir
            r#"["op",1001,1002,"/home/op","bin/bash"]"#,    // relative shell
            r#"["op",1001,1002,"/home/op"]"#,               // short
            r#"["op",1001.0,1002,"/home/op","/bin/bash"]"#, // float uid
            r#"["op",true,1002,"/home/op","/bin/bash"]"#,   // bool uid
            r#""op""#,                                      // not a list
        ] {
            assert!(
                validate_binding(&parse(&binding_doc(bad, "7", "80", "24", "1700000000"))).is_err(),
                "{bad}"
            );
        }
        // Scalar violations.
        for (identity, cols, rows, created) in [
            ("0", "80", "24", "1700000000"),       // identity 0
            ("-3", "80", "24", "1700000000"),      // identity negative
            ("7.0", "80", "24", "1700000000"),     // identity float
            ("7", "1", "24", "1700000000"),        // cols small
            ("7", "501", "24", "1700000000"),      // cols big
            ("7", "80", "301", "1700000000"),      // rows big
            ("7", "\"80\"", "24", "1700000000"),   // cols string
            ("7", "80", "24", "0"),                // created 0
            ("7", "80", "24", "-1"),               // created negative
            ("7", "80", "24", "9007199254740992"), // created > 2^53-1
            ("7", "80", "24", "true"),             // created bool
        ] {
            assert!(
                validate_binding(&parse(&binding_doc(
                    &good_account(),
                    identity,
                    cols,
                    rows,
                    created
                )))
                .is_err(),
                "{identity}/{cols}/{rows}/{created}"
            );
        }
        // Upper bound 2^53-1 accepted.
        assert!(validate_binding(&parse(&binding_doc(
            &good_account(),
            "7",
            "80",
            "24",
            "9007199254740991"
        )))
        .is_ok());
    }

    #[test]
    fn account_match_matrix() {
        let record = validate_binding(&parse(&binding_doc(
            &good_account(),
            "7",
            "80",
            "24",
            "1700000000",
        )))
        .unwrap();
        let account = sample();
        assert!(binding_matches_account(&record, &account));
        let mut other = account.clone();
        other.pw_uid = 1003;
        assert!(!binding_matches_account(&record, &other));
        let mut other = account.clone();
        other.pw_shell = "/bin/sh".to_string();
        assert!(!binding_matches_account(&record, &other));
    }

    #[test]
    fn reservation_matrix() {
        let scope = "c".repeat(64);
        let good = format!("{{\"expires\":9999999999,\"scope\":\"{scope}\"}}");
        let permit = validate_reservation(&parse(&good)).unwrap();
        assert_eq!((permit.expires, permit.scope), (9999999999, scope));
        for bad in [
            r#"{"expires":9999999999}"#.to_string(),
            r#"{"expires":0,"scope":""}"#.to_string(),
            format!("{{\"expires\":-1,\"scope\":\"{}\"}}", "c".repeat(64)),
            format!("{{\"expires\":99.5,\"scope\":\"{}\"}}", "c".repeat(64)),
            r#"{"expires":99,"scope":"short"}"#.to_string(),
            format!("{{\"expires\":99,\"scope\":\"{}\"}}", "C".repeat(64)),
            format!(
                "{{\"expires\":99,\"scope\":\"{}\",\"x\":1}}",
                "c".repeat(64)
            ),
            r#"[]"#.to_string(),
        ] {
            assert!(validate_reservation(&parse(&bad)).is_err(), "{bad}");
        }
    }

    #[test]
    fn missing_reservation_is_none() {
        // Absence probe works without root-owned files.
        let dir = std::env::temp_dir().join(format!("soda-pt-term-{}-res", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let fd = std::fs::File::open(&dir).unwrap();
        assert_eq!(read_reservation(&fd).unwrap(), None);
        assert!(!record_exists(&fd, "reservation").unwrap());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn classify_matrix() {
        use UnitClass::*;
        // Stopped units: live permit → opening, else ended.
        assert_eq!(
            classify_bare_state("inactive", false, false).unwrap(),
            Ended
        );
        assert_eq!(classify_bare_state("failed", true, false).unwrap(), Ended);
        assert_eq!(
            classify_bare_state("inactive", true, true).unwrap(),
            Opening
        );
        // Live units with a permit are corrupt.
        for state in [
            "active",
            "activating",
            "deactivating",
            "reloading",
            "unknown",
        ] {
            assert!(classify_bare_state(state, true, true).is_err(), "{state}");
            assert!(classify_bare_state(state, true, false).is_err(), "{state}");
        }
        // Live states without a permit.
        assert_eq!(
            classify_bare_state("active", false, false).unwrap(),
            ReadyCheck
        );
        assert_eq!(
            classify_bare_state("activating", false, false).unwrap(),
            Opening
        );
        assert_eq!(
            classify_bare_state("deactivating", false, false).unwrap(),
            Ending
        );
        assert!(classify_bare_state("reloading", false, false).is_err());
        assert!(classify_bare_state("", false, false).is_err());
    }

    #[test]
    fn systemd_argv_exact() {
        let id = "a".repeat(32);
        let path = format!("/run/soda-terminals/{id}");
        let account = sample();
        let argv = systemd_run_argv(&id, &account, &path, &[]);
        let head = [
            "/usr/bin/systemd-run",
            "--quiet",
            "--collect",
            &format!("--unit=soda-terminal-{id}"),
            &format!("--description=Soda terminal {id}"),
            "--service-type=exec",
            "--property=User=op",
            "--property=Group=1002",
            "--property=WorkingDirectory=/home/op",
            "--slice=system.slice",
            "--property=KillMode=control-group",
            "--property=SendSIGKILL=yes",
            "--property=Restart=no",
            "--property=TimeoutStartSec=10s",
            "--property=TimeoutStopSec=3s",
            "--property=UMask=0077",
            "--property=LimitCORE=0",
            "--property=StandardInput=null",
            "--property=StandardOutput=null",
            "--property=StandardError=null",
            &format!("--property=ExecStartPost=+{PROGRAM} prepare {id}"),
            "--setenv=HOME=/home/op",
            "--setenv=USER=op",
            "--setenv=LOGNAME=op",
            "--setenv=SHELL=/bin/bash",
            "--setenv=PATH=/usr/local/bin:/usr/bin:/bin",
            "--setenv=TERM=xterm-256color",
            "--setenv=LANG=C.UTF-8",
            "/usr/bin/tmux",
            "-D",
            "-S",
            &format!("{path}/screen/socket"),
            "-f",
            &format!("{path}/tmux.conf"),
        ];
        assert_eq!(
            argv,
            head.into_iter().map(|s| s.to_string()).collect::<Vec<_>>()
        );
        // No python3 anywhere in the hook.
        assert!(!argv.iter().any(|a| a.contains("python")));
        // Lifetime slots in after LimitCORE.
        let argv = systemd_run_argv(
            &id,
            &account,
            &path,
            &["--property=RuntimeMaxSec=60".to_string()],
        );
        let at = argv
            .iter()
            .position(|a| a == "--property=LimitCORE=0")
            .unwrap();
        assert_eq!(argv[at + 1], "--property=RuntimeMaxSec=60");
        assert_eq!(argv[at + 2], "--property=StandardInput=null");
    }

    #[test]
    fn tmux_session_argv_exact() {
        assert_eq!(
            tmux_new_session_args(80, 24, "/home/op", &[]),
            [
                "new-session",
                "-d",
                "-s",
                "soda",
                "-x",
                "80",
                "-y",
                "24",
                "-c",
                "/home/op",
                ";",
                "set-option",
                "-s",
                "exit-empty",
                "on"
            ]
            .into_iter()
            .map(|s| s.to_string())
            .collect::<Vec<_>>()
        );
        let profile = vec!["exec /usr/bin/sleep infinity".to_string()];
        let argv = tmux_new_session_args(100, 30, "/home/op", &profile);
        assert_eq!(argv[10], "exec /usr/bin/sleep infinity");
        assert_eq!(&argv[11..], &[";", "set-option", "-s", "exit-empty", "on"]);
    }

    #[test]
    fn subscription_command_exact() {
        let path = "/run/soda-terminals/abc";
        assert_eq!(
            subscription_command(path),
            vec![
                "-e".to_string(),
                format!("CODEX_HOME={path}/model/auth"),
                "-e".to_string(),
                format!("CODEX_SQLITE_HOME={path}/model/auth/state"),
                "-e".to_string(),
                format!("PATH={path}/model/harness/bin:{path}/model/harness/codex-path:/usr/bin:/bin"),
                format!(
                    "exec {path}/model/harness/bin/codex --config 'cli_auth_credentials_store=\"file\"' --config 'sqlite_home=\"{path}/model/auth/state\"' --config 'log_dir=\"{path}/model/auth/logs\"' --ask-for-approval never --sandbox danger-full-access"
                ),
            ]
        );
    }

    #[test]
    fn lifetime_rule() {
        assert_eq!(
            lifetime_argv(1000, 900).unwrap(),
            vec!["--property=RuntimeMaxSec=100".to_string()]
        );
        assert_eq!(
            lifetime_argv(900 + 43200, 900).unwrap(),
            vec!["--property=RuntimeMaxSec=43200".to_string()]
        );
        assert!(lifetime_argv(900, 900).is_err());
        assert!(lifetime_argv(899, 900).is_err());
        assert!(lifetime_argv(900 + 43201, 900).is_err());
    }

    #[test]
    fn emission_bytes_exact() {
        assert_eq!(
            pyemit::dumps(&status_object("id", "nm", 42, true, false, "ready")),
            r#"{"id":"id","name":"nm","created_at":42,"ready":true,"attached":false,"state":"ready"}"#
        );
        assert_eq!(
            pyemit::dumps(&ready_object(7, 8, 9)),
            r#"{"pid":7,"socket":[8,9]}"#
        );
        assert_eq!(
            pyemit::dumps(&reservation_object(100, &"c".repeat(64))),
            format!("{{\"expires\":100,\"scope\":\"{}\"}}", "c".repeat(64))
        );
        let record = validate_binding(&parse(&binding_doc(
            &good_account(),
            "7",
            "80",
            "24",
            "1700000000",
        )))
        .unwrap();
        assert_eq!(
            pyemit::dumps(&binding_object(&record)),
            r#"{"account":["op",1001,1002,"/home/op","/bin/bash"],"identity":7,"cols":80,"rows":24,"created_at":1700000000}"#
        );
        // Name docs are bare JSON strings.
        assert_eq!(pyemit::line(&JsonValue::Str("nm".to_string())), b"\"nm\"\n");
    }

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
}
