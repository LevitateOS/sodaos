use std::io::Write;

use crate::marker::marker_path;
use crate::system::{Paths, Sys, CONTAINER, STOP_TIMEOUT, UNIT};

pub(crate) fn dispatch(
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

fn cmd_stop(_paths: &Paths, sys: &mut dyn Sys, stdout: &mut dyn Write) -> Result<(), String> {
    let (code, out) = sys.run(&["systemctl", "stop", UNIT]);
    if code != 0 {
        return Err(format!("systemctl stop failed:\n{out}"));
    }
    let deadline = sys.elapsed() + STOP_TIMEOUT;
    while sys.elapsed() < deadline {
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
    marker.create()?;
    let _ = writeln!(
        stdout,
        "inhibited: {UNIT} masked (runtime), marker {}",
        marker.path().display()
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
            let _ = writeln!(
                stdout,
                "marker: unknown (fixed unit declaration unavailable)"
            );
            return Err(err);
        }
    };
    match marker.is_present() {
        Ok(true) => {
            let _ = writeln!(stdout, "marker: present at {}", marker.path().display());
        }
        Ok(false) => {
            let _ = writeln!(
                stdout,
                "marker: absent (expected at {})",
                marker.path().display()
            );
        }
        Err(err) => {
            let _ = writeln!(stdout, "marker: unknown (unsafe or unobservable path)");
            return Err(err);
        }
    }
    Ok(())
}

fn cmd_lift(paths: &Paths, sys: &mut dyn Sys, stdout: &mut dyn Write) -> Result<(), String> {
    // O04-F1: an unresolvable marker mapping refuses before any unmask
    // mutation; only a known-absent marker reads as already absent.
    let marker = marker_path(paths)?;
    if marker.remove()? {
        let _ = writeln!(stdout, "marker removed: {}", marker.path().display());
    } else {
        let _ = writeln!(stdout, "marker already absent");
    }
    let (code, out) = sys.run(&["systemctl", "unmask", UNIT]);
    if code != 0 {
        return Err(format!("systemctl unmask failed:\n{out}"));
    }
    let _ = writeln!(stdout, "unmasked: {UNIT}");
    Ok(())
}

fn cmd_start(paths: &Paths, sys: &mut dyn Sys, stdout: &mut dyn Write) -> Result<(), String> {
    // O04-F1: an unresolvable marker mapping refuses before any start
    // mutation; only a known marker gates the inhibited precondition.
    let marker = marker_path(paths)?;
    if marker.is_present()? {
        return Err(format!(
            "restart inhibited: marker {} present; reconcile, then run lift",
            marker.path().display()
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
