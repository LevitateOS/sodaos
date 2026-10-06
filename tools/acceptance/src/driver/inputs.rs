use crate::error::Error;
use crate::files;
use crate::jsonio;
use crate::provisioning;
use crate::vm;

use super::RunOptions;

fn read_secret_files(files: &[String]) -> Result<Vec<Vec<u8>>, Error> {
    let mut secrets = Vec::new();
    for file in files {
        let bytes = files::private_file(file)?;
        if bytes.is_empty() {
            return Err(Error::msg("empty secret input"));
        }
        let mut trimmed = bytes.clone();
        while trimmed.last().is_some_and(|b| *b == b'\r' || *b == b'\n') {
            trimmed.pop();
        }
        secrets.push(bytes);
        secrets.push(trimmed);
    }
    Ok(secrets)
}

fn load_vm_secrets(
    config_path: &str,
    arch: &str,
    target: &str,
) -> Result<(vm::VmConfig, Vec<Vec<u8>>), Error> {
    let (value, _) = jsonio::read_json_file(config_path)?;
    let vm_config = vm::decode_vm_config(&value)?;
    if vm_config.architecture != arch || vm_config.name != target {
        return Err(Error::msg("VM target/platform mismatch"));
    }
    let private = provisioning::provisioning_secrets(&vm_config.ignition)?;
    Ok((vm_config, private))
}

pub(super) fn collect_all_secrets(
    opts: &RunOptions,
) -> Result<(Vec<Vec<u8>>, vm::VmConfig), Error> {
    let mut secrets = read_secret_files(&opts.secret_files)?;
    let mut vm_config = vm::VmConfig::default();
    if opts.action == "vm" {
        let (config, vm_secrets) = load_vm_secrets(&opts.config, &opts.arch, &opts.target)?;
        vm_config = config;
        secrets.extend(vm_secrets);
    }
    Ok((secrets, vm_config))
}
