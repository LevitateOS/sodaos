use super::os::parse_os_release;
use super::*;
use crate::ssh;
use std::cell::RefCell;
use std::collections::VecDeque;

pub(super) struct Mock {
    pub(super) calls: RefCell<Vec<(String, Vec<String>)>>,
    script: RefCell<VecDeque<Result<Vec<u8>, String>>>,
}

impl Mock {
    pub(super) fn new(responses: Vec<Result<Vec<u8>, String>>) -> Self {
        Mock {
            calls: RefCell::new(Vec::new()),
            script: RefCell::new(responses.into()),
        }
    }
}

impl Executor for Mock {
    fn run(
        &self,
        _stdin: &[u8],
        cmd: &str,
        args: &[&str],
        _deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        self.calls.borrow_mut().push((
            cmd.to_string(),
            args.iter().map(|s| s.to_string()).collect(),
        ));
        self.script
            .borrow_mut()
            .pop_front()
            .unwrap_or(Err("no scripted response".to_string()))
    }
}

pub(super) fn deadline() -> Instant {
    Instant::now() + Duration::from_secs(30)
}

pub(super) fn test_config() -> Config {
    Config {
        muse_socket: String::new(),
        image: "img".to_string(),
        network: "sodanet".to_string(),
        subnet: "10.0.0.0/24".to_string(),
        bridge: "sodabr".to_string(),
    }
}

pub(super) fn sample_profile() -> domain::Profile {
    domain::Profile {
        id: domain::ROCKY_HEADLESS.to_string(),
        distribution: "rocky".to_string(),
        version: "9.7".to_string(),
        interface: "headless".to_string(),
        architecture: "amd64".to_string(),
        image: format!("sha256:{}", "a".repeat(64)),
        revision: "b".repeat(40),
    }
}

pub(super) fn container_inspect(
    id: &str,
    owner: &str,
    profile: Option<&domain::Profile>,
    running: bool,
    ip: &str,
) -> Vec<u8> {
    let mut labels = format!("\"org.soda.project\":{id:?},\"org.soda.owner\":{owner:?}");
    if let Some(p) = profile {
        labels.push_str(&format!(
            ",\"org.soda.creation-profile\":{:?},\"org.soda.profile\":{:?}",
            p.encode(),
            p.id
        ));
    }
    let image = profile.map(|p| p.image.clone()).unwrap_or_default();
    format!(
        "[{{\"Image\":{image:?},\"Config\":{{\"Labels\":{{{labels}}}}},\"State\":{{\"Running\":{running}}},\"NetworkSettings\":{{\"Networks\":{{\"sodanet\":{{\"IPAddress\":{ip:?}}}}}}}}}]"
    )
    .into_bytes()
}

fn image_inspect(profile: &domain::Profile, arch: &str, os: &str) -> Vec<u8> {
    format!(
        "{{\"Id\":{:?},\"Architecture\":{:?},\"Os\":{:?},\"Labels\":{{\"org.soda.profile\":{:?},\"org.soda.distribution\":\"rocky\",\"org.soda.distribution.version\":\"9.7\",\"org.soda.interface\":\"headless\",\"org.opencontainers.image.revision\":{:?}}}}}",
        profile.image, arch, os, profile.id, profile.revision
    )
    .into_bytes()
}

