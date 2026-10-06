use super::tcodex::{self, shell_quote, sleep_until, FactoryCodexPaths};
use crate::project::Executor;
use crate::terminal::{self, Binding, Service, KIND_FACTORY};
use std::time::{Duration, Instant};

/// `factoryRetire`: identity-verified supervisor process-group kill plus a
/// lingering-member scan. Golden-pinned against the Go output.
pub fn factory_retire(p: &FactoryCodexPaths) -> String {
    let script = [
        format!("RUNDIR={}", shell_quote(&p.run_dir)),
        ": >\"$RUNDIR/stop\"".to_string(),
        "[ -f \"$RUNDIR/supervisor.pid\" ] || exit 0".to_string(),
        "read PID START <\"$RUNDIR/supervisor.pid\"".to_string(),
        "case \"$PID\" in ''|*[!0-9]*) exit 0;; esac".to_string(),
        "case \"$START\" in ''|*[!0-9]*) exit 0;; esac".to_string(),
        "if [ -d \"/proc/$PID\" ]; then".to_string(),
        "  if STAT=$(cat \"/proc/$PID/stat\" 2>/dev/null); then".to_string(),
        "    REST=${STAT##*)}; set -- $REST".to_string(),
        "    if [ \"${20}\" = \"$START\" ]; then".to_string(),
        "      CMDLINE=$(tr '\\000' ' ' <\"/proc/$PID/cmdline\" 2>/dev/null) || CMDLINE=\"\"".to_string(),
        "      case \"$CMDLINE\" in *\"$RUNDIR\"*)".to_string(),
        "        if [ \"$3\" = \"$PID\" ]; then kill -KILL -- \"-$PID\" 2>/dev/null || true; else kill -KILL -- \"$PID\" 2>/dev/null || true; fi".to_string(),
        "      ;; esac".to_string(),
        "    fi".to_string(),
        "  fi".to_string(),
        "fi".to_string(),
        "sleep 1".to_string(),
        "for S in /proc/[0-9]*/stat; do".to_string(),
        "  STAT=$(cat \"$S\" 2>/dev/null) || continue".to_string(),
        "  REST=${STAT##*)}; set -- $REST".to_string(),
        "  if [ \"$3\" = \"$PID\" ]; then echo \"lingering: $S\"; exit 1; fi".to_string(),
        "done".to_string(),
        "exit 0".to_string(),
    ];
    script.join("\n") + "\n"
}

impl<E: Executor> Service<E> {
    pub(crate) fn factory_await_inactive(
        &self,
        unit: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let end = Instant::now() + Duration::from_secs(15);
        loop {
            if let Ok(show) = self.factory_unit_state(unit, deadline) {
                if !show.active {
                    return Ok(());
                }
            }
            let now = Instant::now();
            if now >= end || now >= deadline {
                return Err(terminal::err_uncertain());
            }
            if sleep_until(now + Duration::from_millis(250), deadline).is_err() {
                return Err(terminal::err_uncertain());
            }
        }
    }

