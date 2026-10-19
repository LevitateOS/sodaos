use crate::error::Error;
use crate::files;
use crate::jsonio;
use crate::provisioning;
use crate::vm;

use super::RunOptions;

fn read_secret_files(
    files: &[String],
    secrets: &mut provisioning::SecretCollector,
) -> Result<(), Error> {
    for file in files {
        let bytes = files::private_file(file)?;
        if bytes.is_empty() {
            return Err(Error::msg("empty secret input"));
        }
        secrets.push(&bytes)?;
        let trimmed_len = bytes
            .iter()
            .rposition(|b| *b != b'\r' && *b != b'\n')
            .map_or(0, |i| i + 1);
        secrets.push(&bytes[..trimmed_len])?;
    }
    Ok(())
}

fn load_vm_secrets(config_path: &str, arch: &str, target: &str) -> Result<vm::VmConfig, Error> {
    let (value, _) = jsonio::read_json_file(config_path)?;
    let vm_config = vm::decode_vm_config(&value)?;
    if vm_config.architecture != arch || vm_config.name != target {
        return Err(Error::msg("VM target/platform mismatch"));
    }
    Ok(vm_config)
}

pub(super) fn collect_all_secrets(
    opts: &RunOptions,
) -> Result<(Vec<Vec<u8>>, vm::VmConfig), Error> {
    let mut secrets = provisioning::SecretCollector::default();
    read_secret_files(&opts.secret_files, &mut secrets)?;
    let mut vm_config = vm::VmConfig::default();
    if opts.action == "vm" {
        vm_config = load_vm_secrets(&opts.config, &opts.arch, &opts.target)?;
        provisioning::collect_provisioning_secrets(&vm_config.ignition, &mut secrets)?;
    }
    Ok((secrets.into_values(), vm_config))
}
