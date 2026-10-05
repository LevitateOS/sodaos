use super::*;

pub fn base_options() -> Options {
    Options {
        controller: "/admitted/soda-build".to_owned(),
        worker_config: "/restricted/worker.json".to_owned(),
        arch: "x86_64".to_owned(),
        out: "/source/.artifacts/releases/isolated/test".to_owned(),
        mode: "media".to_owned(),
        rootfs_url: "http://fixture:8080".to_owned(),
        repo_prefix: "ghcr.io/levitateos/sodaos".to_owned(),
        ..Options::default()
    }
}

#[test]
fn arch_flag_admits_only_x86_64() {
    assert!(validate_arch_flag("x86_64").is_ok());
    for arch in ["", "aarch64", "amd64", "arm64"] {
        assert!(validate_arch_flag(arch).is_err(), "{arch}");
    }
}

#[test]
fn resolved_boundaries() {
    let mut o = base_options();
    assert!(validate_resolved(&o).is_ok());
    o.mode = "candidate".to_owned();
    assert_eq!(
        validate_resolved(&o).unwrap_err(),
        "candidate refuses media-only inputs"
    );
    o = base_options();
    o.rootfs_url.clear();
    assert_eq!(
        validate_resolved(&o).unwrap_err(),
        "media requires the rootfs base URL"
    );
    o.mode = "production".to_owned();
    assert!(validate_resolved(&o).is_err());
}

#[test]
fn out_leaf_follows_worker_name_rule() {
    for leaf in ["20260915t212541z", "manual-01", "a"] {
        assert!(valid_out_leaf(leaf), "{leaf}");
    }
    for leaf in [
        "",
        "20260915T212541Z",
        "has space",
        "UPPER",
        "under_score",
        &"a".repeat(49),
    ] {
        assert!(!valid_out_leaf(leaf), "{leaf}");
    }
    let mut o = base_options();
    o.out = "/source/.artifacts/releases/isolated/20260915T212541Z".to_owned();
    let err = validate_resolved(&o).unwrap_err();
    assert!(err.contains("lowercase"), "{err}");
}

#[test]
fn dirty_files_skips_blanks() {
    let got = dirty_files(b" M tools/soda-candidate/main.go\n\n?? scratch\n");
    assert_eq!(got, vec!["M tools/soda-candidate/main.go", "?? scratch"]);
    assert!(dirty_files(b"").is_empty());
}

#[test]
fn resolve_options_defers_worker_admission_to_controller() {
    let args = [
        "--controller",
        "/admitted/soda-build",
        "--worker-config",
        "/nonexistent/worker.json",
        "--arch",
        "x86_64",
        "--out",
        "/tmp/fresh-out-01",
        "--mode",
        "candidate",
        "--non-interactive",
    ]
    .into_iter()
    .map(str::to_owned)
    .collect::<Vec<_>>();
    let mut prompt = |_: &mut Options| -> Result<(), String> {
        panic!("must not prompt");
    };
    let o = resolve_options(&args, false, false, &mut prompt).unwrap();
    assert_eq!(o.worker_config, "/nonexistent/worker.json");
}

#[test]
fn check_fresh_out_matrix() {
    let scratch = std::env::temp_dir().join(format!("soda-reltools-fresh-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&scratch);
    std::fs::create_dir_all(&scratch).unwrap();
    let fresh = scratch.join("fresh-01");
    assert!(check_fresh_out(fresh.to_str().unwrap()).is_ok());
    std::fs::create_dir(&fresh).unwrap();
    assert!(check_fresh_out(fresh.to_str().unwrap())
        .unwrap_err()
        .contains("exists"));
    assert!(
        check_fresh_out(scratch.join("missing-parent/fresh").to_str().unwrap())
            .unwrap_err()
            .contains("must already exist")
    );
    let _ = std::fs::remove_dir_all(&scratch);
}
