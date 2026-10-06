use std::fs;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};

use crate::domain::dispatch;
use crate::system::{Paths, Sys};

pub(crate) static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

pub(crate) struct FakeSys {
    pub(crate) calls: Vec<Vec<String>>,
    pub(crate) active: bool,
    pub(crate) enabled_out: String,
    pub(crate) podman_out: String,
    pub(crate) podman_code: i32,
    pub(crate) start_code: i32,
    pub(crate) now_values: Vec<f64>,
    pub(crate) sleeps: u32,
}

impl FakeSys {
    pub(crate) fn new() -> FakeSys {
        FakeSys {
            calls: Vec::new(),
            active: false,
            enabled_out: "enabled\n".to_string(),
            podman_out: String::new(),
            podman_code: 0,
            start_code: 0,
            now_values: vec![0.0],
            sleeps: 0,
        }
    }
}

impl Sys for FakeSys {
    fn run(&mut self, argv: &[&str]) -> (i32, String) {
        self.calls
            .push(argv.iter().map(|s| s.to_string()).collect());
        if argv.get(0..2) == Some(&["systemctl", "is-active"]) {
            return (if self.active { 0 } else { 1 }, String::new());
        }
        if argv.get(0..2) == Some(&["systemctl", "is-enabled"]) {
            return (0, self.enabled_out.clone());
        }
        if argv.first() == Some(&"podman") {
            return (self.podman_code, self.podman_out.clone());
        }
        if argv.get(0..2) == Some(&["systemctl", "start"]) {
            if self.start_code == 0 {
                self.active = true;
            }
            return (self.start_code, String::new());
        }
        (0, String::new())
    }

    fn now(&mut self) -> f64 {
        if self.now_values.len() > 1 {
            self.now_values.remove(0)
        } else {
            self.now_values[0]
        }
    }

    fn sleep(&mut self, secs: u64) {
        assert_eq!(secs, 2);
        self.sleeps += 1;
    }
}

pub(crate) struct Fixture {
    pub(crate) temp: PathBuf,
    pub(crate) paths: Paths,
}

pub(crate) fn fixture(app_ini: Option<&str>, env: Option<&str>) -> Fixture {
    let seq = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
    let temp = std::env::temp_dir().join(format!("soda-domain-test-{}-{seq}", std::process::id()));
    let app_ini_path = temp.join("app.ini");
    let env_path = temp.join("forgejo.env");
    let data_root = temp.join("data");
    fs::create_dir_all(&data_root).expect("data");
    if let Some(text) = app_ini {
        fs::write(&app_ini_path, text).expect("ini");
    }
    if let Some(text) = env {
        fs::write(&env_path, text).expect("env");
    }
    Fixture {
        temp,
        paths: Paths {
            env_file: env_path,
            app_ini: app_ini_path,
            data_root,
        },
    }
}

pub(crate) fn valid_ini() -> Fixture {
    fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None)
}

pub(crate) fn run_verb(
    verb: &str,
    fx: &Fixture,
    sys: &mut FakeSys,
) -> (Result<(), String>, String) {
    let mut stdout: Vec<u8> = Vec::new();
    let result = dispatch(verb, &fx.paths, sys, &mut stdout);
    (result, String::from_utf8(stdout).expect("utf8"))
}
