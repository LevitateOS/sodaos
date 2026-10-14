//! Native access to a stopped retained root without starting its workloads.
use std::time::{Duration, Instant};

use super::{is_clean_absolute_path, terminal_target_ready, Service, TerminalInspection};
use crate::project::Executor;

/// Build a rooted native command using the already-admitted Project mappings.
/// The caller supplies a trusted command and holds Project admission exclusion.
/// No Project-controlled string is interpolated into shell source.
pub fn stopped_namespace_argv(
    inspection: &TerminalInspection,
    root: &str,
    command: &[&str],
) -> Result<Vec<String>, String> {
    if inspection.running
        || !crate::domain::valid_id(&inspection.project)
        || !terminal_target_ready(inspection, &inspection.project, false)
        || !is_clean_absolute_path(root)
        || root.chars().any(char::is_control)
        || !command
            .first()
            .is_some_and(|program| is_clean_absolute_path(program))
    {
        return Err("stopped Project root unavailable".to_string());
    }
    let uid_base = inspection.uid_map[0].split(':').nth(1).unwrap_or_default();
    let gid_base = inspection.gid_map[0].split(':').nth(1).unwrap_or_default();
    let mut args = vec![
        "--user".to_string(),
        format!("--map-users=0:{uid_base}:262144"),
        format!("--map-groups=0:{gid_base}:262144"),
        "--setuid=0".to_string(),
        "--setgid=0".to_string(),
        "--root".to_string(),
        root.to_string(),
        "--wd".to_string(),
        "/".to_string(),
    ];
    args.extend(command.iter().map(|arg| (*arg).to_string()));
    Ok(args)
}

fn mounted_root(data: &[u8]) -> Result<&str, String> {
    let path = std::str::from_utf8(data)
        .ok()
        .and_then(|text| text.strip_suffix('\n'))
        .filter(|text| !text.is_empty() && !text.chars().any(char::is_control))
        .filter(|text| is_clean_absolute_path(text));
    path.ok_or_else(|| "stopped Project mount unavailable".to_string())
}

impl<E: Executor> Service<E> {
    fn confirm_stopped(
        &self,
        inspection: &TerminalInspection,
        deadline: Instant,
    ) -> Result<(), String> {
        let current = self.project_inspection(&inspection.project, false, deadline)?;
        if &current != inspection || current.running {
            return Err("stopped Project identity or state changed".to_string());
        }
        Ok(())
    }

