use super::*;

#[test]
fn oracle_media_url_has_no_credentials_or_mutable_query() {
    // Oracle: Go TestMediaURLHasNoCredentialsOrMutableQuery.
    assert!(media_base_url("https://example.invalid/rootfs").is_ok());
    assert!(media_base_url("http://example.invalid/rootfs").is_ok());
    assert!(media_base_url("https://example.invalid/rootfs?").is_ok());
    assert!(media_base_url("https://example.invalid/rootfs#").is_ok());
    for bad in [
        "https://user@example.invalid/rootfs",
        "https://example.invalid/rootfs?tag=latest",
        "https://example.invalid/rootfs#frag",
        "ftp://example.invalid/rootfs",
        "https:///noroot",
        "https://example.invalid/has space",
        "https://example.invalid\\path",
        "https://example.invalid/%zz",
        "https://localhost/rootfs",
        "https://127.0.0.1/rootfs",
        "https://0177.0.0.1/rootfs",
        "https://2130706433/rootfs",
        "https://[::1]/rootfs",
        "not-a-url",
        "",
    ] {
        assert!(media_base_url(bad).is_err(), "accepted {bad:?}");
    }
}

#[test]
fn oracle_assembler_pin_requires_buildroot_selection() {
    let dir = std::env::temp_dir().join(format!("sri-pin-{}", std::process::id()));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(dir.join("config")).unwrap();
    fs::write(dir.join("config/build-args.conf"), b"OTHER=1\n").unwrap();
    assert_eq!(
        pin_assembler_build_args(dir.to_str().unwrap())
            .unwrap_err()
            .0,
        "missing upstream buildroot selection"
    );
    fs::write(
        dir.join("config/build-args.conf"),
        b"BUILDER_IMG=upstream\n",
    )
    .unwrap();
    pin_assembler_build_args(dir.to_str().unwrap()).unwrap();
    assert_eq!(
        fs::read(dir.join("config/build-args.conf")).unwrap(),
        b"BUILDER_IMG=oci-archive:/srv/tmp/assembler-root.oci\n"
    );
    let _ = fs::remove_dir_all(&dir);
}
