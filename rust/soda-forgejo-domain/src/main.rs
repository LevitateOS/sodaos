//! Whole-domain stop and restart-inhibition controls for native recovery.
//!
//! Host-operator controls for the offline native-mutation recovery procedure:
//! stop every native writer, inhibit restart, reconcile via Forgejo's own
//! `admin native-operation` commands, lift inhibition, restart. There is no
//! force-unlock verb: a fenced reservation stays held for intervention.
//!
//! Verbs: stop, inhibit, status, lift, start. Run as root.
use std::collections::HashMap;
use std::fs;
use std::io::{self, Write};
use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;

#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
}

const UNIT: &str = "forgejo.service";
const CONTAINER: &str = "soda-forgejo";
// Fountain contract: models/nativeoperation OfflineMarkerPath joins this
// name onto the deployment's AppDataPath. This tool never invents a
// second marker name.
const MARKER_NAME: &str = "nativeop-offline";
const STOP_TIMEOUT: f64 = 60.0;

const DESCRIPTION: &str = "Whole-domain stop and restart-inhibition controls for native recovery.\n\nHost-operator controls for the offline native-mutation recovery procedure: stop every native writer, inhibit restart, reconcile via Forgejo's own `admin native-operation` commands, lift inhibition, restart. There is no force-unlock verb: a fenced reservation stays held for intervention.\n\nVerbs: stop, inhibit, status, lift, start. Run as root.";

fn usage(prog: &str) -> String {
    format!("usage: {prog} [-h] {{stop,inhibit,status,lift,start}}")
}

fn help_text(prog: &str) -> String {
    format!(
        "{usage}\n\n{DESCRIPTION}\n\npositional arguments:\n  {{stop,inhibit,status,lift,start}}\n                        stop writers and verify; mask unit and create the offline marker; bounded host-side readout; remove marker and unmask; refuse while inhibited, else start the unit\n\noptions:\n  -h, --help            show this help message and exit\n",
        usage = usage(prog),
    )
}

fn main() {
    let argv: Vec<String> = std::env::args().collect();
    let prog = argv
        .first()
        .map(|a| a.rsplit('/').next().unwrap_or("soda-forgejo-domain"))
        .unwrap_or("soda-forgejo-domain");
    let prog = if prog.is_empty() {
        "soda-forgejo-domain"
    } else {
        prog
    };
    let verb = match parse_args(prog, &argv[1..]) {
        Ok(None) => {
            print!("{}", help_text(prog));
            let _ = io::stdout().flush();
            std::process::exit(0);
        }
        Ok(Some(verb)) => verb,
        Err(msg) => {
            eprintln!("{}\n{prog}: error: {msg}", usage(prog));
            std::process::exit(2);
        }
    };
    if unsafe { geteuid() } != 0 {
        eprintln!(
            "{}\n{prog}: error: native host operator/root required",
            usage(prog)
        );
        std::process::exit(2);
    }
    let paths = Paths::production();
    let mut sys = RealSys;
    let mut stdout = io::stdout();
    if let Err(msg) = dispatch(&verb, &paths, &mut sys, &mut stdout) {
        eprintln!("soda-forgejo-domain: {msg}");
        std::process::exit(1);
    }
}

fn parse_args(prog: &str, args: &[String]) -> Result<Option<String>, String> {
    let mut verb: Option<String> = None;
    let mut i = 0;
    let mut end_of_opts = false;
    let mut extras: Vec<String> = Vec::new();
    while i < args.len() {
        let arg = &args[i];
        if end_of_opts || !arg.starts_with('-') || arg == "-" {
            if verb.is_none() {
                verb = Some(arg.clone());
            } else {
                extras.push(arg.clone());
            }
            i += 1;
            continue;
        }
        if arg == "--" {
            end_of_opts = true;
            i += 1;
            continue;
        }
        if arg == "-h" || arg == "--help" {
            let _ = prog;
            return Ok(None);
        }
        if arg.starts_with("--help=") {
            let explicit = arg["--help=".len()..].to_string();
            return Err(format!("argument -h/--help: ignored explicit argument '{explicit}'"));
        }
        if arg.starts_with("--") && !arg.contains('=') {
            // Unambiguous long-option prefixes (help is the only one).
            let body = &arg[2..];
            if !body.is_empty() && "help".starts_with(body) {
                return Ok(None);
            }
        }
        extras.push(arg.clone());
        i += 1;
    }
    if !extras.is_empty() {
        return Err(format!("unrecognized arguments: {}", extras.join(" ")));
    }
    match verb {
        None => Err("the following arguments are required: verb".to_string()),
        Some(v) => match v.as_str() {
            "stop" | "inhibit" | "status" | "lift" | "start" => Ok(Some(v)),
            _ => Err(format!(
                "argument verb: invalid choice: '{v}' (choose from stop, inhibit, status, lift, start)"
            )),
        },
    }
}

