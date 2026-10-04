//! Go `path`/`filepath` lexical helpers (`Clean`, `Join`, `Dir`, `Base`)
//! and symlink resolution (`EvalSymlinks`) for Linux `/` separators.

/// Go `path.Clean`: lexical normalization; empty becomes `"."`.
pub fn clean(path: &str) -> String {
    if path.is_empty() {
        return ".".to_string();
    }
    let rooted = path.starts_with('/');
    let mut fixed: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if fixed.last().is_some_and(|last| *last != "..") {
                    fixed.pop();
                } else if !rooted {
                    fixed.push("..");
                }
            }
            _ => fixed.push(part),
        }
    }
    let joined = fixed.join("/");
    if rooted {
        format!("/{joined}")
    } else if joined.is_empty() {
        ".".to_string()
    } else {
        joined
    }
}

/// Go `filepath.Join`: join with `/`, then `Clean`.
pub fn join(first: &str, rest: &[&str]) -> String {
    let mut path = first.to_string();
    for part in rest {
        if path.ends_with('/') {
            path.push_str(part);
        } else {
            path.push('/');
            path.push_str(part);
        }
    }
    clean(&path)
}

/// Go `filepath.Dir`: `Clean` of the `Split` directory part, which keeps
/// its trailing slash, so `"/a/b/"` is `"/a/b"`.
pub fn dir(path: &str) -> String {
    match path.rfind('/') {
        Some(i) => clean(&path[..i + 1]),
        None => ".".to_string(),
    }
}

/// Go `filepath.Base`: final element; empty becomes `"."`, root stays `/`.
pub fn base(path: &str) -> String {
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() {
        if path.is_empty() {
            return ".".to_string();
        }
        return "/".to_string();
    }
    match trimmed.rfind('/') {
        Some(i) => trimmed[i + 1..].to_string(),
        None => trimmed.to_string(),
    }
}

/// Go `filepath.EvalSymlinks`: resolve every symlink lexically, following
/// the same plan (walk each element, resolve links against their directory).
/// Failures keep Go's `lstat`/`readlink` error shapes.
pub fn eval_symlinks(path: &str) -> Result<String, crate::errors::Error> {
    use std::os::unix::ffi::OsStrExt;
    if path.is_empty() {
        return Err(crate::errors::path_error(
            "lstat",
            "",
            std::io::Error::from_raw_os_error(libc::ENOENT),
        ));
    }
    let mut link_count = 0;
    let mut current = path.to_string();
    // Resolve one link per pass; each pass rewalks from the start, as Go's
    // element walk does when a link redirects the remainder.
    loop {
        let rooted = current.starts_with('/');
        let parts: Vec<&str> = current.split('/').collect();
        let mut stack: Vec<String> = Vec::new();
        let mut changed = false;
        let mut failed: Option<crate::errors::Error> = None;
        for (i, part) in parts.iter().enumerate() {
            if part.is_empty() || *part == "." {
                continue;
            }
            if *part == ".." {
                if stack.last().is_some_and(|last| last != "..") {
                    stack.pop();
                } else if !rooted {
                    stack.push("..".to_string());
                }
                continue;
            }
            stack.push(part.to_string());
            let lookup = if rooted {
                format!("/{}", stack.join("/"))
            } else {
                stack.join("/")
            };
            match std::fs::symlink_metadata(&lookup) {
                Ok(meta) => {
                    if meta.file_type().is_symlink() {
                        link_count += 1;
                        if link_count > 255 {
                            return Err(crate::errors::Error::msg("EvalSymlinks: too many links"));
                        }
                        let target = std::fs::read_link(&lookup)
                            .map_err(|e| crate::errors::link_error("readlink", &lookup, "", e))?;
                        let target = String::from_utf8_lossy(target.as_os_str().as_bytes()).into_owned();
                        let rest = parts[i + 1..].join("/");
                        let parent = dir(&lookup);
                        current = if target.starts_with('/') {
                            if rest.is_empty() { target } else { format!("{target}/{rest}") }
                        } else if rest.is_empty() {
                            join(&parent, &[&target])
                        } else {
                            join(&parent, &[&target, rest.as_str()])
                        };
                        changed = true;
                        break;
                    }
                }
                Err(e) => {
                    failed = Some(crate::errors::path_error("lstat", &lookup, e));
                    break;
                }
            }
        }
        if changed {
            continue;
        }
        if let Some(err) = failed {
            return Err(err);
        }
        let joined = stack.join("/");
        if rooted {
            return Ok(format!("/{}", joined));
        }
        if joined.is_empty() {
            return Ok(".".to_string());
        }
        return Ok(joined);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn clean_vectors() {
        for (input, want) in [
            ("", "."),
            ("/", "/"),
            ("//", "/"),
            ("/a/b/c", "/a/b/c"),
            ("/a//b/./c/", "/a/b/c"),
            ("/a/../../b", "/b"),
            ("a/../../b", "../b"),
            ("a/./b/../c", "a/c"),
            ("../..", "../.."),
            ("../../a", "../../a"),
            ("a/b/../../../c", "../c"),
            (".", "."),
            ("./", "."),
            ("a/", "a"),
        ] {
            assert_eq!(clean(input), want, "clean({input:?})");
        }
    }

    #[test]
    fn join_dir_base_vectors() {
        assert_eq!(join("/a", &["b", "c"]), "/a/b/c");
        assert_eq!(join("a", &["../b"]), "b");
        assert_eq!(dir("/a/b/c"), "/a/b");
        assert_eq!(dir("/a"), "/");
        assert_eq!(dir("a"), ".");
        assert_eq!(dir("/"), "/");
        assert_eq!(dir(""), ".");
        assert_eq!(dir("/a/b/"), "/a/b");
        assert_eq!(dir("a/b/"), "a/b");
        assert_eq!(base("/a/b/c"), "c");
        assert_eq!(base("/a/b/"), "b");
        assert_eq!(base("/"), "/");
        assert_eq!(base(""), ".");
        assert_eq!(base("a"), "a");
    }

    #[test]
    fn symlinks_resolve() {
        let root = std::env::temp_dir().join(format!("soda-install-pathtest-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&root);
        std::fs::create_dir_all(root.join("a/b")).unwrap();
        std::fs::write(root.join("a/b/file"), b"x").unwrap();
        std::os::unix::fs::symlink(root.join("a"), root.join("link")).unwrap();
        std::os::unix::fs::symlink("b/file", root.join("a/rel")).unwrap();
        let got = eval_symlinks(root.join("link/b/file").to_str().unwrap()).unwrap();
        assert_eq!(got, root.join("a/b/file").to_str().unwrap());
        let got = eval_symlinks(root.join("a/rel").to_str().unwrap()).unwrap();
        assert_eq!(got, root.join("a/b/file").to_str().unwrap());
        assert!(eval_symlinks(root.join("missing").to_str().unwrap()).is_err());
        std::os::unix::fs::symlink("loop", root.join("loop")).unwrap();
        assert!(eval_symlinks(root.join("loop").to_str().unwrap()).is_err());
        let _ = std::fs::remove_dir_all(&root);
    }
}
