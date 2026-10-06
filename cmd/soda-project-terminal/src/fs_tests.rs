use super::*;

static TEST_SEQ: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

/// Unique per-test scratch dir (tests run in parallel threads).
fn test_dir(tag: &str) -> std::path::PathBuf {
    let n = TEST_SEQ.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
    let dir = std::env::temp_dir().join(format!("soda-pt-fs-{tag}-{}-{n}", std::process::id()));
    std::fs::create_dir_all(&dir).expect("test dir");
    dir
}

fn dir_fd(path: &std::path::Path) -> std::fs::File {
    std::fs::File::open(path).unwrap()
}

fn am_root() -> bool {
    unsafe { libc::getuid() == 0 }
}

fn s_isdir(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFDIR
}

#[test]
fn root_chain_matrix() {
    // "/" and "" both yield the root dir itself.
    for top in ["/", "", "usr", "/usr", "usr/", "/usr/bin/", "usr//bin"] {
        let dir = root_chain(top).unwrap_or_else(|e| panic!("root_chain({top:?}): {e}"));
        let (uid, mode) = fstat_uid_mode(&dir).unwrap();
        assert_eq!(uid, 0);
        assert!(s_isdir(mode));
    }
    assert!(root_chain("no-such-dir-soda-pt").is_err());
    // Traversal components fail closed.
    for top in ["..", "usr/../etc", ".", "usr/./bin"] {
        assert!(root_chain(top).is_err(), "root_chain({top:?})");
    }
}

fn fake_stat(mode: u32, uid: u32, nlink: u64) -> libc::stat {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    st.st_mode = mode;
    st.st_uid = uid;
    st.st_nlink = nlink as libc::nlink_t;
    st
}

#[test]
fn safety_predicate_matrix() {
    assert!(stat_is_safe(&fake_stat(libc::S_IFREG | 0o600, 0, 1)));
    assert!(stat_is_safe(&fake_stat(libc::S_IFREG | 0o400, 0, 1)));
    // Every single violation flips the predicate.
    assert!(!stat_is_safe(&fake_stat(libc::S_IFDIR | 0o700, 0, 1))); // not regular
    assert!(!stat_is_safe(&fake_stat(libc::S_IFLNK | 0o777, 0, 1))); // symlink
    assert!(!stat_is_safe(&fake_stat(libc::S_IFIFO | 0o600, 0, 1))); // fifo
    assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o600, 1000, 1))); // uid
    assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o600, 0, 2))); // nlink
    assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o640, 0, 1))); // group bit
    assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o601, 0, 1))); // other bit
    assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o644, 0, 1))); // group+other
    assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o607, 0, 1)));
    assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o600, 0, 0))); // nlink 0
}

#[test]
fn root_file_failures() {
    let dir = test_dir("rootfile");
    std::fs::write(dir.join("world"), b"{}").unwrap();
    let world = std::fs::File::open(dir.join("world")).unwrap();
    fchmod(&world, 0o644).unwrap();
    drop(world);
    std::fs::hard_link(dir.join("world"), dir.join("linked")).unwrap();
    std::fs::create_dir(dir.join("subdir")).unwrap();
    std::os::unix::fs::symlink("world", dir.join("link")).unwrap();
    let fd = dir_fd(&dir);
    // Mode, nlink, and non-regular all fail closed with the fixed string.
    assert_eq!(
        root_file(&fd, "world", false).unwrap_err(),
        "unsafe terminal file"
    );
    assert_eq!(
        root_file(&fd, "linked", false).unwrap_err(),
        "unsafe terminal file"
    );
    assert_eq!(
        root_file(&fd, "subdir", false).unwrap_err(),
        "unsafe terminal file"
    );
    // Symlink and missing keep OS error text (ELOOP / ENOENT).
    let err = root_file(&fd, "link", false).unwrap_err();
    assert!(err.contains("symbolic link"), "{err}");
    let err = root_file(&fd, "absent", false).unwrap_err();
    assert!(err.contains("No such file"), "{err}");
    for bad in ["", ".", "..", "a/b"] {
        assert!(root_file(&fd, bad, false).is_err(), "name {bad:?}");
    }
    if am_root() {
        std::fs::write(dir.join("good"), b"{}").unwrap();
        let f = std::fs::File::open(dir.join("good")).unwrap();
        fchmod(&f, 0o600).unwrap();
        // nlink must be 1: "world" was linked but "good" was not.
        assert!(root_file(&fd, "good", false).is_ok());
        assert!(root_file(&fd, "good", true).is_ok());
    }
    let _ = std::fs::remove_dir_all(&dir);
}

