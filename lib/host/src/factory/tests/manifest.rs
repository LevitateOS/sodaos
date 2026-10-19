//! D05 manifest scope: the immutable execution-manifest envelope binds
//! plan bytes/digest, invocation, phase/run/deadline/profile and child
//! limit (implement 1-4, correct singleton 0), and verifies against its
//! launching run. FW14 stages this envelope at dispatch; these tests pin
//! the contract it must produce.

use crate::factory::*;
use crate::sha256;

fn plan_digest() -> String {
    sha256::hex_lower(&sha256::digest(b"plan-bytes"))
}

fn envelope_json() -> String {
    format!(
        "{{\"plan\":\"cGxhbi1ieXRlcw==\",\"plan_digest\":\"{}\",\"invocation\":\"{}\",\"phase\":\"implement\",\"run\":\"{}\",\"deadline\":\"2030-06-07T08:09:10Z\",\"effort\":\"xhigh\",\"contributor\":true,\"max_children\":3}}",
        plan_digest(),
        "09".repeat(16),
        "a".repeat(32),
    )
}

fn implement_run_for(staged: &[u8]) -> FactoryRun {
    FactoryRun {
        deadline: "2030-06-07T08:09:10Z".to_string(),
        actor: 7,
        id: "a".repeat(32),
        project: format!("p{}", "b".repeat(24)),
        role: "soda-coder".to_string(),
        preparation: format!("f{}", "c".repeat(24)),
        harness: "muse".to_string(),
        harness_vers: "v1".to_string(),
        model: String::new(),
        assignment: "0".repeat(64),
        source_commit: "e".repeat(40),
        connection: "conn-1".to_string(),
        execution_phase: "implement".to_string(),
        manifest_digest: sha256::hex_lower(&sha256::digest(staged)),
    }
}

#[test]
fn manifest_envelope_binds_its_run() {
    let staged = envelope_json().into_bytes();
    let manifest = ExecutionManifest::decode(&staged).unwrap();
    assert!(manifest.validate().is_ok());
    let run = implement_run_for(&staged);
    assert!(run.validate().is_ok());
    assert!(manifest.verify_for_run(&run, &staged).is_ok());
}

#[test]
fn manifest_envelope_refusals() {
    let good = envelope_json();
    let cases = [
        ("empty plan", good.replace("cGxhbi1ieXRlcw==", "")),
        (
            "plan digest drift",
            good.replace(&plan_digest(), &"0".repeat(64)),
        ),
        ("non-implement", good.replace("\"implement\"", "\"review\"")),
        ("bad run", good.replace(&"a".repeat(32), "short")),
        ("bad invocation", good.replace(&"09".repeat(16), "")),
        ("bad effort", good.replace("\"xhigh\"", "\"turbo\"")),
        (
            "zero children",
            good.replace("\"max_children\":3", "\"max_children\":0"),
        ),
        (
            "children above cap",
            good.replace("\"max_children\":3", "\"max_children\":5"),
        ),
        ("unknown field", good.replace("}", ",\"future\":1}")),
        ("missing children", good.replace(",\"max_children\":3", "")),
    ];
    for (name, body) in cases {
        let decoded = ExecutionManifest::decode(body.as_bytes());
        let valid = decoded.map(|m| m.validate().is_ok()).unwrap_or(false);
        assert!(!valid, "{name} validated");
    }
}

/// Lane C15: correct runs carry a singleton manifest with `max_children`
/// 0, bound to the launching run exactly like implement.
#[test]
fn manifest_correct_singleton_binds_its_run() {
    assert!(execution_manifest_required("implement"));
    assert!(execution_manifest_required("correct"));
    assert!(!execution_manifest_required("review"));
    assert!(!execution_manifest_required("investigate"));
    let body = envelope_json().replace("\"phase\":\"implement\"", "\"phase\":\"correct\"");
    let body = body.replace("\"max_children\":3", "\"max_children\":0");
    let staged = body.into_bytes();
    let manifest = ExecutionManifest::decode(&staged).unwrap();
    assert!(manifest.validate().is_ok());
    let mut run = implement_run_for(&staged);
    run.execution_phase = "correct".to_string();
    assert!(run.validate().is_ok());
    assert!(manifest.verify_for_run(&run, &staged).is_ok());
}

