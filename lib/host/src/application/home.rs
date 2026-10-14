//! The fixed CoreOS /home alias is trusted; application-controlled links are not.
use std::fs;
use std::os::unix::fs::MetadataExt;
use std::path::{Component, Path, PathBuf};

fn owned_directory(path: &Path, owner: u32) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(|_| "application home is unavailable")?;
    if !meta.is_dir()
        || meta.file_type().is_symlink()
        || meta.uid() != owner
        || meta.mode() & 0o022 != 0
        || fs::canonicalize(path).map_err(|_| "application home is unavailable")? != path
    {
        return Err("application home must be an operator-owned canonical directory".into());
    }
    Ok(())
}

fn resolve_at(path: &Path, home: &Path, coreos_home: &Path, owner: u32) -> Result<PathBuf, String> {
    if !path.is_absolute()
        || path
            .components()
            .any(|part| !matches!(part, Component::RootDir | Component::Normal(_)))
    {
        return Err("application storage requires a lexical path under /home".into());
    }
    let relative = path
        .strip_prefix(home)
        .map_err(|_| "application storage must remain under /home")?;
    let meta = fs::symlink_metadata(home).map_err(|_| "application home is unavailable")?;
    let canonical_home = fs::canonicalize(home).map_err(|_| "application home is unavailable")?;
    if meta.file_type().is_symlink() {
        let link = fs::read_link(home).map_err(|_| "application home alias is unavailable")?;
        if meta.uid() != owner
            || canonical_home != coreos_home
            || (link != Path::new("var/home") && link != coreos_home)
        {
            return Err("application home has an unsupported alias".into());
        }
        owned_directory(
            coreos_home
                .parent()
                .ok_or("application home has no parent")?,
            owner,
        )?;
        owned_directory(coreos_home, owner)?;
    } else {
        owned_directory(home, owner)?;
        if canonical_home != home {
            return Err("application home has an unsupported alias".into());
        }
    }
    let expected = canonical_home.join(relative);
    if fs::canonicalize(path).map_err(|_| "application storage is unavailable")? != expected {
        return Err("application storage links below /home are unsupported".into());
    }
    Ok(expected)
}

pub(super) fn home_path(path: &Path) -> Result<PathBuf, String> {
    resolve_at(path, Path::new("/home"), Path::new("/var/home"), 0)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::{symlink, PermissionsExt};
    #[test]
    fn permits_only_fixed_home_alias_and_refuses_linked_application_tail() {
        let base = std::env::temp_dir().join(format!(
            "soda-home-alias-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        fs::create_dir(&base).unwrap();
        let home = base.join("home");
        let native = base.join("var/home");
        fs::create_dir_all(&native).unwrap();
        for directory in [base.join("var"), native.clone()] {
            fs::set_permissions(directory, fs::Permissions::from_mode(0o755)).unwrap();
        }
        symlink("var/home", &home).unwrap();
        fs::create_dir(native.join("application")).unwrap();
        let owner = unsafe { libc::geteuid() };
        assert_eq!(
            resolve_at(&home.join("application"), &home, &native, owner).unwrap(),
            native.join("application")
        );
        assert!(resolve_at(
            &home.join("application/../application"),
            &home,
            &native,
            owner
        )
        .is_err());
        assert!(resolve_at(
            &home.join("application"),
            &home,
            &native,
            owner.wrapping_add(1)
        )
        .is_err());
        symlink("application", native.join("linked")).unwrap();
        assert!(resolve_at(&home.join("linked"), &home, &native, owner).is_err());
        fs::remove_file(&home).unwrap();
        symlink("var/home", base.join("through")).unwrap();
        symlink("through", &home).unwrap();
        assert!(resolve_at(&home.join("application"), &home, &native, owner).is_err());
        fs::remove_file(&home).unwrap();
        symlink("var", &home).unwrap();
        assert!(resolve_at(&home.join("home/application"), &home, &native, owner).is_err());
        fs::remove_dir_all(base).unwrap();
    }
}
