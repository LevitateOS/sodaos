// soda-identity-compose registers one explicitly opted-in Compose service.
use serde::Serialize;
use std::ffi::CString;
use std::fs;
use std::io;
use std::os::unix::fs::OpenOptionsExt;
use std::path::Path;
use std::process::{Command, Stdio};

const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";
const TMPFS_MAGIC: i64 = 0x01021994;

#[derive(Debug)]
struct Options {
    login: String,
    service: String,
    file: String,
    muse: bool,
}

fn main() {
    if let Err(e) = run() {
        eprintln!("soda-identity-compose: {e}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), String> {
    let o = load_options()?;
    let actor = account(&o.login)?;
    let (root, registration) = registration_root()?;
    let child = launch_compose(&o, &root)?;
    register(NestedRegistration {
        child_id: child,
        actor_id: actor.to_string(),
        registration_id: registration,
        muse: o.muse,
    })
}

fn load_options() -> Result<Options, String> {
    let args: Vec<String> = std::env::args().skip(1).collect();
    parse_options(&args)
}

fn parse_options(args: &[String]) -> Result<Options, String> {
    let mut login = String::new();
    let mut service = String::new();
    let mut file = String::from("compose.yml");
    let mut muse = false;
    let mut i = 0;
    let mut positional = 0;
    while i < args.len() {
        let arg = &args[i];
        if arg == "--" {
            positional += args.len() - i - 1;
            break;
        }
        if arg.starts_with('-') && arg.len() > 1 {
            let (name, inline) = match arg.find('=') {
                Some(p) => (&arg[..p], Some(&arg[p + 1..])),
                None => (arg.as_str(), None),
            };
            let key = name.trim_start_matches('-');
            match key {
                "login" | "service" | "file" => {
                    let value = match inline {
                        Some(v) => v.to_string(),
                        None => {
                            i += 1;
                            if i >= args.len() {
                                return Err(flag_error(format!("flag needs an argument: -{key}")));
                            }
                            args[i].clone()
                        }
                    };
                    match key {
                        "login" => login = value,
                        "service" => service = value,
                        _ => file = value,
                    }
                }
                "muse" => {
                    match inline {
                        Some(v) => {
                            muse = parse_bool_flag(v).ok_or_else(|| {
                                flag_error(format!(
                                    "invalid boolean value {v:?} for -muse: parse error"
                                ))
                            })?;
                        }
                        None => {
                            // Support `-muse=false` style via next arg only when
                            // explicitly boolean; otherwise bare flag means true.
                            // Go's flag package does not consume the next arg
                            // for bools, so mirror that: bare presence is true.
                            muse = true;
                        }
                    }
                }
                "h" | "help" => {
                    print_usage();
                    return Err(String::from("flag: help requested"));
                }
                _ => {
                    return Err(flag_error(format!("flag provided but not defined: -{key}")));
                }
            }
        } else {
            positional += 1;
        }
        i += 1;
    }
    let o = Options {
        login,
        service,
        file,
        muse,
    };
    if !valid_options(&o, positional) {
        return Err(String::from(
            "project root, provisioned login, one service and Muse access required",
        ));
    }
    Ok(o)
}

fn print_usage() {
    eprintln!("Usage of soda-identity-compose:");
    eprintln!("  -file string");
    eprintln!("    \tCompose file (default \"compose.yml\")");
    eprintln!("  -login string");
    eprintln!("    \tauthorizing provisioned Soda account");
    eprintln!("  -muse");
    eprintln!("    \tallow Muse in the selected service");
    eprintln!("  -service string");
    eprintln!("    \tCompose service to opt in");
}

fn flag_error(msg: String) -> String {
    eprintln!("{msg}");
    print_usage();
    msg
}

fn parse_bool_flag(v: &str) -> Option<bool> {
    match v {
        "1" | "t" | "T" | "true" | "TRUE" | "True" => Some(true),
        "0" | "f" | "F" | "false" | "FALSE" | "False" => Some(false),
        _ => None,
    }
}

fn valid_options(o: &Options, remaining: usize) -> bool {
    remaining == 0
        && unsafe { libc::geteuid() } == 0
        && !o.login.is_empty()
        && go_base(&o.login) == o.login
        && !o.service.is_empty()
        && o.muse
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

fn registration_root() -> Result<(String, String), String> {
    let mut id = [0u8; 16];
    read_random(&mut id)?;
    let registration = hex_encode(&id);
    let root = format!("/run/soda-muse/nested/{registration}");
    let dir = "/run/soda-muse/nested";
    mkdir_p(dir, 0o711)?;
    // Nested registration requires runtime tmpfs.
    let mut fs: libc::statfs64 = unsafe { std::mem::zeroed() };
    let c = CString::new(dir).unwrap();
    let rc = unsafe { libc::statfs64(c.as_ptr(), &mut fs) };
    if rc != 0 || fs.f_type as i64 != TMPFS_MAGIC {
        return Err(String::from("nested registration requires runtime tmpfs"));
    }
    let croot = CString::new(root.clone()).unwrap();
    let rc = unsafe { libc::mkdir(croot.as_ptr(), 0o711) };
    if rc != 0 {
        return Err(io::Error::last_os_error().to_string());
    }
    Ok((root, registration))
}

fn read_random(buf: &mut [u8]) -> Result<(), String> {
    use std::io::Read;
    let mut f = fs::File::open("/dev/urandom").map_err(|e| e.to_string())?;
    f.read_exact(buf).map_err(|e| e.to_string())
}

fn hex_encode(bytes: &[u8]) -> String {
    let mut s = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        s.push(char::from_digit((b >> 4) as u32, 16).unwrap());
        s.push(char::from_digit((b & 0xf) as u32, 16).unwrap());
    }
    s
}

fn mkdir_p(path: &str, mode: u32) -> Result<(), String> {
    // Mirror Go MkdirAll: create missing ancestors, leave existing alone.
    let mut current = String::new();
    for part in path.split('/') {
        if part.is_empty() {
            if current.is_empty() {
                current.push('/');
            }
            continue;
        }
        if current == "/" || current.is_empty() {
            current.push_str(part);
        } else {
            current.push('/');
            current.push_str(part);
        }
        let c = CString::new(current.clone()).unwrap();
        let rc = unsafe { libc::mkdir(c.as_ptr(), mode) };
        if rc != 0 {
            let e = io::Error::last_os_error();
            if e.kind() != io::ErrorKind::AlreadyExists {
                return Err(e.to_string());
            }
        }
    }
    Ok(())
}

fn launch_compose(o: &Options, root: &str) -> Result<String, String> {
    let override_path = format!("{root}/compose.json");
    write_override(&override_path, &o.service, root)?;
    let args = ["-f", o.file.as_str(), "-f", override_path.as_str()];
    let mut up_cmd = Vec::from(args);
    up_cmd.extend(["up", "-d", o.service.as_str()]);
    let status = Command::new("/usr/local/bin/podman-compose")
        .args(&up_cmd)
        .stdin(Stdio::inherit())
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit())
        .status()
        .map_err(|_| String::from("compose did not confirm opted-in service creation"))?;
    if !status.success() {
        return Err(String::from(
            "compose did not confirm opted-in service creation",
        ));
    }
    let mut ps_cmd = Vec::from(args);
    ps_cmd.extend(["ps", "-q"]);
    let out = Command::new("/usr/local/bin/podman-compose")
        .args(&ps_cmd)
        .output()
        .map_err(|_| String::from("compose container identification failed"))?;
    if !out.status.success() {
        return Err(String::from("compose container identification failed"));
    }
    let text = String::from_utf8_lossy(&out.stdout).into_owned();
    select_compose_child(&text, &o.service, &|id| {
        let body = Command::new("/usr/bin/podman")
            .args([
                "--remote=false",
                "inspect",
                "--format",
                "{{.ID}} {{index .Config.Labels \"io.podman.compose.service\"}}",
                id,
            ])
            .output()
            .map_err(|e| e.to_string())?;
        Ok(String::from_utf8_lossy(&body.stdout).into_owned())
    })
}

fn account(login: &str) -> Result<i64, String> {
    let path = format!("/var/lib/soda/accounts/{login}");
    let c = CString::new(path.clone()).unwrap();
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::lstat(c.as_ptr(), &mut st) };
    if rc != 0 {
        return Err(String::from("provisioned account marker required"));
    }
    let is_reg = (st.st_mode & libc::S_IFMT) == libc::S_IFREG;
    let perm = st.st_mode & 0o777;
    if !is_reg || perm != 0o600 {
        return Err(String::from("provisioned account marker required"));
    }
    if st.st_uid != 0 {
        return Err(String::from("root-owned account marker required"));
    }
    let data = fs::read(&path).map_err(|e| e.to_string())?;
    let text = String::from_utf8_lossy(&data);
    let actor: i64 = text
        .trim()
        .parse()
        .map_err(|_| String::from("invalid provisioned account"))?;
    if actor <= 0 {
        return Err(String::from("invalid provisioned account"));
    }
    Ok(actor)
}

