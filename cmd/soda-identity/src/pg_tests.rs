use crate::pg_dsn::Dsn;
use crate::pg_query::command_count;

#[test]
fn dsn_shapes_match_go() {
    // Appliance unix-socket DSN from the setup generator.
    let dsn =
        Dsn::parse("postgres://soda:secret@/soda?host=/run/soda/postgres&sslmode=disable").unwrap();
    assert_eq!(dsn.user, "soda");
    assert_eq!(dsn.host, "/run/soda/postgres");
    assert_eq!(dsn.port, 5432);
    assert_eq!(dsn.database, "soda");
    // Ephemeral fixture DSN.
    let dsn =
        Dsn::parse("postgres://postgres:pw@127.0.0.1:41277/postgres?sslmode=disable").unwrap();
    assert_eq!(dsn.host, "127.0.0.1");
    assert_eq!(dsn.port, 41277);
    // Percent-encoded credentials decode.
    let dsn = Dsn::parse("postgres://user:p%40ss@h/db?sslmode=disable").unwrap();
    assert_eq!(dsn.password, "p@ss");
    assert!(Dsn::parse("").is_err());
    assert!(Dsn::parse("postgres://u@/db?sslmode=require").is_err());
    assert!(Dsn::parse("http://u@/db").is_err());
}

#[test]
fn command_tags_count_rows() {
    assert_eq!(command_count("SELECT 1"), 1);
    assert_eq!(command_count("INSERT 0 1"), 1);
    assert_eq!(command_count("UPDATE 3"), 3);
    assert_eq!(command_count("DELETE 0"), 0);
    assert_eq!(command_count("CREATE TABLE"), 0);
    assert_eq!(command_count("BEGIN"), 0);
}
