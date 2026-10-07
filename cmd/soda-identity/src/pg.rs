pub use crate::pg_dsn::Dsn;
use crate::wire::Error;
pub use tokio_postgres::Row;
use tokio_postgres::{Client, NoTls};

pub(crate) struct Connection {
    pub(crate) client: Client,
    pub(crate) driver: tokio::task::JoinHandle<Result<(), tokio_postgres::Error>>,
}

impl Connection {
    pub(crate) async fn connect(dsn: &Dsn) -> Result<Connection, Error> {
        let (client, driver) = dsn.0.connect(NoTls).await.map_err(pg_error)?;
        let driver = tokio::spawn(driver);
        Ok(Connection { client, driver })
    }
}

pub(crate) fn pg_error(error: tokio_postgres::Error) -> Error {
    if let Some(code) = error.code() {
        Error::internal(format!("postgres operation failed ({})", code.code()))
    } else {
        Error::internal("postgres operation failed")
    }
}
