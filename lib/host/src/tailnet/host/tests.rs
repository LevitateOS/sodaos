use super::*;
use std::os::unix::fs::DirBuilderExt;
use std::sync::{Arc, Mutex};

struct NativeState {
    exists: bool,
    running: bool,
    foreign: bool,
    stops: usize,
    recipe: Vec<String>,
    record: HostRecord,
}
struct Fake(Arc<Mutex<NativeState>>);

impl Executor for Fake {
    fn run(&self, _: &[u8], cmd: &str, args: &[&str], _: Instant) -> Result<Vec<u8>, String> {
        assert_eq!(cmd, "/usr/bin/podman");
        assert_eq!(args[0], "--remote=false");
        let mut state = self.0.lock().unwrap();
        match args[1] {
            "ps" => Ok(if state.exists {
                state.record.id.as_bytes().to_vec()
            } else {
                Vec::new()
            }),
            "inspect" => {
                let mut command = vec!["/usr/bin/podman".to_string()];
                command.extend(state.recipe.clone());
                if state.foreign {
                    command.push("--privileged".into());
                }
                Ok(serde_json::to_vec(
                    &serde_json::json!({"id":state.record.id,"image":state.record.image,
                    "command":command,"running":state.running,"pid":std::process::id(),
                    "role":"host","installation":state.record.installation_id}),
                )
                .unwrap())
            }
            "stop" => {
                assert_eq!(args.last().copied(), Some(state.record.id.as_str()));
                state.stops += 1;
                state.running = false;
                Ok(Vec::new())
            }
            "rm" => {
                assert!(!state.running);
                state.exists = false;
                Ok(Vec::new())
            }
            other => panic!("unexpected runtime command {other}"),
        }
    }
}

fn fixture() -> (HostRuntime<Fake>, Arc<Mutex<NativeState>>, PathBuf) {
    let root = std::env::temp_dir().join(format!(
        "th-{}",
        crate::tcontrol_policy::new_revision().unwrap()
    ));
    let record = HostRecord {
        id: "a".repeat(64),
        image: format!("sha256:{}", "b".repeat(64)),
        installation_id: "550e8400-e29b-41d4-a716-446655440000".into(),
        generation: "3".into(),
    };
    let state = Arc::new(Mutex::new(NativeState {
        exists: true,
        running: true,
        foreign: false,
        stops: 0,
        recipe: Vec::new(),
        record: record.clone(),
    }));
    let runtime = HostRuntime {
        exec: Fake(state.clone()),
        state: OwnedState {
            root: root.clone(),
            owner: unsafe { libc::geteuid() },
        },
        control: root.join("runtime/host/control"),
    };
    state.lock().unwrap().recipe = runtime.recipe(&record).unwrap();
    runtime.state.write("host-runtime.json", &record).unwrap();
    (runtime, state, root)
}

