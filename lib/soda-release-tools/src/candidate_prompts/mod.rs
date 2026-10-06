//! `soda-candidate` interactive prompts (Go `tools/soda-candidate`
//! `prompts.go` plus the `preflight.go` answer checks).

use std::io::{BufRead, Write};

use crate::candidate::{valid_out_leaf, validate_resolved, Options};

pub mod defaults;

pub use defaults::*;

pub struct Prompter<'a> {
    input: Box<dyn BufRead + 'a>,
    output: Box<dyn Write + 'a>,
    exists: Box<dyn Fn(&str) -> bool + 'a>,
}

impl<'a> Prompter<'a> {
    pub fn new(input: Box<dyn BufRead + 'a>, output: Box<dyn Write + 'a>) -> Self {
        Prompter {
            input,
            output,
            exists: Box::new(file_exists),
        }
    }

    pub fn with_exists(
        input: Box<dyn BufRead + 'a>,
        output: Box<dyn Write + 'a>,
        exists: Box<dyn Fn(&str) -> bool + 'a>,
    ) -> Self {
        Prompter {
            input,
            output,
            exists,
        }
    }

    fn line(&mut self, prompt: &str, def: &str) -> Result<String, String> {
        if def.is_empty() {
            write!(self.output, "{prompt}: ").map_err(|e| e.to_string())?;
        } else {
            write!(self.output, "{prompt} [{def}]: ").map_err(|e| e.to_string())?;
        }
        self.output.flush().map_err(|e| e.to_string())?;
        let mut s = String::new();
        match self.input.read_line(&mut s) {
            Ok(0) | Err(_) => {
                return Err("input ended; rerun with flags or --non-interactive".to_owned());
            }
            Ok(_) => {}
        }
        let s = s.trim().to_owned();
        if s.is_empty() {
            return Ok(def.to_owned());
        }
        Ok(s)
    }

    fn choice(&mut self, prompt: &str, options: &[&str], def: usize) -> Result<usize, String> {
        for (i, o) in options.iter().enumerate() {
            writeln!(self.output, "  {}) {o}", i + 1).map_err(|e| e.to_string())?;
        }
        loop {
            let s = self.line(prompt, &(def + 1).to_string())?;
            match s.trim().parse::<usize>() {
                Ok(n) if n >= 1 && n <= options.len() => return Ok(n - 1),
                _ => {
                    writeln!(self.output, "Enter a number 1-{}.", options.len())
                        .map_err(|e| e.to_string())?;
                }
            }
        }
    }

    fn edit_mode(&mut self, o: &mut Options) -> Result<(), String> {
        let sel = self.choice("Build mode", MODE_OPTIONS, mode_index(&o.mode))?;
        o.mode = ["candidate", "media"][sel].to_owned();
        if o.mode == "media" && o.rootfs_url.is_empty() {
            o.rootfs_url = FIXTURE_ROOTFS_URL.to_owned();
        }
        Ok(())
    }

    fn edit_out(&mut self, o: &mut Options) -> Result<(), String> {
        o.out = self.ask_out("Fresh output directory", &o.out.clone())?;
        Ok(())
    }

    fn edit_controller(&mut self, o: &mut Options) -> Result<(), String> {
        o.controller =
            self.ask_absolute("Admitted soda-build executable", &o.controller.clone())?;
        Ok(())
    }

    fn edit_worker_config(&mut self, o: &mut Options) -> Result<(), String> {
        o.worker_config =
            self.ask_absolute("Restricted worker config", &o.worker_config.clone())?;
        Ok(())
    }

    fn edit_rootfs_url(&mut self, o: &mut Options) -> Result<(), String> {
        o.rootfs_url = self.ask_non_empty("Public rootfs base URL", &o.rootfs_url.clone())?;
        Ok(())
    }

    fn edit_fast_compress(&mut self, o: &mut Options) -> Result<(), String> {
        let def = if o.compression == "fast" { "y" } else { "n" };
        o.compression = self.ask_fast(def)?;
        Ok(())
    }

