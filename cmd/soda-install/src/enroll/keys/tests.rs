use super::super::test_support::{temp_dir, test_uid, TEST_KEY, TEST_KEY_2, TEST_KEY_3};
use super::directory::open_dir_nofollow;
use super::*;
use std::io::Write as _;
use std::os::unix::fs::PermissionsExt;

#[test]
fn random_hex_entropy_failure_returns_without_issuing_a_value() {
    let mut calls = 0;
    let result = super::random_hex_with(32, |out| {
        calls += 1;
        out[0] = 1;
        Err(std::io::Error::other("injected entropy failure"))
    });
    assert!(result.is_err());
    assert_eq!(calls, 1);
}

fn make_ssh_dir(home: &str) -> String {
    let dir = format!("{home}/.ssh");
    std::fs::create_dir(&dir).unwrap();
    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700)).unwrap();
    dir
}

fn test_ctx() -> Ctx {
    Ctx::test().0
}

#[test]
fn preserves_authorized_keys() {
    let home = temp_dir();
    std::fs::set_permissions(&home.path, std::fs::Permissions::from_mode(0o700)).unwrap();
    let existing = format!("# existing native policy\nrestrict {TEST_KEY} original comment");
    let dir = make_ssh_dir(&home.path);
    let path = format!("{dir}/authorized_keys");
    std::fs::write(&path, &existing).unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
    append_enrollment_key(&test_ctx(), &home.path, TEST_KEY_2, test_uid()).unwrap();
    let got = std::fs::read(&path).unwrap();
    let mut want = existing.as_bytes().to_vec();
    want.push(b'\n');
    want.extend_from_slice(format!("{TEST_KEY_2}\n").as_bytes());
    assert_eq!(
        got, want,
        "existing authorized keys were not preserved exactly"
    );
    let mode = std::fs::symlink_metadata(&path)
        .unwrap()
        .permissions()
        .mode()
        & 0o777;
    assert_eq!(mode, 0o600, "native file permissions changed unsafely");
    // The restricted existing key matches as an adjacent pair: refused.
    assert!(append_enrollment_key(&test_ctx(), &home.path, TEST_KEY, test_uid()).is_err());
    // The just-imported key is a duplicate: refused.
    assert!(append_enrollment_key(&test_ctx(), &home.path, TEST_KEY_2, test_uid()).is_err());
    assert_eq!(
        std::fs::read(&path).unwrap(),
        got,
        "refusal changed existing keys"
    );
    assert_eq!(
        std::fs::read_dir(&dir).unwrap().count(),
        1,
        "temporary key files remained"
    );
}

#[test]
fn creates_authorized_keys() {
    let home = temp_dir();
    append_enrollment_key(&test_ctx(), &home.path, TEST_KEY, test_uid()).unwrap();
    for (name, mode) in [(".ssh", 0o700), (".ssh/authorized_keys", 0o600)] {
        let st = std::fs::symlink_metadata(format!("{}/{name}", home.path)).unwrap();
        assert_eq!(
            st.permissions().mode() & 0o777,
            mode,
            "wrong mode for {name}"
        );
        assert_eq!(std::os::unix::fs::MetadataExt::uid(&st), test_uid());
    }
}

#[test]
fn refuses_unsafe_key_paths() {
    for scenario in [
        "home-symlink",
        "ssh-symlink",
        "key-symlink",
        "key-hardlink",
        "key-directory",
        "key-writable",
        "ssh-writable",
        "home-writable",
        "oversized",
        "wrong-owner",
    ] {
        let root = temp_dir();
        let mut home = format!("{}/home", root.path);
        std::fs::create_dir(&home).unwrap();
        std::fs::set_permissions(&home, std::fs::Permissions::from_mode(0o700)).unwrap();
        let dir = format!("{home}/.ssh");
        std::fs::create_dir(&dir).unwrap();
        let path = format!("{dir}/authorized_keys");
        let sentinel = format!("{}/sentinel", root.path);
        let original = b"preserve this unrelated file\n";
        std::fs::write(&sentinel, original).unwrap();
        let mut uid = test_uid();
        match scenario {
            "home-symlink" => {
                let alias = format!("{}/alias", root.path);
                std::os::unix::fs::symlink(&home, &alias).unwrap();
                home = alias;
            }
            "ssh-symlink" => {
                std::fs::remove_dir(&dir).unwrap();
                std::os::unix::fs::symlink(&root.path, &dir).unwrap();
            }
            "key-symlink" => {
                std::os::unix::fs::symlink(&sentinel, &path).unwrap();
            }
            "key-hardlink" => {
                std::fs::hard_link(&sentinel, &path).unwrap();
            }
            "key-directory" => {
                std::fs::create_dir(&path).unwrap();
            }
            "key-writable" => {
                std::fs::write(&path, original).unwrap();
                std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o666)).unwrap();
            }
            "ssh-writable" => {
                std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o777)).unwrap();
            }
            "home-writable" => {
                std::fs::set_permissions(&home, std::fs::Permissions::from_mode(0o777)).unwrap();
            }
            "oversized" => {
                std::fs::write(&path, vec![b'x'; (1 << 20) + 1]).unwrap();
            }
            "wrong-owner" => {
                uid += 1;
            }
            _ => unreachable!(),
        }
        assert!(
            append_enrollment_key(&test_ctx(), &home, TEST_KEY, uid).is_err(),
            "unsafe key target accepted: {scenario}"
        );
        assert_eq!(
            std::fs::read(&sentinel).unwrap(),
            original,
            "unrelated file changed: {scenario}"
        );
    }
}

