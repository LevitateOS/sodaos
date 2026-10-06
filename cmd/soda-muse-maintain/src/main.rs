// soda-muse-maintain installs public Muse tools and restores a live project interface.
use std::ffi::CString;
use std::fs;
use std::io::Read;
use std::os::unix::fs::FileTypeExt;
use std::os::unix::fs::MetadataExt;
use std::os::unix::io::{FromRawFd, RawFd};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

mod config;
mod config_validation;
mod config_wire;
mod filesystem;
mod json;
mod json_string;
mod network;
mod options;
mod project;
mod release;
mod release_validation;
mod release_wire;
mod sha256;

use config::{load_config, Config};
use filesystem::{go_base, go_dir, go_errno, last_errno};
use options::{parse, Options};
use project::{confirm_project, wait_project, Observation};
use sha256::{hex_encode, Sha256};

const MUSE_VERSION: &str = "1.4.0-R4161.1";
const RELEASE_PATH: &str = "/usr/share/soda/release.json";

// --wait/--pipe needs the native system bus, not just systemd's private socket.
// Installing this demonstrated prerequisite belongs only to explicit maintenance.
const BUS_SCRIPT: &str = r#"set -eu
if ! rpm -q dbus-broker >/dev/null; then dnf -y install dbus-broker; fi
systemctl start dbus.socket
systemd-run --quiet --wait --pipe --collect /usr/bin/true
"#;

const INTERFACE_SCRIPT: &str = "set -eu; test ! -L /run; test -d /run; test ! -L /run/soda-muse-interface; if test -e /run/soda-muse-interface; then test -d /run/soda-muse-interface; fi; mkdir -p /run/soda-muse-interface";

// Destinations are fixed by the installed project interface, never repository input.
const DESTINATIONS: [&str; 3] = [
    "/usr/local/bin/muse",
    "/usr/local/bin/soda-identity-compose",
    "/usr/local/libexec/soda/muse",
];

const INSTALL_SCRIPT: &str = r#"
set -eu
safe_parent() {
 path=$(dirname "$1")
 while [ "$path" != / ]; do
  test ! -L "$path"
  if test -e "$path"; then test -d "$path"; fi
  path=$(dirname "$path")
 done
}
for target in "$@"; do
 safe_parent "$target"
 test ! -L "$target"
 if test -e "$target"; then test -f "$target"; fi
 done
for target in "$@"; do mkdir -p "$(dirname "$target")"; done
stage=$(mktemp -d "$(dirname "$3")/.soda-muse-maintain.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
tar --extract --file=- --directory="$stage" --no-same-owner
chmod 0755 "$stage/muse" "$stage/soda-identity-compose" "$stage/muse-native"
mv -T -- "$stage/muse" "$1"
mv -T -- "$stage/soda-identity-compose" "$2"
mv -T -- "$stage/muse-native" "$3"
"#;

fn main() {
    if let Err(e) = run() {
        eprintln!("soda-muse-maintain: {e}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("project maintenance requires root"));
    }
    let argv: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    let o = parse(&argv)?;
    let c = load_config(&o.config)?;
    if c.muse_sha256.is_empty() {
        if o.bind_only {
            return Ok(());
        }
        return Err(String::from("muse project runtime is disabled"));
    }
    maintain(&o, &c)
}

fn is_lower_hex(b: u8) -> bool {
    b.is_ascii_digit() || (b'a'..=b'f').contains(&b)
}

fn maintain(o: &Options, c: &Config) -> Result<(), String> {
    // Tool closes its own fd on drop, mirroring deferred closeTools.
    let sources = load_tools(&o.tools, &c.muse_sha256, &c.muse_version)?;
    let deadline = Instant::now() + Duration::from_secs(120);
    let target = wait_project(&o.project, deadline)?;
    if !o.bind_only {
        ensure_system_bus(&target, deadline)?;
        stage_tools(&target, &sources, deadline)?;
    }
    prepare_interface(&target, deadline)?;
    attach_interface(&target, &c.muse_socket, deadline)
}

#[derive(Debug)]
struct Tool {
    name: String,
    fd: RawFd,
    size: u64,
}

impl Drop for Tool {
    fn drop(&mut self) {
        if self.fd >= 0 {
            unsafe { libc::close(self.fd) };
        }
    }
}

fn load_tools(dir: &str, digest: &str, version: &str) -> Result<Vec<Tool>, String> {
    if version != MUSE_VERSION {
        return Err(String::from(
            "muse maintenance version differs from pinned release",
        ));
    }
    let mut tools = Vec::new();
    for name in ["muse", "soda-identity-compose", "muse-native"] {
        tools.push(open_tool(dir, name)?);
    }
    verify_native(&tools[2], digest)?;
    Ok(tools)
}

fn open_tool(dir: &str, name: &str) -> Result<Tool, String> {
    let path = format!("{dir}/{name}");
    let trusted = String::from("public tool source must be a regular root-owned executable");
    let lstat = fs::symlink_metadata(&path).map_err(|_| trusted.clone())?;
    if !trusted_tool(&lstat) {
        return Err(trusted);
    }
    // A NUL byte fails like Go's BytePtrFromString: raw EINVAL.
    let c = CString::new(path).map_err(|_| go_errno(libc::EINVAL))?;
    let fd = unsafe {
        libc::open(
            c.as_ptr(),
            libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_CLOEXEC | libc::O_NONBLOCK,
        )
    };
    if fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let mut fst: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd, &mut fst) } != 0 {
        unsafe { libc::close(fd) };
        return Err(String::from("public tool unavailable"));
    }
    if (fst.st_mode & libc::S_IFMT) != libc::S_IFREG
        || (fst.st_mode & 0o7777) & 0o022 != 0
        || fst.st_uid != 0
        || (fst.st_mode & 0o7777) & 0o111 == 0
    {
        unsafe { libc::close(fd) };
        return Err(String::from("public tool source changed"));
    }
    Ok(Tool {
        name: name.to_string(),
        fd,
        size: fst.st_size as u64,
    })
}

fn trusted_tool(info: &fs::Metadata) -> bool {
    info.is_file() && info.uid() == 0 && info.mode() & 0o022 == 0 && info.mode() & 0o111 != 0
}

fn verify_native(tool: &Tool, digest: &str) -> Result<(), String> {
    // io.Copy hashes from the start to EOF; a short or grown file fails
    // the digest comparison, and read failures keep Go's PathError shape.
    // pread leaves the offset at zero like Go's trailing Seek.
    let mut hasher = Sha256::new();
    let mut offset: i64 = 0;
    let mut chunk = [0u8; 65536];
    loop {
        let n = unsafe {
            libc::pread(
                tool.fd,
                chunk.as_mut_ptr() as *mut libc::c_void,
                chunk.len(),
                offset,
            )
        };
        if n < 0 {
            return Err(format!("read {}: {}", tool.name, go_errno(last_errno())));
        }
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n as usize]);
        offset += n as i64;
    }
    let sum = hasher.finish();
    if hex_encode(&sum) != digest {
        return Err(String::from("muse native digest mismatch"));
    }
    Ok(())
}

fn podman(args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
    let mut cmd = Command::new("podman");
    cmd.arg("--remote=false");
    for a in args {
        cmd.arg(a);
    }
    cmd.stdin(Stdio::null());
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = cmd
        .spawn()
        .map_err(|_| String::from("project maintenance command failed"))?;
    wait_output(child, deadline)
}

