use std::fs;
use std::os::unix::fs::{FileTypeExt, MetadataExt};

use super::filesystem::{go_base, go_dir};

pub(crate) fn public_socket_directory(socket: &str) -> Result<String, String> {
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
