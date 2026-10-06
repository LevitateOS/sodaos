use super::*;

fn stat_bytes(mode: &str) -> Vec<u8> {
    let mut fields: Vec<String> = "S 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 456 1"
        .split_whitespace()
        .map(str::to_string)
        .collect();
    if mode == "dead" {
        fields[0] = "Z".to_string();
    }
    if mode == "missing start" {
        fields[19] = "0".to_string();
    }
    let pid = if mode == "wrong pid" { "124" } else { "123" };
    format!("{pid} (comm with ) spaces) {}", fields.join(" ")).into_bytes()
}

fn mode_read(mode: &'static str) -> ReadFile {
    Box::new(move |path: &str| -> Result<Vec<u8>, String> {
        match path {
            "/proc/123/stat" => Ok(stat_bytes(mode)),
            "/proc/123/uid_map" | "/proc/123/gid_map" => {
                let mut data = "0 524288 262144\n".to_string();
                if mode == "extra map" {
                    data.push_str("300000 900000 1\n");
                }
                if mode == "root map" {
                    data = "0 0 262144\n".to_string();
                }
                Ok(data.into_bytes())
            }
            "/proc/sys/kernel/random/boot_id" => {
                if mode == "bad boot" {
                    return Ok(b"bad".to_vec());
                }
                Ok(b"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\n".to_vec())
            }
            _ => Err("unexpected proc input".to_string()),
        }
    })
}

fn mode_link(mode: &'static str) -> LinkFile {
    Box::new(move |path: &str| -> Result<String, String> {
        if mode == "missing namespace" {
            return Err("gone".to_string());
        }
        let kind = if path.ends_with("/net") {
            "net"
        } else {
            "user"
        };
        let mut inode = "11";
        if path.contains("/123/") {
            inode = "22";
        }
        if mode.strip_prefix("host ") == Some(kind) {
            inode = "11";
        }
        Ok(format!("{kind}:[{inode}]"))
    })
}

#[test]
fn process_identity_matrix() {
    for mode in [
        "valid",
        "host user",
        "host net",
        "extra map",
        "root map",
        "wrong pid",
        "dead",
        "missing start",
        "bad boot",
        "missing namespace",
    ] {
        let out = process_run_identity(123, mode_read(mode), mode_link(mode));
        if mode == "valid" {
            let identity = out.expect("valid identity must admit");
            assert_eq!(identity.uid, 524288, "{mode}");
            assert_eq!(identity.gid, 524288, "{mode}");
            assert_eq!(identity.start, "456", "{mode}");
            assert_eq!(identity.userns, "user:[22]", "{mode}");
            assert_eq!(identity.netns, "net:[22]", "{mode}");
            assert_eq!(
                identity.boot, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
                "{mode}"
            );
        } else {
            assert!(out.is_err(), "{mode}: unsafe identity accepted");
        }
    }
}

#[test]
fn process_identity_rejects_non_root_pid() {
    let out = process_run_identity(1, mode_read("valid"), mode_link("valid"));
    assert_eq!(out.unwrap_err(), ERR_CONFLICT);
}

fn fixture_run() -> ProjectRun {
    ProjectRun {
        target: RunTarget {
            project: format!("p{}", "a".repeat(24)),
            container: "b".repeat(64),
            run: "c".repeat(64),
        },
        uid: 524288,
        gid: 524288,
        ..Default::default()
    }
}

#[test]
fn companion_recipe_keeps_fixed_namespaces() {
    let run = fixture_run();
    let image = format!("sha256:{}", "d".repeat(64));
    let args = companion_create_args(&run, &image).expect("recipe must build");
    let joined = args.join(" ");
    for want in [
        format!("--userns=container:{}", run.target.container),
        format!("--network=container:{}", run.target.container),
        "--pid=private".to_string(),
        "--ipc=private".to_string(),
        "--uts=private".to_string(),
        "--cgroupns=private".to_string(),
        "--cap-drop=ALL".to_string(),
        "--cap-add=NET_ADMIN".to_string(),
        "--device=/dev/net/tun".to_string(),
        "--no-hosts".to_string(),
        "--log-driver=none".to_string(),
        "--pull=never".to_string(),
        image.clone(),
        "--entrypoint=/usr/local/bin/tailscaled".to_string(),
        "--state=mem:".to_string(),
        "--no-logs-no-support".to_string(),
    ] {
        assert!(
            joined.contains(want.as_str()),
            "missing fixed recipe argument {want}"
        );
    }
    for forbidden in [
        "--privileged",
        "--rm",
        "--replace",
        "--env",
        "SYS_MODULE",
        ":U",
        "--network=host",
        "--pid=host",
        "--userns=host",
        "/var/lib/soda-tailnet",
        "/run/podman",
        "/proc/",
        "tskey",
        "client_secret",
    ] {
        assert!(
            !joined.contains(forbidden),
            "unsafe companion argument {forbidden}"
        );
    }
    let mut count = 0;
    for (i, arg) in args.iter().enumerate() {
        if arg == "--volume" {
            count += 1;
            let mount = &args[i + 1];
            let prefix = format!(
                "/run/soda-tailnet/{}/{}/",
                run.target.project, run.target.run
            );
            assert!(mount.starts_with(prefix.as_str()), "non-run mount {mount}");
        }
    }
    assert_eq!(count, 2, "unexpected mount count");
    for bad in [
        "latest".to_string(),
        "docker.io/tailscale/tailscale:latest".to_string(),
        "sha256:bad".to_string(),
        "--privileged".to_string(),
    ] {
        assert!(
            companion_create_args(&run, &bad).is_err(),
            "mutable/caller image accepted: {bad}"
        );
    }
    let mut bad_target = run.clone();
    bad_target.target.container = "../other".to_string();
    assert!(
        companion_create_args(&bad_target, &image).is_err(),
        "invalid namespace target accepted"
    );
    let mut no_ids = run.clone();
    no_ids.uid = 0;
    assert!(companion_create_args(&no_ids, &image).is_err());
}

