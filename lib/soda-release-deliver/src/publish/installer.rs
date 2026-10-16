//! Render the version-bound installer from the candidate's archived source.
use super::assets::Asset;
use crate::{buildx::Root, Error};
use std::path::Path;

pub(super) const NAME: &str = "install-docker.sh";

pub(super) fn render(
    directory: &Path,
    tag: &str,
    revision: &str,
    metadata: &Asset,
) -> Result<Vec<u8>, Error> {
    let root = Root::open(directory.to_str().ok_or_else(Error::refused)?)?;
    let template = crate::docker::installer_template(&root, "sodaos-source.tar")?;
    let mut script = String::from_utf8(template)
        .map_err(|_| Error::msg("Docker installer template must be UTF-8"))?;
    let size = metadata.bytes.to_string();
    for (marker, value) in [
        ("@RELEASE_TAG@", tag),
        ("@SOURCE_REVISION@", revision),
        ("@METADATA_SHA256@", metadata.sha256.as_str()),
        ("@METADATA_BYTES@", size.as_str()),
    ] {
        if !script.contains(marker) {
            return Err(Error::msg("Docker installer template binding missing"));
        }
        script = script.replace(marker, value);
    }
    Ok(script.into_bytes())
}
