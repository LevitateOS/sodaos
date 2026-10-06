use super::*;

fn login() -> Account {
    Account {
        pw_name: "op".to_string(),
        pw_uid: 1001,
        pw_gid: 1001,
        pw_dir: "/home/op".to_string(),
        pw_shell: "/bin/bash".to_string(),
    }
}

#[test]
fn line_bytes_exact() {
    assert_eq!(ready_line(), b"{\"type\":\"ready\"}\n");
    assert_eq!(
        output_line(b"hi"),
        b"{\"type\":\"output\",\"data\":\"aGk=\"}\n"
    );
    assert_eq!(
        output_line(b"\xff\x00binary"),
        b"{\"type\":\"output\",\"data\":\"/wBiaW5hcnk=\"}\n"
    );
    assert_eq!(
        closed_line("expired"),
        b"{\"type\":\"closed\",\"reason\":\"expired\"}\n"
    );
    assert_eq!(
        tmux_attach_argv("/run/s/x"),
        vec![
            "tmux",
            "-N",
            "-S",
            "/run/s/x",
            "attach-session",
            "-E",
            "-t",
            "=soda"
        ]
        .into_iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>()
    );
}

#[test]
fn ingest_matrix() {
    // Input + heartbeat + close across split deliveries.
    let mut incoming = Vec::new();
    let mut to_pty = Vec::new();
    let mut stopped = false;
    incoming.extend_from_slice(b"{\"type\":\"input\",\"data\":\"YWI=\"}\n{\"type\":\"heart");
    let before = sys::monotonic();
    let lease = ingest_control_bytes(&mut incoming, -1, &mut to_pty, before, &mut stopped).unwrap();
    assert_eq!(to_pty, b"ab");
    assert_eq!(lease, before); // no renewal yet
    assert!(!stopped);
    assert_eq!(incoming, b"{\"type\":\"heart"); // partial line retained
    incoming.extend_from_slice(b"beat\"}\n{\"type\":\"close\"}\n");
    let lease = ingest_control_bytes(&mut incoming, -1, &mut to_pty, lease, &mut stopped).unwrap();
    assert!(lease >= before + 60.0 && lease <= before + 61.0);
    assert!(stopped);
    assert!(incoming.is_empty());
    // Oversize frame and oversize tail.
    let mut incoming = vec![b'x'; proto::FRAME_LIMIT + 1];
    assert!(ingest_control_bytes(&mut incoming, -1, &mut Vec::new(), 0.0, &mut false).is_err());
    let mut incoming = format!(
        "{{\"type\":\"input\",\"data\":\"{}\"}}\n",
        "Y".repeat(30000)
    )
    .into_bytes();
    assert!(ingest_control_bytes(&mut incoming, -1, &mut Vec::new(), 0.0, &mut false).is_err());
    // Bad JSON / bad shape.
    let mut incoming = b"not json\n".to_vec();
    assert!(ingest_control_bytes(&mut incoming, -1, &mut Vec::new(), 0.0, &mut false).is_err());
    let mut incoming = b"{\"type\":\"close\",\"x\":1}\n".to_vec();
    assert!(ingest_control_bytes(&mut incoming, -1, &mut Vec::new(), 0.0, &mut false).is_err());
}

#[test]
fn queue_backpressure() {
    let mut to_pty = vec![0u8; proto::QUEUE_LIMIT - 10];
    let mut stopped = false;
    let frame = ControlFrame::Input(vec![0u8; 10]);
    assert!(apply_control_frame(&frame, -1, &mut to_pty, &mut stopped).is_ok());
    let frame = ControlFrame::Input(vec![0u8; 1]);
    assert!(apply_control_frame(&frame, -1, &mut to_pty, &mut stopped).is_err());
    // Resize on a bad fd surfaces (stream failure upstream).
    let frame = ControlFrame::Resize { cols: 80, rows: 24 };
    assert!(apply_control_frame(&frame, -1, &mut Vec::new(), &mut false).is_err());
}

#[test]
fn stop_matrix() {
    // Expiry wins; live reason becomes `expired`, `exited` sticks.
    assert_eq!(
        session_should_stop(10.0, 9.0, 99.0, false, 0.0, false, "disconnected"),
        (true, "expired".to_string(), 9.0)
    );
    assert_eq!(
        session_should_stop(10.0, 99.0, 9.0, false, 0.0, false, "exited"),
        (true, "exited".to_string(), 99.0)
    );
    // Attach watchdog.
    assert_eq!(
        session_should_stop(6.0, 99.0, 99.0, true, 5.0, false, "disconnected"),
        (true, "launch_failed".to_string(), 99.0)
    );
    assert!(!session_should_stop(4.0, 99.0, 99.0, true, 5.0, false, "disconnected").0);
    // Child exit flips reason once and pulls the deadline in.
    assert_eq!(
        session_should_stop(10.0, 99.0, 99.0, false, 0.0, true, "disconnected"),
        (false, "exited".to_string(), 10.5)
    );
    assert_eq!(
        session_should_stop(10.0, 10.2, 99.0, false, 0.0, true, "exited").2,
        10.2
    );
    // Steady state.
    assert_eq!(
        session_should_stop(1.0, 99.0, 99.0, false, 0.0, false, "disconnected"),
        (false, "disconnected".to_string(), 99.0)
    );
}

