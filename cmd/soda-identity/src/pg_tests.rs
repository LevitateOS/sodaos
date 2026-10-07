use crate::pg::pg_error;
use crate::pg_dsn::Dsn;
use crate::store::{Store, schema_integer};
use std::sync::{Arc, mpsc};
use std::thread;
use std::time::Duration;

static TEST_LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());

#[test]
fn dsn_policy_keeps_uri_no_tls_and_socket_support() {
    let dsn = Dsn::parse("postgres://soda@/soda?host=/run/soda/postgres&sslmode=disable").unwrap();
    assert_eq!(dsn.0.get_user(), Some("soda"));
    assert_eq!(dsn.0.get_dbname(), Some("soda"));
    assert_eq!(
        dsn.0.get_ssl_mode(),
        tokio_postgres::config::SslMode::Disable
    );

    let dsn = Dsn::parse("postgres://user@127.0.0.1:41277/db").unwrap();
    assert_eq!(dsn.0.get_ports(), &[41277]);
    assert!(Dsn::parse("").is_err());
    assert!(Dsn::parse("postgres://u@/db?sslmode=require").is_err());
    assert!(Dsn::parse("http://u@/db").is_err());
}

fn integration_store() -> Store {
    let dsn = std::env::var("SODA_IDENTITY_TEST_DSN")
        .expect("set SODA_IDENTITY_TEST_DSN to a task-owned disposable database");
    integration_store_for_dsn(&dsn)
}

fn integration_store_for_dsn(dsn: &str) -> Store {
    let mut store = Store::open(&dsn, None).expect("open disposable identity database");
    store.set_operation_budget_for_test(Duration::from_secs(2));
    store
}

#[test]
#[ignore = "requires task-owned disposable PostgreSQL DSNs for local Unix and loopback TCP"]
fn store_authenticates_over_unix_and_loopback_tcp() {
    let _test_guard = TEST_LOCK.lock().unwrap();
    for key in ["SODA_IDENTITY_TEST_DSN_UNIX", "SODA_IDENTITY_TEST_DSN_TCP"] {
        let dsn =
            std::env::var(key).unwrap_or_else(|_| panic!("set {key} to a disposable database"));
        let store = integration_store_for_dsn(&dsn);
        assert_eq!(
            store
                .query_row("SELECT 1", &[])
                .unwrap()
                .try_get::<_, i32>(0)
                .unwrap(),
            1
        );
    }
}

#[test]
#[ignore = "requires a task-owned disposable PostgreSQL database in SODA_IDENTITY_TEST_DSN"]
fn store_cancellation_discards_session_and_next_operation_reconnects() {
    let _test_guard = TEST_LOCK.lock().unwrap();
    let store = integration_store();
    let before = store
        .query_row("SELECT pg_backend_pid()", &[])
        .unwrap()
        .try_get::<_, i32>(0)
        .unwrap();

    let error = store.query("SELECT pg_sleep(60)", &[]).unwrap_err();
    assert!(error.to_string().contains("deadline"));

    let after = store
        .query_row("SELECT pg_backend_pid()", &[])
        .unwrap()
        .try_get::<_, i32>(0)
        .unwrap();
    assert_ne!(before, after, "the canceled backend session was reused");
}

#[test]
#[ignore = "requires a task-owned disposable PostgreSQL database in SODA_IDENTITY_TEST_DSN"]
fn swallowed_transaction_errors_cannot_commit_or_return_success() {
    let _test_guard = TEST_LOCK.lock().unwrap();
    let store = integration_store();

    let sql_error = store.transaction(|tx| {
        assert!(tx.exec("UPDATE l08_missing_table SET n=1", &[]).is_err());
        Ok(())
    });
    assert!(
        sql_error.is_err(),
        "transaction committed after a swallowed SQL error"
    );

    let result = store.transaction(|tx| {
        let error = tx.query("SELECT pg_sleep(60)", &[]).unwrap_err();
        assert!(error.to_string().contains("deadline"));
        Ok(())
    });
    assert!(
        result.is_err(),
        "transaction succeeded after its session was discarded"
    );

    // A later independent operation may reconnect after the failed Tx ends.
    assert_eq!(
        store
            .query_row("SELECT 1", &[])
            .unwrap()
            .try_get::<_, i32>(0)
            .unwrap(),
        1
    );
}

