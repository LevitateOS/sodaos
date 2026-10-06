use super::launch_json::parse_launch_exit;
use super::shell::shell_request_json;
use super::ShellRequest;
use std::io;
use std::os::unix::io::AsRawFd;

pub(crate) fn seqpacket_connect(path: &str) -> Result<std::os::unix::io::RawFd, ()> {
    let fd = unsafe { libc::socket(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0) };
    if fd < 0 {
        return Err(());
    }
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as libc::sa_family_t;
    let bytes = path.as_bytes();
    if bytes.len() >= addr.sun_path.len() {
        unsafe { libc::close(fd) };
        return Err(());
    }
    for (i, b) in bytes.iter().enumerate() {
        addr.sun_path[i] = *b as libc::c_char;
    }
    let len = (std::mem::size_of::<libc::sa_family_t>() + bytes.len() + 1) as libc::socklen_t;
    let rc = unsafe {
        libc::connect(
            fd,
            &addr as *const libc::sockaddr_un as *const libc::sockaddr,
            len,
        )
    };
    if rc != 0 {
        unsafe { libc::close(fd) };
        return Err(());
    }
    Ok(fd)
}

fn send_with_fds(fd: std::os::unix::io::RawFd, data: &[u8], fds: &[i32]) -> io::Result<()> {
    let mut iov = libc::iovec {
        iov_base: data.as_ptr() as *mut libc::c_void,
        iov_len: data.len(),
    };
    let mut control = [0u8; 64];
    let mut header: libc::msghdr = unsafe { std::mem::zeroed() };
    header.msg_iov = &mut iov;
    header.msg_iovlen = 1;
    header.msg_control = control.as_mut_ptr() as *mut libc::c_void;
    header.msg_controllen = control.len() as _;
    unsafe {
        let cmsg = libc::CMSG_FIRSTHDR(&header as *const libc::msghdr);
        if cmsg.is_null() {
            return Err(io::Error::other("control message unavailable"));
        }
        (*cmsg).cmsg_level = libc::SOL_SOCKET;
        (*cmsg).cmsg_type = libc::SCM_RIGHTS;
        (*cmsg).cmsg_len = libc::CMSG_LEN(std::mem::size_of_val(fds) as _) as _;
        std::ptr::copy_nonoverlapping(fds.as_ptr(), libc::CMSG_DATA(cmsg) as *mut i32, fds.len());
        header.msg_controllen = (*cmsg).cmsg_len as _;
        let n = libc::sendmsg(fd, &header as *const libc::msghdr, 0);
        if n < 0 || (n as usize) != data.len() {
            return Err(io::Error::last_os_error());
        }
    }
    Ok(())
}

pub(crate) fn launch_shell(
    fd: std::os::unix::io::RawFd,
    request: &ShellRequest,
) -> (i32, Option<String>) {
    struct Guard(i32);
    impl Drop for Guard {
        fn drop(&mut self) {
            unsafe { libc::close(self.0) };
        }
    }
    let _guard = Guard(fd);
    let data = shell_request_json(request);
    if send_with_fds(fd, data.as_bytes(), &[0, 1, 2]).is_err() {
        return (1, Some(String::from("muse launch failed")));
    }
    let writer = unsafe { libc::dup(fd) };
    if writer < 0 {
        return (1, Some(String::from("muse launch failed")));
    }
    // Block the forwarded signals process-wide so the control thread owns
    // them, mirroring Go signal.Notify delivery without default actions.
    let mut set: libc::sigset_t = unsafe { std::mem::zeroed() };
    unsafe {
        libc::sigemptyset(&mut set);
        for sig in forward_signals() {
            libc::sigaddset(&mut set, sig);
        }
        libc::pthread_sigmask(libc::SIG_BLOCK, &set, std::ptr::null_mut());
    }
    let tty = request.tty;
    static NEVER: std::sync::atomic::AtomicBool = std::sync::atomic::AtomicBool::new(false);
    std::thread::spawn(move || controls_loop(writer, tty, &NEVER));
    let mut body = [0u8; 65536];
    let r = unsafe { libc::recv(fd, body.as_mut_ptr() as *mut libc::c_void, body.len(), 0) };
    if r <= 0 {
        return (1, Some(String::from("muse launch service ended")));
    }
    let (code, error_text) = match parse_launch_exit(&body[..r as usize]) {
        Ok(v) => v,
        Err(()) => return (1, Some(String::from("muse launch service ended"))),
    };
    let code = code as i32;
    if !error_text.is_empty() {
        return (code, Some(error_text));
    }
    (code, None)
}

pub(crate) fn forward_signals() -> [i32; 9] {
    [
        libc::SIGWINCH,
        libc::SIGINT,
        libc::SIGTERM,
        libc::SIGHUP,
        libc::SIGQUIT,
        libc::SIGTSTP,
        libc::SIGCONT,
        libc::SIGUSR1,
        libc::SIGUSR2,
    ]
}

pub(crate) fn controls_loop(
    fd: std::os::unix::io::RawFd,
    tty: bool,
    stop: &std::sync::atomic::AtomicBool,
) {
    struct Guard(i32);
    impl Drop for Guard {
        fn drop(&mut self) {
            unsafe { libc::close(self.0) };
        }
    }
    let _guard = Guard(fd);
    let mut set: libc::sigset_t = unsafe { std::mem::zeroed() };
    unsafe {
        libc::sigemptyset(&mut set);
        for sig in forward_signals() {
            libc::sigaddset(&mut set, sig);
        }
    }
    loop {
        let mut sig = 0;
        let rc = unsafe { libc::sigwait(&set, &mut sig) };
        if rc != 0 {
            return;
        }
        if stop.load(std::sync::atomic::Ordering::SeqCst) {
            return;
        }
        let control = if sig == libc::SIGWINCH {
            if !tty {
                continue;
            }
            let mut size: libc::winsize = unsafe { std::mem::zeroed() };
            let rc = unsafe { libc::ioctl(io::stdin().as_raw_fd(), libc::TIOCGWINSZ, &mut size) };
            if rc != 0 {
                continue;
            }
            format!("{{\"cols\":{},\"rows\":{}}}\n", size.ws_col, size.ws_row)
        } else {
            format!("{{\"signal\":{sig}}}\n")
        };
        let bytes = control.as_bytes();
        let n = unsafe { libc::send(fd, bytes.as_ptr() as *const libc::c_void, bytes.len(), 0) };
        if n < 0 || (n as usize) != bytes.len() {
            return;
        }
    }
}
