use super::*;

fn call() -> LifecycleCall {
    LifecycleCall {
        controller_id: "soda".into(),
        controller_instance_id: "d".repeat(32),
        package_id: "tailnet".into(),
        package_version: "1.0.0".into(),
        package_sha256: "b".repeat(64),
        installation_id: "550e8400-e29b-41d4-a716-446655440000".into(),
        generation: "7".into(),
        operation: "activate".into(),
        deadline_unix: 1792100300,
    }
}

fn document() -> serde_json::Value {
    let call = call();
    serde_json::json!({
        "catalog_sha256": "c".repeat(64), "controller_id": "soda",
        "controller_instance_id": call.controller_instance_id, "controller_ready": true, "admission_open": false,
        "entry": {"package_id":"tailnet", "installation_id":call.installation_id,
            "generation":"7", "desired_enabled":true, "pending_operation":"activate",
            "installed_version":"1.0.0", "package_sha256":call.package_sha256, "required":false}
    })
}

fn state(value: &serde_json::Value) -> ManagerState {
    crate::json::decode_strict_as(&serde_json::to_vec(value).unwrap()).unwrap()
}

#[test]
fn ordinary_start_requires_current_manager_and_service_epoch() {
    let image = format!("sha256:{}", "a".repeat(64));
    let mut value = document();
    let grant = state(&value)
        .activation_grant(&call(), &image, &"c".repeat(64))
        .unwrap();
    assert!(
        !state(&value).admits(&grant, &image),
        "activation alone cannot open ordinary admission"
    );
    value["entry"]["pending_operation"] = serde_json::Value::Null;
    value["admission_open"] = true.into();
    assert!(state(&value).admits(&grant, &image));
    assert!(
        state(&value)
            .activation_grant(&call(), &image, &"c".repeat(64))
            .is_err(),
        "no stable-intent lifecycle bypass"
    );
    for (key, replacement) in [
        ("controller_instance_id", serde_json::json!("e".repeat(32))),
        ("catalog_sha256", serde_json::json!("f".repeat(64))),
        ("controller_ready", false.into()),
        ("admission_open", false.into()),
    ] {
        let mut changed = value.clone();
        changed[key] = replacement;
        assert!(
            !state(&changed).admits(&grant, &image),
            "stale {key} admitted"
        );
    }
    for (key, replacement) in [
        (
            "installation_id",
            serde_json::json!("550e8400-e29b-41d4-a716-446655440001"),
        ),
        ("generation", "8".into()),
        ("desired_enabled", false.into()),
        ("package_sha256", serde_json::json!("a".repeat(64))),
        ("installed_version", "2.0.0".into()),
        ("pending_operation", "deactivate".into()),
    ] {
        let mut changed = value.clone();
        changed["entry"][key] = replacement;
        assert!(
            !state(&changed).admits(&grant, &image),
            "stale {key} admitted"
        );
    }
    let mut stale = grant.clone();
    stale.service_epoch.start.push('0');
    assert!(!state(&value).admits(&stale, &image));
    assert!(!state(&value).admits(&grant, &format!("sha256:{}", "f".repeat(64))));
}

#[test]
fn missing_nullable_state_is_not_stable_intent() {
    let mut value = document();
    value["entry"]
        .as_object_mut()
        .unwrap()
        .remove("pending_operation");
    assert!(
        crate::json::decode_strict_as::<ManagerState>(&serde_json::to_vec(&value).unwrap())
            .is_err()
    );
    let duplicate = serde_json::to_string(&document()).unwrap().replacen(
        "\"generation\":\"7\"",
        "\"generation\":\"7\",\"generation\":\"7\"",
        1,
    );
    assert!(crate::json::decode_strict_as::<ManagerState>(duplicate.as_bytes()).is_err());
}

#[test]
fn service_epoch_dies_with_its_process() {
    let mut child = std::process::Command::new("/usr/bin/sleep")
        .arg("10")
        .spawn()
        .unwrap();
    let epoch = ServiceEpoch::capture(child.id()).unwrap();
    assert!(epoch.live());
    child.kill().unwrap();
    child.wait().unwrap();
    assert!(!epoch.live());
}

#[test]
fn closing_grant_precedes_drain_and_shared_start_exclusion() {
    let root = std::env::temp_dir().join(format!(
        "ta-{}",
        crate::tcontrol_policy::new_revision().unwrap()
    ));
    let store = GrantStore {
        files: crate::tailnet_files::OwnedState {
            root: root.clone(),
            owner: unsafe { libc::geteuid() },
        },
    };
    let image = format!("sha256:{}", "a".repeat(64));
    let grant = state(&document())
        .activation_grant(&call(), &image, &"c".repeat(64))
        .unwrap();
    store.publish(&grant).unwrap();
    assert_eq!(store.read().unwrap().generation, "7");
    let shared = store
        .start_lock(Instant::now() + Duration::from_secs(1))
        .unwrap();
    store.close().unwrap();
    assert!(
        store.read().is_err(),
        "grant must close before waiting for active starts"
    );
    assert!(store
        .drain_lock(Instant::now() + Duration::from_millis(40))
        .is_err());
    drop(shared);
    let drain = store
        .drain_lock(Instant::now() + Duration::from_secs(1))
        .unwrap();
    assert!(store
        .start_lock(Instant::now() + Duration::from_millis(40))
        .is_err());
    drop(drain);
    fs::remove_dir_all(root).unwrap();
}
