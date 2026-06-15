use crate::types::{GachaType, LuckScoreThreshold};
use serde::Serialize;
use std::{env, path::PathBuf};

const ENV_DB_PATH: &str = "WUWA_TRACKER_DB_PATH";

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Config {
    pub resources_url: String,
    pub tracking_url: String,
    pub standard_five_star_resources: Vec<i32>,
    pub gacha_types: Vec<GachaType>,
    pub luck_score_thresholds: Vec<LuckScoreThreshold>,
    pub astrite_per_pull: usize,
    pub scan_log_paths: Vec<PathBuf>,
    pub db_path: PathBuf,
    pub report_format: String,
    pub report_output: String,
    pub language: String,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            resources_url: "https://aki-gm-resources-oversea.aki-game.net".to_string(),
            tracking_url: "https://gmserver-api.aki-game2.net".to_string(),
            standard_five_star_resources: vec![1203, 1301, 1405, 1104, 1503],
            gacha_types: vec![
                gacha_type(1, "characterEvent", true, 80),
                gacha_type(2, "weaponEvent", false, 55),
                gacha_type(3, "characterPermanent", false, 55),
                gacha_type(4, "weaponPermanent", false, 55),
                gacha_type(5, "beginner", true, 55),
                gacha_type(6, "beginnerSelect", false, 55),
                gacha_type(8, "characterNovice", true, 80),
                gacha_type(9, "weaponNovice", false, 55),
                gacha_type(10, "characterCollaboration", true, 80),
                gacha_type(11, "weaponCollaboration", false, 55),
            ],
            luck_score_thresholds: vec![
                threshold(0.0, "worst"),
                threshold(85.0, "bad"),
                threshold(95.0, "normal"),
                threshold(102.0, "good"),
                threshold(115.0, "best"),
            ],
            astrite_per_pull: 160,
            scan_log_paths: vec![
                PathBuf::from("Client/Saved/Logs/Client.log"),
                PathBuf::from("Client/Binaries/Win64/ThirdParty/KrPcSdk_Global/KRSDKRes/KRSDKWebView/debug.log"),
                PathBuf::from("Data/Library/Logs/Client/Client.log"),
                PathBuf::from("Client/Client.log"),
                PathBuf::from("Client.log"),
            ],
            db_path: default_db_path(),
            report_format: "html".to_string(),
            report_output: "report".to_string(),
            language: "ko".to_string(),
        }
    }
}

fn gacha_type(id: i32, key: &str, has_off_banner_drop: bool, expected_pulls: i32) -> GachaType {
    GachaType {
        id,
        key: key.to_string(),
        has_off_banner_drop,
        name: key.to_string(),
        base_rate: 0.8,
        expected_pulls,
    }
}

fn threshold(min_score: f64, state: &str) -> LuckScoreThreshold {
    LuckScoreThreshold {
        min_score,
        state: state.to_string(),
    }
}

fn default_db_path() -> PathBuf {
    if let Ok(value) = env::var(ENV_DB_PATH) {
        let value = value.trim();
        if !value.is_empty() {
            return PathBuf::from(value);
        }
    }

    dirs::home_dir()
        .unwrap_or_else(|| PathBuf::from("."))
        .join(".wuwa-tracker")
        .join("store.json")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_config_uses_db_path_env() {
        let previous = env::var(ENV_DB_PATH).ok();
        let expected = PathBuf::from("custom-store.json");
        env::set_var(ENV_DB_PATH, &expected);

        let config = Config::default();

        assert_eq!(config.db_path, expected);
        if let Some(value) = previous {
            env::set_var(ENV_DB_PATH, value);
        } else {
            env::remove_var(ENV_DB_PATH);
        }
    }
}
