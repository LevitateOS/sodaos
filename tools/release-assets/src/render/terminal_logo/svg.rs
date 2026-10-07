//! Bounded XML gate for Soda's canonical emblem shape.

pub(crate) type Attrs = Vec<(String, String)>;

const SVG_NAMESPACE: &str = "http://www.w3.org/2000/svg";
pub(super) const MAX_DOCUMENT_BYTES: usize = 64 * 1024;
const MAX_DOCUMENT_NODES: u32 = 256;

fn attributes(node: roxmltree::Node<'_, '_>) -> Result<Attrs, String> {
    let mut attrs = Vec::new();
    for attribute in node.attributes() {
        let name = attribute.name();
        if attribute.namespace().is_some() || name == "transform" {
            return Err("Unsupported emblem attribute".to_string());
        }
        attrs.push((name.to_string(), attribute.value().to_string()));
    }
    Ok(attrs)
}

fn is_svg_element(node: roxmltree::Node<'_, '_>, name: &str) -> bool {
    node.is_element()
        && node.tag_name().namespace() == Some(SVG_NAMESPACE)
        && node.tag_name().name() == name
}

fn only_text_or_comments(node: roxmltree::Node<'_, '_>, allow_text: bool) -> Result<(), String> {
    for child in node.children() {
        match child.node_type() {
            roxmltree::NodeType::Text
                if allow_text || child.text().unwrap_or("").trim().is_empty() => {}
            roxmltree::NodeType::Comment => {}
            _ => return Err("Unexpected emblem element content".to_string()),
        }
    }
    Ok(())
}

/// Parse the canonical document shape: one SVG root holding only title,
/// description, path children and comments. XML syntax and namespace
/// resolution belong to roxmltree; Soda keeps the narrow shape gate.
pub(crate) fn parse_svg(text: &str) -> Result<(Attrs, Vec<Attrs>), String> {
    if text.len() > MAX_DOCUMENT_BYTES {
        return Err("Emblem document exceeds size limit".to_string());
    }
    let document = roxmltree::Document::parse_with_options(
        text,
        roxmltree::ParsingOptions {
            allow_dtd: false,
            nodes_limit: MAX_DOCUMENT_NODES,
            entity_resolver: None,
        },
    )
    .map_err(|_| "Malformed emblem document".to_string())?;
    let root = document.root_element();
    if !root.is_element() || root.tag_name().name() != "svg" {
        return Err("Emblem root is not an SVG document".to_string());
    }
    let root_attrs = attributes(root)?;
    if root.tag_name().namespace() != Some(SVG_NAMESPACE) {
        return Ok((root_attrs, Vec::new()));
    }

    let mut paths = Vec::new();
    for child in root.children() {
        match child.node_type() {
            roxmltree::NodeType::Element if is_svg_element(child, "path") => {
                only_text_or_comments(child, false)?;
                paths.push(attributes(child)?);
            }
            roxmltree::NodeType::Element
                if is_svg_element(child, "title") || is_svg_element(child, "desc") =>
            {
                only_text_or_comments(child, true)?;
            }
            roxmltree::NodeType::Text if child.text().unwrap_or("").trim().is_empty() => {}
            roxmltree::NodeType::Comment => {}
            _ => {
                let name = if child.is_element() {
                    child.tag_name().name()
                } else {
                    "?"
                };
                return Err(format!("Unexpected emblem element: {name}"));
            }
        }
    }
    Ok((root_attrs, paths))
}

pub(crate) fn attr<'a>(attrs: &'a Attrs, name: &str) -> Option<&'a str> {
    attrs
        .iter()
        .find(|(key, _)| key == name)
        .map(|(_, value)| value.as_str())
}
