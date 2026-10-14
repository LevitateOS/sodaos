//! Architecture-selected offline Canlang inputs. Missing native releases remain
//! missing; an archive from another architecture is never a substitute.

use super::Error;

pub fn release_path(arch: &str) -> Result<&'static str, Error> {
    match arch {
        "x86_64" => Ok("system/project/canlang-release.json"),
        "aarch64" => Ok("system/project/canlang-release-aarch64.json"),
        _ => Err(Error("expected x86_64 or aarch64".into())),
    }
}

pub fn archive_filename(arch: &str) -> Result<&'static str, Error> {
    match arch {
        "x86_64" => Ok("canlang-tools-linux-x86_64.tar.gz"),
        "aarch64" => Ok("canlang-tools-linux-aarch64.tar.gz"),
        _ => Err(Error("expected x86_64 or aarch64".into())),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn selected_native_inputs_are_distinct_and_cannot_escape_the_source() {
        assert_eq!(
            release_path("x86_64").unwrap(),
            "system/project/canlang-release.json"
        );
        assert_eq!(
            release_path("aarch64").unwrap(),
            "system/project/canlang-release-aarch64.json"
        );
        assert_eq!(
            archive_filename("aarch64").unwrap(),
            "canlang-tools-linux-aarch64.tar.gz"
        );
        for invalid in ["", "arm64", "../x86_64"] {
            assert!(release_path(invalid).is_err());
            assert!(archive_filename(invalid).is_err());
        }
    }
}
