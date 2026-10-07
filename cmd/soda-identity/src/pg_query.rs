use crate::pg::pg_error;
use crate::wire::Error;
use std::time::Duration;
use std::{future::Future, pin::pin};
use tokio::time::{timeout_at, Instant};
use tokio_postgres::{types::Type, Client, NoTls, Row as PgRow};

const DRIVER_JOIN_BUDGET: Duration = Duration::from_millis(50);

pub(crate) enum Outcome<T> {
    Complete(Result<T, Error>),
    TimedOut,
}

pub(crate) async fn bounded<T, F>(
    client: &Client,
    future: F,
    work_deadline: Instant,
    operation_deadline: Instant,
) -> Outcome<T>
where
    F: Future<Output = Result<T, tokio_postgres::Error>>,
{
    let cancel = client.cancel_token();
    let mut future = pin!(future);
    match timeout_at(work_deadline, &mut future).await {
        Ok(result) => Outcome::Complete(result.map_err(pg_error)),
        Err(_) => {
            // Keep polling the original request after sending the separate
            // cancellation packet. Its cancellation response drains protocol
            // state, but the owning Store still discards this session.
            let drain_deadline = operation_deadline - DRIVER_JOIN_BUDGET;
            let canceled = timeout_at(drain_deadline, cancel.cancel_query(NoTls)).await;
            if canceled.is_ok() {
                let _ = timeout_at(drain_deadline, &mut future).await;
            }
            Outcome::TimedOut
        }
    }
}

#[derive(Debug)]
enum Field {
    Text(Option<String>),
    Integer(Option<i64>),
    Bytes(Option<Vec<u8>>),
    Boolean(Option<bool>),
}

#[derive(Debug)]
pub struct Row {
    fields: Vec<Field>,
}

impl Row {
    pub(crate) fn from_pg(row: PgRow) -> Result<Row, Error> {
        let mut fields = Vec::with_capacity(row.len());
        for (index, column) in row.columns().iter().enumerate() {
            let ty = column.type_();
            let field = if ty == &Type::BOOL {
                Field::Boolean(row.try_get(index).map_err(pg_error)?)
            } else if ty == &Type::INT2 {
                Field::Integer(
                    row.try_get::<_, Option<i16>>(index)
                        .map_err(pg_error)?
                        .map(i64::from),
                )
            } else if ty == &Type::INT4 {
                Field::Integer(
                    row.try_get::<_, Option<i32>>(index)
                        .map_err(pg_error)?
                        .map(i64::from),
                )
            } else if ty == &Type::INT8 {
                Field::Integer(row.try_get(index).map_err(pg_error)?)
            } else if ty == &Type::BYTEA {
                Field::Bytes(row.try_get(index).map_err(pg_error)?)
            } else if ty == &Type::JSON || ty == &Type::JSONB {
                Field::Text(
                    row.try_get::<_, Option<serde_json::Value>>(index)
                        .map_err(pg_error)?
                        .map(|value| value.to_string()),
                )
            } else {
                Field::Text(row.try_get(index).map_err(pg_error)?)
            };
            fields.push(field);
        }
        Ok(Row { fields })
    }

    fn field(&self, index: usize) -> Result<&Field, Error> {
        self.fields
            .get(index)
            .ok_or_else(|| Error::internal(format!("postgres row is missing column {index}")))
    }

    pub fn nullable_text(&self, index: usize) -> Result<Option<&str>, Error> {
        match self.field(index)? {
            Field::Text(value) => Ok(value.as_deref()),
            _ => Err(Error::internal("postgres column is not text")),
        }
    }

    pub fn text(&self, index: usize) -> Result<&str, Error> {
        self.nullable_text(index)?
            .ok_or_else(|| Error::internal(format!("postgres row has NULL in column {index}")))
    }

    pub fn nullable_integer(&self, index: usize) -> Result<Option<i64>, Error> {
        match self.field(index)? {
            Field::Integer(value) => Ok(*value),
            _ => Err(Error::internal("postgres column is not an integer")),
        }
    }

    pub fn integer(&self, index: usize) -> Result<i64, Error> {
        self.nullable_integer(index)?
            .ok_or_else(|| Error::internal(format!("postgres row has NULL in column {index}")))
    }

    pub fn nullable_bytea(&self, index: usize) -> Result<Option<&[u8]>, Error> {
        match self.field(index)? {
            Field::Bytes(value) => Ok(value.as_deref()),
            _ => Err(Error::internal("postgres column is not bytea")),
        }
    }

    pub fn bytea(&self, index: usize) -> Result<&[u8], Error> {
        self.nullable_bytea(index)?
            .ok_or_else(|| Error::internal(format!("postgres row has NULL in column {index}")))
    }

    pub fn nullable_boolean(&self, index: usize) -> Result<Option<bool>, Error> {
        match self.field(index)? {
            Field::Boolean(value) => Ok(*value),
            _ => Err(Error::internal("postgres column is not boolean")),
        }
    }

    pub fn boolean(&self, index: usize) -> Result<bool, Error> {
        self.nullable_boolean(index)?
            .ok_or_else(|| Error::internal(format!("postgres row has NULL in column {index}")))
    }
}
