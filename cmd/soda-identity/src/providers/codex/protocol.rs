// Codex app-server JSON-RPC protocol, folded from identity-providers (A06.M).
use super::super::Error;
use super::{Inner, Session};
use serde::Deserialize;
use std::io::{BufRead, BufReader, Write};
use std::process::ChildStdout;
use std::sync::atomic::Ordering;
use std::sync::mpsc;
use std::time::{Duration, Instant};
#[derive(Debug, Clone, Deserialize)]
pub(super) struct Message {
    #[serde(default)]
    id: i64,
    #[serde(default)]
    method: String,
    #[serde(default)]
    result: serde_json::Value,
    #[serde(default)]
    error: serde_json::Value,
    #[serde(default)]
    params: serde_json::Value,
}

impl Session {
    pub(super) fn send(&self, msg: &serde_json::Value) -> Result<(), Error> {
        let mut guard = self.inner.write.lock().unwrap();
        let stdin = guard
            .as_mut()
            .ok_or_else(|| Error::failed("codex process stopped"))?;
        serde_json::to_writer(&mut *stdin, msg)?;
        stdin.write_all(b"\n")?;
        stdin.flush()?;
        Ok(())
    }

    pub(super) fn call(
        &self,
        method: &str,
        params: &serde_json::Value,
        timeout: Duration,
    ) -> Result<serde_json::Value, Error> {
        let id = {
            let mut next = self.inner.next.lock().unwrap();
            *next += 1;
            *next
        };
        let (tx, rx) = mpsc::sync_channel(1);
        self.inner.replies.lock().unwrap().insert(id, tx);
        let request = serde_json::json!({"id": id, "method": method, "params": params});
        if let Err(err) = self.send(&request) {
            self.inner.replies.lock().unwrap().remove(&id);
            return Err(err);
        }
        let deadline = Instant::now() + timeout;
        loop {
            if self.inner.cancelled.load(Ordering::SeqCst) {
                self.inner.replies.lock().unwrap().remove(&id);
                return Err(Error::failed("context canceled"));
            }
            match rx.recv_timeout(Duration::from_millis(50)) {
                Ok(msg) => {
                    self.inner.replies.lock().unwrap().remove(&id);
                    if !msg.error.is_null() {
                        return Err(protocol_error(&msg.error));
                    }
                    return Ok(msg.result);
                }
                Err(mpsc::RecvTimeoutError::Timeout) => {
                    if self.inner.finished.load(Ordering::SeqCst) {
                        self.inner.replies.lock().unwrap().remove(&id);
                        return Err(Error::failed("codex process stopped"));
                    }
                    if Instant::now() >= deadline {
                        self.inner.replies.lock().unwrap().remove(&id);
                        return Err(Error::failed("codex protocol timed out"));
                    }
                }
                Err(mpsc::RecvTimeoutError::Disconnected) => {
                    self.inner.replies.lock().unwrap().remove(&id);
                    return Err(Error::failed("codex process stopped"));
                }
            }
        }
    }
}

pub(super) fn protocol_error(raw: &serde_json::Value) -> Error {
    #[derive(Deserialize)]
    struct Detail {
        #[serde(default)]
        message: String,
    }
    let detail: Detail = serde_json::from_value(raw.clone()).unwrap_or(Detail {
        message: String::new(),
    });
    let message = detail.message.to_lowercase();
    if message.contains("device")
        && (message.contains("not enabled") || message.contains("disabled"))
    {
        return Error::failed(
            "device-code login is disabled; enable it in ChatGPT security settings or workspace permissions",
        );
    }
    Error::failed("codex protocol request failed")
}

pub(super) fn read_loop(inner: &Inner, out: ChildStdout) {
    let mut reader = BufReader::new(out);
    let mut line = Vec::with_capacity(4096);
    loop {
        line.clear();
        // Mirror bufio.Scanner with a 512 KiB token cap: overlong or
        // malformed lines end the enrollment read loop.
        let mut total = 0usize;
        let mut complete = false;
        while let Ok(chunk) = reader.fill_buf() {
            if chunk.is_empty() {
                break;
            }
            let mut consumed = 0usize;
            for &b in chunk {
                consumed += 1;
                total += 1;
                if total > (512 << 10) {
                    break;
                }
                if b == b'\n' {
                    complete = true;
                    break;
                }
                line.push(b);
            }
            reader.consume(consumed);
            if complete || total > (512 << 10) {
                break;
            }
        }
        if !complete {
            break;
        }
        if line.last() == Some(&b'\r') {
            line.pop();
        }
        let Ok(msg) = serde_json::from_slice::<Message>(&line) else {
            break;
        };
        if msg.id != 0 {
            if let Some(tx) = inner.replies.lock().unwrap().get(&msg.id) {
                // The protocol never repeats an id; a duplicate is dropped
                // instead of blocking the reader on a full channel.
                let _ = tx.try_send(msg);
            }
        } else {
            notify(inner, &msg);
        }
    }
    let mut state = inner.state.lock().unwrap();
    if state.state == "pending" {
        state.state = "failed".to_string();
        state.error = "Provider enrollment ended".to_string();
    }
}

fn notify(inner: &Inner, msg: &Message) {
    if msg.method != "account/login/completed" {
        return;
    }
    #[derive(Deserialize)]
    struct Event {
        #[serde(rename = "loginId")]
        login_id: String,
        success: bool,
    }
    let Ok(event) = serde_json::from_value::<Event>(msg.params.clone()) else {
        return;
    };
    let mut state = inner.state.lock().unwrap();
    if !state.id.is_empty() && event.login_id != state.id {
        return;
    }
    if event.success {
        state.state = "completed".to_string();
        return;
    }
    state.state = "failed".to_string();
    state.error = "Provider enrollment failed; retry or check device-code access".to_string();
}
