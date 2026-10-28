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

    native_worker_cancellation();
}

// Optional real host integration. This dispatches only the installed sleep
// executable and cancels it after observing the exact transient unit alive;
// it uses neither a candidate build nor a guest/provider fixture.
fn native_worker_cancellation() {
    use soda_release_tools::worker::{
        run_worker, Worker, WorkerError, WorkerUnitCustody, WORKER_CONTROLLER,
    };
    use std::{
        cell::RefCell,
        fs::{self, DirBuilder, OpenOptions},
        io::Write,
        os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt},
        path::Path,
        process::Command,
        time::{Duration, Instant},
    };

    let Some(directory) = std::env::var_os("SODA_NATIVE_WORKER_CANCELLATION_DIRECTORY") else {
        return;
    };
    assert_eq!(unsafe { libc::getuid() }, 0, "native fixture requires root");
    let root = Path::new(&directory);
    assert_eq!(root.parent(), Some(Path::new("/home")));
    DirBuilder::new().mode(0o700).create(root).unwrap();
    assert_eq!(fs::canonicalize(root).unwrap(), root);
    assert_eq!(fs::symlink_metadata(root).unwrap().uid(), 0);
    let read_only = root.join("read-only");
    DirBuilder::new().mode(0o755).create(&read_only).unwrap();
    fs::set_permissions(&read_only, fs::Permissions::from_mode(0o755)).unwrap();
    let name = format!("soda-build-cancel-{}", std::process::id());
    let unit = format!("{name}.service");
    let worker = Worker {
        name,
        user: "soda-build-worker".into(),
        executable: "/usr/bin/sleep".into(),
        controller: WORKER_CONTROLLER.into(),
        directory: "/run/soda-worker-cancellation".into(),
        read_only: vec![
            format!("/usr/bin/sleep:{WORKER_CONTROLLER}"),
            format!("{}:/run/soda-worker-cancellation", read_only.display()),
        ],
        arguments: vec!["60".into()],
        ..Worker::default()
    };
    let inspect = || {
        let output = Command::new("/usr/bin/systemctl")
            .args([
                "show",
                &unit,
                "--property=LoadState",
                "--property=ActiveState",
                "--property=MainPID",
            ])
            .env_clear()
            .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
            .env("LANG", "C.UTF-8")
            .output()
            .map_err(|error| error.to_string())?;
        if !output.status.success() {
            return Err(format!("systemctl show failed: {}", output.status));
        }
        String::from_utf8(output.stdout).map_err(|error| error.to_string())
    };
    let live = RefCell::new(None);
    let inspection_error = RefCell::new(None);
    let last_probe = RefCell::new(Instant::now() - Duration::from_secs(1));
    let deadline = Instant::now() + Duration::from_secs(15);
    let cancelled = || {
        if live.borrow().is_some() {
            return true;
        }
        if last_probe.borrow().elapsed() >= Duration::from_millis(100) {
            *last_probe.borrow_mut() = Instant::now();
            match inspect() {
                Ok(state) => {
                    let pid = state
                        .lines()
                        .find_map(|line| line.strip_prefix("MainPID="))
                        .and_then(|pid| pid.parse::<u32>().ok());
                    if state.lines().any(|line| line == "ActiveState=active")
                        && pid.is_some_and(|pid| pid > 0)
                    {
                        *live.borrow_mut() = Some(state);
                        return true;
                    }
                }
                Err(error) => *inspection_error.borrow_mut() = Some(error),
            }
        }
        // An observation failure still requests bounded cleanup, but cannot
        // satisfy the live-process assertion below.
        Instant::now() >= deadline
    };
    let mut stdout = Vec::new();
    let mut stderr = Vec::new();
    let mut custody = WorkerUnitCustody::NeverDispatched;
    let result = run_worker(&worker, &mut stdout, &mut stderr, &cancelled, &mut custody);
    let terminal = inspect();
    let receipt = serde_json::json!({
        "format": 1,
        "unit": unit,
        "live_before_cancellation": live.borrow().as_ref(),
        "native_after_cancellation": terminal.as_ref().ok(),
        "native_inspection_error": terminal.as_ref().err().or(inspection_error.borrow().as_ref()),
        "result": format!("{result:?}"),
        "custody": format!("{custody:?}"),
        "stdout": String::from_utf8_lossy(&stdout),
        "stderr": String::from_utf8_lossy(&stderr),
        "guest_mutations": 0,
        "provider_calls": 0,
        "credentials": 0,
        "builds": 0,
    });
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(root.join("result.json"))
        .unwrap();
    file.write_all(&serde_json::to_vec_pretty(&receipt).unwrap())
        .unwrap();
    file.sync_all().unwrap();
    fs::File::open(root).unwrap().sync_all().unwrap();
    assert!(
        live.borrow().is_some(),
        "worker was never observed alive: {receipt}"
    );
    assert!(
        matches!(result, Err(WorkerError::Cancelled(_))),
        "{receipt}"
    );
    assert_eq!(custody, WorkerUnitCustody::ExactUnitTerminal, "{receipt}");
    let terminal = terminal.unwrap();
    assert!(
        terminal.lines().any(|line| line == "MainPID=0"),
        "{receipt}"
    );
    assert!(
        terminal
            .lines()
            .any(|line| matches!(line, "ActiveState=inactive" | "ActiveState=failed")),
        "{receipt}"
    );
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
