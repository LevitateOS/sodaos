//! Polygon tokenization and even-odd raster geometry.

#[derive(Debug, Clone, PartialEq)]
pub(crate) enum Token {
    Cmd(char),
    Num(String),
}

/// Tokenize path data like `re.findall(r'[A-Za-z]|-?\d+(?:\.\d+)?', data)`:
/// letters are commands, `-?\d+(\.\d+)?` numbers, everything else skipped.
pub(crate) fn tokenize(data: &str) -> Vec<Token> {
    let bytes = data.as_bytes();
    let mut tokens = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        let b = bytes[i];
        if b.is_ascii_alphabetic() {
            tokens.push(Token::Cmd(b as char));
            i += 1;
        } else if b.is_ascii_digit()
            || (b == b'-' && i + 1 < bytes.len() && bytes[i + 1].is_ascii_digit())
        {
            let start = i;
            if bytes[i] == b'-' {
                i += 1;
            }
            while i < bytes.len() && bytes[i].is_ascii_digit() {
                i += 1;
            }
            if i + 1 < bytes.len() && bytes[i] == b'.' && bytes[i + 1].is_ascii_digit() {
                i += 1;
                while i < bytes.len() && bytes[i].is_ascii_digit() {
                    i += 1;
                }
            }
            tokens.push(Token::Num(data[start..i].to_string()));
        } else {
            i += 1;
        }
    }
    tokens
}

fn operand(token: &Token) -> Result<f64, String> {
    match token {
        Token::Num(raw) => raw
            .parse::<f64>()
            .map_err(|_| format!("could not convert string to float: '{raw}'")),
        Token::Cmd(c) => Err(format!("could not convert string to float: '{c}'")),
    }
}

/// Parse absolute-polygon path data into rings, with the script's exact
/// gates (points accumulate across moves until `Z`, like the owner).
pub(crate) fn polygons(data: &str) -> Result<Vec<Vec<(f64, f64)>>, String> {
    let tokens = tokenize(data);
    let mut result: Vec<Vec<(f64, f64)>> = Vec::new();
    let mut points: Vec<(f64, f64)> = Vec::new();
    let (mut x, mut y) = (0.0, 0.0);
    let mut i = 0;
    while i < tokens.len() {
        match &tokens[i] {
            Token::Cmd('M') | Token::Cmd('L') => {
                let command = match &tokens[i] {
                    Token::Cmd(c) => *c,
                    _ => unreachable!(),
                };
                i += 1;
                if i + 2 > tokens.len() {
                    return Err(format!("Truncated emblem command: {command}"));
                }
                x = operand(&tokens[i])?;
                y = operand(&tokens[i + 1])?;
                i += 2;
            }
            Token::Cmd('H') => {
                i += 1;
                if i + 1 > tokens.len() {
                    return Err("Truncated emblem command: H".to_string());
                }
                x = operand(&tokens[i])?;
                i += 1;
            }
            Token::Cmd('V') => {
                i += 1;
                if i + 1 > tokens.len() {
                    return Err("Truncated emblem command: V".to_string());
                }
                y = operand(&tokens[i])?;
                i += 1;
            }
            Token::Cmd('Z') => {
                i += 1;
                if points.len() < 3 {
                    return Err("Emblem polygon has fewer than 3 points".to_string());
                }
                result.push(std::mem::take(&mut points));
                continue;
            }
            Token::Cmd(c) => return Err(format!("Unsupported emblem command: {c}")),
            Token::Num(raw) => return Err(format!("Unsupported emblem command: {raw}")),
        }
        points.push((x, y));
    }
    if !points.is_empty() || result.is_empty() {
        return Err("Unterminated emblem geometry".to_string());
    }
    Ok(result)
}

fn inside(x: f64, y: f64, polygon: &[(f64, f64)]) -> bool {
    let mut hit = false;
    for (i, (ax, ay)) in polygon.iter().enumerate() {
        let (bx, by) = polygon[(i + 1) % polygon.len()];
        if (*ay > y) != (by > y) && x < (bx - ax) * (y - ay) / (by - ay) + ax {
            hit = !hit;
        }
    }
    hit
}

pub(crate) fn render_layers(layers: &[Vec<Vec<(f64, f64)>>]) -> (String, String) {
    let mut rows = Vec::new();
    for row in 0..16 {
        let mut cells = String::new();
        for column in 0..32 {
            let mut char = ' ';
            for (shape, glyph) in layers.iter().zip(['#', '@']) {
                let mut hits = 0;
                for polygon in shape {
                    if inside(
                        (column as f64 + 0.5) * 4.0,
                        (row as f64 + 0.5) * 8.0,
                        polygon,
                    ) {
                        hits += 1;
                    }
                }
                if hits % 2 == 1 {
                    char = glyph;
                }
            }
            cells.push(char);
        }
        rows.push(cells.trim_end().to_string());
    }
    let mut colored = Vec::new();
    for row in &rows {
        let mut line = String::new();
        let mut previous: Option<char> = None;
        for char in row.chars() {
            let color = if char == '#' { '1' } else { '2' };
            if char != ' ' && Some(color) != previous {
                line.push('$');
                line.push(color);
                previous = Some(color);
            }
            line.push(char);
        }
        line.push_str("$2");
        colored.push(line);
    }
    (colored.join("\n") + "\n", rows.join("\n") + "\n\nSODA OS\n")
}
