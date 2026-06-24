use serde::Serialize;
use std::path::PathBuf;
use wuwa_tracker_types::{GachaType, LuckScoreThreshold};

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
    pub log_path: PathBuf,
    pub report_format: String,
    pub report_output: String,
    pub language: String,
}

impl Default for Config {
    fn default() -> Self {
        ConfigBuilder::default().build()
    }
}

impl Config {
    pub fn builder() -> ConfigBuilder {
        ConfigBuilder::default()
    }
}

#[derive(Debug, Clone)]
pub struct ConfigBuilder {
    resources_url: String,
    tracking_url: String,
    standard_five_star_resources: Vec<i32>,
    gacha_types: Vec<GachaType>,
    luck_score_thresholds: Vec<LuckScoreThreshold>,
    astrite_per_pull: usize,
    scan_log_paths: Vec<PathBuf>,
    db_path: PathBuf,
    log_path: PathBuf,
    report_format: String,
    report_output: String,
    language: String,
}

impl Default for ConfigBuilder {
    fn default() -> Self {
        let app_dir = default_app_dir();
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
            db_path: app_dir.join("store.json"),
            log_path: app_dir.join("wuwa-tracker.log"),
            report_format: "html".to_string(),
            report_output: "report".to_string(),
            language: "ko".to_string(),
        }
    }
}

impl ConfigBuilder {
    pub fn resources_url(mut self, url: impl Into<String>) -> Self {
        self.resources_url = url.into();
        self
    }

    pub fn tracking_url(mut self, url: impl Into<String>) -> Self {
        self.tracking_url = url.into();
        self
    }

    pub fn standard_five_star_resources(mut self, resources: impl Into<Vec<i32>>) -> Self {
        self.standard_five_star_resources = resources.into();
        self
    }

    pub fn gacha_types(mut self, gacha_types: impl Into<Vec<GachaType>>) -> Self {
        self.gacha_types = gacha_types.into();
        self
    }

    pub fn luck_score_thresholds(mut self, thresholds: impl Into<Vec<LuckScoreThreshold>>) -> Self {
        self.luck_score_thresholds = thresholds.into();
        self
    }

    pub fn astrite_per_pull(mut self, value: usize) -> Self {
        self.astrite_per_pull = value;
        self
    }

    pub fn scan_log_paths(mut self, paths: impl Into<Vec<PathBuf>>) -> Self {
        self.scan_log_paths = paths.into();
        self
    }

    pub fn db_path(mut self, path: impl Into<PathBuf>) -> Self {
        self.db_path = path.into();
        self
    }

    pub fn log_path(mut self, path: impl Into<PathBuf>) -> Self {
        self.log_path = path.into();
        self
    }

    pub fn report_format(mut self, format: impl Into<String>) -> Self {
        self.report_format = format.into();
        self
    }

    pub fn report_output(mut self, output: impl Into<String>) -> Self {
        self.report_output = output.into();
        self
    }

    pub fn language(mut self, language: impl Into<String>) -> Self {
        self.language = language.into();
        self
    }

    pub fn build(self) -> Config {
        Config {
            resources_url: self.resources_url,
            tracking_url: self.tracking_url,
            standard_five_star_resources: self.standard_five_star_resources,
            gacha_types: self.gacha_types,
            luck_score_thresholds: self.luck_score_thresholds,
            astrite_per_pull: self.astrite_per_pull,
            scan_log_paths: self.scan_log_paths,
            db_path: self.db_path,
            log_path: self.log_path,
            report_format: self.report_format,
            report_output: self.report_output,
            language: self.language,
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

// 기본 베이스가 되는 앱 폴더 경로 구하기 (~/.wuwa-tracker)
fn default_app_dir() -> PathBuf {
    dirs::home_dir()
        .unwrap_or_else(|| PathBuf::from("."))
        .join(".wuwa-tracker")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_builder_overrides_db_path() {
        let expected = PathBuf::from("custom-store.json");

        let config = Config::builder().db_path(&expected).build();

        assert_eq!(config.db_path, expected);
    }

    #[test]
    fn config_builder_overrides_log_path() {
        let expected = PathBuf::from("custom-app.log");

        let config = Config::builder().log_path(&expected).build();

        assert_eq!(config.log_path, expected);
    }

    #[test]
    fn config_builder_overrides_runtime_fields() {
        let gacha_types = vec![GachaType {
            id: 99,
            key: "testBanner".to_string(),
            has_off_banner_drop: true,
            name: "testBanner".to_string(),
            base_rate: 1.5,
            expected_pulls: 10,
        }];
        let thresholds = vec![LuckScoreThreshold {
            min_score: 10.0,
            state: "test".to_string(),
        }];
        let scan_paths = vec![PathBuf::from("test.log")];

        let config = Config::builder()
            .resources_url("https://resources.example.test")
            .tracking_url("https://tracking.example.test")
            .standard_five_star_resources(vec![42])
            .gacha_types(gacha_types.clone())
            .luck_score_thresholds(thresholds.clone())
            .astrite_per_pull(10)
            .scan_log_paths(scan_paths.clone())
            .report_format("json")
            .report_output("test-report")
            .language("en")
            .build();

        assert_eq!(config.resources_url, "https://resources.example.test");
        assert_eq!(config.tracking_url, "https://tracking.example.test");
        assert_eq!(config.standard_five_star_resources, vec![42]);
        assert_eq!(config.gacha_types.len(), gacha_types.len());
        assert_eq!(config.gacha_types[0].id, 99);
        assert_eq!(config.gacha_types[0].key, "testBanner");
        assert_eq!(config.gacha_types[0].base_rate, 1.5);
        assert_eq!(config.luck_score_thresholds.len(), thresholds.len());
        assert_eq!(config.luck_score_thresholds[0].min_score, 10.0);
        assert_eq!(config.luck_score_thresholds[0].state, "test");
        assert_eq!(config.astrite_per_pull, 10);
        assert_eq!(config.scan_log_paths, scan_paths);
        assert_eq!(config.report_format, "json");
        assert_eq!(config.report_output, "test-report");
        assert_eq!(config.language, "en");
    }
}
