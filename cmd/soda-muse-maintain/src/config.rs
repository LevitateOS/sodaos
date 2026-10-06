use std::fs;
use std::io::Read;

#[derive(Default, Debug)]
pub(crate) struct Config {
    pub(crate) muse_sha256: String,
    pub(crate) muse_version: String,
    pub(crate) muse_socket: String,
    pub(crate) identity_socket: String,
    pub(crate) codex_harness: String,
    pub(crate) codex_harness_sha256: String,
    pub(crate) codex_harness_version: String,
    pub(crate) tailnet_management: bool,
    pub(crate) tailnet_image: String,
    pub(crate) image: String,
    pub(crate) network: String,
    pub(crate) subnet: String,
    pub(crate) bridge: String,
}

pub(crate) fn load_config(path: &str) -> Result<Config, String> {
    let mut file =
        fs::File::open(path).map_err(|e| super::filesystem::path_error("open", path, e))?;
    let mut data = Vec::new();
    file.read_to_end(&mut data)
        .map_err(|e| super::filesystem::path_error("read", path, e))?;
    let mut c = super::config_wire::decode_host_config(&data)?;
    super::release::apply_release_images(&mut c, super::RELEASE_PATH)?;
    super::config_validation::validate_runtime_config(&c)?;
    Ok(c)
}

#[cfg(test)]
#[path = "config_tests.rs"]
mod config_tests;
