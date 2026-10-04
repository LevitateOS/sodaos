//! `complete.go`: Quadlet localization for ordinary Podman import.

use crate::error::Error;

fn rewrite_quadlet_line(
    line: &str,
    reference: &str,
    image: &mut i32,
    container: &mut i32,
    unit: &mut i32,
) -> Result<Vec<String>, Error> {
    if line == "[Unit]" {
        *unit += 1;
        return Ok(vec![
            line.to_string(),
            "Requires=soda-image-import.service".to_string(),
            "After=soda-image-import.service".to_string(),
        ]);
    }
    if line == "[Container]" {
        *container += 1;
        return Ok(vec![line.to_string(), "Pull=never".to_string()]);
    }
    if let Some(_rest) = line.strip_prefix("Image=") {
        *image += 1;
        return Ok(vec![format!("Image={reference}")]);
    }
    if line.starts_with("Pull=") {
        return Ok(Vec::new());
    }
    if line.starts_with("GlobalArgs=") {
        return Err(Error::msg("unexpected existing Quadlet storage arguments"));
    }
    Ok(vec![line.to_string()])
}

pub fn local_quadlet(body: &str, reference: &str) -> Result<String, Error> {
    let lines: Vec<&str> = body.split('\n').collect();
    let (mut image, mut container, mut unit) = (0, 0, 0);
    let mut out: Vec<String> = Vec::new();
    for line in lines {
        let mut extra =
            rewrite_quadlet_line(line, reference, &mut image, &mut container, &mut unit)?;
        out.append(&mut extra);
    }
    if image != 1 || container != 1 || unit != 1 {
        return Err(Error::msg("one fixed Quadlet container/image required"));
    }
    Ok(out.join("\n"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_quadlet_rewrite_and_refusals() {
        // Oracle: Go TestPublicStageAndQuadletRefuseAmbiguity vectors.
        let body = "[Unit]\nDescription=x\n[Container]\nImage=localhost/soda-dashboard:dev\nPull=newer\nExec=a\n";
        let out = local_quadlet(body, "sha256:deadbeef").unwrap();
        assert!(out.contains("Requires=soda-image-import.service"));
        assert!(out.contains("Pull=never"));
        assert!(out.contains("Image=sha256:deadbeef"));
        assert!(!out.contains("Pull=newer"));
        assert!(local_quadlet("[Unit]\n[Container]\nGlobalArgs=x\n", "r").is_err());
        assert!(local_quadlet("[Unit]\n[Container]\n", "r").is_err());
        assert!(local_quadlet("[Unit]\n[Unit]\n[Container]\nImage=a\n", "r").is_err());
    }
}
