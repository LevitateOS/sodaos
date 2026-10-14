//! Bounded observation of installed support bytes; never invokes a legacy helper.
use super::{agent_program_hash, Service};
use crate::project::{admission, Executor};
use std::time::{Duration, Instant};

pub const MISMATCH: &str = "terminal_support_mismatch";
pub const UNAVAILABLE: &str = "terminal_support_unavailable";

#[derive(Debug, serde::Serialize)]
pub struct SupportObservation {
    pub state: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expected_sha256: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub observed_sha256: Option<String>,
}

impl SupportObservation {
    pub fn failure_code(&self) -> Option<&'static str> {
        match self.state {
            "ready" => None,
            "support_mismatch" => Some(MISMATCH),
            "maintenance_in_progress" => Some(admission::MAINTENANCE),
            _ => Some(UNAVAILABLE),
        }
    }
    fn unavailable() -> Self {
        Self {
            state: "inspection_unavailable",
            expected_sha256: None,
            observed_sha256: None,
        }
    }
}

// Fixed paths and script. No Project-controlled string enters shell source. Read
// regular, root-owned, non-writable helper bytes without running that helper.
const SUPPORT_INSPECT: &str = "for d in /usr /usr/libexec /usr/libexec/soda; do test -d \"$d\" && test ! -L \"$d\" && test \"$(stat -c %u -- \"$d\")\" = 0 || exit 1; m=$(stat -c %a -- \"$d\") || exit 1; test \"$((0$m & 022))\" -eq 0 || exit 1; done; p=/usr/libexec/soda/project-terminal; test -x \"$p\" && test -f \"$p\" && test ! -L \"$p\" && stat -c '%u:%a:%s' -- \"$p\" && sha256sum -- \"$p\"";

fn observed_digest(bytes: &[u8]) -> Option<String> {
    let text = std::str::from_utf8(bytes).ok()?;
    let mut lines = text.lines();
    let fields: Vec<_> = lines.next()?.split(':').collect();
    if fields.len() != 3 || fields[0] != "0" {
        return None;
    }
    let mode = u32::from_str_radix(fields[1], 8).ok()?;
    let size = fields[2].parse::<u64>().ok()?;
    if mode & 0o022 != 0 || mode & 0o111 == 0 || !(1..=32 << 20).contains(&size) {
        return None;
    }
    let line = lines.next()?;
    let (digest, path) = line.split_once("  ")?;
    if lines.next().is_some()
        || path != "/usr/libexec/soda/project-terminal"
        || digest.len() != 64
        || !digest
            .bytes()
            .all(|c| c.is_ascii_digit() || (b'a'..=b'f').contains(&c))
    {
        return None;
    }
    Some(digest.to_string())
}

impl<E: Executor> Service<E> {
    pub fn support_in_container(&self, container: &str, deadline: Instant) -> SupportObservation {
        if !crate::domain::valid_container_id(container) {
            return SupportObservation::unavailable();
        }
        let Ok(expected) = agent_program_hash(0) else {
            return SupportObservation::unavailable();
        };
        self.support_against(container, &expected, deadline)
    }

    fn support_against(
        &self,
        container: &str,
        expected: &str,
        deadline: Instant,
    ) -> SupportObservation {
        let args = [
            "--remote=false",
            "exec",
            container,
            "/usr/bin/sh",
            "-c",
            SUPPORT_INSPECT,
        ];
        let data = self.exec.run_bounded(
            &[],
            "/usr/bin/podman",
            &args,
            deadline.min(Instant::now() + Duration::from_secs(5)),
            512,
            1024,
        );
        let observed = data
            .ok()
            .filter(|data| data.len() <= 512)
            .and_then(|data| observed_digest(&data));
        let state = match observed.as_deref() {
            Some(value) if value == expected => "ready",
            Some(_) => "support_mismatch",
            None => "inspection_unavailable",
        };
        SupportObservation {
            state,
            expected_sha256: Some(expected.to_string()),
            observed_sha256: observed,
        }
    }

    pub fn terminal_support(&self, project: &str, deadline: Instant) -> SupportObservation {
        let deadline = deadline.min(Instant::now() + Duration::from_secs(5));
        let _guard = match admission::admit(Some(project)) {
            Ok(guard) => guard,
            Err(reason) if reason == admission::MAINTENANCE => {
                return SupportObservation {
                    state: "maintenance_in_progress",
                    expected_sha256: None,
                    observed_sha256: None,
                }
            }
            Err(_) => return SupportObservation::unavailable(),
        };
        // A stopped root is explicitly unknown until a native read-only mount
        // mechanism has been qualified. Observation never starts a Project.
        let Ok(container) = self.project_container(project, true, deadline) else {
            return SupportObservation::unavailable();
        };
        self.support_in_container(&container, deadline)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn observed_helper_requires_admitted_metadata() {
        let digest = "a".repeat(64);
        let valid = format!("0:755:12345\n{digest}  /usr/libexec/soda/project-terminal\n");
        assert_eq!(observed_digest(valid.as_bytes()), Some(digest));
        for invalid in [
            valid.replace("0:755", "1:755"),
            valid.replace("0:755", "0:777"),
            valid.replace("12345", "0"),
            format!("{valid}extra"),
        ] {
            assert_eq!(observed_digest(invalid.as_bytes()), None);
        }
    }
}

#[cfg(test)]
mod admission_tests {
    use super::*;
    use std::sync::Mutex;
    struct ReadOnlyFixture {
        reply: Result<Vec<u8>, String>,
        calls: Mutex<usize>,
    }
    impl Executor for ReadOnlyFixture {
        fn run(
            &self,
            stdin: &[u8],
            command: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            assert!(stdin.is_empty());
            assert_eq!(command, "/usr/bin/podman");
            assert_eq!(
                args[0..4],
                [
                    "--remote=false",
                    "exec",
                    "a".repeat(64).as_str(),
                    "/usr/bin/sh"
                ]
            );
            assert_eq!(args[4], "-c");
            assert_eq!(args[5], SUPPORT_INSPECT);
            *self.calls.lock().unwrap() += 1;
            self.reply.clone()
        }
    }
    #[test]
    fn legacy_support_observation_never_executes_helper_or_reserves_shell() {
        let expected = "b".repeat(64);
        for (reply, state, reason) in [
            (
                Ok(format!(
                    "0:755:100\n{}  /usr/libexec/soda/project-terminal\n",
                    expected
                )
                .into_bytes()),
                "ready",
                None,
            ),
            (
                Ok(format!(
                    "0:755:100\n{}  /usr/libexec/soda/project-terminal\n",
                    "c".repeat(64)
                )
                .into_bytes()),
                "support_mismatch",
                Some(MISMATCH),
            ),
            (
                Err("transport error with arbitrary private text".into()),
                "inspection_unavailable",
                Some(UNAVAILABLE),
            ),
        ] {
            let service = Service {
                exec: ReadOnlyFixture {
                    reply,
                    calls: Mutex::new(0),
                },
                codex_harness: String::new(),
                codex_harness_sha256: String::new(),
                codex_harness_version: String::new(),
                muse_harness: String::new(),
                muse_harness_sha256: String::new(),
                muse_harness_version: String::new(),
            };
            let observation = service.support_against(
                &"a".repeat(64),
                &expected,
                Instant::now() + Duration::from_secs(1),
            );
            assert_eq!(observation.state, state);
            assert_eq!(observation.failure_code(), reason);
            assert_eq!(*service.exec.calls.lock().unwrap(), 1);
        }
    }
}
