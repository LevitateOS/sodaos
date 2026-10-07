use super::LaunchExit;

/// `museCommandExit`: shell wait status to launch outcome.
pub fn muse_command_exit(status: std::io::Result<std::process::ExitStatus>) -> LaunchExit {
    use std::os::unix::process::ExitStatusExt;
    match status {
        Ok(status) => match status.code() {
            Some(code) => LaunchExit {
                code,
                error: String::new(),
            },
            None => LaunchExit {
                code: 128 + status.signal().unwrap_or(0),
                error: String::new(),
            },
        },
        Err(_) => LaunchExit {
            code: 1,
            error: String::new(),
        },
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum JsonPacket {
    Complete(usize),
    Incomplete,
    Invalid,
}

/// Read one raw JSON value from a Muse packet stream without consuming the
/// following packet. Syntax framing is left to serde_json; the launch DTO
/// performs strict duplicate and field admission after this boundary.
pub fn split_json_object(buffer: &[u8]) -> JsonPacket {
    let mut stream = serde_json::Deserializer::from_slice(buffer)
        .into_iter::<Box<serde_json::value::RawValue>>();
    match stream.next() {
        Some(Ok(raw)) if raw.get().starts_with('{') => JsonPacket::Complete(stream.byte_offset()),
        Some(Ok(_)) => JsonPacket::Invalid,
        Some(Err(error)) if error.is_eof() => JsonPacket::Incomplete,
        Some(Err(_)) => JsonPacket::Invalid,
        None => JsonPacket::Incomplete,
    }
}