#[test]
fn resolve_profile_accepts_native_image() {
    let p = sample_profile();
    let mock = Mock::new(vec![Ok(image_inspect(&p, "amd64", "linux"))]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(rt.resolve_profile(deadline()).unwrap(), p);
    let calls = mock.calls.borrow();
    assert_eq!(calls[0].0, "/usr/bin/podman");
    assert!(calls[0].1.contains(&"img".to_string()));
}

#[test]
fn resolve_profile_refuses_foreign_or_invalid() {
    let p = sample_profile();
    for (arch, os) in [("arm64", "linux"), ("amd64", "windows")] {
        let mock = Mock::new(vec![Ok(image_inspect(&p, arch, os))]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.resolve_profile(deadline()).unwrap_err(),
            "project image is not native Linux"
        );
    }
    let mock = Mock::new(vec![Ok(b"{}".to_vec())]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert!(rt.resolve_profile(deadline()).is_err());
    let mock = Mock::new(vec![Err("boom".to_string())]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert_eq!(rt.resolve_profile(deadline()).unwrap_err(), "boom");
}

#[test]
fn create_validates_before_exec() {
    let mock = Mock::new(vec![]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let bad = domain::Create {
        profile: None,
        id: "bogus".to_string(),
        owner: 0,
    };
    assert!(rt.create(&bad, deadline()).is_err());
    assert!(mock.calls.borrow().is_empty());
}

#[test]
fn os_release_parsing_mirrors_python_cases() {
    // Selected fields; unknown keys ignored.
    let r = parse_os_release(
        b"ID=\"rocky\"\nVERSION_ID=\"9.7\"\nPRETTY_NAME=\"Rocky Linux 9.7\"\nOTHER=\"x\"\n",
    )
    .unwrap();
    assert_eq!(
        (r.id, r.version, r.name),
        (
            "rocky".to_string(),
            "9.7".to_string(),
            "Rocky Linux 9.7".to_string()
        )
    );
    // Shell syntax never executes.
    let r =
        parse_os_release(b"ID=rocky\nVERSION_ID=9.7\nPRETTY_NAME=\"$(touch /tmp/x)\"\n").unwrap();
    assert!(r.name.contains("$(touch "));
    // PRETTY_NAME defaults to ID.
    let r = parse_os_release(b"ID=rocky\nVERSION_ID=9\n").unwrap();
    assert_eq!(r.name, "rocky");
    // Comments and blanks skipped.
    let r = parse_os_release(b"# c\n\n  \nID=rocky\nVERSION_ID=9\n").unwrap();
    assert_eq!(r.id, "rocky");
    // Malformed inputs fail.
    assert!(parse_os_release(b"ID=rocky\nID=fedora\nVERSION_ID=9").is_err());
    assert!(parse_os_release(b"ID=rocky").is_err());
    assert!(parse_os_release(b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"oops").is_err());
    assert!(parse_os_release(b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"a\x00b\"").is_err());
    assert!(parse_os_release(b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"two words\" extra").is_err());
    assert!(parse_os_release(&[b'x'; 257]).is_err()); // no valid identity anyway
    let mut big = b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"".to_vec();
    big.extend(vec![b'x'; 257]);
    big.extend(b"\"");
    assert!(parse_os_release(&big).is_err());
    assert!(parse_os_release(b"ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"\xff\"").is_err());
    assert!(parse_os_release(b"ID=rocky\n").is_err()); // missing version
    assert!(parse_os_release(b"ID\nVERSION_ID=9\n").is_err()); // bare key with no '='
}

#[test]
fn observe_os_reports_unavailable_without_starting() {
    let id = format!("p{}", "e".repeat(24));
    let p = sample_profile();
    // Stopped container: single inspect, no exec into the root.
    let mock = Mock::new(vec![Ok(container_inspect(&id, "42", Some(&p), false, ""))]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let obs = rt.observe_os(&id, deadline()).unwrap();
    assert!(obs.unavailable && obs.release.is_none());
    assert_eq!(mock.calls.borrow().len(), 1);
    // Running with a valid release.
    let mock = Mock::new(vec![
        Ok(container_inspect(&id, "42", Some(&p), true, "10.0.0.5")),
        Ok(Vec::new()),
        Ok(b"ID=rocky\nVERSION_ID=\"9.7\"\nPRETTY_NAME=\"Rocky Linux\"\n".to_vec()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let obs = rt.observe_os(&id, deadline()).unwrap();
    assert!(!obs.unavailable);
    assert_eq!(obs.release.unwrap().name, "Rocky Linux");
    // Unparsable content degrades to unavailable, not an error.
    let mock = Mock::new(vec![
        Ok(container_inspect(&id, "42", Some(&p), true, "10.0.0.5")),
        Ok(Vec::new()),
        Ok(b"garbage".to_vec()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    assert!(rt.observe_os(&id, deadline()).unwrap().unavailable);
}

#[test]
fn connection_returns_ed25519_material() {
    let id = format!("p{}", "e".repeat(24));
    let p = sample_profile();
    let mut blob = vec![0, 0, 0, 11];
    blob.extend_from_slice(b"ssh-ed25519");
    blob.extend_from_slice(&[0, 0, 0, 32]);
    blob.extend_from_slice(&[0x77; 32]);
    let line = ssh::marshal_authorized_key(ssh::ALGO_ED25519, &blob);
    let mock = Mock::new(vec![
        Ok(container_inspect(&id, "42", Some(&p), true, "10.0.0.5")),
        Ok(line.clone().into_bytes()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let conn = rt.connection(&id, deadline()).unwrap();
    assert_eq!(conn.host_key, line);
    assert_eq!(conn.fingerprint, ssh::fingerprint_sha256(&blob));
    // Stopped containers stay stopped with empty key material.
    let mock = Mock::new(vec![Ok(container_inspect(&id, "42", Some(&p), false, ""))]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let conn = rt.connection(&id, deadline()).unwrap();
    assert!(conn.host_key.is_empty() && conn.fingerprint.is_empty());
}
