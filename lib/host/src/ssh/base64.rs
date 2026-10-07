use base64::engine::general_purpose::{GeneralPurpose, GeneralPurposeConfig};
use base64::engine::DecodePaddingMode;
use base64::{alphabet, DecodeError, Engine};

fn go_std() -> GeneralPurpose {
    GeneralPurpose::new(
        &alphabet::STANDARD,
        GeneralPurposeConfig::new()
            .with_decode_padding_mode(DecodePaddingMode::RequireCanonical)
            .with_decode_allow_trailing_bits(true),
    )
}

fn source_offset(indices: &[usize], src_len: usize, compact_offset: usize) -> usize {
    indices.get(compact_offset).copied().unwrap_or(src_len)
}

/// Translate a crate error back to Go's `CorruptInputError` offset. Padding
/// errors need a little caller-local mapping because Go reports the expected
/// second `=` or the first byte after final padding in those cases.
fn go_error_offset(
    error: DecodeError,
    compact: &[u8],
    source_indices: &[usize],
    src_len: usize,
) -> usize {
    match error {
        DecodeError::InvalidByte(index, b'=') => match index % 4 {
            0 | 1 => source_offset(source_indices, src_len, index),
            2 => {
                let second_pad = index + 1;
                if compact.get(second_pad) != Some(&b'=') {
                    if let Some(&next_source_index) = source_indices.get(second_pad) {
                        return next_source_index.saturating_sub(1);
                    }
                    return src_len;
                }
                source_offset(source_indices, src_len, second_pad + 1)
            }
            _ => source_offset(source_indices, src_len, index + 1),
        },
        DecodeError::InvalidByte(index, _) | DecodeError::InvalidLastSymbol(index, _) => {
            source_offset(source_indices, src_len, index)
        }
        DecodeError::InvalidLength(_) | DecodeError::InvalidPadding => {
            let last_quantum = compact.len() - compact.len() % 4;
            let trailing_pad_count = compact[last_quantum..]
                .iter()
                .rev()
                .take_while(|byte| **byte == b'=')
                .count();
            let symbol_count = compact[last_quantum..].len() - trailing_pad_count;
            if trailing_pad_count == 1 && symbol_count == 2 {
                src_len
            } else {
                source_len_for_incomplete_quantum(
                    compact,
                    source_indices,
                    src_len,
                    last_quantum,
                    symbol_count,
                )
            }
        }
    }
}

fn source_len_for_incomplete_quantum(
    compact: &[u8],
    source_indices: &[usize],
    src_len: usize,
    start: usize,
    symbol_count: usize,
) -> usize {
    if start + symbol_count == compact.len() {
        src_len.saturating_sub(symbol_count)
    } else {
        source_offset(source_indices, src_len, start)
    }
}

/// Decode a padded SSH blob. SSH key lines refuse all whitespace at their
/// parser boundary; this engine preserves Go's lenient unused trailing bits.
pub fn b64_decode(s: &str) -> Result<Vec<u8>, String> {
    if s.is_empty() {
        return Err("invalid base64".to_string());
    }
    go_std().decode(s).map_err(|_| "invalid base64".to_string())
}

/// `base64.StdEncoding.Decode` for JSON `[]byte` fields: decoded bytes on
/// success, the `CorruptInputError` byte offset on failure.
pub fn b64_decode_go(src: &[u8]) -> Result<Vec<u8>, usize> {
    if src.is_empty() {
        return Ok(Vec::new());
    }

    let mut compact = Vec::with_capacity(src.len());
    let mut source_indices = Vec::with_capacity(src.len());
    for (index, &byte) in src.iter().enumerate() {
        if byte != b'\r' && byte != b'\n' {
            compact.push(byte);
            source_indices.push(index);
        }
    }

    go_std()
        .decode(&compact)
        .map_err(|error| go_error_offset(error, &compact, &source_indices, src.len()))
}

/// `CorruptInputError.Error()`: `illegal base64 data at input byte N`.
pub fn b64_corrupt(offset: usize) -> String {
    format!("illegal base64 data at input byte {offset}")
}

pub fn b64_encode(raw: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(raw)
}

pub fn b64_encode_raw(raw: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD_NO_PAD.encode(raw)
}
