//! soda-forgejo-migrate: one-shot Forgejo config migrations before start.
//! Today: drop a legacy inline [database] PASSWD= line. The appliance sets
//! PASSWD_URI=file:... in the environment and Forgejo refuses both at once,
//! so a password left by the installer (or an older settings save) blocks
//! every start until removed. The match is section-exact: other sections'
//! PASSWD entries (notably [mailer]) must survive.
//!
//! Rust port of appliance/bin/soda-forgejo-migrate. Std-only so the tree
//! builds without vendoring (`cargo build`). Behavior matches the
//! shell original: missing config skips silently, the file is rewritten in
//! place (mode preserved), and awk's print/trailing-newline semantics hold.

use std::env;
use std::fs::{self, OpenOptions};
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};

fn main() {
    std::process::exit(run());
}

fn run() -> i32 {
    // `${VAR:-default}`: unset or empty falls back. Extra argv ignored.
    let conf = match env::var("SODA_FORGEJO_APP_INI") {
        Ok(value) if !value.is_empty() => PathBuf::from(value),
        _ => PathBuf::from("/var/lib/soda/forgejo/gitea/conf/app.ini"),
    };
    // `[ -f conf ] || exit 0`: only a regular file is migrated.
    if !fs::metadata(&conf).is_ok_and(|info| info.is_file()) {
        return 0;
    }
    let tmp = match create_tmp(&conf) {
        Ok(tmp) => tmp,
        Err(_) => return 1,
    };
    let rewritten = match migrate(&conf) {
        Ok((rewritten, scrubbed)) => {
            if scrubbed > 0 {
                eprintln!("scrubbed {scrubbed} inline [database] PASSWD line(s)");
            }
            rewritten
        }
        Err(_) => {
            remove_tmp(&tmp);
            return 1;
        }
    };
    // `cat tmp >conf`: truncate in place so mode and ownership survive.
    if fs::write(&conf, rewritten).is_err() {
        remove_tmp(&tmp);
        return 1;
    }
    remove_tmp(&tmp);
    0
}

/// `mktemp conf.XXXXXX` (mode 0600). The shell stages through a real temp
/// file so a failed rewrite never truncates the config; do the same.
fn create_tmp(conf: &Path) -> Result<PathBuf, ()> {
    let mut rand = [0u8; 6];
    getrandom::fill(&mut rand).map_err(|_| ())?;
    let suffix: String = rand
        .iter()
        .map(|b| {
            const ALPHABET: &[u8] =
                b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
            ALPHABET[(b % 62) as usize] as char
        })
        .collect();
    let name = conf
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("app.ini");
    let tmp = conf.with_file_name(format!("{name}.{suffix}"));
    OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(&tmp)
        .map_err(|_| ())?;
    fs::set_permissions(&tmp, fs::Permissions::from_mode(0o600)).map_err(|_| ())?;
    Ok(tmp)
}

fn remove_tmp(tmp: &Path) {
    let _ = fs::remove_file(tmp);
}

/// awk's record split, byte-wise: every line is printed with a trailing
/// newline, so a final unterminated line gains one; an empty file yields no
/// records. Non-UTF8 bytes pass through untouched, as in awk.
fn records(content: &[u8]) -> Vec<&[u8]> {
    if content.is_empty() {
        return Vec::new();
    }
    let body = content.strip_suffix(b"\n").unwrap_or(content);
    body.split(|b| *b == b'\n').collect()
}

/// `/^[ \t]*[Pp][Aa][Ss][Ss][Ww][Dd][ \t]*=/`, byte-wise like awk in C locale.
fn is_passwd_assignment(line: &[u8]) -> bool {
    let mut i = 0;
    while i < line.len() && (line[i] == b' ' || line[i] == b'\t') {
        i += 1;
    }
    for want in *b"passwd" {
        if i < line.len() && line[i].to_ascii_lowercase() == want {
            i += 1;
        } else {
            return false;
        }
    }
    while i < line.len() && (line[i] == b' ' || line[i] == b'\t') {
        i += 1;
    }
    i < line.len() && line[i] == b'='
}

/// Returns the rewritten file plus the scrubbed-line count.
fn migrate(conf: &Path) -> Result<(Vec<u8>, usize), ()> {
    let raw = fs::read(conf).map_err(|_| ())?;
    let mut out = Vec::with_capacity(raw.len());
    let mut in_db = false;
    let mut scrubbed = 0;
    for line in records(&raw) {
        // `/^\[/ { in_db = ($0 == "[database]") }`: every section header
        // resets the flag, exactly and without trimming.
        if line.first() == Some(&b'[') {
            in_db = line == b"[database]";
        }
        if in_db && is_passwd_assignment(line) {
            scrubbed += 1;
            continue;
        }
        out.extend_from_slice(line);
        out.push(b'\n');
    }
    Ok((out, scrubbed))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn passwd_shape_matches_awk() {
        for ok in [
            "PASSWD=x",
            "passwd =",
            "  Passwd\t= x",
            "\tPASSWD\t=",
            "PASSWD =",
        ] {
            assert!(is_passwd_assignment(ok.as_bytes()), "{ok:?}");
        }
        for bad in ["", "PASSWD", "PASSWDX=", "XPASSWD=", "PASS WD="] {
            assert!(!is_passwd_assignment(bad.as_bytes()), "{bad:?}");
        }
    }

    #[test]
    fn record_split_matches_awk_print() {
        assert!(records(b"").is_empty());
        assert_eq!(records(b"a\nb\n"), [b"a".as_slice(), b"b".as_slice()]);
        assert_eq!(records(b"a\nb"), [b"a".as_slice(), b"b".as_slice()]);
        assert_eq!(records(b"a\n\n"), [b"a".as_slice(), b"".as_slice()]);
    }
}
