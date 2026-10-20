use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::{artifacts_bin, run, TempDir};

#[test]
fn artifacts_dispatch_matches_go() {
    let scratch = TempDir::new("art-dispatch");
    for (args, want) in [
        (vec![], "usage: soda-artifacts inspect-oci|fetch-coreos|fetch-coreos-iso|convert-butane|check-fountain-pin [flags]\n"),
        (
            vec!["bogus-action"],
            "invalid artifact command flags\n",
        ),
        (vec!["inspect-oci"], "expected x86_64\n"),
        (
            vec!["inspect-oci", "--arch", "x86_64", "--source", "/nope", "--revision", "abc"],
            "full source revision required\n",
        ),
        (
            vec!["inspect-oci", "extra", "--arch", "x86_64"],
            "invalid artifact command flags\n",
        ),
        (
            vec!["inspect-oci", "--arch", "x86_64", "--bogus", "x"],
            "invalid artifact command flags\n",
        ),
        (
            vec!["convert-butane", "--arch", "x86_64", "--source", "/nope", "--out", "/tmp/x.json"],
            "real private output parent required\n",
        ),
        (
            vec!["fetch-coreos", "--arch", "x86_64", "--keyring", "/k", "--signer", "deadbeef", "--out", "/tmp/x"],
            "full trusted signer fingerprint required\n",
        ),
        (
            vec!["fetch-coreos-iso", "--arch", "aarch64"],
            "expected x86_64\n",
        ),
    ] {
        let (code, out, err) = run(&artifacts_bin(), &scratch.path, &args);
        assert_eq!(code, 1, "{args:?}");
        assert_eq!(out, "", "{args:?}");
        assert_eq!(err, want, "{args:?}");
    }
}

#[test]
fn artifacts_generated_help_exits_before_work() {
    let scratch = TempDir::new("art-help");
    for args in [vec!["-h"], vec!["--help"], vec!["inspect-oci", "--help"]] {
        let (code, out, err) = run(&artifacts_bin(), &scratch.path, &args);
        assert_eq!(code, 0, "{args:?}: {err}");
        assert!(out.contains("--keyring"), "{out}");
        assert!(err.is_empty(), "{err}");
    }
    assert!(fs::read_dir(&scratch.path).unwrap().next().is_none());
}