fn write_override(path: &str, service: &str, root: &str) -> Result<(), String> {
    let sock_dir = Path::new(MUSE_LAUNCH_SOCKET)
        .parent()
        .and_then(|p| p.to_str())
        .unwrap_or("/run/soda-muse-interface");
    let mounts = vec![
        format!("{root}:/run/soda-muse/credentials:ro"),
        String::from("/usr/local/bin/muse:/usr/local/bin/muse:ro"),
        String::from("/usr/local/libexec/soda/muse:/usr/local/libexec/soda/muse:ro"),
        format!("{sock_dir}:{sock_dir}:ro"),
    ];
    let wire = serde_json::json!({
        "services": {
            service: {
                "volumes": mounts,
            }
        }
    });
    let data = serde_json::to_vec(&wire).map_err(|e| e.to_string())?;
    let mut opts = fs::OpenOptions::new();
    opts.write(true).create(true).truncate(true);
    opts.mode(0o600);
    use std::io::Write;
    let mut f = opts.open(path).map_err(|e| e.to_string())?;
    f.write_all(&data).map_err(|e| e.to_string())?;
    Ok(())
}

#[derive(Serialize)]
struct NestedRegistration {
    child_id: String,
    actor_id: String,
    registration_id: String,
    muse: bool,
}

