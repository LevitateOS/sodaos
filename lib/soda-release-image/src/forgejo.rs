//! `forgejo_source.go`: exact fork snapshot extraction.

use crate::build_runner::Runner;
use crate::error::Error;
use crate::model;
use crate::request;
use crate::sys;

/// The fork is a second exact source input. It is archived beside the Soda
/// snapshot before any generated bindata or native compilation changes it.
/// The directory name must stay forgejo-ext: sodaos go.mod replaces the
/// extension SDK with ../forgejo-ext/sdk relative to the soda snapshot.
pub fn extract_forgejo_snapshot(runner: &Runner, request: &request::Request) -> Result<(), Error> {
    if !model::is_revision(&request.forgejo_revision) {
        return Err(Error::msg("exact Forgejo source revision required"));
    }
    let snapshot = sys::join(&[&request.out, "work/forgejo-ext"]);
    sys::create_dir(&snapshot, 0o700)?;
    let archive = sys::join(&[&request.out, "artifacts/forgejo-source.tar"]);
    runner.execute(
        &request.forgejo_source,
        "git",
        &[
            "-c".to_string(),
            format!("safe.directory={}", request.forgejo_source),
            "archive".to_string(),
            "--format=tar".to_string(),
            "--output".to_string(),
            archive.clone(),
            request.forgejo_revision.clone(),
        ],
    )?;
    runner.execute(
        &snapshot,
        "tar",
        &[
            "--extract".to_string(),
            "--file".to_string(),
            archive,
            "--no-same-owner".to_string(),
        ],
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_forgejo_snapshot_refuses_bad_revision() {
        // Oracle: Go TestExtractForgejoSnapshotRefusesBadRevision.
        let request = request::Request {
            forgejo_revision: "short".to_string(),
            ..request::Request::default()
        };
        let runner = Runner::new(std::rc::Rc::new(crate::build_runner::Cancel::new()));
        assert_eq!(
            extract_forgejo_snapshot(&runner, &request).unwrap_err().0,
            "exact Forgejo source revision required"
        );
    }
}
