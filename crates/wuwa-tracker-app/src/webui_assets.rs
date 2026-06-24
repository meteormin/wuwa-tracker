pub struct EmbeddedAsset {
    pub path: &'static str,
    pub content_type: &'static str,
    pub bytes: &'static [u8],
}

include!(concat!(env!("OUT_DIR"), "/webui_assets.rs"));

pub fn get(path: &str) -> Option<&'static EmbeddedAsset> {
    let path = path.trim_start_matches('/');
    WEBUI_ASSETS.iter().find(|asset| asset.path == path)
}

pub fn index() -> Option<&'static EmbeddedAsset> {
    get("index.html")
}

pub fn has_assets() -> bool {
    !WEBUI_ASSETS.is_empty()
}