#[derive(Serialize)]
struct LaunchRequest {
    #[serde(skip_serializing_if = "String::is_empty")]
    home: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    register: Option<NestedRegistration>,
    #[serde(skip_serializing_if = "String::is_empty", rename = "config_home")]
    config_home: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    term: String,
    #[serde(rename = "connection_id")]
    connection_id: String,
    cwd: String,
    args: Option<Vec<String>>,
    tty: bool,
    cols: u16,
    rows: u16,
}

fn register(request: NestedRegistration) -> Result<(), String> {
    let fd = unsafe { libc::socket(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0) };
    if fd < 0 {
        return Err(String::from("identity registration service unavailable"));
    }
    struct Guard(i32);
    impl Drop for Guard {
        fn drop(&mut self) {
            unsafe { libc::close(self.0) };
        }
    }
    let _guard = Guard(fd);
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as libc::sa_family_t;
    let bytes = MUSE_LAUNCH_SOCKET.as_bytes();
    if bytes.len() >= addr.sun_path.len() {
        return Err(String::from("identity registration service unavailable"));
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
        return Err(String::from("identity registration service unavailable"));
    }
    let tv = libc::timeval {
        tv_sec: 30,
        tv_usec: 0,
    };
    unsafe {
        libc::setsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_SNDTIMEO,
            &tv as *const libc::timeval as *const libc::c_void,
            std::mem::size_of::<libc::timeval>() as libc::socklen_t,
        );
        libc::setsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_RCVTIMEO,
            &tv as *const libc::timeval as *const libc::c_void,
            std::mem::size_of::<libc::timeval>() as libc::socklen_t,
        );
    }
    let req = LaunchRequest {
        home: String::new(),
        register: Some(request),
        config_home: String::new(),
        term: String::new(),
        connection_id: String::new(),
        cwd: String::new(),
        args: None,
        tty: false,
        cols: 0,
        rows: 0,
    };
    let data = serde_json::to_vec(&req).map_err(|e| e.to_string())?;
    let n = unsafe { libc::send(fd, data.as_ptr() as *const libc::c_void, data.len(), 0) };
    if n < 0 || (n as usize) != data.len() {
        return Err(io::Error::last_os_error().to_string());
    }
    let mut body = [0u8; 4096];
    let r = unsafe { libc::recv(fd, body.as_mut_ptr() as *mut libc::c_void, body.len(), 0) };
    if r <= 0 {
        return Err(String::from("identity registration unconfirmed"));
    }
    let resp: serde_json::Value = serde_json::from_slice(&body[..r as usize])
        .map_err(|_| String::from("identity registration rejected"))?;
    let code = resp.get("code").and_then(|v| v.as_i64()).unwrap_or(-1);
    let err_text = resp.get("error").and_then(|v| v.as_str()).unwrap_or("x");
    if code != 0 || !err_text.is_empty() {
        return Err(String::from("identity registration rejected"));
    }
    Ok(())
}

// Resolve only this Compose project's handles to native immutable IDs.
fn select_compose_child(
    output: &str,
    service: &str,
    inspect: &dyn Fn(&str) -> Result<String, String>,
) -> Result<String, String> {
    let mut child = String::new();
    for id in output.split_whitespace() {
        if !compose_container_handle(id) {
            return Err(String::from("invalid Compose container identity"));
        }
        let label = inspect(id).map_err(|_| String::from("compose service attribution failed"))?;
        let full = compose_observed_child(id, service, &label)?;
        if full.is_empty() {
            continue;
        }
        if !child.is_empty() {
            return Err(String::from(
                "exactly one immutable Compose container required",
            ));
        }
        child = full;
    }
    if child.is_empty() {
        return Err(String::from("requested Compose service container missing"));
    }
    Ok(child)
}

fn compose_container_handle(id: &str) -> bool {
    (id.len() == 12 || id.len() == 64) && is_hex(id)
}

