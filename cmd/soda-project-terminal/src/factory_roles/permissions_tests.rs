use super::*;
use std::os::unix::fs::PermissionsExt;

fn mode_of(path: &Path) -> u32 {
    std::fs::symlink_metadata(path)
        .unwrap()
        .permissions()
        .mode()
        & 0o777
}

/// Real `/bin/bash` append attempt. Returns the observed exit code; the
/// kernel (mode bits), not the test, decides success.
fn bash_append_exit(target: &Path, cwd: &Path) -> i32 {
    let output = std::process::Command::new("/bin/bash")
        .arg("-c")
        .arg("printf 'x' >> \"$T_TARGET\"")
        .env("T_TARGET", target)
        .current_dir(cwd)
        .output()
        .expect("spawn bash");
    output.status.code().unwrap_or(127)
}

fn bash_mkdir_exit(dir: &Path, cwd: &Path) -> i32 {
    let output = std::process::Command::new("/bin/bash")
        .arg("-c")
        .arg("mkdir \"$T_TARGET\"")
        .env("T_TARGET", dir)
        .current_dir(cwd)
        .output()
        .expect("spawn bash");
    output.status.code().unwrap_or(127)
}

/// Real `env` dump of a worker environment for leak scanning.
fn worker_env_output(env: &[(String, String)]) -> String {
    let mut command = std::process::Command::new("/usr/bin/env");
    command.env_clear();
    for (key, value) in env {
        command.env(key, value);
    }
    let output = command.output().expect("spawn env");
    assert!(output.status.success());
    String::from_utf8(output.stdout).expect("env is utf8")
}

fn spec(model: &str, reasoning: &str, contributor: bool, lease: bool) -> ChildSpec {
    ChildSpec {
        model: model.to_string(),
        reasoning: reasoning.to_string(),
        contributor,
        account: "soda-coder".to_string(),
        deadline: "2026-10-17T01:00:00Z".to_string(),
        extra_lease: lease,
    }
}

#[test]
fn profile_catalog_maps_seven_profiles_to_two_actors() {
    assert_eq!(PROFILES.len(), 7);
    assert_eq!(READONLY_PROFILES.len(), 5);
    assert_eq!(WRITER_PROFILES.len(), 2);
    for profile in PROFILES {
        let role = profile_role(profile).expect("known profile maps");
        assert!(role == "soda-coder" || role == "soda-reviewer");
    }
    assert_eq!(profile_role("builder").unwrap(), "soda-coder");
    assert_eq!(profile_role("fixer").unwrap(), "soda-coder");
    assert_eq!(profile_role("inspector").unwrap(), "soda-reviewer");
    assert!(profile_role("root").is_err());
    assert!(is_readonly("strategist").unwrap());
    assert!(!is_readonly("builder").unwrap());
}

#[test]
fn parent_only_tools_refuse_workers_but_admit_parents() {
    for tool in READ_TOOLS {
        admit_tool(tool, true).expect("worker reads");
        admit_tool(tool, false).expect("parent reads");
    }
    for tool in PARENT_ONLY_TOOLS {
        assert!(admit_tool(tool, true).is_err(), "{tool}: worker admitted");
        admit_tool(tool, false).expect("parent mutation admitted");
    }
    assert!(admit_tool("factory.delete_everything", false).is_err());
    assert!(admit_tool("factory.delete_everything", true).is_err());
}

/// T50 native: every readonly profile's shell write fails against the
/// staged read-only tree while scratch writes succeed.
#[test]
fn readonly_shell_writes_fail_native() {
    for profile in READONLY_PROFILES {
        let scratch = crate::testutil::Scratch::fresh();
        let src = scratch.root.join("src");
        std::fs::create_dir_all(src.join("sub")).unwrap();
        std::fs::write(src.join("main.rs"), "fn main() {}\n").unwrap();
        let work = scratch.root.join("scratch");
        std::fs::create_dir_all(&work).unwrap();
        stage_readonly_tree(&src).expect("stage readonly");
        assert_eq!(mode_of(&src), 0o555);
        assert_eq!(mode_of(&src.join("main.rs")), 0o444);

        let target = src.join("main.rs");
        assert!(classify_write(profile, &target, &src, &work).is_err());
        let before = std::fs::read(&target).unwrap();
        assert_ne!(bash_append_exit(&target, &scratch.root), 0);
        assert_eq!(std::fs::read(&target).unwrap(), before);
        assert_ne!(bash_mkdir_exit(&src.join("newdir"), &scratch.root), 0);
        assert!(!src.join("newdir").exists());

        let allowed = work.join("notes.txt");
        classify_write(profile, &allowed, &src, &work).expect("scratch");
        assert_eq!(bash_append_exit(&allowed, &scratch.root), 0);
        assert_eq!(std::fs::read(&allowed).unwrap(), b"x");
    }
}

