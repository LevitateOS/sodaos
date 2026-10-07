use crate::wire::Error;
use std::str::FromStr;
use tokio_postgres::{config::SslMode, Config};

pub struct Dsn(pub(crate) Config);

impl Dsn {
    pub fn parse(dsn: &str) -> Result<Dsn, Error> {
        let dsn = dsn.trim();
        if !(dsn.starts_with("postgres://") || dsn.starts_with("postgresql://")) {
            return Err(Error::internal("postgres connection string is required"));
        }
        let mut config = Config::from_str(dsn)
            .map_err(|_| Error::internal("invalid postgres connection string"))?;

        // The appliance uses NoTls. Preserve the existing URI policy: absent
        // sslmode means this NoTls connection, while any explicit mode must
        // say disable.
        let has_sslmode = dsn.split_once('?').is_some_and(|(_, query)| {
            query
                .split('&')
                .any(|part| part.split('=').next() == Some("sslmode"))
        });
        if config.get_ssl_mode() == SslMode::Require
            || (has_sslmode && config.get_ssl_mode() != SslMode::Disable)
        {
            return Err(Error::internal("postgres TLS is not supported"));
        }
        config.ssl_mode(SslMode::Disable);
        if config.get_user().map_or(true, str::is_empty)
            || config.get_dbname().map_or(true, str::is_empty)
        {
            return Err(Error::internal("postgres connection string is required"));
        }
        Ok(Dsn(config))
    }
}
