use soda_release_deliver::fetch::{fetch, init_state};
use soda_release_deliver::model::Highwater;
use soda_release_deliver::native::Runner;
use soda_release_deliver::publish::Ledger;

use super::{decode_trust, goldens, private_dir};

#[test]
fn state_and_ledger_init_round_trip() {
    let g = goldens();
    let trust = decode_trust(&g);
    let dir = private_dir("srd-state");
    let state_path = format!("{dir}/state.json");
    init_state(&state_path, &trust).expect("init state");
    let raw = std::fs::read(&state_path).unwrap();
    let state: Highwater = serde_json::from_slice(&raw).unwrap();
    assert_eq!(state.format, 1);
    assert_eq!(state.trust_epoch, trust.epoch);
    assert!(state.checked_at > 0);
    let ledger_path = format!("{dir}/ledger.json");
    let repo = format!("{}-release", trust.prefix);
    soda_release_deliver::publish::init_ledger(&ledger_path, &trust, &repo).expect("init ledger");
    let raw = std::fs::read(&ledger_path).unwrap();
    let ledger: Ledger = serde_json::from_slice(&raw).unwrap();
    assert_eq!(ledger.phase, "idle");
    ledger.validate(&trust).unwrap();
}

/// Scripted runner proving fetch wiring without a network.
struct ScriptRunner {
    calls: std::sync::Mutex<Vec<Vec<String>>>,
    inspect_raw: Vec<u8>,
    copy_manifest: Vec<u8>,
    copy_config: Vec<u8>,
}

impl Runner for ScriptRunner {
    fn run(&self, args: &[&str]) -> Result<Vec<u8>, soda_release_deliver::Error> {
        self.calls
            .lock()
            .unwrap()
            .push(args.iter().map(|s| s.to_string()).collect());
        if args.contains(&"inspect") && args.contains(&"--raw") {
            return Ok(self.inspect_raw.clone());
        }
        if args.contains(&"inspect") && args.contains(&"--config") {
            return Ok(self.copy_config.clone());
        }
        if args.contains(&"copy") {
            // Materialize the destination dir copy the verifier reads.
            for arg in args {
                if let Some(dest) = arg.strip_prefix("dir:") {
                    // Destination layout: <dest>/manifest.json plus blobs.
                    std::fs::create_dir_all(dest).unwrap();
                    std::fs::write(format!("{dest}/manifest.json"), &self.copy_manifest).unwrap();
                }
            }
            return Ok(Vec::new());
        }
        Err(soda_release_deliver::Error::unavailable())
    }
}

#[test]
fn fetch_rejects_bad_runner_output() {
    let g = goldens();
    let trust = decode_trust(&g);
    let dir = private_dir("srd-fetch");
    let state_path = format!("{dir}/state.json");
    init_state(&state_path, &trust).unwrap();
    let runner = ScriptRunner {
        calls: std::sync::Mutex::new(Vec::new()),
        inspect_raw: b"not-a-manifest".to_vec(),
        copy_manifest: b"{}".to_vec(),
        copy_config: b"{}".to_vec(),
    };
    let out = format!("{dir}/out");
    let result = fetch(&runner, &trust, "candidate", "x86_64", &state_path, &out, 0);
    assert!(result.is_err());
    assert!(!runner.calls.lock().unwrap().is_empty());
}
