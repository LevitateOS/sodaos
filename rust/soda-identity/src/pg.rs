// Minimal synchronous PostgreSQL client over TCP or unix sockets: startup
// with cleartext/MD5/SCRAM-SHA-256 authentication, the extended query
// protocol with text results, and simple-protocol commands for
// transactions and DDL. Only what the identity store needs, with the same
// observable behavior as the Go pgx database/sql driver for those paths.
use crate::wire::Error;
use bytes::BytesMut;
use fallible_iterator::FallibleIterator;
use postgres_protocol::authentication::{md5_hash, sasl};
use postgres_protocol::message::{backend, frontend};
use std::collections::HashMap;
use std::io::{Read, Write};
use std::net::TcpStream;
use std::os::unix::net::UnixStream;
use std::time::Duration;

const IO_TIMEOUT: Duration = Duration::from_secs(30);

pub struct Dsn {
    user: String,
    password: String,
    host: String,
    port: u16,
    database: String,
}

impl Dsn {
    pub fn parse(dsn: &str) -> Result<Dsn, Error> {
        let dsn = dsn.trim();
        if dsn.is_empty() {
            return Err(Error::internal("postgres connection string is required"));
        }
        let rest = dsn
            .strip_prefix("postgres://")
            .or_else(|| dsn.strip_prefix("postgresql://"))
            .ok_or_else(|| Error::internal("postgres connection string is required"))?;
        let (authority, path_query) = match rest.find('/') {
            Some(i) => (&rest[..i], &rest[i + 1..]),
            None => (rest, ""),
        };
        let (path, query) = match path_query.find('?') {
            Some(i) => (&path_query[..i], &path_query[i + 1..]),
            None => (path_query, ""),
        };
        let (credentials, hostport) = match authority.rfind('@') {
            Some(i) => (&authority[..i], &authority[i + 1..]),
            None => ("", authority),
        };
        let (user, password) = match credentials.find(':') {
            Some(i) => (&credentials[..i], &credentials[i + 1..]),
            None => (credentials, ""),
        };
        let mut params = HashMap::new();
        for pair in query.split('&') {
            if pair.is_empty() {
                continue;
            }
            let (key, value) = match pair.find('=') {
                Some(i) => (&pair[..i], &pair[i + 1..]),
                None => (pair, ""),
            };
            params.insert(percent_decode(key)?, percent_decode(value)?);
        }
        if let Some(mode) = params.get("sslmode") {
            if mode != "disable" {
                return Err(Error::internal(format!(
                    "unsupported postgres sslmode {mode:?}"
                )));
            }
        }
        let (mut host, mut port) = match hostport.rfind(':') {
            Some(i) => (
                &hostport[..i],
                hostport[i + 1..].parse::<u16>().unwrap_or(5432),
            ),
            None => (hostport, 5432),
        };
        if let Some(value) = params.get("host") {
            host = value;
        }
        if let Some(value) = params.get("port") {
            port = value
                .parse::<u16>()
                .map_err(|_| Error::internal("invalid postgres port"))?;
        }
        let mut database = percent_decode(path)?;
        if let Some(value) = params.get("dbname") {
            database = value.clone();
        }
        if database.is_empty() || user.is_empty() {
            return Err(Error::internal("postgres connection string is required"));
        }
        Ok(Dsn {
            user: percent_decode(user)?,
            password: percent_decode(password)?,
            host: host.to_string(),
            port,
            database,
        })
    }
}

fn percent_decode(input: &str) -> Result<String, Error> {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() {
                return Err(Error::internal("invalid postgres connection string"));
            }
            let hex = std::str::from_utf8(&bytes[i + 1..i + 3])
                .map_err(|_| Error::internal("invalid postgres connection string"))?;
            let byte = u8::from_str_radix(hex, 16)
                .map_err(|_| Error::internal("invalid postgres connection string"))?;
            out.push(byte);
            i += 3;
        } else {
            out.push(bytes[i]);
            i += 1;
        }
    }
    String::from_utf8(out).map_err(|_| Error::internal("invalid postgres connection string"))
}

enum Stream {
    Tcp(TcpStream),
    Unix(UnixStream),
}

