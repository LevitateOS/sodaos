//! Singleton and exact stale socket recovery after an ungraceful host exit.

use std::ffi::CString;
use std::fs::File;
use std::os::fd::{AsRawFd, FromRawFd};
use std::os::unix::fs::{FileTypeExt, MetadataExt, PermissionsExt};
use std::path::{Component, Path};

fn owned_directory(path: &Path) -> Result<File, String> {
    let mut directory = File::open("/").map_err(|_| "muse interface root unavailable")?;
    for component in path.components() {
        let Component::Normal(name) = component else {
            if component == Component::RootDir {
                continue;
            }
            return Err("muse interface path is not clean".into());
        };
        let name =
            CString::new(name.as_encoded_bytes()).map_err(|_| "invalid muse interface path")?;
        let fd = unsafe {
            libc::openat(
                directory.as_raw_fd(),
                name.as_ptr(),
                libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            )
        };
        if fd < 0 {
            return Err("muse interface directory unavailable".into());
        }
        directory = unsafe { File::from_raw_fd(fd) };
        let meta = directory
            .metadata()
            .map_err(|_| "muse interface directory metadata unavailable")?;
        if (meta.uid() != 0 && meta.uid() != unsafe { libc::geteuid() })
            || meta.permissions().mode() & 0o022 != 0
        {
            return Err("muse interface directory has an untrusted ancestor".into());
        }
    }
    Ok(directory)
}

fn singleton(parent: &File, path: &Path) -> Result<File, String> {
    let digest =
        crate::sha256::hex_lower(&crate::sha256::digest(path.as_os_str().as_encoded_bytes()));
    let name = CString::new(format!(".soda-muse-listener-{}.lock", &digest[..32])).unwrap();
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            name.as_ptr(),
            libc::O_RDWR | libc::O_CREAT | libc::O_NOFOLLOW | libc::O_CLOEXEC | libc::O_NONBLOCK,
            0o600,
        )
    };
    if fd < 0 {
        return Err("muse listener singleton unavailable".into());
    }
    let file = unsafe { File::from_raw_fd(fd) };
    let meta = file
        .metadata()
        .map_err(|_| "muse listener singleton metadata unavailable")?;
    if !meta.is_file()
        || meta.uid() != unsafe { libc::geteuid() }
        || meta.nlink() != 1
        || meta.permissions().mode() & 0o077 != 0
    {
        return Err("muse listener singleton is not service-owned".into());
    }
    if unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
        return Err("muse launch listener is already active".into());
    }
    Ok(file)
}

fn stale_socket(path: &Path, metadata: &std::fs::Metadata) -> Result<(), String> {
    if !metadata.file_type().is_socket()
        || metadata.uid() != unsafe { libc::geteuid() }
        || metadata.nlink() != 1
        || metadata.permissions().mode() & 0o777 != 0o666
    {
        return Err("muse launch socket is not the retired service endpoint".into());
    }
    let socket = rustix::net::socket_with(
        rustix::net::AddressFamily::UNIX,
        rustix::net::SocketType::SEQPACKET,
        rustix::net::SocketFlags::CLOEXEC | rustix::net::SocketFlags::NONBLOCK,
        None,
    )
    .map_err(|_| "muse launch socket observation failed")?;
    let address =
        rustix::net::SocketAddrUnix::new(path).map_err(|_| "invalid muse launch socket path")?;
    match rustix::net::connect(&socket, &address) {
        Err(rustix::io::Errno::CONNREFUSED) => Ok(()),
        _ => Err("muse launch socket is occupied or live".into()),
    }
}

/// The singleton file is outside the public Project mount. Unknown entries
/// remain untouched. Only a sole exact owned socket proven refused is retired.
pub(super) fn prepare(socket_path: &str) -> Result<File, String> {
    let socket = Path::new(socket_path);
    let interface = socket
        .parent()
        .filter(|p| !p.as_os_str().is_empty())
        .ok_or("invalid muse interface path")?;
    std::fs::create_dir_all(interface).map_err(|_| "muse interface directory unavailable")?;
    if !std::fs::symlink_metadata(interface)
        .map_err(|_| "muse interface directory unavailable")?
        .is_dir()
    {
        return Err("muse interface directory is redirected".into());
    }
    let canonical =
        std::fs::canonicalize(interface).map_err(|_| "muse interface directory unavailable")?;
    let parent = owned_directory(canonical.parent().ok_or("invalid muse interface parent")?)?;
    let directory = owned_directory(&canonical)?;
    let file_name = socket
        .file_name()
        .ok_or("invalid muse launch socket name")?;
    let socket_canonical = canonical.join(file_name);
    let lock = singleton(&parent, &socket_canonical)?;
    let entries: Vec<_> = std::fs::read_dir(&canonical)
        .map_err(|_| "muse interface directory unavailable")?
        .collect::<Result<_, _>>()
        .map_err(|_| "muse interface directory unavailable")?;
    if entries.is_empty() {
        return Ok(lock);
    }
    if entries.len() != 1 || entries[0].file_name() != file_name {
        return Err("muse interface directory must be empty before launch service startup".into());
    }
    let metadata = std::fs::symlink_metadata(&socket_canonical)
        .map_err(|_| "muse launch socket observation failed")?;
    stale_socket(&socket_canonical, &metadata)?;
    let name = CString::new(file_name.as_encoded_bytes())
        .map_err(|_| "invalid muse launch socket name")?;
    let mut current: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe {
        libc::fstatat(
            directory.as_raw_fd(),
            name.as_ptr(),
            &mut current,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
        || (current.st_dev, current.st_ino) != (metadata.dev(), metadata.ino())
        || current.st_mode & libc::S_IFMT != libc::S_IFSOCK
    {
        return Err("muse launch socket changed during retirement".into());
    }
    if unsafe { libc::unlinkat(directory.as_raw_fd(), name.as_ptr(), 0) } != 0 {
        return Err("muse launch socket retirement failed".into());
    }
    directory
        .sync_all()
        .map_err(|_| "muse interface directory sync failed")?;
    Ok(lock)
}
