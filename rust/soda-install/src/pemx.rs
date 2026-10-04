//! Go `encoding/pem` decode: finds the first `-----BEGIN <type>-----`
//! block, returning its type, decoded bytes, and trailing remainder.

use crate::sshkey::b64_decode_go;

/// Decoded block: type line, header count (for blank-line accounting), and
/// base64-decoded bytes.
pub struct Block {
    pub der_type: String,
    pub bytes: Vec<u8>,
}

fn get_line(data: &[u8]) -> (&[u8], &[u8], usize) {
    match data.iter().position(|b| *b == b'\n') {
        Some(i) => {
            let mut end = i;
            if end > 0 && data[end - 1] == b'\r' {
                end -= 1;
            }
            let line = &data[..end];
            let trim = line.iter().rposition(|b| *b != b' ' && *b != b'\t').map(|p| p + 1).unwrap_or(0);
            (&line[..trim], &data[i + 1..], i + 1)
        }
        None => {
            let trim = data.iter().rposition(|b| *b != b' ' && *b != b'\t').map(|p| p + 1).unwrap_or(0);
            (&data[..trim], &[], data.len())
        }
    }
}

fn find(haystack: &[u8], needle: &[u8]) -> Option<usize> {
    if haystack.len() < needle.len() {
        return None;
    }
    haystack.windows(needle.len()).position(|w| w == needle)
}

fn rfind(haystack: &[u8], needle: &[u8]) -> Option<usize> {
    if haystack.len() < needle.len() {
        return None;
    }
    haystack.windows(needle.len()).rposition(|w| w == needle)
}

