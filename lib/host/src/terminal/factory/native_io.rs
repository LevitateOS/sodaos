//! Host-private stdio for the existing admitted Factory transient unit.

use std::ffi::CString;
use std::fs::File;
use std::os::fd::{AsRawFd, FromRawFd};
use std::os::unix::fs::{FileTypeExt, MetadataExt, PermissionsExt};

use crate::factory::PrivateState;

pub struct NativeChannels {
    directory: File,
    input: Option<File>,
    output: Option<File>,
    input_name: CString,
    output_name: CString,
    input_path: String,
    output_path: String,
    input_identity: (u64, u64),
    output_identity: (u64, u64),
}

fn name(project: &str, run: &str, suffix: &str) -> Result<CString, String> {
    if !crate::domain::valid_id(project) || !super::run::valid_factory_run_id(run) {
        return Err("invalid native channel identity".into());
    }
    CString::new(format!("native-{project}-{run}.{suffix}"))
        .map_err(|_| "invalid native channel name".into())
}

fn create(directory: &File, name: &CString, flags: i32) -> Result<(File, (u64, u64)), String> {
    if unsafe { libc::mkfifoat(directory.as_raw_fd(), name.as_ptr(), 0o600) } != 0 {
        return Err("native channel already exists or cannot be created".into());
    }
    let fd = unsafe {
        libc::openat(
            directory.as_raw_fd(),
            name.as_ptr(),
            flags | libc::O_NOFOLLOW | libc::O_CLOEXEC | libc::O_NONBLOCK,
        )
    };
    if fd < 0 {
        return Err("native channel open failed".into());
    }
    let file = unsafe { File::from_raw_fd(fd) };
    let meta = file
        .metadata()
        .map_err(|_| "native channel metadata unavailable")?;
    if !meta.file_type().is_fifo()
        || meta.uid() != unsafe { libc::geteuid() }
        || meta.permissions().mode() & 0o077 != 0
        || meta.nlink() != 1
    {
        return Err("native channel is not service-private".into());
    }
    label(&file)?;
    Ok((file, (meta.dev(), meta.ino())))
}

fn label(file: &File) -> Result<(), String> {
    if !std::path::Path::new("/sys/fs/selinux/enforce").exists() {
        return Ok(());
    }
    // Apply only to the new verified FIFO descriptor, never a caller path or
    // the surrounding private journal directory. Policy ships in the host image.
    const CONTEXT: &[u8] = b"system_u:object_r:soda_factory_io_t:s0";
    const ATTRIBUTE: &[u8] = b"security.selinux\0";
    if unsafe {
        libc::fsetxattr(
            file.as_raw_fd(),
            ATTRIBUTE.as_ptr().cast(),
            CONTEXT.as_ptr().cast(),
            CONTEXT.len(),
            0,
        )
    } != 0
    {
        return Err("native channel SELinux label unavailable".into());
    }
    let mut observed = [0u8; 128];
    let count = unsafe {
        libc::fgetxattr(
            file.as_raw_fd(),
            ATTRIBUTE.as_ptr().cast(),
            observed.as_mut_ptr().cast(),
            observed.len(),
        )
    };
    if count < 0
        || observed[..count as usize]
            .strip_suffix(&[0])
            .unwrap_or(&observed[..count as usize])
            != CONTEXT
    {
        return Err("native channel SELinux label unconfirmed".into());
    }
    Ok(())
}

impl NativeChannels {
    pub fn create(state_dir: &str, project: &str, run: &str) -> Result<Self, String> {
        let private = PrivateState::open(state_dir)?;
        let directory = private.try_clone_directory()?;
        let input_name = name(project, run, "in")?;
        let output_name = name(project, run, "out")?;
        // The input's read end prevents systemd's open from blocking before
        // admission. The output is read-only, so actual native EOF is visible.
        let (input, input_identity) = create(&directory, &input_name, libc::O_RDWR)?;
        let (output, output_identity) = create(&directory, &output_name, libc::O_RDONLY)?;
        directory
            .sync_all()
            .map_err(|_| "native channel directory sync failed")?;
        Ok(Self {
            directory,
            input: Some(input),
            output: Some(output),
            input_path: format!(
                "{}/{}",
                state_dir.trim_end_matches('/'),
                input_name.to_string_lossy()
            ),
            output_path: format!(
                "{}/{}",
                state_dir.trim_end_matches('/'),
                output_name.to_string_lossy()
            ),
            input_name,
            output_name,
            input_identity,
            output_identity,
        })
    }

    pub(crate) fn input_path(&self) -> &str {
        &self.input_path
    }
    pub(crate) fn output_path(&self) -> &str {
        &self.output_path
    }

    pub fn take_pipes(&mut self) -> Result<(File, File), String> {
        let input = self.input.take().ok_or("native channel already consumed")?;
        let output = self
            .output
            .take()
            .ok_or("native channel already consumed")?;
        Ok((input, output))
    }

    /// Call only after the existing owner confirms native unit retirement.
    /// A crash retains the exact paths; an unknown occupied channel is fenced.
    pub fn retire(self) -> Result<(), String> {
        let Self {
            directory,
            input,
            output,
            input_name,
            output_name,
            input_identity,
            output_identity,
            ..
        } = self;
        drop(input);
        drop(output);
        unlink(&directory, &input_name, input_identity)?;
        unlink(&directory, &output_name, output_identity)?;
        directory
            .sync_all()
            .map_err(|_| "native channel retirement sync failed".into())
    }
}

fn unlink(directory: &File, name: &CString, identity: (u64, u64)) -> Result<(), String> {
    let mut stat: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe {
        libc::fstatat(
            directory.as_raw_fd(),
            name.as_ptr(),
            &mut stat,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        return Err("native channel retirement observation failed".into());
    }
    if (stat.st_dev, stat.st_ino) != identity || stat.st_mode & libc::S_IFMT != libc::S_IFIFO {
        return Err("native channel identity changed".into());
    }
    if unsafe { libc::unlinkat(directory.as_raw_fd(), name.as_ptr(), 0) } != 0 {
        return Err("native channel retirement failed".into());
    }
    Ok(())
}
