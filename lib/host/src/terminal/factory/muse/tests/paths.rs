use super::common::{muse_lease, muse_run, muse_run_dir, PREP, RID, ROLE};
use crate::terminal::factory::tcodex::{self, FactoryRun};
use crate::terminal::factory::tmuse::*;
use crate::terminal::{Binding, ERR_DENIED};

#[test]
fn paths_and_guest() {
    let run = muse_run();
    let p = factory_muse_paths(&run).unwrap();
    assert_eq!(p.checkout, format!("/home/{ROLE}/checkouts/{PREP}"));
    assert!(p.run_dir.ends_with(&format!("/runs/{RID}")));
    assert_eq!(p.home, format!("{}/home", p.run_dir));
    assert_eq!(p.muse_config, format!("{}/home/.config/muse", p.run_dir));
    assert_eq!(p.prompt, format!("{}/prompt", p.run_dir));
    assert_eq!(p.output, format!("{}/last-message.txt", p.run_dir));
    assert_eq!(p.stdout, format!("{}/stdout.log", p.run_dir));
    assert_eq!(p.auth, format!("{}/home/.config/muse/auth.json", p.run_dir));
    assert_eq!(p.credential, format!("{}/muse-auth.json", p.run_dir));
    assert_eq!(p.guest, "/usr/local/bin/muse-factory-1.4.2");
    assert_eq!(
        factory_muse_guest("1.4.2").unwrap(),
        "/usr/local/bin/muse-factory-1.4.2"
    );
    assert!(factory_muse_guest("bad vers!").is_none());
    // The codex family never takes muse paths.
    let codex = FactoryRun {
        harness: tcodex::FACTORY_HARNESS_CODEX.to_string(),
        harness_vers: "0.153.4".to_string(),
        ..muse_run()
    };
    assert_eq!(factory_muse_paths(&codex).unwrap_err(), ERR_DENIED);
}

#[test]
fn binding_gates() {
    let lease = muse_lease();
    let p = factory_muse_binding(&lease).unwrap();
    assert_eq!(p.run_dir, muse_run_dir());
    for mutate in [
        Box::new(|b: &mut Binding| b.scope = tcodex::FACTORY_SCOPE_CODEX.to_string())
            as Box<dyn Fn(&mut Binding)>,
        Box::new(|b: &mut Binding| b.kind = "terminal".to_string()),
        Box::new(|b: &mut Binding| b.id = "short".to_string()),
        Box::new(|b: &mut Binding| b.project = "x".to_string()),
        Box::new(|b: &mut Binding| b.login = "dev".to_string()),
        Box::new(|b: &mut Binding| b.uid = 0),
        Box::new(|b: &mut Binding| b.gid = 0),
        Box::new(|b: &mut Binding| b.generation = 99),
        Box::new(|b: &mut Binding| b.invocation_id = "x".to_string()),
        Box::new(|b: &mut Binding| b.child_id = "x".to_string()),
        Box::new(|b: &mut Binding| b.credential_root = "/elsewhere".to_string()),
    ] {
        let mut broken = lease.clone();
        mutate(broken.binding.as_mut().unwrap());
        assert_eq!(factory_muse_binding(&broken).unwrap_err(), ERR_DENIED);
    }
    let mut broken = lease.clone();
    broken.provider_id = "codex".to_string();
    assert_eq!(factory_muse_binding(&broken).unwrap_err(), ERR_DENIED);
    broken = lease.clone();
    broken.kind = "terminal".to_string();
    assert_eq!(factory_muse_binding(&broken).unwrap_err(), ERR_DENIED);
}
