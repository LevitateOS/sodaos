use super::tests::{file_run, image_id, temp_dir, test_view};
use super::*;
use crate::tailnet_domain::{ERR_CONFLICT, ERR_UNAVAILABLE, ERR_UNCONFIRMED};
use crate::tailnet_runtime::companion_create_args;

#[test]
fn resolver_uses_actual_inode_rather_than_generated_metadata() {
    let dir = temp_dir("resolver");
    let owned = dir.join("resolver");
    let other = dir.join("other");
    std::fs::write(&owned, b"nameserver 10.89.0.1\n").unwrap();
    std::fs::write(&other, b"nameserver 10.89.0.1\n").unwrap();
    let mut run = file_run();
    run.resolver = owned.to_str().unwrap().to_string();
    for same in [true, false] {
        let stat = |path: &str| -> Result<std::fs::Metadata, String> {
            let mapped = if path == "/proc/99/root/etc/resolv.conf" {
                if same {
                    &owned
                } else {
                    &other
                }
            } else {
                std::path::Path::new(path)
            };
            std::fs::metadata(mapped).map_err(|e| e.to_string())
        };
        let result = companion_resolver(&run, 99, &stat);
        assert_eq!(result.is_ok(), same, "resolver identity same={same}");
    }
    // A missing owned resolver never matches.
    run.resolver = dir.join("absent").to_str().unwrap().to_string();
    let stat = |path: &str| std::fs::metadata(path).map_err(|_| ERR_CONFLICT.to_string());
    assert_eq!(
        companion_resolver(&run, 99, &stat),
        Err(ERR_CONFLICT.to_string())
    );
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn record_requires_immutable_cid_recipe_and_running_incarnation() {
    let mut run = file_run();
    run.resolver = format!(
        "/var/lib/containers/storage/overlay-containers/{}/userdata/resolv.conf",
        run.target.container
    );
    let image = image_id();
    let id = "e".repeat(64);
    let args = companion_create_args(&run, &image).unwrap();
    let mut command = vec!["/usr/bin/podman".to_string()];
    command.extend(args);
    let original = CompanionRecord {
        id: id.clone(),
        image: image.clone(),
        command,
        running: true,
        pid: 99,
        started: "2026-09-12T12:00:00Z".to_string(),
        execs: Vec::new(),
    };
    validate_companion_record(&original, &run, &image, &id).unwrap();
    // A new release default is not authority to adopt an existing run's
    // companion.
    assert!(
        validate_companion_record(&original, &run, &format!("sha256:{}", "f".repeat(64)), &id)
            .is_err()
    );
    // Before first native start there is no daemon PID.
    let created = CompanionRecord {
        running: false,
        pid: 0,
        started: String::new(),
        ..original.clone()
    };
    validate_companion_record(&created, &run, &image, &id).unwrap();
    // The native argv[0] spelling is admitted.
    let native_spelling = CompanionRecord {
        command: {
            let mut cmd = vec!["podman".to_string()];
            cmd.extend(original.command[1..].iter().cloned());
            cmd
        },
        ..original.clone()
    };
    validate_companion_record(&native_spelling, &run, &image, &id).unwrap();
    // The digest prefix spelling is immaterial.
    let bare_image = CompanionRecord {
        image: "d".repeat(64),
        ..original.clone()
    };
    validate_companion_record(&bare_image, &run, &image, &id).unwrap();
    for kind in [
        "cid",
        "image",
        "namespace",
        "secret-env",
        "missing-command",
        "foreign-exec",
        "many-execs",
        "pid",
        "started",
        "started-text",
    ] {
        let mut changed = original.clone();
        match kind {
            "cid" => changed.id = "f".repeat(64),
            "image" => changed.image = format!("sha256:{}", "f".repeat(64)),
            "namespace" => {
                for arg in &mut changed.command {
                    if arg.starts_with("--network=") {
                        *arg = "--network=host".to_string();
                    }
                }
            }
            "secret-env" => changed
                .command
                .push("--env=TS_AUTHKEY=synthetic-secret".to_string()),
            "missing-command" => changed.command.clear(),
            "foreign-exec" => changed.execs = vec!["unknown".to_string()],
            "many-execs" => changed.execs = vec![id.clone(); 17],
            "pid" => changed.pid = 1,
            "started" => changed.started = "0001-01-01T00:00:00Z".to_string(),
            "started-text" => changed.started = "not-a-time".to_string(),
            _ => unreachable!(),
        }
        assert!(
            validate_companion_record(&changed, &run, &image, &id).is_err(),
            "changed companion admitted: {kind}"
        );
    }
}

#[test]
fn exec_identity_and_command_matchers() {
    let run = file_run();
    let id = "e".repeat(64);
    let rec = CompanionRecord {
        id: id.clone(),
        image: image_id(),
        ..CompanionRecord::default()
    };
    assert!(companion_identity_matches(&rec, &run, &image_id(), &id));
    assert!(!companion_identity_matches(
        &rec,
        &run,
        &image_id(),
        &"f".repeat(64)
    ));
    assert!(!companion_identity_matches(
        &rec,
        &run,
        &format!("sha256:{}", "f".repeat(64)),
        &id
    ));
    assert!(!companion_identity_matches(
        &rec,
        &run,
        &image_id(),
        "short"
    ));
    let args = vec!["a".to_string(), "b".to_string()];
    assert!(companion_command_matches(
        &[
            "/usr/bin/podman".to_string(),
            "a".to_string(),
            "b".to_string()
        ],
        &args
    ));
    assert!(companion_command_matches(
        &["podman".to_string(), "a".to_string(), "b".to_string()],
        &args
    ));
    assert!(!companion_command_matches(&["a".to_string()], &args));
    assert!(!companion_command_matches(&Vec::new(), &args));
    assert!(!companion_command_matches(
        &["/bin/podman".to_string(), "a".to_string(), "b".to_string()],
        &args
    ));
    assert!(companion_execs_valid(&[]).is_ok());
    assert!(companion_execs_valid(std::slice::from_ref(&id)).is_ok());
    assert_eq!(
        companion_execs_valid(&["unknown".to_string()]),
        Err(ERR_UNAVAILABLE.to_string())
    );
    assert_eq!(
        companion_execs_valid(&vec![id.clone(); 17]),
        Err(ERR_UNAVAILABLE.to_string())
    );
}

#[test]
fn inspect_decode_is_strict() {
    let good = br#"{"id":"abc","image":"sha256:00","command":["podman","a"],"running":true,"pid":99,"started":"2026-09-12T12:00:00Z","execs":[]}"#;
    let rec = decode_companion_record(good).unwrap();
    assert_eq!(rec.id, "abc");
    assert_eq!(rec.pid, 99);
    assert!(rec.running);
    assert_eq!(rec.command, vec!["podman".to_string(), "a".to_string()]);
    // Nulls bind zero values like encoding/json.
    let nulls = br#"{"id":null,"image":null,"command":null,"running":null,"pid":null,"started":null,"execs":null}"#;
    let rec = decode_companion_record(nulls).unwrap();
    assert_eq!(rec, CompanionRecord::default());
    for bad in [
        "not json",
        "[]",
        "{\"id\":\"a\",\"id\":\"b\",\"image\":\"i\",\"command\":[],\"running\":false,\"pid\":0,\"started\":\"\",\"execs\":[]}",
        "{\"id\":\"a\",\"image\":\"i\",\"command\":[],\"running\":false,\"pid\":0,\"started\":\"\",\"execs\":[],\"extra\":1}",
        "{\"id\":\"a\",\"image\":\"i\",\"command\":[],\"running\":false,\"pid\":1.5,\"started\":\"\",\"execs\":[]}",
        "{\"id\":\"a\",\"image\":\"i\",\"command\":{},\"running\":false,\"pid\":0,\"started\":\"\",\"execs\":[]}",
    ] {
        assert!(
            decode_companion_record(bad.as_bytes()).is_err(),
            "decoded: {bad}"
        );
    }
}

#[test]
fn preparation_stages_preserve_typed_causes() {
    for (stage, cause) in [
        ("project runtime not ready", ERR_UNAVAILABLE),
        ("Tailnet policy unconfirmed", ERR_CONFLICT),
        ("companion startup unconfirmed", ERR_UNCONFIRMED),
        ("companion status unavailable", ERR_UNAVAILABLE),
        ("enrollment unconfirmed", ERR_UNCONFIRMED),
    ] {
        let tagged = preparation_error(stage, Some(cause.to_string())).expect("tagged");
        assert!(tagged.contains(stage), "stage lost: {tagged}");
        assert!(tagged.contains(cause), "cause lost: {tagged}");
    }
    assert_eq!(preparation_error("stage", None), None);
}

#[test]
fn run_freshness_and_idle_state() {
    let run = file_run();
    let mut other = run.clone();
    other.target.run = "d".repeat(64);
    assert!(!is_companion_run_fresh(None, &run, &run));
    assert!(is_companion_run_fresh(None, &other, &run));
    let missing = "runtime record not found".to_string();
    assert!(is_companion_run_fresh(Some(&missing), &run, &run));
    let other_err = ERR_UNAVAILABLE.to_string();
    assert!(!is_companion_run_fresh(Some(&other_err), &other, &run));

    let mut view = test_view();
    view.enabled = true;
    assert!(apply_companion_idle_state(&mut view, true));
    assert_eq!(view.state, "");
    assert!(!apply_companion_idle_state(&mut view, false));
    assert_eq!(view.state, "");
    view.enabled = false;
    assert!(!apply_companion_idle_state(&mut view, true));
    assert!(!apply_companion_idle_state(&mut view, false));
    assert_eq!(view.state, "off");
}

#[test]
fn namespaces_match_only_exact_incarnation() {
    let run = file_run();
    let stopped = CompanionRecord {
        running: false,
        ..CompanionRecord::default()
    };
    assert!(match_companion_namespaces(&stopped, &run).is_ok());
    // PID 1 or below can never be a companion incarnation.
    let init = CompanionRecord {
        running: true,
        pid: 1,
        ..CompanionRecord::default()
    };
    assert_eq!(
        match_companion_namespaces(&init, &run),
        Err(ERR_CONFLICT.to_string())
    );
    // An absent process never matches.
    let absent = CompanionRecord {
        running: true,
        pid: 4242424242,
        ..CompanionRecord::default()
    };
    assert_eq!(
        match_companion_namespaces(&absent, &run),
        Err(ERR_CONFLICT.to_string())
    );
    // A live process with synthetic namespaces never matches either.
    let live = CompanionRecord {
        running: true,
        pid: std::process::id() as i64,
        ..CompanionRecord::default()
    };
    assert_eq!(
        match_companion_namespaces(&live, &run),
        Err(ERR_CONFLICT.to_string())
    );
}
