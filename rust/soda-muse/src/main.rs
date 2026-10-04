// soda-muse delegates a normal shell invocation over the launch-only socket.
use std::ffi::{CStr, CString};
use std::fs;
use std::io;
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt};
use std::os::unix::io::AsRawFd;
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";
const MUSE_NATIVE: &str = "/usr/local/libexec/soda/muse";

fn main() {
    let (code, err) = run();
    if let Some(e) = err {
        eprintln!("{e}");
    }
    std::process::exit(code);
}

fn run() -> (i32, Option<String>) {
    // Never panic on non-UTF-8 argv; Go tolerates arbitrary bytes.
    let argv: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    if let Some(done) = native_action(&argv) {
        return done;
    }
    let request = match shell_request(&argv) {
        Ok(r) => r,
        Err(e) => return (1, Some(e)),
    };
    let fd = match seqpacket_connect(MUSE_LAUNCH_SOCKET) {
        Ok(fd) => fd,
        Err(_) => return (1, Some(String::from("muse launch service unavailable"))),
    };
    launch_shell(fd, &request)
}

fn native_action(args: &[String]) -> Option<(i32, Option<String>)> {
    if args.is_empty() {
        return None;
    }
    match args[0].as_str() {
        "--soda-copy-config" => {
            if args.len() != 3 {
                return Some((1, Some(String::from("config source and view required"))));
            }
            Some((0, copy_config(&args[1], &args[2]).err()))
        }
        "--soda-exec" => {
            if args.len() < 3 {
                return Some((1, Some(String::from("execution root and cwd required"))));
            }
            let (code, err) = execute(&args[1], &args[2], &args[3..]);
            Some((code, err))
        }
        _ => metadata_action(args),
    }
}

fn metadata_action(args: &[String]) -> Option<(i32, Option<String>)> {
    match args[0].as_str() {
        "--soda-check" | "--soda-read-config" | "--soda-account" => {}
        _ => return None,
    }
    if args.len() != 2 {
        return Some((1, Some(String::from("one metadata input required"))));
    }
    let err = match args[0].as_str() {
        "--soda-check" => check_runtime(&args[1]).err(),
        "--soda-read-config" => read_config(&args[1]).err(),
        _ => account_for(&args[1]).err(),
    };
    Some((0, err))
}

struct ShellRequest {
    cwd: String,
    args: Vec<String>,
    home: String,
    connection_id: String,
    config_home: String,
    term: String,
    tty: bool,
    cols: u16,
    rows: u16,
}

fn shell_request(argv: &[String]) -> Result<ShellRequest, String> {
    let cwd = match std::env::current_dir() {
        Ok(p) => p.as_os_str().as_bytes().to_vec(),
        Err(e) => return Err(format!("getwd: {}", go_strerror(e.raw_os_error()))),
    };
    // Go carries cwd bytes through JSON, replacing invalid UTF-8.
    let cwd = String::from_utf8_lossy(&cwd).into_owned();
    let home = env_lossy("HOME");
    let connection_id = env_lossy("SODA_MUSE_CONNECTION");
    let mut config_home = env_lossy("XDG_CONFIG_HOME");
    if config_home.is_empty() {
        if home.is_empty() {
            return Err(String::from("$HOME is not defined"));
        }
        config_home = go_join(&home, ".config");
    }
    let term = env_lossy("TERM");
    let mut request = ShellRequest {
        cwd,
        args: argv.to_vec(),
        home,
        connection_id,
        config_home,
        term,
        tty: false,
        cols: 0,
        rows: 0,
    };
    let mut size: libc::winsize = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::ioctl(io::stdin().as_raw_fd(), libc::TIOCGWINSZ, &mut size) };
    if rc == 0 {
        request.tty = true;
        request.cols = size.ws_col;
        request.rows = size.ws_row;
    }
    validate_shell(&request)?;
    Ok(request)
}

fn env_lossy(name: &str) -> String {
    std::env::var_os(name)
        .map(|v| v.to_string_lossy().into_owned())
        .unwrap_or_default()
}