#[test]
#[ignore = "requires a task-owned disposable PostgreSQL database in SODA_IDENTITY_TEST_DSN"]
fn typed_postgres_rows_and_transaction_guard_match_store_contract() {
    let _test_guard = TEST_LOCK.lock().unwrap();
    let store = Arc::new(integration_store());
    let bound_bytea: [u8; 2] = [0xca, 0xfe];
    let (rows, _) = store
        .query(
            "SELECT NULL::TEXT,'value'::TEXT,NULL::BIGINT,7::BIGINT,NULL::BYTEA,decode('dead','hex'),NULL::BOOL,TRUE",
            &[],
        )
        .unwrap();
    let row = &rows[0];
    assert_eq!(
        row.try_get::<_, Option<String>>(0)
            .map_err(pg_error)
            .unwrap(),
        None
    );
    assert_eq!(
        row.try_get::<_, String>(1).map_err(pg_error).unwrap(),
        "value"
    );
    assert_eq!(
        row.try_get::<_, Option<i64>>(2).map_err(pg_error).unwrap(),
        None
    );
    assert_eq!(row.try_get::<_, i64>(3).map_err(pg_error).unwrap(), 7);
    assert_eq!(
        row.try_get::<_, Option<Vec<u8>>>(4)
            .map_err(pg_error)
            .unwrap(),
        None
    );
    assert_eq!(
        row.try_get::<_, Vec<u8>>(5).map_err(pg_error).unwrap(),
        &[0xde, 0xad]
    );
    assert_eq!(
        row.try_get::<_, Option<bool>>(6).map_err(pg_error).unwrap(),
        None
    );
    assert!(row.try_get::<_, bool>(7).map_err(pg_error).unwrap());

    let (rows, _) = store
        .query(
            "SELECT $1::TEXT,$2::INTEGER,$3::BYTEA,$4::BOOL,$5::JSONB,$6::BIGINT",
            &[
                &"bound text" as &(dyn tokio_postgres::types::ToSql + Sync),
                &11_i32 as &(dyn tokio_postgres::types::ToSql + Sync),
                &&bound_bytea[..] as &(dyn tokio_postgres::types::ToSql + Sync),
                &true as &(dyn tokio_postgres::types::ToSql + Sync),
                &tokio_postgres::types::Json(serde_json::json!({"value": 1}))
                    as &(dyn tokio_postgres::types::ToSql + Sync),
                &12_i64 as &(dyn tokio_postgres::types::ToSql + Sync),
            ],
        )
        .unwrap();
    assert_eq!(rows[0].try_get::<_, String>(0).unwrap(), "bound text");
    assert_eq!(rows[0].try_get::<_, i32>(1).unwrap(), 11);
    assert_eq!(rows[0].try_get::<_, Vec<u8>>(2).unwrap(), &[0xca, 0xfe]);
    assert!(rows[0].try_get::<_, bool>(3).unwrap());
    assert_eq!(
        rows[0]
            .try_get::<_, tokio_postgres::types::Json<serde_json::Value>>(4)
            .unwrap()
            .0,
        serde_json::json!({"value": 1})
    );
    assert_eq!(rows[0].try_get::<_, i64>(5).unwrap(), 12);

    store
        .simple("CREATE TEMP TABLE l08_transaction_order (n INTEGER NOT NULL)")
        .unwrap();
    let (entered_tx, entered_rx) = mpsc::channel();
    let first_store = store.clone();
    let first = thread::spawn(move || {
        first_store.transaction(|tx| {
            tx.exec("INSERT INTO l08_transaction_order VALUES (1)", &[])?;
            entered_tx.send(()).unwrap();
            thread::sleep(Duration::from_millis(500));
            Ok(())
        })
    });
    entered_rx.recv_timeout(Duration::from_secs(2)).unwrap();

    let (competitor_done_tx, competitor_done_rx) = mpsc::channel();
    let competitor_store = store.clone();
    let competitor = thread::spawn(move || {
        let result = competitor_store.exec("INSERT INTO l08_transaction_order VALUES (2)", &[]);
        competitor_done_tx.send(result).unwrap();
    });
    assert!(
        competitor_done_rx
            .recv_timeout(Duration::from_millis(100))
            .is_err(),
        "competitor entered before the transaction committed"
    );
    first.join().unwrap().unwrap();
    competitor_done_rx
        .recv_timeout(Duration::from_secs(2))
        .unwrap()
        .unwrap();
    competitor.join().unwrap();

    let (rows, _) = store
        .query("SELECT n FROM l08_transaction_order ORDER BY n", &[])
        .unwrap();
    assert_eq!(
        rows.iter()
            .map(|row| row.try_get::<_, i32>(0).unwrap())
            .collect::<Vec<_>>(),
        [1, 2]
    );
}

#[test]
fn schema_integer_rejects_values_outside_postgres_integer_range() {
    assert_eq!(schema_integer(i64::from(i32::MIN)).unwrap(), i32::MIN);
    assert_eq!(schema_integer(i64::from(i32::MAX)).unwrap(), i32::MAX);
    assert!(schema_integer(i64::from(i32::MIN) - 1).is_err());
    assert!(schema_integer(i64::from(i32::MAX) + 1).is_err());
}