    fn edit_repo_prefix(&mut self, o: &mut Options) -> Result<(), String> {
        o.repo_prefix = self.ask_non_empty(
            "Image repository prefix (intent only; no publication)",
            &o.repo_prefix.clone(),
        )?;
        Ok(())
    }

    pub fn ask_absolute(&mut self, prompt: &str, def: &str) -> Result<String, String> {
        let mut def = def.to_owned();
        loop {
            let s = self.line(prompt, &def)?;
            def = s.clone();
            if !s.starts_with('/') {
                writeln!(self.output, "Absolute path required.").map_err(|e| e.to_string())?;
                continue;
            }
            return Ok(s);
        }
    }

    pub fn ask_non_empty(&mut self, prompt: &str, def: &str) -> Result<String, String> {
        let mut def = def.to_owned();
        loop {
            let s = self.line(prompt, &def)?;
            def = s.clone();
            if s.trim().is_empty() {
                writeln!(self.output, "A value is required here.").map_err(|e| e.to_string())?;
                continue;
            }
            return Ok(s);
        }
    }

    pub fn ask_out(&mut self, prompt: &str, def: &str) -> Result<String, String> {
        let mut def = def.to_owned();
        loop {
            let s = self.line(prompt, &def)?;
            def = s.clone();
            let leaf = s.rsplit('/').next().unwrap_or_default();
            let parent = match s.rfind('/') {
                Some(0) => "/".to_owned(),
                Some(i) => s[..i].to_owned(),
                None => ".".to_owned(),
            };
            let reason = if !s.starts_with('/') {
                Some("Absolute path required.".to_owned())
            } else if !valid_out_leaf(leaf) {
                Some("Lowercase letters, digits, or dashes only (worker name rule).".to_owned())
            } else if !parent_dir_exists(&parent) {
                Some(format!("Parent {parent} must already exist."))
            } else if !path_absent(&s) {
                Some("That output exists; each attempt needs a fresh directory.".to_owned())
            } else {
                None
            };
            match reason {
                Some(r) => {
                    writeln!(self.output, "{r}").map_err(|e| e.to_string())?;
                }
                None => return Ok(s),
            }
        }
    }

    fn ask_fast(&mut self, def: &str) -> Result<String, String> {
        let mut def = def.to_owned();
        loop {
            let s = self.line("Fast iteration compression (dev ISO only) [y/n]", &def)?;
            match s.trim().to_lowercase().as_str() {
                "y" | "yes" => return Ok("fast".to_owned()),
                "n" | "no" => return Ok(String::new()),
                _ => {
                    def = s;
                    writeln!(self.output, "Answer y or n.").map_err(|e| e.to_string())?;
                }
            }
        }
    }

    fn show_fast_compress(o: &Options) -> String {
        if o.compression == "fast" {
            "yes".to_owned()
        } else {
            "no".to_owned()
        }
    }

    fn render_overview(&mut self, o: &Options) -> Result<(), String> {
        writeln!(
            self.output,
            "\nsoda-candidate | {} | arch {} (this host)",
            mode_label(&o.mode),
            o.arch
        )
        .map_err(|e| e.to_string())?;
        for (i, (label, show, status)) in overview_rows(o).iter().enumerate() {
            let extra = match status {
                Some(s) => format!("  [{s}]"),
                None => String::new(),
            };
            writeln!(self.output, "  {}) {label:<13} {show}{extra}", i + 1)
                .map_err(|e| e.to_string())?;
        }
        Ok(())
    }

    /// Overview screen: edit fields by number, start with `go`.
    pub fn overview(
        &mut self,
        o: &mut Options,
        suggest: &dyn Fn() -> String,
    ) -> Result<(), String> {
        default_overview(o, suggest, &self.exists_as_fn());
        loop {
            self.render_overview(o)?;
            let s = self.line("Number to edit, 'go' to start, 'quit' to abort", "go")?;
            match s.trim().to_lowercase().as_str() {
                "go" | "run" | "" => {
                    if let Err(e) = validate_resolved(o) {
                        writeln!(self.output, "Cannot start: {e}").map_err(|e| e.to_string())?;
                        continue;
                    }
                    return Ok(());
                }
                "quit" | "q" | "abort" => return Err("aborted by operator".to_owned()),
                _ => {
                    let count = overview_rows(o).len();
                    match s.trim().parse::<usize>() {
                        Ok(n) if n >= 1 && n <= count => self.edit_field(o, n - 1)?,
                        _ => {
                            writeln!(self.output, "Type a field number, 'go', or 'quit'.")
                                .map_err(|e| e.to_string())?;
                        }
                    }
                }
            }
        }
    }

