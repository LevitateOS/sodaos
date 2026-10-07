use std::path::PathBuf;
use std::sync::OnceLock;
use std::time::{Duration, Instant};

pub(crate) const UNIT: &str = "forgejo.service";
pub(crate) const CONTAINER: &str = "soda-forgejo";
// Fountain contract: models/nativeoperation OfflineMarkerPath joins this
// name onto the deployment's AppDataPath. This tool never invents a
// second marker name.
pub(crate) const MARKER_NAME: &str = "nativeop-offline";
pub(crate) const STOP_TIMEOUT: Duration = Duration::from_secs(60);

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
    fn elapsed(&mut self) -> Duration;
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

    fn elapsed(&mut self) -> Duration {
        static ORIGIN: OnceLock<Instant> = OnceLock::new();
        ORIGIN.get_or_init(Instant::now).elapsed()
    }

    fn sleep(&mut self, secs: u64) {
        std::thread::sleep(std::time::Duration::from_secs(secs));
    }
}
