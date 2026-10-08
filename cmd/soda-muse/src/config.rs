use super::launch_wire::base64_encode;
use super::paths::path_error;
use std::fs;
use std::io;
use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt};
use std::path::Path;

// copy_config always runs as the provisioned account, including symlink reads.
// Native provider credentials are excluded; custody supplies the single auth file.
pub(crate) fn copy_config(source: &str, destination: &str) -> Result<(), String> {
    if !source.starts_with('/') || !destination.starts_with('/') {
        return Err(String::from("absolute config paths required"));
    }
    match fs::metadata(source) {
        Ok(_) => {}
        Err(e) if e.kind() == io::ErrorKind::NotFound => return Ok(()),
        Err(_) => {}
    }
    let source = Path::new(source);
    let destination = Path::new(destination);
    // Keep native entry names and a bounded number of open directory handles.
    // Directory symlinks are not traversed; regular leaf targets remain allowed.
    for entry in walkdir::WalkDir::new(source)
        .follow_links(false)
        .follow_root_links(false)
        .sort_by_file_name()
        .max_open(16)
    {
        let entry = entry.map_err(|e| format!("cannot walk config: {e}"))?;
        if entry.file_name() == "auth.json" {
            // Existing policy excludes the entry but still descends directories.
            continue;
        }
        let relative = entry
            .path()
            .strip_prefix(source)
            .map_err(|_| "config entry escaped its source".to_string())?;
        let target = destination.join(relative);
        if entry.file_type().is_dir() {
            fs::DirBuilder::new()
                .recursive(true)
                .mode(0o700)
                .create(&target)
                .map_err(|e| format!("cannot create config directory: {e}"))?;
        } else {
            copy_config_file_paths(entry.path(), &target)?;
        }
    }
    Ok(())
}

#[cfg(test)]
fn copy_config_file(path: &str, target: &str) -> Result<(), String> {
    copy_config_file_paths(Path::new(path), Path::new(target))
}

fn copy_config_file_paths(path: &Path, target: &Path) -> Result<(), String> {
    // O_NONBLOCK keeps a raced FIFO from hanging before we can reject its
    // opened type. A leaf symlink is intentionally followed, as in the Go
    // implementation; all admission below applies to the opened target.
    let mut opts = fs::OpenOptions::new();
    opts.read(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NONBLOCK);
    let file = opts
        .open(path)
        .map_err(|e| path_error("stat", &path.to_string_lossy(), e))?;
    let info = file
        .metadata()
        .map_err(|e| path_error("stat", &path.to_string_lossy(), e))?;
    if !info.is_file() {
        return Ok(());
    }
    if info.len() > 1 << 20 {
        return Err(String::from("muse config file exceeds private view limit"));
    }
    let body = read_limited(file, 1 << 20)
        .map_err(|e| path_error("open", &path.to_string_lossy(), e))?
        .ok_or_else(|| String::from("muse config file exceeds private view limit"))?;
    let mut opts = fs::OpenOptions::new();
    opts.write(true)
        .create(true)
        .truncate(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NOFOLLOW | libc::O_NONBLOCK);
    opts.mode(0o600);
    use std::io::Write;
    let mut f = opts
        .open(target)
        .map_err(|e| path_error("open", &target.to_string_lossy(), e))?;
    f.write_all(&body).map_err(|e| e.to_string())?;
    Ok(())
}

fn read_limited(file: fs::File, maximum: usize) -> io::Result<Option<Vec<u8>>> {
    use std::io::Read;
    let mut body = Vec::new();
    file.take(maximum as u64 + 1).read_to_end(&mut body)?;
    Ok((body.len() <= maximum).then_some(body))
}

#[cfg(test)]
mod tests {
    use super::{copy_config_file, read_limited};
    use std::io::Write;