#[test]
fn artifacts_butane_refusals_match_go() {
    let scratch = TempDir::new("art-butane");
    // Private parent, missing source: the Go owner reports the missing file.
    let private = scratch.path.join("p");
    fs::create_dir(&private).unwrap();
    fs::set_permissions(&private, fs::Permissions::from_mode(0o700)).unwrap();
    let missing = private.join("missing.bu");
    let dest = private.join("out.json");
    let (code, _, err) = run(
        &artifacts_bin(),
        &scratch.path,
        &[
            "convert-butane",
            "--arch",
            "x86_64",
            "--source",
            missing.to_str().unwrap(),
            "--out",
            dest.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 1);
    // The tool check precedes the source open in both implementations.
    let butane_present = env::var_os("PATH")
        .map(|paths| env::split_paths(&paths).any(|dir| dir.join("butane").is_file()))
        .unwrap_or(false);
    if butane_present {
        assert!(err.contains("No such file"), "{err}");
    } else {
        assert_eq!(
            err,
            "exec: \"butane\": executable file not found in $PATH\n"
        );
    }
    assert!(!dest.exists());
}

#[test]
fn artifacts_inspect_validates_archive_shape() {
    let scratch = TempDir::new("art-oci");
    let dir = scratch.path.join("adir");
    fs::create_dir(&dir).unwrap();
    let revision = "a".repeat(40);
    let (code, _, err) = run(
        &artifacts_bin(),
        &scratch.path,
        &[
            "inspect-oci",
            "--arch",
            "x86_64",
            "--source",
            dir.to_str().unwrap(),
            "--revision",
            &revision,
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "OCI archive must be regular\n");
}

#[test]
fn artifacts_fountain_pin_refusals_match_go() {
    let scratch = TempDir::new("art-pin");
    let run_pin = |args: &[&str]| run(&artifacts_bin(), &scratch.path, args);
    // Missing pin file reports the filesystem error.
    let (code, _, err) = run_pin(&[
        "check-fountain-pin",
        "--pin-file",
        scratch.path.join("absent.json").to_str().unwrap(),
    ]);
    assert_eq!(code, 1);
    assert!(err.contains("No such file"), "{err}");
    // Malformed pin content is refused before any lookup.
    let bad = scratch.path.join("bad.json");
    fs::write(&bad, "{bad").unwrap();
    let (code, _, err) = run_pin(&["check-fountain-pin", "--pin-file", bad.to_str().unwrap()]);
    assert_eq!(code, 1);
    assert_eq!(err, "invalid Fountain pin file\n");
    // Wrong shapes are refused with plain messages.
    let short = scratch.path.join("short.json");
    fs::write(&short, r#"{"repository":"o/r","revision":"abc"}"#).unwrap();
    let (code, _, err) = run_pin(&["check-fountain-pin", "--pin-file", short.to_str().unwrap()]);
    assert_eq!(code, 1);
    assert_eq!(err, "Fountain pin file must pin a 40-character revision\n");
    let repo = scratch.path.join("repo.json");
    fs::write(
        &repo,
        r#"{"repository":"noslash","revision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}"#,
    )
    .unwrap();
    let (code, _, err) = run_pin(&["check-fountain-pin", "--pin-file", repo.to_str().unwrap()]);
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "Fountain pin file must name an owner/name repository\n"
    );
    // A well-formed pin against a dead endpoint stops at the lookup boundary.
    let ok = scratch.path.join("ok.json");
    fs::write(
        &ok,
        r#"{"repository":"o/r","revision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}"#,
    )
    .unwrap();
    let (code, _, err) = run_pin(&[
        "check-fountain-pin",
        "--pin-file",
        ok.to_str().unwrap(),
        "--github-api-url",
        "http://127.0.0.1:1",
    ]);
    assert_eq!(code, 1);
    assert!(err.starts_with("Fountain lookup failed"), "{err}");
}

/// Minimal loopback GitHub stub: the releases list template and asset bodies
/// may contain BASE, replaced with the actual loopback base URL.
fn stub_github(releases_template: &str, assets: Vec<(&str, String)>, expect: usize) -> String {
    use std::io::{Read, Write};
    use std::net::TcpListener;
    use std::time::{Duration, Instant};
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let base = format!("http://{}", listener.local_addr().unwrap());
    let releases = releases_template.replace("BASE", &base);
    let assets = assets
        .into_iter()
        .map(|(path, body)| (path.to_owned(), body))
        .collect::<Vec<_>>();
    std::thread::spawn(move || {
        listener.set_nonblocking(true).unwrap();
        let deadline = Instant::now() + Duration::from_secs(30);
        let mut served = 0usize;
        while served < expect && Instant::now() < deadline {
            let Ok((mut stream, _)) = listener.accept() else {
                std::thread::sleep(Duration::from_millis(10));
                continue;
            };
            let mut request = vec![0u8; 8192];
            let mut used = 0usize;
            while used < request.len() {
                match stream.read(&mut request[used..]) {
                    Ok(0) | Err(_) => break,
                    Ok(n) => {
                        used += n;
                        if request[..used].windows(4).any(|w| w == b"\r\n\r\n") {
                            break;
                        }
                    }
                }
            }
            let head = String::from_utf8_lossy(&request[..used]);
            let path = head
                .lines()
                .next()
                .unwrap_or("")
                .split_whitespace()
                .nth(1)
                .unwrap_or("");
            let body = if path.starts_with("/repos/") {
                releases.clone()
            } else {
                assets
                    .iter()
                    .find(|(prefix, _)| path.starts_with(prefix))
                    .map(|(_, body)| body.clone())
                    .unwrap_or_else(|| "{\"message\":\"not found\"}".to_owned())
            };
            let _ = write!(
                stream,
                "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
                body.len()
            );
            served += 1;
        }
    });
    base
}

#[test]
fn artifacts_fountain_pin_reports_matching_release() {
    let scratch = TempDir::new("art-pin-ok");
    let pin = "a".repeat(40);
    let image = format!("ghcr.io/o/fountain@sha256:{}", "b".repeat(64));
    let meta = serde_json::json!({
        "version": "99.0.0-fountain.9",
        "source_commit": pin,
        "image": image,
    })
    .to_string();
    let base = stub_github(
        r#"[{"tag_name":"v9","assets":[]},{"tag_name":"v10","assets":[{"name":"release.json","browser_download_url":"BASE/r2.json"}]}]"#,
        vec![("/r2.json", meta)],
        2,
    );
    let pin_path = scratch.path.join("pin.json");
    fs::write(
        &pin_path,
        serde_json::json!({"repository": "o/r", "revision": pin}).to_string(),
    )
    .unwrap();
    let (code, out, err) = run(
        &artifacts_bin(),
        &scratch.path,
        &[
            "check-fountain-pin",
            "--pin-file",
            pin_path.to_str().unwrap(),
            "--github-api-url",
            &base,
        ],
    );
    assert_eq!(code, 0, "{err}");
    assert!(err.is_empty(), "{err}");
    assert!(
        out.contains(&format!("Fountain 99.0.0-fountain.9 serves pin {pin}")),
        "{out}"
    );
    assert!(
        out.contains(&format!("release.json: {base}/r2.json")),
        "{out}"
    );
    assert!(out.contains(&format!("image: {image}")), "{out}");
}

#[test]
fn artifacts_fountain_pin_reports_gap() {
    let scratch = TempDir::new("art-pin-gap");
    let pin = "c".repeat(40);
    let meta = serde_json::json!({
        "version": "9.0.0-fountain.1",
        "source_commit": "d".repeat(40),
        "image": format!("ghcr.io/o/fountain@sha256:{}", "d".repeat(64)),
    })
    .to_string();
    let base = stub_github(
        r#"[{"tag_name":"v9","assets":[{"name":"release.json","browser_download_url":"BASE/r.json"}]}]"#,
        vec![("/r.json", meta)],
        2,
    );
    let pin_path = scratch.path.join("pin.json");
    fs::write(
        &pin_path,
        serde_json::json!({"repository": "o/r", "revision": pin}).to_string(),
    )
    .unwrap();
    let (code, out, err) = run(
        &artifacts_bin(),
        &scratch.path,
        &[
            "check-fountain-pin",
            "--pin-file",
            pin_path.to_str().unwrap(),
            "--github-api-url",
            &base,
        ],
    );
    assert_eq!(code, 1);
    assert!(out.is_empty(), "{out}");
    assert_eq!(
        err,
        format!(
            "no published Fountain release serves pin {pin} (checked 1 releases (1 with readable release.json); newest: v9)\n"
        )
    );
}