    /// Caller retains admission throughout mount, observation and cleanup.
    pub(super) fn in_stopped_root(
        &self,
        inspection: &TerminalInspection,
        command: &[&str],
        deadline: Instant,
        stdout_limit: usize,
        stderr_limit: usize,
    ) -> Result<Vec<u8>, String> {
        // Validate identity/mappings before the first mount effect.
        stopped_namespace_argv(inspection, "/", command)?;
        let mounted = self.exec.run_bounded(
            &[],
            "/usr/bin/podman",
            &["--remote=false", "mount", &inspection.id],
            deadline,
            4096,
            1024,
        );
        let result = (|| {
            let data = mounted?;
            let root = mounted_root(&data)?;
            self.confirm_stopped(inspection, deadline)?;
            let args = stopped_namespace_argv(inspection, root, command)?;
            let args: Vec<&str> = args.iter().map(String::as_str).collect();
            let output = self.exec.run_bounded(
                &[],
                "/usr/bin/unshare",
                &args,
                deadline,
                stdout_limit,
                stderr_limit,
            );
            self.confirm_stopped(inspection, deadline)?;
            output
        })();
        // Mount can succeed before its observer fails. Always reconcile this
        // exact target, including malformed replies, command errors and timeouts.
        // Cleanup has its own small bound after the observation deadline.
        let cleanup = self.exec.run_bounded(
            &[],
            "/usr/bin/podman",
            &["--remote=false", "unmount", &inspection.id],
            Instant::now() + Duration::from_secs(2),
            1024,
            1024,
        );
        cleanup.map_err(|_| "stopped Project mount cleanup unavailable".to_string())?;
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;

    fn target() -> TerminalInspection {
        TerminalInspection {
            id: "a".repeat(64),
            project: "p0123456789abcdef01234567".into(),
            owner: "1".into(),
            userns: "private".into(),
            uid_map: vec!["0:1000000:262144".into()],
            gid_map: vec!["0:2000000:262144".into()],
            ..Default::default()
        }
    }
    fn inspection(running: bool, id: &str) -> Vec<u8> {
        format!(r#"{{"id":"{id}","project":"p0123456789abcdef01234567","owner":"1","running":{running},"privileged":false,"userns":"private","mappings":{{"UidMap":["0:1000000:262144"],"GidMap":["0:2000000:262144"]}}}}"#).into_bytes()
    }
    struct Fixture {
        replies: Mutex<Vec<Result<Vec<u8>, String>>>,
        calls: Mutex<Vec<(String, Vec<String>)>>,
    }
    impl Executor for Fixture {
        fn run(
            &self,
            input: &[u8],
            program: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            assert!(input.is_empty());
            self.calls.lock().unwrap().push((
                program.into(),
                args.iter().map(|arg| (*arg).to_string()).collect(),
            ));
            self.replies.lock().unwrap().remove(0)
        }
    }
    fn service(replies: Vec<Result<Vec<u8>, String>>) -> Service<Fixture> {
        Service {
            exec: Fixture {
                replies: Mutex::new(replies),
                calls: Mutex::new(Vec::new()),
            },
            codex_harness: String::new(),
            codex_harness_sha256: String::new(),
            codex_harness_version: String::new(),
            muse_harness: String::new(),
            muse_harness_sha256: String::new(),
            muse_harness_version: String::new(),
        }
    }
    #[test]
    fn namespace_requires_exact_stopped_isolation_and_rooted_argv() {
        let target = target();
        let argv = stopped_namespace_argv(
            &target,
            "/home/fixture/merged",
            &["/usr/bin/sh", "-c", "fixed command"],
        )
        .unwrap();
        assert_eq!(
            argv,
            [
                "--user",
                "--map-users=0:1000000:262144",
                "--map-groups=0:2000000:262144",
                "--setuid=0",
                "--setgid=0",
                "--root",
                "/home/fixture/merged",
                "--wd",
                "/",
                "/usr/bin/sh",
                "-c",
                "fixed command"
            ]
        );
        let mut bad = target.clone();
        bad.running = true;
        assert!(stopped_namespace_argv(&bad, "/home/fixture/merged", &["/usr/bin/sh"]).is_err());
        bad = target.clone();
        bad.uid_map = vec!["0:0:262144".into()];
        assert!(stopped_namespace_argv(&bad, "/home/fixture/merged", &["/usr/bin/sh"]).is_err());
        assert!(
            stopped_namespace_argv(&target, "/home/fixture/../merged", &["/usr/bin/sh"]).is_err()
        );
    }
    #[test]
    fn stopped_observation_rechecks_identity_and_always_unmounts_exact_target() {
        let target = target();
        for command in [
            Ok(b"read-only bytes".to_vec()),
            Err("command timed out".to_string()),
        ] {
            let expected = command.clone();
            let service = service(vec![
                Ok(b"/home/fixture/merged\n".to_vec()),
                Ok(inspection(false, &target.id)),
                command,
                Ok(inspection(false, &target.id)),
                Ok(Vec::new()),
            ]);
            let result = service.in_stopped_root(
                &target,
                &["/usr/bin/sh", "-c", "fixed observation"],
                Instant::now() + Duration::from_secs(1),
                512,
                1024,
            );
            assert_eq!(result, expected);
            let calls = service.exec.calls.lock().unwrap();
            assert_eq!(calls.len(), 5);
            assert_eq!(calls[2].0, "/usr/bin/unshare");
            assert_eq!(
                calls.last().unwrap().1,
                ["--remote=false", "unmount", &target.id]
            );
            assert!(calls
                .iter()
                .all(|(_, args)| !args.iter().any(|a| a == "start" || a == "exec")));
        }
        for changed in [
            inspection(true, &target.id),
            inspection(false, &"b".repeat(64)),
        ] {
            let service = service(vec![
                Ok(b"/home/fixture/merged\n".to_vec()),
                Ok(changed),
                Ok(Vec::new()),
            ]);
            assert!(service
                .in_stopped_root(
                    &target,
                    &["/usr/bin/sh"],
                    Instant::now() + Duration::from_secs(1),
                    512,
                    1024
                )
                .is_err());
            assert_eq!(
                service.exec.calls.lock().unwrap().last().unwrap().1,
                ["--remote=false", "unmount", &target.id]
            );
        }
    }
    #[test]
    fn mount_reply_and_cleanup_failure_do_not_report_success() {
        let target = target();
        for mount in [
            Ok(b"/home/fixture/../merged\n".to_vec()),
            Err("mount observer timed out".into()),
        ] {
            let service = service(vec![mount, Ok(Vec::new())]);
            assert!(service
                .in_stopped_root(
                    &target,
                    &["/usr/bin/sh"],
                    Instant::now() + Duration::from_secs(1),
                    512,
                    1024
                )
                .is_err());
            assert_eq!(service.exec.calls.lock().unwrap().len(), 2);
        }
        let service = service(vec![
            Ok(b"/home/fixture/merged\n".to_vec()),
            Ok(inspection(false, &target.id)),
            Ok(b"observed".to_vec()),
            Ok(inspection(false, &target.id)),
            Err("unmount failed".into()),
        ]);
        assert!(service
            .in_stopped_root(
                &target,
                &["/usr/bin/sh"],
                Instant::now() + Duration::from_secs(1),
                512,
                1024
            )
            .is_err());
    }
}
