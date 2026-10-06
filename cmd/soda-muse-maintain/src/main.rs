// soda-muse-maintain installs public Muse tools and restores a live project interface.
use std::time::{Duration, Instant};

mod archive;
mod command;
mod config;
mod config_validation;
mod config_wire;
mod filesystem;
mod interface;
mod interface_admission;
mod json;
mod json_string;
mod network;
mod options;
mod project;
mod release;
mod release_validation;
mod release_wire;
mod sha256;
mod stage;
#[cfg(test)]
mod test_support;

use config::{load_config, Config};
use filesystem::load_tools;
use interface::{attach_interface, prepare_interface};
use options::{parse, Options};
use project::wait_project;
use stage::{ensure_system_bus, stage_tools};

const MUSE_VERSION: &str = "1.4.0-R4161.1";
const RELEASE_PATH: &str = "/usr/share/soda/release.json";

fn main() {
    if let Err(e) = run() {
        eprintln!("soda-muse-maintain: {e}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("project maintenance requires root"));
    }
    let argv: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    let o = parse(&argv)?;
    let c = load_config(&o.config)?;
    if c.muse_sha256.is_empty() {
        if o.bind_only {
            return Ok(());
        }
        return Err(String::from("muse project runtime is disabled"));
    }
    maintain(&o, &c)
}

fn is_lower_hex(b: u8) -> bool {
    b.is_ascii_digit() || (b'a'..=b'f').contains(&b)
}

fn maintain(o: &Options, c: &Config) -> Result<(), String> {
    // Tool closes its own fd on drop, mirroring deferred closeTools.
    let sources = load_tools(&o.tools, &c.muse_sha256, &c.muse_version)?;
    let deadline = Instant::now() + Duration::from_secs(120);
    let target = wait_project(&o.project, deadline)?;
    if !o.bind_only {
        ensure_system_bus(&target, deadline)?;
        stage_tools(&target, &sources, deadline)?;
    }
    prepare_interface(&target, deadline)?;
    attach_interface(&target, &c.muse_socket, deadline)
}

#[cfg(test)]
mod tests {
    use super::archive::{emit_archive, feed_archive, tar_header};
    use super::command::{podman_streamed, wait_output};
    use super::interface::FdGuard;
    use super::stage::INSTALL_SCRIPT;
    use super::test_support::{emit_synthetic, run_install_script, synthetic_feeds, TestDir};
    use super::*;
    use std::ffi::CString;
    use std::fs;
    use std::os::unix::fs::{MetadataExt, PermissionsExt};
    use std::process::{Command, Stdio};

    #[test]
    fn tar_header_layout_matches_ustar() {
        let h = tar_header("muse", 14).unwrap();
        assert_eq!(&h[..4], b"muse");
        assert!(h[4..100].iter().all(|b| *b == 0));
        assert_eq!(&h[100..108], b"0000755\0");
        assert_eq!(&h[108..116], b"0000000\0");
        assert_eq!(&h[116..124], b"0000000\0");
        assert_eq!(&h[124..136], b"00000000016\0");
        assert_eq!(&h[136..148], b"00000000000\0");
        assert_eq!(h[156], b'0');
        assert_eq!(&h[257..263], b"ustar\0");
        assert_eq!(&h[263..265], b"00");
        let mut sum: u32 = 0;
        for (i, b) in h.iter().enumerate() {
            sum += if (148..156).contains(&i) {
                b' ' as u32
            } else {
                *b as u32
            };
        }
        assert_eq!(&h[148..156], format!("{sum:06o}\0 ").as_bytes());
        assert!(tar_header(&"n".repeat(101), 0).is_err());
    }

    #[test]
    fn tool_replacement_preserves_other_files_and_refuses_symlinks() {
        let dir = TestDir::make("stage");
        let targets: Vec<String> = ["bin/muse", "bin/soda-identity-compose", "libexec/soda/muse"]
            .iter()
            .map(|t| dir.path(t))
            .collect();
        for t in &targets {
            fs::create_dir_all(std::path::Path::new(t).parent().unwrap()).unwrap();
            fs::write(t, b"obsolete public tool").unwrap();
        }
        let retained = dir.path("account-state");
        fs::write(&retained, b"preserved account and project state").unwrap();
        fs::set_permissions(&retained, fs::Permissions::from_mode(0o600)).unwrap();
        let before = fs::symlink_metadata(&retained).unwrap();
        let archive = emit_synthetic();
        // 3 headers + 3 data blocks + 2 trailer blocks.
        assert_eq!(archive.len(), 3 * 512 + 3 * 512 + 1024);
        // The install script is byte-pinned where transcription once dropped
        // the space before the first loop's done.
        assert!(INSTALL_SCRIPT.contains("fi\n done\nfor target"));
        let (ok, stderr) = run_install_script(&archive, &targets);
        assert!(ok, "public replacement failed: {stderr}");
        let after = fs::symlink_metadata(&retained).unwrap();
        assert_eq!((before.dev(), before.ino()), (after.dev(), after.ino()));
        assert_eq!(
            fs::read(&retained).unwrap(),
            b"preserved account and project state"
        );
        for t in &targets {
            let body = fs::read(t).unwrap();
            assert!(body.starts_with(b"synthetic "), "tool not replaced: {t}");
        }
        fs::remove_file(&targets[0]).unwrap();
        std::os::unix::fs::symlink(&retained, &targets[0]).unwrap();
        let (ok, _) = run_install_script(&archive, &targets);
        assert!(!ok, "target symlink accepted");
        assert_eq!(
            fs::read(&retained).unwrap(),
            b"preserved account and project state"
        );
    }

    #[test]
    fn streamed_stage_delivery() {
        // A fake podman proves the feeder/child contract: the child must
        // not inherit the stdin write end, or its reader never sees EOF.
        // (A raw pipe without O_CLOEXEC deadlocked here until the deadline.)
        let dir = TestDir::make("streamed");
        let bindir = dir.path("bin");
        fs::create_dir(&bindir).unwrap();
        fs::write(
            format!("{bindir}/podman"),
            "#!/bin/sh\nprintf \"%s\\0\" \"$@\" > \"$E2E/argv.bin\"\ncat > \"$E2E/stdin.bin\"\nexit 0\n",
        )
        .unwrap();
        fs::set_permissions(
            format!("{bindir}/podman"),
            fs::Permissions::from_mode(0o755),
        )
        .unwrap();
        std::env::set_var(
            "PATH",
            format!("{bindir}:{}", std::env::var("PATH").unwrap()),
        );
        std::env::set_var("E2E", dir.dir_str());
        let tools = TestDir::make("streamed-tools");
        let (feeds, guards) = synthetic_feeds(&tools);
        let deadline = Instant::now() + Duration::from_secs(15);
        podman_streamed(
            move |w| feed_archive(w, &feeds, deadline),
            &["exec", "--user", "0:0", "-i", "some-id"],
            deadline,
        )
        .unwrap();
        drop(guards);
        assert_eq!(fs::read(dir.path("stdin.bin")).unwrap(), emit_synthetic());
        let argv = fs::read(dir.path("argv.bin")).unwrap();
        assert!(argv.starts_with(b"--remote=false\0exec\0--user\0"));
    }

    #[test]
    fn wait_output_reports_status() {
        let c = Command::new("/bin/sh")
            .args(["-c", "echo hello; exit 0"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .unwrap();
        let out = wait_output(c, Instant::now() + Duration::from_secs(5)).unwrap();
        assert_eq!(out, b"hello\n");
        let c = Command::new("/bin/sh")
            .args(["-c", "exit 3"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .unwrap();
        assert_eq!(
            wait_output(c, Instant::now() + Duration::from_secs(5)).unwrap_err(),
            "project maintenance command failed"
        );
    }

    #[test]
    fn short_tool_file_ends_copy_with_eof() {
        let dir = TestDir::make("short");
        let p = dir.path("tool");
        fs::write(&p, b"twelve bytes").unwrap();
        let c = CString::new(p).unwrap();
        let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
        assert!(fd >= 0);
        let _guard = FdGuard(fd);
        // Claim more bytes than the file holds, like a shrink race.
        let feeds = vec![(String::from("tool"), fd, 1_000_000u64)];
        let mut out = Vec::new();
        assert_eq!(
            emit_archive(
                &mut |b: &[u8]| {
                    out.extend_from_slice(b);
                    Ok(())
                },
                &feeds,
            )
            .unwrap_err(),
            "EOF"
        );
    }
}
