use std::collections::{HashMap, HashSet};
use std::sync::atomic::AtomicU64;

use super::super::*;

static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

pub(super) const FIXTURE: &str =
    include_str!("../../../../lib/release-inputs/fixtures/payload-v4.json");

pub(super) fn test_root(name: &str) -> PathBuf {
    let dir = std::env::temp_dir().join(format!(
        "soda-image-import-test-{}-{}-{name}",
        std::process::id(),
        TEST_SEQ.fetch_add(1, Ordering::SeqCst)
    ));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).expect("test root");
    dir
}

pub(super) fn test_ctx(cancelled: &AtomicBool) -> ImportCtx<'_> {
    ImportCtx {
        deadline: Instant::now() + Duration::from_secs(600),
        cancelled,
    }
}

pub(super) fn write_release(root: &Path) -> PathBuf {
    let path = root.join("release.json");
    fs::write(&path, FIXTURE).expect("write release");
    path
}

/// Fake rootful store: present configs answer `exists`, pulls record their
/// reference and stage the config, inspects report the sealed platform.
pub(super) struct FakeStore {
    present: HashSet<String>,
    by_reference: HashMap<String, String>,
    pub(super) pulls: Vec<String>,
    pub(super) queries: usize,
    bad_inspect: HashSet<String>,
}

impl FakeStore {
    pub(super) fn new(payload: &Payload, present: &[&str]) -> FakeStore {
        let mut by_reference = HashMap::new();
        for set in [ImageSet::Required, ImageSet::Tailnet] {
            for (_, binding) in super::super::import::selected(payload, set) {
                by_reference.insert(binding.reference.clone(), binding.config.clone());
            }
        }
        FakeStore {
            present: present.iter().map(|s| s.to_string()).collect(),
            by_reference,
            pulls: Vec::new(),
            queries: 0,
            bad_inspect: HashSet::new(),
        }
    }

    pub(super) fn corrupt_inspect(&mut self, config: &str) {
        self.bad_inspect.insert(config.to_string());
    }

    pub(super) fn run(&mut self, cmd: &str, args: &[String]) -> (PodmanOutcome, Vec<u8>) {
        assert_eq!(cmd, "/usr/bin/podman");
        assert_eq!(args[0], "--remote=false");
        match args[1].as_str() {
            "image" if args[2] == "exists" => {
                assert_eq!(args.len(), 4);
                self.queries += 1;
                let code = if self.present.contains(&args[3]) {
                    0
                } else {
                    1
                };
                (PodmanOutcome::Code(code), Vec::new())
            }
            "pull" => {
                assert_eq!(
                    args[..3].to_vec(),
                    vec![
                        "--remote=false".to_string(),
                        "pull".to_string(),
                        "--retry=0".to_string(),
                    ]
                );
                assert_eq!(args.len(), 4);
                let config = self.by_reference.get(&args[3]).cloned().unwrap_or_default();
                self.pulls.push(args[3].clone());
                self.present.insert(config);
                (PodmanOutcome::Code(0), Vec::new())
            }
            "image" if args[2] == "inspect" => {
                assert_eq!(args.len(), 4);
                let config = &args[3];
                let (os, arch) = if self.bad_inspect.contains(config) {
                    ("linux", "arm64")
                } else {
                    ("linux", "amd64")
                };
                let doc = format!(
                    "[{{\"Id\":\"{config}\",\"Os\":\"{os}\",\"Architecture\":\"{arch}\"}}]"
                );
                (PodmanOutcome::Code(0), doc.into_bytes())
            }
            other => panic!("unexpected native operation: {other}"),
        }
    }
}