struct Paths {
    env_file: PathBuf,
    app_ini: PathBuf,
    data_root: PathBuf,
}

impl Paths {
    fn production() -> Paths {
        Paths {
            env_file: PathBuf::from("/etc/soda/forgejo.env"),
            app_ini: PathBuf::from("/var/lib/soda/forgejo/gitea/conf/app.ini"),
            data_root: PathBuf::from("/var/lib/soda/forgejo"),
        }
    }
}

/// Process, identity, and time surface, injectable for tests.
trait Sys {
    fn run(&mut self, argv: &[&str]) -> (i32, String);
    fn now(&mut self) -> f64;
    fn sleep(&mut self, secs: u64);
}

struct RealSys;

impl Sys for RealSys {
    fn run(&mut self, argv: &[&str]) -> (i32, String) {
        match std::process::Command::new(argv[0])
            .args(&argv[1..])
            .output()
        {
            Ok(output) => {
                let mut combined = output.stdout;
                combined.extend_from_slice(&output.stderr);
                (
                    output.status.code().unwrap_or(1),
                    String::from_utf8_lossy(&combined).into_owned(),
                )
            }
            Err(e) => (1, e.to_string()),
        }
    }

    fn now(&mut self) -> f64 {
        use std::time::{SystemTime, UNIX_EPOCH};
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs_f64())
            .unwrap_or(0.0)
    }

    fn sleep(&mut self, secs: u64) {
        std::thread::sleep(std::time::Duration::from_secs(secs));
    }
}

fn dispatch(
    verb: &str,
    paths: &Paths,
    sys: &mut dyn Sys,
    stdout: &mut dyn Write,
) -> Result<(), String> {
    match verb {
        "stop" => cmd_stop(paths, sys, stdout),
        "inhibit" => cmd_inhibit(paths, sys, stdout),
        "status" => cmd_status(paths, sys, stdout),
        "lift" => cmd_lift(paths, sys, stdout),
        "start" => cmd_start(paths, sys, stdout),
        _ => Err(format!("unknown verb {verb}")),
    }
}

fn unit_active(sys: &mut dyn Sys) -> bool {
    sys.run(&["systemctl", "is-active", "--quiet", UNIT]).0 == 0
}

fn unit_masked(sys: &mut dyn Sys) -> bool {
    sys.run(&["systemctl", "is-enabled", UNIT])
        .1
        .contains("masked")
}

fn container_present(sys: &mut dyn Sys) -> Result<bool, String> {
    let (code, out) = sys.run(&[
        "podman",
        "ps",
        "--filter",
        &format!("name={CONTAINER}"),
        "--format",
        "{{.Names}}",
    ]);
    if code != 0 {
        return Err(format!("podman ps failed:\n{out}"));
    }
    Ok(out.split_whitespace().any(|name| name == CONTAINER))
}

// --- configparser-compatible INI subset ---
//
// Matches CPython configparser defaults for the shapes app_data_path uses:
// case-sensitive sections, lowercased keys, `=`/`:` separators, full-line
// `#`/`;` comments, indented continuations joined with `\n`, strict
// duplicates, DEFAULT fallback, and BasicInterpolation on read.

struct IniFile {
    defaults: HashMap<String, String>,
    sections: HashMap<String, HashMap<String, String>>,
}

