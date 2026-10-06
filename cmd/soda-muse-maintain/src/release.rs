use super::config::Config;
use std::ffi::CString;
use std::fs;
use std::os::unix::fs::MetadataExt;
use std::os::unix::io::RawFd;

pub(crate) fn apply_release_images(c: &mut Config, path: &str) -> Result<(), String> {
    if path.is_empty() {
        return Ok(());
    }
    let failed = String::from("immutable appliance image defaults unavailable");
    let payload = load_release_payload(path).map_err(|_| failed.clone())?;
    // RequireNative: x86_64 payload on a linux/amd64 binary only.
    if payload.architecture != "x86_64"
        || !cfg!(target_arch = "x86_64")
        || !cfg!(target_os = "linux")
    {
        return Err(failed);
    }
    let project = payload
        .image_config("project-os")
        .ok_or_else(|| failed.clone())?;
    let companion = payload
        .image_config("tailnet")
        .ok_or_else(|| failed.clone())?;
    if (!c.image.is_empty() && c.image != project)
        || (!c.tailnet_image.is_empty() && c.tailnet_image != companion)
    {
        return Err(String::from(
            "saved image selection conflicts with appliance release; explicit migration required",
        ));
    }
    c.image = project;
    if c.tailnet_management {
        c.tailnet_image = companion;
    }
    Ok(())
}

#[derive(Debug)]
pub(crate) struct ReleasePayload {
    pub(crate) architecture: String,
    pub(crate) images: Vec<(String, ReleaseImage)>,
}

#[derive(Debug)]
pub(crate) struct ReleaseImage {
    pub(crate) reference: String,
    pub(crate) config: String,
    pub(crate) manifest: String,
    pub(crate) archive_sha256: String,
}

impl ReleasePayload {
    pub(crate) fn image_config(&self, name: &str) -> Option<String> {
        self.images
            .iter()
            .find(|(n, _)| n == name)
            .map(|(_, image)| image.config.clone())
    }
}

// load_release_payload mirrors deliver.Load plus build.ReadJSON: confined
// regular file under 4 MiB, strict single object, full payload validation.
// Every failure collapses to the caller's unavailable message.
pub(crate) fn load_release_payload(path: &str) -> Result<ReleasePayload, ()> {
    let (dir, base) = match path.rfind('/') {
        Some(i) => (&path[..i], &path[i + 1..]),
        None => return Err(()),
    };
    let dir_path = if dir.is_empty() { "/" } else { dir };
    let lstat = fs::symlink_metadata(path).map_err(|_| ())?;
    if !lstat.is_file() || lstat.len() > 4 << 20 {
        return Err(());
    }
    let dir_fd = unsafe {
        let c = CString::new(dir_path).map_err(|_| ())?;
        libc::open(
            c.as_ptr(),
            libc::O_RDONLY | libc::O_CLOEXEC | libc::O_DIRECTORY,
        )
    };
    if dir_fd < 0 {
        return Err(());
    }
    struct Guard(RawFd);
    impl Drop for Guard {
        fn drop(&mut self) {
            unsafe { libc::close(self.0) };
        }
    }
    let _dir_guard = Guard(dir_fd);
    let base_c = CString::new(base).map_err(|_| ())?;
    let fd = unsafe {
        libc::openat(
            dir_fd,
            base_c.as_ptr(),
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_CLOEXEC,
            0,
        )
    };
    if fd < 0 {
        return Err(());
    }
    let _fd_guard = Guard(fd);
    let mut fst: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd, &mut fst) } != 0 {
        return Err(());
    }
    if (fst.st_mode & libc::S_IFMT) != libc::S_IFREG
        || fst.st_dev as u64 != lstat.dev()
        || fst.st_ino as u64 != lstat.ino()
    {
        return Err(());
    }
    let mut body = Vec::new();
    let mut chunk = [0u8; 65536];
    loop {
        let n = unsafe { libc::read(fd, chunk.as_mut_ptr() as *mut libc::c_void, chunk.len()) };
        if n < 0 {
            return Err(());
        }
        if n == 0 {
            break;
        }
        body.extend_from_slice(&chunk[..n as usize]);
        if body.len() > (4 << 20) + 1 {
            return Err(());
        }
    }
    if body.len() > 4 << 20 {
        return Err(());
    }
    super::decode_release_payload(&body)
}