    pub(crate) fn factory_retire(
        &self,
        container: &str,
        run_dir: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        // The retire script only needs the run directory; build the shim
        // paths so the golden-pinned builder stays untouched.
        let p = FactoryCodexPaths {
            run_dir: run_dir.to_string(),
            ..Default::default()
        };
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            factory_retire(&p),
        ];
        self.run_podman(&[], &argv, deadline)
            .map(|_| ())
            .map_err(|_| terminal::err_uncertain())
    }

    pub(crate) fn factory_read_pid(
        &self,
        container: &str,
        pid_file: &str,
        deadline: Instant,
    ) -> String {
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/cat".to_string(),
            pid_file.to_string(),
        ];
        match self.run_podman(&[], &argv, deadline) {
            Ok(out) if out.len() <= 256 && out.contains(&b' ') => {
                String::from_utf8_lossy(&out).trim().to_string()
            }
            _ => String::new(),
        }
    }

    pub(crate) fn factory_container_exists(
        &self,
        container: &str,
        deadline: Instant,
    ) -> Result<bool, String> {
        match self.run_podman(&[], &terminal::container_exists_argv(container), deadline) {
            Ok(_) => Ok(true),
            Err(err) if terminal::exit_code_of(&err) == Some(1) => Ok(false),
            Err(_) => Err(terminal::err_uncertain()),
        }
    }

    /// Shared wait core: unit quiescence, then the exit code and the
    /// head of the answer file. `project` is the recorded container,
    /// `run_dir`/`output` the derived run paths.
    pub(crate) fn factory_wait_result(
        &self,
        execution_id: &str,
        project: &str,
        run_dir: &str,
        output: &str,
        deadline: Instant,
    ) -> Result<(i32, String), String> {
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        loop {
            let show = self.factory_unit_state(&unit, deadline)?;
            if !show.active {
                break;
            }
            if tcodex::sleep_until(Instant::now() + Duration::from_millis(500), deadline).is_err() {
                return Err("context deadline exceeded".to_string());
            }
        }
        let mut exit = -1;
        let cat = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/cat".to_string(),
            format!("{run_dir}/exit"),
        ];
        if let Ok(out) = self.run_podman(&[], &cat, deadline) {
            if let Some(code) = terminal::parse_go_int(String::from_utf8_lossy(&out).trim()) {
                if (0..=255).contains(&code) {
                    exit = code as i32;
                }
            }
        }
        let mut answer = String::new();
        let head = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/head".to_string(),
            "-c".to_string(),
            "65537".to_string(),
            output.to_string(),
        ];
        if let Ok(out) = self.run_podman(&[], &head, deadline) {
            answer = String::from_utf8_lossy(&out).into_owned();
        }
        Ok((exit, answer))
    }

    /// Shared validate core: attest the live supervised boundary
    /// before the broker releases credential bytes.
    pub(crate) fn factory_attest_live(
        &self,
        project_id: &str,
        binding: &Binding,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let container = self.factory_project_container(project_id, true, deadline)?;
        if container != binding.project {
            return Err(terminal::err_denied());
        }
        let (uid, gid) = self.factory_role_ids(&container, &binding.login, deadline)?;
        if uid != binding.uid || gid != binding.gid {
            return Err(terminal::err_denied());
        }
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        let show = self
            .factory_unit_state(&unit, deadline)
            .map_err(|_| terminal::err_denied())?;
        if !show.active || show.invocation != binding.invocation_id {
            return Err(terminal::err_denied());
        }
        Ok(())
    }

    /// Shared stop core: retire the recorded unit and container
    /// process group. Idempotent.
    pub(crate) fn factory_stop_confirmed(
        &self,
        container: &str,
        execution_id: &str,
        pid_file: &str,
        run_dir: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        let before = self.factory_read_pid(container, pid_file, deadline);
        let _ = self.factory_systemctl(&["stop", &unit], deadline);
        self.factory_await_inactive(&unit, deadline)?;
        match self.factory_container_exists(container, deadline) {
            Ok(true) => {}
            Ok(false) => return Ok(()),
            Err(err) => return Err(err),
        }
        self.factory_retire(container, run_dir, deadline)?;
        let after = self.factory_read_pid(container, pid_file, deadline);
        if !after.is_empty() && after != before {
            self.factory_retire(container, run_dir, deadline)?;
            if self.factory_read_pid(container, pid_file, deadline) != after {
                return Err(terminal::err_uncertain());
            }
        }
        Ok(())
    }

    /// Shared unbound-stop core: retire a run whose binding was never
    /// recorded.
    pub(crate) fn factory_stop_unbound_confirmed(
        &self,
        project: &str,
        execution_id: &str,
        pid_file: &str,
        run_dir: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        let _ = self.factory_systemctl(&["stop", &unit], deadline);
        self.factory_await_inactive(&unit, deadline)?;
        let container = match self.factory_project_container(project, false, deadline) {
            Ok(container) => container,
            Err(_) => match self.factory_container_exists(&format!("soda-{project}"), deadline) {
                Ok(false) => return Ok(()),
                Ok(true) => return Err(terminal::err_uncertain()),
                Err(err) => return Err(err),
            },
        };
        let before = self.factory_read_pid(&container, pid_file, deadline);
        self.factory_retire(&container, run_dir, deadline)?;
        let after = self.factory_read_pid(&container, pid_file, deadline);
        if !after.is_empty() && after != before {
            self.factory_retire(&container, run_dir, deadline)?;
            if self.factory_read_pid(&container, pid_file, deadline) != after {
                return Err(terminal::err_uncertain());
            }
        }
        Ok(())
    }

    /// Shared capture core: one bounded read of the staged credential
    /// file. Codex rotates on these bytes; muse only echoes them for
    /// the in-process return path (the broker ignores muse bytes).
    pub(crate) fn factory_capture_valid(
        &self,
        project: &str,
        path: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/head".to_string(),
            "-c".to_string(),
            "262145".to_string(),
            path.to_string(),
        ];
        match self.run_podman(&[], &argv, deadline) {
            Ok(out) if terminal::credential_valid(&out) => Ok(out),
            _ => Err(terminal::err_uncertain()),
        }
    }

    /// Shared live core: recorded unit currently active with the
    /// recorded invocation. Observation only.
    pub(crate) fn factory_live_scoped(
        &self,
        binding: &Binding,
        scope: &str,
        deadline: Instant,
    ) -> bool {
        if binding.kind != KIND_FACTORY
            || binding.scope != scope
            || !terminal::valid_terminal_id(&binding.id)
            || !terminal::valid_terminal_id(&binding.invocation_id)
        {
            return false;
        }
        let Some(unit) = tcodex::factory_unit_name(&binding.id) else {
            return false;
        };
        match self.factory_unit_state(&unit, deadline) {
            Ok(show) => show.active && show.invocation == binding.invocation_id,
            Err(_) => false,
        }
    }
}