#[test]
fn record_size_and_json_limits() {
    let dir = test_dir("record");
    // Exactly 4096 bytes of valid JSON: quoted string of 4094 chars.
    let exact = format!("\"{}\"", "a".repeat(4094));
    assert_eq!(exact.len(), 4096);
    std::fs::write(dir.join("exact"), &exact).unwrap();
    let value = read_record_inner(std::fs::File::open(dir.join("exact")).unwrap()).unwrap();
    assert_eq!(value, soda_json::JsonValue::Str("a".repeat(4094)));
    // 4097 bytes trips the limit even when the JSON is valid.
    let over = format!("\"{}\"", "b".repeat(4095));
    std::fs::write(dir.join("over"), &over).unwrap();
    assert_eq!(
        read_record_inner(std::fs::File::open(dir.join("over")).unwrap()).unwrap_err(),
        "terminal record size"
    );
    // Malformed, empty, and non-UTF8 bodies share the fixed JSON string.
    std::fs::write(dir.join("bad"), b"{oops").unwrap();
    assert_eq!(
        read_record_inner(std::fs::File::open(dir.join("bad")).unwrap()).unwrap_err(),
        "terminal record json"
    );
    std::fs::write(dir.join("empty"), b"").unwrap();
    assert_eq!(
        read_record_inner(std::fs::File::open(dir.join("empty")).unwrap()).unwrap_err(),
        "terminal record json"
    );
    std::fs::write(dir.join("bin"), [0xff, 0xfe, 0x00]).unwrap();
    assert_eq!(
        read_record_inner(std::fs::File::open(dir.join("bin")).unwrap()).unwrap_err(),
        "terminal record json"
    );
    let _ = std::fs::remove_dir_all(&dir);
}

#[test]
fn read_record_public_paths() {
    let dir = test_dir("readrec");
    std::fs::write(dir.join("world"), b"{}").unwrap();
    let world = std::fs::File::open(dir.join("world")).unwrap();
    fchmod(&world, 0o644).unwrap();
    let fd = dir_fd(&dir);
    // Unsafe file surfaces the safety string through read_record.
    assert_eq!(
        read_record(&fd, "world").unwrap_err(),
        "unsafe terminal file"
    );
    assert!(read_record(&fd, "absent").is_err());
    let _ = std::fs::remove_dir_all(&dir);
}

#[test]
fn new_file_create_and_collision() {
    let dir = test_dir("newfile");
    let fd = dir_fd(&dir);
    new_file(&fd, "rec", b"{\"a\":1}", 0o600).unwrap();
    assert_eq!(std::fs::read(dir.join("rec")).unwrap(), b"{\"a\":1}");
    let (_, mode) = fstat_uid_mode(&std::fs::File::open(dir.join("rec")).unwrap()).unwrap();
    assert_eq!(mode & 0o7777, 0o600);
    // 0o660 is NOT producible through a 022 umask at create time, so an
    // exact 0o660 proves the explicit fchmod ran.
    new_file(&fd, "rec2", b"x", 0o660).unwrap();
    let (_, mode) = fstat_uid_mode(&std::fs::File::open(dir.join("rec2")).unwrap()).unwrap();
    assert_eq!(mode & 0o7777, 0o660);
    // O_EXCL collision keeps EEXIST text.
    let err = new_file(&fd, "rec", b"z", 0o600).unwrap_err();
    assert!(err.contains("File exists"), "{err}");
    assert_eq!(std::fs::read(dir.join("rec")).unwrap(), b"{\"a\":1}");
    for bad in ["", ".", "..", "a/b"] {
        assert!(new_file(&fd, bad, b"x", 0o600).is_err(), "name {bad:?}");
    }
    let _ = std::fs::remove_dir_all(&dir);
}