/// T50 native: Builder/Fixer shell writes to assigned source succeed but
/// every protected path fails, with policy and kernel outcomes agreeing.
#[test]
fn writer_protected_writes_fail_but_source_succeeds_native() {
    for profile in WRITER_PROFILES {
        let scratch = crate::testutil::Scratch::fresh();
        let assigned = scratch.root.join("checkout");
        std::fs::create_dir_all(assigned.join("src")).unwrap();
        let good = assigned.join("src").join("lib.rs");
        std::fs::write(&good, "pub fn f() {}\n").unwrap();
        let work = scratch.root.join("scratch");
        std::fs::create_dir_all(&work).unwrap();

        // Protected fixtures staged read-only, mirroring FW39 staging.
        let git_head = assigned.join(".git").join("HEAD");
        std::fs::create_dir_all(git_head.parent().unwrap()).unwrap();
        std::fs::write(&git_head, "ref: refs/heads/main\n").unwrap();
        let journal = assigned.join("journal.log");
        std::fs::write(&journal, "{\"seq\":1}\n").unwrap();
        let cred = assigned.join("credentials").join("token");
        std::fs::create_dir_all(cred.parent().unwrap()).unwrap();
        std::fs::write(&cred, "parent-secret-bytes\n").unwrap();
        let ipc = assigned.join("factory-ipc").join("mcp.sock");
        std::fs::create_dir_all(ipc.parent().unwrap()).unwrap();
        std::fs::write(&ipc, "socket\n").unwrap();
        let config = assigned.join("mcp.json");
        std::fs::write(&config, "{}\n").unwrap();
        let receipt = assigned.join("finished.json");
        std::fs::write(&receipt, "{}\n").unwrap();
        for path in [&git_head, &journal, &cred, &ipc, &config, &receipt] {
            std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o444)).unwrap();
        }

        classify_write(profile, &good, &assigned, &work).expect("source");
        assert_eq!(bash_append_exit(&good, &scratch.root), 0);

        for forbidden in [&git_head, &journal, &cred, &ipc, &config, &receipt] {
            assert!(
                classify_write(profile, forbidden, &assigned, &work).is_err(),
                "{profile} {}: policy admitted",
                forbidden.display()
            );
            let before = std::fs::read(forbidden).unwrap();
            assert_ne!(
                bash_append_exit(forbidden, &scratch.root),
                0,
                "{profile} {}: kernel admitted",
                forbidden.display()
            );
            assert_eq!(std::fs::read(forbidden).unwrap(), before);
        }

        let allowed = work.join("out.txt");
        classify_write(profile, &allowed, &assigned, &work).expect("scratch");
        assert_eq!(bash_append_exit(&allowed, &scratch.root), 0);
    }
}