#[test]
fn child_lifecycle() {
    // Exited child reports true (and stays reaped by us).
    let pid = unsafe { libc::fork() };
    assert!(pid >= 0);
    if pid == 0 {
        unsafe { libc::_exit(0) };
    }
    std::thread::sleep(std::time::Duration::from_millis(100));
    assert!(child_exited(pid).unwrap());
    let mut status = 0;
    assert_eq!(unsafe { libc::waitpid(pid, &mut status, 0) }, pid);
    // Live child reports false.
    let pid = unsafe { libc::fork() };
    assert!(pid >= 0);
    if pid == 0 {
        unsafe {
            libc::sleep(30);
            libc::_exit(0);
        }
    }
    assert!(!child_exited(pid).unwrap());
    unsafe {
        libc::kill(pid, libc::SIGKILL);
        assert_eq!(libc::waitpid(pid, &mut status, 0), pid);
    }
}

#[test]
fn end_child_terms() {
    let pid = unsafe { libc::fork() };
    assert!(pid >= 0);
    if pid == 0 {
        unsafe {
            libc::pause();
            libc::_exit(0);
        }
    }
    let mut pipe = [0; 2];
    assert_eq!(
        unsafe { libc::pipe2(pipe.as_mut_ptr(), libc::O_CLOEXEC) },
        0
    );
    unsafe {
        libc::close(pipe[1]);
    }
    let status = end_child(pid, pipe[0]).expect("reaped");
    assert!(libc::WIFSIGNALED(status));
    assert_eq!(libc::WTERMSIG(status), libc::SIGTERM);
}

#[test]
fn pty_output_lines() {
    let mut pipe = [0; 2];
    assert_eq!(
        unsafe { libc::pipe2(pipe.as_mut_ptr(), libc::O_CLOEXEC) },
        0
    );
    assert_eq!(
        unsafe { libc::write(pipe[1], b"hi".as_ptr() as *const libc::c_void, 2) },
        2
    );
    let mut outgoing = Vec::new();
    let (ended, attaching) = read_pty_output(pipe[0], &mut outgoing, true).unwrap();
    assert!(ended.is_none() && !attaching);
    assert_eq!(
        outgoing,
        b"{\"type\":\"ready\"}\n{\"type\":\"output\",\"data\":\"aGk=\"}\n"
    );
    unsafe {
        libc::close(pipe[1]);
    }
    let (ended, _) = read_pty_output(pipe[0], &mut Vec::new(), false).unwrap();
    assert_eq!(ended, Some("exited".to_string()));
    unsafe {
        libc::close(pipe[0]);
    }
}

#[test]
fn resize_needs_tty() {
    assert!(set_size(-1, 80, 24).is_err());
    // Positive path on a real pty when the sandbox provides one.
    let mut master = 0;
    let mut slave = 0;
    let mut name = [0 as libc::c_char; 64];
    let rc = unsafe {
        libc::openpty(
            &mut master,
            &mut slave,
            name.as_mut_ptr(),
            std::ptr::null_mut(),
            std::ptr::null_mut(),
        )
    };
    if rc != 0 {
        return; // no pty in this sandbox; negative case above still holds
    }
    assert!(set_size(master, 80, 24).is_ok());
    assert!(set_size(slave, 100, 30).is_ok());
    unsafe {
        libc::close(master);
        libc::close(slave);
    }
}

#[test]
fn bounds_rejected() {
    // Invalid bounds exit 1 with a `launch_failed` line on captured stdout.
    assert_eq!(run_terminal(&login(), 1, 24, 60, "/nonexistent.sock"), 1);
    assert_eq!(run_terminal(&login(), 80, 24, 0, "/nonexistent.sock"), 1);
    assert_eq!(
        run_terminal(&login(), 80, 24, 43201, "/nonexistent.sock"),
        1
    );
}

#[test]
fn env_entries_are_nul_terminated() {
    // S04-F1: every `KEY=value` entry handed to execve must be a real
    // C string (single trailing NUL, no interior NUL).
    let env: Vec<(std::ffi::CString, std::ffi::CString)> = user_environment(&login())
        .iter()
        .map(|(k, v)| (cstring(k).unwrap(), cstring(v).unwrap()))
        .collect();
    let entries = env_entries(&env);
    assert_eq!(entries.len(), env.len());
    for ((key, value), entry) in env.iter().zip(entries.iter()) {
        let bytes = entry.as_bytes_with_nul();
        assert_eq!(*bytes.last().unwrap(), 0);
        assert!(!bytes[..bytes.len() - 1].contains(&0));
        let mut expected = key.as_bytes().to_vec();
        expected.push(b'=');
        expected.extend_from_slice(value.as_bytes());
        assert_eq!(&bytes[..bytes.len() - 1], expected.as_slice());
    }
    // envp pointers derive from these same owned objects.
    let ptrs: Vec<*const libc::c_char> = entries.iter().map(|e| e.as_ptr()).collect();
    for (entry, ptr) in entries.iter().zip(ptrs.iter()) {
        assert_eq!(entry.as_ptr(), *ptr);
    }
}

#[test]
fn select_reports_writable_fds() {
    // S05-F1: queued relay writes must observe writability, not
    // exceptional conditions. Uses real descriptors (/dev/null as the
    // master stand-in, real stdout) with known queued bytes.
    let null = unsafe { libc::open(c"/dev/null".as_ptr(), libc::O_RDWR | libc::O_CLOEXEC) };
    assert!(null >= 0);
    // Queued PTY output watches stdout for writability.
    let (readable, writable) = pty_select(null, 0, 1, true, 0.0).unwrap();
    assert_eq!(writable, vec![1]);
    assert!(readable.contains(&null));
    // Queued input watches the master for writability once attached.
    let (_, writable) = pty_select(null, 1, 0, false, 0.0).unwrap();
    assert_eq!(writable, vec![null]);
    // Nothing queued watches nothing for writability.
    let (_, writable) = pty_select(null, 0, 0, true, 0.0).unwrap();
    assert!(writable.is_empty());
    unsafe {
        libc::close(null);
    }
}
