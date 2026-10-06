use std::path::PathBuf;

pub(crate) const UNIT: &str = "forgejo.service";
pub(crate) const CONTAINER: &str = "soda-forgejo";
// Fountain contract: models/nativeoperation OfflineMarkerPath joins this
// name onto the deployment's AppDataPath. This tool never invents a
// second marker name.
pub(crate) const MARKER_NAME: &str = "nativeop-offline";
pub(crate) const STOP_TIMEOUT: f64 = 60.0;

pub(crate) struct Paths {
    pub(crate) env_file: PathBuf,
    pub(crate) app_ini: PathBuf,
    pub(crate) data_root: PathBuf,
}

impl Paths {
    pub(crate) fn production() -> Paths {
        Paths {
            env_file: PathBuf::from("/etc/soda/forgejo.env"),
            app_ini: PathBuf::from("/var/lib/soda/forgejo/gitea/conf/app.ini"),
            data_root: PathBuf::from("/var/lib/soda/forgejo"),
        }
    }
}

/// Process, identity, and time surface, injectable for tests.
pub(crate) trait Sys {
    fn run(&mut self, argv: &[&str]) -> (i32, String);
    fn now(&mut self) -> f64;
    fn sleep(&mut self, secs: u64);
}

pub(crate) struct RealSys;

impl Sys for RealSys {
    fn run(&mut self, argv: &[&str]) -> (i32, String) {
        match std::process::Command::new(argv[0])
            .args(&argv[1..])
            .output()
        {
            Ok(output) => {
                let mut combined = output.stdout;
                combined.extend_from_slice(&output.stderr);
                (
                    output.status.code().unwrap_or(1),
                    String::from_utf8_lossy(&combined).into_owned(),
                )
            }
            Err(e) => (1, e.to_string()),
        }
    }

    fn now(&mut self) -> f64 {
        use std::time::{SystemTime, UNIX_EPOCH};
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs_f64())
            .unwrap_or(0.0)
    }

    fn sleep(&mut self, secs: u64) {
        std::thread::sleep(std::time::Duration::from_secs(secs));
    }
}
