use crate::terminal::factory::run::{
    valid_factory_role, valid_factory_run_id, valid_harness_version, valid_preparation_id,
    FactoryRun, FACTORY_HARNESS_MUSE, FACTORY_SCOPE_MUSE,
};
use crate::terminal::{self, Lease};

// ---------- run paths, scripts, bindings ----------

/// Fixed container paths for one supervised Muse Code run.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryMusePaths {
    pub checkout: String,
    pub run_dir: String,
    pub home: String,
    pub muse_config: String,
    pub prompt: String,
    pub marker: String,
    pub started: String,
    pub stop: String,
    pub pid_file: String,
    pub output: String,
    pub stdout: String,
    pub auth: String,
    pub credential: String,
    pub guest: String,
}

/// Validated run identities to fixed checkout/run/home/config paths.
pub fn factory_muse_run_paths(
    role: &str,
    preparation: &str,
    run: &str,
) -> Option<(String, String, String, String)> {
    if !valid_factory_role(role) || !valid_preparation_id(preparation) || !valid_factory_run_id(run)
    {
        return None;
    }
    let checkout = format!("/home/{role}/checkouts/{preparation}");
    let run_dir = format!("{checkout}/.soda-home/runs/{run}");
    let home = format!("{run_dir}/home");
    let muse_config = format!("{home}/.config/muse");
    Some((checkout, run_dir, home, muse_config))
}

/// `factoryCodexPaths` shape for Muse runs: validated run identities to
/// fixed paths. The guest pins the run's harness version.
pub fn factory_muse_paths(run: &FactoryRun) -> Result<FactoryMusePaths, String> {
    run.validate()?;
    if run.harness != FACTORY_HARNESS_MUSE {
        return Err(terminal::err_denied());
    }
    let (checkout, run_dir, home, muse_config) =
        factory_muse_run_paths(&run.role, &run.preparation, &run.id)
            .ok_or_else(terminal::err_denied)?;
    let guest = factory_muse_guest(&run.harness_vers).ok_or_else(terminal::err_denied)?;
    Ok(FactoryMusePaths {
        checkout,
        run_dir: run_dir.clone(),
        home: home.clone(),
        muse_config,
        prompt: format!("{run_dir}/prompt"),
        marker: format!("{run_dir}/marker"),
        started: format!("{run_dir}/started"),
        stop: format!("{run_dir}/stop"),
        pid_file: format!("{run_dir}/supervisor.pid"),
        output: format!("{run_dir}/last-message.txt"),
        stdout: format!("{run_dir}/stdout.log"),
        auth: format!("{home}/.config/muse/auth.json"),
        credential: format!("{run_dir}/muse-auth.json"),
        guest,
    })
}

/// `factoryCodexBinding` shape for Muse runs: supervised
/// factory-muse binding check plus derived run paths. The recorded
/// credential root must equal the derived run directory. Family policy
/// (factory role, generation) wraps the shared `binding` checks.
pub fn factory_muse_binding(lease: &Lease) -> Result<FactoryMusePaths, String> {
    let Some(b) = &lease.binding else {
        return Err(terminal::err_denied());
    };
    if !valid_factory_role(&b.login) {
        return Err(terminal::err_denied());
    }
    if b.uid <= 0 || b.gid <= 0 || b.generation != lease.generation || b.generation <= 0 {
        return Err(terminal::err_denied());
    }
    let (checkout, run_dir, home, muse_config) =
        crate::terminal::factory::binding::checked_binding_paths(
            lease,
            terminal::PROVIDER_MUSE,
            FACTORY_SCOPE_MUSE,
            factory_muse_run_paths,
        )?;
    Ok(FactoryMusePaths {
        checkout,
        run_dir: run_dir.clone(),
        home: home.clone(),
        muse_config,
        prompt: format!("{run_dir}/prompt"),
        marker: format!("{run_dir}/marker"),
        started: format!("{run_dir}/started"),
        stop: format!("{run_dir}/stop"),
        pid_file: format!("{run_dir}/supervisor.pid"),
        output: format!("{run_dir}/last-message.txt"),
        stdout: format!("{run_dir}/stdout.log"),
        auth: format!("{home}/.config/muse/auth.json"),
        credential: format!("{run_dir}/muse-auth.json"),
        guest: String::new(),
    })
}

/// `FactoryCodexGuest` shape for Muse runs: fixed versioned guest path
/// for staged harness bytes.
pub fn factory_muse_guest(version: &str) -> Option<String> {
    if !valid_harness_version(version) {
        return None;
    }
    Some(format!("/usr/local/bin/muse-factory-{version}"))
}