// podman_streamed feeds stdin from a writer thread through a pipe while
// podman consumes it, mirroring Go's io.Pipe archive delivery. A podman
// failure wins over the archive result exactly like stagePublicTools.
fn podman_streamed(
    feed: impl FnOnce(fs::File) -> Result<(), String> + Send + 'static,
    args: &[&str],
    deadline: Instant,
) -> Result<(), String> {
    let mut fds = [0; 2];
    // O_CLOEXEC is load-bearing: without it the spawned child inherits
    // the stdin write end and its stdin reader never sees EOF.
    if unsafe { libc::pipe2(fds.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
        return Err(String::from("project maintenance command failed"));
    }
    let writer = unsafe { fs::File::from_raw_fd(fds[1]) };
    let reader = unsafe { fs::File::from_raw_fd(fds[0]) };
    let feeder = std::thread::spawn(move || feed(writer));
    let mut cmd = Command::new("podman");
    cmd.arg("--remote=false");
    for a in args {
        cmd.arg(a);
    }
    cmd.stdin(Stdio::from(reader));
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = match cmd.spawn() {
        Ok(child) => child,
        Err(_) => {
            let _ = feeder.join();
            return Err(String::from("project maintenance command failed"));
        }
    };
    let output = wait_output(child, deadline);
    let fed = feeder
        .join()
        .map_err(|_| String::from("project maintenance command failed"))?;
    output?;
    fed
}

fn wait_output(mut child: std::process::Child, deadline: Instant) -> Result<Vec<u8>, String> {
    // Stdout drains on its own thread like exec.Output, so large output
    // never deadlocks against the exit wait. The child handle stays here
    // for the deadline kill.
    let failed = String::from("project maintenance command failed");
    let stdout = child.stdout.take();
    let reader = std::thread::spawn(move || {
        let mut out = Vec::new();
        match stdout {
            Some(mut pipe) => match pipe.read_to_end(&mut out) {
                Ok(_) => Some(out),
                Err(_) => None,
            },
            None => Some(out),
        }
    });
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                let out = reader
                    .join()
                    .map_err(|_| failed.clone())?
                    .ok_or(failed.clone())?;
                if !status.success() {
                    return Err(failed);
                }
                return Ok(out);
            }
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    let _ = reader.join();
                    return Err(failed);
                }
                std::thread::sleep(Duration::from_millis(10));
            }
            Err(_) => {
                let _ = reader.join();
                return Err(failed);
            }
        }
    }
}

fn ensure_system_bus(target: &Observation, deadline: Instant) -> Result<(), String> {
    confirm_project(target, deadline)?;
    podman(
        &[
            "exec", "--user", "0:0", &target.id, "/bin/sh", "-ceu", BUS_SCRIPT,
        ],
        deadline,
    )?;
    Ok(())
}

fn stage_tools(target: &Observation, sources: &[Tool], deadline: Instant) -> Result<(), String> {
    confirm_project(target, deadline)?;
    // The feeder thread borrows no Tool state: the fds stay open in the
    // caller while only plain ints cross the thread boundary.
    let feeds: Vec<(String, RawFd, u64)> = sources
        .iter()
        .map(|t| (t.name.clone(), t.fd, t.size))
        .collect();
    let args: Vec<&str> = vec![
        "exec",
        "--user",
        "0:0",
        "-i",
        &target.id,
        "/bin/sh",
        "-ceu",
        INSTALL_SCRIPT,
        "soda-muse-maintain",
        DESTINATIONS[0],
        DESTINATIONS[1],
        DESTINATIONS[2],
    ];
    podman_streamed(
        move |writer| feed_archive(writer, &feeds, deadline),
        &args,
        deadline,
    )?;
    Ok(())
}

// feed_archive streams the tar byte sequence to podman's stdin. Writes
// poll non-blocking against the deadline so an abandoned pipe ends with
// the maintenance failure instead of hanging past it; a dead reader
// surfaces Go's closed-pipe error.
fn feed_archive(
    writer: fs::File,
    feeds: &[(String, RawFd, u64)],
    deadline: Instant,
) -> Result<(), String> {
    use std::os::unix::io::AsRawFd;
    let wfd = writer.as_raw_fd();
    let flags = unsafe { libc::fcntl(wfd, libc::F_GETFL) };
    if flags >= 0 {
        unsafe {
            libc::fcntl(wfd, libc::F_SETFL, flags | libc::O_NONBLOCK);
        }
    }
    let mut emit = |buf: &[u8]| -> Result<(), String> {
        let mut rest = buf;
        while !rest.is_empty() {
            let n = unsafe { libc::write(wfd, rest.as_ptr() as *const libc::c_void, rest.len()) };
            if n > 0 {
                rest = &rest[n as usize..];
                continue;
            }
            if n == 0 {
                continue;
            }
            let no = last_errno();
            if no == libc::EINTR {
                continue;
            }
            if no == libc::EAGAIN {
                if Instant::now() >= deadline {
                    return Err(String::from("project maintenance command failed"));
                }
                std::thread::sleep(Duration::from_millis(10));
                continue;
            }
            return Err(String::from("io: read/write on closed pipe"));
        }
        Ok(())
    };
    emit_archive(&mut emit, feeds)
}

// emit_archive ports archiveTools: one USTAR header per tool, CopyN of
// exactly size bytes, 512 padding, and the two zero trailer blocks.
// A short file ends CopyN with raw io.EOF; read failures keep the
// PathError shape with the tool's archive name.
fn emit_archive(
    emit: &mut dyn FnMut(&[u8]) -> Result<(), String>,
    feeds: &[(String, RawFd, u64)],
) -> Result<(), String> {
    for (name, fd, size) in feeds {
        emit(&tar_header(name, *size)?)?;
        let mut remaining = *size;
        let mut offset: i64 = 0;
        let mut chunk = [0u8; 65536];
        while remaining > 0 {
            let want = remaining.min(chunk.len() as u64) as usize;
            let n =
                unsafe { libc::pread(*fd, chunk.as_mut_ptr() as *mut libc::c_void, want, offset) };
            if n < 0 {
                return Err(format!("read {name}: {}", go_errno(last_errno())));
            }
            if n == 0 {
                return Err(String::from("EOF"));
            }
            emit(&chunk[..n as usize])?;
            offset += n as i64;
            remaining -= n as u64;
        }
        let pad = (512 - (size % 512)) % 512;
        if pad > 0 {
            let zeros = vec![0u8; pad as usize];
            emit(&zeros)?;
        }
    }
    emit(&[0u8; 1024])?;
    Ok(())
}

// tar_header writes the USTAR byte stream Go's archive/tar emits for
// these short regular names: fixed headers, 512-block data, two zero
// blocks. Names are fixed constants, so the length guard never fires.
fn tar_header(name: &str, size: u64) -> Result<[u8; 512], String> {
    let mut header = [0u8; 512];
    if name.len() > 100 || name.contains('\0') {
        return Err(String::from("public tool name exceeds archive limit"));
    }
    header[..name.len()].copy_from_slice(name.as_bytes());
    // Mode 0755, uid/gid 0, size octal, mtime 0, regular file, USTAR.
    header[100..108].copy_from_slice(format!("{:07o}\0", 0o755).as_bytes());
    header[108..116].copy_from_slice(b"0000000\0");
    header[116..124].copy_from_slice(b"0000000\0");
    header[124..136].copy_from_slice(format!("{:011o}\0", size).as_bytes());
    header[136..148].copy_from_slice(b"00000000000\0");
    header[156] = b'0';
    header[329..337].copy_from_slice(b"0000000\0");
    header[337..345].copy_from_slice(b"0000000\0");
    header[257..263].copy_from_slice(b"ustar\0");
    header[263..265].copy_from_slice(b"00");
    // Checksum over spaces, then six octal digits, NUL, space.
    header[148..156].copy_from_slice(b"        ");
    let sum: u32 = header.iter().map(|b| *b as u32).sum();
    header[148..156].copy_from_slice(format!("{sum:06o}\0 ").as_bytes());
    Ok(header)
}

fn prepare_interface(target: &Observation, deadline: Instant) -> Result<(), String> {
    confirm_project(target, deadline)?;
    podman(
        &[
            "exec",
            "--user",
            "0:0",
            &target.id,
            "/bin/sh",
            "-ceu",
            INTERFACE_SCRIPT,
        ],
        deadline,
    )?;
    Ok(())
}

struct FdGuard(RawFd);

impl Drop for FdGuard {
    fn drop(&mut self) {
        if self.0 >= 0 {
            unsafe { libc::close(self.0) };
        }
    }
}

