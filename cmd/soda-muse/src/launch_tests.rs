use super::launch::{controls_loop, forward_signals, launch_shell};
use super::ShellRequest;

#[test]
fn launch_shell_passes_fds_and_maps_exit() {
    for (reply, want_code, want_err) in [
        ("{\"code\":0}", 0, None),
        ("{\"code\":3}", 3, None),
        ("{\"code\":1,\"error\":\"denied\"}", 1, Some("denied")),
    ] {
        let mut pair = [0; 2];
        assert_eq!(
            unsafe { libc::socketpair(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0, pair.as_mut_ptr()) },
            0
        );
        let server = std::thread::spawn(move || {
            let fd = pair[1];
            struct Guard(i32);
            impl Drop for Guard {
                fn drop(&mut self) {
                    unsafe { libc::close(self.0) };
                }
            }
            let _guard = Guard(fd);
            // Read the request plus exactly three passed fds.
            let mut data = [0u8; 65536];
            let mut iov = libc::iovec {
                iov_base: data.as_mut_ptr() as *mut libc::c_void,
                iov_len: data.len(),
            };
            let mut control = [0u8; 64];
            let mut header: libc::msghdr = unsafe { std::mem::zeroed() };
            header.msg_iov = &mut iov;
            header.msg_iovlen = 1;
            header.msg_control = control.as_mut_ptr() as *mut libc::c_void;
            header.msg_controllen = control.len() as _;
            let n = unsafe { libc::recvmsg(fd, &mut header, 0) };
            assert!(n > 0);
            let body = &data[..n as usize];
            let text = String::from_utf8_lossy(body);
            assert!(text.starts_with("{\"config_home\":\"/c\","), "{text}");
            assert!(text.contains("\"cwd\":\"/w\""), "{text}");
            let cmsg = unsafe { libc::CMSG_FIRSTHDR(&header as *const libc::msghdr) };
            assert!(!cmsg.is_null());
            let got = unsafe {
                let len = (*cmsg).cmsg_len as usize - libc::CMSG_LEN(0) as usize;
                std::slice::from_raw_parts(
                    libc::CMSG_DATA(cmsg) as *const i32,
                    len / std::mem::size_of::<i32>(),
                )
                .to_vec()
            };
            assert_eq!(got.len(), 3);
            for received in &got {
                unsafe { libc::close(*received) };
            }
            let bytes = reply.as_bytes();
            let w =
                unsafe { libc::send(fd, bytes.as_ptr() as *const libc::c_void, bytes.len(), 0) };
            assert_eq!(w as usize, bytes.len());
        });
        let request = ShellRequest {
            cwd: String::from("/w"),
            args: Vec::new(),
            home: String::new(),
            connection_id: String::new(),
            config_home: String::from("/c"),
            term: String::new(),
            tty: false,
            cols: 0,
            rows: 0,
        };
        // launch_shell blocks signals and spawns its control thread.
        let (code, err) = launch_shell(pair[0], &request);
        unsafe { libc::close(pair[0]) };
        server.join().unwrap();
        assert_eq!(code, want_code);
        assert_eq!(err.as_deref(), want_err);
        // Restore the process mask the launch path blocked.
        let mut set: libc::sigset_t = unsafe { std::mem::zeroed() };
        unsafe {
            libc::sigemptyset(&mut set);
            for sig in forward_signals() {
                libc::sigaddset(&mut set, sig);
            }
            libc::pthread_sigmask(libc::SIG_UNBLOCK, &set, std::ptr::null_mut());
        }
    }
}

#[test]
fn controls_forward_signal_number() {
    let mut pair = [0; 2];
    assert_eq!(
        unsafe { libc::socketpair(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0, pair.as_mut_ptr()) },
        0
    );
    // Block in this thread so the control thread inherits the mask.
    let mut set: libc::sigset_t = unsafe { std::mem::zeroed() };
    unsafe {
        libc::sigemptyset(&mut set);
        for sig in forward_signals() {
            libc::sigaddset(&mut set, sig);
        }
        libc::pthread_sigmask(libc::SIG_BLOCK, &set, std::ptr::null_mut());
    }
    let (tx, rx) = std::sync::mpsc::channel();
    let stop = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
    let stop_child = stop.clone();
    let handle = std::thread::spawn(move || {
        let tid = unsafe { libc::pthread_self() };
        tx.send(tid).unwrap();
        controls_loop(pair[1], false, &stop_child);
    });
    let tid = rx.recv().unwrap();
    // Deterministic thread-directed delivery, not a process broadcast.
    assert_eq!(unsafe { libc::pthread_kill(tid, libc::SIGUSR2) }, 0);
    let tv = libc::timeval {
        tv_sec: 5,
        tv_usec: 0,
    };
    unsafe {
        libc::setsockopt(
            pair[0],
            libc::SOL_SOCKET,
            libc::SO_RCVTIMEO,
            &tv as *const libc::timeval as *const libc::c_void,
            std::mem::size_of::<libc::timeval>() as libc::socklen_t,
        );
    }
    let mut body = [0u8; 64];
    let r = unsafe {
        libc::recv(
            pair[0],
            body.as_mut_ptr() as *mut libc::c_void,
            body.len(),
            0,
        )
    };
    assert_eq!(&body[..r as usize], b"{\"signal\":12}\n");
    stop.store(true, std::sync::atomic::Ordering::SeqCst);
    assert_eq!(unsafe { libc::pthread_kill(tid, libc::SIGWINCH) }, 0);
    handle.join().unwrap();
    unsafe {
        libc::close(pair[0]);
        libc::pthread_sigmask(libc::SIG_UNBLOCK, &set, std::ptr::null_mut());
    }
}