/// R10 parity: repair and verify join the admitted execution phases with
/// their prompt bounds and roles, and stay single-agent with no manifest.
#[test]
fn repair_verify_phase_parity() {
    assert!(valid_execution_phase("repair"));
    assert!(valid_execution_phase("verify"));
    assert!(!valid_execution_phase("publish"));
    assert_eq!(max_prompt_for_phase("repair"), MAX_REASSESS_PROMPT);
    assert_eq!(max_prompt_for_phase("verify"), MAX_FACTORY_PROMPT);
    assert!(!execution_manifest_required("repair"));
    assert!(!execution_manifest_required("verify"));
    let mut run = implement_run_for(&envelope_json().into_bytes());
    run.execution_phase = "repair".to_string();
    run.role = "soda-coder".to_string();
    run.manifest_digest = String::new();
    assert!(run.validate().is_ok());
    run.role = "soda-reviewer".to_string();
    assert!(run.validate().is_err());
    run.execution_phase = "verify".to_string();
    assert!(run.validate().is_ok());
    run.role = "soda-coder".to_string();
    assert!(run.validate().is_err());
}

#[test]
fn library_maintenance_uses_reviewer_without_manifest() {
    assert!(valid_execution_phase("maintain_library"));
    assert!(!execution_manifest_required("maintain_library"));
    let mut run = implement_run_for(&envelope_json().into_bytes());
    run.execution_phase = "maintain_library".to_string();
    run.manifest_digest.clear();
    run.role = "soda-reviewer".to_string();
    assert!(run.validate().is_ok());
    run.role = "soda-coder".to_string();
    assert!(run.validate().is_err());
}

#[test]
fn manifest_correct_singleton_refusals() {
    // Correct with child concurrency refuses.
    let with_children = envelope_json().replace("\"phase\":\"implement\"", "\"phase\":\"correct\"");
    let decoded = ExecutionManifest::decode(with_children.as_bytes()).unwrap();
    assert!(decoded.validate().is_err());
    // Implement with zero children still refuses.
    let zero = envelope_json().replace("\"max_children\":3", "\"max_children\":0");
    let decoded = ExecutionManifest::decode(zero.as_bytes()).unwrap();
    assert!(decoded.validate().is_err());
    // A correct run without its manifest digest refuses.
    let staged = envelope_json()
        .replace("\"phase\":\"implement\"", "\"phase\":\"correct\"")
        .replace("\"max_children\":3", "\"max_children\":0")
        .into_bytes();
    let manifest = ExecutionManifest::decode(&staged).unwrap();
    let mut run = implement_run_for(&staged);
    run.execution_phase = "correct".to_string();
    run.manifest_digest = String::new();
    assert!(run.validate().is_err());
    assert_eq!(
        run.validate().unwrap_err(),
        "correct run lacks its manifest digest"
    );
    let _ = manifest;
}

#[test]
fn manifest_run_binding_refusals() {
    let staged = envelope_json().into_bytes();
    let manifest = ExecutionManifest::decode(&staged).unwrap();
    let run = implement_run_for(&staged);
    // Digest drift refuses.
    let mut drifted = run.clone();
    drifted.manifest_digest = "0".repeat(64);
    assert!(manifest.verify_for_run(&drifted, &staged).is_err());
    // Wrong phase/run refuses.
    let mut wrong_phase = run.clone();
    wrong_phase.execution_phase = "review".to_string();
    assert!(manifest.verify_for_run(&wrong_phase, &staged).is_err());
    // Deadline drift refuses.
    let mut wrong_deadline = run.clone();
    wrong_deadline.deadline = "2030-06-07T08:09:11Z".to_string();
    assert!(manifest.verify_for_run(&wrong_deadline, &staged).is_err());
    // Contributor mode binds the Muse harness.
    let mut codex = run.clone();
    codex.harness = "codex".to_string();
    assert!(manifest.verify_for_run(&codex, &staged).is_err());
    // A non-implement run stages no envelope.
    let mut single = run.clone();
    single.execution_phase = "investigate".to_string();
    single.manifest_digest = String::new();
    assert!(manifest.verify_for_run(&single, &staged).is_err());
}