fn parse_ini(text: &str) -> Result<IniFile, String> {
    let mut defaults: HashMap<String, String> = HashMap::new();
    let mut sections: HashMap<String, HashMap<String, String>> = HashMap::new();
    let mut current: Option<String> = None;
    let mut current_key: Option<String> = None;
    for raw_line in text.split('\n') {
        let line = raw_line.strip_suffix('\r').unwrap_or(raw_line);
        if line.trim().is_empty() {
            // Blank lines join an open value; otherwise they are skipped.
            if let (Some(section), Some(key)) = (current.as_ref(), current_key.as_ref()) {
                let map = if section == "DEFAULT" {
                    &mut defaults
                } else {
                    sections.get_mut(section).unwrap()
                };
                if let Some(value) = map.get_mut(key) {
                    value.push('\n');
                }
            }
            continue;
        }
        let stripped = line.trim_start();
        if stripped.starts_with('#') || stripped.starts_with(';') {
            continue;
        }
        if line.starts_with([' ', '\t']) {
            if let (Some(section), Some(key)) = (current.as_ref(), current_key.as_ref()) {
                let map = if section == "DEFAULT" {
                    &mut defaults
                } else {
                    sections.get_mut(section).unwrap()
                };
                match map.get_mut(key) {
                    Some(value) => {
                        value.push('\n');
                        value.push_str(stripped);
                    }
                    None => return Err("continuation without an option".to_string()),
                }
                continue;
            }
            return Err("continuation without a section".to_string());
        }
        if stripped.starts_with('[') {
            let end = stripped
                .find(']')
                .ok_or_else(|| "missing section header".to_string())?;
            let header = stripped[1..end].to_string();
            if header.is_empty() {
                if current.is_none() {
                    return Err("missing section header".to_string());
                }
                return Err("missing delimiter".to_string());
            }
            if header == "DEFAULT" {
                current = Some(header);
            } else {
                if sections.contains_key(&header) {
                    return Err(format!("duplicate section {header}"));
                }
                sections.insert(header.clone(), HashMap::new());
                current = Some(header);
            }
            current_key = None;
            continue;
        }
        let section = current
            .clone()
            .ok_or_else(|| "missing section header".to_string())?;
        let sep = line
            .find(['=', ':'])
            .ok_or_else(|| "missing delimiter".to_string())?;
        let key = line[..sep].trim().to_ascii_lowercase();
        let value = line[sep + 1..].trim().to_string();
        let map = if section == "DEFAULT" {
            &mut defaults
        } else {
            sections.get_mut(&section).unwrap()
        };
        if map.contains_key(&key) {
            return Err(format!("duplicate option {key}"));
        }
        map.insert(key.clone(), value);
        current_key = Some(key);
    }
    Ok(IniFile { defaults, sections })
}

fn interpolate(
    value: &str,
    section: &HashMap<String, String>,
    defaults: &HashMap<String, String>,
) -> Result<String, String> {
    interpolate_depth(value, section, defaults, 0)
}

fn interpolate_depth(
    value: &str,
    section: &HashMap<String, String>,
    defaults: &HashMap<String, String>,
    depth: u32,
) -> Result<String, String> {
    if depth > 10 {
        return Err("interpolation depth exceeded".to_string());
    }
    let mut out = String::new();
    let mut rest = value;
    while let Some(at) = rest.find('%') {
        out.push_str(&rest[..at]);
        rest = &rest[at + 1..];
        if rest.starts_with('%') {
            out.push('%');
            rest = &rest[1..];
        } else if rest.starts_with('(') {
            let end = rest
                .find(')')
                .ok_or_else(|| "interpolation syntax error".to_string())?;
            let name = rest[1..end].to_ascii_lowercase();
            let after = &rest[end + 1..];
            if !after.starts_with('s') {
                return Err("interpolation syntax error".to_string());
            }
            let raw = section
                .get(&name)
                .or_else(|| defaults.get(&name))
                .ok_or_else(|| "interpolation option missing".to_string())?;
            out.push_str(&interpolate_depth(raw, section, defaults, depth + 1)?);
            rest = &after[1..];
        } else {
            return Err("interpolation syntax error".to_string());
        }
    }
    out.push_str(rest);
    Ok(out)
}

fn ini_get(ini: &IniFile, section: &str, key: &str) -> Option<String> {
    let key = key.to_ascii_lowercase();
    let map = ini.sections.get(section)?;
    let raw = map.get(&key).or_else(|| ini.defaults.get(&key))?;
    Some(raw.clone())
}

