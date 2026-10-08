use super::*;
use std::os::unix::fs::PermissionsExt;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};

static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

/// Unique per-test root with `accounts`/`keys` dirs (tests run in
/// parallel threads). Marker tests stay inside it; one separate NSS test
/// performs read-only lookup of the current process account/groups.
fn test_root(tag: &str) -> PathBuf {
    let n = TEST_SEQ.fetch_add(1, Ordering::Relaxed);
    let dir = std::env::temp_dir().join(format!(
        "soda-project-account-{tag}-{}-{n}",
        std::process::id()
    ));
    std::fs::create_dir_all(dir.join("accounts")).expect("accounts dir");
    std::fs::create_dir_all(dir.join("keys")).expect("keys dir");
    std::fs::set_permissions(dir.join("accounts"), std::fs::Permissions::from_mode(0o700))
        .expect("accounts mode");
    std::fs::set_permissions(dir.join("keys"), std::fs::Permissions::from_mode(0o755))
        .expect("keys mode");
    dir
}

fn request(login: &str, identity: i64, admin: bool, keys: &[&str]) -> Request {
    Request {
        login: login.to_string(),
        identity,
        admin,
        keys: keys.iter().map(ToString::to_string).collect(),
    }
}

/// `Config::test_root` with an in-process useradd/usermod double:
/// records argv, simulates home creation plus the passwd entry.
fn direct_config(root: &Path, log: &Arc<Mutex<Vec<Vec<String>>>>) -> Config {
    let mut config = Config::test_root(root);
    let home_base = root.join("home");
    std::fs::create_dir_all(&home_base).expect("home base");
    let passwd = root.join("passwd");
    let log = Arc::clone(log);
    config.run_command = Box::new(move |argv| {
        log.lock().expect("log").push(argv.to_vec());
        if argv[0] == "useradd" {
            let login = argv.last().expect("login argv").clone();
            let home = home_base.join(&login);
            std::fs::create_dir_all(&home).expect("fake home");
            let mut file = std::fs::OpenOptions::new()
                .create(true)
                .append(true)
                .open(&passwd)
                .expect("fake passwd");
            writeln!(file, "{login}:{}", home.display()).expect("passwd line");
        } else if argv[0] != "usermod" {
            panic!("unexpected native command: {argv:?}");
        }
        Ok(())
    });
    config
}

#[test]
fn login_matrix() {
    for login in ["op", "a", "a0_-b", "alice", &format!("a{}", "b".repeat(30))] {
        assert!(valid_login(login), "{login:?}");
    }
    for login in [
        "",
        "Root",
        "ALICE",
        "0abc",
        "-abc",
        "_abc",
        "ab.cd",
        "ab/cd",
        "ab cd",
        "alice\n",
        "éclair",
        &format!("a{}", "b".repeat(31)),
    ] {
        assert!(!valid_login(login), "{login:?}");
    }
    // `root` passes the character class; the refusal is a separate gate.
    assert!(valid_login("root"));
}

/// Key oracle baked against CPython `re.fullmatch` plus the
/// length/strip pre-checks (see the port notes for the probe matrix).
#[test]
fn key_oracle() {
    let valid = [
        "ssh-ed25519 YWJj",
        "ssh-ed25519 YWJj\n",
        "ssh-ed25519 YWJj\r\n",
        "ssh-ed25519 YWJj ",
        "ssh-ed25519 YWJj  \t ",
        "ssh-rsa AAAA",
        "ecdsa-sha2-nistp256 AAAA",
        "sk-ssh-ed25519@openssh.com AAAA",
        "sk-ecdsa-sha2-nistp256@openssh.com AAAA",
        "ssh-x AAAA\u{1c}",
        "ssh-x AAAA\u{85}",
        "ssh-x AAAA\u{a0}",
        "ssh-x AAAA\u{2003}",
        "ssh-x AAAA\n\n",
        "ecdsa-x AAAA",
        "sk-x AAAA",
        "sk-foo.bar@x AAAA",
        "sk-a_b-c.d@e AAAA",
        "ssh-x +/==",
    ];
    for key in valid {
        assert!(valid_key(key), "must accept {key:?}");
    }
    // Deliberate fail-closed deviation: the retired Python admitted
    // non-ASCII `\w` heads, but SSH key types are ASCII by protocol
    // (sshd would reject these), so the port refuses them rather
    // than ship a regex engine in a root guest binary.
    for key in [
        "ssh-é AAAA",
        "ssh-½ AAAA",
        "ssh-² AAAA",
        "ssh-Ⅷ AAAA",
        "ssh-中 AAAA",
    ] {
        assert!(!valid_key(key), "must refuse exotic {key:?}");
    }
    let invalid = [
        "PRIVATE KEY",
        "ssh-ed25519",
        "ssh-ed25519 ",
        "ssh-ed25519 YW Jj",
        "ssh-ed25519 YWJj\nssh-ed25519 ZGVm",
        "SSH-ED25519 YWJj",
        "ssh_YWJj AAAA",
        "ssh-a\u{301} AAAA",
        "ssh-a\u{200d} AAAA",
        "ssh-x \u{a0}AAAA",
        " ssh-x AAAA",
        "sk- AAAA",
        "ssh- AAAA",
        "ssh-x",
        "xssh-x AAAA",
        "ssh-foo.bar AAAA",
        "ecdsa-a.b AAAA",
        "ssh-ְ AAAA",
        "",
        " ",
        "ssh-x AAAA\u{200b}",
        "ssh-xAAAA",
        "ssh-x AA\nAA",
        "ssh-x é",
    ];
    for key in invalid {
        assert!(!valid_key(key), "must refuse {key:?}");
    }
}

