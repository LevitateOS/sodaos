use std::collections::BTreeMap;
use std::io::{Read, Write};
use std::os::fd::AsRawFd;
use std::time::Instant;

use serde_json::{json, Value};

use super::wire::{self, Notice};

pub const MAX_FRAME_BYTES: usize = 8 * 1024 * 1024;
const MAX_ITEMS: usize = 8192;

#[derive(Debug)]
pub enum Error {
    Io(std::io::Error),
    Deadline,
    Bounds,
    End,
    Gap,
    InputRequired,
    Invalid(&'static str),
    Refused { code: i64, reason: Option<String> },
}

impl From<std::io::Error> for Error {
    fn from(error: std::io::Error) -> Self {
        Self::Io(error)
    }
}

/// One sequential command channel with streaming observations. There is no
/// autonomous retry: a caller persists its command identity before sending and
/// fences uncertain outcomes instead of starting another session or child.
pub struct Connection<W, R> {
    writer: Option<W>,
    reader: R,
    buffered: Vec<u8>,
    next_id: u64,
    revisions: BTreeMap<(String, String), (u64, [u8; 32])>,
    pub(super) session: Option<String>,
    pub(super) codex: bool,
}

impl<W: Write + AsRawFd, R: Read + AsRawFd> Connection<W, R> {
    pub fn new(writer: W, reader: R) -> Self {
        Self {
            writer: Some(writer),
            reader,
            buffered: Vec::new(),
            next_id: 0,
            revisions: BTreeMap::new(),
            session: None,
            codex: false,
        }
    }

    pub fn codex(writer: W, reader: R) -> Self {
        let mut connection = Self::new(writer, reader);
        connection.codex = true;
        connection
    }

    /// EOF only closes input. The existing native run owner must still wait for
    /// process/unit retirement and reconcile broker custody before success.
    pub fn close_input(&mut self) {
        drop(self.writer.take());
    }

    fn write(&mut self, value: &Value, deadline: Instant) -> Result<(), Error> {
        let mut bytes = serde_json::to_vec(value).map_err(|_| Error::Invalid("native request"))?;
        if bytes.len() >= MAX_FRAME_BYTES {
            return Err(Error::Bounds);
        }
        bytes.push(b'\n');
        let writer = self.writer.as_mut().ok_or(Error::End)?;
        for chunk in bytes.chunks(4096) {
            ready(writer.as_raw_fd(), libc::POLLOUT, deadline)?;
            writer.write_all(chunk)?;
        }
        writer.flush()?;
        Ok(())
    }

    fn frame(&mut self, deadline: Instant) -> Result<Value, Error> {
        loop {
            if let Some(end) = self.buffered.iter().position(|b| *b == b'\n') {
                if end >= MAX_FRAME_BYTES {
                    return Err(Error::Bounds);
                }
                let line: Vec<u8> = self.buffered.drain(..=end).collect();
                if line.iter().all(u8::is_ascii_whitespace) {
                    continue;
                }
                let frame: Value =
                    serde_json::from_slice(&line).map_err(|_| Error::Invalid("native frame"))?;
                if !self.codex && frame.get("jsonrpc").and_then(Value::as_str) != Some("2.0") {
                    return Err(Error::Invalid("native envelope"));
                }
                return Ok(frame);
            }
            if self.buffered.len() >= MAX_FRAME_BYTES {
                return Err(Error::Bounds);
            }
            ready(self.reader.as_raw_fd(), libc::POLLIN, deadline)?;
            let mut chunk = [0u8; 8192];
            let length = self.reader.read(&mut chunk)?;
            if length == 0 {
                return if self.buffered.is_empty() {
                    Err(Error::End)
                } else {
                    Err(Error::Invalid("incomplete native frame"))
                };
            }
            self.buffered.extend_from_slice(&chunk[..length]);
        }
    }

    fn notification(&mut self, frame: Value) -> Result<Option<Notice>, Error> {
        let method = frame
            .get("method")
            .and_then(Value::as_str)
            .ok_or(Error::Invalid("native envelope"))?;
        let params = frame.get("params").cloned().unwrap_or_else(|| json!({}));
        if self.codex {
            if !wire::short(method) {
                return Err(Error::Invalid("native notification"));
            }
            if let Some(id) = frame.get("id") {
                if !id.is_i64() && !id.is_u64() && !id.as_str().is_some_and(wire::short) {
                    return Err(Error::Invalid("native server request identity"));
                }
                return Ok(Some(Notice::CodexRequest {
                    id: id.clone(),
                    method: method.into(),
                    params,
                }));
            }
            return Ok(Some(Notice::Codex {
                method: method.into(),
                params,
            }));
        }
        if frame.get("id").is_some() {
            return Err(Error::InputRequired);
        }
        let notice = wire::notice(method, params)?;
        if let Notice::Item { session, item, .. } = &notice {
            let digest = crate::sha256::digest(
                &serde_json::to_vec(&frame["params"]["item"])
                    .map_err(|_| Error::Invalid("native item"))?,
            );
            let key = (session.clone(), item.item_id.clone());
            if let Some((revision, recorded)) = self.revisions.get(&key) {
                if item.revision < *revision {
                    return Ok(None);
                }
                if item.revision == *revision {
                    return if recorded == &digest {
                        Ok(None)
                    } else {
                        Err(Error::Invalid("conflicting native item revision"))
                    };
                }
            } else if self.revisions.len() >= MAX_ITEMS {
                return Err(Error::Bounds);
            }
            self.revisions.insert(key, (item.revision, digest));
        }
        Ok(Some(notice))
    }