fn attach_interface(
    target: &Observation,
    muse_socket: &str,
    deadline: Instant,
) -> Result<(), String> {
    // attachLaunchInterface with kind "muse".
    let source = public_socket_directory(muse_socket)?;
    let c = CString::new(source).map_err(|_| go_errno(libc::EINVAL))?;
    let source_fd = unsafe {
        libc::open(
            c.as_ptr(),
            libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if source_fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let _source_guard = FdGuard(source_fd);
    let tree = syscall_open_tree(source_fd)?;
    let _tree_guard = FdGuard(tree);
    restrict_interface_mount(tree)?;
    attach_project_mount(target, tree, deadline)
}

fn syscall_open_tree(source_fd: RawFd) -> Result<RawFd, String> {
    let empty = c"";
    let tree = unsafe {
        libc::syscall(
            libc::SYS_open_tree,
            source_fd as libc::c_long,
            empty.as_ptr(),
            (libc::OPEN_TREE_CLONE | libc::OPEN_TREE_CLOEXEC | libc::AT_EMPTY_PATH as u32)
                as libc::c_long,
        )
    };
    if tree < 0 {
        return Err(format!(
            "clone public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(tree as RawFd)
}

fn restrict_interface_mount(tree: RawFd) -> Result<(), String> {
    let mut attr: libc::mount_attr = unsafe { std::mem::zeroed() };
    attr.attr_set = libc::MOUNT_ATTR_RDONLY
        | libc::MOUNT_ATTR_NOSUID
        | libc::MOUNT_ATTR_NODEV
        | libc::MOUNT_ATTR_NOEXEC;
    let empty = c"";
    let rc = unsafe {
        libc::syscall(
            libc::SYS_mount_setattr,
            tree as libc::c_long,
            empty.as_ptr(),
            libc::AT_EMPTY_PATH as libc::c_long,
            &mut attr as *mut libc::mount_attr,
            std::mem::size_of::<libc::mount_attr>() as libc::c_long,
        )
    };
    if rc != 0 {
        return Err(format!(
            "restrict public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(())
}

fn attach_project_mount(
    target: &Observation,
    tree: RawFd,
    deadline: Instant,
) -> Result<(), String> {
    let root = format!("/proc/{}", target.pid);
    let dest = format!("{root}/root/run/soda-muse-interface");
    let cdest = CString::new(dest).map_err(|_| go_errno(libc::EINVAL))?;
    let target_fd = unsafe {
        libc::open(
            cdest.as_ptr(),
            libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if target_fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let _target_guard = FdGuard(target_fd);
    let ns = format!("{root}/ns/mnt");
    let cns = CString::new(ns).map_err(|_| go_errno(libc::EINVAL))?;
    let namespace = unsafe { libc::open(cns.as_ptr(), libc::O_RDONLY | libc::O_CLOEXEC, 0) };
    if namespace < 0 {
        return Err(go_errno(last_errno()));
    }
    let _ns_guard = FdGuard(namespace);
    // The incarnation check runs after both opens, just before entry.
    confirm_project(target, deadline)?;
    if unsafe { libc::unshare(libc::CLONE_FS) } != 0 {
        return Err(format!(
            "separate maintenance filesystem context: {}",
            go_errno(last_errno())
        ));
    }
    if unsafe { libc::setns(namespace, libc::CLONE_NEWNS) } != 0 {
        return Err(format!(
            "enter project mount namespace: {}",
            go_errno(last_errno())
        ));
    }
    let empty = c"";
    let rc = unsafe {
        libc::syscall(
            libc::SYS_move_mount,
            tree as libc::c_long,
            empty.as_ptr(),
            target_fd as libc::c_long,
            empty.as_ptr(),
            (libc::MOVE_MOUNT_F_EMPTY_PATH | libc::MOVE_MOUNT_T_EMPTY_PATH) as libc::c_long,
        )
    };
    if rc != 0 {
        return Err(format!(
            "attach public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(())
}

fn public_socket_directory(socket: &str) -> Result<String, String> {
    if !socket.starts_with('/') || go_base(socket) != "launch.sock" {
        return Err(String::from("explicit public launch socket required"));
    }
    let root = go_dir(socket);
    let only = String::from("launch directory must contain only the public socket");
    let entries = fs::read_dir(&root).map_err(|_| only.clone())?;
    let mut names: Vec<String> = Vec::new();
    for entry in entries {
        let entry = entry.map_err(|_| only.clone())?;
        names.push(entry.file_name().to_string_lossy().into_owned());
    }
    names.sort();
    if names.len() != 1 || names[0] != "launch.sock" {
        return Err(only);
    }
    validate_public_socket(socket)?;
    validate_interface_directory(&root)?;
    Ok(root)
}

fn validate_public_socket(socket: &str) -> Result<(), String> {
    let info = fs::symlink_metadata(socket)
        .map_err(|_| String::from("public launch socket is unavailable"))?;
    if !info.file_type().is_socket() {
        return Err(String::from("public launch socket is unavailable"));
    }
    if info.uid() != 0 || info.mode() & 0o777 != 0o666 {
        return Err(String::from(
            "public launch socket must be root-owned and public",
        ));
    }
    Ok(())
}

fn validate_interface_directory(root: &str) -> Result<(), String> {
    let info = fs::symlink_metadata(root)
        .map_err(|_| String::from("public launch directory must be a protected directory"))?;
    if !info.is_dir() || info.mode() & 0o777 & 0o022 != 0 {
        return Err(String::from(
            "public launch directory must be a protected directory",
        ));
    }
    if info.uid() != 0 || info.mode() & 0o777 & 0o055 != 0o055 {
        return Err(String::from(
            "public launch directory must be root-owned and accessible",
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::config::{load_config, Config};
    use super::config_validation::validate_runtime_config;
    use super::config_wire::{decode_host_config, go_quoted};
    use super::filesystem::{go_base, go_clean, go_dir, go_errno};
    use super::network::parse_prefix;
    use super::options::{default_tools, parse, usage_text};
    use super::project::{decode_observation, validate_observation, Observation};
    use super::release::{apply_release_images, load_release_payload};
    use super::sha256::{hex_encode, Sha256};
    use super::*;
    use std::io::Write as _;
    use std::os::unix::fs::PermissionsExt;
    use std::sync::atomic::{AtomicU64, Ordering};

    static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

    struct TestDir(std::path::PathBuf);

    impl TestDir {
        fn make(tag: &str) -> TestDir {
            let id = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
            let dir = std::env::temp_dir().join(format!(
                "smm-test-{}-{}-{}",
                std::process::id(),
                id,
                tag
            ));
            let _ = fs::remove_dir_all(&dir);
            fs::create_dir_all(&dir).unwrap();
            TestDir(dir)
        }

        fn path(&self, name: &str) -> String {
            self.0.join(name).to_string_lossy().into_owned()
        }

        fn dir_str(&self) -> String {
            self.0.to_string_lossy().into_owned()
        }
    }

    impl Drop for TestDir {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }

    fn is_root() -> bool {
        unsafe { libc::geteuid() == 0 }
    }

    #[test]
    fn sha256_known_answers() {
        let vectors: &[(&[u8], &str)] = &[
            (
                b"",
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
            ),
            (
                b"abc",
                "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
            ),
            // 55 bytes: padding fits in the final block.
            (
                b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "9f4390f8d30c2dd92ec9f095b65e2b9ae9b0a925a5258e241c9f1e910f734318",
            ),
            // 56 bytes: padding spills into a second block.
            (
                b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "b35439a4ac6f0948b6d6f9e3c6af0f5f590ce20f1bde7090ef7970686ec6738a",
            ),
            // 64 bytes: exactly one full block plus padding block.
            (
                b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb",
            ),
        ];
        for (input, want) in vectors {
            let mut h = Sha256::new();
            // Feed one byte at a time to stress the streaming buffer.
            for b in input.iter() {
                h.update(std::slice::from_ref(b));
            }
            assert_eq!(&hex_encode(&h.finish()), want);
        }
    }

    #[test]
    fn go_quoted_vectors() {
        assert_eq!(go_quoted("plain"), "\"plain\"");
        assert_eq!(
            go_quoted("a\x01b\x7f\u{80}é\"\\"),
            "\"a\\x01b\\x7f\\u0080é\\\"\\\\\""
        );
        assert_eq!(go_quoted("\u{a0}"), "\"\\u00a0\"");
        assert_eq!(
            go_quoted("\u{200b}\u{2028}\u{e000}\u{f0000}"),
            "\"\\u200b\\u2028\\ue000\\U000f0000\""
        );
        assert_eq!(go_quoted("\n\r\t"), "\"\\n\\r\\t\"");
        assert_eq!(go_quoted("\u{7}\u{8}\u{c}\u{b}"), "\"\\a\\b\\f\\v\"");
        assert_eq!(go_quoted("space kept"), "\"space kept\"");
    }

    #[test]
    fn errno_table_spot_checks() {
        assert_eq!(go_errno(libc::ENOENT), "no such file or directory");
        assert_eq!(go_errno(libc::EACCES), "permission denied");
        assert_eq!(go_errno(libc::EPIPE), "broken pipe");
        assert_eq!(go_errno(libc::ENOSYS), "function not implemented");
        assert_eq!(go_errno(libc::ELOOP), "too many levels of symbolic links");
        assert_eq!(go_errno(libc::EINVAL), "invalid argument");
        assert_eq!(go_errno(libc::EISDIR), "is a directory");
        assert_eq!(go_errno(9999), "errno 9999");
    }

    #[test]
    fn config_read_error_shapes() {
        let dir = TestDir::make("cfgread");
        let missing = dir.path("nope.json");
        assert_eq!(
            load_config(&missing).unwrap_err(),
            format!("open {missing}: no such file or directory")
        );
        let sub = dir.path("sub");
        fs::create_dir(&sub).unwrap();
        assert_eq!(
            load_config(&sub).unwrap_err(),
            format!("read {sub}: is a directory")
        );
        // Decode runs before the release lookup.
        let bad = dir.path("bad.json");
        fs::write(&bad, b"{oops").unwrap();
        assert_eq!(
            load_config(&bad).unwrap_err(),
            "invalid character 'o' looking for beginning of object key string"
        );
    }

    #[test]
    fn go_path_helpers() {
        assert_eq!(
            go_clean("/usr/libexec/soda/../../share/soda/muse-tools"),
            "/usr/share/soda/muse-tools"
        );
        assert_eq!(go_clean("a//b/./c/"), "a/b/c");
        assert_eq!(go_clean(""), ".");
        assert_eq!(go_clean("/"), "/");
        assert_eq!(go_base("/x/launch.sock"), "launch.sock");
        assert_eq!(go_base("/x/launch.sock/"), "launch.sock");
        assert_eq!(go_base("/"), "/");
        assert_eq!(go_base(""), ".");
        assert_eq!(go_dir("/x/launch.sock"), "/x");
        assert_eq!(go_dir("/launch.sock"), "/");
        assert_eq!(go_dir("/x/launch.sock/"), "/x/launch.sock");
        assert_eq!(default_tools(), "/usr/share/soda/muse-tools");
    }

    fn args(v: &[&str]) -> Vec<String> {
        v.iter().map(|s| s.to_string()).collect()
    }

    const PROJECT: &str = "p0123456789abcdef01234567";

    #[test]
    fn flag_parsing_vectors() {
        let o = parse(&args(&["--project", PROJECT])).unwrap();
        assert_eq!(o.config, "/etc/soda/host.json");
        assert_eq!(o.project, PROJECT);
        assert_eq!(o.tools, "/usr/share/soda/muse-tools");
        assert!(!o.bind_only);
        let o = parse(&args(&[
            "--config=/x/host.json",
            "--tools",
            "/y/tools",
            "--bind-only",
            "--project",
            PROJECT,
        ]))
        .unwrap();
        assert_eq!(o.config, "/x/host.json");
        assert_eq!(o.tools, "/y/tools");
        assert!(o.bind_only);
        let o = parse(&args(&["-project", PROJECT, "-bind-only=false"])).unwrap();
        assert!(!o.bind_only);
        let o = parse(&args(&["--project", PROJECT, "--", "--bind-only"]));
        assert!(o.is_err());
        // First non-flag argument stops parsing.
        assert!(parse(&args(&["--project", PROJECT, "extra"])).is_err());
        assert!(parse(&args(&["extra", "--project", PROJECT])).is_err());
        // Usage errors print usage and name the flag.
        assert_eq!(
            parse(&args(&["--project"])).unwrap_err(),
            "flag needs an argument: -project"
        );
        assert_eq!(
            parse(&args(&["--nope", "x", "--project", PROJECT])).unwrap_err(),
            "flag provided but not defined: -nope"
        );
        assert_eq!(
            parse(&args(&["---project", PROJECT])).unwrap_err(),
            "bad flag syntax: ---project"
        );
        assert_eq!(parse(&args(&["-=x"])).unwrap_err(), "bad flag syntax: -=x");
        assert_eq!(
            parse(&args(&["--bind-only=maybe"])).unwrap_err(),
            "invalid boolean value \"maybe\" for -bind-only: parse error"
        );
        assert_eq!(parse(&args(&["-h"])).unwrap_err(), "flag: help requested");
        assert_eq!(
            usage_text("/usr/share/soda/muse-tools"),
            "Usage of soda-muse-maintain:\n  -bind-only\n    \trestore only the launch interface\n  -config string\n    \toperator host configuration (default \"/etc/soda/host.json\")\n  -project string\n    \texact project identity\n  -tools string\n    \tinstalled public tool directory (default \"/usr/share/soda/muse-tools\")\n"
        );
        // Validation errors.
        assert_eq!(
            parse(&args(&["--project", "../foreign"])).unwrap_err(),
            "explicit project and absolute maintenance paths required"
        );
        assert_eq!(
            parse(&args(&["--project", "P0123456789ABCDEF01234567"])).unwrap_err(),
            "explicit project and absolute maintenance paths required"
        );
        assert_eq!(
            parse(&args(&["--project", PROJECT, "--config", "relative.json"])).unwrap_err(),
            "explicit project and absolute maintenance paths required"
        );
        assert_eq!(
            parse(&args(&[])).unwrap_err(),
            "explicit project and absolute maintenance paths required"
        );
    }

    #[test]
    fn host_config_decode_vectors() {
        assert_eq!(decode_host_config(b"").unwrap_err(), "EOF");
        assert_eq!(decode_host_config(b"   ").unwrap_err(), "EOF");
        let c = decode_host_config(b"null").unwrap();
        assert_eq!(c.image, "");
        assert_eq!(
            decode_host_config(b"[1,2]").unwrap_err(),
            "json: cannot unmarshal array into Go value of type host.Config"
        );
        assert_eq!(
            decode_host_config(b"\"str\"").unwrap_err(),
            "json: cannot unmarshal string into Go value of type host.Config"
        );
        assert_eq!(decode_host_config(b"[1,2").unwrap_err(), "unexpected EOF");
        // Last duplicate wins; keys match case-insensitively.
        let c = decode_host_config(br#"{"image": "a", "image": "b", "NETWORK": "n"}"#).unwrap();
        assert_eq!(c.image, "b");
        assert_eq!(c.network, "n");
        let c = decode_host_config(br#"{"tailnet_management": true}"#).unwrap();
        assert!(c.tailnet_management);
        let c = decode_host_config(br#"{"image": null, "tailnet_management": null}"#).unwrap();
        assert_eq!(c.image, "");
        assert!(!c.tailnet_management);
        // Type errors name the key as written.
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": 1}"#).unwrap_err(),
            "json: cannot unmarshal number into Go struct field Config.muse_sha256 of type string"
        );
        assert_eq!(
            decode_host_config(br#"{"MUSE_SHA256": 1}"#).unwrap_err(),
            "json: cannot unmarshal number into Go struct field Config.MUSE_SHA256 of type string"
        );
        assert_eq!(
            decode_host_config(br#"{"tailnet_management": "yes"}"#).unwrap_err(),
            "json: cannot unmarshal string into Go struct field Config.tailnet_management of type bool"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": [1,2]}"#).unwrap_err(),
            "json: cannot unmarshal array into Go struct field Config.muse_sha256 of type string"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": {"a":1}}"#).unwrap_err(),
            "json: cannot unmarshal object into Go struct field Config.muse_sha256 of type string"
        );
        // Unknown fields, including null-valued ones.
        assert_eq!(
            decode_host_config(br#"{"bogus": null}"#).unwrap_err(),
            "json: unknown field \"bogus\""
        );
        assert_eq!(
            decode_host_config(br#"{"quo\"te": 1}"#).unwrap_err(),
            "json: unknown field \"quo\\\"te\""
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": "x", "bogus": 1}"#).unwrap_err(),
            "json: unknown field \"bogus\""
        );
        // Broken values beat unknown fields; a later syntax error beats a
        // saved error; the first saved error beats later saved errors.
        assert_eq!(
            decode_host_config(br#"{"bogus": truX}"#).unwrap_err(),
            "invalid character 'X' in literal true (expecting 'e')"
        );
        assert_eq!(
            decode_host_config(br#"{"bogus": [1,2}"#).unwrap_err(),
            "invalid character '}' after array element"
        );
        assert_eq!(
            decode_host_config(br#"{"bogus": 1, "tailnet_management": truX}"#).unwrap_err(),
            "invalid character 'X' in literal true (expecting 'e')"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": 1, "tailnet_management": truX}"#).unwrap_err(),
            "invalid character 'X' in literal true (expecting 'e')"
        );
        assert_eq!(
            decode_host_config(br#"{"tailnet_management": "x", "muse_sha256": 1}"#).unwrap_err(),
            "json: cannot unmarshal string into Go struct field Config.tailnet_management of type bool"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": 1, "bogus": 2}"#).unwrap_err(),
            "json: cannot unmarshal number into Go struct field Config.muse_sha256 of type string"
        );
        assert_eq!(
            decode_host_config(br#"{"bogus": 2, "muse_sha256": 1}"#).unwrap_err(),
            "json: unknown field \"bogus\""
        );
        // Escapes and structural errors, byte-identical to encoding/json.
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": "a\qb"}"#).unwrap_err(),
            "invalid escape sequence `\\q` in string"
        );
        assert_eq!(
            decode_host_config(b"{\"muse_sha256\": \"a\x01b\"}").unwrap_err(),
            "invalid character '\\x01' in string"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": "\u00G1"}"#).unwrap_err(),
            "invalid escape sequence `\\u00G1` in string"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": "\uD800\u00G1"}"#).unwrap_err(),
            "invalid escape sequence `\\u00G1` in string"
        );
        let c = decode_host_config("{\"image\": \"a�b\"}".as_bytes()).unwrap();
        assert_eq!(c.image, "a�b");
        let c = decode_host_config(br#"{"image": "a\uD800b"}"#).unwrap();
        assert_eq!(c.image, "a\u{FFFD}b");
        let c = decode_host_config(br#"{"image": "a\uD800\u0041"}"#).unwrap();
        assert_eq!(c.image, "a\u{FFFD}A");
        assert_eq!(
            decode_host_config(b"{': 1}").unwrap_err(),
            "invalid character '\\'' looking for beginning of object key string"
        );
        assert_eq!(
            decode_host_config(br#"{"a": "b",}"#).unwrap_err(),
            "invalid character '}' looking for beginning of object key string"
        );
        assert_eq!(
            decode_host_config(b"{\"a\": \"b\"").unwrap_err(),
            "unexpected EOF"
        );
        assert_eq!(decode_host_config(b"nul").unwrap_err(), "unexpected EOF");
    }

    #[test]
    fn prefix_parity_vectors() {
        let cases: &[(&str, Option<&str>)] = &[
            ("10.0.0.0/24", None),
            ("10.0.0.0/0", None),
            ("10.0.0.0/32", None),
            ("::/0", None),
            ("::/128", None),
            ("fe80::1/128", None),
            ("::ffff:1.2.3.4/128", None),
            ("1:2:3:4:5:6:1.2.3.4/128", None),
            (
                "1.2.3.4/33",
                Some("netip.ParsePrefix(\"1.2.3.4/33\"): prefix length out of range"),
            ),
            (
                "::gggg/1",
                Some(
                    "netip.ParsePrefix(\"::gggg/1\"): ParseAddr(\"::gggg\"): each colon-separated field must have at least one digit (at \"gggg\")",
                ),
            ),
            ("1.2.3.4", Some("netip.ParsePrefix(\"1.2.3.4\"): no '/'")),
            (
                "1.2.3/24",
                Some("netip.ParsePrefix(\"1.2.3/24\"): ParseAddr(\"1.2.3\"): IPv4 address too short"),
            ),
            (
                "1.2.3.4.5/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4.5/24\"): ParseAddr(\"1.2.3.4.5\"): IPv4 address too long",
                ),
            ),
            (
                "fe80::1%eth0/64",
                Some(
                    "netip.ParsePrefix(\"fe80::1%eth0/64\"): IPv6 zones cannot be present in a prefix",
                ),
            ),
            (
                "1.2.3.4/ab",
                Some("netip.ParsePrefix(\"1.2.3.4/ab\"): bad bits after slash: \"ab\""),
            ),
            (
                "1.2.3.4/",
                Some("netip.ParsePrefix(\"1.2.3.4/\"): bad bits after slash: \"\""),
            ),
            (
                "/24",
                Some("netip.ParsePrefix(\"/24\"): ParseAddr(\"\"): unable to parse IP"),
            ),
            (
                "1.2.3.4/-1",
                Some("netip.ParsePrefix(\"1.2.3.4/-1\"): bad bits after slash: \"-1\""),
            ),
            (
                "::/129",
                Some("netip.ParsePrefix(\"::/129\"): prefix length out of range"),
            ),
            (
                "1::2::3/64",
                Some(
                    "netip.ParsePrefix(\"1::2::3/64\"): ParseAddr(\"1::2::3\"): multiple :: in address (at \":3\")",
                ),
            ),
            (
                "1.2.3.256/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.256/24\"): ParseAddr(\"1.2.3.256\"): IPv4 field has value >255",
                ),
            ),
            (
                "12345::/33",
                Some(
                    "netip.ParsePrefix(\"12345::/33\"): ParseAddr(\"12345::\"): each group must have 4 or less digits (at \"12345::\")",
                ),
            ),
            (
                "1.2.3.4/ 24",
                Some("netip.ParsePrefix(\"1.2.3.4/ 24\"): bad bits after slash: \" 24\""),
            ),
            (
                "1.2.3.4\x01/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4\\x01/24\"): ParseAddr(\"1.2.3.4\\x01\"): unexpected character (at \"\\x01\")",
                ),
            ),
            (
                "1.2.3.4/\x034",
                Some("netip.ParsePrefix(\"1.2.3.4/\\x034\"): bad bits after slash: \"\\x034\""),
            ),
            (
                "01.2.3.4/24",
                Some(
                    "netip.ParsePrefix(\"01.2.3.4/24\"): ParseAddr(\"01.2.3.4\"): IPv4 field has octet with leading zero",
                ),
            ),
            // Left-to-right order: the value error fires before the length check.
            (
                "1.2.3.99999.5/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.99999.5/24\"): ParseAddr(\"1.2.3.99999.5\"): IPv4 field has value >255",
                ),
            ),
            (
                "1..2.3/24",
                Some(
                    "netip.ParsePrefix(\"1..2.3/24\"): ParseAddr(\"1..2.3\"): IPv4 field must have at least one digit (at \".2.3\")",
                ),
            ),
            (
                "1.2.3./24",
                Some(
                    "netip.ParsePrefix(\"1.2.3./24\"): ParseAddr(\"1.2.3.\"): IPv4 field must have at least one digit (at \".\")",
                ),
            ),
            (
                "12g4::/64",
                Some(
                    "netip.ParsePrefix(\"12g4::/64\"): ParseAddr(\"12g4::\"): unexpected character, want colon (at \"g4::\")",
                ),
            ),
            (
                "1.2.3.4::/64",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4::/64\"): ParseAddr(\"1.2.3.4::\"): unexpected character (at \"::\")",
                ),
            ),
            (
                "1:2.3.4.5/24",
                Some(
                    "netip.ParsePrefix(\"1:2.3.4.5/24\"): ParseAddr(\"1:2.3.4.5\"): embedded IPv4 address must replace the final 2 fields of the address (at \"2.3.4.5\")",
                ),
            ),
            (
                "::1:2:3:4:5:6:7:1.2.3.4/64",
                Some(
                    "netip.ParsePrefix(\"::1:2:3:4:5:6:7:1.2.3.4/64\"): ParseAddr(\"::1:2:3:4:5:6:7:1.2.3.4\"): too many hex fields to fit an embedded IPv4 at the end of the address (at \"1.2.3.4\")",
                ),
            ),
            (
                "::ffff:1.2.3.999/64",
                Some(
                    "netip.ParsePrefix(\"::ffff:1.2.3.999/64\"): ParseAddr(\"::ffff:1.2.3.999\"): IPv4 field has value >255",
                ),
            ),
            (
                "fe80::1%/64",
                Some(
                    "netip.ParsePrefix(\"fe80::1%/64\"): ParseAddr(\"fe80::1%\"): zone must be a non-empty string",
                ),
            ),
            (
                "abcd%eth0/64",
                Some(
                    "netip.ParsePrefix(\"abcd%eth0/64\"): ParseAddr(\"abcd%eth0\"): missing IPv6 address",
                ),
            ),
            (
                "1.2.3.4%eth0/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4%eth0/24\"): ParseAddr(\"1.2.3.4%eth0\"): unexpected character (at \"%eth0\")",
                ),
            ),
            (
                "1:2:3:4:5:6:7:8:9/64",
                Some(
                    "netip.ParsePrefix(\"1:2:3:4:5:6:7:8:9/64\"): ParseAddr(\"1:2:3:4:5:6:7:8:9\"): trailing garbage after address (at \"9\")",
                ),
            ),
            (
                "1:2:3/64",
                Some(
                    "netip.ParsePrefix(\"1:2:3/64\"): ParseAddr(\"1:2:3\"): address string too short",
                ),
            ),
            (
                "::1:2:3:4:5:6:7:8/64",
                Some(
                    "netip.ParsePrefix(\"::1:2:3:4:5:6:7:8/64\"): ParseAddr(\"::1:2:3:4:5:6:7:8\"): the :: must expand to at least one field of zeros",
                ),
            ),
            (
                "1.2.3.4/00",
                Some("netip.ParsePrefix(\"1.2.3.4/00\"): bad bits after slash: \"00\""),
            ),
            (
                "1.2.3.4/+5",
                Some("netip.ParsePrefix(\"1.2.3.4/+5\"): bad bits after slash: \"+5\""),
            ),
            (
                "1.2.3.4/99999999999999999999999",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4/99999999999999999999999\"): bad bits after slash: \"99999999999999999999999\"",
                ),
            ),
        ];
        for (input, expected) in cases {
            match (parse_prefix(input), expected) {
                (Ok(()), None) => {}
                (Err(got), Some(want)) => assert_eq!(&got, want, "input {input:?}"),
                (Ok(()), Some(want)) => panic!("input {input:?}: accepted, want {want}"),
                (Err(got), None) => panic!("input {input:?}: rejected: {got}"),
            }
        }
    }

    fn hex_string(c: char, len: usize) -> String {
        std::iter::repeat_n(c, len).collect()
    }

    fn valid_release_json() -> (String, String, String) {
        let rev = hex_string('b', 40);
        let coreos = "41.20250101.3.0";
        let id = format!("{coreos}.soda-{}", &rev[..12]);
        let prefix = "ghcr.io/test/soda";
        let mut images = String::from("{");
        let mut cfgs = std::collections::HashMap::new();
        for (i, name) in [
            "dashboard",
            "forgejo",
            "extension",
            "proxy",
            "project-os",
            "tailnet",
        ]
        .iter()
        .enumerate()
        {
            let c = char::from_digit(i as u32 + 1, 16).unwrap();
            let m = char::from_digit(i as u32 + 7, 16).unwrap();
            let cfg = format!("sha256:{}", hex_string(c, 64));
            let man = format!("sha256:{}", hex_string(m, 64));
            let arch = hex_string('d', 64);
            let reference = format!("{prefix}-{name}@{man}");
            images.push_str(&format!(
                "\"{name}\":{{\"Reference\":\"{reference}\",\"Config\":\"{cfg}\",\"Manifest\":\"{man}\",\"ArchiveSHA256\":\"{arch}\"}},"
            ));
            cfgs.insert(*name, cfg);
        }
        images.pop();
        images.push('}');
        let base = format!(
            "quay.io/fedora/fedora-coreos@sha256:{}",
            hex_string('0', 64)
        );
        let json = format!(
            "{{\"Format\":3,\"ID\":\"{id}\",\"Revision\":\"{rev}\",\"Architecture\":\"x86_64\",\"CoreOS\":\"{coreos}\",\"Base\":\"{base}\",\"RepositoryPrefix\":\"{prefix}\",\"Schema\":1,\"PresentationSHA256\":\"{}\",\"HostPackagesSHA256\":\"{}\",\"Images\":{images},\"UpgradeFrom\":[]}}",
            hex_string('e', 64),
            hex_string('f', 64),
        );
        (json, cfgs["project-os"].clone(), cfgs["tailnet"].clone())
    }

    #[test]
    fn release_payload_accept_and_reject() {
        let dir = TestDir::make("release");
        let (valid, project_cfg, tailnet_cfg) = valid_release_json();
        let path = dir.path("release.json");
        fs::write(&path, &valid).unwrap();
        let payload = load_release_payload(&path).expect("valid payload refused");
        assert_eq!(payload.architecture, "x86_64");
        assert_eq!(
            payload.image_config("project-os").as_deref(),
            Some(project_cfg.as_str())
        );
        assert_eq!(
            payload.image_config("tailnet").as_deref(),
            Some(tailnet_cfg.as_str())
        );
        // Each mutation is rejected.
        let forgejo_cfg = format!("sha256:{}", hex_string('2', 64));
        let ext_cfg = format!("sha256:{}", hex_string('3', 64));
        let mutations: Vec<(&str, String)> = vec![
            ("format", valid.replacen("\"Format\":3", "\"Format\":2", 1)),
            (
                "revision",
                valid.replacen(&hex_string('b', 40), &hex_string('b', 39), 1),
            ),
            ("id", valid.replacen(".soda-bbbbbbbbbbbb", ".soda-cccccccccccc", 1)),
            ("coreos", valid.replacen("41.20250101.3.0", "41.1", 1)),
            ("arch", valid.replacen("\"x86_64\"", "\"aarch64\"", 1)),
            ("prefix", valid.replacen("ghcr.io/test/soda", "docker.io/test/soda", 1)),
            ("schema", valid.replacen("\"Schema\":1", "\"Schema\":0", 1)),
            (
                "presentation",
                valid.replacen(&hex_string('e', 64), &format!("g{}", hex_string('e', 63)), 1),
            ),
            ("reference", valid.replacen("-project-os@sha256:", "-project-os@sha257:", 1)),
            ("ext-forgejo", valid.replacen(&ext_cfg, &forgejo_cfg, 1)),
            ("missing-name", valid.replacen("\"tailnet\":{", "\"tailnet2\":{", 1)),
            (
                "seventh",
                valid.replacen("\"UpgradeFrom\"", "\"extra\":{\"Reference\":\"\",\"Config\":\"\",\"Manifest\":\"\",\"ArchiveSHA256\":\"\"},\"UpgradeFrom", 1),
            ),
            ("unknown", valid.replacen("\"Format\":3,", "\"Format\":3,\"Bogus\":1,", 1)),
            ("trailing", format!("{valid} {{}}")),
            ("trailing-comma", valid.replacen("\"UpgradeFrom\":[]}", "\"UpgradeFrom\":[],}", 1)),
            (
                "null-images",
                valid.replacen(
                    &valid[valid.find("\"Images\":").unwrap()..valid.find(",\"UpgradeFrom\"").unwrap()],
                    "\"Images\":null",
                    1,
                ),
            ),
        ];
        for (tag, bad) in &mutations {
            let p = dir.path(&format!("bad-{tag}.json"));
            fs::write(&p, bad).unwrap();
            assert!(
                load_release_payload(&p).is_err(),
                "mutation accepted: {tag}"
            );
        }
        // Duplicates apply last-wins; null UpgradeFrom stays empty.
        let dup = valid.replacen("\"Format\":3,", "\"Format\":2,\"Format\":3,", 1);
        let p = dir.path("dup.json");
        fs::write(&p, &dup).unwrap();
        assert!(load_release_payload(&p).is_ok(), "duplicate rejected");
        let dup_image = valid.replacen(
            "\"Images\":{\"dashboard\":{",
            "\"Images\":{\"dashboard\":{\"Reference\":\"\",\"Config\":\"\",\"Manifest\":\"\",\"ArchiveSHA256\":\"\"},\"dashboard\":{",
            1,
        );
        let p = dir.path("dup-image.json");
        fs::write(&p, &dup_image).unwrap();
        assert!(load_release_payload(&p).is_ok(), "duplicate image rejected");
        let null_upgrade = valid.replacen("\"UpgradeFrom\":[]", "\"UpgradeFrom\":null", 1);
        let p = dir.path("null-upgrade.json");
        fs::write(&p, &null_upgrade).unwrap();
        assert!(
            load_release_payload(&p).is_ok(),
            "null UpgradeFrom rejected"
        );
        // Shape rejections: symlink, missing, directory, oversize.
        let link = dir.path("link.json");
        std::os::unix::fs::symlink(&path, &link).unwrap();
        assert!(load_release_payload(&link).is_err(), "symlink accepted");
        assert!(load_release_payload(&dir.path("missing.json")).is_err());
        assert!(load_release_payload(&dir.dir_str()).is_err());
        let big = dir.path("big.json");
        fs::write(&big, vec![b' '; (4 << 20) + 1]).unwrap();
        assert!(load_release_payload(&big).is_err(), "oversize accepted");
    }

    #[test]
    fn release_images_apply_and_conflict() {
        let dir = TestDir::make("relapply");
        let (valid, project_cfg, tailnet_cfg) = valid_release_json();
        let path = dir.path("release.json");
        fs::write(&path, &valid).unwrap();
        let mut c = Config::default();
        apply_release_images(&mut c, &path).unwrap();
        assert_eq!(c.image, project_cfg);
        assert_eq!(c.tailnet_image, "");
        c.tailnet_management = true;
        apply_release_images(&mut c, &path).unwrap();
        assert_eq!(c.tailnet_image, tailnet_cfg);
        c.image = String::from("other");
        assert_eq!(
            apply_release_images(&mut c, &path).unwrap_err(),
            "saved image selection conflicts with appliance release; explicit migration required"
        );
        let mut c = Config::default();
        assert_eq!(
            apply_release_images(&mut c, &dir.path("missing.json")).unwrap_err(),
            "immutable appliance image defaults unavailable"
        );
    }

    #[test]
    fn runtime_config_validation_order() {
        // Muse errors precede identity, subnet, tailnet, and network errors.
        let c = Config {
            muse_sha256: hex_string('a', 64),
            subnet: String::from("bogus"),
            ..Default::default()
        };
        assert_eq!(
            validate_runtime_config(&c).unwrap_err(),
            "explicit muse socket, broker socket and release digest required"
        );
        // Subnet errors precede tailnet and network errors.
        let c = Config {
            subnet: String::from("bogus"),
            ..Default::default()
        };
        assert!(validate_runtime_config(&c)
            .unwrap_err()
            .starts_with("netip.ParsePrefix"));
        // Tailnet errors precede network errors.
        let c = Config {
            subnet: String::from("10.0.0.0/24"),
            tailnet_image: String::from("bare-hex"),
            ..Default::default()
        };
        assert_eq!(
            validate_runtime_config(&c).unwrap_err(),
            "invalid immutable Tailnet companion configuration"
        );
        // Network errors come last.
        let c = Config {
            subnet: String::from("10.0.0.0/24"),
            ..Default::default()
        };
        assert_eq!(
            validate_runtime_config(&c).unwrap_err(),
            "invalid native runtime configuration"
        );
        // A fully valid config passes.
        let c = Config {
            image: String::from("img"),
            network: String::from("net"),
            bridge: String::from("br"),
            subnet: String::from("10.0.0.0/24"),
            ..Default::default()
        };
        validate_runtime_config(&c).unwrap();
    }

    #[test]
    fn observation_decode_and_validate() {
        let id = hex_string('a', 64);
        let body = format!(
            "{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":123,\"running\":true}}"
        );
        let o = decode_observation(body.as_bytes()).unwrap();
        assert_eq!(o.pid, 123);
        assert!(o.running);
        validate_observation(&o, PROJECT).unwrap();
        // Identity failures.
        let mut foreign = Observation {
            id: id.clone(),
            project: String::from("pabcdef0123456789abcdef01"),
            owner: String::from("42"),
            pid: 1,
            running: true,
        };
        assert_eq!(
            validate_observation(&foreign, PROJECT).unwrap_err(),
            "project container identity differs"
        );
        foreign.project = PROJECT.to_string();
        foreign.id = hex_string('A', 64);
        assert_eq!(
            validate_observation(&foreign, PROJECT).unwrap_err(),
            "project container identity differs"
        );
        foreign.id = hex_string('a', 63);
        assert_eq!(
            validate_observation(&foreign, PROJECT).unwrap_err(),
            "project container identity differs"
        );
        // Owner failures and edge acceptances.
        foreign.id = id.clone();
        for owner in ["0", "-5", "abc", "", "42 ", "99999999999999999999999"] {
            foreign.owner = owner.to_string();
            assert_eq!(
                validate_observation(&foreign, PROJECT).unwrap_err(),
                "project owner label is invalid",
                "owner {owner:?}"
            );
        }
        foreign.owner = String::from("+42");
        validate_observation(&foreign, PROJECT).unwrap();
        foreign.owner = String::from("42");
        foreign.pid = -5;
        validate_observation(&foreign, PROJECT).unwrap();
        // Decode failures collapse to the observation message.
        for bad in [
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1.5,\"running\":true}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":\"12\",\"running\":true}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":1}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":\"true\"}}"),
            format!("{{\"id\":\"{id}\",\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true,\"bogus\":1}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true,}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}} {{}}"),
            String::from("[]"),
            String::from("null"),
            String::from("{oops"),
        ] {
            assert_eq!(
                decode_observation(bad.as_bytes()).unwrap_err(),
                "invalid project observation",
                "input {bad:?}"
            );
        }
        assert_eq!(
            decode_observation(&vec![b'{'; 2 << 20]).unwrap_err(),
            "invalid project observation"
        );
        assert_eq!(
            decode_observation(b"{\"id\":\"\xff\"}").unwrap_err(),
            "invalid project observation"
        );
        // Nulls decode to zero values, then fail validation.
        let nulls = decode_observation(
            format!("{{\"id\":null,\"project\":\"{PROJECT}\",\"owner\":null,\"pid\":null,\"running\":null}}")
                .as_bytes(),
        )
        .unwrap();
        assert_eq!(
            validate_observation(&nulls, PROJECT).unwrap_err(),
            "project container identity differs"
        );
        // Exact keys win over earlier case variants in sorted order.
        let other = hex_string('b', 64);
        let o = decode_observation(
            format!("{{\"ID\":\"{other}\",\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}}")
                .as_bytes(),
        )
        .unwrap();
        assert_eq!(o.id, id);
        validate_observation(&o, PROJECT).unwrap();
    }

    #[test]
    fn tool_source_and_digest_admission() {
        let dir = TestDir::make("tools");
        let native = dir.path("muse-native");
        let body = b"synthetic native bytes";
        fs::write(&native, body).unwrap();
        fs::set_permissions(&native, fs::Permissions::from_mode(0o755)).unwrap();
        // Symlinks never admit, regardless of owner.
        let link = dir.path("muse");
        std::os::unix::fs::symlink(&native, &link).unwrap();
        assert_eq!(
            open_tool(&dir.dir_str(), "muse").unwrap_err(),
            "public tool source must be a regular root-owned executable"
        );
        // Group-writable and non-executable sources refuse alike.
        let weak = dir.path("weak");
        fs::write(&weak, body).unwrap();
        fs::set_permissions(&weak, fs::Permissions::from_mode(0o775)).unwrap();
        assert!(open_tool(&dir.dir_str(), "weak").is_err());
        fs::set_permissions(&weak, fs::Permissions::from_mode(0o644)).unwrap();
        assert!(open_tool(&dir.dir_str(), "weak").is_err());
        // Digest admission through an open tool fd.
        let c = CString::new(native.clone()).unwrap();
        let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
        assert!(fd >= 0);
        let tool = Tool {
            name: String::from("muse-native"),
            fd,
            size: body.len() as u64,
        };
        assert_eq!(
            verify_native(&tool, &hex_string('0', 64)).unwrap_err(),
            "muse native digest mismatch"
        );
        let mut h = Sha256::new();
        h.update(body);
        verify_native(&tool, &hex_encode(&h.finish())).unwrap();
        // pread verification leaves the offset at zero.
        assert_eq!(unsafe { libc::lseek(fd, 0, libc::SEEK_CUR) }, 0);
        drop(tool);
        if is_root() {
            open_tool(&dir.dir_str(), "muse-native").expect("root-owned source refused");
        } else {
            assert!(open_tool(&dir.dir_str(), "muse-native").is_err());
        }
        // The version gate fires before any tool opens.
        assert_eq!(
            load_tools(
                "definitely-missing",
                &hex_string('0', 64),
                &format!("{MUSE_VERSION}-other")
            )
            .unwrap_err(),
            "muse maintenance version differs from pinned release"
        );
    }

    #[test]
    fn public_interface_admission() {
        use std::os::unix::net::UnixListener;
        let dir = TestDir::make("sock");
        let sock = dir.path("launch.sock");
        let listener = UnixListener::bind(&sock).unwrap();
        fs::set_permissions(&sock, fs::Permissions::from_mode(0o666)).unwrap();
        fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o755)).unwrap();
        if is_root() {
            assert_eq!(public_socket_directory(&sock).unwrap(), dir.dir_str());
            fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o775)).unwrap();
            assert_eq!(
                public_socket_directory(&sock).unwrap_err(),
                "public launch directory must be a protected directory"
            );
            fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o700)).unwrap();
            assert_eq!(
                public_socket_directory(&sock).unwrap_err(),
                "public launch directory must be root-owned and accessible"
            );
            fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o755)).unwrap();
        }
        assert_eq!(
            public_socket_directory("relative/launch.sock").unwrap_err(),
            "explicit public launch socket required"
        );
        assert_eq!(
            public_socket_directory("/tmp/other.sock").unwrap_err(),
            "explicit public launch socket required"
        );
        fs::write(dir.path("credential"), b"synthetic private data").unwrap();
        assert_eq!(
            public_socket_directory(&sock).unwrap_err(),
            "launch directory must contain only the public socket"
        );
        fs::remove_file(dir.path("credential")).unwrap();
        // Same message whether the uid or the mode check fires.
        fs::set_permissions(&sock, fs::Permissions::from_mode(0o600)).unwrap();
        assert_eq!(
            public_socket_directory(&sock).unwrap_err(),
            "public launch socket must be root-owned and public"
        );
        fs::set_permissions(&sock, fs::Permissions::from_mode(0o666)).unwrap();
        drop(listener);
        fs::remove_file(&sock).unwrap();
        fs::write(&sock, b"not a socket").unwrap();
        assert_eq!(
            public_socket_directory(&sock).unwrap_err(),
            "public launch socket is unavailable"
        );
    }

    #[test]
    fn tar_header_layout_matches_ustar() {
        let h = tar_header("muse", 14).unwrap();
        assert_eq!(&h[..4], b"muse");
        assert!(h[4..100].iter().all(|b| *b == 0));
        assert_eq!(&h[100..108], b"0000755\0");
        assert_eq!(&h[108..116], b"0000000\0");
        assert_eq!(&h[116..124], b"0000000\0");
        assert_eq!(&h[124..136], b"00000000016\0");
        assert_eq!(&h[136..148], b"00000000000\0");
        assert_eq!(h[156], b'0');
        assert_eq!(&h[257..263], b"ustar\0");
        assert_eq!(&h[263..265], b"00");
        let mut sum: u32 = 0;
        for (i, b) in h.iter().enumerate() {
            sum += if (148..156).contains(&i) {
                b' ' as u32
            } else {
                *b as u32
            };
        }
        assert_eq!(&h[148..156], format!("{sum:06o}\0 ").as_bytes());
        assert!(tar_header(&"n".repeat(101), 0).is_err());
    }

    fn synthetic_feeds(tools_dir: &TestDir) -> (Vec<(String, RawFd, u64)>, Vec<FdGuard>) {
        let mut feeds = Vec::new();
        let mut guards = Vec::new();
        for name in ["muse", "soda-identity-compose", "muse-native"] {
            let p = tools_dir.path(name);
            fs::write(&p, format!("synthetic {name}")).unwrap();
            let c = CString::new(p).unwrap();
            let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
            assert!(fd >= 0);
            let len = format!("synthetic {name}").len() as u64;
            guards.push(FdGuard(fd));
            feeds.push((name.to_string(), fd, len));
        }
        (feeds, guards)
    }

    fn emit_synthetic() -> Vec<u8> {
        let tools_dir = TestDir::make("emit");
        let (feeds, guards) = synthetic_feeds(&tools_dir);
        let mut archive = Vec::new();
        emit_archive(
            &mut |b: &[u8]| {
                archive.extend_from_slice(b);
                Ok(())
            },
            &feeds,
        )
        .unwrap();
        drop(guards);
        archive
    }

    fn run_install_script(archive: &[u8], targets: &[String]) -> (bool, String) {
        let mut cmd = Command::new("/bin/sh");
        cmd.args(["-ceu", INSTALL_SCRIPT, "soda-muse-maintain"]);
        for t in targets {
            cmd.arg(t);
        }
        cmd.stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        let mut child = cmd.spawn().unwrap();
        child.stdin.take().unwrap().write_all(archive).unwrap();
        let out = child.wait_with_output().unwrap();
        (
            out.status.success(),
            String::from_utf8_lossy(&out.stderr).into_owned(),
        )
    }

    #[test]
    fn tool_replacement_preserves_other_files_and_refuses_symlinks() {
        let dir = TestDir::make("stage");
        let targets: Vec<String> = ["bin/muse", "bin/soda-identity-compose", "libexec/soda/muse"]
            .iter()
            .map(|t| dir.path(t))
            .collect();
        for t in &targets {
            fs::create_dir_all(std::path::Path::new(t).parent().unwrap()).unwrap();
            fs::write(t, b"obsolete public tool").unwrap();
        }
        let retained = dir.path("account-state");
        fs::write(&retained, b"preserved account and project state").unwrap();
        fs::set_permissions(&retained, fs::Permissions::from_mode(0o600)).unwrap();
        let before = fs::symlink_metadata(&retained).unwrap();
        let archive = emit_synthetic();
        // 3 headers + 3 data blocks + 2 trailer blocks.
        assert_eq!(archive.len(), 3 * 512 + 3 * 512 + 1024);
        // The install script is byte-pinned where transcription once dropped
        // the space before the first loop's done.
        assert!(INSTALL_SCRIPT.contains("fi\n done\nfor target"));
        let (ok, stderr) = run_install_script(&archive, &targets);
        assert!(ok, "public replacement failed: {stderr}");
        let after = fs::symlink_metadata(&retained).unwrap();
        assert_eq!((before.dev(), before.ino()), (after.dev(), after.ino()));
        assert_eq!(
            fs::read(&retained).unwrap(),
            b"preserved account and project state"
        );
        for t in &targets {
            let body = fs::read(t).unwrap();
            assert!(body.starts_with(b"synthetic "), "tool not replaced: {t}");
        }
        fs::remove_file(&targets[0]).unwrap();
        std::os::unix::fs::symlink(&retained, &targets[0]).unwrap();
        let (ok, _) = run_install_script(&archive, &targets);
        assert!(!ok, "target symlink accepted");
        assert_eq!(
            fs::read(&retained).unwrap(),
            b"preserved account and project state"
        );
    }

    #[test]
    fn streamed_stage_delivery() {
        // A fake podman proves the feeder/child contract: the child must
        // not inherit the stdin write end, or its reader never sees EOF.
        // (A raw pipe without O_CLOEXEC deadlocked here until the deadline.)
        let dir = TestDir::make("streamed");
        let bindir = dir.path("bin");
        fs::create_dir(&bindir).unwrap();
        fs::write(
            format!("{bindir}/podman"),
            "#!/bin/sh\nprintf \"%s\\0\" \"$@\" > \"$E2E/argv.bin\"\ncat > \"$E2E/stdin.bin\"\nexit 0\n",
        )
        .unwrap();
        fs::set_permissions(
            format!("{bindir}/podman"),
            fs::Permissions::from_mode(0o755),
        )
        .unwrap();
        std::env::set_var(
            "PATH",
            format!("{bindir}:{}", std::env::var("PATH").unwrap()),
        );
        std::env::set_var("E2E", dir.dir_str());
        let tools = TestDir::make("streamed-tools");
        let (feeds, guards) = synthetic_feeds(&tools);
        let deadline = Instant::now() + Duration::from_secs(15);
        podman_streamed(
            move |w| feed_archive(w, &feeds, deadline),
            &["exec", "--user", "0:0", "-i", "some-id"],
            deadline,
        )
        .unwrap();
        drop(guards);
        assert_eq!(fs::read(dir.path("stdin.bin")).unwrap(), emit_synthetic());
        let argv = fs::read(dir.path("argv.bin")).unwrap();
        assert!(argv.starts_with(b"--remote=false\0exec\0--user\0"));
    }

    #[test]
    fn wait_output_reports_status() {
        let c = Command::new("/bin/sh")
            .args(["-c", "echo hello; exit 0"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .unwrap();
        let out = wait_output(c, Instant::now() + Duration::from_secs(5)).unwrap();
        assert_eq!(out, b"hello\n");
        let c = Command::new("/bin/sh")
            .args(["-c", "exit 3"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .unwrap();
        assert_eq!(
            wait_output(c, Instant::now() + Duration::from_secs(5)).unwrap_err(),
            "project maintenance command failed"
        );
    }

    #[test]
    fn short_tool_file_ends_copy_with_eof() {
        let dir = TestDir::make("short");
        let p = dir.path("tool");
        fs::write(&p, b"twelve bytes").unwrap();
        let c = CString::new(p).unwrap();
        let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
        assert!(fd >= 0);
        let _guard = FdGuard(fd);
        // Claim more bytes than the file holds, like a shrink race.
        let feeds = vec![(String::from("tool"), fd, 1_000_000u64)];
        let mut out = Vec::new();
        assert_eq!(
            emit_archive(
                &mut |b: &[u8]| {
                    out.extend_from_slice(b);
                    Ok(())
                },
                &feeds,
            )
            .unwrap_err(),
            "EOF"
        );
    }
}
