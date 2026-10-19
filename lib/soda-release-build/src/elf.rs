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
