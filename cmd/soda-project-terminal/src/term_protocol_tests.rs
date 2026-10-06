use super::*;
use crate::pyemit;
use crate::term::term_binding_tests::{binding_doc, good_account, parse, sample};
use crate::term_attach::lifetime_argv;
use crate::term_binding::{binding_object, reservation_object, validate_binding};
use crate::term_create::systemd_run_argv;
use crate::term_paths::PROGRAM;
use crate::term_prepare::tmux_new_session_args;
use crate::term_status::{ready_object, status_object};

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
