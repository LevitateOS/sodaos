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

/// Split one complete JSON object off the front of `buffer`, tracking
/// strings and escapes like a streaming decoder. Returns the object
/// length when balanced.
pub fn split_json_object(buffer: &[u8]) -> Option<usize> {
    let mut depth = 0i32;
    let mut in_string = false;
    let mut escape = false;
    let mut started = false;
    for (i, &b) in buffer.iter().enumerate() {
        if in_string {
            if escape {
                escape = false;
            } else if b == b'\\' {
                escape = true;
            } else if b == b'"' {
                in_string = false;
            }
            continue;
        }
        match b {
            b'"' => in_string = true,
            b'{' => {
                depth += 1;
                started = true;
            }
            b'}' => {
                depth -= 1;
                if started && depth == 0 {
                    return Some(i + 1);
                }
                if depth < 0 {
                    return None;
                }
            }
            _ => {}
        }
    }
    None
}
