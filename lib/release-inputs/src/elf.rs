//! Bounded ELF header access; callers retain their machine/type allowlists.

pub const HEADER_LEN: usize = 64;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Elf64Le {
    pub machine: u16,
    pub file_type: u16,
}

pub fn elf64_le_header(bytes: &[u8]) -> Option<Elf64Le> {
    if bytes.len() < HEADER_LEN || bytes[..6] != [0x7f, b'E', b'L', b'F', 2, 1] {
        return None;
    }
    Some(Elf64Le {
        file_type: u16::from_le_bytes([bytes[16], bytes[17]]),
        machine: u16::from_le_bytes([bytes[18], bytes[19]]),
    })
}