fn validate_shell(r: &ShellRequest) -> Result<(), String> {
    let denied = String::from("identity authority denied");
    if !launch_absolute_path(&r.cwd, false) || !r.config_paths_valid() {
        return Err(denied);
    }
    if !r.launch_sizes_valid() {
        return Err(denied);
    }
    if r.tty && (r.cols == 0 || r.rows == 0) {
        return Err(denied);
    }
    launch_arguments_valid(&r.args).map_err(|_| denied)
}

impl ShellRequest {
    fn config_paths_valid(&self) -> bool {
        launch_absolute_path(&self.config_home, true) && launch_absolute_path(&self.home, true)
    }

    fn launch_sizes_valid(&self) -> bool {
        launch_text(&self.term, 128) && self.connection_id.len() <= 128 && self.args.len() <= 256
    }
}

fn launch_absolute_path(value: &str, optional: bool) -> bool {
    if optional && value.is_empty() {
        return true;
    }
    value.starts_with('/') && launch_text(value, 4096)
}

fn launch_text(value: &str, limit: usize) -> bool {
    value.len() <= limit && !value.contains('\0')
}

fn launch_arguments_valid(args: &[String]) -> Result<(), ()> {
    let mut size = 0;
    for arg in args {
        size += arg.len();
        if !launch_text(arg, 32768) {
            return Err(());
        }
    }
    if size > 32768 {
        return Err(());
    }
    Ok(())
}

// shell_request_json emits the exact Go LaunchRequest field order for a
// shell: empty home/config_home/term omitted, argv always an array.
fn shell_request_json(r: &ShellRequest) -> String {
    let mut out = String::from("{");
    let mut first = true;
    let mut field = |out: &mut String, first: &mut bool, name: &str, value: &str| {
        if !*first {
            out.push(',');
        }
        *first = false;
        out.push_str(&json_string(name));
        out.push(':');
        out.push_str(value);
    };
    if !r.home.is_empty() {
        field(&mut out, &mut first, "home", &json_string(&r.home));
    }
    if !r.config_home.is_empty() {
        field(
            &mut out,
            &mut first,
            "config_home",
            &json_string(&r.config_home),
        );
    }
    if !r.term.is_empty() {
        field(&mut out, &mut first, "term", &json_string(&r.term));
    }
    field(
        &mut out,
        &mut first,
        "connection_id",
        &json_string(&r.connection_id),
    );
    field(&mut out, &mut first, "cwd", &json_string(&r.cwd));
    let mut args = String::from("[");
    for (i, a) in r.args.iter().enumerate() {
        if i > 0 {
            args.push(',');
        }
        args.push_str(&json_string(a));
    }
    args.push(']');
    field(&mut out, &mut first, "args", &args);
    field(
        &mut out,
        &mut first,
        "tty",
        if r.tty { "true" } else { "false" },
    );
    field(&mut out, &mut first, "cols", &r.cols.to_string());
    field(&mut out, &mut first, "rows", &r.rows.to_string());
    out.push('}');
    out
}

fn seqpacket_connect(path: &str) -> Result<std::os::unix::io::RawFd, ()> {
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
    header.msg_control_len = control.len() as _;
    unsafe {
        let cmsg = libc::CMSG_FIRSTHDR(&header);
        if cmsg.is_null() {
            return Err(io::Error::other("control message unavailable"));
        }
        (*cmsg).cmsg_level = libc::SOL_SOCKET;
        (*cmsg).cmsg_type = libc::SCM_RIGHTS;
        (*cmsg).cmsg_len =
            libc::CMSG_LEN((fds.len() * std::mem::size_of::<i32>()) as _) as _;
        std::ptr::copy_nonoverlapping(
            fds.as_ptr(),
            libc::CMSG_DATA(cmsg) as *mut i32,
            fds.len(),
        );
        header.msg_control_len = (*cmsg).cmsg_len as _;
        let n = libc::sendmsg(fd, &header, 0);
        if n < 0 || (n as usize) != data.len() {
            return Err(io::Error::last_os_error());
        }
    }
    Ok(())
}

