use super::launch_json::serialize_go;
use super::launch_wire::base64_encode;
use super::paths::{go_base, go_join, path_error};
use std::fs;
use std::io;
use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt};

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
    // O_NONBLOCK keeps a raced FIFO from hanging before we can reject its
    // opened type. A leaf symlink is intentionally followed, as in the Go
    // implementation; all admission below applies to the opened target.
    let mut opts = fs::OpenOptions::new();
    opts.read(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NONBLOCK);
    let file = opts.open(path).map_err(|e| path_error("stat", path, e))?;
    let info = file.metadata().map_err(|e| path_error("stat", path, e))?;
    if !info.is_file() {
        return Ok(());
    }
    if info.len() > 1 << 20 {
        return Err(String::from("muse config file exceeds private view limit"));
    }
    let body = read_limited(file, 1 << 20)
        .map_err(|e| path_error("open", path, e))?
        .ok_or_else(|| String::from("muse config file exceeds private view limit"))?;
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
    let out = format!("{}\n", serialize_go(&encoded));
    print!("{out}");
    use std::io::Write;
    io::stdout().flush().map_err(|e| e.to_string())?;
    Ok(())
}
