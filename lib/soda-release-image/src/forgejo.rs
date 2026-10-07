//! `forgejo_source.go`: exact fork snapshot extraction.

use crate::error::Error;
use crate::foreign::Production;
use crate::model;
use crate::request;
use crate::sys;

/// The fork is a second exact source input. It is archived beside the Soda
/// snapshot before any generated bindata or native compilation changes it.
/// The directory name must stay forgejo-ext: sodaos go.mod replaces the
/// extension SDK with ../forgejo-ext/sdk relative to the soda snapshot.
pub fn extract_forgejo_snapshot(
    production: &dyn Production,
    request: &request::Request,
) -> Result<(), Error> {
    if !model::is_revision(&request.forgejo_revision) {
        return Err(Error::msg("exact Forgejo source revision required"));
    }
    let snapshot = sys::join(&[&request.out, "work/forgejo-ext"]);
    sys::create_dir(&snapshot, 0o700)?;
    let archive = sys::join(&[&request.out, "artifacts/forgejo-source.tar"]);
    production.execute(
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
    production.execute(
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
        struct Stub;
        impl Production for Stub {
            fn source(&self) -> &str {
                ""
            }
            fn forgejo_source(&self) -> &str {
                ""
            }
            fn forgejo_revision(&self) -> &str {
                ""
            }
            fn native(&self) -> &str {
                ""
            }
            fn out(&self) -> &str {
                ""
            }
            fn arch(&self) -> &str {
                "x86_64"
            }
            fn revision(&self) -> &str {
                ""
            }
            fn live_inputs(&self) -> &str {
                ""
            }
            fn execute(&self, _: &str, _: &str, _: &[String]) -> Result<(), Error> {
                panic!("dispatched")
            }
            fn capture(&self, _: &str, _: &str, _: &[String]) -> Result<String, Error> {
                Ok(String::new())
            }
            fn next(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn resolve_inputs(&mut self) -> Result<(), Error> {
                Ok(())
            }
            fn dependencies(&self) -> Result<(), Error> {
                Ok(())
            }
            fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn compile_rust(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn stage_fork_binary(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn assets(&self, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn images(
                &self,
                _: &str,
            ) -> Result<std::collections::HashMap<String, model::ProducedImage>, Error>
            {
                Ok(Default::default())
            }
            fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
                Ok(Default::default())
            }
            fn verify_content(
                &self,
                _: &model::Payload,
                _: &str,
            ) -> Result<(std::collections::HashMap<String, String>, u64), Error> {
                Ok(Default::default())
            }
            fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, Error> {
                Ok(Default::default())
            }
            fn read_live_inputs(&self, _: &str) -> Result<model::LiveInputs, Error> {
                Ok(Default::default())
            }
            fn check_native(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn sign_media(
                &self,
                _: &model::Trust,
                _: &model::Permit,
                _: &str,
                _: &str,
                _: &str,
                _: &model::SecretFiles,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn verify_copy(
                &self,
                _: &model::Trust,
                _: &str,
                _: &str,
                _: &str,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn write_document(
                &self,
                _: &str,
                _: &crate::foreign::PackagingInputs,
            ) -> Result<String, Error> {
                Ok(String::new())
            }
        }
        let request = request::Request {
            forgejo_revision: "short".to_string(),
            ..request::Request::default()
        };
        assert_eq!(
            extract_forgejo_snapshot(&Stub, &request).unwrap_err().0,
            "exact Forgejo source revision required"
        );
    }
}
