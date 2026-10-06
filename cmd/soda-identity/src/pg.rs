// Minimal synchronous PostgreSQL client over TCP or unix sockets: startup
// with cleartext/MD5/SCRAM-SHA-256 authentication, the extended query
// protocol with text results, and simple-protocol commands for
// transactions and DDL. Only what the identity store needs, with the same
// observable behavior as the Go pgx database/sql driver for those paths.
pub use crate::pg_dsn::Dsn;
pub use crate::pg_query::Row;
use crate::wire::Error;
use bytes::BytesMut;
use fallible_iterator::FallibleIterator;
use postgres_protocol::authentication::{md5_hash, sasl};
use postgres_protocol::message::{backend, frontend};
use std::io::{Read, Write};
use std::net::TcpStream;
use std::os::unix::net::UnixStream;
use std::time::Duration;

const IO_TIMEOUT: Duration = Duration::from_secs(30);

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

    pub(crate) fn send(&mut self, message: &BytesMut) -> Result<(), Error> {
        self.stream.write_all(message).map_err(Error::from)
    }

    pub(crate) fn receive(&mut self) -> Result<backend::Message, Error> {
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
}

pub(crate) fn error_response(body: backend::ErrorResponseBody) -> Error {
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
#[path = "pg_tests.rs"]
mod pg_tests;
