use std::time::{Duration, Instant};

use crate::project::Executor;
use crate::tailnet_domain::{
    valid_container_id, ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE, ERR_UNCONFIRMED,
};
use crate::tailnet_files::read_companion_id;
use crate::tailnet_runtime::ProjectRun;

use super::{
    arg_refs, companion_resolver, companion_still_running, decode_companion_record,
    match_companion_namespaces, stat_metadata, validate_companion_record, Companion,
    CompanionRecord, TailnetControl, COMPANION_INSPECT, RUNTIME_ROOT,
};

impl<E: Executor, T: TailnetControl> Companion<E, T> {
    pub(crate) fn runtime_command(
        &self,
        cmd: &str,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        // The ordinary host executor predates secret-bearing native protocols.
        // Never collect its stderr for the companion, and never return native
        // error text.
        if self.exec.is_host_native() {
            return self.run_native_command(cmd, args, deadline);
        }
        match self.exec.run(&[], cmd, &arg_refs(args), deadline) {
            Ok(body) if body.len() <= 65536 => Ok(body),
            _ => Err(ERR_UNCONFIRMED.to_string()),
        }
    }

    pub(crate) fn runtime_podman(
        &self,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let mut full = vec!["--remote=false".to_string()];
        full.extend(args.iter().cloned());
        self.runtime_command("/usr/bin/podman", &full, deadline)
    }

    /// Direct native execution with deadline kill. Stdout is capped at 65536
    /// bytes total; overflow, spawn failure, deadline kill and non-zero exit
    /// all collapse to unconfirmed without leaking native diagnostics.
    pub(crate) fn run_native_command(
        &self,
        cmd: &str,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        use std::io::Read;
        use std::process::Stdio;
        let mut child = std::process::Command::new(cmd)
            .args(args)
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .map_err(|_| ERR_UNCONFIRMED.to_string())?;
        let mut stdout = child.stdout.take();
        std::thread::scope(|scope| {
            let reader = scope.spawn(|| {
                let mut capped = Vec::new();
                let mut overflow = false;
                let mut chunk = [0u8; 8192];
                if let Some(out) = stdout.as_mut() {
                    loop {
                        match out.read(&mut chunk) {
                            Ok(0) => break,
                            Ok(n) => {
                                if capped.len() + n > 65536 {
                                    overflow = true;
                                } else {
                                    capped.extend_from_slice(&chunk[..n]);
                                }
                            }
                            Err(_) => {
                                overflow = true;
                                break;
                            }
                        }
                    }
                }
                (capped, overflow)
            });
            loop {
                match child.try_wait() {
                    Ok(Some(status)) => {
                        let (capped, overflow) =
                            reader.join().map_err(|_| ERR_UNCONFIRMED.to_string())?;
                        if !status.success() || overflow {
                            return Err(ERR_UNCONFIRMED.to_string());
                        }
                        return Ok(capped);
                    }
                    Ok(None) => {
                        if Instant::now() >= deadline {
                            let _ = child.kill();
                            let _ = reader.join();
                            return Err(ERR_UNCONFIRMED.to_string());
                        }
                        std::thread::sleep(Duration::from_millis(5));
                    }
                    Err(_) => {
                        let _ = reader.join();
                        return Err(ERR_UNCONFIRMED.to_string());
                    }
                }
            }
        })
    }

    pub(crate) fn inspect_companion(
        &self,
        run: &ProjectRun,
        deadline: Instant,
    ) -> Result<CompanionRecord, String> {
        let id = read_companion_id(RUNTIME_ROOT, run, 0, 0)?;
        let body = self
            .runtime_podman(
                &[
                    "inspect".to_string(),
                    "--format".to_string(),
                    COMPANION_INSPECT.to_string(),
                    id.clone(),
                ],
                deadline,
            )
            .map_err(|_| ERR_CONFLICT.to_string())?;
        let out = decode_companion_record(&body).map_err(|_| ERR_CONFLICT.to_string())?;
        validate_companion_record(&out, run, &self.image, &id)?;
        match_companion_namespaces(&out, run)?;
        Ok(out)
    }

    pub(crate) fn companion_cli(
        &self,
        run: &ProjectRun,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        if args.is_empty() {
            return Err(ERR_INVALID.to_string());
        }
        let rec = match self.inspect_companion(run, deadline) {
            Ok(rec) if rec.running => rec,
            _ => return Err(ERR_UNAVAILABLE.to_string()),
        };
        if args[0] != "logout" {
            companion_resolver(run, rec.pid, &stat_metadata)?;
        }
        let mut full = vec![
            "exec".to_string(),
            "--user=0:0".to_string(),
            rec.id.clone(),
            "/usr/local/bin/tailscale".to_string(),
            "--socket=/run/tailscale/tailscaled.sock".to_string(),
        ];
        full.extend(args.iter().cloned());
        let output = self.runtime_podman(&full, deadline);
        let after = self.inspect_companion(run, deadline);
        if !companion_still_running(&rec, after.as_ref().ok(), output.is_ok()) {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        if args[0] != "logout" {
            let after = after.map_err(|_| ERR_UNCONFIRMED.to_string())?;
            companion_resolver(run, after.pid, &stat_metadata)
                .map_err(|_| ERR_UNCONFIRMED.to_string())?;
        }
        output.map_err(|_| ERR_UNCONFIRMED.to_string())
    }

    /// Delegates lifetime to native Podman/systemd, not a reenrollment
    /// worker. Each daemon activation has its own in-memory ephemeral identity.
    pub fn wait_tailnet(&self, cid: &str, deadline: Instant) -> Result<(), String> {
        if cid.is_empty() {
            return Ok(());
        }
        if !valid_container_id(cid) {
            return Err(ERR_INVALID.to_string());
        }
        let output = self.runtime_podman(
            &[
                "wait".to_string(),
                "--condition=stopped".to_string(),
                cid.to_string(),
            ],
            deadline,
        );
        if Instant::now() >= deadline {
            return Ok(());
        }
        let body = output.map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if String::from_utf8_lossy(&body).trim() != "0" {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        // Unexpected successful daemon exit is also a supervision failure, not
        // a reason to silently leave an enabled project without its daemon.
        Err(ERR_UNAVAILABLE.to_string())
    }
}
