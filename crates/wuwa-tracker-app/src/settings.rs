use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::{
    fs,
    path::{Path, PathBuf},
};

pub const DEFAULT_FORMAT: &str = "html";
pub const DEFAULT_OUTPUT: &str = "report";
pub const DEFAULT_LANG: &str = "ko";

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Settings {
    pub path: Option<PathBuf>,
    pub format: Option<String>,
    pub output: Option<PathBuf>,
    pub lang: Option<String>,
    pub interval_secs: Option<u64>,
}

pub fn load(path: &Path) -> Result<Settings> {
    match fs::read(path) {
        Ok(bytes) => Ok(serde_json::from_slice(&bytes)?),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(Settings::default()),
        Err(error) => Err(error.into()),
    }
}

pub fn save(path: &Path, settings: &Settings) -> Result<()> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)?;
    }
    fs::write(path, serde_json::to_vec_pretty(settings)?)?;
    Ok(())
}

pub fn clear(path: &Path) -> Result<()> {
    match fs::remove_file(path) {
        Ok(()) => Ok(()),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(error) => Err(error.into()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::{SystemTime, UNIX_EPOCH};

    #[test]
    fn settings_round_trip() {
        let path = std::env::temp_dir().join(format!(
            "wuwa-tracker-settings-{}.json",
            SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        let settings = Settings {
            path: Some(PathBuf::from("game")),
            format: Some("json".to_string()),
            output: Some(PathBuf::from("out")),
            lang: Some("en".to_string()),
            interval_secs: Some(30),
        };

        save(&path, &settings).unwrap();
        let loaded = load(&path).unwrap();
        clear(&path).unwrap();

        assert_eq!(loaded.path, settings.path);
        assert_eq!(loaded.format, settings.format);
        assert_eq!(loaded.output, settings.output);
        assert_eq!(loaded.lang, settings.lang);
        assert_eq!(loaded.interval_secs, settings.interval_secs);
    }
}