#[test]
fn key_length_counts_characters_not_bytes() {
    // Exactly at the character limit -- and a well-formed key
    // otherwise.
    let edge = format!("ssh-{} A", "a".repeat(16384 - 6));
    assert_eq!(edge.chars().count(), 16384);
    assert!(valid_key(&edge));
    let over = format!("ssh-{} A", "a".repeat(16384 - 5));
    assert_eq!(over.chars().count(), 16385);
    assert!(!valid_key(&over));
    assert!(!valid_key(&"a".repeat(16385)));
}

fn parse_doc(doc: &str) -> Option<Request> {
    parse_request(doc)
}

#[test]
fn request_shape_matrix() {
    let good = parse_doc(r#"{"login":"alice","identity":1,"admin":false,"keys":[]}"#);
    assert_eq!(
        good,
        Some(Request {
            login: "alice".to_string(),
            identity: 1,
            admin: false,
            keys: vec![],
        })
    );
    // Field order is irrelevant; the set is what matters.
    assert!(parse_doc(r#"{"keys":[],"admin":true,"identity":7,"login":"op"}"#).is_some());
    // Duplicate keys collapse last-wins, like `json.loads`.
    let dup = parse_doc(r#"{"login":"alice","identity":1,"admin":false,"keys":[],"login":"op"}"#)
        .expect("dup login wins");
    assert_eq!(dup.login, "op");
    let overwritten_unrepresentable =
        parse_doc(r#"{"login":"op","identity":1e400,"admin":true,"keys":[],"identity":7}"#)
            .expect("only the final duplicate is typed");
    assert_eq!(overwritten_unrepresentable.identity, 7);
    assert!(
        parse_doc(r#"{"login":"op","identity":7,"admin":true,"keys":[],"identity":1e400}"#)
            .is_none()
    );
    for doc in [
        r#"{"login":"alice","identity":1,"admin":false}"#,
        r#"{"login":"alice","identity":1,"admin":false,"keys":[],"extra":0}"#,
        r#"{}"#,
        r#"[]"#,
        r#"null"#,
        r#"42"#,
        "\"\"",
        r#"{"login":"root","identity":1,"admin":false,"keys":[]}"#,
        r#"{"login":"Op","identity":1,"admin":false,"keys":[]}"#,
        r#"{"login":"","identity":1,"admin":false,"keys":[]}"#,
        r#"{"login":7,"identity":1,"admin":false,"keys":[]}"#,
    ] {
        assert!(parse_doc(doc).is_none(), "must refuse {doc}");
    }
}

#[test]
fn privilege_request_is_exact_and_bounded_by_identity_fields() {
    let good = parse_privilege_request(r#"{"login":"alice","identity":17}"#);
    assert_eq!(
        good,
        Some(PrivilegeRequest {
            login: "alice".to_string(),
            identity: 17,
        })
    );
    for invalid in [
        r#"{"login":"alice","identity":0}"#,
        r#"{"login":"root","identity":17}"#,
        r#"{"login":"alice","identity":17,"admin":true}"#,
        r#"{"login":"alice","identity":17,"identity":18}"#,
        r#"{"login":"alice","identity":1.0}"#,
        r#"{"login":"alice","identity":9223372036854775808}"#,
        "[]",
        "null",
    ] {
        assert!(
            parse_privilege_request(invalid).is_none(),
            "accepted {invalid}"
        );
    }
}

fn write_status_marker(root: &Path, name: &str, contents: &[u8], mode: u32) -> PathBuf {
    let marker = root.join("accounts").join(name);
    std::fs::write(&marker, contents).expect("write status marker");
    std::fs::set_permissions(&marker, std::fs::Permissions::from_mode(mode))
        .expect("set marker mode");
    marker
}

#[test]
fn privilege_marker_is_bound_to_open_managed_directory_and_exact_identity() {
    let root = test_root("status-marker");
    let config = Config::test_root(&root);
    let marker = write_status_marker(&root, "alice", b"17", 0o600);
    confirm_account_marker(&config, "alice", 17).expect("exact managed identity");
    assert!(confirm_account_marker(&config, "alice", 18).is_err());

    // Exercise the marker-owner check independently from the directory-owner
    // check while using a real root-owned-by-test-process inode.
    let directory = open_accounts_directory(&config).expect("open managed directory");
    assert!(
        read_account_marker(&directory, config.expect_uid.wrapping_add(1), "alice", 17).is_err()
    );

    std::fs::set_permissions(&marker, std::fs::Permissions::from_mode(0o640))
        .expect("make marker group-readable");
    assert!(confirm_account_marker(&config, "alice", 17).is_err());
    std::fs::set_permissions(&marker, std::fs::Permissions::from_mode(0o600))
        .expect("restore marker mode");

    let linked = root.join("linked-marker");
    std::fs::hard_link(&marker, &linked).expect("hard-link marker");
    assert!(confirm_account_marker(&config, "alice", 17).is_err());
    std::fs::remove_file(&linked).expect("remove marker hard link");

    std::fs::remove_file(&marker).expect("remove marker");
    let target = root.join("target-marker");
    std::fs::write(&target, b"17").expect("write symlink target");
    std::fs::set_permissions(&target, std::fs::Permissions::from_mode(0o600))
        .expect("set symlink target mode");
    std::os::unix::fs::symlink(&target, &marker).expect("symlink marker");
    assert!(confirm_account_marker(&config, "alice", 17).is_err());

    std::fs::remove_file(&marker).expect("remove symlink marker");
    write_status_marker(&root, "alice", &vec![b'1'; 65], 0o600);
    assert!(confirm_account_marker(&config, "alice", 17).is_err());
}

#[test]
fn privilege_marker_refuses_unsafe_accounts_directory() {
    let root = test_root("status-directory");
    let config = Config::test_root(&root);
    write_status_marker(&root, "alice", b"17", 0o600);
    std::fs::set_permissions(
        root.join("accounts"),
        std::fs::Permissions::from_mode(0o777),
    )
    .expect("make accounts directory writable");
    assert!(confirm_account_marker(&config, "alice", 17).is_err());
}

#[test]
fn privilege_decision_and_group_count_fail_closed() {
    let wheel = 10 as libc::gid_t;
    let ordinary = 1000 as libc::uid_t;
    assert!(is_project_administrator(0, wheel, &[]));
    assert!(is_project_administrator(ordinary, wheel, &[1000, wheel]));
    assert!(!is_project_administrator(ordinary, wheel, &[1000]));
    // Removing wheel from the current NSS result revokes the positive result.
    assert!(!is_project_administrator(ordinary, wheel, &[1000, 1001]));

    assert_eq!(checked_native_group_count(2, 2, 2).unwrap(), 2);
    for (result, count, capacity) in [(-1, 300, 256), (0, 0, 256), (0, 257, 256)] {
        assert!(checked_native_group_count(result, count, capacity).is_err());
    }
    let limit = native_group_limit().expect("native group limit");
    assert!((1..=LINUX_GROUP_HARD_LIMIT).contains(&limit));
}

#[test]
fn current_process_passwd_and_groups_are_resolved_through_nss() {
    let uid = unsafe { libc::geteuid() };
    let mut entry: libc::passwd = unsafe { std::mem::zeroed() };
    let mut result: *mut libc::passwd = std::ptr::null_mut();
    let mut buffer = vec![0u8; NSS_BUFFER_LIMIT];
    let rc = unsafe {
        libc::getpwuid_r(
            uid,
            &mut entry,
            buffer.as_mut_ptr() as *mut libc::c_char,
            buffer.len(),
            &mut result,
        )
    };
    assert_eq!(rc, 0);
    assert!(!result.is_null());
    assert!(!entry.pw_name.is_null());
    let login = unsafe { CStr::from_ptr(entry.pw_name) }
        .to_str()
        .expect("native login")
        .to_string();
    let account = lookup_native_account(&login).expect("lookup current passwd account");
    assert_eq!(account.uid, uid);
    let groups = lookup_native_groups(&login, account.primary_gid)
        .expect("lookup current NSS supplementary groups");
    assert!(groups.contains(&account.primary_gid));
    if let Ok(wheel) = lookup_wheel_gid() {
        let expected = groups.contains(&wheel);
        assert_eq!(
            is_project_administrator(account.uid, wheel, &groups),
            uid == 0 || expected
        );
    }
}

#[test]
fn request_scalar_matrix() {
    assert!(
        parse_doc(r#"{"login":"op","identity":9223372036854775807,"admin":true,"keys":[]}"#)
            .is_some()
    );
    for doc in [
        r#"{"login":"op","identity":0,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":-0,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":-1,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":9223372036854775808,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":99999999999999999999999999,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":1.0,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":1e3,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":true,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":"1","admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":null,"admin":false,"keys":[]}"#,
        r#"{"login":"op","identity":1,"admin":1,"keys":[]}"#,
        r#"{"login":"op","identity":1,"admin":"true","keys":[]}"#,
        r#"{"login":"op","identity":1,"admin":null,"keys":[]}"#,
        r#"{"login":"op","identity":1,"admin":false,"keys":null}"#,
        r#"{"login":"op","identity":1,"admin":false,"keys":"ssh-x AAAA"}"#,
        r#"{"login":"op","identity":1,"admin":false,"keys":[42]}"#,
        r#"{"login":"op","identity":1,"admin":false,"keys":[null]}"#,
        r#"{"login":"op","identity":1,"admin":false,"keys":["PRIVATE KEY"]}"#,
    ] {
        assert!(parse_doc(doc).is_none(), "must refuse {doc}");
    }
    // 32 keys fit; 33 do not.
    let many: Vec<String> = (0..33).map(|_| r#""ssh-x AAAA""#.to_string()).collect();
    assert!(parse_doc(&format!(
        "{{\"login\":\"op\",\"identity\":1,\"admin\":false,\"keys\":[{}]}}",
        many[..32].join(",")
    ))
    .is_some());
    assert!(parse_doc(&format!(
        "{{\"login\":\"op\",\"identity\":1,\"admin\":false,\"keys\":[{}]}}",
        many.join(",")
    ))
    .is_none());
}

fn mode_of(path: &Path) -> u32 {
    std::fs::metadata(path).expect("meta").mode() & 0o777
}

#[test]
fn provisions_locked_home_marker_shared_empty_keyfile() {
    let root = test_root("provisions");
    let log = Arc::new(Mutex::new(Vec::new()));
    let config = direct_config(&root, &log);
    let response = provision(&config, &request("alice", 1, false, &[])).expect("provision");
    assert_eq!(
        response,
        Response {
            login: "alice".to_string(),
            identity: 1,
        }
    );
    assert_eq!(std::fs::read(root.join("keys/alice")).expect("keys"), b"");
    assert_eq!(
        std::fs::read(root.join("accounts/alice")).expect("marker"),
        b"1"
    );
    assert_eq!(mode_of(&root.join("accounts/alice")), 0o600);
    assert_eq!(mode_of(&root.join("keys/alice")), 0o644);
    assert_eq!(
        std::fs::read_link(root.join("home/alice/shared")).expect("shared"),
        Path::new("/srv/project/shared")
    );
    let log = log.lock().expect("log");
    assert_eq!(log.len(), 1);
    assert_eq!(
        log[0],
        [
            "useradd",
            "--create-home",
            "--shell",
            "/bin/bash",
            "--password",
            "!",
            "--groups",
            "soda-project",
            "alice"
        ]
    );
}

#[test]
fn selected_keys_rerun_is_idempotent() {
    let root = test_root("idempotent");
    let log = Arc::new(Mutex::new(Vec::new()));
    let config = direct_config(&root, &log);
    let req = request("alice", 1, true, &["ssh-ed25519 YWJj\n"]);
    provision(&config, &req).expect("first");
    assert_eq!(
        std::fs::read(root.join("keys/alice")).expect("keys"),
        b"ssh-ed25519 YWJj\n"
    );
    assert_eq!(
        log.lock().expect("log").last().expect("last").as_slice(),
        ["usermod", "--append", "--groups", "wheel", "alice"]
    );
    std::fs::write(root.join("home/alice/work"), "later work").expect("work");
    let before = std::fs::metadata(root.join("keys/alice"))
        .expect("meta")
        .ino();
    provision(&config, &req).expect("second");
    assert_eq!(
        std::fs::metadata(root.join("keys/alice"))
            .expect("meta")
            .ino(),
        before
    );
    assert_eq!(
        std::fs::read_to_string(root.join("home/alice/work")).expect("work"),
        "later work"
    );
    assert_eq!(
        log.lock()
            .expect("log")
            .iter()
            .filter(|argv| argv[0] == "useradd")
            .count(),
        1
    );
}

#[test]
fn join_never_applies_keys_and_preserves_drift() {
    let root = test_root("join");
    let log = Arc::new(Mutex::new(Vec::new()));
    let config = direct_config(&root, &log);
    provision(&config, &request("alice", 1, false, &["ssh-ed25519 YWJj"])).expect("first");
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    assert_eq!(
        std::fs::read(root.join("keys/alice")).expect("keys"),
        b"ssh-ed25519 YWJj\n"
    );
    assert_eq!(log.lock().expect("log").len(), 1);
    // Drifted marker refuses without being touched.
    std::fs::write(root.join("accounts/alice"), "2").expect("drift");
    assert!(provision(&config, &request("alice", 1, false, &["ssh-ed25519 YWJj"])).is_err());
    assert_eq!(
        std::fs::read(root.join("accounts/alice")).expect("marker"),
        b"2"
    );
}

#[test]
fn occupied_inputs_and_unassociated_users_refuse() {
    let root = test_root("occupied");
    let log = Arc::new(Mutex::new(Vec::new()));
    let config = direct_config(&root, &log);
    std::os::unix::fs::symlink(root.join("absent"), root.join("keys/alice")).expect("dangle");
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    assert!(log.lock().expect("log").is_empty());
    std::fs::remove_file(root.join("keys/alice")).expect("unlink");
    // Passwd entry without a marker: unassociated, refused silently.
    std::fs::create_dir_all(root.join("home/alice")).expect("home");
    std::fs::write(
        root.join("passwd"),
        format!("alice:{}/home/alice\n", root.display()),
    )
    .expect("passwd");
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    assert!(log.lock().expect("log").is_empty());
}

#[test]
fn key_symlink_is_not_followed() {
    let root = test_root("symlink");
    let log = Arc::new(Mutex::new(Vec::new()));
    let config = direct_config(&root, &log);
    provision(&config, &request("alice", 1, false, &[])).expect("first");
    std::fs::write(root.join("other"), "preserve").expect("other");
    std::fs::remove_file(root.join("keys/alice")).expect("unlink");
    std::os::unix::fs::symlink(root.join("other"), root.join("keys/alice")).expect("link");
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    assert_eq!(
        std::fs::read(root.join("other")).expect("other"),
        b"preserve"
    );
}

#[test]
fn existing_account_binding_matrix() {
    let root = test_root("binding");
    let log = Arc::new(Mutex::new(Vec::new()));
    let config = direct_config(&root, &log);
    std::fs::create_dir_all(root.join("home/alice")).expect("home");
    std::fs::write(
        root.join("passwd"),
        format!("alice:{}/home/alice\n", root.display()),
    )
    .expect("passwd");
    // Missing marker refuses.
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    // Oversized marker refuses.
    std::fs::write(root.join("accounts/alice"), vec![b'1'; 65]).expect("marker");
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    // Wrong identity refuses.
    std::fs::write(root.join("accounts/alice"), "2").expect("marker");
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    // Matching marker provisions the (empty) keyfile.
    std::fs::write(root.join("accounts/alice"), "1").expect("marker");
    provision(&config, &request("alice", 1, false, &[])).expect("adopt");
    assert_eq!(std::fs::read(root.join("keys/alice")).expect("keys"), b"");
    assert!(log.lock().expect("log").is_empty());
}

#[test]
fn lock_contention_refuses_before_effects() {
    let root = test_root("contend");
    let log = Arc::new(Mutex::new(Vec::new()));
    let config = direct_config(&root, &log);
    let held = std::fs::File::open(root.join("keys")).expect("keys open");
    let rc = unsafe { libc::flock(held.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
    assert_eq!(rc, 0);
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    assert!(log.lock().expect("log").is_empty());
    assert!(std::fs::read_dir(root.join("keys"))
        .expect("readdir")
        .next()
        .is_none());
    assert!(std::fs::read_dir(root.join("accounts"))
        .expect("readdir")
        .next()
        .is_none());
    drop(held);
    provision(&config, &request("alice", 1, false, &[])).expect("after release");
}

#[test]
fn fsync_order_holds_lock_and_orders_durably() {
    let root = test_root("fsyncorder");
    let log = Arc::new(Mutex::new(Vec::new()));
    let mut config = direct_config(&root, &log);
    let events = Arc::new(Mutex::new(Vec::new()));
    let contents = Arc::new(Mutex::new(Vec::new()));
    let keys_path = root.join("keys");
    let accounts_path = root.join("accounts");
    let marker_path = root.join("accounts/alice");
    let keyfile_path = root.join("keys/alice");
    let events_in = Arc::clone(&events);
    let contents_in = Arc::clone(&contents);
    config.sync_file = Box::new(move |file| {
        // Every sync runs under the directory lock.
        let probe = std::fs::File::open(&keys_path).expect("probe open");
        let rc = unsafe { libc::flock(probe.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
        assert_ne!(rc, 0, "sync ran without the directory lock");
        assert_eq!(
            std::io::Error::last_os_error().raw_os_error(),
            Some(libc::EWOULDBLOCK)
        );
        let synced = file.metadata().expect("synced meta").clone();
        let at = |path: &Path| {
            std::fs::metadata(path)
                .ok()
                .filter(|meta| (meta.dev(), meta.ino()) == (synced.dev(), synced.ino()))
                .is_some()
        };
        if synced.mode() & libc::S_IFMT == libc::S_IFREG {
            let mut seen = None;
            for (path, want, name) in [
                (&marker_path, b"1".as_slice(), "marker"),
                (&keyfile_path, b"ssh-ed25519 YWJj\n".as_slice(), "keyfile"),
            ] {
                if at(path) {
                    contents_in
                        .lock()
                        .expect("contents")
                        .push(std::fs::read(path).expect("read") == want);
                    seen = Some(name);
                    break;
                }
            }
            events_in
                .lock()
                .expect("events")
                .push(seen.expect("synced an unknown file").to_string());
        } else if at(&accounts_path) {
            events_in
                .lock()
                .expect("events")
                .push("markers".to_string());
        } else if at(&keys_path) {
            events_in.lock().expect("events").push("keys".to_string());
        } else {
            panic!("synced an unknown directory");
        }
        file.sync_all()
    });
    provision(&config, &request("alice", 1, true, &["ssh-ed25519 YWJj"])).expect("provision");
    assert_eq!(
        *events.lock().expect("events"),
        ["marker", "markers", "keyfile", "keys"]
    );
    assert_eq!(*contents.lock().expect("contents"), [true, true]);
    assert_eq!(log.lock().expect("log").last().expect("last")[0], "usermod");
}

#[test]
fn failed_sync_releases_lock_and_keeps_partial_files() {
    let root = test_root("failsync");
    let log = Arc::new(Mutex::new(Vec::new()));
    let mut config = direct_config(&root, &log);
    config.fail_sync = true;
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    assert_eq!(
        std::fs::read(root.join("accounts/alice")).expect("marker"),
        b"1"
    );
    assert!(root.join("home/alice").is_dir());
    let relock = std::fs::File::open(root.join("keys")).expect("keys open");
    let rc = unsafe { libc::flock(relock.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
    assert_eq!(rc, 0, "lock must not strand a failed provisioning");
}

#[test]
fn unsafe_directories_refuse() {
    let root = test_root("unsafedir");
    let log = Arc::new(Mutex::new(Vec::new()));
    let config = direct_config(&root, &log);
    std::fs::set_permissions(root.join("keys"), std::fs::Permissions::from_mode(0o775))
        .expect("relax keys");
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    std::fs::set_permissions(root.join("keys"), std::fs::Permissions::from_mode(0o755))
        .expect("restore keys");
    std::fs::set_permissions(
        root.join("accounts"),
        std::fs::Permissions::from_mode(0o775),
    )
    .expect("relax accounts");
    assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
    assert!(log.lock().expect("log").is_empty());
}

#[test]
fn unprivileged_production_gate() {
    // The fixed guest config requires root before touching anything.
    if unsafe { libc::geteuid() } != 0 {
        assert!(provision(&Config::production(), &request("alice", 1, false, &[])).is_err());
    }
}
