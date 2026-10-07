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

fn load_vm_secrets(
    config_path: &str,
    arch: &str,
    target: &str,
) -> Result<vm::VmConfig, Error> {
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn repeated_secret_files_share_the_aggregate_candidate_limit() {
        let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../.artifacts")
            .join(format!("soda-secret-bound-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&root);
        std::fs::create_dir_all(&root).unwrap();
        let path = root.join("secret");
        std::fs::write(&path, b"x").unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
        let name = path.to_string_lossy().into_owned();
        let mut secrets = provisioning::SecretCollector::default();
        let repeated = vec![name.clone(); provisioning::SECRET_COLLECTION_COUNT / 2];
        read_secret_files(&repeated, &mut secrets).unwrap();
        assert_eq!(secrets.into_values().len(), provisioning::SECRET_COLLECTION_COUNT);

        let mut secrets = provisioning::SecretCollector::default();
        let repeated = vec![name; provisioning::SECRET_COLLECTION_COUNT / 2 + 1];
        assert!(read_secret_files(&repeated, &mut secrets).is_err());
        std::fs::remove_dir_all(root).unwrap();
    }
}