fn compose_observed_child(handle: &str, service: &str, output: &str) -> Result<String, String> {
    let parts: Vec<&str> = output.split_whitespace().collect();
    if parts.len() != 2 || !immutable_compose_id(parts[0]) || !parts[0].starts_with(handle) {
        return Err(String::from(
            "native Compose container identity unconfirmed",
        ));
    }
    if parts[1] != service {
        return Ok(String::new());
    }
    Ok(parts[0].to_string())
}

fn immutable_compose_id(id: &str) -> bool {
    id.len() == 64 && is_hex(id)
}

fn is_hex(s: &str) -> bool {
    !s.is_empty() && s.bytes().all(|b| b.is_ascii_hexdigit())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn selects_only_requested_service_within_project() {
        let first = "a".repeat(64);
        let second = "b".repeat(64);
        let output = format!("{first}\n{second}\n");
        let child = select_compose_child(&output, "development", &|id| {
            if id == first {
                Ok(format!("{id} database\n"))
            } else {
                Ok(format!("{id} development\n"))
            }
        })
        .unwrap();
        assert_eq!(child, second);
    }

    #[test]
    fn rejects_ambiguous_or_unconfirmed_identity() {
        let id = "a".repeat(64);
        for output in ["", "--all", &format!("{id}\n{id}")] {
            let out = if output == "--all" {
                "--all".to_string()
            } else {
                output.to_string()
            };
            assert!(
                select_compose_child(&out, "development", &|v| Ok(format!("{v} development")))
                    .is_err(),
                "unconfirmed unique container admitted for {out:?}"
            );
        }
        assert!(
            select_compose_child(&id, "development", &|_| Err(String::from(
                "runtime unavailable"
            )))
            .is_err()
        );
    }

    #[test]
    fn resolves_upstream_short_handle_to_immutable_identity() {
        let id = "a".repeat(64);
        let child = select_compose_child(&id[..12], "development", &|_| {
            Ok(format!("{id} development"))
        })
        .unwrap();
        assert_eq!(child, id);
        assert!(select_compose_child(&id[..12], "development", &|_| {
            Ok(format!("{} development", "b".repeat(64)))
        })
        .is_err());
    }

    #[test]
    fn compose_muse_mounts_are_explicit() {
        let dir = std::env::temp_dir().join(format!("soda-compose-test-{}", std::process::id()));
        let _ = fs::create_dir_all(&dir);
        let path = dir.join("compose.json");
        write_override(
            path.to_str().unwrap(),
            "development",
            "/run/soda-muse/nested/registration",
        )
        .unwrap();
        let body = fs::read(&path).unwrap();
        let wire: serde_json::Value = serde_json::from_slice(&body).unwrap();
        let volumes = wire["services"]["development"]["volumes"]
            .as_array()
            .unwrap();
        for want in [
            "/run/soda-muse/credentials",
            "/usr/local/bin/muse",
            "/run/soda-muse-interface",
        ] {
            let mut found = false;
            for m in volumes {
                if let Some(s) = m.as_str() {
                    if mount_destination(s) == Some(want) {
                        found = true;
                    }
                }
            }
            assert!(found, "missing {want} from {volumes:?}");
        }
        let _ = fs::remove_dir_all(&dir);
    }

    fn mount_destination(mount: &str) -> Option<&str> {
        let (_, rest) = mount.split_once(':')?;
        let (target, _) = rest.split_once(':')?;
        Some(target)
    }

    #[test]
    fn go_base_matches_filepath_semantics() {
        assert_eq!(go_base("login"), "login");
        assert_eq!(go_base("a/b"), "b");
        assert_eq!(go_base("/"), "/");
        assert_eq!(go_base(""), ".");
    }

    #[test]
    fn bool_flag_values_match_go() {
        assert_eq!(parse_bool_flag("true"), Some(true));
        assert_eq!(parse_bool_flag("1"), Some(true));
        assert_eq!(parse_bool_flag("false"), Some(false));
        assert_eq!(parse_bool_flag("0"), Some(false));
        assert_eq!(parse_bool_flag("yes"), None);
    }

    #[test]
    fn launch_request_wire_matches_go() {
        let req = LaunchRequest {
            home: String::new(),
            register: Some(NestedRegistration {
                child_id: "c".repeat(64),
                actor_id: "42".to_string(),
                registration_id: "r".repeat(32),
                muse: true,
            }),
            config_home: String::new(),
            term: String::new(),
            connection_id: String::new(),
            cwd: String::new(),
            args: None,
            tty: false,
            cols: 0,
            rows: 0,
        };
        let v: serde_json::Value =
            serde_json::from_slice(&serde_json::to_vec(&req).unwrap()).unwrap();
        assert_eq!(v["register"]["actor_id"], "42");
        assert!(v.get("home").is_none());
        assert_eq!(v["connection_id"], "");
        assert!(v["args"].is_null());
        assert_eq!(v["tty"], false);
    }
}
