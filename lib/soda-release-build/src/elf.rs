//! Native executable admission (`elf.go`): 64-bit little-endian x86-64
//! executables or shared objects only.

use crate::{io_error, Error};
use soda_build_tools::elf::{elf64_le_header, HEADER_LEN};
use std::io::{ErrorKind, Read};
use std::path::Path;

/// Verifies a natively executable artifact for the architecture.
pub fn inspect_elf(path: &Path, arch: &str) -> Result<(), Error> {
    if arch != "x86_64" {
        return Err(Error::msg("unknown native architecture"));
    }
    let mut file = std::fs::File::open(path).map_err(|e| io_error("open", path, e))?;
    let mut data = [0; HEADER_LEN];
    if let Err(error) = file.read_exact(&mut data) {
        return Err(if error.kind() == ErrorKind::UnexpectedEof {
            Error::msg("native executable format/platform mismatch")
        } else {
            io_error("read", path, error)
        });
    }
    if !is_native_executable(&data) {
        return Err(Error::msg("native executable format/platform mismatch"));
    }
    Ok(())
}

fn is_native_executable(data: &[u8]) -> bool {
    elf64_le_header(data)
        .is_some_and(|header| header.machine == 62 && matches!(header.file_type, 2 | 3))
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;

    pub(crate) fn fixture_elf() -> [u8; 64] {
        let mut elf = [0u8; 64];
        elf[0..7].copy_from_slice(&[0x7f, b'E', b'L', b'F', 2, 1, 1]);
        elf[16..18].copy_from_slice(&2u16.to_le_bytes());
        elf[18..20].copy_from_slice(&62u16.to_le_bytes());
        elf[20..24].copy_from_slice(&1u32.to_le_bytes());
        elf[52..54].copy_from_slice(&64u16.to_le_bytes());
        elf
    }

    #[test]
    fn native_elf_header_profile() {
        let dir = std::env::temp_dir().join(format!("soda-elf-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let good = dir.join("good");
        std::fs::write(&good, fixture_elf()).unwrap();
        assert!(inspect_elf(&good, "x86_64").is_ok());
        assert_eq!(
            inspect_elf(&good, "aarch64").unwrap_err().message(),
            "unknown native architecture"
        );
        let bad = dir.join("bad");
        std::fs::write(&bad, b"not ELF").unwrap();
        assert_eq!(
            inspect_elf(&bad, "x86_64").unwrap_err().message(),
            "native executable format/platform mismatch"
        );
        let mut wrong_machine = fixture_elf();
        wrong_machine[18..20].copy_from_slice(&3u16.to_le_bytes());
        std::fs::write(&bad, wrong_machine).unwrap();
        assert!(inspect_elf(&bad, "x86_64").is_err());
    }
}
