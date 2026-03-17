use serde::Deserialize;
use std::fs;
use std::path::{Component, Path, PathBuf};

#[derive(Deserialize)]
struct PathInput {
    path: String,
}

#[derive(Deserialize)]
struct WriteInput {
    path: String,
    content: String,
}

pub fn call_tool(root: &Path, name: &str, raw: &str) -> String {
    match name {
        "listFiles" => list_files(root, raw),
        "readFile" => read_file(root, raw),
        "writeFile" => write_file(root, raw),
        _ => format!("未知工具: {name}"),
    }
}

fn list_files(root: &Path, raw: &str) -> String {
    let input = if raw.trim().is_empty() {
        PathInput {
            path: ".".to_string(),
        }
    } else {
        match serde_json::from_str::<PathInput>(raw) {
            Ok(value) => value,
            Err(_) => return r#"listFiles 参数应为 {"path":"."}"#.to_string(),
        }
    };

    let (target, show) = match safe_path(root, &input.path) {
        Ok(value) => value,
        Err(err) => return err,
    };

    let entries = match fs::read_dir(&target) {
        Ok(value) => value,
        Err(err) => return err.to_string(),
    };

    let mut lines = vec![format!("目录 {show}:")];
    for entry in entries {
        let entry = match entry {
            Ok(value) => value,
            Err(err) => return err.to_string(),
        };

        let mut name = entry.file_name().to_string_lossy().to_string();
        if entry.path().is_dir() {
            name.push('/');
        }
        lines.push(name);
    }

    if lines.len() == 1 {
        return format!("{show} 是空目录");
    }

    lines.join("\n")
}

fn read_file(root: &Path, raw: &str) -> String {
    let input = match serde_json::from_str::<PathInput>(raw) {
        Ok(value) if !value.path.trim().is_empty() => value,
        _ => return r#"readFile 参数应为 {"path":"src/main.rs"}"#.to_string(),
    };

    let (target, show) = match safe_path(root, &input.path) {
        Ok(value) => value,
        Err(err) => return err,
    };

    let data = match fs::read(&target) {
        Ok(value) => value,
        Err(err) => return err.to_string(),
    };

    let text = String::from_utf8_lossy(&data);
    if text.chars().count() > 8_000 {
        let preview: String = text.chars().take(8_000).collect();
        return format!("文件 {show}:\n{preview}\n...[已截断]");
    }

    format!("文件 {show}:\n{text}")
}

fn write_file(root: &Path, raw: &str) -> String {
    let input = match serde_json::from_str::<WriteInput>(raw) {
        Ok(value) if !value.path.trim().is_empty() => value,
        _ => return r#"writeFile 参数应为 {"path":"a.txt","content":"hello"}"#.to_string(),
    };

    let (target, show) = match safe_path(root, &input.path) {
        Ok(value) => value,
        Err(err) => return err,
    };

    if let Some(parent) = target.parent() {
        if let Err(err) = fs::create_dir_all(parent) {
            return err.to_string();
        }
    }

    if let Err(err) = fs::write(&target, input.content) {
        return err.to_string();
    }

    format!("已写入 {show}")
}

fn safe_path(root: &Path, input: &str) -> Result<(PathBuf, String), String> {
    let root = root.canonicalize().map_err(|err| err.to_string())?;
    let raw = if input.trim().is_empty() {
        "."
    } else {
        input.trim()
    };

    if Path::new(raw).is_absolute() {
        return Err("只允许相对路径".to_string());
    }

    let mut full = root.clone();
    for component in Path::new(raw).components() {
        match component {
            Component::CurDir => {}
            Component::Normal(part) => full.push(part),
            Component::ParentDir => {
                if full == root {
                    return Err(format!("路径超出工作目录: {raw}"));
                }
                full.pop();
            }
            _ => return Err(format!("非法路径: {raw}")),
        }
    }

    let show = full
        .strip_prefix(&root)
        .ok()
        .and_then(|path| path.to_str())
        .unwrap_or(".")
        .to_string();

    if show.is_empty() {
        Ok((full, ".".to_string()))
    } else {
        Ok((full, show))
    }
}
