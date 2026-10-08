//! Bounded IO for the shared live-input wire records.

use crate::files::write_new;
use crate::json_input::read_json;
use crate::Error;
use soda_build_tools::reader::stream::{valid_live_inputs, LiveInputs};
use std::path::Path;

/// Records one attempt's live inputs after shared shape validation.
pub fn write_live_inputs(path: &Path, inputs: &LiveInputs) -> Result<(), Error> {
    valid_live_inputs(inputs)?;
    let body = serde_json::to_string_pretty(inputs)
        .expect("serializing validated live inputs cannot fail")
        + "\n";
    write_new(path, body.as_bytes(), 0o644)
}

/// Admits controller-resolved inputs for the isolated worker.
pub fn read_live_inputs(path: &Path) -> Result<LiveInputs, Error> {
    let inputs: LiveInputs = read_json(path)?;
    valid_live_inputs(&inputs)?;
    Ok(inputs)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_support::fixture_live_inputs;

    #[test]
    fn oracle_live_inputs_round_trip() {
        let dir = std::env::temp_dir().join(format!(
            "soda-live-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("live-inputs.json");
        let inputs = fixture_live_inputs();
        write_live_inputs(&path, &inputs).unwrap();
        let bytes = std::fs::read(&path).unwrap();
        assert_eq!(bytes.last(), Some(&b'\n'));
        let text = std::str::from_utf8(&bytes).unwrap();
        assert!(text.find("\"ISO\"").unwrap() < text.find("\"QEMU\"").unwrap());
        assert_eq!(read_live_inputs(&path).unwrap(), inputs);

        let alias_path = dir.join("alias.json");
        let aliased = text.replacen("\"CoreOS\"", "\"coreos\"", 1);
        assert_ne!(aliased, text);
        std::fs::write(&alias_path, aliased).unwrap();
        assert!(read_live_inputs(&alias_path).is_err());

        let invalid_path = dir.join("invalid.json");
        let invalid_version = text.replace("\"Version\": \"1.98.2\"", "\"Version\": \"bad\"");
        assert_ne!(invalid_version, text);
        std::fs::write(&invalid_path, invalid_version).unwrap();
        assert!(read_live_inputs(&invalid_path).is_err());

        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            assert_eq!(
                std::fs::metadata(&path).unwrap().permissions().mode() & 0o777,
                0o644
            );
        }

        let mut bad = inputs.clone();
        bad.tailnet.version = "yesterday".to_string();
        assert!(write_live_inputs(&dir.join("bad.json"), &bad).is_err());
        let mut bad = inputs;
        bad.coreos.release = "tomorrow".to_string();
        assert!(write_live_inputs(&dir.join("bad2.json"), &bad).is_err());
        std::fs::remove_dir_all(dir).unwrap();
    }
}
