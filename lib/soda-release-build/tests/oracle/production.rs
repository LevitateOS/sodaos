//! Production oracle vectors: the full native sequence with recorded
//! observations against Go ground truth.

use super::{data_path, oracle, oracle_live_inputs, scratch, FIXTURE_REVISION};
use soda_release_build::coreos_stream::write_live_inputs;
use soda_release_build::oci::inspect_oci;
use soda_release_build::production::Production;
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
            "system/host/services/forgejo.container",
            "[Container]\nImage=codeberg.org/forgejo/forgejo:15.0.9\n",
        ),
        (
            "system/host/services/soda-proxy.container",
            "[Container]\nImage=docker.io/library/caddy:2\n",
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
    let seed_image = inspect_oci(&seed_path, "x86_64", FIXTURE_REVISION).unwrap();
    assert_eq!(seed_image.config, oracle::OCI_CONFIG);
    let live_path = root.join("live-inputs.json");
    write_live_inputs(&live_path, &oracle_live_inputs()).unwrap();
    let calls = Arc::new(std::sync::Mutex::new(Vec::<String>::new()));
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
    let calls_capture = calls.clone();
    let config = seed_image.config.clone();
    prod.capture = Some(Box::new(move |_dir, name, args| {
        calls_capture
            .lock()
            .unwrap()
            .push(format!("{name} {}", args.join(" ")));
        if name == "bun" {
            return Ok("1.4.2".to_string());
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
                        let dest = PathBuf::from(dir).join("target/release").join(bin);
                        std::fs::create_dir_all(dest.parent().unwrap()).unwrap();
                        std::fs::write(&dest, fixture_elf()).unwrap();
                    }
                    return Ok(());
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
            if arg == "--output" {
                return soda_release_build::files::write_new(
                    std::path::Path::new(&args[i + 1]),
                    &seed_bytes,
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
    assert_eq!(images.len(), 6);
    let root_str = root.to_string_lossy().into_owned();
    let normalized: Vec<String> = calls
        .lock()
        .unwrap()
        .iter()
        .map(|call| call.replace(&root_str, "$ROOT"))
        .collect();
    assert_eq!(normalized.join("\n") + "\n", oracle::PRODUCTION_SEQUENCE);
    let app_inputs = std::fs::read_to_string(out.join("app-inputs.json")).unwrap();
    assert_eq!(
        app_inputs.replace(&root_str, "$ROOT"),
        oracle::APP_INPUTS_JSON
    );
}