fn launch_shell(fd: std::os::unix::io::RawFd, request: &ShellRequest) -> (i32, Option<String>) {
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
    std::thread::spawn(move || controls_loop(writer, tty));
    let mut body = [0u8; 65536];
    let r = unsafe {
        libc::recv(
            fd,
            body.as_mut_ptr() as *mut libc::c_void,
            body.len(),
            0,
        )
    };
    if r <= 0 {
        return (1, Some(String::from("muse launch service ended")));
    }
    let (code, error_text) = match parse_launch_exit(&body[..r as usize]) {
        Ok(v) => v,
        Err(()) => return (1, Some(String::from("muse launch service ended"))),
    };
    if !error_text.is_empty() {
        return (code, Some(error_text));
    }
    (code, None)
}

fn forward_signals() -> [i32; 9] {
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

fn controls_loop(fd: std::os::unix::io::RawFd, tty: bool) {
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
        let control = if sig == libc::SIGWINCH {
            if !tty {
                continue;
            }
            let mut size: libc::winsize = unsafe { std::mem::zeroed() };
            let rc = unsafe {
                libc::ioctl(io::stdin().as_raw_fd(), libc::TIOCGWINSZ, &mut size)
            };
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

// execute is fixed product code started by the native supervisor.
fn execute(root: &str, cwd: &str, args: &[String]) -> (i32, Option<String>) {
    let ccwd = match CString::new(cwd) {
        Ok(c) => c,
        Err(_) => return (1, Some(format!("chdir {cwd}: invalid argument"))),
    };
    if unsafe { libc::chdir(ccwd.as_ptr()) } != 0 {
        return (
            1,
            Some(format!("chdir {cwd}: {}", errno_str())),
        );
    }
    let root = go_clean(root);
    if !root.starts_with("/run/soda-muse/") || root.contains("..") {
        return (1, Some(String::from("invalid Muse execution root")));
    }
    if let Err(e) = await_admission(&root) {
        return (1, Some(e));
    }
    let state = match execution_state(&root) {
        Ok(s) => s,
        Err(e) => return (1, Some(e)),
    };
    let env = muse_environment(&root, &state);
    // execve only returns on failure; success replaces this process.
    let binary = CString::new(MUSE_NATIVE).unwrap();
    let mut argv: Vec<CString> = vec![CString::new("muse").unwrap()];
    for a in args {
        match CString::new(a.as_str()) {
            Ok(c) => argv.push(c),
            Err(_) => {
                return (
                    1,
                    Some(format!("muse execution failed: {}", go_strerror(Some(libc::EINVAL)))),
                );
            }
        }
    }
    let mut envp: Vec<CString> = Vec::with_capacity(env.len());
    for e in &env {
        match CString::new(e.as_str()) {
            Ok(c) => envp.push(c),
            Err(_) => {
                return (
                    1,
                    Some(format!("muse execution failed: {}", go_strerror(Some(libc::EINVAL)))),
                );
            }
        }
    }
    let argv_ptr: Vec<*const libc::c_char> =
        argv.iter().map(|c| c.as_ptr()).chain(std::iter::once(std::ptr::null())).collect();
    let env_ptr: Vec<*const libc::c_char> =
        envp.iter().map(|c| c.as_ptr()).chain(std::iter::once(std::ptr::null())).collect();
    unsafe { libc::execve(binary.as_ptr(), argv_ptr.as_ptr(), env_ptr.as_ptr()) };
    (
        1,
        Some(format!("muse execution failed: {}", errno_str())),
    )
}

fn await_admission(root: &str) -> Result<(), String> {
    let ready = format!("{root}/ready");
    let deadline = Instant::now() + Duration::from_secs(10);
    loop {
        if let Ok(info) = fs::metadata(&ready) {
            if info.is_file() {
                return Ok(());
            }
        }
        if Instant::now() >= deadline {
            return Err(String::from("muse admission did not complete"));
        }
        std::thread::sleep(Duration::from_millis(25));
    }
}

fn execution_state(root: &str) -> Result<String, String> {
    let id = go_base(root);
    if id.len() != 32 {
        return Err(String::from("invalid Muse execution ID"));
    }
    if let Some(b) = id.bytes().find(|b| !b.is_ascii_hexdigit()) {
        return Err(format!("encoding/hex: invalid byte: {}", go_quote_rune(b)));
    }
    let state = format!("/tmp/soda-muse-state-{id}");
    mkdir_mode(&state, 0o700)?;
    for name in ["state", "cache", "data", "tmp"] {
        mkdir_mode(&format!("{state}/{name}"), 0o700)?;
    }
    Ok(state)
}

fn mkdir_mode(path: &str, mode: u32) -> Result<(), String> {
    let c = match CString::new(path) {
        Ok(c) => c,
        Err(_) => return Err(format!("mkdir {path}: invalid argument")),
    };
    if unsafe { libc::mkdir(c.as_ptr(), mode) } != 0 {
        return Err(format!("mkdir {path}: {}", errno_str()));
    }
    Ok(())
}

fn muse_environment(root: &str, state: &str) -> Vec<String> {
    muse_environment_with(root, state, &|n| env_lossy_opt(n))
}

fn env_lossy_opt(name: &str) -> Option<String> {
    std::env::var_os(name).map(|v| v.to_string_lossy().into_owned())
}

fn muse_environment_with(
    root: &str,
    state: &str,
    lookup: &dyn Fn(&str) -> Option<String>,
) -> Vec<String> {
    let mut env = vec![
        String::from("PATH=/usr/local/bin:/usr/bin:/bin"),
        String::from("LANG=C.UTF-8"),
        String::from("TBH_CREDENTIAL_BACKEND=file"),
        format!("XDG_CONFIG_HOME={root}/config"),
        format!("XDG_STATE_HOME={state}/state"),
        format!("XDG_CACHE_HOME={state}/cache"),
        format!("XDG_DATA_HOME={state}/data"),
        format!("TMPDIR={state}/tmp"),
    ];
    for name in ["HOME", "USER", "LOGNAME", "TERM", "COLORTERM"] {
        if let Some(value) = lookup(name) {
            if !value.is_empty() {
                env.push(format!("{name}={value}"));
            }
        }
    }
    env
}

fn go_base(path: &str) -> &str {
    if path.is_empty() {
        return ".";
    }
    let stripped = path.trim_end_matches('/');
    if stripped.is_empty() {
        return "/";
    }
    match stripped.rfind('/') {
        Some(i) => &stripped[i + 1..],
        None => stripped,
    }
}

// go_clean mirrors filepath.Clean lexical rules.
fn go_clean(path: &str) -> String {
    if path.is_empty() {
        return String::from(".");
    }
    let rooted = path.starts_with('/');
    let mut out: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if let Some(last) = out.pop() {
                    if last == ".." {
                        out.push("..");
                        out.push("..");
                    }
                } else if !rooted {
                    out.push("..");
                }
            }
            _ => out.push(part),
        }
    }
    let mut clean = out.join("/");
    if rooted {
        clean.insert(0, '/');
    }
    if clean.is_empty() {
        clean.push('.');
    }
    clean
}