    fn exists_as_fn(&self) -> Box<dyn Fn(&str) -> bool + '_> {
        let exists: &dyn Fn(&str) -> bool = &self.exists;
        Box::new(move |p| exists(p))
    }

    fn edit_field(&mut self, o: &mut Options, index: usize) -> Result<(), String> {
        // Field order mirrors `overview_rows`.
        let media = o.mode == "media";
        match (index, media) {
            (0, _) => self.edit_mode(o),
            (1, _) => self.edit_out(o),
            (2, _) => self.edit_controller(o),
            (3, _) => self.edit_worker_config(o),
            (4, true) => self.edit_rootfs_url(o),
            (5, true) => self.edit_fast_compress(o),
            (4, false) | (6, true) => self.edit_repo_prefix(o),
            _ => Ok(()),
        }
    }
}

fn overview_rows(o: &Options) -> Vec<(String, String, Option<String>)> {
    let mut rows = vec![
        ("mode".to_owned(), o.mode.clone(), None),
        ("output".to_owned(), o.out.clone(), Some(out_status(&o.out))),
        ("controller".to_owned(), o.controller.clone(), None),
        ("worker config".to_owned(), o.worker_config.clone(), None),
    ];
    if o.mode == "media" {
        rows.push(("rootfs URL".to_owned(), o.rootfs_url.clone(), None));
        rows.push((
            "fast compress".to_owned(),
            Prompter::show_fast_compress(o),
            None,
        ));
    }
    rows.push(("repo prefix".to_owned(), o.repo_prefix.clone(), None));
    rows
}

fn out_status(out: &str) -> String {
    if !out.starts_with('/') {
        "✗ not absolute".to_owned()
    } else {
        let parent = match out.rfind('/') {
            Some(0) => "/",
            Some(i) => &out[..i],
            None => ".",
        };
        if !parent_dir_exists(parent) {
            "✗ parent missing".to_owned()
        } else if !path_absent(out) {
            "✗ already exists".to_owned()
        } else {
            "✓ fresh".to_owned()
        }
    }
}

fn parent_dir_exists(path: &str) -> bool {
    matches!(std::fs::metadata(path), Ok(st) if st.file_type().is_dir())
}

fn path_absent(path: &str) -> bool {
    std::fs::symlink_metadata(path).is_err()
}

fn default_overview(o: &mut Options, suggest: &dyn Fn() -> String, exists: &dyn Fn(&str) -> bool) {
    if o.mode.is_empty() {
        o.mode = "media".to_owned();
    }
    if o.mode == "media" && o.rootfs_url.is_empty() {
        o.rootfs_url = FIXTURE_ROOTFS_URL.to_owned();
    }
    if o.out.is_empty() {
        o.out = suggest();
    }
    o.controller = default_path(&o.controller.clone(), exists, STANDARD_CONTROLLER_PATHS);
    o.worker_config = default_path(
        &o.worker_config.clone(),
        exists,
        STANDARD_WORKER_CONFIG_PATHS,
    );
}

/// Real-terminal overview entry used by the binary.
pub fn prompter_overview(o: &mut Options, suggest: fn() -> String) -> Result<(), String> {
    let stdin = std::io::stdin();
    let input: Box<dyn BufRead> = Box::new(std::io::BufReader::new(stdin.lock()));
    // The Go owner prompts on stderr.
    let output: Box<dyn Write> = Box::new(std::io::stderr());
    let mut p = Prompter::new(input, output);
    p.overview(o, &suggest)
}

#[cfg(test)]
mod tests;
