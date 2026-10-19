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
