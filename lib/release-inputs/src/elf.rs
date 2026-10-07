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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn header_fields_require_a_complete_elf64_little_endian_prefix() {
        let mut bytes = [0u8; HEADER_LEN];
        bytes[..6].copy_from_slice(&[0x7f, b'E', b'L', b'F', 2, 1]);
        bytes[16..18].copy_from_slice(&3u16.to_le_bytes());
        bytes[18..20].copy_from_slice(&183u16.to_le_bytes());
        assert_eq!(
            elf64_le_header(&bytes),
            Some(Elf64Le {
                machine: 183,
                file_type: 3
            })
        );
        assert_eq!(elf64_le_header(&bytes[..63]), None);
        for index in [0, 4, 5] {
            let mut invalid = bytes;
            invalid[index] = 0;
            assert_eq!(elf64_le_header(&invalid), None);
        }
    }
}
