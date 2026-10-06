use super::*;

#[test]
fn toolchain_validation_matches_go() {
    let good = ForgejoToolchain {
        compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
        apk_packages: ["build-base-0.5-r4", "gcc-14.2.0-r6", "musl-dev-1.2.5-r10"]
            .iter()
            .map(|s| s.to_string())
            .collect(),
    };
    assert!(good.validate().is_ok());
    let bad = ForgejoToolchain {
        compiler_image: "other".to_string(),
        apk_packages: good.apk_packages.clone(),
    };
    assert_eq!(
        bad.validate().unwrap_err(),
        Error::msg("invalid Forgejo compiler provenance")
    );
    let unsorted = ForgejoToolchain {
        compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
        apk_packages: ["gcc-14.2.0-r6", "build-base-0.5-r4"]
            .iter()
            .map(|s| s.to_string())
            .collect(),
    };
    assert!(unsorted.validate().is_err());
    let missing = ForgejoToolchain {
        compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
        apk_packages: ["build-base-0.5-r4".to_string()].to_vec(),
    };
    assert_eq!(
        missing.validate().unwrap_err(),
        Error::msg("incomplete Forgejo APK provenance")
    );
}

#[test]
fn file_helpers_match_go_errors() {
    assert_eq!(
        fresh_directory("relative/path").unwrap_err(),
        Error::msg("absolute new directory required")
    );
    assert_eq!(
        private_destination("relative").unwrap_err(),
        Error::msg("absolute private output required")
    );
    let dir = std::env::temp_dir().join(format!("srd-{}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let target = dir.join("fresh");
    fresh_directory(target.to_str().unwrap()).unwrap();
    assert!(fresh_directory(target.to_str().unwrap()).is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}
