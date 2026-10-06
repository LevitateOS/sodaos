//! Single-use run enrollment: credential checks and project auth-key minting.
//!
//! PR26 port of `internal/tailnet/enrollment.go` (`EnrollRun`, `projectKey`,
//! `tokenFromEnrollment`, `createProjectAuthKey`) and the `control.go`
//! credential check (`checkCredential`). Neither callback nor projection
//! ever sees the OAuth secret or bearer token; transient copies are zeroed
//! after use.

use std::time::{Duration, Instant, SystemTime};

use crate::tailnet_domain;
use crate::tcontrol_policy::{Credential, EnrollmentPolicy, PolicyStore};
use crate::tcontrol_provider as provider;
use crate::tcontrol_wire as wire;

/// Provider dispatch: production shells out to curl, tests inject a stub.
#[derive(Clone, Copy)]
pub enum Provider<'a> {
    Curl {
        exec: &'a dyn crate::project::Executor,
        curl: &'a str,
    },
    Stub(&'a provider::ProviderTransport),
}

impl Provider<'_> {
    fn token(
        &self,
        client_id: &str,
        client_secret: &str,
        tags: &[String],
        deadline: Instant,
    ) -> Result<(u16, Vec<u8>), String> {
        match *self {
            Provider::Curl { exec, curl } => {
                provider::fetch_token(exec, curl, client_id, client_secret, tags, deadline)
            }
            Provider::Stub(t) => t(
                provider::ProviderRequest::Token {
                    client_id: client_id.to_string(),
                    client_secret: client_secret.to_string(),
                    tags: tags.to_vec(),
                },
                deadline,
            ),
        }
    }

    fn key(
        &self,
        tailnet: &str,
        tags: &[String],
        preauthorized: bool,
        token: &str,
        deadline: Instant,
    ) -> Result<(u16, Vec<u8>), String> {
        match *self {
            Provider::Curl { exec, curl } => {
                provider::create_key(exec, curl, tailnet, tags, preauthorized, token, deadline)
            }
            Provider::Stub(t) => t(
                provider::ProviderRequest::KeyCreate {
                    tailnet: tailnet.to_string(),
                    tags: tags.to_vec(),
                    preauthorized,
                    token: token.to_string(),
                },
                deadline,
            ),
        }
    }
}

/// Verify a credential against the provider without mutating anything,
/// mirroring `checkCredential`. Every failure is unavailable.
pub fn check_credential(
    provider: &Provider,
    r: &wire::EnrollmentRequest,
    deadline: Instant,
) -> Result<(), String> {
    if Instant::now() >= deadline {
        return Err(wire::err_unavailable());
    }
    let tags = r.tags.clone().unwrap_or_default();
    let (status, body) = provider
        .token(&r.client_id, &r.client_secret, &tags, deadline)
        .map_err(|_| wire::err_unavailable())?;
    let mut token = provider::validate_token(status, &body, false, &wire::err_unavailable)
        .map_err(|_| wire::err_unavailable())?;
    wire::zero_string(&mut token);
    Ok(())
}

/// Mint one single-use project auth key, mirroring `projectKey`.
/// The credential copies are zeroed before return, like Go.
pub fn project_key(
    provider: &Provider,
    policy: &EnrollmentPolicy,
    credential: &Credential,
    deadline: Instant,
) -> Result<String, String> {
    let mut probe = wire::EnrollmentRequest {
        action: "save".to_string(),
        revision: policy.revision.clone(),
        tailnet: policy.tailnet.clone(),
        tags: Some(policy.tags.clone()),
        preauthorized: Some(policy.preauthorized),
        client_id: credential.client_id.clone(),
        client_secret: credential.secret.clone(),
        default: None,
    };
    if probe.validate().is_err() {
        probe.zero_secret();
        return Err(wire::err_unavailable());
    }
    if Instant::now() >= deadline {
        probe.zero_secret();
        return Err(wire::err_unconfirmed());
    }
    let inner = deadline.min(Instant::now() + Duration::from_secs(20));
    let (status, body) = provider
        .token(
            &probe.client_id,
            &probe.client_secret,
            &probe.tags.clone().unwrap_or_default(),
            inner,
        )
        .map_err(|_| wire::err_unconfirmed())?;
    probe.zero_secret();
    let mut token = provider::validate_token(status, &body, true, &wire::err_unconfirmed)
        .map_err(|_| wire::err_unconfirmed())?;
    let before = SystemTime::now();
    let fetched = provider.key(
        &policy.tailnet,
        &policy.tags,
        policy.preauthorized,
        &token,
        inner,
    );
    wire::zero_string(&mut token);
    let (status, body) = fetched.map_err(|_| wire::err_unconfirmed())?;
    provider::validate_key(
        status,
        &body,
        &policy.tags,
        policy.preauthorized,
        before,
        SystemTime::now(),
    )
}

fn admit_enroll_run(
    policy: &EnrollmentPolicy,
    project: &crate::tcontrol_policy::ProjectPolicyEntry,
    recheck: &dyn Fn(Instant) -> Result<(), String>,
    deadline: Instant,
) -> Result<(), String> {
    if !project.enabled
        || !policy.admission
        || project.binding != policy.binding
        || policy.revision == "0"
    {
        return Err(wire::err_conflict());
    }
    if recheck(deadline).is_err() || Instant::now() >= deadline {
        return Err(wire::err_conflict());
    }
    Ok(())
}

/// Enroll one run incarnation, mirroring `Control.EnrollRun`. A root-native
/// operation, not an HTTP credential/key endpoint: `recheck` re-validates
/// the exact incarnation, `consume` receives only the single-use key.
pub fn enroll_run(
    store: &PolicyStore,
    provider: &Provider,
    target: &tailnet_domain::RunTarget,
    recheck: &dyn Fn(Instant) -> Result<(), String>,
    consume: &dyn Fn(Instant, &str) -> Result<(), String>,
    deadline: Instant,
) -> Result<(), String> {
    if !tailnet_domain::valid_project_id(&target.project)
        || !tailnet_domain::valid_container_id(&target.container)
        || !tailnet_domain::valid_container_id(&target.run)
    {
        return Err(wire::err_invalid());
    }
    let capped = deadline.min(Instant::now() + Duration::from_secs(30));
    let lock = match store.lock(capped, false) {
        Ok(Some(l)) => l,
        Ok(None) => return Err(wire::err_conflict()),
        Err(e) => return Err(e),
    };
    let mut policy = store.load(&lock)?;
    let project = store.load_project(&lock, &target.project, &target.container)?;
    admit_enroll_run(&policy, &project, recheck, capped)?;
    // Any key-minting failure (including the policy probe) is an
    // unconfirmed submission, like Go's `confirmEnrollSubmission`.
    let mut key = project_key(provider, &policy, &policy.credential, capped)
        .map_err(|_| wire::err_unconfirmed())?;
    policy.credential.zero_secret();
    // Neither callback receives the OAuth secret/token.
    let mut outcome = if recheck(capped).is_err() || Instant::now() >= capped {
        Err(wire::err_unconfirmed())
    } else {
        consume(capped, &key).map_err(|_| wire::err_unconfirmed())
    };
    wire::zero_string(&mut key);
    // Submission is not enrollment/approval/reachability confirmation.
    if outcome.is_ok() && (recheck(capped).is_err() || Instant::now() >= capped) {
        outcome = Err(wire::err_unconfirmed());
    }
    outcome
}
