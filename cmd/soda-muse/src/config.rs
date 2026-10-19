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
