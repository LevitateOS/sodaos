use super::json::JsonParser;
use super::release::{ReleaseImage, ReleasePayload};

pub(crate) fn decode_release_payload(body: &[u8]) -> Result<ReleasePayload, ()> {
    let mut p = JsonParser::new(body);
    p.skip_ws();
    if p.peek() != Some(b'{') {
        return Err(());
    }
    p.bump();
    let mut format = -1i64;
    let mut id = String::new();
    let mut revision = String::new();
    let mut architecture = String::new();
    let mut coreos = String::new();
    let mut base = String::new();
    let mut repository_prefix = String::new();
    let mut schema = -1i64;
    let mut presentation = String::new();
    let mut host_packages = String::new();
    let mut images: Vec<(String, ReleaseImage)> = Vec::new();
    let mut upgrade_from: Vec<String> = Vec::new();
    let mut upgrade_seen = false;
    // encoding/json applies duplicates in order; the last value wins.
    // Trailing commas are rejected.
    let mut first = true;
    loop {
        p.skip_ws();
        if p.eof() {
            return Err(());
        }
        if first && p.peek() == Some(b'}') {
            p.bump();
            break;
        }
        if p.peek() != Some(b'"') {
            return Err(());
        }
        let key = p.parse_string().map_err(|_| ())?;
        p.skip_ws();
        if p.peek() != Some(b':') {
            return Err(());
        }
        p.bump();
        p.skip_ws();
        match payload_slot(&key) {
            Some(0) => format = p.parse_payload_int().map_err(|_| ())?,
            Some(1) => id = p.parse_payload_string().map_err(|_| ())?,
            Some(2) => revision = p.parse_payload_string().map_err(|_| ())?,
            Some(3) => architecture = p.parse_payload_string().map_err(|_| ())?,
            Some(4) => coreos = p.parse_payload_string().map_err(|_| ())?,
            Some(5) => base = p.parse_payload_string().map_err(|_| ())?,
            Some(6) => repository_prefix = p.parse_payload_string().map_err(|_| ())?,
            Some(7) => schema = p.parse_payload_int().map_err(|_| ())?,
            Some(8) => presentation = p.parse_payload_string().map_err(|_| ())?,
            Some(9) => host_packages = p.parse_payload_string().map_err(|_| ())?,
            Some(10) => images = p.parse_payload_images().map_err(|_| ())?,
            Some(11) => {
                upgrade_from = p.parse_payload_string_list().map_err(|_| ())?;
                upgrade_seen = true;
            }
            _ => return Err(()),
        }
        p.skip_ws();
        if p.eof() {
            return Err(());
        }
        match p.peek() {
            Some(b',') => {
                p.bump();
            }
            Some(b'}') => {
                p.bump();
                break;
            }
            _ => return Err(()),
        }
        first = false;
    }
    p.skip_ws();
    if !p.eof() {
        return Err(());
    }
    if !upgrade_seen {
        upgrade_from = Vec::new();
    }
    super::validate_release_payload(
        format,
        &id,
        &revision,
        &architecture,
        &coreos,
        &base,
        &repository_prefix,
        schema,
        &presentation,
        &host_packages,
        &images,
        &upgrade_from,
    )?;
    Ok(ReleasePayload {
        architecture,
        images,
    })
}

// payload_slot matches Payload's untagged Go field names, exact first with
// the same case-insensitive fallback encoding/json applies.
fn payload_slot(key: &str) -> Option<usize> {
    const KEYS: [&str; 12] = [
        "Format",
        "ID",
        "Revision",
        "Architecture",
        "CoreOS",
        "Base",
        "RepositoryPrefix",
        "Schema",
        "PresentationSHA256",
        "HostPackagesSHA256",
        "Images",
        "UpgradeFrom",
    ];
    if let Some(i) = KEYS.iter().position(|k| *k == key) {
        return Some(i);
    }
    let mut found = None;
    for (i, k) in KEYS.iter().enumerate() {
        if k.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(i);
        }
    }
    found
}

