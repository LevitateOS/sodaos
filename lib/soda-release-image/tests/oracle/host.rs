use super::*;

#[test]
fn oracle_local_quadlet() {
    check_ok(
        "B-quad-0",
        r"W1VuaXRdClJlcXVpcmVzPXNvZGEtaW1hZ2UtaW1wb3J0LnNlcnZpY2UKQWZ0ZXI9c29kYS1pbWFnZS1pbXBvcnQuc2VydmljZQpEZXNjcmlwdGlvbj14CltDb250YWluZXJdClB1bGw9bmV2ZXIKSW1hZ2U9c2hhMjU2OnJlZgpFeGVjPWEK",
        quadlet::local_quadlet(
            r"[Unit]
Description=x
[Container]
Image=localhost/soda-dashboard:dev
Pull=newer
Exec=a
",
            "sha256:ref",
        )
        .unwrap()
        .as_bytes(),
    );
    check_ok(
        "B-quad-1",
        r"W1VuaXRdClJlcXVpcmVzPXNvZGEtaW1hZ2UtaW1wb3J0LnNlcnZpY2UKQWZ0ZXI9c29kYS1pbWFnZS1pbXBvcnQuc2VydmljZQpbQ29udGFpbmVyXQpQdWxsPW5ldmVyCkltYWdlPXNoYTI1NjpyZWYK",
        quadlet::local_quadlet(
            r"[Unit]
[Container]
Image=a
",
            "sha256:ref",
        )
        .unwrap()
        .as_bytes(),
    );
    assert!(
        quadlet::local_quadlet(
            r"[Unit]
[Container]
GlobalArgs=--x
Image=a
",
            "sha256:ref",
        )
        .is_err(),
        "B-quad-2"
    );
    assert!(
        quadlet::local_quadlet(
            r"[Unit]
[Unit]
[Container]
Image=a
",
            "sha256:ref",
        )
        .is_err(),
        "B-quad-3"
    );
    assert!(
        quadlet::local_quadlet(
            r"[Container]
Image=a
",
            "sha256:ref",
        )
        .is_err(),
        "B-quad-4"
    );
}

