use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Spec {
    pub id: String,
    pub revision: i64,
    pub name: String,
    pub artifact_sha256: String,
    pub bundle_sha256: String,
    pub tools_sha256: String,
    pub runtime_image: String,
    pub image: String,
    pub data: String,
    pub owner_team: String,
    pub origin: String,
    pub port: u16,
}

pub fn hex(value: &str, size: usize) -> bool {
    value.len() == size
        && value
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}

pub fn id(value: &str) -> bool {
    value.strip_prefix('a').is_some_and(|tail| hex(tail, 24))
}

impl Spec {
    fn valid_hostname(hostname: &str) -> bool {
        hostname.contains('.')
            && hostname.trim_end_matches('.').split('.').all(|label| {
                !label.is_empty()
                    && label.len() <= 63
                    && !label.starts_with('-')
                    && !label.ends_with('-')
                    && label
                        .bytes()
                        .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
            })
    }

    fn valid_identity(&self) -> bool {
        id(&self.id)
            && id(&self.data)
            && self.revision >= 1
            && !self.name.is_empty()
            && self.name.len() <= 128
            && self
                .name
                .bytes()
                .all(|b| b.is_ascii_alphanumeric() || b"_.-".contains(&b))
    }

    fn valid_digests(&self) -> bool {
        hex(&self.artifact_sha256, 64)
            && hex(&self.bundle_sha256, 64)
            && hex(&self.tools_sha256, 64)
            && self
                .image
                .strip_prefix("sha256:")
                .is_some_and(|s| hex(s, 64))
            && self
                .runtime_image
                .strip_prefix("sha256:")
                .is_some_and(|s| hex(s, 64))
    }

    fn valid_team(&self, team: &str) -> bool {
        self.owner_team.len() == 36
            && hex(team, 32)
            && [8, 13, 18, 23]
                .iter()
                .all(|i| self.owner_team.as_bytes()[*i] == b'-')
    }

    fn valid_origin(&self, origin: &url::Url, hostname_ok: bool) -> bool {
        origin.scheme() == "https"
            && origin.username().is_empty()
            && origin.password().is_none()
            && origin.query().is_none()
            && origin.fragment().is_none()
            && origin.path() == "/"
            && origin.origin().ascii_serialization() == self.origin
            && hostname_ok
            && (20000..30000).contains(&self.port)
    }

    pub fn validate(&self) -> Result<(), String> {
        let origin = url::Url::parse(&self.origin).map_err(|_| "invalid application origin")?;
        let hostname = match origin.host() {
            Some(url::Host::Domain(host)) => host,
            _ => "",
        };
        let hostname_ok = Self::valid_hostname(hostname);
        let team = self.owner_team.replace('-', "");
        if !self.valid_identity()
            || !self.valid_digests()
            || !self.valid_team(&team)
            || !self.valid_origin(&origin, hostname_ok)
        {
            return Err("invalid admitted application references".into());
        }
        Ok(())
    }
}

#[derive(Clone, Debug, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Request {
    pub application: Spec,
    pub action: String,
    #[serde(default)]
    pub expected_container: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub restore: Option<RestoreOperation>,
}

#[derive(Clone, Debug, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct RestoreOperation {
    pub id: String,
    pub snapshot: String,
    pub previous: Spec,
    pub next: Spec,
    pub expected_container: String,
    pub state: String,
}

impl RestoreOperation {
    pub fn validate(&self) -> Result<(), String> {
        self.previous.validate()?;
        self.next.validate()?;
        let mut expected = self.previous.clone();
        expected.revision = expected
            .revision
            .checked_add(1)
            .ok_or("application revision exhausted")?;
        expected.data = self.next.data.clone();
        if !id(&self.id)
            || !id(&self.snapshot)
            || !hex(&self.expected_container, 64)
            || self.state != "pending"
            || self.previous.data == self.next.data
            || expected != self.next
        {
            return Err("invalid application restore intent".into());
        }
        Ok(())
    }
}

impl Request {
    pub fn validate(&self) -> Result<(), String> {
        self.application.validate()?;
        if ![
            "install",
            "inspect",
            "start",
            "stop",
            "replace",
            "restore",
            "restore_abort",
        ]
        .contains(&self.action.as_str())
            || (self.action == "install" && !self.expected_container.is_empty())
            || (self.action != "install"
                && self.action != "inspect"
                && !hex(&self.expected_container, 64))
            || (!self.expected_container.is_empty() && !hex(&self.expected_container, 64))
        {
            return Err("invalid application operation or generation".into());
        }
        if matches!(self.action.as_str(), "restore" | "restore_abort") {
            let restore = self.restore.as_ref().ok_or("restore intent is required")?;
            restore.validate()?;
            let expected = if self.action == "restore" {
                &restore.next
            } else {
                &restore.previous
            };
            if &self.application != expected
                || self.expected_container != restore.expected_container
            {
                return Err("application differs from restore intent".into());
            }
        } else if self.restore.is_some() {
            return Err("restore intent is not valid for this operation".into());
        }
        Ok(())
    }
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Observation {
    pub id: String,
    pub revision: i64,
    pub container: String,
    pub image: String,
    pub running: bool,
    pub ready: bool,
    pub origin: String,
}
