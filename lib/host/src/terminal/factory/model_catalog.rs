//! Explicit, pinned native Codex backend configuration. Stock native catalog
//! behavior remains unchanged when the operator has not selected this profile.

use std::fs::OpenOptions;
use std::io::Read;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::time::Instant;

use serde_json::{json, Value};

use super::native::shell_quote;
use super::run::FactoryRun;
use crate::project::Executor;
use crate::terminal::Service;

pub(crate) fn guest_path(run: &FactoryRun) -> String {
    format!("/var/lib/soda/factory/model-catalogs/{}.json", run.id)
}

impl<E: Executor> Service<E> {
    pub(crate) fn factory_stage_codex_catalog(
        &self,
        container: &str,
        run: &FactoryRun,
        deadline: Instant,
    ) -> Result<Option<Value>, String> {
        if self.codex_model_catalog.is_empty() && self.codex_model_catalog_sha256.is_empty() {
            return Ok(None);
        }
        run.validate()?;
        let mut file = OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
            .open(&self.codex_model_catalog)
            .map_err(|_| "configured Codex catalog unavailable")?;
        let metadata = file
            .metadata()
            .map_err(|_| "configured Codex catalog unconfirmed")?;
        if !metadata.is_file()
            || metadata.uid() != 0
            || metadata.mode() & 0o022 != 0
            || metadata.nlink() != 1
            || metadata.len() > 2 * 1024 * 1024
        {
            return Err("configured Codex catalog is not protected".into());
        }
        let mut bytes = Vec::new();
        file.by_ref()
            .take(2 * 1024 * 1024 + 1)
            .read_to_end(&mut bytes)
            .map_err(|_| "configured Codex catalog unreadable")?;
        let digest = crate::sha256::hex_lower(&crate::sha256::digest(&bytes));
        if digest != self.codex_model_catalog_sha256 {
            return Err("configured Codex catalog digest differs".into());
        }
        let catalog: Value =
            serde_json::from_slice(&bytes).map_err(|_| "configured Codex catalog invalid")?;
        let selected: Vec<_> = catalog["models"]
            .as_array()
            .ok_or("configured Codex catalog models absent")?
            .iter()
            .filter(|model| model["slug"] == run.model)
            .collect();
        if run.model.is_empty() || selected.len() != 1 || selected[0]["multi_agent_version"] != "v1"
        {
            return Err("configured Codex native backend does not match assigned model".into());
        }
        let path = guest_path(run);
        let script=format!("set -eu\nbase=/var/lib/soda/factory\n[ ! -L \"$base\" ] && [ \"$(/usr/bin/stat -c '%u:%g:%a' \"$base\")\" = 0:0:755 ]\n/usr/bin/install -d -m 755 -o 0 -g 0 \"$base/model-catalogs\"\n[ ! -L \"$base/model-catalogs\" ] && [ \"$(/usr/bin/stat -c '%u:%g:%a' \"$base/model-catalogs\")\" = 0:0:755 ]\n[ ! -e {path} ] && [ ! -L {path} ]\n/usr/bin/install -m 444 -o 0 -g 0 /dev/stdin {path}\n[ \"$(/usr/bin/stat -c '%u:%g:%a:%h' {path})\" = 0:0:444:1 ]\n/usr/bin/sha256sum {path}\n",path=shell_quote(&path));
        let output = self
            .run_podman(
                &bytes,
                &[
                    "--remote=false".into(),
                    "exec".into(),
                    "--interactive".into(),
                    container.into(),
                    "/usr/bin/sh".into(),
                    "-c".into(),
                    script,
                ],
                deadline,
            )
            .map_err(|_| "Codex native catalog staging unconfirmed")?;
        let text =
            std::str::from_utf8(&output).map_err(|_| "Codex native catalog readback invalid")?;
        let fields: Vec<_> = text.split_whitespace().collect();
        if fields.len() != 2 || fields[0] != digest || fields[1] != path {
            return Err("Codex native catalog readback differs".into());
        }
        Ok(Some(
            json!({"model":run.model,"backend":"v1","catalog_sha256":digest,"harness_sha256":self.codex_harness_sha256,"guest_path":path}),
        ))
    }
}