/// Resolve the deployment's Forgejo AppDataPath, container-side.
///
/// Reads [server] APP_DATA_PATH from the deployment app.ini with the
/// /etc/soda/forgejo.env FORGEJO__server__APP_DATA_PATH override winning,
/// matching Forgejo's own precedence. Refuses to guess: an absent or
/// relative value is an explicit error, never a default.
fn app_data_path(paths: &Paths) -> Result<String, String> {
    let mut value: Option<String> = None;
    if paths.env_file.exists() {
        let text = fs::read_to_string(&paths.env_file).map_err(|e| e.to_string())?;
        for line in text.split('\n') {
            let line = line.strip_suffix('\r').unwrap_or(line).trim();
            if line.starts_with("FORGEJO__server__APP_DATA_PATH=") {
                value = Some(
                    line.split_once('=')
                        .map(|(_, v)| v)
                        .unwrap_or("")
                        .trim()
                        .trim_matches(|c| c == '\'' || c == '"')
                        .to_string(),
                );
            }
        }
    }
    if value.is_none() {
        if !paths.app_ini.is_file() {
            return Err(format!(
                "deployment app.ini not found at {}",
                paths.app_ini.display()
            ));
        }
        // configparser.read swallows OS errors (missing file reads as
        // empty); only decode and syntax failures surface.
        let raw = fs::read(&paths.app_ini).unwrap_or_default();
        let text = String::from_utf8(raw).map_err(|e| {
            format!(
                "cannot parse deployment app.ini at {}: {e}",
                paths.app_ini.display()
            )
        })?;
        let ini = parse_ini(&text).map_err(|e| {
            format!(
                "cannot parse deployment app.ini at {}: {e}",
                paths.app_ini.display()
            )
        })?;
        // has_option consults DEFAULT too; get interpolates before strip.
        // A missing [server] section raises NoSectionError outside the
        // parse guard, so it surfaces as a bare failure here too.
        let section = ini
            .sections
            .get("server")
            .ok_or_else(|| "No section: 'server'".to_string())?;
        let key = "app_data_path".to_string();
        if section.contains_key(&key) || ini.defaults.contains_key(&key) {
            let raw = ini_get(&ini, "server", "APP_DATA_PATH").unwrap_or_default();
            let interpolated = interpolate(&raw, section, &ini.defaults).map_err(|e| {
                format!(
                    "cannot parse deployment app.ini at {}: {e}",
                    paths.app_ini.display()
                )
            })?;
            value = Some(interpolated.trim().to_string());
        }
    }
    let value = value.unwrap_or_default();
    if value.is_empty() {
        return Err(format!(
            "APP_DATA_PATH is not set in {} nor {}; refusing to guess",
            paths.app_ini.display(),
            paths.env_file.display()
        ));
    }
    if !value.starts_with('/') {
        return Err(format!(
            "APP_DATA_PATH {value:?} is relative; refusing to resolve against an unknown work path"
        ));
    }
    Ok(value)
}

/// Map the container-side marker path onto the host data volume.
fn marker_path(paths: &Paths) -> Result<PathBuf, String> {
    let app_data = app_data_path(paths)?;
    let container = format!("{}/{MARKER_NAME}", app_data.trim_end_matches('/'));
    if container != "/data" && !container.starts_with("/data/") {
        return Err(format!(
            "AppDataPath {container:?} is outside the /data volume; cannot map to the host"
        ));
    }
    let relative = if container == "/data" {
        ".".to_string()
    } else {
        container["/data/".len()..].to_string()
    };
    let host = paths.data_root.join(relative);
    for parent in [&host, &host.parent().unwrap_or(&host).to_path_buf()] {
        if fs::symlink_metadata(parent)
            .map(|st| st.file_type().is_symlink())
            .unwrap_or(false)
        {
            return Err(format!(
                "marker path {} must not contain symlinks",
                parent.display()
            ));
        }
    }
    Ok(host)
}

fn cmd_stop(_paths: &Paths, sys: &mut dyn Sys, stdout: &mut dyn Write) -> Result<(), String> {
    let (code, out) = sys.run(&["systemctl", "stop", UNIT]);
    if code != 0 {
        return Err(format!("systemctl stop failed:\n{out}"));
    }
    let deadline = sys.now() + STOP_TIMEOUT;
    while sys.now() < deadline {
        if !unit_active(sys) && !container_present(sys)? {
            let _ = writeln!(
                stdout,
                "stopped: {UNIT} inactive, container {CONTAINER} absent"
            );
            return Ok(());
        }
        sys.sleep(2);
    }
    let mut survivors: Vec<String> = Vec::new();
    if unit_active(sys) {
        survivors.push(format!("unit {UNIT} still active"));
    }
    if container_present(sys)? {
        survivors.push(format!("container {CONTAINER} still present"));
    }
    Err(format!(
        "whole-domain stop failed: {}",
        survivors.join("; ")
    ))
}

fn cmd_inhibit(paths: &Paths, sys: &mut dyn Sys, stdout: &mut dyn Write) -> Result<(), String> {
    if unit_active(sys) || container_present(sys)? {
        return Err("native writers still running; run stop first and verify it".to_string());
    }
    let (code, out) = sys.run(&["systemctl", "mask", "--runtime", UNIT]);
    if code != 0 {
        return Err(format!("systemctl mask failed:\n{out}"));
    }
    let marker = marker_path(paths)?;
    fs::write(&marker, "offline recovery\n").map_err(|e| e.to_string())?;
    fs::set_permissions(&marker, fs::Permissions::from_mode(0o600)).map_err(|e| e.to_string())?;
    let _ = writeln!(
        stdout,
        "inhibited: {UNIT} masked (runtime), marker {}",
        marker.display()
    );
    Ok(())
}