fn go_join(first: &str, second: &str) -> String {
    go_clean(&format!("{first}/{second}"))
}

// errno_str renders the current errno the way Go formats syscall errors.
fn errno_str() -> String {
    go_strerror(Some(unsafe { *libc::__errno_location() }))
}

fn go_strerror(errno: Option<i32>) -> String {
    let no = errno.unwrap_or(0);
    let text = unsafe {
        let ptr = libc::strerror(no);
        if ptr.is_null() {
            return format!("errno {no}");
        }
        CStr::from_ptr(ptr).to_string_lossy().into_owned()
    };
    // Go spells errno text lowercase; glibc capitalizes the first letter.
    let mut chars = text.chars();
    match chars.next() {
        Some(c) => c.to_lowercase().collect::<String>() + chars.as_str(),
        None => text,
    }
}

// go_quote_rune mirrors strconv.QuoteRune for the invalid-hex-byte error.
fn go_quote_rune(b: u8) -> String {
    match b {
        0x20..=0x7e if b != b'\'' && b != b'\\' => format!("'{}'", b as char),
        b'\'' => String::from("'\\''"),
        b'\\' => String::from("'\\\\'"),
        0x00..=0x7f => format!("'\\x{b:02x}'"),
        _ => format!("'\\u{b:04x}'"),
    }
}

