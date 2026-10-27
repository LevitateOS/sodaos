//! Retain the exact optional Canlang build output of the approved check before
//! its disposable checkout is retired. No caller supplies a path or digest.
use super::fsx;
use crate::error::{fail, Error};
use std::io::Read;
use std::os::unix::fs::MetadataExt;
use std::path::Path;

pub(crate) const MAX_BUNDLE: usize = 32 * 1024 * 1024;
pub(crate) const OUTPUTS: [(&str, &str, &str, &str); 2] = [
    (
        "company",
        ".soda/application/bundle.json",
        "application-bundle.json",
        "application_bundle_sha256",
    ),
    (
        "preview",
        ".soda/application/preview/bundle.json",
        "application-preview-bundle.json",
        "application_preview_bundle_sha256",
    ),
];

fn read_output(dir: &Path, relative: &str) -> Result<Option<Vec<u8>>, Error> {
    let path = dir.join("work/checkout").join(relative);
    match std::fs::symlink_metadata(&path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(e) => return Err(Error::classify(e)),
        Ok(_) => {}
    }
    if std::fs::canonicalize(&path).map_err(Error::classify)? != path {
        return fail("application build output must be a regular checkout file");
    }
    let file = fsx::open_ro(&path, true)?;
    let metadata = file.metadata().map_err(Error::classify)?;
    if !metadata.is_file()
        || metadata.nlink() != 1
        || metadata.len() == 0
        || metadata.len() > MAX_BUNDLE as u64
    {
        return fail("application build output exceeds its file bound");
    }
    let mut bytes = Vec::new();
    file.take(MAX_BUNDLE as u64 + 1)
        .read_to_end(&mut bytes)
        .map_err(Error::classify)?;
    if bytes.is_empty() || bytes.len() > MAX_BUNDLE {
        return fail("application build output exceeds its byte bound");
    }
    Ok(Some(bytes))
}

pub(crate) fn retain(ctx: &crate::Ctx, dir: &Path) -> Result<(), Error> {
    let mut outputs = Vec::new();
    for (_, relative, retained, _) in OUTPUTS {
        if let Some(bytes) = read_output(dir, relative)? {
            outputs.push((retained, bytes));
        }
    }
    if outputs.len() == 2 {
        let origin = |bytes: &[u8]| -> Result<String, Error> {
            let value: serde_json::Value = serde_json::from_slice(bytes)
                .map_err(|_| Error::fail("application bundle metadata is invalid"))?;
            value
                .pointer("/environment/vars/CAN_AUTH_ORIGIN")
                .and_then(serde_json::Value::as_str)
                .filter(|v| !v.is_empty())
                .map(str::to_string)
                .ok_or_else(|| Error::fail("application bundle has no intended origin"))
        };
        if origin(&outputs[0].1)? == origin(&outputs[1].1)? {
            return fail("preview and company outputs must have distinct intended origins");
        }
    }
    fsx::owned_dir(ctx, dir, 0o755)?;
    for (retained, bytes) in outputs {
        fsx::write_new(&dir.join(retained), &bytes, 0o644)?;
        fsx::open_ro(&dir.join(retained), true)?
            .sync_all()
            .map_err(Error::classify)?;
    }
    super::verification_execution::sync_dir(dir)
}
