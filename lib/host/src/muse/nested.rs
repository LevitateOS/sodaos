use std::time::Instant;

use super::{
    muse_child_pid, muse_mapped_uid, muse_peer_alive, muse_readonly_mount, muse_registration_valid,
    MuseCaller, MuseHooks, MuseNested, MusePeer, MuseRuntime, NestedRegistration,
    MUSE_CHILD_INSPECT,
};
use crate::domain;
use crate::json;
use crate::project::Executor;
use crate::terminal;

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    /// `MuseRuntime.RegisterNested`: bind a current project-root registration.
    pub fn register_nested(
        &self,
        peer: &MusePeer,
        input: &NestedRegistration,
        deadline: Instant,
    ) -> Result<(), String> {
        if !muse_registration_valid(input) {
            return Err(terminal::err_denied());
        }
        let caller = self.resolve_project(peer, deadline)?;
        if !self.registration_authority(&caller, input.actor_id, deadline) {
            return Err(terminal::err_denied());
        }
        let record = self.registered_child(&caller, input, deadline)?;
        self.validate_nested(&record, deadline)?;
        if !muse_peer_alive(peer) {
            return Err(terminal::err_denied());
        }
        self.nested
            .lock()
            .unwrap()
            .insert(record.namespace.clone(), record);
        Ok(())
    }

    fn registration_authority(&self, caller: &MuseCaller, actor: i64, deadline: Instant) -> bool {
        if caller.uid != 0 {
            return false;
        }
        if self
            .hooks
            .nested_authorize(actor, &caller.project, deadline)
            .is_err()
        {
            return false;
        }
        let ns = match std::fs::read_link(format!("/proc/{}/ns/pid", caller.project_pid)) {
            Ok(ns) => ns.to_string_lossy().into_owned(),
            Err(_) => return false,
        };
        if caller.namespace != ns {
            return false;
        }
        self.actor_account(&caller.container, actor, deadline)
            .is_ok()
    }

    fn registered_child(
        &self,
        caller: &MuseCaller,
        input: &NestedRegistration,
        deadline: Instant,
    ) -> Result<MuseNested, String> {
        let body = self.guest_refs(
            &caller.container,
            &[],
            &[
                "/usr/bin/podman",
                "--remote=false",
                "inspect",
                "--format",
                MUSE_CHILD_INSPECT,
                &input.child_id,
            ],
            deadline,
        );
        let (body, failed) = match body {
            Ok(body) => (body, false),
            Err(_) => (Vec::new(), true),
        };
        let pid = muse_child_pid(&body, failed, &input.child_id)?;
        let body = self
            .guest_refs(
                &caller.container,
                &[],
                &["/usr/bin/readlink", &format!("/proc/{pid}/ns/pid")],
                deadline,
            )
            .map_err(|_| terminal::err_denied())?;
        let ns = String::from_utf8_lossy(&body).trim().to_string();
        if !ns.starts_with("pid:[") || ns == caller.namespace {
            return Err(terminal::err_denied());
        }
        Ok(MuseNested {
            parent: caller.container.clone(),
            project: caller.project.clone(),
            child: input.child_id.clone(),
            namespace: ns,
            registration: input.registration_id.clone(),
            actor: input.actor_id,
            pid,
            muse: input.muse,
        })
    }

    pub(in crate::muse) fn validate_nested(
        &self,
        record: &MuseNested,
        deadline: Instant,
    ) -> Result<(), String> {
        let body = self.guest_refs(
            &record.parent,
            &[],
            &[
                "/usr/bin/podman",
                "--remote=false",
                "inspect",
                "--format",
                "{{.ID}} {{.State.Pid}} {{.State.Running}}",
                &record.child,
            ],
            deadline,
        );
        let Ok(body) = body else {
            return Err(terminal::err_denied());
        };
        if String::from_utf8_lossy(&body).trim() != format!("{} {} true", record.child, record.pid)
        {
            return Err(terminal::err_denied());
        }
        self.nested_namespace(record, deadline)?;
        let body = self.guest_refs(
            &record.parent,
            &[],
            &[
                "/usr/bin/podman",
                "--remote=false",
                "inspect",
                "--format",
                "{{json .Mounts}}",
                &record.child,
            ],
            deadline,
        );
        let (body, failed) = match body {
            Ok(body) => (body, false),
            Err(_) => (Vec::new(), true),
        };
        if record.muse
            && muse_readonly_mount(
                &body,
                failed,
                &format!("/run/soda-muse/nested/{}", record.registration),
                "/run/soda-muse/credentials",
            )
            .is_err()
        {
            return Err(terminal::err_denied());
        }
        Ok(())
    }

    fn nested_namespace(&self, record: &MuseNested, deadline: Instant) -> Result<(), String> {
        let body = self
            .guest_refs(
                &record.parent,
                &[],
                &["/usr/bin/readlink", &format!("/proc/{}/ns/pid", record.pid)],
                deadline,
            )
            .map_err(|_| terminal::err_denied())?;
        if String::from_utf8_lossy(&body).trim() != record.namespace {
            return Err(terminal::err_denied());
        }
        let body = self
            .guest_refs(
                &record.parent,
                &[],
                &[
                    "/usr/bin/readlink",
                    &format!("/proc/{}/ns/user", record.pid),
                    "/proc/1/ns/user",
                ],
                deadline,
            )
            .map_err(|_| terminal::err_denied())?;
        let text = String::from_utf8_lossy(&body);
        let names: Vec<&str> = text.split_whitespace().collect();
        if names.len() != 2 || names[0] != names[1] {
            return Err(terminal::err_denied());
        }
        Ok(())
    }

    /// `actorAccount`: guest username/home for a broker actor.
    /// Decoded tolerantly (`encoding/json`) like Go.
    pub fn actor_account(
        &self,
        container: &str,
        actor: i64,
        deadline: Instant,
    ) -> Result<(String, String), String> {
        let body = self.guest_refs(
            container,
            &[],
            &["/usr/local/bin/muse", "--soda-account", &actor.to_string()],
            deadline,
        );
        let Ok(body) = body else {
            return Err(terminal::err_denied());
        };
        if body.len() > 4096 {
            return Err(terminal::err_denied());
        }
        #[derive(serde::Deserialize, Default)]
        struct AccountWire {
            #[serde(rename = "Username")]
            username: Option<String>,
            #[serde(rename = "HomeDir")]
            home_dir: Option<String>,
        }
        let account: AccountWire =
            json::decode_tolerant_as(&body).map_err(|_| terminal::err_denied())?;
        let username = account.username.unwrap_or_default();
        if !domain::valid_login(&username) || username == "root" {
            return Err(terminal::err_denied());
        }
        Ok((username, account.home_dir.unwrap_or_default()))
    }

    pub(in crate::muse) fn nested_caller(
        &self,
        mut caller: MuseCaller,
        peer: &MusePeer,
        deadline: Instant,
    ) -> Result<MuseCaller, String> {
        let (username, home) = self.actor_account(&caller.container, caller.actor, deadline)?;
        caller.login = username;
        caller.home = home;
        let mappings = std::fs::read_to_string(format!("/proc/{}/gid_map", peer.pid))
            .map_err(|_| terminal::err_denied())?;
        caller.gid = muse_mapped_uid(&mappings, peer.gid)?;
        if !self.authorized_caller(&caller, peer, deadline) {
            return Err(terminal::err_denied());
        }
        Ok(caller)
    }
}