fn account_for(actor: &str) -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("project root required"));
    }
    let parsed: i64 = actor.parse().map_err(|_| String::from("invalid account identity"))?;
    if parsed <= 0 {
        return Err(String::from("invalid account identity"));
    }
    let directory = "/var/lib/soda/accounts";
    for path in ["/var", "/var/lib", "/var/lib/soda", directory] {
        account_node(path, true)?;
    }
    let mut names: Vec<String> = Vec::new();
    let entries = fs::read_dir(directory).map_err(|e| path_error("open", directory, e))?;
    for entry in entries {
        let entry = entry.map_err(|e| path_error("open", directory, e))?;
        names.push(entry.file_name().to_string_lossy().into_owned());
    }
    names.sort();
    for name in &names {
        if account_entry(directory, name, actor)? {
            return Ok(());
        }
    }
    Err(String::from("provisioned account missing"))
}

fn path_error(op: &str, path: &str, e: io::Error) -> String {
    format!("{op} {path}: {}", go_strerror(e.raw_os_error()))
}

fn account_entry(directory: &str, name: &str, actor: &str) -> Result<bool, String> {
    let marker = format!("{directory}/{name}");
    if account_node(&marker, false).is_err() {
        return Ok(false);
    }
    let body = fs::read(&marker).map_err(|e| path_error("open", &marker, e))?;
    if String::from_utf8_lossy(&body).trim() != actor {
        return Ok(false);
    }
    let (uid, gid, gecos, dir) = lookup_user(name)?;
    let mut out = String::from("{\"Uid\":");
    out.push_str(&json_string(&uid.to_string()));
    out.push_str(",\"Gid\":");
    out.push_str(&json_string(&gid.to_string()));
    out.push_str(",\"Username\":");
    out.push_str(&json_string(name));
    out.push_str(",\"Name\":");
    out.push_str(&json_string(&gecos));
    out.push_str(",\"HomeDir\":");
    out.push_str(&json_string(&dir));
    out.push_str("}\n");
    print!("{out}");
    use std::io::Write;
    io::stdout().flush().map_err(|e| e.to_string())?;
    Ok(true)
}

fn account_node(path: &str, directory: bool) -> Result<(), String> {
    let info = fs::symlink_metadata(path).map_err(|e| path_error("lstat", path, e))?;
    if info.uid() != 0 || info.gid() != 0 || info.mode() & 0o022 != 0 {
        return Err(String::from("unsafe account record"));
    }
    if directory {
        if !info.is_dir() {
            return Err(String::from("account ancestor is not a directory"));
        }
        return Ok(());
    }
    if !info.is_file() || info.mode() & 0o777 != 0o600 || info.len() > 64 {
        return Err(String::from("unsafe account marker"));
    }
    Ok(())
}

fn lookup_user(name: &str) -> Result<(u32, u32, String, String), String> {
    let cname = CString::new(name).map_err(|_| format!("user: unknown user {name}"))?;
    let mut pwd: libc::passwd = unsafe { std::mem::zeroed() };
    let mut buf = [0u8; 16384];
    let mut result: *mut libc::passwd = std::ptr::null_mut();
    let rc = unsafe {
        libc::getpwnam_r(
            cname.as_ptr(),
            &mut pwd,
            buf.as_mut_ptr() as *mut libc::c_char,
            buf.len(),
            &mut result,
        )
    };
    if rc != 0 || result.is_null() {
        return Err(format!("user: unknown user {name}"));
    }
    let gecos = unsafe { CStr::from_ptr(pwd.pw_gecos) }.to_string_lossy().into_owned();
    let dir = unsafe { CStr::from_ptr(pwd.pw_dir) }.to_string_lossy().into_owned();
    Ok((pwd.pw_uid, pwd.pw_gid, gecos, dir))
}