#[test]
fn cancelled_append_preserves_key() {
    let home = temp_dir();
    let (ctx, flag) = Ctx::test();
    flag.store(true, std::sync::atomic::Ordering::SeqCst);
    assert!(append_enrollment_key(&ctx, &home.path, TEST_KEY, test_uid()).is_err());
    assert!(matches!(
        std::fs::symlink_metadata(format!("{}/.ssh", home.path)),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound
    ));
}

#[test]
fn concurrent_writer_preserves_key() {
    let home = temp_dir();
    append_enrollment_key(&test_ctx(), &home.path, TEST_KEY, test_uid()).unwrap();
    let dir_fd = open_dir_nofollow(&format!("{}/.ssh", home.path)).unwrap();
    assert_eq!(
        unsafe { libc::flock(dir_fd.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) },
        0
    );
    assert!(append_enrollment_key(&test_ctx(), &home.path, TEST_KEY_2, test_uid()).is_err());
    assert_eq!(
        std::fs::read(format!("{}/.ssh/authorized_keys", home.path)).unwrap(),
        format!("{TEST_KEY}\n").as_bytes()
    );
}

#[test]
fn native_editor_race_preserves_newer_file() {
    for replacement in [true, false] {
        let home = temp_dir();
        let dir = make_ssh_dir(&home.path);
        let path = format!("{dir}/authorized_keys");
        let original = format!("{TEST_KEY} original\n");
        let newer = format!("{TEST_KEY_2} native editor without trailing newline");
        std::fs::write(&path, &original).unwrap();
        let calls = std::cell::Cell::new(0);
        let write = |mut file: &File, data: &[u8]| -> std::io::Result<usize> {
            calls.set(calls.get() + 1);
            // This is the former Fstatat-to-Renameat race window: a native
            // editor acts after Soda's last pre-write pathname check.
            let target = if replacement {
                format!("{dir}/native-editor-new-file")
            } else {
                path.clone()
            };
            std::fs::write(&target, &newer).unwrap();
            if replacement {
                std::fs::rename(&target, &path).unwrap();
            }
            file.write(data)
        };
        let err = append_enrollment_key_with_writer(
            &test_ctx(),
            &home.path,
            TEST_KEY_3,
            test_uid(),
            &write,
        )
        .unwrap_err();
        assert_eq!(
            err,
            Error::EnrollUncertain,
            "race not reported as uncertain: {err}"
        );
        assert_eq!(calls.get(), 1);
        let mut want = newer.clone();
        if !replacement {
            want.push_str(&format!("\n{TEST_KEY_3}\n"));
        }
        assert_eq!(
            std::fs::read(&path).unwrap(),
            want.as_bytes(),
            "Soda overwrote or merged into the native editor's newer data"
        );
    }
}

#[test]
fn concurrent_creation_is_never_replaced() {
    let home = temp_dir();
    let path = format!("{}/.ssh/authorized_keys", home.path);
    let native = format!("{TEST_KEY} created by native editor\n");
    let write = |mut file: &File, data: &[u8]| -> std::io::Result<usize> {
        std::fs::write(&path, &native).unwrap();
        file.write(data)
    };
    assert!(append_enrollment_key_with_writer(
        &test_ctx(),
        &home.path,
        TEST_KEY_2,
        test_uid(),
        &write
    )
    .is_err());
    assert_eq!(std::fs::read(&path).unwrap(), native.as_bytes());
    assert_eq!(
        std::fs::read_dir(format!("{}/.ssh", home.path))
            .unwrap()
            .count(),
        1,
        "unpublished temporary key file remained"
    );
}

#[test]
fn partial_append_does_not_roll_back_native_data() {
    let home = temp_dir();
    let dir = make_ssh_dir(&home.path);
    let path = format!("{dir}/authorized_keys");
    let original = format!("{TEST_KEY} preserved original\n");
    std::fs::write(&path, &original).unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o640)).unwrap();
    let before = std::fs::symlink_metadata(&path).unwrap();
    let partial_len = std::cell::Cell::new(0usize);
    let write = |mut file: &File, data: &[u8]| -> std::io::Result<usize> {
        let half = &data[..data.len() / 2];
        partial_len.set(half.len());
        let n = file.write(half)?;
        assert_eq!(n, half.len());
        Err(std::io::Error::other("synthetic partial-write failure"))
    };
    let err =
        append_enrollment_key_with_writer(&test_ctx(), &home.path, TEST_KEY_2, test_uid(), &write)
            .unwrap_err();
    assert_eq!(
        err,
        Error::EnrollUncertain,
        "partial append not uncertain: {err}"
    );
    let mut want = original.as_bytes().to_vec();
    let addition = format!("\n{TEST_KEY_2}\n");
    want.extend_from_slice(&addition.as_bytes()[..partial_len.get()]);
    assert_eq!(std::fs::read(&path).unwrap(), want);
    let after = std::fs::symlink_metadata(&path).unwrap();
    use std::os::unix::fs::MetadataExt;
    assert_eq!(
        (before.dev(), before.ino()),
        (after.dev(), after.ino()),
        "append replaced the native inode"
    );
    assert_eq!(before.mode(), after.mode(), "append changed the file mode");
}
