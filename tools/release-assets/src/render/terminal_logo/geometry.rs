//! Polygon tokenization and even-odd raster geometry.

/// Parse absolute polygon path data into explicitly closed rings.
pub(crate) fn polygons(data: &str) -> Result<Vec<Vec<(f64, f64)>>, String> {
    let mut result: Vec<Vec<(f64, f64)>> = Vec::new();
    let mut points: Vec<(f64, f64)> = Vec::new();
    let (mut x, mut y) = (0.0, 0.0);
    for parsed in svgtypes::PathParser::from(data) {
        let segment = parsed.map_err(|_| "Malformed emblem geometry".to_string())?;
        match segment {
            svgtypes::PathSegment::MoveTo {
                abs: true,
                x: next_x,
                y: next_y,
            } => {
                if !points.is_empty() {
                    return Err("Unterminated emblem geometry".to_string());
                }
                x = finite(next_x)?;
                y = finite(next_y)?;
                points.push((x, y));
            }
            svgtypes::PathSegment::LineTo {
                abs: true,
                x: next_x,
                y: next_y,
            } => {
                require_ring(&points)?;
                x = finite(next_x)?;
                y = finite(next_y)?;
                points.push((x, y));
            }
            svgtypes::PathSegment::HorizontalLineTo {
                abs: true,
                x: next_x,
            } => {
                require_ring(&points)?;
                x = finite(next_x)?;
                points.push((x, y));
            }
            svgtypes::PathSegment::VerticalLineTo {
                abs: true,
                y: next_y,
            } => {
                require_ring(&points)?;
                y = finite(next_y)?;
                points.push((x, y));
            }
            svgtypes::PathSegment::ClosePath { abs: true } => {
                if points.len() < 3 {
                    return Err("Emblem polygon has fewer than 3 points".to_string());
                }
                result.push(std::mem::take(&mut points));
            }
            other => {
                return Err(format!(
                    "Unsupported emblem command: {}",
                    other.command() as char
                ));
            }
        }
    }
    if !points.is_empty() || result.is_empty() {
        return Err("Unterminated emblem geometry".to_string());
    }
    Ok(result)
}

fn finite(value: f64) -> Result<f64, String> {
    if value.is_finite() {
        Ok(value)
    } else {
        Err("Non-finite emblem coordinate".to_string())
    }
}

fn require_ring(points: &[(f64, f64)]) -> Result<(), String> {
    if points.is_empty() {
        Err("Emblem geometry must start with M".to_string())
    } else {
        Ok(())
    }
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