#[test]
fn wrapper_matrix() {
    let dir = test_dir("wrappers");
    let fd = dir_fd(&dir);
    mkdir_at(&fd, "sub", 0o755).unwrap();
    assert!(dir.join("sub").is_dir());
    assert!(mkdir_at(&fd, "sub", 0o755).is_err()); // EEXIST
    assert!(mkdir_at(&fd, "../esc", 0o755).is_err());
    assert!(mkdir_at(&fd, "", 0o755).is_err());

    std::fs::write(dir.join("f"), b"data").unwrap();
    let f = std::fs::File::open(dir.join("f")).unwrap();
    fchmod(&f, 0o640).unwrap();
    let (uid, mode) = fstat_uid_mode(&f).unwrap();
    assert_eq!(uid, unsafe { libc::getuid() });
    assert!(s_isreg(mode));
    assert_eq!(mode & 0o7777, 0o640);
    fsync_file(&f).unwrap();

    replace_at(&fd, "f", "g").unwrap();
    assert!(!dir.join("f").exists());
    assert_eq!(std::fs::read(dir.join("g")).unwrap(), b"data");
    assert!(replace_at(&fd, "f", "h").is_err()); // missing source
    assert!(replace_at(&fd, "../x", "h").is_err());

    unlink_at(&fd, "g").unwrap();
    assert!(!dir.join("g").exists());
    assert!(unlink_at(&fd, "g").is_err()); // already gone
    assert!(unlink_at(&fd, ".").is_err());

    rmdir_at(&fd, "sub").unwrap();
    assert!(!dir.join("sub").exists());
    mkdir_at(&fd, "full", 0o755).unwrap();
    std::fs::write(dir.join("full").join("x"), b"").unwrap();
    assert!(rmdir_at(&fd, "full").is_err()); // ENOTEMPTY
    let _ = std::fs::remove_dir_all(&dir);
}

#[test]
fn path_chown_chmod_matrix() {
    let dir = test_dir("pathops");
    let path = dir.join("obj");
    std::fs::write(&path, b"z").unwrap();
    let path_str = path.to_str().unwrap().to_string();
    let uid = unsafe { libc::getuid() };
    let gid = unsafe { libc::getgid() };
    // Chown to self always succeeds.
    chown_path(&path_str, uid, gid, false).unwrap();
    chown_path(&path_str, uid, gid, true).unwrap();
    assert!(chown_path(dir.join("absent").to_str().unwrap(), uid, gid, false).is_err());

    chmod_path(&path_str, 0o640, false).unwrap();
    chmod_path(&path_str, 0o600, true).unwrap();
    let (_, mode) = fstat_uid_mode(&std::fs::File::open(&path).unwrap()).unwrap();
    assert_eq!(mode & 0o7777, 0o600);

    // Symlink discipline: follow works, no_follow refuses.
    let link = dir.join("link");
    std::os::unix::fs::symlink(&path, &link).unwrap();
    let link_str = link.to_str().unwrap().to_string();
    chmod_path(&link_str, 0o640, false).unwrap();
    let err = chmod_path(&link_str, 0o640, true).unwrap_err();
    assert_eq!(err.kind(), std::io::ErrorKind::InvalidInput);
    chown_path(&link_str, uid, gid, true).unwrap(); // lchown on the link
    let _ = std::fs::remove_dir_all(&dir);
}

#[test]
fn stat_all_and_single_read() {
    let dir = test_dir("statread");
    std::fs::write(dir.join("f"), b"hello").unwrap();
    let f = std::fs::File::open(dir.join("f")).unwrap();
    let st = fstat_all(&f).unwrap();
    assert_eq!(st.st_uid, unsafe { libc::getuid() });
    assert_eq!(st.st_gid, unsafe { libc::getgid() });
    assert_eq!(st.st_nlink, 1);
    assert!(s_isreg(st.st_mode));
    assert_eq!(st.st_size, 5);
    // Single bounded read: exact bytes, EOF on the second call.
    assert_eq!(read_up_to(&f, 65537).unwrap(), b"hello");
    assert!(read_up_to(&f, 65537).unwrap().is_empty());
    assert_eq!(read_up_to(&f, 0).unwrap(), b"");
    let _ = std::fs::remove_dir_all(&dir);
}
