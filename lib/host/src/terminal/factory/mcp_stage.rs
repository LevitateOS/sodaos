use super::muse::paths::factory_muse_binding;
use super::native::shell_quote;
use super::run::FACTORY_SCOPE_MUSE;
use super::tcodex::factory_codex_binding;
use crate::project::Executor;
use crate::terminal::{self, Lease, Service};
use std::time::Instant;

const RELAY: &str = "/usr/libexec/soda/project-terminal";
const SOCKET: &str = "/run/soda/factory-ipc/mcp.sock";

fn codex_config(home: &str) -> Vec<u8> {
    format!("[mcp_servers.factory]\ncommand = \"{RELAY}\"\nargs = [\"factory-mcp\"]\nrequired = true\nstartup_timeout_sec = 20\ntool_timeout_sec = 120\n[mcp_servers.factory.env]\nHOME = \"{home}\"\n").into_bytes()
}

fn muse_config(home: &str) -> Vec<u8> {
    serde_json::to_vec(&serde_json::json!({
        "schema_version": 1,
        "mcpServers": {"factory": {
            "type":"stdio", "command":RELAY, "args":["factory-mcp"],
            "env":{"HOME":home}, "mode":"required", "tool_timeout_sec":120
        }}
    }))
    .expect("fixed MCP configuration serializes")
}

fn stage_command(path: &str, gid: i64) -> String {
    format!("test -S {SOCKET} && test -x {RELAY} && /usr/bin/install -m 440 -o 0 -g {gid} /dev/stdin {}\n", shell_quote(path))
}

impl<E: Executor> Service<E> {
    /// Prepare the worker's fixed MCP configuration and private credential
    /// before the existing start marker can release the model process.
    pub fn factory_stage_mcp(
        &self,
        lease: &Lease,
        credential: &[u8],
        deadline: Instant,
    ) -> Result<(), String> {
        if credential.is_empty() || credential.len() > 4096 {
            return Err("invalid Factory MCP staging".to_string());
        }
        let binding = lease.binding.as_ref().ok_or_else(terminal::err_denied)?;
        let (run_dir, config_path, config) = if binding.scope == FACTORY_SCOPE_MUSE {
            let paths = factory_muse_binding(lease)?;
            (
                paths.run_dir,
                format!("{}/settings.json", paths.muse_config),
                muse_config(&paths.home),
            )
        } else {
            let paths = factory_codex_binding(lease)?;
            (
                paths.run_dir,
                format!("{}/config.toml", paths.codex),
                codex_config(&paths.home),
            )
        };
        self.factory_attest_live(&lease.project_id, binding, &lease.execution_id, deadline)?;
        let container = binding.project.clone();
        for (path, data) in [
            (format!("{run_dir}/factory-mcp.json"), credential),
            (config_path, config.as_slice()),
        ] {
            let argv = vec![
                "--remote=false".to_string(),
                "exec".to_string(),
                "--interactive".to_string(),
                container.clone(),
                "/usr/bin/sh".to_string(),
                "-c".to_string(),
                stage_command(&path, binding.gid),
            ];
            self.run_podman(data, &argv, deadline)
                .map_err(|_| "Factory MCP staging unconfirmed".to_string())?;
        }
        Ok(())
    }
}