/// Go `pem.Decode`: `(block, rest)`, or `(None, whole input)`.
pub fn decode(data: &[u8]) -> (Option<Block>, &[u8]) {
    const BEGIN: &[u8] = b"-----BEGIN ";
    const END: &[u8] = b"\n-----END ";
    const END_OF_LINE: &[u8] = b"-----";
    let mut rest = data;
    let mut end_trailer: isize = 0;
    loop {
        if end_trailer < 0 || end_trailer as usize > rest.len() {
            return (None, data);
        }
        rest = &rest[end_trailer as usize..];
        let end_index = match find(rest, END) {
            Some(i) => i as isize,
            None => return (None, data),
        };
        let mut trailer = end_index + END.len() as isize;
        let begin_index = match rfind(&rest[..end_index as usize], BEGIN) {
            Some(i) => i,
            None => {
                end_trailer = trailer;
                continue;
            }
        };
        if begin_index > 0 && rest[begin_index - 1] != b'\n' {
            end_trailer = trailer;
            continue;
        }
        // Our BEGIN is Go's pemStart[1:], so its length already equals
        // len(pemStart)-1.
        let mut end_index = end_index - begin_index as isize - BEGIN.len() as isize;
        trailer -= begin_index as isize + BEGIN.len() as isize;
        rest = &rest[begin_index + BEGIN.len()..];
        let (type_line, next, consumed) = get_line(rest);
        rest = next;
        end_index -= consumed as isize;
        trailer -= consumed as isize;
        if !type_line.ends_with(END_OF_LINE) {
            end_trailer = trailer;
            continue;
        }
        let der_type = &type_line[..type_line.len() - END_OF_LINE.len()];
        let mut headers = 0;
        loop {
            if rest.is_empty() {
                return (None, data);
            }
            let (line, next, consumed) = get_line(rest);
            let cut = line.iter().position(|b| *b == b':');
            match cut {
                None => break,
                Some(_) => {
                    // Headers only matter for blank-line accounting; the
                    // installer never reads them.
                    headers += 1;
                    rest = next;
                    end_index -= consumed as isize;
                    trailer -= consumed as isize;
                }
            }
        }
        if headers > 0 && end_index < 0 {
            end_trailer = trailer;
            continue;
        }
        if trailer < 0 || trailer as usize > rest.len() {
            end_trailer = trailer;
            continue;
        }
        let trailer_bytes = &rest[trailer as usize..];
        let want = der_type.len() + END_OF_LINE.len();
        if trailer_bytes.len() < want {
            end_trailer = trailer;
            continue;
        }
        if &trailer_bytes[..der_type.len()] != der_type || !trailer_bytes[..want].ends_with(END_OF_LINE) {
            end_trailer = trailer;
            continue;
        }
        let (tail, _, _) = get_line(&trailer_bytes[want..]);
        if !tail.is_empty() {
            end_trailer = trailer;
            continue;
        }
        let mut bytes = Vec::new();
        if end_index > 0 {
            if end_index as usize > rest.len() {
                end_trailer = trailer;
                continue;
            }
            let clean: Vec<u8> = rest[..end_index as usize]
                .iter()
                .filter(|b| **b != b' ' && **b != b'\t')
                .copied()
                .collect();
            match b64_decode_go(&clean) {
                Ok(decoded) => bytes = decoded,
                Err(_) => {
                    end_trailer = trailer;
                    continue;
                }
            }
        }
        let at = end_index + END.len() as isize - 1;
        if at < 0 || at as usize > rest.len() {
            return (None, data);
        }
        let (_, tail, _) = get_line(&rest[at as usize..]);
        return (
            Some(Block { der_type: String::from_utf8_lossy(der_type).into_owned(), bytes }),
            tail,
        );
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn block(der_type: &str, bytes: &[u8]) -> Vec<u8> {
        let mut out = format!("-----BEGIN {der_type}-----\n").into_bytes();
        out.extend(crate::sshkey::b64_encode(bytes).as_bytes());
        out.push(b'\n');
        out.extend(format!("-----END {der_type}-----\n").as_bytes());
        out
    }

    #[test]
    fn oracle_pem_vectors() {
        // Oracle: TestZZOraclePEM `PEM` lines.
        let der = [0x30u8, 0x03, 0x02, 0x01, 0x05];
        let clean = block("CERTIFICATE", &der);
        let (found, rest) = decode(&clean);
        let found = found.unwrap();
        assert_eq!(found.der_type, "CERTIFICATE");
        assert_eq!(found.bytes, der);
        assert!(rest.is_empty());

        let mut leading = b"garbage\nline2\n".to_vec();
        leading.extend(block("CERTIFICATE", &der));
        let (found, rest) = decode(&leading);
        assert!(found.is_some());
        assert!(rest.is_empty());

        let mut two = block("CERTIFICATE", &der);
        two.extend(block("CERTIFICATE", &der));
        let (found, rest) = decode(&two);
        assert!(found.is_some());
        assert_eq!(rest, block("CERTIFICATE", &der));

        let mut trailing = block("CERTIFICATE", &der);
        trailing.extend(b"  \n\t");
        let (found, rest) = decode(&trailing);
        assert!(found.is_some());
        assert_eq!(rest, b"  \n\t");

        let (found, _) = decode(&block("PRIVATE KEY", &der));
        assert_eq!(found.unwrap().der_type, "PRIVATE KEY");

        let lower = format!(
            "-----begin certificate-----\n{}\n-----end certificate-----\n",
            crate::sshkey::b64_encode(&der)
        );
        let (found, rest) = decode(lower.as_bytes());
        assert!(found.is_none());
        assert_eq!(rest, lower.as_bytes());

        let noend = format!("-----BEGIN CERTIFICATE-----\n{}\n", crate::sshkey::b64_encode(&der));
        let (found, rest) = decode(noend.as_bytes());
        assert!(found.is_none());
        assert_eq!(rest, noend.as_bytes());

        let (found, rest) = decode(b"");
        assert!(found.is_none());
        assert!(rest.is_empty());

        let crlf = format!(
            "-----BEGIN CERTIFICATE-----\r\n{}\r\n-----END CERTIFICATE-----\r\n",
            crate::sshkey::b64_encode(&der)
        );
        let (found, rest) = decode(crlf.as_bytes());
        assert!(found.is_some());
        assert!(rest.is_empty());
    }
}
