use crate::pg::pg_error;
use crate::wire::Error;
use std::time::Duration;
use std::{future::Future, pin::pin};
use tokio::time::{timeout_at, Instant};
use tokio_postgres::{Client, NoTls};

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