fn cmd_status(paths: &Paths, sys: &mut dyn Sys, stdout: &mut dyn Write) -> Result<(), String> {
    // Bounded host-side readout only: unit state, container presence and
    // marker presence. Reservation diagnostics stay behind Forgejo's own
    // `admin native-operation status`, which reads the deployment database
    // through the native binary; this tool never opens native SQL.
    let _ = writeln!(
        stdout,
        "unit: {UNIT} {}",
        if unit_active(sys) {
            "active"
        } else {
            "inactive"
        }
    );
    let _ = writeln!(
        stdout,
        "masked: {}",
        if unit_masked(sys) { "yes" } else { "no" }
    );
    let _ = writeln!(
        stdout,
        "container: {CONTAINER} {}",
        if container_present(sys)? {
            "present"
        } else {
            "absent"
        }
    );
    let marker = match marker_path(paths) {
        Ok(marker) => marker,
        Err(err) => {
            let _ = writeln!(stdout, "marker: unknown (AppDataPath unresolved)");
            return Err(err);
        }
    };
    if marker.exists() {
        let _ = writeln!(stdout, "marker: present at {}", marker.display());
    } else {
        let _ = writeln!(stdout, "marker: absent (expected at {})", marker.display());
    }
    Ok(())
}

fn cmd_lift(paths: &Paths, sys: &mut dyn Sys, stdout: &mut dyn Write) -> Result<(), String> {
    let marker = marker_path(paths).ok();
    match marker {
        Some(marker) if marker.exists() => {
            fs::remove_file(&marker).map_err(|e| e.to_string())?;
            let _ = writeln!(stdout, "marker removed: {}", marker.display());
        }
        _ => {
            let _ = writeln!(stdout, "marker already absent");
        }
    }
    let (code, out) = sys.run(&["systemctl", "unmask", UNIT]);
    if code != 0 {
        return Err(format!("systemctl unmask failed:\n{out}"));
    }
    let _ = writeln!(stdout, "unmasked: {UNIT}");
    Ok(())
}