impl JsonParser<'_> {
    fn parse_payload_string(&mut self) -> Result<String, String> {
        if self.peek() == Some(b'n') && self.bytes[self.pos..].starts_with(b"null") {
            self.pos += 4;
            return Ok(String::new());
        }
        self.parse_string()
    }

    pub(crate) fn parse_payload_int(&mut self) -> Result<i64, String> {
        if self.peek() == Some(b'n') && self.bytes[self.pos..].starts_with(b"null") {
            self.pos += 4;
            return Ok(0);
        }
        let start = self.pos;
        if self.peek() == Some(b'-') {
            self.bump();
        }
        match self.peek() {
            Some(b'0') => {
                self.bump();
            }
            Some(b'1'..=b'9') => {
                while matches!(self.peek(), Some(b'0'..=b'9')) {
                    self.bump();
                }
            }
            _ => return Err(String::from("invalid number")),
        }
        if matches!(self.peek(), Some(b'.' | b'e' | b'E')) {
            return Err(String::from("invalid number"));
        }
        std::str::from_utf8(&self.bytes[start..self.pos])
            .map_err(|_| String::from("invalid number"))?
            .parse::<i64>()
            .map_err(|_| String::from("invalid number"))
    }

    fn parse_payload_images(&mut self) -> Result<Vec<(String, ReleaseImage)>, String> {
        if self.peek() != Some(b'{') {
            return Err(String::from("invalid images"));
        }
        self.bump();
        // Map entries apply in order; a repeated name replaces the
        // earlier binding exactly like encoding/json into a map.
        let mut images = Vec::new();
        let mut first = true;
        loop {
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid images"));
            }
            if first && self.peek() == Some(b'}') {
                self.bump();
                return Ok(images);
            }
            if self.peek() != Some(b'"') {
                return Err(String::from("invalid images"));
            }
            let name = self.parse_string()?;
            self.skip_ws();
            if self.peek() != Some(b':') {
                return Err(String::from("invalid images"));
            }
            self.bump();
            self.skip_ws();
            let image = self.parse_payload_image()?;
            match images.iter_mut().find(|(n, _)| *n == name) {
                Some(slot) => slot.1 = image,
                None => images.push((name, image)),
            }
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid images"));
            }
            match self.peek() {
                Some(b',') => {
                    self.bump();
                }
                Some(b'}') => {
                    self.bump();
                    return Ok(images);
                }
                _ => return Err(String::from("invalid images")),
            }
            first = false;
        }
    }

    fn parse_payload_image(&mut self) -> Result<ReleaseImage, String> {
        if self.peek() != Some(b'{') {
            return Err(String::from("invalid image"));
        }
        self.bump();
        let mut reference = String::new();
        let mut config = String::new();
        let mut manifest = String::new();
        let mut archive = String::new();
        // Repeated fields overwrite in order like encoding/json; trailing
        // commas are rejected.
        let mut first = true;
        loop {
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid image"));
            }
            if first && self.peek() == Some(b'}') {
                self.bump();
                return Ok(ReleaseImage {
                    reference,
                    config,
                    manifest,
                    archive_sha256: archive,
                });
            }
            if self.peek() != Some(b'"') {
                return Err(String::from("invalid image"));
            }
            let key = self.parse_string()?;
            let slot = match image_slot(&key) {
                Some(s) => s,
                None => return Err(String::from("invalid image")),
            };
            self.skip_ws();
            if self.peek() != Some(b':') {
                return Err(String::from("invalid image"));
            }
            self.bump();
            self.skip_ws();
            let value = self.parse_payload_string()?;
            match slot {
                "Reference" => reference = value,
                "Config" => config = value,
                "Manifest" => manifest = value,
                _ => archive = value,
            }
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid image"));
            }
            match self.peek() {
                Some(b',') => {
                    self.bump();
                }
                Some(b'}') => {
                    self.bump();
                    return Ok(ReleaseImage {
                        reference,
                        config,
                        manifest,
                        archive_sha256: archive,
                    });
                }
                _ => return Err(String::from("invalid image")),
            }
            first = false;
        }
    }

    fn parse_payload_string_list(&mut self) -> Result<Vec<String>, String> {
        if self.peek() == Some(b'n') && self.bytes[self.pos..].starts_with(b"null") {
            self.pos += 4;
            return Ok(Vec::new());
        }
        if self.peek() != Some(b'[') {
            return Err(String::from("invalid list"));
        }
        self.bump();
        let mut out = Vec::new();
        let mut first = true;
        loop {
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid list"));
            }
            if first && self.peek() == Some(b']') {
                self.bump();
                return Ok(out);
            }
            if !first {
                if self.peek() != Some(b',') {
                    return Err(String::from("invalid list"));
                }
                self.bump();
                self.skip_ws();
            }
            out.push(self.parse_string()?);
            first = false;
        }
    }
}

fn image_slot(key: &str) -> Option<&'static str> {
    const KEYS: [&str; 4] = ["Reference", "Config", "Manifest", "ArchiveSHA256"];
    if let Some(k) = KEYS.iter().find(|k| **k == key).copied() {
        return Some(k);
    }
    let mut found = None;
    for k in KEYS {
        if k.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(k);
        }
    }
    found
}
