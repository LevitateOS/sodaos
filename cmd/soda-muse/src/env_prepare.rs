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