fn check_runtime(version: &str) -> Result<(), String> {
    let private = make_private_dir("soda-muse-check-")?;
    struct Cleanup(String);
    impl Drop for Cleanup {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }
    let _cleanup = Cleanup(private.clone());
    let output = run_pristine(
        MUSE_NATIVE,
        &["--version".to_string()],
        &private,
        Duration::from_secs(5),
    )
    .map_err(|e| format!("muse native runtime prerequisite failed: {e}"))?;
    if output.trim() != format!("Muse Code 1.4.0 ({version})") {
        return Err(format!("muse version differs from pinned {version}"));
    }
    Ok(())
}

fn make_private_dir(prefix: &str) -> Result<String, String> {
    let pid = unsafe { libc::getpid() };
    for n in 0..1000 {
        let path = if n == 0 {
            format!("/tmp/{prefix}{pid}")
        } else {
            format!("/tmp/{prefix}{pid}-{n}")
        };
        let c = CString::new(path.clone()).unwrap();
        if unsafe { libc::mkdir(c.as_ptr(), 0o700) } == 0 {
            return Ok(path);
        }
        let e = io::Error::last_os_error();
        if e.kind() != io::ErrorKind::AlreadyExists {
            return Err(path_error("mkdir", "/tmp", e));
        }
    }
    Err(String::from("temporary directory unavailable"))
}

// run_pristine runs the native binary with the fixed clean environment,
// killing it after the deadline the way Go's CommandContext does.
fn run_pristine(
    program: &str,
    args: &[String],
    root: &str,
    timeout: Duration,
) -> Result<String, String> {
    let mut child = Command::new(program)
        .args(args)
        .env_clear()
        .env("PATH", "/usr/local/bin:/usr/bin:/bin")
        .env("LANG", "C.UTF-8")
        .env("HOME", root)
        .env("XDG_CONFIG_HOME", format!("{root}/config"))
        .env("XDG_STATE_HOME", format!("{root}/state"))
        .env("XDG_CACHE_HOME", format!("{root}/cache"))
        .env("XDG_DATA_HOME", format!("{root}/data"))
        .env("TMPDIR", root)
        .env("TBH_CREDENTIAL_BACKEND", "file")
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| format!("fork/exec {program}: {}", go_strerror(e.raw_os_error())))?;
    let deadline = Instant::now() + timeout;
    loop {
        match child.try_wait().map_err(|e| e.to_string())? {
            Some(status) => {
                let mut stdout = Vec::new();
                if let Some(mut out) = child.stdout.take() {
                    use std::io::Read;
                    let _ = out.read_to_end(&mut stdout);
                }
                if !status.success() {
                    if let Some(code) = status.code() {
                        return Err(format!("exit status {code}"));
                    }
                    return Err(String::from("signal: killed"));
                }
                return Ok(String::from_utf8_lossy(&stdout).into_owned());
            }
            None => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(String::from("signal: killed"));
                }
                std::thread::sleep(Duration::from_millis(10));
            }
        }
    }
}

// copy_config always runs as the provisioned account, including symlink reads.
// Native provider credentials are excluded; custody supplies the single auth file.
fn copy_config(source: &str, destination: &str) -> Result<(), String> {
    if !source.starts_with('/') || !destination.starts_with('/') {
        return Err(String::from("absolute config paths required"));
    }
    match fs::metadata(source) {
        Ok(_) => {}
        Err(e) if e.kind() == io::ErrorKind::NotFound => return Ok(()),
        Err(_) => {}
    }
    copy_config_walk(source, destination, source)
}

