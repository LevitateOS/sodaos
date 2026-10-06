use super::paths::{go_join, go_strerror};
use super::ShellRequest;
use std::io;
use std::os::unix::ffi::OsStrExt;
use std::os::unix::io::AsRawFd;

pub(crate) fn shell_request(argv: &[String]) -> Result<ShellRequest, String> {
    let cwd = match std::env::current_dir() {
        Ok(p) => p.as_os_str().as_bytes().to_vec(),
        Err(e) => return Err(format!("getwd: {}", go_strerror(e.raw_os_error()))),
    };
    // Go carries cwd bytes through JSON, replacing invalid UTF-8.
    let cwd = String::from_utf8_lossy(&cwd).into_owned();
    let home = env_lossy("HOME");
    let connection_id = env_lossy("SODA_MUSE_CONNECTION");
    let mut config_home = env_lossy("XDG_CONFIG_HOME");
    if config_home.is_empty() {
        if home.is_empty() {
            return Err(String::from("$HOME is not defined"));
        }
        config_home = go_join(&home, ".config");
    }
    let term = env_lossy("TERM");
    let mut request = ShellRequest {
        cwd,
        args: argv.to_vec(),
        home,
        connection_id,
        config_home,
        term,
        tty: false,
        cols: 0,
        rows: 0,
    };
    let mut size: libc::winsize = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::ioctl(io::stdin().as_raw_fd(), libc::TIOCGWINSZ, &mut size) };
    if rc == 0 {
        request.tty = true;
        request.cols = size.ws_col;
        request.rows = size.ws_row;
    }
    validate_shell(&request)?;
    Ok(request)
}

fn env_lossy(name: &str) -> String {
    std::env::var_os(name)
        .map(|v| v.to_string_lossy().into_owned())
        .unwrap_or_default()
}

pub(crate) fn validate_shell(r: &ShellRequest) -> Result<(), String> {
    let denied = String::from("identity authority denied");
    if !launch_absolute_path(&r.cwd, false) || !r.config_paths_valid() {
        return Err(denied);
    }
    if !r.launch_sizes_valid() {
        return Err(denied);
    }
    if r.tty && (r.cols == 0 || r.rows == 0) {
        return Err(denied);
    }
    launch_arguments_valid(&r.args).map_err(|_| denied)
}

impl ShellRequest {
    fn config_paths_valid(&self) -> bool {
        launch_absolute_path(&self.config_home, true) && launch_absolute_path(&self.home, true)
    }

    fn launch_sizes_valid(&self) -> bool {
        launch_text(&self.term, 128) && self.connection_id.len() <= 128 && self.args.len() <= 256
    }
}

fn launch_absolute_path(value: &str, optional: bool) -> bool {
    if optional && value.is_empty() {
        return true;
    }
    value.starts_with('/') && launch_text(value, 4096)
}

fn launch_text(value: &str, limit: usize) -> bool {
    value.len() <= limit && !value.contains('\0')
}

fn launch_arguments_valid(args: &[String]) -> Result<(), ()> {
    let mut size = 0;
    for arg in args {
        size += arg.len();
        if !launch_text(arg, 32768) {
            return Err(());
        }
    }
    if size > 32768 {
        return Err(());
    }
    Ok(())
}
