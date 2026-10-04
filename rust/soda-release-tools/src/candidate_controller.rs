//! `soda-candidate` controller execution (Go `tools/soda-candidate`
//! `controller.go`): argv construction, child spawn, stderr relay, signal
//! forwarding, and post-run rootfs filing.

use std::io::{BufRead, Write};
use std::process::{Child, Command, Stdio};
use std::sync::atomic::{AtomicI32, Ordering};

use crate::candidate::{describe, monotonic_ns, Options};
use crate::candidate_display::{is_terminal, term_width, Renderer};
use crate::candidate_fixture::{copy_built_rootfs, fixture_addr, fixture_wanted, serve_fixture};
use crate::candidate_prompts::controller_args;

static CHILD_PID: AtomicI32 = AtomicI32::new(0);

extern "C" fn forward_signal(signum: libc::c_int) {
    let pid = CHILD_PID.load(Ordering::SeqCst);
    if pid > 0 {
        unsafe {
            libc::kill(pid, signum);
        }
    }
}

pub fn controller_env(start_ns: i64) -> Vec<(String, String)> {
    let mut env: Vec<(String, String)> = std::env::vars().collect();
    if std::env::var("SODA_BUILD_START_NS").unwrap_or_default().is_empty() {
        env.push(("SODA_BUILD_START_NS".to_owned(), start_ns.to_string()));
    }
    env
}

pub fn controller_argv(o: &Options, start_ns: i64) -> Vec<String> {
    let mut argv = vec![o.controller.clone()];
    argv.extend(controller_args(o));
    if unsafe { libc::geteuid() } != 0 {
        let mut sudo = vec![
            "sudo".to_owned(),
            format!("SODA_BUILD_START_NS={start_ns}"),
        ];
        sudo.append(&mut argv);
        return sudo;
    }
    argv
}

pub struct ControllerRun {
    child: Child,
    child_stderr: Option<std::process::ChildStderr>,
    view: Renderer,
    tty: bool,
}

pub fn start_controller_run(o: &Options) -> Result<ControllerRun, String> {
    let start_ns = monotonic_ns()?;
    let argv = controller_argv(o, start_ns);
    let mut command = Command::new(&argv[0]);
    command.args(&argv[1..]);
    if let Ok(cwd) = std::env::current_dir() {
        command.current_dir(cwd);
    }
    command.envs(controller_env(start_ns));
    command.stdin(Stdio::inherit());
    command.stdout(Stdio::inherit());
    command.stderr(Stdio::piped());
    let mut child = command.spawn().map_err(|e| e.to_string())?;
    let child_stderr = child.stderr.take();
    let tty = is_terminal(libc::STDERR_FILENO) && !o.non_interactive;
    let width = term_width(libc::STDERR_FILENO);
    let stderr_box: Box<dyn Write + Send> = Box::new(std::io::stderr());
    let view = Renderer::new(stderr_box, tty, width);
    view.set_out_dir(&o.out);
    view.note(&format!("soda-candidate: {}", describe(o)))?;
    if let Ok(pid) = i32::try_from(child.id()) {
        CHILD_PID.store(pid, Ordering::SeqCst);
        unsafe {
            libc::signal(libc::SIGINT, forward_signal as *const () as libc::sighandler_t);
            libc::signal(libc::SIGTERM, forward_signal as *const () as libc::sighandler_t);
        }
    }
    if tty {
        view.start_ticker();
    }
    Ok(ControllerRun { child, child_stderr, view, tty })
}

impl ControllerRun {
    fn relay_output(&mut self) -> Result<(), String> {
        let Some(pipe) = self.child_stderr.take() else {
            return Ok(());
        };
        let mut scanner = std::io::BufReader::new(pipe);
        let mut line = String::new();
        let mut feed_err: Option<String> = None;
        loop {
            line.clear();
            match scanner.read_line(&mut line) {
                Ok(0) => break,
                Ok(_) => {
                    let text = line.trim_end_matches(['\r', '\n']);
                    if let Err(e) = self.view.feed(text) {
                        if feed_err.is_none() {
                            feed_err = Some(e);
                        }
                    }
                }
                Err(e) => return Err(e.to_string()),
            }
        }
        feed_err.map_or(Ok(()), Err)
    }

    fn wait_exit(mut self) -> Result<i32, String> {
        let status = self.child.wait().map_err(|e| e.to_string())?;
        CHILD_PID.store(0, Ordering::SeqCst);
        if self.tty {
            self.view.stop_ticker();
        }
        let code = status.code().unwrap_or(1);
        self.view.finish(code)?;
        Ok(code)
    }

    pub fn wait(mut self) -> Result<i32, String> {
        self.relay_output()?;
        self.wait_exit()
    }
}

/// Serve the loopback pickup address for development media; otherwise no-op.
pub fn maybe_serve_fixture(o: &Options) -> Result<Box<dyn FnOnce()>, String> {
    if !fixture_wanted(&o.mode, &o.rootfs_url) {
        return Ok(Box::new(|| {}));
    }
    let addr = fixture_addr(&o.rootfs_url)?;
    let (stop, _) = serve_fixture(&addr, &o.rootfs_dir)?;
    Ok(stop)
}

/// File the built image in the pickup folder for any media run.
pub fn file_built_rootfs(o: &Options, stderr: &mut dyn Write) -> Result<(), String> {
    if o.mode != "media" {
        return Ok(());
    }
    if url_hostname(&o.rootfs_url).is_empty() {
        return Ok(());
    }
    let name = copy_built_rootfs(&o.out, &o.rootfs_dir)?;
    let base = o.rootfs_url.trim_end_matches('/');
    writeln!(stderr, "soda-candidate: serving {name} from {base}/{name}").map_err(|e| e.to_string())
}

fn url_hostname(url: &str) -> String {
    let Some((scheme, rest)) = url.split_once("://") else {
        return String::new();
    };
    if scheme.is_empty() {
        return String::new();
    }
    let authority = rest.split('/').next().unwrap_or_default();
    if let Some(bracketed) = authority.strip_prefix('[') {
        return bracketed.split(']').next().unwrap_or_default().to_owned();
    }
    let host_port = authority.split('@').next_back().unwrap_or_default();
    host_port.split(':').next().unwrap_or_default().to_owned()
}
