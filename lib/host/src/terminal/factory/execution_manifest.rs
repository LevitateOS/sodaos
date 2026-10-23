use std::time::Instant;

use crate::factory::ExecutionManifest;
use crate::project::Executor;
use crate::terminal::{self, Lease, Service};

use super::native::shell_quote;

impl<E: Executor> Service<E> {
    /// Stage a read-only plan under the existing privileged role-helper root.
    /// The parent and children can read their exact inputs but cannot replace
    /// the file or its ancestor with an agent-owned checkout path.
    pub fn factory_stage_execution_manifest(
        &self,
        lease: &Lease,
        bytes: &[u8],
        deadline: Instant,
    ) -> Result<(), String> {
        let manifest = ExecutionManifest::decode(bytes)?;
        let binding = lease.binding.as_ref().ok_or_else(terminal::err_denied)?;
        if manifest.run != lease.execution_id || binding.id != lease.execution_id {
            return Err(terminal::err_denied());
        }
        self.factory_attest_live(&lease.project_id, binding, &lease.execution_id, deadline)?;
        let path = format!(
            "/var/lib/soda/factory/executions/{}.json",
            lease.execution_id
        );
        let command = format!(
            "set -eu\nbase=/var/lib/soda/factory\n[ ! -L \"$base\" ] && [ \"$(/usr/bin/stat -c '%u:%g:%a' \"$base\")\" = 0:0:755 ]\n/usr/bin/install -d -m 755 -o 0 -g 0 \"$base/executions\"\n[ ! -L \"$base/executions\" ] && [ \"$(/usr/bin/stat -c '%u:%g:%a' \"$base/executions\")\" = 0:0:755 ]\n/usr/bin/install -m 440 -o 0 -g {} /dev/stdin {}\n",
            binding.gid, shell_quote(&path),
        );
        self.run_podman(
            bytes,
            &[
                "--remote=false".to_string(),
                "exec".to_string(),
                "--interactive".to_string(),
                binding.project.clone(),
                "/usr/bin/sh".to_string(),
                "-c".to_string(),
                command,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| "execution-manifest staging unconfirmed".to_string())
    }
}
