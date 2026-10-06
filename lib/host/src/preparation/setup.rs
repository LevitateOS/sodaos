// Approved setup inputs and their digest.
use std::collections::HashMap;

use super::{
    valid_approved_name, FACTORY_CHECK_ENTRY, FACTORY_SETUP_ENTRY, MAX_APPROVED_FILES,
    MAX_APPROVED_FILE_SIZE, MAX_APPROVED_TOTAL, MAX_SOURCE_BUNDLE,
};
use crate::json::{BoundMap, Kind, Spec};
use crate::sha256;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ApprovedSetup {
    pub files: HashMap<String, Vec<u8>>,
    pub bundle: Vec<u8>,
}

pub(crate) const APPROVED_SETUP_SPECS: &[Spec] = &[
    Spec {
        name: "files",
        kind: Kind::BytesMap,
    },
    Spec {
        name: "bundle",
        kind: Kind::Bytes,
    },
];

impl ApprovedSetup {
    pub fn validate(&self) -> Result<(), String> {
        if self.files.is_empty() || self.files.len() > MAX_APPROVED_FILES {
            return Err("invalid approved file set".to_string());
        }
        if !self.files.contains_key(FACTORY_SETUP_ENTRY) {
            return Err("approved setup entrypoint is required".to_string());
        }
        if !self.files.contains_key(FACTORY_CHECK_ENTRY) {
            return Err("approved check entrypoint is required".to_string());
        }
        // Go iterates the map in random order; sort so multi-fault reports
        // are deterministic (single faults agree either way).
        let mut names: Vec<&String> = self.files.keys().collect();
        names.sort();
        let mut total = 0usize;
        for name in names {
            let contents = &self.files[name];
            if !valid_approved_name(name) {
                return Err("invalid approved file name".to_string());
            }
            if contents.is_empty() || contents.len() > MAX_APPROVED_FILE_SIZE {
                return Err("invalid approved file size".to_string());
            }
            total += contents.len();
        }
        if total > MAX_APPROVED_TOTAL {
            return Err("approved inputs exceed the bounded size".to_string());
        }
        if self.bundle.is_empty() || self.bundle.len() > MAX_SOURCE_BUNDLE {
            return Err("invalid source bundle size".to_string());
        }
        Ok(())
    }

    pub fn from_map(m: &BoundMap) -> Self {
        ApprovedSetup {
            files: m.take_bytes_map("files"),
            bundle: m.take_bytes("bundle"),
        }
    }
}

/// `SetupDigestOf`: sha256 over each file name, a zero byte, then contents,
/// names sorted; lowercase hex.
pub fn setup_digest_of(files: &HashMap<String, Vec<u8>>) -> String {
    let mut names: Vec<&String> = files.keys().collect();
    names.sort();
    let mut input = Vec::new();
    for name in names {
        input.extend_from_slice(name.as_bytes());
        input.push(0);
        input.extend_from_slice(&files[name]);
    }
    sha256::hex_lower(&sha256::digest(&input))
}