    pub fn receive(&mut self, deadline: Instant) -> Result<Notice, Error> {
        loop {
            let frame = match self.frame(deadline) {
                Err(Error::End) => return Ok(Notice::End),
                value => value?,
            };
            if let Some(notice) = self.notification(frame)? {
                return Ok(notice);
            }
        }
    }

    pub(super) fn notify(&mut self, method: &str, deadline: Instant) -> Result<(), Error> {
        self.write(&json!({"jsonrpc":"2.0", "method":method}), deadline)
    }

    /// The run owner must retain the native request identity before replying.
    /// This does not authorize a tool; it only sends its already-validated result.
    pub fn reply_codex_tool(
        &mut self,
        id: &Value,
        result: &str,
        success: bool,
        deadline: Instant,
    ) -> Result<(), Error> {
        if !self.codex
            || result.len() > MAX_FRAME_BYTES / 2
            || (!id.is_i64() && !id.is_u64() && !id.as_str().is_some_and(wire::short))
        {
            return Err(Error::Invalid("native tool reply"));
        }
        self.write(&json!({"id":id,"result":{"contentItems":[{"type":"inputText","text":result}],"success":success}}),deadline)
    }

    pub(super) fn call(
        &mut self,
        method: &str,
        params: Value,
        deadline: Instant,
        observe: &mut impl FnMut(Notice) -> Result<(), Error>,
    ) -> Result<Value, Error> {
        self.next_id = self.next_id.checked_add(1).ok_or(Error::Bounds)?;
        let id = self.next_id;
        self.write(
            &json!({"jsonrpc":"2.0", "id":id, "method":method, "params":params}),
            deadline,
        )?;
        loop {
            let frame = self.frame(deadline)?;
            if frame.get("method").is_none() && frame.get("id").is_some() {
                if frame.get("id").and_then(Value::as_u64) != Some(id) {
                    return Err(Error::Invalid("native reply identity"));
                }
                return response(frame);
            }
            if let Some(notice) = self.notification(frame)? {
                observe(notice)?;
            }
        }
    }
}

fn response(frame: Value) -> Result<Value, Error> {
    if let Some(error) = frame.get("error") {
        // Only the strictly validated structured reason code is retained.
        // Daemon free-text messages never enter logs or stored summaries:
        // they can carry paths, prompt content or credentials.
        let reason = error
            .get("data")
            .and_then(|v| v.get("reason"))
            .and_then(Value::as_str)
            .filter(|r| {
                r.len() <= 128
                    && r.bytes()
                        .all(|b| b.is_ascii_alphanumeric() || b"_.:-".contains(&b))
            })
            .map(str::to_owned);
        return Err(Error::Refused {
            code: error
                .get("code")
                .and_then(Value::as_i64)
                .ok_or(Error::Invalid("native refusal"))?,
            reason,
        });
    }
    frame
        .get("result")
        .cloned()
        .ok_or(Error::Invalid("native reply"))
}

fn ready(fd: i32, events: i16, deadline: Instant) -> Result<(), Error> {
    loop {
        let remaining = deadline
            .checked_duration_since(Instant::now())
            .ok_or(Error::Deadline)?;
        let timeout = remaining.as_millis().clamp(1, i32::MAX as u128) as i32;
        let mut poll = libc::pollfd {
            fd,
            events,
            revents: 0,
        };
        let result = unsafe { libc::poll(&mut poll, 1, timeout) };
        if result == 0 {
            return Err(Error::Deadline);
        }
        if result < 0 {
            let error = std::io::Error::last_os_error();
            if error.kind() == std::io::ErrorKind::Interrupted {
                continue;
            }
            return Err(Error::Io(error));
        }
        if poll.revents & (events | libc::POLLHUP) != 0 {
            return Ok(());
        }
        return Err(Error::Invalid("native transport closed"));
    }
}
