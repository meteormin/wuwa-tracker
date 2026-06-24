use std::path::Path;

fn main() {
    embed_webui_assets();
    tauri_build::build();
}

fn embed_webui_assets() {
    use std::{env, fs, path::PathBuf};

    let manifest_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("manifest dir"));
    let workspace_dir = manifest_dir
        .parent()
        .and_then(Path::parent)
        .map(Path::to_path_buf)
        .unwrap_or_else(|| PathBuf::from("."));
    let dist_dir = workspace_dir.join("crates/wuwa-tracker-webui/dist");
    println!("cargo:rerun-if-changed={}", dist_dir.display());

    let mut entries = Vec::new();
    if let Ok(read_dir) = fs::read_dir(&dist_dir) {
        for entry in read_dir.flatten() {
            let path = entry.path();
            if !path.is_file() {
                continue;
            }
            let Some(name) = path.file_name().and_then(|name| name.to_str()) else {
                continue;
            };
            entries.push((name.to_string(), path));
        }
    }
    entries.sort_by(|left, right| left.0.cmp(&right.0));

    let mut output = String::from("pub static WEBUI_ASSETS: &[EmbeddedAsset] = &[\n");
    for (name, path) in entries {
        let content_type = content_type(&name);
        output.push_str(&format!(
            "    EmbeddedAsset {{ path: {:?}, content_type: {:?}, bytes: include_bytes!({:?}) }},\n",
            name,
            content_type,
            path.display().to_string(),
        ));
    }
    output.push_str("];\n");

    let out_dir = PathBuf::from(env::var("OUT_DIR").expect("out dir"));
    fs::write(out_dir.join("webui_assets.rs"), output).expect("write embedded webui assets");
}

fn content_type(path: &str) -> &'static str {
    match Path::new(path).extension().and_then(|value| value.to_str()) {
        Some("html") => "text/html; charset=utf-8",
        Some("js") => "text/javascript; charset=utf-8",
        Some("css") => "text/css; charset=utf-8",
        Some("wasm") => "application/wasm",
        Some("svg") => "image/svg+xml",
        Some("png") => "image/png",
        Some("jpg" | "jpeg") => "image/jpeg",
        Some("webp") => "image/webp",
        Some("ico") => "image/x-icon",
        Some("json") => "application/json; charset=utf-8",
        _ => "application/octet-stream",
    }
}