impl Stream {
    fn read(&mut self, out: &mut [u8]) -> std::io::Result<usize> {
        match self {
            Stream::Tcp(s) => s.read(out),
            Stream::Unix(s) => s.read(out),
        }
    }

    fn write_all(&mut self, data: &[u8]) -> std::io::Result<()> {
        match self {
            Stream::Tcp(s) => s.write_all(data),
            Stream::Unix(s) => s.write_all(data),
        }
    }
}

pub struct Row {
    pub columns: Vec<String>,
    pub fields: Vec<Option<Vec<u8>>>,
}

impl Row {
    pub fn text(&self, index: usize) -> Result<&str, Error> {
        let bytes = self
            .fields
            .get(index)
            .and_then(|f| f.as_ref())
            .ok_or_else(|| Error::internal(format!("postgres row is missing column {index}")))?;
        std::str::from_utf8(bytes).map_err(|_| Error::internal("postgres text is not UTF-8"))
    }

    pub fn integer(&self, index: usize) -> Result<i64, Error> {
        self.text(index)?
            .parse::<i64>()
            .map_err(|_| Error::internal("postgres integer malformed"))
    }

    pub fn bytea(&self, index: usize) -> Result<Vec<u8>, Error> {
        let text = self.text(index)?;
        let hex = text
            .strip_prefix("\\x")
            .ok_or_else(|| Error::internal("postgres bytea is not hex"))?;
        if hex.len() % 2 != 0 {
            return Err(Error::internal("postgres bytea is not hex"));
        }
        let mut out = Vec::with_capacity(hex.len() / 2);
        for i in (0..hex.len()).step_by(2) {
            out.push(
                u8::from_str_radix(&hex[i..i + 2], 16)
                    .map_err(|_| Error::internal("postgres bytea is not hex"))?,
            );
        }
        Ok(out)
    }
}

pub struct Client {
    stream: Stream,
    buffer: BytesMut,
}

impl Client {
    pub fn connect(dsn: &Dsn) -> Result<Client, Error> {
        let mut stream = if dsn.host.starts_with('/') {
            let path = format!("{}/.s.PGSQL.{}", dsn.host, dsn.port);
            let socket = UnixStream::connect(&path)
                .map_err(|e| Error::internal(format!("postgres connect {path}: {e}")))?;
            socket
                .set_read_timeout(Some(IO_TIMEOUT))
                .map_err(Error::from)?;
            socket
                .set_write_timeout(Some(IO_TIMEOUT))
                .map_err(Error::from)?;
            Stream::Unix(socket)
        } else {
            let socket = TcpStream::connect((dsn.host.as_str(), dsn.port))
                .map_err(|e| Error::internal(format!("postgres connect: {e}")))?;
            socket
                .set_read_timeout(Some(IO_TIMEOUT))
                .map_err(Error::from)?;
            socket
                .set_write_timeout(Some(IO_TIMEOUT))
                .map_err(Error::from)?;
            Stream::Tcp(socket)
        };
        let mut out = BytesMut::new();
        frontend::startup_message(
            [
                ("user", dsn.user.as_str()),
                ("database", dsn.database.as_str()),
            ],
            &mut out,
        )
        .map_err(|e| Error::internal(e.to_string()))?;
        stream.write_all(&out).map_err(Error::from)?;
        let mut client = Client {
            stream,
            buffer: BytesMut::new(),
        };
        client.authenticate(dsn)?;
        Ok(client)
    }

    fn send(&mut self, message: &BytesMut) -> Result<(), Error> {
        self.stream.write_all(message).map_err(Error::from)
    }

    fn receive(&mut self) -> Result<backend::Message, Error> {
        loop {
            if let Some(message) = backend::Message::parse(&mut self.buffer)
                .map_err(|e| Error::internal(e.to_string()))?
            {
                return Ok(message);
            }
            let mut chunk = [0u8; 8192];
            let n = self.stream.read(&mut chunk).map_err(Error::from)?;
            if n == 0 {
                return Err(Error::internal("postgres connection closed"));
            }
            self.buffer.extend_from_slice(&chunk[..n]);
        }
    }

