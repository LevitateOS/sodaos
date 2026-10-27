//! Read immutable image bytes with the native owner, never execute a utility
//! supplied by the application image or trust an operator-provided label.
use super::{wire::Spec, Runtime};
use crate::{project::Executor, sha256};
use std::fs::File;
use std::io::Read;
use std::path::PathBuf;
use std::time::Instant;

impl<E: Executor> Runtime<E> {
    pub(super) fn bundle_file_sha256(
        &self,
        spec: &Spec,
        deadline: Instant,
    ) -> Result<String, String> {
        self.admit_image(spec, deadline)?;
        let mounted = self.command(
            "podman",
            &["image".into(), "mount".into(), spec.image.clone()],
            deadline,
        )?;
        let result = (|| {
            let path = std::str::from_utf8(&mounted)
                .map_err(|_| "invalid image mount observation")?
                .trim();
            let root = PathBuf::from(path);
            if !root.is_absolute()
                || path.contains(['\n', '\r'])
                || std::fs::canonicalize(&root).map_err(|_| "image mount unavailable")? != root
            {
                return Err("invalid image mount observation".into());
            }
            let bundle = root.join("opt/application/bundle.json");
            if std::fs::canonicalize(&bundle).map_err(|_| "application bundle unavailable")?
                != bundle
            {
                return Err("application bundle must be a regular image file".into());
            }
            let file = File::open(&bundle).map_err(|_| "application bundle unavailable")?;
            let metadata = file
                .metadata()
                .map_err(|_| "application bundle unavailable")?;
            const LIMIT: u64 = 32 * 1024 * 1024;
            if !metadata.is_file() || metadata.len() == 0 || metadata.len() > LIMIT {
                return Err("application bundle exceeds its readback bound".into());
            }
            let mut bytes = Vec::new();
            file.take(LIMIT + 1)
                .read_to_end(&mut bytes)
                .map_err(|_| "application bundle readback failed")?;
            if bytes.is_empty() || bytes.len() as u64 > LIMIT {
                return Err("application bundle exceeds its readback bound".into());
            }
            Ok(sha256::hex_lower(&sha256::digest(&bytes)))
        })();
        // Release only the mount reference acquired by this exact operation.
        self.command(
            "podman",
            &["image".into(), "unmount".into(), spec.image.clone()],
            deadline,
        )?;
        result
    }
}
