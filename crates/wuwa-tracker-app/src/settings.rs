use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::{
    fs,
    path::{Path, PathBuf},
};
use tracing::debug;

pub const DEFAULT_FORMAT: &str = "html";
pub const DEFAULT_OUTPUT: &str = "report";
pub const DEFAULT_LANG: &str = "ko";

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
/// CLI 명령이 공유하는 선택적 기본값입니다.
pub struct Settings {
    /// 기존 설정 파일의 `path` key를 읽되 이후 저장부터 `scanPath`를 사용합니다.
    #[serde(alias = "path")]
    pub scan_path: Option<PathBuf>,
    pub format: Option<String>,
    pub output: Option<PathBuf>,
    pub lang: Option<String>,
    pub interval_secs: Option<u64>,
}

/// 설정 파일이 없으면 오류 대신 빈 기본값을 반환합니다.
pub fn load(path: &Path) -> Result<Settings> {
    match fs::read(path) {
        Ok(bytes) => {
            debug!(event = "settings_loaded", path = %path.display(), bytes = bytes.len());
            Ok(serde_json::from_slice(&bytes)?)
        }
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            debug!(event = "settings_missing", path = %path.display());
            Ok(Settings::default())
        }
        Err(error) => Err(error.into()),
    }
}

pub fn save(path: &Path, settings: &Settings) -> Result<()> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)?;
    }
    fs::write(path, serde_json::to_vec_pretty(settings)?)?;
    debug!(event = "settings_saved", path = %path.display());
    Ok(())
}

/// 설정 파일이 이미 없어도 성공으로 처리합니다.
pub fn clear(path: &Path) -> Result<()> {
    match fs::remove_file(path) {
        Ok(()) => {
            debug!(event = "settings_cleared", path = %path.display());
            Ok(())
        }
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
            scan_path: Some(PathBuf::from("game")),
            format: Some("json".to_string()),
            output: Some(PathBuf::from("out")),
            lang: Some("en".to_string()),
            interval_secs: Some(30),
        };

        save(&path, &settings).unwrap();
        let loaded = load(&path).unwrap();
        clear(&path).unwrap();

        assert_eq!(loaded.scan_path, settings.scan_path);
        assert_eq!(loaded.format, settings.format);
        assert_eq!(loaded.output, settings.output);
        assert_eq!(loaded.lang, settings.lang);
        assert_eq!(loaded.interval_secs, settings.interval_secs);
    }

    #[test]
    fn settings_accept_legacy_path() {
        let settings: Settings = serde_json::from_str(r#"{"path":"game"}"#).unwrap();

        assert_eq!(settings.scan_path, Some(PathBuf::from("game")));
    }
}
