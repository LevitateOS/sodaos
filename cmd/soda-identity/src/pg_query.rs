// PostgreSQL row decoding and query helpers, extracted from pg.rs (A-R02a).
use crate::pg::{error_response, Client};
use crate::wire::Error;
use bytes::BytesMut;
use fallible_iterator::FallibleIterator;
use postgres_protocol::message::{backend, frontend};

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

impl Client {
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

pub(crate) fn command_count(tag: &str) -> u64 {
    tag.split_whitespace()
        .next_back()
        .and_then(|n| n.parse::<u64>().ok())
        .unwrap_or(0)
}
