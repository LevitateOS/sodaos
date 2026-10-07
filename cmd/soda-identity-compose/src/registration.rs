use super::launch_wire::launch_request_json;
use super::launch_wire::parse_launch_exit;
use super::launch_wire::NestedRegistration;
use super::MUSE_LAUNCH_SOCKET;
use std::ffi::CString;
use std::fs;
use std::io;
use std::os::unix::fs::DirBuilderExt;

const TMPFS_MAGIC: i64 = 0x01021994;

pub(crate) fn registration_root() -> Result<(String, String), String> {
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
    getrandom::fill(buf).map_err(|e| e.to_string())
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
    let mut builder = fs::DirBuilder::new();
    builder.recursive(true).mode(mode);
    builder.create(path).map_err(|error| error.to_string())
}

pub(crate) fn account(login: &str) -> Result<i64, String> {
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

pub(crate) fn register(request: NestedRegistration) -> Result<(), String> {
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
    let data = launch_request_json(&request);
    let bytes = data.as_bytes();
    let n = unsafe { libc::send(fd, bytes.as_ptr() as *const libc::c_void, bytes.len(), 0) };
    if n < 0 || (n as usize) != bytes.len() {
        return Err(io::Error::last_os_error().to_string());
    }
    let mut body = [0u8; 4096];
    let r = unsafe { libc::recv(fd, body.as_mut_ptr() as *mut libc::c_void, body.len(), 0) };
    if r <= 0 {
        return Err(String::from("identity registration unconfirmed"));
    }
    let (code, err_text) = parse_launch_exit(&body[..r as usize])
        .map_err(|_| String::from("identity registration rejected"))?;
    if code != 0 || !err_text.is_empty() {
        return Err(String::from("identity registration rejected"));
    }
    Ok(())
}