#[test]
fn stat_fields_reject_bad_shapes() {
    assert!(parse_stat_fields(123, &vec![b'x'; 8193]).is_err());
    assert!(parse_stat_fields(123, b"123 (a)").is_err());
    assert!(parse_stat_fields(123, b"124 (a) S 1 2").is_err());
    assert!(parse_stat_fields(123, &stat_bytes("dead")).is_err());
    let fields = parse_stat_fields(123, &stat_bytes("valid")).unwrap();
    assert!(fields.len() >= 20);
    assert_eq!(fields[19], "456");
}

#[test]
fn namespace_link_shapes() {
    assert_eq!(
        parse_namespace_link("user", "user:[22]").unwrap(),
        "user:[22]"
    );
    for bad in [
        "user:22",
        "user:[0]",
        "user:[01]",
        "net:[22]",
        "user:[22",
        "user:[]",
    ] {
        assert!(parse_namespace_link("user", bad).is_err(), "{bad}");
    }
}

#[test]
fn id_map_edges() {
    assert!(id_map(&["0:524288:262144".to_string()]));
    assert!(id_map(&["0:4294705151:262144".to_string()]));
    for bad in [
        vec![],
        vec!["0:524288:262144".to_string(), "0:1:2".to_string()],
        vec!["0:0:262144".to_string()],
        vec!["0:4294705152:262144".to_string()],
        vec!["0:4294967295:262144".to_string()],
        vec!["1:524288:262144".to_string()],
        vec!["0:524288:1".to_string()],
        vec!["0:+524288:262144".to_string()],
        vec!["0:0524288:262144".to_string()],
        vec!["nope".to_string()],
    ] {
        assert!(!id_map(&bad), "{bad:?}");
    }
}

#[test]
fn marshal_and_decode_round_trip() {
    let run = ProjectRun {
        target: RunTarget {
            project: "p0123456789abcdef01234567".to_string(),
            container: "b".repeat(64),
            run: "c".repeat(64),
        },
        pid: 123,
        started: "2026-01-02T03:04:05.678901234Z".to_string(),
        userns: "user:[22]".to_string(),
        netns: "net:[22]".to_string(),
        uid: 524288,
        gid: 524289,
        resolver: "/run/x".to_string(),
    };
    let text = marshal_project_run(&run);
    assert!(text.starts_with("{\"Target\":{\"Project\":\"p0123456789abcdef01234567\""));
    assert!(text.contains("\"PID\":123"));
    assert!(text.contains("\"UID\":524288"));
    assert!(text.ends_with("\"Resolver\":\"/run/x\"}"));
    let back = decode_project_run(text.as_bytes()).unwrap();
    assert_eq!(back, run);
    assert!(decode_project_run(b"{\"Target\":{},\"PID\":1,\"bogus\":true}").is_err());
}

#[test]
fn decode_run_inspect_binds_container() {
    let good = br#"{"id":"abc","running":true,"pid":123,"started":"s","resolver":"r"}"#;
    let raw = decode_project_run_inspect(good, "abc").unwrap();
    assert_eq!(raw.pid, 123);
    assert!(raw.running);
    // Missing optional snapshot fields still bind (admission checks them).
    let sparse = br#"{"id":"abc","running":true,"pid":123}"#;
    assert!(decode_project_run_inspect(sparse, "abc").is_ok());
    for bad in [
        br#"{"id":"other","running":true,"pid":123,"started":"s","resolver":"r"}"#.as_slice(),
        br#"{"id":"abc","running":false,"pid":123,"started":"s","resolver":"r"}"#.as_slice(),
        br#"{"id":"abc","running":true,"pid":1,"started":"s","resolver":"r"}"#.as_slice(),
        br#"{"id":"abc","running":true,"pid":1}"#.as_slice(),
        br#"[]"#.as_slice(),
        br#"{"id":"abc","running":true,"pid":123,"started":"s","resolver":"r","extra":1}"#
            .as_slice(),
    ] {
        assert!(decode_project_run_inspect(bad, "abc").is_err());
    }
}

#[test]
fn admit_snapshot_rejects_before_touching_proc() {
    let cid = "c".repeat(64);
    let resolver =
        format!("/var/lib/containers/storage/overlay-containers/{cid}/userdata/resolv.conf");
    let raw = ProjectRunInspect {
        id: cid.clone(),
        running: true,
        pid: 123,
        started: "not-a-time".to_string(),
        resolver: resolver.clone(),
    };
    assert_eq!(
        admit_project_run_snapshot(&cid, &raw).unwrap_err(),
        ERR_UNAVAILABLE
    );
    let raw = ProjectRunInspect {
        started: "2026-01-02T03:04:05Z".to_string(),
        resolver: "/etc/resolv.conf".to_string(),
        ..raw
    };
    assert_eq!(
        admit_project_run_snapshot(&cid, &raw).unwrap_err(),
        ERR_UNSUPPORTED
    );
}
