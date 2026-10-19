//! Current production command plan exercised through fake capture/execute hooks.
//! Hooks write synthetic ELF files and replay the retained OCI input. Command
//! text/order and staged records are checked; this does not run native builds.

use super::{data_path, oracle, scratch, FIXTURE_REVISION};
use soda_release_build::production::Production;
use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;
use std::sync::Arc;

fn fixture_elf() -> [u8; 64] {
    let mut elf = [0u8; 64];
    elf[0..7].copy_from_slice(&[0x7f, b'E', b'L', b'F', 2, 1, 1]);
    elf[16..18].copy_from_slice(&2u16.to_le_bytes());
    elf[18..20].copy_from_slice(&62u16.to_le_bytes());
    elf[20..24].copy_from_slice(&1u32.to_le_bytes());
    elf[52..54].copy_from_slice(&64u16.to_le_bytes());
    elf
}

#[test]
fn oracle_production_sequence() {
    let root = scratch("prodseq");
    let out = root.join(".artifacts/native/x86_64");
    std::fs::create_dir_all(&out).unwrap();
    for (name, body) in [
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
    ] {
        let path = root.join(name);
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(&path, body).unwrap();
    }
    // The Go-built seed keeps config IDs identical to the oracle.
    let seed_bytes = std::fs::read(data_path("go-fixture.oci")).unwrap();
    let seed_path = root.join("seed.oci");
    std::fs::write(&seed_path, &seed_bytes).unwrap();
    let seed_image = soda_release_deliver::oci::inspect_oci(
        seed_path.to_str().unwrap(),
        "x86_64",
        FIXTURE_REVISION,
    )
    .unwrap();
    assert_eq!(seed_image.config, oracle::OCI_CONFIG);
    // Input freezing uses no floating toolchain file: the pinned official
    // Tailscale image joins the same frozen pull record.
    let calls = Arc::new(std::sync::Mutex::new(Vec::<String>::new()));
    let mut prod = Production::default();
    prod.source = root.to_string_lossy().into_owned();
    prod.native = out.to_string_lossy().into_owned();
    prod.out = out.to_string_lossy().into_owned();
    prod.arch = "x86_64".to_string();
    prod.revision = FIXTURE_REVISION.to_string();
    let mut live = super::oracle_live_inputs();
    live.fountain.image = format!("ghcr.io/example/fountain@sha256:{}", "b".repeat(64));
    live.fountain.image_config = seed_image.config.clone();
    prod.forgejo_revision = live.fountain.source_revision.clone();
    prod.live_inputs = root.join("live-inputs.json").to_string_lossy().into_owned();
    soda_release_build::live_inputs::write_live_inputs(
        std::path::Path::new(&prod.live_inputs),
        &live,
    )
    .unwrap();

    let calls_capture = calls.clone();
    let config = seed_image.config.clone();
    let fountain_identity=serde_json::json!({"os":"linux","architecture":"amd64","revision":live.fountain.source_revision,"version":live.fountain.version}).to_string();
    prod.capture = Some(Box::new(move |_dir, name, args| {
        calls_capture
            .lock()
            .unwrap()
            .push(format!("{name} {}", args.join(" ")));
        if name == "git" {
            return Ok("1735689600".into());
        }
        if args
            .get(4)
            .is_some_and(|format| format.contains("architecture"))
        {
            return Ok(fountain_identity.clone());
        }
        if args[1] == "pull" {
            return Ok(config.clone());
        }
        Ok(format!("sha256:{}", "b".repeat(64)))
    }));
    let calls_exec = calls.clone();
    let image_config = seed_image.config.clone();
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
                    std::fs::write(&dest, fixture_elf()).unwrap();
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
                        std::fs::write(&dest, fixture_elf()).unwrap();
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
                    return soda_release_build::files::write_new(
                        std::path::Path::new(path),
                        &seed_bytes,
                        0o600,
                    );
                }
            }
        }
        if name != "podman" {
            return Ok(());
        }
        for (i, arg) in args.iter().enumerate() {
            if arg == "--iidfile" {
                return soda_release_build::files::write_new(
                    std::path::Path::new(&args[i + 1]),
                    image_config.as_bytes(),
                    0o600,
                );
            }
        }
        panic!("unexpected execution: {args:?}");
    }));
    let host = out.join("context").to_string_lossy().into_owned();
    let forgejo = out.join("forgejo-context").to_string_lossy().into_owned();
    prod.dependencies().unwrap();
    prod.resolve_inputs().unwrap();
    prod.assets(&host, &forgejo).unwrap();
    let images = prod.images(&forgejo).unwrap();
    // Four built roles with sealed local archives, two upstream roles
    // bound by identity without a duplicate archive.
    assert_eq!(images.len(), 6);
    for name in ["dashboard", "forgejo", "project-os", "canlang-runtime"] {
        assert!(!images[name].archive_sha256.is_empty(), "{name} archive");
    }
    for name in ["proxy", "postgres"] {
        assert!(images[name].archive_sha256.is_empty(), "{name} bind");
        assert!(images[name].image.config.starts_with("sha256:"));
        assert!(images[name].image.manifest.starts_with("sha256:"));
    }
    let root_str = root.to_string_lossy().into_owned();
    let normalized: Vec<String> = calls
        .lock()
        .unwrap()
        .iter()
        .map(|call| call.replace(&root_str, "$ROOT"))
        .collect();
    assert_eq!(normalized.join("\n") + "\n", oracle::PRODUCTION_SEQUENCE);
    let app_inputs_path = out.join("app-inputs.json");
    let app_inputs_bytes = std::fs::read(&app_inputs_path).unwrap();
    assert!(app_inputs_bytes.ends_with(b"\n"));
    assert_eq!(
        std::fs::metadata(&app_inputs_path)
            .unwrap()
            .permissions()
            .mode()
            & 0o777,
        0o600
    );
    let app_inputs: serde_json::Value = serde_json::from_slice(&app_inputs_bytes).unwrap();
    let entries = app_inputs.as_array().unwrap();
    assert_eq!(entries.len(), 5);
    let requests = [
        "docker.io/rockylinux/rockylinux:10.2",
        "ghcr.io/example/fountain@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
        "docker.io/library/caddy:2",
        "docker.io/library/postgres:17",
        "docker.io/tailscale/tailscale@sha256:99691662a2e82fbfd2bf55db0fd52bacd2f4664ce6fd1deca3746a024144a2b7",
    ];
    for (entry, request) in entries.iter().zip(requests) {
        assert_eq!(entry["Requested"], request);
        // The recorded reference pins the exact platform manifest digest;
        // the repository part never carries a tag or digest.
        let bare = request.split('@').next().unwrap();
        let repo = match bare.rfind(':') {
            Some(colon) if colon > bare.rfind('/').unwrap_or(0) => &bare[..colon],
            _ => bare,
        };
        assert_eq!(
            entry["Reference"],
            format!("{repo}@sha256:{}", "b".repeat(64))
        );
        assert_eq!(entry["Config"], oracle::OCI_CONFIG);
    }
}