/// T49 native: a real worker cannot invoke parent-only tools and never
/// receives a usable parent credential.
#[test]
fn worker_attenuation_holds_native() {
    // Parent-only submission refused for the worker binding, admitted for
    // the parent binding, across the whole mutation catalog.
    for tool in PARENT_ONLY_TOOLS {
        let worker = admit_tool(tool, true);
        assert!(worker.is_err(), "{tool}: worker admitted");
        admit_tool(tool, false).expect("parent admitted");
    }

    // The real worker environment carries no credential file and no
    // parent secret bytes.
    let parent_secret = "parent-capability-secret-abcdef";
    let env = crate::execution::child_environ(
        "/bin",
        "/worker-home",
        "",
        "soda-coder",
        "f0123456789abcdef01234567",
        "/snapshot",
    );
    assert!(!env.iter().any(|(k, _)| k == "SODA_CREDENTIAL_FILE"));
    assert!(!env_leaks_secret(&env, parent_secret));
    let dump = worker_env_output(&env);
    assert!(!dump.contains("SODA_CREDENTIAL_FILE"));
    assert!(!dump.contains(parent_secret));

    // Credential scope pins to the worker's own role: the parent scope
    // and the sibling role scope both refuse.
    let scratch = crate::testutil::Scratch::fresh();
    let creds = scratch.root.join("credentials");
    let own = credential_path_for(&creds, "soda-coder", "tok").unwrap();
    assert_eq!(own, creds.join("soda-coder").join("tok"));
    check_worker_credential("soda-coder", &own, &creds).expect("own scope");
    assert!(check_worker_credential("soda-reviewer", &own, &creds).is_err());
    let parent_path = creds.join("parent").join("capability");
    assert!(check_worker_credential("soda-coder", &parent_path, &creds).is_err());

    // The parent secret file itself is 0600: only its owner reads it.
    let secret_file = scratch.root.join("parent-secret");
    crate::fsx::write_new(secret_file.as_path(), parent_secret.as_bytes(), 0o600).unwrap();
    assert_eq!(mode_of(&secret_file), 0o600);
}

/// T49 data check: children retain the exact admitted identity with no
/// extra lease; any deviation refuses.
#[test]
fn child_inheritance_is_exact_with_no_extra_lease() {
    let parent = spec("muse-spark-1.3", "xhigh", true, false);
    check_inheritance(&parent, &spec("muse-spark-1.3", "xhigh", true, false)).expect("exact");
    assert!(check_inheritance(&parent, &spec("other-model", "xhigh", true, false)).is_err());
    assert!(check_inheritance(&parent, &spec("muse-spark-1.3", "low", true, false)).is_err());
    assert!(check_inheritance(&parent, &spec("muse-spark-1.3", "xhigh", false, false)).is_err());
    let mut moved_account = spec("muse-spark-1.3", "xhigh", true, false);
    moved_account.account = "soda-reviewer".to_string();
    assert!(check_inheritance(&parent, &moved_account).is_err());
    let mut moved_deadline = spec("muse-spark-1.3", "xhigh", true, false);
    moved_deadline.deadline = "2026-10-18T01:00:00Z".to_string();
    assert!(check_inheritance(&parent, &moved_deadline).is_err());
    assert!(check_inheritance(&parent, &spec("muse-spark-1.3", "xhigh", true, true)).is_err());
}

/// T50 native: forged journals and symlinks are rejected; genuine bytes
/// verify; journals are create-only.
#[test]
fn forged_journal_and_symlink_rejected_native() {
    use sha2::{Digest, Sha256};
    let scratch = crate::testutil::Scratch::fresh();
    let journal = scratch.root.join("events.journal");
    let genuine = b"{\"seq\":1,\"kind\":\"started\"}\n";
    crate::fsx::write_new(&journal, genuine, 0o644).unwrap();
    let mut hasher = Sha256::new();
    hasher.update(genuine);
    let digest = sha::hex(&hasher.finalize());
    assert_eq!(verify_journal(&journal, &digest).unwrap(), genuine);

    // Forged bytes under the genuine path refuse on digest.
    std::fs::write(&journal, b"{\"seq\":1,\"kind\":\"forged\"}\n").unwrap();
    assert!(verify_journal(&journal, &digest).is_err());

    // A symlinked journal path fails the O_NOFOLLOW open outright.
    let link = scratch.root.join("link.journal");
    std::os::unix::fs::symlink(&journal, &link).unwrap();
    assert!(open_journal_no_follow(&link).is_err());

    // Symlinked staging itself refuses before any mode change.
    assert!(stage_readonly_tree(&link).is_err());

    // Journals are create-only: appending over an existing record refuses.
    assert!(crate::fsx::write_new(&journal, b"{\"seq\":2}\n", 0o644).is_err());

    // Oversized journals refuse.
    let big = scratch.root.join("big.journal");
    crate::fsx::write_new(&big, &vec![b'j'; MAX_JOURNAL], 0o644).unwrap();
    assert!(open_journal_no_follow(&big).is_ok());
    std::fs::write(&big, vec![b'j'; MAX_JOURNAL + 1]).unwrap();
    assert!(open_journal_no_follow(&big).is_err());
}

