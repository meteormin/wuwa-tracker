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
    /// `scan`, `run`, `autorun`에서 URL을 탐색할 기본 경로입니다.
    pub scan_path: Option<PathBuf>,
    /// `report`, `run`, `autorun`이 생성할 리포트 형식입니다.
    pub report_format: Option<String>,
    /// `report`, `run`, `autorun`이 생성할 리포트 경로입니다.
    pub report_output: Option<PathBuf>,
    /// `report`, `run`, `autorun`이 사용할 리포트 언어입니다.
    pub report_language: Option<String>,
    /// `autorun`의 URL 탐색 주기이며 단위는 초입니다.
    pub autorun_interval_secs: Option<u64>,
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
            report_format: Some("json".to_string()),
            report_output: Some(PathBuf::from("out")),
            report_language: Some("en".to_string()),
            autorun_interval_secs: Some(30),
        };

        save(&path, &settings).unwrap();
        let json: serde_json::Value = serde_json::from_slice(&fs::read(&path).unwrap()).unwrap();
        let loaded = load(&path).unwrap();
        clear(&path).unwrap();

        assert_eq!(json["reportFormat"], "json");
        assert_eq!(json["reportOutput"], "out");
        assert_eq!(json["reportLanguage"], "en");
        assert_eq!(json["autorunIntervalSecs"], 30);
        assert_eq!(loaded.scan_path, settings.scan_path);
        assert_eq!(loaded.report_format, settings.report_format);
        assert_eq!(loaded.report_output, settings.report_output);
        assert_eq!(loaded.report_language, settings.report_language);
        assert_eq!(loaded.autorun_interval_secs, settings.autorun_interval_secs);
    }
}
