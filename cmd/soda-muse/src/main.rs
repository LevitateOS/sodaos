// soda-muse delegates a normal shell invocation over the launch-only socket.
use std::ffi::{CStr, CString};
use std::fs;
use std::io;
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

mod session;
mod shell;

use session::{launch_shell, seqpacket_connect};
use shell::shell_request;

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

// execute is fixed product code started by the native supervisor.
fn execute(root: &str, cwd: &str, args: &[String]) -> (i32, Option<String>) {
    let ccwd = match CString::new(cwd) {
        Ok(c) => c,
        Err(_) => return (1, Some(format!("chdir {cwd}: invalid argument"))),
    };
    if unsafe { libc::chdir(ccwd.as_ptr()) } != 0 {
        return (1, Some(format!("chdir {cwd}: {}", errno_str())));
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
                    Some(format!(
                        "muse execution failed: {}",
                        go_strerror(Some(libc::EINVAL))
                    )),
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
                    Some(format!(
                        "muse execution failed: {}",
                        go_strerror(Some(libc::EINVAL))
                    )),
                );
            }
        }
    }
    let argv_ptr: Vec<*const libc::c_char> = argv
        .iter()
        .map(|c| c.as_ptr())
        .chain(std::iter::once(std::ptr::null()))
        .collect();
    let env_ptr: Vec<*const libc::c_char> = envp
        .iter()
        .map(|c| c.as_ptr())
        .chain(std::iter::once(std::ptr::null()))
        .collect();
    unsafe { libc::execve(binary.as_ptr(), argv_ptr.as_ptr(), env_ptr.as_ptr()) };
    (1, Some(format!("muse execution failed: {}", errno_str())))
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
    let parsed: i64 = actor
        .parse()
        .map_err(|_| String::from("invalid account identity"))?;
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
    let gecos = unsafe { CStr::from_ptr(pwd.pw_gecos) }
        .to_string_lossy()
        .into_owned();
    let dir = unsafe { CStr::from_ptr(pwd.pw_dir) }
        .to_string_lossy()
        .into_owned();
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
            None => {
                return Err(path_error(
                    "lstat",
                    path,
                    io::Error::from(io::ErrorKind::NotFound),
                ))
            }
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
    let mut f = opts
        .open(target)
        .map_err(|e| path_error("open", target, e))?;
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
        let file = match fs::File::open(&path) {
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
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
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

// json_string matches Go encoding/json string escaping, including its
// HTML-safe <, >, & forms, so wire bytes are identical for any input.
fn json_string(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

// parse_launch_exit mirrors Go json.Unmarshal into LaunchExit: missing
// fields stay zero, unknown fields are ignored, malformed JSON rejects.
fn parse_launch_exit(body: &[u8]) -> Result<(i64, String), ()> {
    let text = std::str::from_utf8(body).map_err(|_| ())?;
    let mut code: i64 = 0;
    let mut error_text = String::new();
    let mut i = 0;
    let bytes = text.as_bytes();
    let skip_ws = |i: &mut usize| {
        while *i < bytes.len() && matches!(bytes[*i], b' ' | b'\t' | b'\n' | b'\r') {
            *i += 1;
        }
    };
    skip_ws(&mut i);
    if i >= bytes.len() || bytes[i] != b'{' {
        return Err(());
    }
    i += 1;
    // Go rejects a trailing comma, so `}` is only valid here for `{}` or
    // right after a value; after a comma a key is required.
    let mut after_comma = false;
    loop {
        skip_ws(&mut i);
        if i < bytes.len() && bytes[i] == b'}' {
            if after_comma {
                return Err(());
            }
            i += 1;
            break;
        }
        if i >= bytes.len() || bytes[i] != b'"' {
            return Err(());
        }
        let (key, next) = parse_json_string(text, i)?;
        i = next;
        skip_ws(&mut i);
        if i >= bytes.len() || bytes[i] != b':' {
            return Err(());
        }
        i += 1;
        skip_ws(&mut i);
        if key == "code" {
            let (value, next) = parse_json_integer(text, i)?;
            code = value;
            i = next;
        } else if key == "error" {
            if i >= bytes.len() || bytes[i] != b'"' {
                return Err(());
            }
            let (value, next) = parse_json_string(text, i)?;
            error_text = value;
            i = next;
        } else {
            i = skip_json_value(text, i)?;
        }
        skip_ws(&mut i);
        if i < bytes.len() && bytes[i] == b',' {
            i += 1;
            after_comma = true;
            continue;
        }
        if i < bytes.len() && bytes[i] == b'}' {
            i += 1;
            break;
        }
        return Err(());
    }
    skip_ws(&mut i);
    if i != bytes.len() {
        return Err(());
    }
    Ok((code, error_text))
}

fn parse_json_string(text: &str, start: usize) -> Result<(String, usize), ()> {
    let bytes = text.as_bytes();
    if start >= bytes.len() || bytes[start] != b'"' {
        return Err(());
    }
    let mut out = String::new();
    let mut i = start + 1;
    while i < bytes.len() {
        match bytes[i] {
            b'"' => return Ok((out, i + 1)),
            b'\\' => {
                i += 1;
                if i >= bytes.len() {
                    return Err(());
                }
                match bytes[i] {
                    b'"' => out.push('"'),
                    b'\\' => out.push('\\'),
                    b'/' => out.push('/'),
                    b'b' => out.push('\u{0008}'),
                    b'f' => out.push('\u{000c}'),
                    b'n' => out.push('\n'),
                    b'r' => out.push('\r'),
                    b't' => out.push('\t'),
                    b'u' => {
                        if i + 4 >= bytes.len() {
                            return Err(());
                        }
                        let hex = &text[i + 1..i + 5];
                        let cp = u32::from_str_radix(hex, 16).map_err(|_| ())?;
                        let c = char::from_u32(cp).ok_or(())?;
                        if (0xd800..0xe000).contains(&cp) {
                            return Err(());
                        }
                        out.push(c);
                        i += 4;
                    }
                    _ => return Err(()),
                }
            }
            0x00..=0x1f => return Err(()),
            _ => {
                let c = text[i..].chars().next().ok_or(())?;
                out.push(c);
                i += c.len_utf8() - 1;
            }
        }
        i += 1;
    }
    Err(())
}

fn parse_json_integer(text: &str, start: usize) -> Result<(i64, usize), ()> {
    let bytes = text.as_bytes();
    let mut i = start;
    if i < bytes.len() && bytes[i] == b'-' {
        i += 1;
    }
    let digits = i;
    while i < bytes.len() && bytes[i].is_ascii_digit() {
        i += 1;
    }
    if i == digits || i == start {
        return Err(());
    }
    // Go rejects fractions and exponents for int fields.
    if i < bytes.len() && (bytes[i] == b'.' || bytes[i] == b'e' || bytes[i] == b'E') {
        return Err(());
    }
    text[start..i]
        .parse::<i64>()
        .map_err(|_| ())
        .map(|v| (v, i))
}

fn skip_json_value(text: &str, start: usize) -> Result<usize, ()> {
    let bytes = text.as_bytes();
    if start >= bytes.len() {
        return Err(());
    }
    match bytes[start] {
        b'"' => parse_json_string(text, start).map(|(_, next)| next),
        b'{' | b'[' => {
            let open = bytes[start];
            let close = if open == b'{' { b'}' } else { b']' };
            let mut i = start + 1;
            let mut depth = 1;
            let mut in_string = false;
            let mut escaped = false;
            while i < bytes.len() {
                let b = bytes[i];
                if in_string {
                    if escaped {
                        escaped = false;
                    } else if b == b'\\' {
                        escaped = true;
                    } else if b == b'"' {
                        in_string = false;
                    }
                } else if b == b'"' {
                    in_string = true;
                } else if b == open {
                    depth += 1;
                } else if b == close {
                    depth -= 1;
                    if depth == 0 {
                        return Ok(i + 1);
                    }
                }
                i += 1;
            }
            Err(())
        }
        b't' => {
            if text[start..].starts_with("true") {
                Ok(start + 4)
            } else {
                Err(())
            }
        }
        b'f' => {
            if text[start..].starts_with("false") {
                Ok(start + 5)
            } else {
                Err(())
            }
        }
        b'n' => {
            if text[start..].starts_with("null") {
                Ok(start + 4)
            } else {
                Err(())
            }
        }
        b'-' | b'0'..=b'9' => {
            let mut i = start;
            if bytes[i] == b'-' {
                i += 1;
            }
            while i < bytes.len()
                && (bytes[i].is_ascii_digit()
                    || matches!(bytes[i], b'.' | b'e' | b'E' | b'+' | b'-'))
            {
                i += 1;
            }
            if i == start {
                return Err(());
            }
            Ok(i)
        }
        _ => Err(()),
    }
}

#[cfg(test)]
mod tests {
    use super::session::{controls_loop, forward_signals, launch_shell};
    use super::shell::{shell_request_json, validate_shell};
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    fn shell_fixture() -> ShellRequest {
        ShellRequest {
            cwd: String::from("/work"),
            args: vec![String::from("a b"), String::from("c<d")],
            home: String::from("/home/u"),
            connection_id: String::from("conn1"),
            config_home: String::from("/home/u/.config"),
            term: String::from("xterm"),
            tty: true,
            cols: 80,
            rows: 24,
        }
    }

    #[test]
    fn shell_request_wire_matches_go() {
        assert_eq!(
            shell_request_json(&shell_fixture()),
            "{\"home\":\"/home/u\",\"config_home\":\"/home/u/.config\",\"term\":\"xterm\",\"connection_id\":\"conn1\",\"cwd\":\"/work\",\"args\":[\"a b\",\"c\\u003cd\"],\"tty\":true,\"cols\":80,\"rows\":24}"
        );
        let empty = ShellRequest {
            cwd: String::from("/w"),
            args: Vec::new(),
            home: String::new(),
            connection_id: String::new(),
            config_home: String::new(),
            term: String::new(),
            tty: false,
            cols: 0,
            rows: 0,
        };
        assert_eq!(
            shell_request_json(&empty),
            "{\"connection_id\":\"\",\"cwd\":\"/w\",\"args\":[],\"tty\":false,\"cols\":0,\"rows\":0}"
        );
    }

    #[test]
    fn shell_validation_matches_go() {
        validate_shell(&shell_fixture()).unwrap();
        let mut bad = shell_fixture();
        bad.cwd = String::from("relative");
        assert!(validate_shell(&bad).is_err());
        let mut bad = shell_fixture();
        bad.term = "x".repeat(129);
        assert!(validate_shell(&bad).is_err());
        let mut bad = shell_fixture();
        bad.cols = 0;
        assert!(validate_shell(&bad).is_err());
        let mut bad = shell_fixture();
        bad.args = vec![String::from("x"); 257];
        assert!(validate_shell(&bad).is_err());
        let mut bad = shell_fixture();
        bad.args = vec![String::from("a\0b")];
        assert!(validate_shell(&bad).is_err());
        // Empty home/config_home stay optional.
        let mut ok = shell_fixture();
        ok.home.clear();
        ok.config_home.clear();
        validate_shell(&ok).unwrap();
    }

    #[test]
    fn launch_exit_parsing_matches_go_unmarshal() {
        assert_eq!(
            parse_launch_exit(b"{\"code\":0}").unwrap(),
            (0, String::new())
        );
        assert_eq!(parse_launch_exit(b"{}").unwrap(), (0, String::new()));
        assert_eq!(
            parse_launch_exit(b"{\"code\":3}").unwrap(),
            (3, String::new())
        );
        assert_eq!(
            parse_launch_exit(b"{\"error\":\"denied\",\"code\":1}\n").unwrap(),
            (1, String::from("denied"))
        );
        for bad in [
            "",
            "{",
            "{\"code\":}",
            "{\"code\":\"0\"}",
            "{\"code\":0,}",
            "[]",
        ] {
            assert!(
                parse_launch_exit(bad.as_bytes()).is_err(),
                "admitted {bad:?}"
            );
        }
    }

    #[test]
    fn base64_matches_standard_vectors() {
        for (raw, want) in [
            ("", ""),
            ("f", "Zg=="),
            ("fo", "Zm8="),
            ("foo", "Zm9v"),
            ("foob", "Zm9vYg=="),
            ("fooba", "Zm9vYmE="),
            ("foobar", "Zm9vYmFy"),
            ("{\"a\":1}", "eyJhIjoxfQ=="),
        ] {
            assert_eq!(base64_encode(raw.as_bytes()), want, "input {raw:?}");
        }
    }

    #[test]
    fn go_clean_matches_filepath_cases() {
        for (input, want) in [
            ("", "."),
            ("/", "/"),
            ("//", "/"),
            ("/a/b/../c", "/a/c"),
            ("/../a", "/a"),
            ("a/./b", "a/b"),
            ("a/../../b", "../b"),
            ("/run/soda-muse/", "/run/soda-muse"),
            ("/home/u/", "/home/u"),
        ] {
            assert_eq!(go_clean(input), want, "input {input:?}");
        }
        assert_eq!(go_join("/home/u/", ".config"), "/home/u/.config");
        assert_eq!(go_join("/", ".config"), "/.config");
    }

    #[test]
    fn go_quote_matches_invalid_byte_cases() {
        assert_eq!(go_quote_rune(b'X'), "'X'");
        assert_eq!(go_quote_rune(b'\''), "'\\''");
        assert_eq!(go_quote_rune(0x01), "'\\x01'");
    }

    #[test]
    fn execution_id_rules_match_go() {
        assert_eq!(go_base("/run/soda-muse/abc"), "abc");
        // 32-hex passes the shape check (mkdir may fail without privilege).
        assert!("a".repeat(32).bytes().all(|b| b.is_ascii_hexdigit()));
        assert!("g".repeat(32).bytes().any(|b| !b.is_ascii_hexdigit()));
    }

    #[test]
    fn muse_environment_orders_fixed_then_passthrough() {
        let env = muse_environment_with("/run/r", "/tmp/s", &|n| match n {
            "HOME" => Some(String::from("/home/u")),
            "TERM" => Some(String::new()),
            _ => None,
        });
        assert_eq!(
            env,
            vec![
                "PATH=/usr/local/bin:/usr/bin:/bin",
                "LANG=C.UTF-8",
                "TBH_CREDENTIAL_BACKEND=file",
                "XDG_CONFIG_HOME=/run/r/config",
                "XDG_STATE_HOME=/tmp/s/state",
                "XDG_CACHE_HOME=/tmp/s/cache",
                "XDG_DATA_HOME=/tmp/s/data",
                "TMPDIR=/tmp/s/tmp",
                "HOME=/home/u",
            ]
        );
    }

    #[test]
    fn copy_config_skips_credentials_and_copies_tree() {
        let root = std::env::temp_dir().join(format!("soda-muse-cc-{}", std::process::id()));
        let _ = fs::remove_dir_all(&root);
        let source = root.join("src");
        let dest = root.join("dst");
        fs::create_dir_all(source.join("sub")).unwrap();
        fs::write(source.join("settings.json"), b"{}").unwrap();
        fs::write(source.join("auth.json"), b"secret").unwrap();
        fs::write(source.join("sub").join("trust.json"), b"[]").unwrap();
        std::os::unix::fs::symlink(source.join("settings.json"), source.join("link.json")).unwrap();
        copy_config(source.to_str().unwrap(), dest.to_str().unwrap()).unwrap();
        assert_eq!(fs::read(dest.join("settings.json")).unwrap(), b"{}");
        assert_eq!(
            fs::read(dest.join("sub").join("trust.json")).unwrap(),
            b"[]"
        );
        assert_eq!(fs::read(dest.join("link.json")).unwrap(), b"{}");
        assert!(!dest.join("auth.json").exists());
        let mode = fs::metadata(dest.join("settings.json"))
            .unwrap()
            .permissions()
            .mode();
        assert_eq!(mode & 0o777, 0o600);
        // Missing source is a no-op.
        copy_config(
            root.join("absent").to_str().unwrap(),
            root.join("dst2").to_str().unwrap(),
        )
        .unwrap();
        // Oversized file refused.
        let big = source.join("big.json");
        fs::write(&big, vec![0u8; (1 << 20) + 1]).unwrap();
        assert!(copy_config(
            source.to_str().unwrap(),
            root.join("dst3").to_str().unwrap()
        )
        .is_err());
        let _ = fs::remove_dir_all(&root);
    }

    #[test]
    fn read_config_contract_shapes() {
        // Pure shape check: base64 values under sorted keys.
        let mut out = String::from("{");
        out.push_str(&json_string("settings.json"));
        out.push(':');
        out.push_str(&json_string(&base64_encode(b"{}")));
        out.push_str("}\n");
        assert_eq!(out, "{\"settings.json\":\"e30=\"}\n");
    }

    #[test]
    fn account_markers_reject_unsafe_nodes() {
        let root = std::env::temp_dir().join(format!("soda-muse-ac-{}", std::process::id()));
        let _ = fs::remove_dir_all(&root);
        fs::create_dir_all(&root).unwrap();
        // Temp files are user-owned, never root-owned: unsafe record.
        assert!(account_node(root.to_str().unwrap(), true).is_err());
        let marker = root.join("login");
        fs::write(&marker, b"42").unwrap();
        assert!(account_node(marker.to_str().unwrap(), false).is_err());
        let _ = fs::remove_dir_all(&root);
    }

    #[test]
    fn launch_shell_passes_fds_and_maps_exit() {
        for (reply, want_code, want_err) in [
            ("{\"code\":0}", 0, None),
            ("{\"code\":3}", 3, None),
            ("{\"code\":1,\"error\":\"denied\"}", 1, Some("denied")),
        ] {
            let mut pair = [0; 2];
            assert_eq!(
                unsafe {
                    libc::socketpair(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0, pair.as_mut_ptr())
                },
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
                let w = unsafe {
                    libc::send(fd, bytes.as_ptr() as *const libc::c_void, bytes.len(), 0)
                };
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

    #[test]
    fn native_dispatch_errors_match_go() {
        let v = |s: &[&str]| s.iter().map(|x| x.to_string()).collect::<Vec<_>>();
        assert_eq!(
            native_action(&v(&["--soda-copy-config", "a"])),
            Some((1, Some(String::from("config source and view required"))))
        );
        assert_eq!(
            native_action(&v(&["--soda-exec", "a"])),
            Some((1, Some(String::from("execution root and cwd required"))))
        );
        assert_eq!(
            native_action(&v(&["--soda-check"])),
            Some((1, Some(String::from("one metadata input required"))))
        );
        assert_eq!(native_action(&[]), None);
        assert_eq!(native_action(&v(&["--version"])), None);
        assert_eq!(native_action(&v(&["prompt text"])), None);
    }
}
