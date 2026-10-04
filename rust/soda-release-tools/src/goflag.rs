//! A faithful port of Go's `flag` package semantics (single/double dash,
//! `-flag value` and `-flag=value`, bool non-consumption, first-positional
//! stop, sorted `PrintDefaults`, `-h` handling) shared by the three CLIs.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FlagKind {
    Bool,
    Text,
}

#[derive(Debug, Clone)]
pub struct FlagSpec {
    pub name: &'static str,
    pub kind: FlagKind,
    pub usage: &'static str,
    pub default_text: &'static str,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FlagValue {
    Bool(bool),
    Text(String),
}

/// `flag.ErrHelp` equivalent.
pub const ERR_HELP: &str = "flag: help requested";

#[derive(Debug, Clone)]
pub struct ParseOutcome {
    pub values: Vec<(&'static str, FlagValue)>,
    pub positionals: Vec<String>,
}

impl ParseOutcome {
    pub fn boolean(&self, name: &str) -> bool {
        self.values.iter().find_map(|(n, v)| {
            if *n == name {
                if let FlagValue::Bool(b) = v {
                    Some(*b)
                } else {
                    None
                }
            } else {
                None
            }
        }).unwrap_or(false)
    }

    pub fn text(&self, name: &str) -> String {
        self.values.iter().find_map(|(n, v)| {
            if *n == name {
                if let FlagValue::Text(s) = v {
                    Some(s.clone())
                } else {
                    None
                }
            } else {
                None
            }
        }).unwrap_or_default()
    }
}

fn parse_bool_literal(value: &str) -> Option<bool> {
    match value {
        "1" | "t" | "T" | "true" | "TRUE" | "True" => Some(true),
        "0" | "f" | "F" | "false" | "FALSE" | "False" => Some(false),
        _ => None,
    }
}

/// Parse `args` against `specs`, returning the outcome or the exact Go
/// failure string (`flag provided but not defined: ...`, `flag needs an
/// argument: ...`, `bad flag syntax: ...`, `invalid boolean value ...`,
/// or `ERR_HELP`). Defaults seed every value first.
pub fn parse(specs: &[FlagSpec], args: &[String]) -> Result<ParseOutcome, String> {
    let mut values: Vec<(&'static str, FlagValue)> = specs
        .iter()
        .map(|s| {
            let value = match s.kind {
                FlagKind::Bool => FlagValue::Bool(s.default_text == "true"),
                FlagKind::Text => FlagValue::Text(s.default_text.to_owned()),
            };
            (s.name, value)
        })
        .collect();
    let mut index = 0;
    while index < args.len() {
        let arg = &args[index];
        if arg.len() < 2 || !arg.starts_with('-') {
            break;
        }
        let minuses = if arg.starts_with("--") { 2 } else { 1 };
        let name = &arg[minuses..];
        if name.is_empty() || name.starts_with('-') || name.starts_with('=') {
            return Err(format!("bad flag syntax: {arg}"));
        }
        let (name, inline) = match name.split_once('=') {
            Some((n, v)) => (n, Some(v)),
            None => (name, None),
        };
        let slot = values.iter().position(|(n, _)| *n == name);
        if name == "help" || name == "h" {
            if slot.is_none() {
                return Err(ERR_HELP.to_owned());
            }
        }
        let Some(slot) = slot else {
            return Err(format!("flag provided but not defined: -{name}"));
        };
        let kind = specs.iter().find(|s| s.name == name).unwrap().kind;
        match kind {
            FlagKind::Bool => {
                let text = match inline {
                    Some(v) => v,
                    None => {
                        values[slot].1 = FlagValue::Bool(true);
                        index += 1;
                        continue;
                    }
                };
                match parse_bool_literal(text) {
                    Some(b) => values[slot].1 = FlagValue::Bool(b),
                    None => {
                        return Err(format!(
                            "invalid boolean value \"{text}\" for -{name}: strconv.ParseBool: parsing \"{text}\": invalid syntax"
                        ));
                    }
                }
                index += 1;
            }
            FlagKind::Text => {
                let text = match inline {
                    Some(v) => v.to_owned(),
                    None => {
                        index += 1;
                        if index >= args.len() {
                            return Err(format!("flag needs an argument: -{name}"));
                        }
                        args[index].clone()
                    }
                };
                values[slot].1 = FlagValue::Text(text);
                index += 1;
            }
        }
    }
    Ok(ParseOutcome {
        values,
        positionals: args[index..].to_vec(),
    })
}

/// Render Go `PrintDefaults` for `specs` under the `Usage of {name}:` header.
pub fn print_defaults(name: &str, specs: &[FlagSpec]) -> String {
    let mut ordered: Vec<&FlagSpec> = specs.iter().collect();
    ordered.sort_by(|a, b| a.name.cmp(b.name));
    let mut out = format!("Usage of {name}:\n");
    for spec in ordered {
        let mut line = format!("  -{}", spec.name);
        if spec.kind == FlagKind::Text {
            line.push_str(" string");
        }
        line.push_str("\n    \t");
        line.push_str(spec.usage);
        if is_nonzero_default(spec) {
            if spec.kind == FlagKind::Text {
                line.push_str(&format!(" (default {:?})", spec.default_text));
            } else {
                line.push_str(&format!(" (default {})", spec.default_text));
            }
        }
        line.push('\n');
        out.push_str(&line);
    }
    out
}

fn is_nonzero_default(spec: &FlagSpec) -> bool {
    match spec.kind {
        FlagKind::Bool => spec.default_text == "true",
        FlagKind::Text => !spec.default_text.is_empty(),
    }
}