#[test]
fn exact_owned_old_image_stops_without_grant_or_image_selection() {
    let (runtime, state, root) = fixture();
    runtime
        .stop(
            Some("550e8400-e29b-41d4-a716-446655440000"),
            Instant::now() + Duration::from_secs(1),
        )
        .unwrap();
    assert_eq!(state.lock().unwrap().stops, 1);
    assert!(!state.lock().unwrap().exists);
    assert!(
        root.join("host-runtime.json").exists(),
        "ownership and credentials are preserved"
    );
    runtime
        .stop(None, Instant::now() + Duration::from_secs(1))
        .unwrap();
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn modified_recipe_or_installation_never_authorizes_stop() {
    let (runtime, state, root) = fixture();
    state.lock().unwrap().foreign = true;
    assert!(runtime
        .stop(None, Instant::now() + Duration::from_secs(1))
        .is_err());
    state.lock().unwrap().foreign = false;
    assert!(runtime
        .stop(
            Some("550e8400-e29b-41d4-a716-446655440001"),
            Instant::now() + Duration::from_secs(1)
        )
        .is_err());
    assert_eq!(state.lock().unwrap().stops, 0);
    assert!(state.lock().unwrap().running);
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn native_recipe_has_no_separator_or_mutable_image() {
    let (runtime, state, root) = fixture();
    let state = state.lock().unwrap();
    let position = state
        .recipe
        .iter()
        .position(|arg| arg == &state.record.image)
        .unwrap();
    assert_eq!(
        state.recipe[position + 1],
        "--state=/var/lib/tailscale/tailscaled.state"
    );
    assert!(!state.recipe.iter().any(|arg| arg == "--"));
    assert!(state.recipe.iter().any(|arg| arg == "--pull=never"));
    drop(state);
    drop(runtime);
    fs::remove_dir_all(root).unwrap();
}

// Small native owner check, explicitly selected only in the disposable VM.
// Authority admission and the Go peer chain have their own independent checks.
#[test]
fn native_official_host_lifecycle() {
    if std::env::var("SODA_NATIVE_TAILNET_HOST_GATE").as_deref() != Ok("1") {
        return;
    }
    assert_eq!(unsafe { libc::geteuid() }, 0);
    let root = std::env::temp_dir().join(format!(
        "host-native-{}",
        crate::tcontrol_policy::new_revision().unwrap()
    ));
    fs::DirBuilder::new().mode(0o700).create(&root).unwrap();
    let runtime = HostRuntime {
        exec: crate::project::Native,
        state: OwnedState {
            root: root.join("state"),
            owner: 0,
        },
        control: root.join("runtime/host/control"),
    };
    let store = crate::tailnet_admission::GrantStore {
        files: OwnedState {
            root: root.join("runtime"),
            owner: 0,
        },
    };
    let deadline = Instant::now() + Duration::from_secs(45);
    let result = (|| -> Result<(), String> {
        let grant = Grant {
            installation_id: "550e8400-e29b-41d4-a716-446655440000".into(),
            generation: "7".into(),
            controller_instance_id: "d".repeat(32),
            package_version: "1.0.0".into(),
            package_sha256: "b".repeat(64),
            catalog_sha256: "c".repeat(64),
            image: "sha256:7f6fc04f2deab0d0c309f29d7d27517579ebb7047ad23907d9fa2797fa7227df".into(),
            service_epoch: crate::tailnet_admission::ServiceEpoch::capture(std::process::id())?,
        };
        let starts = store.start_lock(deadline)?;
        let cid = runtime.ensure(&grant, &starts, deadline)?;
        let before = runtime
            .inspect(Some(&grant.installation_id), deadline)?
            .ok_or_else(unconfirmed)?;
        let body = runtime.cli(&["status".into(), "--json".into()], deadline)?;
        let value: serde_json::Value = serde_json::from_slice(&body).map_err(|_| unconfirmed())?;
        if value.get("BackendState").and_then(|v| v.as_str()) != Some("NeedsLogin") {
            return Err("native host did not report NeedsLogin".into());
        }
        use sha2::Digest;
        let state = runtime.state.root.join("host/tailscaled.state");
        let digest = sha2::Sha256::digest(fs::read(&state).map_err(|_| unconfirmed())?);
        if runtime.ensure(&grant, &starts, deadline)? != cid {
            return Err("native adoption changed CID".into());
        }
        let after = runtime
            .inspect(Some(&grant.installation_id), deadline)?
            .ok_or_else(unconfirmed)?;
        if before.pid != after.pid {
            return Err("native adoption restarted daemon".into());
        }
        store.publish(&grant)?;
        store.close()?;
        drop(starts);
        let _drain = store.drain_lock(deadline)?;
        runtime.stop(Some(&grant.installation_id), deadline)?;
        if runtime
            .inspect(Some(&grant.installation_id), deadline)?
            .is_some()
        {
            return Err("native host survived stop".into());
        }
        if runtime.control.join("tailscaled.sock").exists() {
            return Err("native socket survived stop".into());
        }
        if sha2::Sha256::digest(fs::read(&state).map_err(|_| unconfirmed())?) != digest {
            return Err("native stop changed credentials".into());
        }
        Ok(())
    })();
    // Never force-remove a name or unknown resource after a failed assertion.
    let cleanup = runtime.stop(None, Instant::now() + Duration::from_secs(15));
    if cleanup.is_ok() {
        fs::remove_dir_all(&root).unwrap();
    }
    assert!(
        cleanup.is_ok(),
        "owned native cleanup unconfirmed; retain {}",
        root.display()
    );
    assert!(result.is_ok(), "native host gate failed: {result:?}");
}