#[test]
fn oracle_live_ignition() {
    use flate2::write::GzEncoder;
    use flate2::Compression;
    use std::io::Write;
    let fragment =
        br#"{"ignition":{"version":"3.5.0"},"storage":{"files":[]},"systemd":{"units":[]}}"#;
    let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
    encoder.write_all(fragment).unwrap();
    let gzipped = encoder.finish().unwrap();
    let wrapped = format!("{{\"ignition\":{{\"config\":{{\"merge\":[{{\"source\":\"data:;base64,{}\",\"compression\":\"gzip\"}}]}}}}}}", b64(&gzipped));
    assert!(
        ignition::verify_live_ignition(wrapped.as_bytes(), fragment).is_ok(),
        "C-live-ok"
    );
    let nulls = br#"{"ignition":{"version":"3.5.0","config":null},"storage":{"files":[]},"systemd":{"units":[]},"networkd":null}"#;
    assert!(
        ignition::verify_live_ignition(wrapped.as_bytes(), nulls).is_ok(),
        "C-live-nulls"
    );
    assert!(
        ignition::verify_live_ignition(wrapped.as_bytes(), br#"{"ignition":{"version":"3.4.0"}}"#)
            .is_err(),
        "C-live-diff"
    );
    assert!(
        ignition::verify_live_ignition(b"{}", fragment).is_err(),
        "C-live-bad"
    );
}

#[test]
fn oracle_package_inputs() {
    let (pkgs, repo) = packages::package_inputs(r#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo bar-1.0\n"}]}}"#.as_bytes()).unwrap();
    check_ok(
        "D-pkg-good",
        r"Zm9vLGJhci0xLjB8aHR0cHM6Ly9wa2dzLnRhaWxzY2FsZS5jb20vc3RhYmxlL2ZlZG9yYS90YWlsc2NhbGUucmVwbw==",
        format!("{}|{}", pkgs.join(","), repo).as_bytes(),
    );
    assert!(packages::package_inputs(r#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo foo\n"}]}}"#.as_bytes()).is_err(), "D-pkg-dup");
    assert!(packages::package_inputs(r#"{"storage":{"files":[{"path":"/etc/yum.repos.d/tailscale.repo","contents":{"source":"https://pkgs.tailscale.com/stable/fedora/tailscale.repo"}}]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/bin/false\n"}]}}"#.as_bytes()).is_err(), "D-pkg-bad");
    assert!(packages::package_inputs(r#"{"storage":{"files":[]},"systemd":{"units":[{"name":"soda-extensions.service","contents":"ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive foo\n"}]}}"#.as_bytes()).is_err(), "D-pkg-nore");
    assert!(
        packages::package_inputs(r"{}".as_bytes()).is_err(),
        "D-pkg-none"
    );
}

#[test]
fn oracle_rpm_inventory() {
    assert!(
        packages::valid_rpm_inventory(&[
            r"bash 0:5.1.8-1.fc38.x86_64".to_string(),
            r"rpm-ostree 0:2023.1-1.fc38.x86_64".to_string()
        ])
        .is_ok(),
        "F-rpm-0"
    );
    assert!(
        packages::valid_rpm_inventory(&[r"b 0:1-1.x".to_string(), r"a 0:1-1.x".to_string()])
            .is_err(),
        "F-rpm-1"
    );
    assert!(
        packages::valid_rpm_inventory(&[r"nope".to_string()]).is_err(),
        "F-rpm-2"
    );
    assert!(
        packages::valid_rpm_inventory(&[r"a 0:1-1.x".to_string(), r"a 0:1-1.x".to_string()])
            .is_err(),
        "F-rpm-3"
    );
    assert!(
        packages::valid_rpm_inventory(&[r"a 0:1~rc1-1^git.x86_64".to_string()]).is_ok(),
        "F-rpm-4"
    );
}

#[test]
fn oracle_candidate_live_config() {
    let payload_b64 = r"eyJBcmNoaXRlY3R1cmUiOiJ4ODZfNjQiLCJCYXNlIjoicXVheS5pby9mZWRvcmEvZmVkb3JhLWNvcmVvc0BzaGEyNTY6YmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYiIsIkNvcmVPUyI6IjQxLjIwMjUwMTAxLjMuMCIsIkZvcm1hdCI6MywiSG9zdFBhY2thZ2VzU0hBMjU2IjoiZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZmZiIsIklEIjoiNDEuMjAyNTAxMDEuMy4wLnNvZGEtYWFhYWFhYWFhYWFhIiwiSW1hZ2VzIjp7ImRhc2hib2FyZCI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjpjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtZGFzaGJvYXJkQHNoYTI1Njo5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5In0sImV4dGVuc2lvbiI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjplZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtZXh0ZW5zaW9uQHNoYTI1Njo5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5In0sImZvcmdlam8iOnsiQXJjaGl2ZVNIQTI1NiI6ImFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWEiLCJDb25maWciOiJzaGEyNTY6Y2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjYyIsIk1hbmlmZXN0Ijoic2hhMjU2Ojk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTkiLCJSZWZlcmVuY2UiOiJnaGNyLmlvL2Uvc29kYW9zLWZvcmdlam9Ac2hhMjU2Ojk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTkifSwicHJvamVjdC1vcyI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjpjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtcHJvamVjdC1vc0BzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSJ9LCJwcm94eSI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjpjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtcHJveHlAc2hhMjU2Ojk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTkifSwidGFpbG5ldCI6eyJBcmNoaXZlU0hBMjU2IjoiYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYSIsIkNvbmZpZyI6InNoYTI1NjpjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjIiwiTWFuaWZlc3QiOiJzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSIsIlJlZmVyZW5jZSI6ImdoY3IuaW8vZS9zb2Rhb3MtdGFpbG5ldEBzaGEyNTY6OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OTk5OSJ9fSwiUHJlc2VudGF0aW9uU0hBMjU2IjoiZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZCIsIlJlcG9zaXRvcnlQcmVmaXgiOiJnaGNyLmlvL2Uvc29kYW9zIiwiUmV2aXNpb24iOiJhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhIiwiU2NoZW1hIjoyNiwiVXBncmFkZUZyb20iOltdfQ==";
    let payload = base64::engine::general_purpose::STANDARD
        .decode(payload_b64.as_bytes())
        .unwrap();
    let dest = br#"{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}"#;
    let manifest = format!("sha256:{}", "1".repeat(64));
    let console = "2".repeat(64);
    let candidate = ignition::candidate_live_config(&payload, dest, &manifest, &console).unwrap();
    assert_eq!(
        candidate,
        ignition::candidate_live_config(&payload, dest, &manifest, &console).unwrap(),
        "candidate live config serialization is deterministic"
    );

    let document: serde_json::Value = serde_json::from_slice(&candidate).unwrap();
    assert_eq!(document["ignition"]["version"], "3.5.0");
    let files = document["storage"]["files"].as_array().unwrap();
    assert_eq!(files.len(), 2);
    let media_file = files
        .iter()
        .find(|file| file["path"] == "/var/usrlocal/share/soda-installer/media.json")
        .unwrap();
    assert_eq!(media_file["mode"], 420);
    let media_source = media_file["contents"]["source"].as_str().unwrap();
    let media_bytes = base64::engine::general_purpose::STANDARD
        .decode(media_source.strip_prefix("data:;base64,").unwrap())
        .unwrap();
    let identity: serde_json::Value = serde_json::from_slice(&media_bytes).unwrap();
    let parsed_payload = model::Payload::parse(std::str::from_utf8(&payload).unwrap()).unwrap();
    assert_eq!(identity["Architecture"], parsed_payload.architecture);
    assert_eq!(identity["Release"], parsed_payload.core_os);
    assert_eq!(identity["Revision"], parsed_payload.revision);
    assert_eq!(identity["InstallerVersion"], "coreos-installer 0.26.0");
    assert_eq!(identity["HostManifest"], manifest);
    assert_eq!(identity["PayloadSHA256"], sys::hex_sha256(&payload));
    assert_eq!(identity["ConsoleSHA256"], console);

    let destination_file = files
        .iter()
        .find(|file| file["path"] == "/var/usrlocal/share/soda-installer/destination.ign")
        .unwrap();
    assert_eq!(destination_file["mode"], 420);
    assert_eq!(
        destination_file["contents"]["source"].as_str().unwrap(),
        format!("data:;base64,{}", b64(dest))
    );

    let units = document["systemd"]["units"].as_array().unwrap();
    let masked: Vec<_> = units
        .iter()
        .filter(|unit| unit["mask"] == true)
        .map(|unit| unit["name"].as_str().unwrap())
        .collect();
    assert_eq!(
        masked,
        [
            "getty@tty1.service",
            "forgejo.service",
            "soda-dashboard.service",
            "soda-proxy.service",
            "soda-host.service",
            "soda-host.socket",
            "soda-image-import.service",
        ]
    );
    let console_unit = units
        .iter()
        .find(|unit| unit["name"] == "soda-installer-console.service")
        .unwrap();
    assert_eq!(console_unit["enabled"], true);
    assert!(console_unit["contents"]
        .as_str()
        .unwrap()
        .contains("ExecStart=/usr/libexec/soda/soda-install disk"));
    assert!(
        ignition::candidate_live_config(
            &payload,
            br#"{"ignition":{"version":"3.4.0"}}"#,
            &manifest,
            &console,
        )
        .is_err(),
        "L-live-baddest"
    );
    assert!(
        ignition::candidate_live_config(
            &payload,
            br#"{"ignition":{"version":"3.5.0"},"passwd":{"users":[]}}"#,
            &manifest,
            &console,
        )
        .is_err(),
        "L-live-passwd"
    );
}
