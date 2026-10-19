use std::io::Read;
use std::time::Duration;

use crate::system::{FORGEJO_RESPONSE_LIMIT, FORGEJO_TIMEOUT};
use serde::Deserialize;

#[derive(Debug)]
pub(crate) struct ForgejoUser {
    pub(crate) id: i64,
    pub(crate) admin: bool,
}

#[derive(Debug, Default)]
struct ForgejoUserResponse {
    id: i64,
    admin: bool,
}

impl<'de> Deserialize<'de> for ForgejoUserResponse {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        use serde::de::{IgnoredAny, MapAccess, Visitor};
        use std::fmt;

        struct UserVisitor;

        impl<'de> Visitor<'de> for UserVisitor {
            type Value = ForgejoUserResponse;

            fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                formatter.write_str("a Forgejo user object")
            }

            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut user = ForgejoUserResponse::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "id" => user.id = map.next_value()?,
                        "is_admin" => user.admin = map.next_value()?,
                        _ => {
                            map.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(user)
            }
        }

        deserializer.deserialize_map(UserVisitor)
    }
}

pub(crate) fn forgejo_get_user(base: &str, token: &str) -> Result<ForgejoUser, String> {
    let (status, body) = forgejo_request(base, "GET", "/api/v1/user", token)?;
    if !(200..300).contains(&status) {
        return Err(format!("Forgejo rejected operation (HTTP {status})"));
    }
    decode_user(&body)
}

pub(crate) fn forgejo_revoke_token(base: &str, token: &str) -> Result<(), String> {
    let (status, _) = forgejo_request(base, "DELETE", "/api/v1/user/token", token)?;
    if !(200..300).contains(&status) {
        return Err(format!("Forgejo rejected operation (HTTP {status})"));
    }
    Ok(())
}

fn forgejo_request(
    base: &str,
    method: &str,
    path: &str,
    token: &str,
) -> Result<(u16, Vec<u8>), String> {
    forgejo_request_with_timeout(base, method, path, token, FORGEJO_TIMEOUT)
}

fn forgejo_request_with_timeout(
    base: &str,
    method: &str,
    path: &str,
    token: &str,
    timeout: Duration,
) -> Result<(u16, Vec<u8>), String> {
    let base = base.trim_end_matches('/');
    let url = format!("{base}{path}");
    let agent = ureq::AgentBuilder::new()
        .timeout(timeout)
        .redirects(0)
        .try_proxy_from_env(false)
        .build();
    let response = match agent
        .request(method, &url)
        .set("Accept", "application/json")
        .set("Content-Type", "application/json")
        .set("Authorization", &format!("token {token}"))
        .call()
    {
        Ok(response) => response,
        Err(ureq::Error::Status(_, response)) => response,
        Err(ureq::Error::Transport(_)) => {
            return Err("forgejo could not be reached".to_string());
        }
    };
    let status = response.status();
    let mut body = Vec::new();
    response
        .into_reader()
        .take((FORGEJO_RESPONSE_LIMIT + 1) as u64)
        .read_to_end(&mut body)
        .map_err(|_| "forgejo could not be reached".to_string())?;
    if body.len() > FORGEJO_RESPONSE_LIMIT {
        return Err("forgejo response exceeds the supported size".to_string());
    }
    Ok((status, body))
}

// decode_user mirrors the forgejo client's decodeResponse into User: the body
// must be one JSON value within the size bound, "null" is invalid, and id /
// is_admin keep Go's strict scalar types (unknown fields allowed).
pub(crate) fn decode_user(body: &[u8]) -> Result<ForgejoUser, String> {
    if body.len() > FORGEJO_RESPONSE_LIMIT {
        return Err("forgejo response exceeds the supported size".to_string());
    }
    let text = std::str::from_utf8(body).map_err(|_| "invalid Forgejo response".to_string())?;
    if text.trim() == "null" {
        return Err("invalid Forgejo response".to_string());
    }
    let mut deserializer = serde_json::Deserializer::from_str(text);
    let user = ForgejoUserResponse::deserialize(&mut deserializer)
        .map_err(|_| "invalid Forgejo response".to_string())?;
    deserializer
        .end()
        .map_err(|_| "invalid Forgejo response".to_string())?;
    Ok(ForgejoUser {
        id: user.id,
        admin: user.admin,
    })
}