fn copy_config_walk(source: &str, destination: &str, path: &str) -> Result<(), String> {
    let info = fs::symlink_metadata(path).map_err(|e| path_error("lstat", path, e))?;
    let name = go_base(path);
    let relative = if path == source {
        String::from(".")
    } else {
        match path.strip_prefix(&format!("{source}/")) {
            Some(r) => r.to_string(),
            None => return Err(path_error("lstat", path, io::Error::from(io::ErrorKind::NotFound))),
        }
    };
    if name == "auth.json" {
        // Mirror Go: the entry itself is skipped, but a directory by that
        // name is still descended into (its children then fail to stage).
        if info.file_type().is_dir() {
            return copy_config_children(source, destination, path);
        }
        return Ok(());
    }
    let target = go_join(destination, &relative);
    if info.file_type().is_dir() {
        fs::DirBuilder::new()
            .recursive(true)
            .mode(0o700)
            .create(&target)
            .map_err(|e| path_error("mkdir", &target, e))?;
        return copy_config_children(source, destination, path);
    }
    copy_config_file(path, &target)
}

fn copy_config_children(source: &str, destination: &str, dir: &str) -> Result<(), String> {
    let mut names: Vec<String> = Vec::new();
    let entries = fs::read_dir(dir).map_err(|e| path_error("open", dir, e))?;
    for entry in entries {
        let entry = entry.map_err(|e| path_error("open", dir, e))?;
        names.push(entry.file_name().to_string_lossy().into_owned());
    }
    names.sort();
    for name in &names {
        copy_config_walk(source, destination, &format!("{dir}/{name}"))?;
    }
    Ok(())
}

fn copy_config_file(path: &str, target: &str) -> Result<(), String> {
    let info = fs::metadata(path).map_err(|e| path_error("stat", path, e))?;
    if !info.is_file() {
        return Ok(());
    }
    if info.len() > 1 << 20 {
        return Err(String::from("muse config file exceeds private view limit"));
    }
    let body = fs::read(path).map_err(|e| path_error("open", path, e))?;
    let mut opts = fs::OpenOptions::new();
    opts.write(true).create(true).truncate(true);
    opts.mode(0o600);
    use std::io::Write;
    let mut f = opts.open(target).map_err(|e| path_error("open", target, e))?;
    f.write_all(&body).map_err(|e| e.to_string())?;
    Ok(())
}

// read_config emits the upstream release's saved settings and trust records only.
// It executes as the invoking account, never as a privileged config reader.
fn read_config(source: &str) -> Result<(), String> {
    if !source.starts_with('/') {
        return Err(String::from("absolute config path required"));
    }
    let mut view: Vec<(&str, Vec<u8>)> = Vec::new();
    for name in ["settings.json", "trust.json"] {
        let path = format!("{source}/{name}");
        let mut file = match fs::File::open(&path) {
            Ok(f) => f,
            Err(e) if e.kind() == io::ErrorKind::NotFound => continue,
            Err(e) => return Err(path_error("open", &path, e)),
        };
        use std::io::Read;
        let mut body = Vec::new();
        file.take((1 << 20) + 1)
            .read_to_end(&mut body)
            .map_err(|e| e.to_string())?;
        if body.len() > 1 << 20 {
            return Err(String::from("muse config file exceeds private view limit"));
        }
        view.push((name, body));
    }
    let mut out = String::from("{");
    for (i, (name, body)) in view.iter().enumerate() {
        if i > 0 {
            out.push(',');
        }
        out.push_str(&json_string(name));
        out.push(':');
        out.push_str(&json_string(&base64_encode(body)));
    }
    out.push_str("}\n");
    print!("{out}");
    use std::io::Write;
    io::stdout().flush().map_err(|e| e.to_string())?;
    Ok(())
}

fn base64_encode(data: &[u8]) -> String {
    const ALPHABET: &[u8; 64] =
        b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(data.len().div_ceil(3) * 4);
    for chunk in data.chunks(3) {
        let mut n: u32 = 0;
        for (i, b) in chunk.iter().enumerate() {
            n |= (*b as u32) << (16 - 8 * i);
        }
        out.push(ALPHABET[(n >> 18) as usize & 63] as char);
        out.push(ALPHABET[(n >> 12) as usize & 63] as char);
        if chunk.len() > 1 {
            out.push(ALPHABET[(n >> 6) as usize & 63] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(ALPHABET[n as usize & 63] as char);
        } else {
            out.push('=');
        }
    }
    out
}
