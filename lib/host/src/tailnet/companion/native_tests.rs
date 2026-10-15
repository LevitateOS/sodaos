use super::tests::{project_id, test_binding, test_view, MockTailnet};
use super::*;
use crate::project::Native;
use crate::tailnet_admission::{Grant, GrantStore, ServiceEpoch};
use crate::tailnet_files::read_companion_record;
use crate::tailnet_runtime::project_run;
use std::time::{Duration, Instant};

#[test]
fn native_owned_child_stops_after_parent_exit() {
    if std::env::var("SODA_NATIVE_TAILNET_COMPANION_GATE").as_deref() != Ok("1") {
        return;
    }
    assert_eq!(unsafe { libc::geteuid() }, 0);
    let id = project_id();
    let name = format!("soda-{id}");
    let deadline = Instant::now() + Duration::from_secs(50);
    let podman = |args: &[String]| crate::tailnet_host::podman(&Native, args, deadline);
    let prior = podman(&[
        "ps".into(),
        "-a".into(),
        "--format={{.ID}}".into(),
        "--filter".into(),
        format!("name=^{name}$"),
    ])
    .unwrap();
    assert!(prior.is_empty(), "fixture parent name is already owned");
    let runtime_dir = std::path::Path::new(RUNTIME_ROOT).join(&id);
    assert!(
        !runtime_dir.exists(),
        "fixture runtime directory is already owned"
    );
    let parent = podman(&[
        "create".into(),
        "--name".into(),
        name,
        "--label".into(),
        format!("org.soda.project={id}"),
        "--label".into(),
        "org.soda.owner=42".into(),
        "--network=soda-projects".into(),
        "--userns=auto:size=262144".into(),
        "--systemd=always".into(),
        "--cgroupns=private".into(),
        "--cap-add=SYS_ADMIN,MKNOD,NET_ADMIN,SYS_PTRACE".into(),
        "--device=/dev/fuse".into(),
        "--security-opt=label=disable".into(),
        "--pull=never".into(),
        "sha256:f67487336845858e59d845be3360262d58de6920a6086ad45939d730a08ad6d1".into(),
    ])
    .unwrap();
    let parent = std::str::from_utf8(&parent).unwrap().trim().to_string();
    assert!(crate::tailnet_domain::valid_container_id(&parent));
    let mut tailnet = MockTailnet::inert();
    tailnet.project_fn = Box::new(|_, _| {
        Ok(crate::tailnet_domain::ProjectView {
            enabled: true,
            ..test_view()
        })
    });
    tailnet.binding_fn = Box::new(|_| Ok(test_binding()));
    let companion = Companion {
        exec: Native,
        tailnet,
        image: "sha256:7f6fc04f2deab0d0c309f29d7d27517579ebb7047ad23907d9fa2797fa7227df".into(),
        enabled_check: None,
    };
    let result = (|| -> Result<(), String> {
        podman(&["start".into(), parent.clone()])?;
        let grant = Grant {
            installation_id: "550e8400-e29b-41d4-a716-446655440000".into(),
            generation: "7".into(),
            controller_instance_id: "d".repeat(32),
            package_version: "1.0.0".into(),
            package_sha256: "b".repeat(64),
            catalog_sha256: "c".repeat(64),
            image: companion.image.clone(),
            service_epoch: ServiceEpoch::capture(std::process::id())?,
        };
        let starts = GrantStore::default().start_lock(deadline)?;
        let cid = companion.start_tailnet_admitted(&id, &grant, &starts, deadline)?;
        let run = project_run(&Native, &id, deadline)?;
        let record = read_companion_record(RUNTIME_ROOT, &run.target, 0, 0)?;
        if record.id != cid
            || record.parent != run
            || record.installation_id != grant.installation_id
            || record.generation != "7"
        {
            return Err("owned run receipt changed".into());
        }
        let before = companion.inspect_companion(&run, deadline)?;
        let adopted = companion.start_tailnet_admitted(&id, &grant, &starts, deadline)?;
        let after = companion.inspect_companion(&run, deadline)?;
        if cid != adopted || before.pid != after.pid {
            return Err("owned child adoption restarted daemon".into());
        }
        drop(starts);
        podman(&["stop".into(), "--time=5".into(), parent.clone()])?;
        let receipt = runtime_dir.join(&run.target.run).join("companion.json");
        let original =
            std::fs::read(&receipt).map_err(|_| "native receipt unavailable".to_string())?;
        let mut changed: serde_json::Value =
            serde_json::from_slice(&original).map_err(|_| "native receipt invalid".to_string())?;
        changed["generation"] = "8".into();
        std::fs::write(&receipt, serde_json::to_vec(&changed).unwrap())
            .map_err(|_| "native fixture mutation failed".to_string())?;
        let refused = companion.native_children(&grant.installation_id, None, deadline);
        if !refused.unknown || !refused.children.is_empty() {
            return Err("conflicting receipt authorized cleanup".into());
        }
        std::fs::write(&receipt, &original)
            .map_err(|_| "native fixture restore failed".to_string())?;
        std::fs::remove_file(&receipt)
            .map_err(|_| "native fixture receipt retirement failed".to_string())?;
        let inventory = companion.native_children(&grant.installation_id, None, deadline);
        if inventory.unknown
            || inventory.children.len() != 1
            || inventory.children[0].record.id != cid
        {
            return Err("native reboot inventory lost exact ownership".into());
        }
        companion.stop_native_child(&inventory.children[0], deadline)?;
        let after = companion.native_children(&grant.installation_id, None, deadline);
        if after.unknown || after.children.iter().any(|child| child.record.running) {
            return Err("owned child survived parentless cleanup".into());
        }
        Ok(())
    })();
    // Cleanup uses the same ownership checks; retain evidence on an unknown child.
    let cleanup = (|| -> Result<(), String> {
        let end = Instant::now() + Duration::from_secs(15);
        let inventory =
            companion.native_children("550e8400-e29b-41d4-a716-446655440000", None, end);
        if inventory.unknown {
            return Err("native fixture inventory is unknown".into());
        }
        for child in inventory.children {
            companion.stop_native_child(&child, end)?;
            crate::tailnet_host::podman(&Native, &["rm".into(), child.record.id], end)?;
        }
        Ok(())
    })();
    if cleanup.is_ok() {
        let _ = crate::tailnet_host::podman(
            &Native,
            &["stop".into(), "--time=5".into(), parent.clone()],
            Instant::now() + Duration::from_secs(8),
        );
        crate::tailnet_host::podman(
            &Native,
            &["rm".into(), parent.clone()],
            Instant::now() + Duration::from_secs(5),
        )
        .unwrap();
        std::fs::remove_dir_all(&runtime_dir).unwrap();
    }
    assert!(
        cleanup.is_ok(),
        "native child cleanup unconfirmed: {cleanup:?}; result {result:?}; retain parent CID {parent} and runtime {}",
        runtime_dir.display()
    );
    assert!(result.is_ok(), "native companion gate failed: {result:?}");
}