    fn authenticate(&mut self, dsn: &Dsn) -> Result<(), Error> {
        loop {
            match self.receive()? {
                backend::Message::AuthenticationOk => {}
                backend::Message::AuthenticationCleartextPassword => {
                    let mut out = BytesMut::new();
                    frontend::password_message(dsn.password.as_bytes(), &mut out)
                        .map_err(|e| Error::internal(e.to_string()))?;
                    self.send(&out)?;
                }
                backend::Message::AuthenticationMd5Password(body) => {
                    let response =
                        md5_hash(dsn.user.as_bytes(), dsn.password.as_bytes(), body.salt());
                    let mut out = BytesMut::new();
                    frontend::password_message(response.as_bytes(), &mut out)
                        .map_err(|e| Error::internal(e.to_string()))?;
                    self.send(&out)?;
                }
                backend::Message::AuthenticationSasl(body) => {
                    let mut offered = body.mechanisms();
                    let mut scram = false;
                    while let Some(mechanism) = offered
                        .next()
                        .map_err(|e: std::io::Error| Error::internal(e.to_string()))?
                    {
                        if mechanism == "SCRAM-SHA-256" {
                            scram = true;
                        }
                    }
                    if !scram {
                        return Err(Error::internal("postgres SCRAM-SHA-256 unavailable"));
                    }
                    self.authenticate_scram(dsn)?;
                }
                backend::Message::ParameterStatus(_) | backend::Message::BackendKeyData(_) => {}
                backend::Message::ReadyForQuery(_) => return Ok(()),
                backend::Message::ErrorResponse(body) => return Err(error_response(body)),
                _ => return Err(Error::internal("unexpected postgres message")),
            }
        }
    }

    fn authenticate_scram(&mut self, dsn: &Dsn) -> Result<(), Error> {
        let mut scram =
            sasl::ScramSha256::new(dsn.password.as_bytes(), sasl::ChannelBinding::unsupported());
        let mut out = BytesMut::new();
        frontend::sasl_initial_response("SCRAM-SHA-256", scram.message(), &mut out)
            .map_err(|e| Error::internal(e.to_string()))?;
        self.send(&out)?;
        loop {
            match self.receive()? {
                backend::Message::AuthenticationSaslContinue(body) => {
                    scram.update(body.data()).map_err(|e| {
                        Error::internal(format!("postgres SCRAM exchange failed: {e}"))
                    })?;
                    let mut out = BytesMut::new();
                    frontend::sasl_response(scram.message(), &mut out)
                        .map_err(|e| Error::internal(e.to_string()))?;
                    self.send(&out)?;
                }
                backend::Message::AuthenticationSaslFinal(body) => {
                    scram.finish(body.data()).map_err(|e| {
                        Error::internal(format!("postgres SCRAM verification failed: {e}"))
                    })?;
                    return Ok(());
                }
                backend::Message::ErrorResponse(body) => return Err(error_response(body)),
                _ => return Err(Error::internal("unexpected postgres message")),
            }
        }
    }