    #[test]
    fn walker_keeps_native_names_and_leaf_link_policy() {
        use std::os::unix::ffi::OsStrExt;
        use std::os::unix::fs::{symlink, MetadataExt};
        let temp = tempfile::tempdir().unwrap();
        let source = temp.path().join("source");
        let dest = temp.path().join("dest");
        let outside = temp.path().join("outside");
        std::fs::create_dir(&source).unwrap();
        std::fs::create_dir(&outside).unwrap();
        std::fs::write(outside.join("hidden"), b"outside").unwrap();
        std::fs::write(source.join("auth.json"), b"credential").unwrap();
        let native = std::ffi::OsStr::from_bytes(b"config-\xff");
        std::fs::write(source.join(native), b"native").unwrap();
        symlink(source.join(native), source.join("leaf")).unwrap();
        symlink(&outside, source.join("directory-link")).unwrap();
        super::copy_config(source.to_str().unwrap(), dest.to_str().unwrap()).unwrap();
        assert_eq!(std::fs::read(dest.join(native)).unwrap(), b"native");
        assert_eq!(std::fs::read(dest.join("leaf")).unwrap(), b"native");
        assert!(!dest.join("auth.json").exists());
        assert!(!dest.join("directory-link").exists());
        assert_eq!(
            std::fs::metadata(dest.join(native)).unwrap().mode() & 0o777,
            0o600
        );
        assert_eq!(std::fs::metadata(&dest).unwrap().mode() & 0o777, 0o700);
    }

    #[test]
    fn walker_retains_auth_directory_descent_failure() {
        let temp = tempfile::tempdir().unwrap();
        let source = temp.path().join("source");
        let dest = temp.path().join("dest");
        std::fs::create_dir_all(source.join("auth.json")).unwrap();
        std::fs::write(source.join("auth.json/child"), b"value").unwrap();
        assert!(super::copy_config(source.to_str().unwrap(), dest.to_str().unwrap()).is_err());
    }

    #[test]
    fn config_destination_symlink_is_rejected() {
        let temp = tempfile::tempdir().unwrap();
        let input = temp.path().join("input");
        let protected = temp.path().join("protected");
        let target = temp.path().join("target");
        std::fs::write(&input, b"new").unwrap();
        std::fs::write(&protected, b"old").unwrap();
        std::os::unix::fs::symlink(&protected, &target).unwrap();
        assert!(super::copy_config_file_paths(&input, &target).is_err());
        assert_eq!(std::fs::read(protected).unwrap(), b"old");
    }

    #[test]
    fn bounded_read_uses_open_inode_and_detects_growth() {
        let dir = std::env::temp_dir().join(format!("muse-bounded-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir(&dir).unwrap();
        let path = dir.join("input");
        std::fs::write(&path, b"start").unwrap();
        let opened = std::fs::File::open(&path).unwrap();
        let mut writer = std::fs::OpenOptions::new()
            .append(true)
            .open(&path)
            .unwrap();
        writer.write_all(b"-grown").unwrap();
        let replacement = dir.join("replacement");
        std::fs::write(&replacement, b"replacement").unwrap();
        std::fs::rename(&replacement, &path).unwrap();
        assert_eq!(
            read_limited(opened, 32).unwrap(),
            Some(b"start-grown".to_vec())
        );
        assert_eq!(
            read_limited(std::fs::File::open(&path).unwrap(), 11).unwrap(),
            Some(b"replacement".to_vec())
        );
        assert_eq!(
            read_limited(std::fs::File::open(&path).unwrap(), 10).unwrap(),
            None
        );
        let fifo = dir.join("fifo");
        use std::os::unix::ffi::OsStrExt;
        let fifo_name = std::ffi::CString::new(fifo.as_os_str().as_bytes()).unwrap();
        assert_eq!(unsafe { libc::mkfifo(fifo_name.as_ptr(), 0o600) }, 0);
        assert!(
            copy_config_file(fifo.to_str().unwrap(), dir.join("out").to_str().unwrap()).is_ok()
        );
        let link_target = dir.join("link-target");
        std::fs::write(&link_target, b"allowed-link").unwrap();
        let link = dir.join("link");
        std::os::unix::fs::symlink(&link_target, &link).unwrap();
        copy_config_file(link.to_str().unwrap(), dir.join("out").to_str().unwrap()).unwrap();
        assert_eq!(std::fs::read(dir.join("out")).unwrap(), b"allowed-link");
        let _ = std::fs::remove_dir_all(&dir);
    }
}

// read_config emits the upstream release's saved settings and trust records only.
// It executes as the invoking account, never as a privileged config reader.
pub(crate) fn read_config(source: &str) -> Result<(), String> {
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
    let mut encoded = std::collections::BTreeMap::new();
    for (name, body) in &view {
        encoded.insert(*name, base64_encode(body));
    }
    let out = format!(
        "{}\n",
        serde_json::to_string(&encoded).expect("serializing a config view cannot fail")
    );
    print!("{out}");
    use std::io::Write;
    io::stdout().flush().map_err(|e| e.to_string())?;
    Ok(())
}