/// T49/T50 mount and config guards: fixed read-only IPC socket-only mount
/// admits; broad, foreign, secret-exposing and read-write mounts refuse;
/// repo overrides refuse; digests must match exactly.
#[test]
fn mount_and_config_guards_hold() {
    let exact = [MountEntry {
        path: "factory-ipc".to_string(),
        readonly: true,
        exposes_secrets: false,
    }];
    check_mount(&exact).expect("exact IPC mount");
    let rooted = [MountEntry {
        path: "/run/soda/factory-ipc".to_string(),
        readonly: true,
        exposes_secrets: false,
    }];
    check_mount(&rooted).expect("rooted IPC mount");

    let rw = [MountEntry {
        path: "factory-ipc".to_string(),
        readonly: false,
        exposes_secrets: false,
    }];
    assert!(check_mount(&rw).is_err());
    let missing = [MountEntry {
        path: "scratch".to_string(),
        readonly: true,
        exposes_secrets: false,
    }];
    assert!(check_mount(&missing).is_err());
    for bad in [
        "/",
        "/var/lib/soda",
        "/var/lib/soda/extension.sock",
        "/run/soda/operator.sock",
        "/run/soda/broker.sock",
    ] {
        let entries = [
            MountEntry {
                path: bad.to_string(),
                readonly: true,
                exposes_secrets: false,
            },
            MountEntry {
                path: "factory-ipc".to_string(),
                readonly: true,
                exposes_secrets: false,
            },
        ];
        assert!(check_mount(&entries).is_err(), "{bad}: admitted");
    }
    let secret_mount = [
        MountEntry {
            path: "factory-ipc".to_string(),
            readonly: true,
            exposes_secrets: true,
        },
        MountEntry {
            path: "creds".to_string(),
            readonly: true,
            exposes_secrets: true,
        },
    ];
    assert!(check_mount(&secret_mount).is_err());

    // Repo-supplied MCP configuration never stages.
    let scratch = crate::testutil::Scratch::fresh();
    let repo_config = scratch.root.join("repo").join("mcp.json");
    assert!(repo_config_refused(&repo_config).is_err());

    // Managed staging is helper-owned read-only under the recorded digest.
    use sha2::{Digest, Sha256};
    let staged = scratch.root.join("managed-mcp.json");
    let bytes = b"{\"mcpServers\":{}}\n";
    stage_managed(&staged, bytes).expect("stage managed");
    assert_eq!(mode_of(&staged), 0o444);
    let mut hasher = Sha256::new();
    hasher.update(bytes);
    verify_config_digest(&sha::hex(&hasher.finalize()), bytes).expect("digest");
    assert!(verify_config_digest(&"0".repeat(64), bytes).is_err());
    assert_ne!(bash_append_exit(&staged, &scratch.root), 0);
    assert_eq!(std::fs::read(&staged).unwrap(), bytes);

    // Socket and relay pins stay shared with the FW33 precedent modules.
    assert_eq!(crate::mcp::IPC_DIR_NAME, "factory-ipc");
    assert_eq!(crate::mcp::SOCKET_NAME, "mcp.sock");
    assert!(crate::mcp::check_relay_argv(&["soda-project-terminal", "factory-mcp"]).is_ok());
    assert!(crate::mcp::check_relay_argv(&["evil-relay"]).is_err());
}

#[test]
fn worker_parent_completion_write_fails_native() {
    // The parent completion record is staged read-only: a real worker
    // shell cannot forge it, and the tool gate refuses the submission.
    let scratch = crate::testutil::Scratch::fresh();
    let finished = scratch.root.join("finished.json");
    crate::fsx::write_new(&finished, b"{\"setup_exit\": 0}\n", 0o444).unwrap();
    assert!(admit_tool("factory.submit_task_output", true).is_err());
    assert!(admit_tool("factory.finish_implementation", true).is_err());
    let before = std::fs::read(&finished).unwrap();
    assert_ne!(bash_append_exit(&finished, &scratch.root), 0);
    assert_eq!(std::fs::read(&finished).unwrap(), before);
}
