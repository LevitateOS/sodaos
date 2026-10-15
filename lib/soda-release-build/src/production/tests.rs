use super::*;
use crate::files::{is_digest, write_new};
use crate::production_inputs::parse_image_repo;
use crate::test_support::{fixture_oci_bytes, FIXTURE_REVISION};
use std::os::unix::fs::PermissionsExt;
use std::sync::{Arc, Mutex};

#[allow(clippy::field_reassign_with_default)]
fn production_fixture() -> (Production, Arc<Mutex<Vec<String>>>, PathBuf) {
    let root = std::env::temp_dir().join(format!(
        "soda-prod-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    let out = root.join(".artifacts/native/x86_64");
    std::fs::create_dir_all(&out).unwrap();
    let files = [
        (
            "system/docker/Dockerfile",
            "ARG BASE_IMAGE=registry.fedoraproject.org/fedora:44\n",
        ),
        ("system/docker/compose.yaml", "services: {}\n"),
        (
            "package.json",
            "{\"packageManager\":\"bun@1.4.2\",\"unrelated\":true}",
        ),
        (
            "system/project/Containerfile",
            "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
        ),
        (
            "system/containers/dashboard/Containerfile",
            "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
        ),
        (
            "system/containers/canlang/Containerfile",
            "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
        ),
        (
            "system/host/services/forgejo.container",
            "[Container]\nImage=codeberg.org/forgejo/forgejo:15.0.10\n",
        ),
        (
            "system/host/services/soda-proxy.container",
            "[Container]\nImage=docker.io/library/caddy:2\n",
        ),
        (
            "system/host/services/soda-postgres.container",
            "[Container]\nImage=docker.io/library/postgres:17\n",
        ),
    ];
    for (name, body) in files {
        let path = root.join(name);
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(&path, body).unwrap();
    }
    let seed = fixture_oci_bytes("amd64");
    let seed_path = root.join("seed.oci");
    std::fs::write(&seed_path, &seed).unwrap();
    let image = soda_release_deliver::oci::inspect_oci(
        seed_path.to_str().unwrap(),
        "x86_64",
        FIXTURE_REVISION,
    )
    .unwrap();
    // Input freezing uses no floating toolchain file: the pinned official
    // Tailscale image joins the same frozen pull record.
    let calls: Arc<Mutex<Vec<String>>> = Arc::new(Mutex::new(Vec::new()));
    let calls_capture = calls.clone();
    let config = image.config.clone();
    let mut prod = Production::default();
    prod.source = root.to_string_lossy().into_owned();
    prod.native = out.to_string_lossy().into_owned();
    prod.out = out.to_string_lossy().into_owned();
    prod.arch = "x86_64".to_string();
    prod.revision = FIXTURE_REVISION.to_string();
    prod.capture = Some(Box::new(move |_dir, name, args| {
        calls_capture
            .lock()
            .unwrap()
            .push(format!("{name} {}", args.join(" ")));
        if name == "bun" {
            return Ok("1.4.2".to_string());
        }
        assert_eq!(name, "podman");
        assert_eq!(args[0], "--remote=false");
        if args[1] == "pull" {
            return Ok(config.clone());
        }
        if args[1] == "image" {
            return Ok(format!("sha256:{}", "b".repeat(64)));
        }
        panic!("unexpected observation: {args:?}");
    }));
    let calls_exec = calls.clone();
    let seed_bytes = seed.clone();
    let image_config = image.config.clone();
    prod.execute = Some(Box::new(move |dir, name, args| {
        calls_exec
            .lock()
            .unwrap()
            .push(format!("{name} {}", args.join(" ")));
        if name == "go" && args.first().map(String::as_str) == Some("build") {
            for (i, arg) in args.iter().enumerate() {
                if arg == "-o" {
                    let dest = PathBuf::from(&args[i + 1]);
                    std::fs::create_dir_all(dest.parent().unwrap()).unwrap();
                    std::fs::write(&dest, crate::elf::tests::fixture_elf()).unwrap();
                    return Ok(());
                }
            }
        }
        if name == "cargo" && args.first().map(String::as_str) == Some("build") {
            for (i, arg) in args.iter().enumerate() {
                if arg == "-p" && i + 1 < args.len() {
                    // Post-A01 fold, -p soda-project-terminal builds three
                    // bins like cargo; every other package keeps one.
                    let bins: Vec<&str> = match args[i + 1].as_str() {
                        "soda-project-terminal" => {
                            vec![
                                "project-terminal",
                                "project-account",
                                "project-factory-roles",
                            ]
                        }
                        pkg => vec![pkg],
                    };
                    for bin in bins {
                        let configured_target = std::env::var_os("CARGO_TARGET_DIR")
                            .map(PathBuf::from)
                            .unwrap_or_else(|| PathBuf::from("target"));
                        let target = if configured_target.is_absolute() {
                            configured_target
                        } else {
                            PathBuf::from(dir).join(configured_target)
                        };
                        let dest = target.join("release").join(bin);
                        std::fs::create_dir_all(dest.parent().unwrap()).unwrap();
                        std::fs::write(&dest, crate::elf::tests::fixture_elf()).unwrap();
                    }
                    return Ok(());
                }
            }
        }
        // Built roles export through the Skopeo containers-storage to
        // oci-archive transport; the sealed archive is the replayed seed.
        if name == "skopeo" {
            for arg in args {
                if let Some(path) = arg.strip_prefix("oci-archive:") {
                    return write_new(Path::new(path), &seed_bytes, 0o600);
                }
            }
        }
        if name != "podman" {
            return Ok(());
        }
        assert_eq!(args[0], "--remote=false");
        for (i, arg) in args.iter().enumerate() {
            if arg == "--iidfile" {
                return write_new(Path::new(&args[i + 1]), image_config.as_bytes(), 0o600);
            }
        }
        panic!("unexpected execution: {args:?}");
    }));
    (prod, calls, root)
}

#[test]
fn docker_packaging_reuses_host_and_refuses_an_unverified_export() {
    let (mut prod, calls, root) = production_fixture();
    prod.resolve_inputs().unwrap();
    prod.prepare_docker().unwrap();
    assert!(prod.prepare_docker().is_err());
    let golden: serde_json::Value = serde_json::from_str(include_str!(
        "../../../soda-release-deliver/tests/goldens/deliver.json"
    ))
    .unwrap();
    let payload = golden["payload"].as_str().unwrap();
    let mut candidate: soda_release_deliver::model::Candidate =
        serde_json::from_str(golden["candidate"].as_str().unwrap()).unwrap();
    candidate.payload_sha256 = crate::sha256_hex(payload.as_bytes());
    std::fs::write(PathBuf::from(&prod.out).join("payload.json"), payload).unwrap();
    std::fs::write(
        PathBuf::from(&prod.out).join("candidate.json"),
        serde_json::to_vec(&candidate).unwrap(),
    )
    .unwrap();
    let capture = prod.capture.take().unwrap();
    prod.capture = Some(Box::new(move |dir, name, args| {
        if args.iter().any(|a| a == "--entrypoint=/usr/bin/cat") {
            Ok("fedora 0:44-1.x86_64".into())
        } else {
            capture(dir, name, args)
        }
    }));
    // The scripted exporter emits a normal app OCI, not a Docker image carrying
    // the candidate. Packaging must fail closed without a metadata receipt.
    assert!(prod
        .build_docker(
            "v1.2.3",
            soda_release_deliver::model::Distribution::Docker,
            soda_release_deliver::model::BuildPurpose::Development
        )
        .is_err());
    let calls = calls.lock().unwrap().join("\n");
    // Five frozen production inputs plus the Docker base pull. The Docker
    // image exports through the same Skopeo transport; no podman save
    // remains anywhere in the pipeline.
    assert_eq!(calls.matches(" pull ").count(), 6);
    assert_eq!(calls.matches(" build ").count(), 1);
    assert_eq!(calls.matches(" save ").count(), 0);
    assert_eq!(calls.matches("skopeo copy --format oci").count(), 1);
    for arg in [
        format!("--build-arg=SODA_BASE_IMAGE={}", candidate.host.config),
        format!(
            "--build-arg=BASE_IMAGE=registry.fedoraproject.org/fedora@sha256:{}",
            "b".repeat(64)
        ),
        "--pull=never".into(),
        "--label=io.soda.build-purpose=development".into(),
        "--label=org.opencontainers.image.version=v1.2.3".into(),
        format!(
            "--label=io.soda.payload-sha256={}",
            candidate.payload_sha256
        ),
    ] {
        assert!(calls.contains(&arg), "missing {arg} in {calls}");
    }
    assert!(
        !calls.contains("cargo build") && !calls.contains("go build") && !calls.contains("bun")
    );
    assert!(!PathBuf::from(&prod.out).join("docker/docker.json").exists());
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn oracle_production_sequence() {
    // Oracle: TestProductionUsesOneAssetAndImageSequence.
    let (mut prod, calls, _root) = production_fixture();
    let host = PathBuf::from(&prod.out).join("context");
    let forgejo = PathBuf::from(&prod.out).join("forgejo-context");
    prod.dependencies().unwrap();
    prod.resolve_inputs().unwrap();
    prod.assets(&host.to_string_lossy(), &forgejo.to_string_lossy())
        .unwrap();
    for tool in [
        "muse",
        "soda-identity-compose",
        "project-terminal",
        "project-account",
    ] {
        let path = PathBuf::from(&prod.native)
            .join("project-tools/bin")
            .join(tool);
        let mode = std::fs::metadata(&path).unwrap().permissions().mode() & 0o777;
        assert_eq!(mode, 0o755, "{tool}");
    }
    let images = prod.images(&forgejo.to_string_lossy()).unwrap();
    assert_eq!(images.len(), 6);
    let text = calls.lock().unwrap().join("\n");
    for needle in [
        "bun install --frozen-lockfile",
        "bun scripts/build-forgejo.ts",
        "cargo run --release --locked -p soda-release-assets --bin soda-fetch-terminal -- --out ",
        "cargo run --release --locked -p soda-release-assets --bin soda-fetch-muse -- --arch x86_64 --out ",
        "cargo run --release --locked -p soda-release-assets --bin soda-fetch-tea -- --arch x86_64 --out ",
        "cargo run --release --locked -p soda-release-assets --bin soda-stage-canlang -- --archive ",
        "cargo run --release --locked -p soda-release-assets --bin soda-forgejo-locales -- --lock frontend/forgejo/locale.lock.json --out ",
        "cargo run --release --locked -p soda-release-assets --bin soda-stage -- --arch x86_64 --host-context ",
        "bun scripts/build-soda-extension.ts --package=soda --out ",
        "-p soda-project-terminal --bin project-terminal",
    ] {
        assert_eq!(text.matches(needle).count(), 1, "needle: {needle}\n{text}");
    }
    assert!(!text.contains("python3 scripts/fetch-"));
    assert!(!text.contains("tools/soda-fetch-muse"));
    assert!(text.contains(&format!(
        "--host-context {} --forgejo-context {}",
        host.display(),
        forgejo.display()
    )));
    // Four built roles export through the Skopeo transport; the two
    // upstream binds and the removed extension/Tailnet builds emit no
    // export command and no local archive.
    assert_eq!(
        text.matches(
            "skopeo copy --format oci --dest-compress-format gzip --dest-compress-level 1"
        )
        .count(),
        4
    );
    assert!(!text.contains(" save --uncompressed --format=oci-archive"));
    assert!(!text.contains("extension.iid") && !text.contains("tailnet.iid"));
    assert!(images.contains_key("canlang-runtime"));
    assert!(!text.contains(" push ") && !text.contains(" --rm ") && !text.contains("--replace"));
    assert!(images.contains_key("proxy"));
    assert!(images.contains_key("postgres"));
    assert!(!images.contains_key("caddy"));
    assert!(!images.contains_key("extension") && !images.contains_key("tailnet"));
    for name in ["dashboard", "forgejo", "project-os", "canlang-runtime"] {
        let produced = &images[name];
        assert!(is_digest(&produced.archive_sha256), "{name}");
        assert!(!produced.image.manifest.is_empty() && !produced.image.config.is_empty());
        assert!(PathBuf::from(&prod.out)
            .join("images")
            .join(format!("{name}.oci"))
            .exists());
    }
    for name in ["proxy", "postgres"] {
        let produced = &images[name];
        assert!(produced.archive_sha256.is_empty(), "{name}");
        assert!(!produced.image.manifest.is_empty() && !produced.image.config.is_empty());
        assert!(!PathBuf::from(&prod.out)
            .join("images")
            .join(format!("{name}.oci"))
            .exists());
    }
    let before = calls.lock().unwrap().len();
    assert!(prod.images(&forgejo.to_string_lossy()).is_err());
    assert_eq!(calls.lock().unwrap().len(), before);
}

#[test]
fn oracle_production_failure_stops() {
    // Oracle: TestProductionFailureStopsBeforeLaterImages.
    let (mut prod, calls, _root) = production_fixture();
    let execute = prod.execute.take();
    prod.execute = Some(Box::new(move |dir, name, args| {
        if name == "podman" && args.join(" ").contains("dashboard/Containerfile") {
            return Err(Error::msg("fixture build refused"));
        }
        execute.as_ref().unwrap()(dir, name, args)
    }));
    prod.resolve_inputs().unwrap();
    let err = prod.images("forgejo-context").unwrap_err();
    assert_eq!(err.message(), "fixture build refused");
    let text = calls.lock().unwrap().join("\n");
    assert!(PathBuf::from(&prod.out).join("app-inputs.json").exists());
}

#[test]
fn oracle_production_refusals() {
    // Oracle: TestProductionRefusesWrongInputsAndLayout. The floating
    // Tailnet toolchain is gone: input freezing admits only the five pinned
    // roles, and production refuses anything else before building.
    for mode in ["bun", "base", "forgejo", "platform", "revision", "inputs"] {
        let (mut prod, _calls, _root) = production_fixture();
        match mode {
            "bun" => {
                prod.capture = Some(Box::new(|_, _, _| Ok("different".to_string())));
                assert!(prod.dependencies().is_err(), "{mode}");
                continue;
            }
            "base" => {
                std::fs::write(
                    PathBuf::from(&prod.source).join("system/containers/dashboard/Containerfile"),
                    b"ARG BASE_IMAGE=unrelated\n",
                )
                .unwrap();
            }
            "forgejo" => {
                assert!(prod.images("").is_err(), "{mode}");
                continue;
            }
            "platform" => prod.arch = "other".to_string(),
            "revision" => prod.revision = "dirty".to_string(),
            "inputs" => {
                // Five frozen inputs are required; anything else refuses
                // before any build runs.
                prod.inputs.clear();
            }
            _ => unreachable!(),
        }
        assert!(prod.images("forgejo-context").is_err(), "{mode}");
    }
}

#[test]
fn oracle_asset_destinations_refuse_early() {
    // Oracle: TestProductionAssetDestinationsRefuseBeforeCommands.
    for destinations in [
        ("/host-context", ""),
        ("", "/forgejo-context"),
        ("relative-host", "/forgejo-context"),
        ("/host-context", "relative-forgejo"),
    ] {
        let (prod, calls, _root) = production_fixture();
        assert!(prod.assets(destinations.0, destinations.1).is_err());
        assert!(calls.lock().unwrap().is_empty());
    }
}

#[test]
fn shipping_compile_recipes() {
    // Oracle: TestProductionCompileKeepsLayoutAndVerifiesELF +
    // TestProductionCompileRustKeepsLayoutAndVerifiesELF.
    let (prod, calls, _root) = production_fixture();
    let dest = PathBuf::from(&prod.out).join("program");
    prod.compile("./cmd/soda-dashboard", &dest.to_string_lossy())
        .unwrap();
    let text = calls.lock().unwrap().join("\n");
    assert!(text.contains("-buildvcs=false") && !text.contains("-tags="));
    assert!(text.contains("-ldflags=-s -w"));
    assert_eq!(text.matches("go build").count(), 1);
    let dest = PathBuf::from(&prod.out).join("soda-identity-compose");
    prod.compile_rust(
        "soda-identity-compose",
        "soda-identity-compose",
        &dest.to_string_lossy(),
    )
    .unwrap();
    let text = calls.lock().unwrap().join("\n");
    assert!(
        text.contains("cargo build --release --locked --offline")
            && text.contains("--config profile.release.strip=\"symbols\"")
            && text.contains("-p soda-identity-compose --bin soda-identity-compose")
    );
    assert_eq!(text.matches("cargo build").count(), 1);
    for (bad_crate, bad_bin) in [
        ("", "bin"),
        ("crate", ""),
        ("a/b", "bin"),
        ("crate", "a/bin"),
    ] {
        assert!(prod
            .compile_rust(bad_crate, bad_bin, &dest.to_string_lossy())
            .is_err());
    }
}

#[test]
fn oracle_image_repo_parsing() {
    assert_eq!(
        parse_image_repo("docker.io/library/caddy:2"),
        "docker.io/library/caddy"
    );
    assert_eq!(parse_image_repo("quay.io/r@sha256:aaaa"), "quay.io/r");
    assert_eq!(parse_image_repo("localhost:5000/r:tag"), "localhost:5000/r");
    assert_eq!(parse_image_repo("localhost:5000/r"), "localhost:5000/r");
}
