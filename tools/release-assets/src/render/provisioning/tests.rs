use super::*;

#[test]
fn regular_input_reads_admitted_open_file_with_cap_and_private_policy() {
    use std::os::unix::fs::PermissionsExt;

    let dir = std::env::temp_dir().join(format!("prov-private-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir(&dir).unwrap();
    let path = dir.join("secret");
    std::fs::write(&path, b"secret").unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
    assert_eq!(private_files::regular(&path, true).unwrap(), "secret");
    let link = dir.join("link");
    std::os::unix::fs::symlink(&path, &link).unwrap();
    assert!(private_files::regular(&link, true).is_err());

    let opened = std::fs::File::open(&path).unwrap();
    let mut append = std::fs::OpenOptions::new()
        .append(true)
        .open(&path)
        .unwrap();
    use std::io::Write;
    append.write_all(b"-grown").unwrap();
    let replacement = dir.join("replacement");
    std::fs::write(&replacement, b"replacement").unwrap();
    std::fs::rename(&replacement, &path).unwrap();
    assert_eq!(
        private_files::read_open_text(opened, &path, Some(16)).unwrap(),
        "secret-grown"
    );
    assert!(
        private_files::read_open_text(std::fs::File::open(&path).unwrap(), &path, Some(10))
            .is_err()
    );
    assert_eq!(
        private_files::read_open_text(std::fs::File::open(&path).unwrap(), &path, Some(11))
            .unwrap(),
        "replacement"
    );
    let fifo = dir.join("fifo");
    use std::os::unix::ffi::OsStrExt;
    let fifo_name = std::ffi::CString::new(fifo.as_os_str().as_bytes()).unwrap();
    assert_eq!(unsafe { libc::mkfifo(fifo_name.as_ptr(), 0o600) }, 0);
    assert!(private_files::regular(&fifo, true).is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn hostnames_follow_the_script_regexes() {
    assert!(is_appliance_hostname("factory-01.lab.example"));
    assert!(is_appliance_hostname("a"));
    assert!(!is_appliance_hostname(""));
    assert!(!is_appliance_hostname("-lead.example"));
    assert!(!is_appliance_hostname("trail-.example"));
    assert!(!is_appliance_hostname("UPPER.example"));
    assert!(!is_appliance_hostname("under_score.example"));
    assert!(!is_appliance_hostname(".leading.example"));
    assert!(!is_appliance_hostname("trailing.example."));
    assert!(!is_appliance_hostname("double..dot"));
    assert!(!is_appliance_hostname(&"a".repeat(64)));
    assert!(is_appliance_hostname(&("a".repeat(63) + ".example")));
    assert!(!is_appliance_hostname(&"a".repeat(254)));
    assert!(is_fixture_hostname("soda-native-fixture"));
    assert!(is_fixture_hostname("soda-native-a"));
    assert!(!is_fixture_hostname("soda-native-"));
    assert!(!is_fixture_hostname("soda-native-UPPER"));
    assert!(!is_fixture_hostname("other-fixture"));
    assert!(!is_fixture_hostname(
        "soda-native-a-very-long-tail-that-keeps-going-past-forty-two"
    ));
}