    /// Run an extended-protocol query with text parameters, returning rows
    /// and the command completion count.
    pub fn query(&mut self, sql: &str, params: &[Option<&[u8]>]) -> Result<(Vec<Row>, u64), Error> {
        let mut out = BytesMut::new();
        frontend::parse("", sql, Vec::<u32>::new(), &mut out)
            .map_err(|e| Error::internal(e.to_string()))?;
        let formats = vec![0i16; params.len()];
        frontend::bind(
            "",
            "",
            formats,
            params.iter().copied(),
            |value: Option<&[u8]>, buf: &mut BytesMut| match value {
                Some(data) => {
                    buf.extend_from_slice(data);
                    Ok(postgres_protocol::IsNull::No)
                }
                None => Ok(postgres_protocol::IsNull::Yes),
            },
            [0i16],
            &mut out,
        )
        .map_err(|_| Error::internal("postgres bind failed"))?;
        frontend::describe(b'P', "", &mut out).map_err(|e| Error::internal(e.to_string()))?;
        frontend::execute("", 0, &mut out).map_err(|e| Error::internal(e.to_string()))?;
        frontend::sync(&mut out);
        self.send(&out)?;
        let mut columns = Vec::new();
        let mut rows = Vec::new();
        let mut count = 0u64;
        let mut failure: Option<Error> = None;
        loop {
            match self.receive()? {
                backend::Message::ParseComplete
                | backend::Message::BindComplete
                | backend::Message::ParameterDescription(_)
                | backend::Message::NoData
                | backend::Message::NoticeResponse(_)
                | backend::Message::NotificationResponse(_)
                | backend::Message::PortalSuspended
                | backend::Message::EmptyQueryResponse => {}
                backend::Message::RowDescription(body) => {
                    columns = body
                        .fields()
                        .map(|f| Ok(f.name().to_string()))
                        .collect()
                        .map_err(|e: std::io::Error| Error::internal(e.to_string()))?;
                }
                backend::Message::DataRow(body) => {
                    let mut fields = Vec::new();
                    let mut ranges = body.ranges();
                    while let Some(range) = ranges
                        .next()
                        .map_err(|e: std::io::Error| Error::internal(e.to_string()))?
                    {
                        fields.push(range.map(|r| body.buffer()[r].to_vec()));
                    }
                    rows.push(Row {
                        columns: columns.clone(),
                        fields,
                    });
                }
                backend::Message::CommandComplete(body) => {
                    count = command_count(body.tag().unwrap_or(""));
                }
                backend::Message::ErrorResponse(body) => {
                    failure = Some(error_response(body));
                }
                backend::Message::ReadyForQuery(_) => break,
                backend::Message::CopyInResponse(_)
                | backend::Message::CopyOutResponse(_)
                | backend::Message::CopyData(_)
                | backend::Message::CopyDone => {
                    return Err(Error::internal("unexpected postgres COPY message"));
                }
                _ => return Err(Error::internal("unexpected postgres message")),
            }
        }
        if let Some(err) = failure {
            return Err(err);
        }
        Ok((rows, count))
    }

    /// Run a simple-protocol command (transactions, DDL): no parameters,
    /// no rows retained.
    pub fn simple(&mut self, sql: &str) -> Result<u64, Error> {
        let mut out = BytesMut::new();
        frontend::query(sql, &mut out).map_err(|e| Error::internal(e.to_string()))?;
        self.send(&out)?;
        let mut count = 0u64;
        let mut failure: Option<Error> = None;
        loop {
            match self.receive()? {
                backend::Message::RowDescription(_)
                | backend::Message::DataRow(_)
                | backend::Message::ParameterStatus(_)
                | backend::Message::NoticeResponse(_)
                | backend::Message::NotificationResponse(_)
                | backend::Message::EmptyQueryResponse => {}
                backend::Message::CommandComplete(body) => {
                    count = command_count(body.tag().unwrap_or(""));
                }
                backend::Message::ErrorResponse(body) => {
                    failure = Some(error_response(body));
                }
                backend::Message::ReadyForQuery(_) => break,
                backend::Message::CopyInResponse(_)
                | backend::Message::CopyOutResponse(_)
                | backend::Message::CopyData(_)
                | backend::Message::CopyDone => {
                    return Err(Error::internal("unexpected postgres COPY message"));
                }
                _ => return Err(Error::internal("unexpected postgres message")),
            }
        }
        if let Some(err) = failure {
            return Err(err);
        }
        Ok(count)
    }
}

fn command_count(tag: &str) -> u64 {
    tag.split_whitespace()
        .next_back()
        .and_then(|n| n.parse::<u64>().ok())
        .unwrap_or(0)
}

fn error_response(body: backend::ErrorResponseBody) -> Error {
    let mut message = String::from("postgres operation failed");
    let mut code = String::new();
    let mut fields = body.fields();
    while let Ok(Some(field)) = fields.next() {
        match field.type_() {
            b'M' => message = String::from_utf8_lossy(field.value_bytes()).into_owned(),
            b'C' => code = String::from_utf8_lossy(field.value_bytes()).into_owned(),
            _ => {}
        }
    }
    // SQLSTATE is retained for operators; the broker maps failures without
    // branching on codes, exactly like the Go store.
    if code.is_empty() {
        Error::internal(message)
    } else {
        Error::internal(format!("{message} ({code})"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dsn_shapes_match_go() {
        // Appliance unix-socket DSN from the setup generator.
        let dsn =
            Dsn::parse("postgres://soda:secret@/soda?host=/run/soda/postgres&sslmode=disable")
                .unwrap();
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
}
