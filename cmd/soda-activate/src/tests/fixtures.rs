use std::collections::HashMap;
use std::fs;
use std::io;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};

use crate::cli::CliArgs;
use crate::system::{Paths, Sys};

pub(crate) static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

pub(crate) struct FakeSys {
    pub(crate) euid: u32,
    pub(crate) user: Option<(u32, u32)>,
    pub(crate) calls: Vec<Vec<String>>,
    pub(crate) chowns: Vec<(PathBuf, u32, u32)>,
    pub(crate) probe_code: i32,
    pub(crate) now_values: Vec<f64>,
    pub(crate) sleeps: u32,
}

impl FakeSys {
    pub(crate) fn new() -> FakeSys {
        FakeSys {
            euid: 0,
            user: Some((2000, 2000)),
            calls: Vec::new(),
            chowns: Vec::new(),
            probe_code: 0,
            now_values: vec![0.0],
            sleeps: 0,
        }
    }
}

impl Sys for FakeSys {
    fn euid(&self) -> u32 {
        self.euid
    }
    fn lookup_user(&self, name: &str) -> Result<(u32, u32), String> {
        assert_eq!(name, "soda");
        self.user
            .ok_or_else(|| "getpwnam(): name not found: 'soda'".to_string())
    }
    fn chown(&mut self, path: &Path, uid: u32, gid: u32) -> io::Result<()> {
        self.chowns.push((path.to_path_buf(), uid, gid));
        Ok(())
    }
    fn run(&mut self, argv: &[&str]) -> io::Result<i32> {
        self.calls
            .push(argv.iter().map(|s| s.to_string()).collect());
        if argv.get(0..2) == Some(&["systemctl", "is-active"]) {
            return Ok(self.probe_code);
        }
        Ok(0)
    }
    fn now(&mut self) -> f64 {
        if self.now_values.len() > 1 {
            self.now_values.remove(0)
        } else {
            self.now_values[0]
        }
    }
    fn sleep(&mut self, secs: u64) {
        assert_eq!(secs, 5);
        self.sleeps += 1;
    }
}

pub(crate) struct Fixture {
    pub(crate) temp: PathBuf,
    pub(crate) paths: Paths,
}

pub(crate) fn fixture(dashboard: &str, forgejo_env: &str) -> Fixture {
    let seq = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
    let temp =
        std::env::temp_dir().join(format!("soda-activate-test-{}-{seq}", std::process::id()));
    let root = temp.join("etc/soda");
    fs::create_dir_all(&root).expect("root");
    fs::write(root.join("dashboard.json"), dashboard).expect("dashboard");
    fs::write(root.join("grant-key"), "synthetic, not a credential").expect("key");
    fs::set_permissions(root.join("grant-key"), fs::Permissions::from_mode(0o600))
        .expect("key mode");
    fs::write(root.join("forgejo.env"), forgejo_env).expect("env");
    fs::create_dir_all(temp.join("var/lib/soda/forgejo/gitea")).expect("ext parent");
    Fixture {
        paths: Paths {
            root,
            var_lib: temp.join("var/lib/soda"),
            containers_systemd: temp.join("etc/containers/systemd"),
        },
        temp,
    }
}

pub(crate) fn dashboard(origin: &str, identity_socket: &str, operator_id: &str) -> String {
    format!(
        "{{\"forgejo_url\":{origin:?},\"forgejo_internal_url\":\"http://127.0.0.1:3000\",\"listen\":\"127.0.0.1:8080\",\"grant_key_file\":\"/etc/soda/grant-key\",\"host_socket\":\"/run/soda/host.sock\",\"identity_socket\":{identity_socket:?},\"operator_id\":{operator_id}}}"
    )
}

pub(crate) fn cli(bind_ip: &str, local_tls: bool, temp: &Path) -> CliArgs {
    if local_tls {
        CliArgs {
            bind_ip: bind_ip.to_string(),
            certificate: None,
            private_key: None,
            local_tls: true,
        }
    } else {
        let cert = temp.join("certificate");
        let key = temp.join("private-key");
        fs::write(&cert, "synthetic fixture").expect("cert");
        fs::write(&key, "synthetic fixture").expect("key");
        CliArgs {
            bind_ip: bind_ip.to_string(),
            certificate: Some(cert.to_string_lossy().into_owned()),
            private_key: Some(key.to_string_lossy().into_owned()),
            local_tls: false,
        }
    }
}

pub(crate) fn env_map(path: &Path) -> HashMap<String, String> {
    fs::read_to_string(path)
        .expect("env")
        .lines()
        .filter_map(|line| {
            line.split_once('=')
                .map(|(k, v)| (k.to_string(), v.to_string()))
        })
        .collect()
}
