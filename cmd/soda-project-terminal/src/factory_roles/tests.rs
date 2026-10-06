use super::Ctx;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};

static NEXT: AtomicU64 = AtomicU64::new(0);

pub struct Scratch {
    pub root: PathBuf,
    pub factory: PathBuf,
}

impl Scratch {
    pub fn fresh() -> Scratch {
        let id = NEXT.fetch_add(1, Ordering::SeqCst);
        let root = std::env::temp_dir().join(format!("factory-roles-{}-{id}", std::process::id()));
        let _ = std::fs::remove_dir_all(&root);
        std::fs::create_dir_all(&root).expect("scratch root");
        let factory = root.join("factory");
        Scratch { root, factory }
    }

    /// Recording fake `git`: appends each argv (`<arg>` per line under
    /// a `---` marker) to the record file, then exits `code`.
    pub fn git_script(&self, name: &str, code: i32) -> (PathBuf, PathBuf) {
        let record = self.root.join(format!("{name}.record"));
        let script = self.root.join(name);
        let body = format!(
            "#!/bin/sh\n{{ echo '---'; printf '<%s>\\n' \"$@\"; }} >> '{}'\nexit {code}\n",
            record.to_string_lossy(),
        );
        std::fs::write(&script, body).expect("git script");
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&script, std::fs::Permissions::from_mode(0o755))
            .expect("chmod script");
        (script, record)
    }

    pub fn ctx(&self, git: &std::path::Path) -> Ctx {
        Ctx::test_at(self.factory.clone(), git.to_path_buf())
    }

    pub fn record_text(&self, record: &std::path::Path) -> String {
        std::fs::read_to_string(record).unwrap_or_default()
    }
}

impl Drop for Scratch {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.root);
    }
}

pub const PID: &str = "f0123456789abcdef01234567";
pub const PID2: &str = "f123456789abcdef012345678";
pub const COMMIT: &str = "cccccccccccccccccccccccccccccccccccccccc";

pub fn b64_encode(data: &[u8]) -> String {
    const ALPHA: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::new();
    for chunk in data.chunks(3) {
        let mut block = [0u8; 3];
        block[..chunk.len()].copy_from_slice(chunk);
        let triple = (u32::from(block[0]) << 16) | (u32::from(block[1]) << 8) | u32::from(block[2]);
        out.push(ALPHA[(triple >> 18) as usize] as char);
        out.push(ALPHA[((triple >> 12) & 63) as usize] as char);
        if chunk.len() > 1 {
            out.push(ALPHA[((triple >> 6) & 63) as usize] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(ALPHA[(triple & 63) as usize] as char);
        } else {
            out.push('=');
        }
    }
    out
}

pub fn fixture_files() -> Vec<(String, Vec<u8>)> {
    vec![
        ("setup.sh".to_string(), b"true\n".to_vec()),
        ("check.sh".to_string(), b"true\n".to_vec()),
    ]
}

pub fn approve_value(
    pid: &str,
    role: &str,
    files: &[(String, Vec<u8>)],
    bundle: &[u8],
    credential: &str,
) -> soda_json::JsonValue {
    let digest = crate::sha::approved_digest(files);
    let files_obj: Vec<(String, soda_json::JsonValue)> = files
        .iter()
        .map(|(name, contents)| {
            (
                name.clone(),
                soda_json::JsonValue::Str(b64_encode(contents)),
            )
        })
        .collect();
    crate::emit::obj(vec![
        ("op", soda_json::JsonValue::Str("approve".to_string())),
        ("id", soda_json::JsonValue::Str(pid.to_string())),
        ("role", soda_json::JsonValue::Str(role.to_string())),
        ("setup_digest", soda_json::JsonValue::Str(digest)),
        (
            "source_commit",
            soda_json::JsonValue::Str(COMMIT.to_string()),
        ),
        ("files", soda_json::JsonValue::Object(files_obj)),
        ("bundle", soda_json::JsonValue::Str(b64_encode(bundle))),
        (
            "credential",
            soda_json::JsonValue::Str(credential.to_string()),
        ),
    ])
}

pub fn approve_default(pid: &str) -> soda_json::JsonValue {
    approve_value(pid, "soda-coder", &fixture_files(), b"bundle", "")
}

pub fn record_value(pid: &str, missing: &str, refusal: Option<&str>) -> soda_json::JsonValue {
    let mut verified = vec![
        (
            "uid".to_string(),
            soda_json::JsonValue::Str("1000".to_string()),
        ),
        (
            "login".to_string(),
            soda_json::JsonValue::Str("soda-coder".to_string()),
        ),
        (
            "groups".to_string(),
            soda_json::JsonValue::Str("soda-coder".to_string()),
        ),
    ];
    if let Some(text) = refusal {
        verified.push((
            "refusal".to_string(),
            soda_json::JsonValue::Str(text.to_string()),
        ));
    }
    crate::emit::obj(vec![
        ("op", soda_json::JsonValue::Str("record".to_string())),
        ("id", soda_json::JsonValue::Str(pid.to_string())),
        ("tools", soda_json::JsonValue::Array(Vec::new())),
        ("missing", soda_json::JsonValue::Str(missing.to_string())),
        ("verified", soda_json::JsonValue::Object(verified)),
    ])
}

pub fn op_value(op: &str, pid: Option<&str>) -> soda_json::JsonValue {
    let mut pairs = vec![("op", soda_json::JsonValue::Str(op.to_string()))];
    if let Some(pid) = pid {
        pairs.push(("id", soda_json::JsonValue::Str(pid.to_string())));
    }
    crate::emit::obj(pairs)
}

/// Assert a `Fail` carrying the exact `.py` message.
pub fn assert_fail(result: Result<soda_json::JsonValue, crate::error::Error>, message: &str) {
    match result {
        Err(crate::error::Error::Fail(text)) => assert_eq!(text, message),
        other => panic!("expected Fail({message:?}), got {other:?}"),
    }
}

#[test]
fn b64_round_trip() {
    for len in [0, 1, 2, 3, 5, 55, 256] {
        let data: Vec<u8> = (0..len).map(|i| (i * 37 + 11) as u8).collect();
        if len == 0 {
            assert_eq!(b64_encode(&data), "");
        } else {
            assert_eq!(crate::b64::decode(&b64_encode(&data)).unwrap(), data);
        }
    }
}
