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
    pub fn validate(&self) -> Result<(), String> {
        let origin = url::Url::parse(&self.origin).map_err(|_| "invalid application origin")?;
        let hostname = match origin.host() {
            Some(url::Host::Domain(host)) => host,
            _ => "",
        };
        let valid_hostname = hostname.contains('.')
            && hostname.trim_end_matches('.').split('.').all(|label| {
                !label.is_empty()
                    && label.len() <= 63
                    && !label.starts_with('-')
                    && !label.ends_with('-')
                    && label
                        .bytes()
                        .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
            });
        let team = self.owner_team.replace('-', "");
        if !id(&self.id)
            || !id(&self.data)
            || self.revision < 1
            || self.name.is_empty()
            || self.name.len() > 128
            || !self
                .name
                .bytes()
                .all(|b| b.is_ascii_alphanumeric() || b"_.-".contains(&b))
            || !hex(&self.artifact_sha256, 64)
            || !hex(&self.bundle_sha256, 64)
            || !hex(&self.tools_sha256, 64)
            || !self
                .image
                .strip_prefix("sha256:")
                .is_some_and(|s| hex(s, 64))
            || !self
                .runtime_image
                .strip_prefix("sha256:")
                .is_some_and(|s| hex(s, 64))
            || self.owner_team.len() != 36
            || !hex(&team, 32)
            || [8, 13, 18, 23]
                .iter()
                .any(|i| self.owner_team.as_bytes()[*i] != b'-')
            || origin.scheme() != "https"
            || !origin.username().is_empty()
            || origin.password().is_some()
            || origin.query().is_some()
            || origin.fragment().is_some()
            || origin.path() != "/"
            || origin.origin().ascii_serialization() != self.origin
            || !valid_hostname
            || !(20000..30000).contains(&self.port)
        {
            return Err("invalid admitted application references".into());
        }
        Ok(())
    }
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Request {
    pub application: Spec,
    pub action: String,
    #[serde(default)]
    pub expected_container: String,
}

impl Request {
    pub fn validate(&self) -> Result<(), String> {
        self.application.validate()?;
        if !["install", "inspect", "start", "stop", "replace"].contains(&self.action.as_str())
            || (self.action == "install" && !self.expected_container.is_empty())
            || (self.action != "install"
                && self.action != "inspect"
                && !hex(&self.expected_container, 64))
            || (!self.expected_container.is_empty() && !hex(&self.expected_container, 64))
        {
            return Err("invalid application operation or generation".into());
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