fn cmd_start(paths: &Paths, sys: &mut dyn Sys, stdout: &mut dyn Write) -> Result<(), String> {
    let marker = marker_path(paths).ok();
    if matches!(marker.as_ref(), Some(marker) if marker.exists()) {
        return Err(format!(
            "restart inhibited: marker {} present; reconcile, then run lift",
            marker.unwrap().display()
        ));
    }
    if unit_masked(sys) {
        return Err(format!("unit {UNIT} is masked; run lift first"));
    }
    let (code, out) = sys.run(&["systemctl", "start", UNIT]);
    if code != 0 {
        return Err(format!("systemctl start failed:\n{out}"));
    }
    if !unit_active(sys) {
        return Err(format!("unit {UNIT} did not become active"));
    }
    let _ = writeln!(stdout, "started: {UNIT} active");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};

    static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

    struct FakeSys {
        calls: Vec<Vec<String>>,
        active: bool,
        enabled_out: String,
        podman_out: String,
        podman_code: i32,
        start_code: i32,
        now_values: Vec<f64>,
        sleeps: u32,
    }

    impl FakeSys {
        fn new() -> FakeSys {
            FakeSys {
                calls: Vec::new(),
                active: false,
                enabled_out: "enabled\n".to_string(),
                podman_out: String::new(),
                podman_code: 0,
                start_code: 0,
                now_values: vec![0.0],
                sleeps: 0,
            }
        }
    }

    impl Sys for FakeSys {
        fn run(&mut self, argv: &[&str]) -> (i32, String) {
            self.calls
                .push(argv.iter().map(|s| s.to_string()).collect());
            if argv.get(0..2) == Some(&["systemctl", "is-active"]) {
                return (if self.active { 0 } else { 1 }, String::new());
            }
            if argv.get(0..2) == Some(&["systemctl", "is-enabled"]) {
                return (0, self.enabled_out.clone());
            }
            if argv.first() == Some(&"podman") {
                return (self.podman_code, self.podman_out.clone());
            }
            if argv.get(0..2) == Some(&["systemctl", "start"]) {
                if self.start_code == 0 {
                    self.active = true;
                }
                return (self.start_code, String::new());
            }
            (0, String::new())
        }

        fn now(&mut self) -> f64 {
            if self.now_values.len() > 1 {
                self.now_values.remove(0)
            } else {
                self.now_values[0]
            }
        }

        fn sleep(&mut self, secs: u64) {
            assert_eq!(secs, 2);
            self.sleeps += 1;
        }
    }

    struct Fixture {
        temp: PathBuf,
        paths: Paths,
    }

    fn fixture(app_ini: Option<&str>, env: Option<&str>) -> Fixture {
        let seq = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
        let temp =
            std::env::temp_dir().join(format!("soda-domain-test-{}-{seq}", std::process::id()));
        let app_ini_path = temp.join("app.ini");
        let env_path = temp.join("forgejo.env");
        let data_root = temp.join("data");
        fs::create_dir_all(&data_root).expect("data");
        if let Some(text) = app_ini {
            fs::write(&app_ini_path, text).expect("ini");
        }
        if let Some(text) = env {
            fs::write(&env_path, text).expect("env");
        }
        Fixture {
            temp,
            paths: Paths {
                env_file: env_path,
                app_ini: app_ini_path,
                data_root,
            },
        }
    }

    fn valid_ini() -> Fixture {
        fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None)
    }

    fn run_verb(verb: &str, fx: &Fixture, sys: &mut FakeSys) -> (Result<(), String>, String) {
        let mut stdout: Vec<u8> = Vec::new();
        let result = dispatch(verb, &fx.paths, sys, &mut stdout);
        (result, String::from_utf8(stdout).expect("utf8"))
    }

    #[test]
    fn bare_app_name_status_reports_clear_error() {
        let fx = fixture(Some("APP_NAME = Soda\n"), None);
        let mut sys = FakeSys::new();
        let (result, stdout) = run_verb("status", &fx, &mut sys);
        assert!(result.is_err());
        assert!(stdout.contains("marker: unknown"), "{stdout}");
        let err = result.expect_err("status fails");
        assert!(err.contains("app.ini"), "{err}");
        assert!(!stdout.contains("Traceback"));
        assert!(!err.contains("Traceback"));
    }

    #[test]
    fn bare_app_name_start_has_no_crash() {
        let fx = fixture(Some("APP_NAME = Soda\n"), None);
        let mut sys = FakeSys::new();
        let (result, stdout) = run_verb("start", &fx, &mut sys);
        let _ = result;
        assert!(!stdout.contains("Traceback"));
    }

    #[test]
    fn valid_app_ini_still_resolves() {
        let fx = valid_ini();
        assert_eq!(app_data_path(&fx.paths).expect("ini"), "/data/soda");
        fs::write(
            &fx.paths.env_file,
            "FORGEJO__server__APP_DATA_PATH=/data/override\n",
        )
        .expect("env");
        assert_eq!(
            app_data_path(&fx.paths).expect("override"),
            "/data/override"
        );
    }

    #[test]
    fn parsed_but_missing_app_data_path_reports_clear_error() {
        let fx = fixture(Some("[server]\nAPP_NAME = Soda\n"), None);
        let err = app_data_path(&fx.paths).expect_err("missing");
        assert!(err.contains("not set"), "{err}");
    }

    #[test]
    fn ini_shapes_match_configparser() {
        // (content, expected value or "parse-error"/"missing")
        for (content, expected) in [
            ("[server] trailing\nAPP_DATA_PATH = /data\n", Ok("/data")),
            ("[server]\nAPP_DATA_PATH : /data\n", Ok("/data")),
            ("[Server]\nAPP_DATA_PATH = /data\n", Err("nosection")),
            ("[server]\nApp_Data_Path = /data\n", Ok("/data")),
            ("[server]\njustwords\n", Err("parse")),
            (
                "[DEFAULT]\nAPP_DATA_PATH = /dflt\n[server]\nX = 1\n",
                Ok("/dflt"),
            ),
            ("[server]\nA = 1\nA = 2\n", Err("parse")),
            ("[server]\nA = 1\n[server]\nB = 2\n", Err("parse")),
            ("[server]\nAPP_DATA_PATH = /da\n ta\n", Ok("/da\nta")),
            ("[server]\nAPP_DATA_PATH = /da%ta\n", Err("parse")),
            (
                "[server]\nBASE = /d\nAPP_DATA_PATH = %(BASE)s/x\n",
                Ok("/d/x"),
            ),
            ("[server]\nAPP_DATA_PATH = %(NOPE)s/x\n", Err("parse")),
            ("[server]\n  # note\nAPP_DATA_PATH = /data\n", Ok("/data")),
            (
                "[server]\nAPP_DATA_PATH = /data ; note\n",
                Ok("/data ; note"),
            ),
            ("[server]\nAPP_DATA_PATH =\n", Err("missing")),
            ("[]\nA = 1\n", Err("parse")),
            ("[other]\nA = 1\n", Err("nosection")),
        ] {
            let fx = fixture(Some(content), None);
            match expected {
                Ok(want) => assert_eq!(
                    app_data_path(&fx.paths).expect(content),
                    want,
                    "{content:?}"
                ),
                Err("missing") => assert!(app_data_path(&fx.paths)
                    .expect_err(content)
                    .contains("not set")),
                Err("nosection") => assert!(app_data_path(&fx.paths)
                    .expect_err(content)
                    .contains("No section")),
                _ => assert!(app_data_path(&fx.paths)
                    .expect_err(content)
                    .contains("cannot parse")),
            }
        }
        // Env override wins and strips one layer of quotes; last match wins.
        let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/ini\n"), Some("FORGEJO__server__APP_DATA_PATH=\"/data/first\"\nFORGEJO__server__APP_DATA_PATH='/data/second'\n"));
        assert_eq!(app_data_path(&fx.paths).expect("override"), "/data/second");
        // Relative values refuse to resolve.
        let fx = fixture(Some("[server]\nAPP_DATA_PATH = data/rel\n"), None);
        assert!(app_data_path(&fx.paths)
            .expect_err("relative")
            .contains("relative"));
        // Missing app.ini with no override is explicit.
        let fx = fixture(None, None);
        assert!(app_data_path(&fx.paths)
            .expect_err("absent")
            .contains("not found"));
    }

    #[test]
    fn marker_mapping_rejects_outside_volume_and_symlinks() {
        let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None);
        let marker = marker_path(&fx.paths).expect("marker");
        assert_eq!(marker, fx.paths.data_root.join("soda").join(MARKER_NAME));
        let fx = fixture(Some("[server]\nAPP_DATA_PATH = /etc/soda\n"), None);
        assert!(marker_path(&fx.paths)
            .expect_err("outside")
            .contains("outside"));
        // Symlinked marker parent is refused.
        let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None);
        let target = fx.temp.join("target");
        fs::create_dir_all(&target).expect("target");
        std::os::unix::fs::symlink(&target, fx.paths.data_root.join("soda")).expect("link");
        assert!(marker_path(&fx.paths)
            .expect_err("symlink")
            .contains("symlinks"));
    }

    #[test]
    fn stop_verifies_quiescence_before_returning() {
        let fx = valid_ini();
        let mut sys = FakeSys::new();
        let (result, stdout) = run_verb("stop", &fx, &mut sys);
        assert!(result.is_ok(), "{result:?}");
        assert!(
            stdout.contains("stopped: forgejo.service inactive"),
            "{stdout}"
        );
        assert!(sys
            .calls
            .iter()
            .any(|c| c == &["systemctl", "stop", "forgejo.service"]));
    }

    #[test]
    fn stop_reports_survivors_after_timeout() {
        let fx = valid_ini();
        let mut sys = FakeSys::new();
        sys.active = true;
        sys.podman_out = "soda-forgejo\n".to_string();
        sys.now_values = vec![0.0, 61.0, 61.0, 61.0];
        let (result, _) = run_verb("stop", &fx, &mut sys);
        let err = result.expect_err("survivors");
        assert!(err.contains("unit forgejo.service still active"), "{err}");
        assert!(
            err.contains("container soda-forgejo still present"),
            "{err}"
        );
    }

    #[test]
    fn inhibit_requires_quiescence_then_masks_and_marks() {
        let fx = valid_ini();
        let mut sys = FakeSys::new();
        sys.active = true;
        let (result, _) = run_verb("inhibit", &fx, &mut sys);
        assert!(result.expect_err("running").contains("still running"));
        let mut sys = FakeSys::new();
        let marker = marker_path(&fx.paths).expect("marker");
        fs::create_dir_all(marker.parent().expect("parent")).expect("parent");
        let (result, stdout) = run_verb("inhibit", &fx, &mut sys);
        assert!(result.is_ok(), "{result:?}");
        assert!(
            stdout.contains("inhibited: forgejo.service masked"),
            "{stdout}"
        );
        assert!(sys
            .calls
            .iter()
            .any(|c| c == &["systemctl", "mask", "--runtime", "forgejo.service"]));
        let marker = marker_path(&fx.paths).expect("marker");
        assert_eq!(
            fs::read_to_string(&marker).expect("marker"),
            "offline recovery\n"
        );
        assert_eq!(
            fs::metadata(&marker).expect("m").permissions().mode() & 0o777,
            0o600
        );
    }

    #[test]
    fn status_reads_all_four_signals() {
        let fx = valid_ini();
        let mut sys = FakeSys::new();
        sys.enabled_out = "masked\n".to_string();
        sys.podman_out = "other\nsoda-forgejo\n".to_string();
        let (result, stdout) = run_verb("status", &fx, &mut sys);
        assert!(result.is_ok(), "{result:?}");
        assert!(
            stdout.contains("unit: forgejo.service inactive"),
            "{stdout}"
        );
        assert!(stdout.contains("masked: yes"), "{stdout}");
        assert!(
            stdout.contains("container: soda-forgejo present"),
            "{stdout}"
        );
        assert!(stdout.contains("marker: absent"), "{stdout}");
    }

    #[test]
    fn lift_removes_marker_and_unmasks() {
        let fx = valid_ini();
        let marker = marker_path(&fx.paths).expect("marker");
        fs::create_dir_all(marker.parent().expect("parent")).expect("parent");
        fs::write(&marker, "offline recovery\n").expect("marker");
        let mut sys = FakeSys::new();
        let (result, stdout) = run_verb("lift", &fx, &mut sys);
        assert!(result.is_ok(), "{result:?}");
        assert!(stdout.contains("marker removed"), "{stdout}");
        assert!(stdout.contains("unmasked: forgejo.service"), "{stdout}");
        assert!(!marker.exists());
        // Tolerant of unresolvable markers: still unmasks.
        let fx = fixture(Some("APP_NAME = Soda\n"), None);
        let mut sys = FakeSys::new();
        let (result, stdout) = run_verb("lift", &fx, &mut sys);
        assert!(result.is_ok(), "{result:?}");
        assert!(stdout.contains("marker already absent"), "{stdout}");
    }

    #[test]
    fn start_refuses_inhibited_and_masked() {
        let fx = valid_ini();
        let marker = marker_path(&fx.paths).expect("marker");
        fs::create_dir_all(marker.parent().expect("parent")).expect("parent");
        fs::write(&marker, "offline recovery\n").expect("marker");
        let mut sys = FakeSys::new();
        let (result, _) = run_verb("start", &fx, &mut sys);
        assert!(result.expect_err("inhibited").contains("restart inhibited"));
        fs::remove_file(&marker).expect("remove");
        let mut sys = FakeSys::new();
        sys.enabled_out = "masked\nmasked-runtime\n".to_string();
        let (result, _) = run_verb("start", &fx, &mut sys);
        assert!(result.expect_err("masked").contains("is masked"));
        let mut sys = FakeSys::new();
        let (result, stdout) = run_verb("start", &fx, &mut sys);
        assert!(result.is_ok(), "{result:?}");
        assert!(
            stdout.contains("started: forgejo.service active"),
            "{stdout}"
        );
        // Start failure and never-active are distinct errors.
        let mut sys = FakeSys::new();
        sys.start_code = 1;
        let (result, _) = run_verb("start", &fx, &mut sys);
        assert!(result
            .expect_err("start-fail")
            .contains("systemctl start failed"));
    }

    #[test]
    fn cli_parsing_matches_argparse_verbs() {
        let argv = |words: &[&str]| words.iter().map(|w| w.to_string()).collect::<Vec<_>>();
        assert_eq!(
            parse_args("p", &argv(&["status"])),
            Ok(Some("status".to_string()))
        );
        assert!(parse_args("p", &argv(&["-h"])).expect("help").is_none());
        assert!(parse_args("p", &argv(&["--help"])).expect("help").is_none());
        assert_eq!(
            parse_args("p", &argv(&[])),
            Err("the following arguments are required: verb".to_string())
        );
        assert_eq!(
            parse_args("p", &argv(&["freeze"])),
            Err("argument verb: invalid choice: 'freeze' (choose from stop, inhibit, status, lift, start)".to_string())
        );
        assert_eq!(
            parse_args("p", &argv(&["status", "extra"])),
            Err("unrecognized arguments: extra".to_string())
        );
        assert_eq!(
            parse_args("p", &argv(&["--bogus", "status"])),
            Err("unrecognized arguments: --bogus".to_string())
        );
    }
}
