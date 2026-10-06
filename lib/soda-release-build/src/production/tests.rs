use super::*;
use crate::coreos_stream::tests::fixture_live_inputs;
use crate::coreos_stream::write_live_inputs;
use crate::files::{is_digest, write_new};
use crate::oci::inspect_oci;
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
            "system/host/services/forgejo.container",
            "[Container]\nImage=codeberg.org/forgejo/forgejo:15.0.9\n",
        ),
        (
            "system/host/services/soda-proxy.container",
            "[Container]\nImage=docker.io/library/caddy:2\n",
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
    let image = inspect_oci(&seed_path, "x86_64", FIXTURE_REVISION).unwrap();
    let live_path = root.join("live-inputs.json");
    write_live_inputs(&live_path, &fixture_live_inputs()).unwrap();
    let calls: Arc<Mutex<Vec<String>>> = Arc::new(Mutex::new(Vec::new()));
    let calls_capture = calls.clone();
    let config = image.config.clone();
    let mut prod = Production::default();
    prod.source = root.to_string_lossy().into_owned();
    prod.native = out.to_string_lossy().into_owned();
    prod.out = out.to_string_lossy().into_owned();
    prod.arch = "x86_64".to_string();
    prod.revision = FIXTURE_REVISION.to_string();
    prod.live_inputs = live_path.to_string_lossy().into_owned();
    let calls_next = calls.clone();
    prod.next = Some(Box::new(move |label| {
        calls_next.lock().unwrap().push(format!("STEP {label}"));
        Ok(())
    }));
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
                        let dest = PathBuf::from(dir).join("target/release").join(bin);
                        std::fs::create_dir_all(dest.parent().unwrap()).unwrap();
                        std::fs::write(&dest, crate::elf::tests::fixture_elf()).unwrap();
                    }
                    return Ok(());
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
            if arg == "--output" {
                return write_new(Path::new(&args[i + 1]), &seed_bytes, 0o600);
            }
        }
        panic!("unexpected execution: {args:?}");
    }));
    (prod, calls, root)
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
        "cargo run --release --locked -p soda-release-assets --bin soda-forgejo-locales -- --lock frontend/forgejo/locale.lock.json --out ",
        "cargo run --release --locked -p soda-release-assets --bin soda-stage -- --arch x86_64 --host-context ",
        "STEP Build image: dashboard\n",
        "STEP Build image: project-os\n",
        "STEP Build image: tailnet\n",
        "STEP Build image: forgejo\n",
        "bun scripts/build-soda-extension.ts --out ",
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
    assert_eq!(text.matches(" save --format=oci-archive").count(), 6);
    assert!(!text.contains(" push ") && !text.contains(" --rm ") && !text.contains("--replace"));
    assert!(images.contains_key("proxy"));
    assert!(!images.contains_key("caddy"));
    for (name, produced) in &images {
        assert!(is_digest(&produced.archive_sha256));
        assert!(!produced.image.manifest.is_empty() && !produced.image.config.is_empty());
        assert!(PathBuf::from(&prod.out)
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
    assert!(!text.contains("STEP Build image: project-os"));
    assert!(!text.contains("STEP Export and verify"));
    assert!(PathBuf::from(&prod.out).join("app-inputs.json").exists());
}

#[test]
fn oracle_production_refusals() {
    // Oracle: TestProductionRefusesWrongToolchainInputsAndLayout.
    for mode in ["bun", "base", "tailnet", "forgejo", "platform", "revision"] {
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
            "tailnet" => {
                let raw = fixture_live_inputs();
                let mut text = raw.marshal();
                text = text.replacen("\"Version\": \"1.98.2\"", "\"Version\": \"yesterday\"", 1);
                std::fs::write(PathBuf::from(&prod.source).join("live-inputs.json"), text).unwrap();
            }
            "forgejo" => {
                assert!(prod.images("").is_err(), "{mode}");
                continue;
            }
            "platform" => prod.arch = "other".to_string(),
            "revision" => prod.revision = "dirty".to_string(),
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
fn oracle_compile_recipes() {
    // Oracle: TestProductionCompileKeepsLayoutAndVerifiesELF +
    // TestProductionCompileRustKeepsLayoutAndVerifiesELF.
    let (prod, calls, _root) = production_fixture();
    let dest = PathBuf::from(&prod.out).join("program");
    prod.compile("soda-host", "./cmd/soda-host", &dest.to_string_lossy())
        .unwrap();
    let text = calls.lock().unwrap().join("\n");
    assert!(text.contains("-buildvcs=false") && !text.contains("-tags="));
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
        text.contains("cargo build --release --locked")
            && text.contains("-p soda-identity-compose")
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
