use super::{build_bin, run, TempDir};

#[test]
fn build_help_is_generated_and_safe() {
    let scratch = TempDir::new("build-help");
    let (code, out, err) = run(&build_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert!(err.is_empty(), "{err}");
    for flag in [
        "--development",
        "--target",
        "--distribution",
        "--release-tag",
        "--worker-build",
        "--worker-config",
    ] {
        assert!(out.contains(flag), "help omitted {flag}: {out}");
    }
    assert!(!out.contains("--sign"), "help suggests signing: {out}");
}

#[test]
fn docker_inputs_are_admitted_before_worker_dispatch() {
    let scratch = TempDir::new("docker-build-admission");
    for (args, expected) in [
        (vec!["--distribution", "docker"], "Payload v4 requires an explicit version --release-tag"),
        (vec!["--distribution", "docker", "--release-tag", "latest"], "Payload v4 requires an explicit version --release-tag"),
        (vec!["--distribution", "docker", "--release-tag", "v1.2.3", "--rootfs-base-url", "https://example.invalid"], "candidate target refuses media-only inputs"),
        (vec!["--development", "--target", "docker"], "development media requires appliance; development docker requires docker; development all requires all; candidate selects no packaging"),
        // The combined development target pairs only with the all
        // distribution; a mismatched pairing refuses before dispatch.
        (vec!["--development", "--target", "all", "--distribution", "appliance", "--release-tag", "v9.9.9-dev", "--rootfs-base-url", "https://example.invalid"], "development media requires appliance; development docker requires docker; development all requires all; candidate selects no packaging"),
        (vec!["--distribution", "all", "--release-tag", "v1.2.3", "--rootfs-base-url", "https://github.com/LevitateOS/sodaos/releases/download/v1.2.4/"], "all distribution requires a rootfs URL for the same release tag"),
        (vec!["--distribution", "docker", "--release-tag", "v1.2.3"], "root-owned --worker-config required; media authority belongs to the isolated worker"),
    ] {
        let (code, out, err) = run(&build_bin(), &scratch.path, &args);
        assert_eq!(code, 1, "{args:?}");
        assert!(out.is_empty(), "{args:?}: {out}");
        assert_eq!(err, format!("{expected}\n"), "{args:?}");
    }
}

#[test]
fn build_flag_errors_refuse_without_running() {
    let scratch = TempDir::new("build-flags");
    let (code, out, err) = run(&build_bin(), &scratch.path, &["--bogus", "x"]);
    assert_eq!(code, 2);
    assert_eq!(out, "");
    assert!(err.contains("invalid build command flags"), "{err}");
    assert!(err.contains("Usage:") || err.contains("--help"), "{err}");
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--arch"]);
    assert_eq!(code, 2);
    assert!(err.contains("--arch"), "{err}");
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--development=maybe"]);
    assert_eq!(code, 2);
    assert!(
        err.contains("invalid build command flags") && !err.contains("maybe"),
        "{err}"
    );
    let (code, out, err) = run(&build_bin(), &scratch.path, &["positional"]);
    assert_eq!(code, 2);
    assert!(out.is_empty());
    assert!(err.contains("invalid build command flags"));
}

#[test]
fn build_admission_matrix_preserves_release_boundaries() {
    let scratch = TempDir::new("build-admit");
    for (args, want) in [
        (
            vec!["--development"],
            "--development requires --target candidate, media, docker or all\n",
        ),
        (
            vec!["--development", "--target", "bogus"],
            "--development requires --target candidate, media, docker or all\n",
        ),
        (
            vec!["--target", "candidate"],
            "--target requires --development\n",
        ),
        (
            vec!["--media-compression", "fast"],
            "--media-compression accepts only fast with --development --target media or all\n",
        ),
        (
            vec![
                "--development",
                "--target",
                "all",
                "--distribution",
                "all",
                "--release-tag",
                "v9.9.9-dev",
                "--rootfs-base-url",
                "https://example.invalid",
            ],
            "root-owned --worker-config required; media authority belongs to the isolated worker\n",
        ),
        (
            vec![
                "--development",
                "--target",
                "candidate",
                "--release-tag",
                "v9.9.9-dev",
            ],
            "root-owned --worker-config required; media authority belongs to the isolated worker\n",
        ),
        (
            vec!["--development", "--target", "media"],
            "explicit public HTTP(S) rootfs base URL required\n",
        ),
        (
            vec![
                "--development",
                "--target",
                "media",
                "--rootfs-base-url",
                "http://127.0.0.1:8080/",
            ],
            "rootfs base URL must be reachable from the installing machine, not loopback\n",
        ),
        (
            vec!["--worker-build"],
            "explicit public HTTP(S) rootfs base URL required\n",
        ),
    ] {
        let (code, out, err) = run(&build_bin(), &scratch.path, &args);
        assert_eq!(code, 1, "{args:?}");
        assert_eq!(out, "", "{args:?}");
        assert_eq!(err, want, "{args:?}");
    }

    // Explicit native integration mode: the operator admits a separate root
    // controller/config/collector and existing completed development candidate.
    // Ordinary CLI runs do not claim to exercise the protected systemd boundary.
    if let Some(path) = std::env::var_os("SODA_NATIVE_QUALIFICATION_FIXTURE") {
        use std::{fs, os::unix::fs::MetadataExt, path::Path, process::Command};
        assert_eq!(unsafe { libc::getuid() }, 0, "native fixture requires root");
        let fixture: serde_json::Value = serde_json::from_slice(&fs::read(path).unwrap()).unwrap();
        let controller = Path::new(fixture["controller"].as_str().unwrap());
        let candidate = fixture["candidate"].as_str().unwrap();
        assert!(
            Path::new(candidate)
                .join("producer-private-extra")
                .is_file(),
            "native input fixture includes an excluded private sentinel"
        );
        let parent = Path::new(fixture["output_parent"].as_str().unwrap());
        let mut receipts = serde_json::Map::new();
        // Independent native attempts have disjoint snapshots, output roots and
        // transient units; archive admission need not serialize their I/O.
        let mut attempts = std::thread::scope(|scope| {
            let mut jobs = Vec::new();
            for mode in ["valid", "failed", "incomplete", "cancelled", "recovery"] {
                let config = fixture["configs"][mode].as_str().unwrap();
                let out = parent.join(format!("{mode}-{}", std::process::id()));
                jobs.push((
                    mode,
                    scope.spawn(move || {
                        run(
                            controller,
                            parent,
                            &[
                                "collect",
                                "--config",
                                config,
                                "--candidate",
                                candidate,
                                "--out",
                                out.to_str().unwrap(),
                            ],
                        )
                    }),
                ));
            }
            jobs.into_iter()
                .map(|(mode, job)| (mode, job.join().unwrap()))
                .collect::<std::collections::BTreeMap<_, _>>()
        });
        for mode in ["valid", "failed", "incomplete", "cancelled", "recovery"] {
            let config = fixture["configs"][mode].as_str().unwrap();
            let out = parent.join(format!("{mode}-{}", std::process::id()));
            let out_text = out.to_str().unwrap();
            let args = [
                "collect",
                "--config",
                config,
                "--candidate",
                candidate,
                "--out",
                out_text,
            ];
            let (code, stdout, stderr) = attempts.remove(mode).unwrap();
            let request_bytes = fs::read(out.join("request.json")).unwrap_or_else(|error| {
                panic!("{mode} request missing: {error}; code={code}; {stdout}{stderr}")
            });
            let request: serde_json::Value = serde_json::from_slice(&request_bytes).unwrap();
            let unit = request["unit"].as_str().unwrap();
            let native = Command::new("/usr/bin/systemctl")
                .args([
                    "show",
                    unit,
                    "--property=LoadState",
                    "--property=ActiveState",
                    "--property=MainPID",
                ])
                .output()
                .unwrap();
            assert!(native.status.success());
            let native_state = String::from_utf8(native.stdout).unwrap();
            assert!(
                native_state.contains("MainPID=0\n"),
                "{mode}: {native_state}"
            );
            assert!(
                native_state.contains("ActiveState=inactive\n")
                    || native_state.contains("ActiveState=failed\n"),
                "{mode}: {native_state}"
            );
            assert!(
                !out.join("attempt/.work").exists(),
                "retired private work must be removed"
            );
            let sealed = out.join("observations");
            if mode == "valid" {
                assert_eq!(code, 0, "{stdout}{stderr}");
                let first = fs::read(sealed.join("release-evidence.json")).unwrap();
                let binding =
                    soda_release_deliver::publish::qualification::read_protected(&sealed).unwrap();
                assert_eq!(binding.purpose, "development");
                assert_eq!(binding.exit_code, 0);
                assert_eq!(binding.native_retirement, "confirmed");
                assert_eq!(binding.unit, unit);
                assert!(
                    fs::symlink_metadata(sealed.join("collector.json"))
                        .unwrap()
                        .uid()
                        == 0
                );
                let native_check: serde_json::Value =
                    serde_json::from_slice(&fs::read(sealed.join("native-check.json")).unwrap())
                        .unwrap();
                assert!(native_check["uid"].as_u64().unwrap() > 0);
                assert_eq!(native_check["candidate_read_only"], true);
                if fixture["runtime_access"] == true {
                    assert_eq!(native_check["runtime_read_only"], true);
                    assert_eq!(native_check["masked_home"], true);
                    assert_eq!(native_check["chromium_sandbox"], true);
                    assert_eq!(native_check["offline_requests"], 0);
                    assert_eq!(
                        native_check["credential_names"],
                        serde_json::json!(["guest-ssh-key", "workspace-owner-password"])
                    );
                }
                assert!(!sealed.join(".work").exists());
                assert!(!binding.files.keys().any(|name| name.starts_with(".work/")));
                assert!(!out.join("input/producer-private-extra").exists());
                let replay = run(controller, parent, &args);
                assert_eq!(replay.0, 0, "{}{}", replay.1, replay.2);
                assert_eq!(
                    fs::read(sealed.join("release-evidence.json")).unwrap(),
                    first
                );
                assert_eq!(fs::read(out.join("request.json")).unwrap(), request_bytes);
                // Qualification now requires a prepared public release. Check
                // the shared stamp boundary directly against this actual
                // sealed development result, without a GitHub request.
                let refused = soda_release_deliver::upgrade::stamp(Path::new(candidate), &sealed)
                    .expect_err("development evidence cannot stamp a production upgrade");
                assert!(
                    refused
                        .0
                        .contains("production-bound protected collector evidence required"),
                    "{refused}"
                );
                let check = sealed.join("native-check.json");
                let bytes = fs::read(&check).unwrap();
                fs::write(&check, b"changed after finalization").unwrap();
                assert!(
                    soda_release_deliver::publish::qualification::read_protected(&sealed).is_err()
                );
                fs::write(check, bytes).unwrap();
                soda_release_deliver::publish::qualification::read_protected(&sealed).unwrap();
            } else if mode == "recovery" {
                assert_ne!(code, 0, "deliberately occupied final record must refuse");
                assert!(out.join("executed.json").is_file());
                let executed = fs::read(out.join("executed.json")).unwrap();
                let before: serde_json::Value =
                    serde_json::from_slice(&fs::read(sealed.join("native-check.json")).unwrap())
                        .unwrap();
                assert_eq!(
                    fs::read(sealed.join("release-evidence.json")).unwrap(),
                    b"retained fixture finalization interruption\n"
                );
                fs::remove_file(sealed.join("release-evidence.json")).unwrap();
                let recovered = run(
                    controller,
                    parent,
                    &["finalize-evidence", "--out", out_text],
                );
                assert_eq!(recovered.0, 0, "{}{}", recovered.1, recovered.2);
                assert_eq!(fs::read(out.join("executed.json")).unwrap(), executed);
                let after: serde_json::Value =
                    serde_json::from_slice(&fs::read(sealed.join("native-check.json")).unwrap())
                        .unwrap();
                assert_eq!(before, after, "recovery must not rerun collector");
                assert_eq!(
                    fs::read(sealed.join("release-evidence.json")).unwrap(),
                    executed
                );
                soda_release_deliver::publish::qualification::read_protected(&sealed).unwrap();
            } else {
                assert_ne!(code, 0, "{mode} must not seal");
                assert!(!sealed.join("release-evidence.json").exists());
                assert!(!out.join("executed.json").exists());
                assert!(
                    out.join("attempt/started").is_file(),
                    "{mode} ran an actual collector"
                );
                if mode == "failed" {
                    assert_eq!(
                        fs::read(out.join("attempt/failure")).unwrap(),
                        b"intentional native collector failure\n"
                    );
                }
                let refused = run(
                    controller,
                    parent,
                    &["finalize-evidence", "--out", out_text],
                );
                assert_ne!(refused.0, 0, "partial native output cannot be finalized");
                assert_eq!(fs::read(out.join("request.json")).unwrap(), request_bytes);
            }
            receipts.insert(mode.to_owned(), serde_json::json!({"controller_exit":code,"unit":unit,"native_state":native_state,"request_sha256":soda_release_deliver::hash_bytes(&request_bytes),"output":out}));
        }
        for mode in ["mismatched", "forged", "changed-public-input"] {
            let config = fixture["configs"][mode].as_str().unwrap();
            let out = parent.join(format!("{mode}-{}", std::process::id()));
            let result = run(
                controller,
                parent,
                &[
                    "collect",
                    "--config",
                    config,
                    "--candidate",
                    candidate,
                    "--out",
                    out.to_str().unwrap(),
                ],
            );
            assert_ne!(result.0, 0);
            assert!(
                !out.join("request.json").exists(),
                "{mode} must refuse before native attempt"
            );
            if mode != "changed-public-input" {
                assert!(!out.exists(), "{mode} must refuse before output creation");
            } else {
                assert!(
                    result
                        .2
                        .contains("public artifact differs from admitted collector input"),
                    "{}",
                    result.2
                );
            }
            receipts.insert(
                mode.to_owned(),
                serde_json::json!({"controller_exit":result.0,"refused_before_attempt":true}),
            );
        }
        if fixture["runtime_access"] == true {
            for mode in [
                "unknown-credential",
                "changed-runtime",
                "credential-mismatch",
                "production-missing-runtime",
            ] {
                let out = parent.join(format!("{mode}-{}", std::process::id()));
                let result = run(
                    controller,
                    parent,
                    &[
                        "collect",
                        "--config",
                        fixture["configs"][mode].as_str().unwrap(),
                        "--candidate",
                        candidate,
                        "--out",
                        out.to_str().unwrap(),
                    ],
                );
                assert_ne!(result.0, 0);
                assert!(
                    !out.exists(),
                    "invalid access must refuse before output/native attempt"
                );
            }
        }
        // A root-owned config under a caller-writable ancestor is still forged.
        let forged = scratch.path.join("root-leaf.json");
        fs::copy(fixture["configs"]["valid"].as_str().unwrap(), &forged).unwrap();
        let out = parent.join(format!("writable-parent-{}", std::process::id()));
        let result = run(
            controller,
            parent,
            &[
                "collect",
                "--config",
                forged.to_str().unwrap(),
                "--candidate",
                candidate,
                "--out",
                out.to_str().unwrap(),
            ],
        );
        assert_ne!(result.0, 0);
        assert!(!out.exists());
        fs::write(
            parent.join(format!("native-result-{}.json", std::process::id())),
            serde_json::to_vec_pretty(&receipts).unwrap(),
        )
        .unwrap();
    }
}

#[test]
fn build_bool_and_scalar_forms_preserve_contract() {
    let scratch = TempDir::new("build-bool");
    // Explicit false keeps production dispatch, which the parent refuses.
    let (code, _, err) = run(
        &build_bin(),
        &scratch.path,
        &[
            "--development=false",
            "--target=candidate",
            "--target=media",
            "--worker-config",
            "/cfg",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "--target requires --development\n");
    // A scalar repeated with documented long syntax resolves to its last value.
    let (code, _, err) = run(
        &build_bin(),
        &scratch.path,
        &[
            "--development",
            "--target=candidate",
            "--target=media",
            "--worker-config",
            "/cfg",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "explicit public HTTP(S) rootfs base URL required\n");
    // A bare bool never consumes the next argument: `false` is positional.
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--development", "false"]);
    assert_eq!(code, 2);
    assert!(err.contains("invalid build command flags"));
}

#[test]
fn build_dispatch_refuses_nested_checkout_before_privileged_work() {
    // A nested directory must fail source admission before worker dispatch,
    // regardless of where the test's temporary directory lives.
    let scratch = TempDir::new("build-deep");
    let init = std::process::Command::new("git")
        .args(["init", "--quiet"])
        .arg(&scratch.path)
        .output()
        .unwrap();
    assert!(init.status.success());
    let nested = scratch.path.join("nested");
    std::fs::create_dir(&nested).unwrap();
    let candidate = scratch.path.join("candidate");
    let (code, out, err) = run(
        &build_bin(),
        &nested,
        &[
            "--development",
            "--target",
            "candidate",
            "--release-tag",
            "v9.9.9-dev",
            "--arch",
            "x86_64",
            "--out",
            candidate.to_str().unwrap(),
            "--repository-prefix",
            "ghcr.io/levitateos/sodaos",
            "--worker-config",
            "/cfg",
            "--forgejo-source",
            "/forgejo",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(out, "");
    // The FAILED progress line precedes the final admission error.
    assert!(
        err.starts_with("FAILED   Soda development candidate (not release-qualified) | total "),
        "{err}"
    );
    // A dirty controller is rejected before inspecting the source directory.
    let expected = if env!("SODA_BUILD_VCS_MODIFIED") == "0" {
        "canonical checkout root required\n"
    } else {
        "controller must be compiled from committed source\n"
    };
    assert!(err.ends_with(expected), "{err}");
    assert!(
        !candidate.exists(),
        "refused input created candidate output"
    );
}
