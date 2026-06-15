use crate::{
    config::Config,
    error::AppError,
    reporter, scanner,
    stats::StatsCalculator,
    store::JsonStore,
    tracker::{self, TrackerClient},
    types::{FetchResult, GachaType, ReportData, ReportFormat, ScanResponse, StatsResponse},
};
use std::{collections::BTreeMap, fs, path::Path, sync::Arc};

#[derive(Clone)]
pub struct Service {
    config: Arc<Config>,
    store: Arc<JsonStore>,
    calc: StatsCalculator,
    tracker: TrackerClient,
}

impl Service {
    pub fn new(config: Config) -> Result<Self, AppError> {
        let calc = StatsCalculator::new(&config);
        let store = Arc::new(JsonStore::new(config.db_path.clone())?);
        let tracker = TrackerClient::new(config.resources_url.clone(), config.tracking_url.clone());
        Ok(Self {
            config: Arc::new(config),
            store,
            calc,
            tracker,
        })
    }

    pub fn config(&self) -> &Config {
        &self.config
    }

    pub fn list_players(&self) -> Vec<String> {
        self.store.list_players()
    }

    pub fn scan(&self, path: impl AsRef<Path>) -> Result<ScanResponse, AppError> {
        let url = scanner::scan_url(path.as_ref(), &self.config.scan_log_paths)?;
        Ok(ScanResponse { success: true, url })
    }

    pub fn upload(&self, fetch_result: FetchResult) -> Result<StatsResponse, AppError> {
        let player_id = fetch_result.payload.player_id.trim().to_string();
        self.save_fetch_result(fetch_result)?;
        self.get_stats(player_id)
    }

    pub fn save_fetch_result(&self, fetch_result: FetchResult) -> Result<(), AppError> {
        let player_id = fetch_result.payload.player_id.trim();
        if player_id.is_empty() {
            return Err(AppError::MissingPlayerId);
        }
        if fetch_result.records.is_empty() {
            return Err(AppError::EmptyUploadData);
        }

        for gacha_type in &self.config.gacha_types {
            let records = fetch_result
                .records
                .get(&gacha_type.key)
                .cloned()
                .unwrap_or_default();
            self.store
                .save_gacha_records(player_id, &gacha_type.key, &records)?;
        }
        Ok(())
    }

    pub fn get_stats(&self, player_id: impl AsRef<str>) -> Result<StatsResponse, AppError> {
        let player_id = player_id.as_ref().trim();
        if player_id.is_empty() {
            return Err(AppError::MissingPlayerId);
        }

        let mut stats = Vec::with_capacity(self.config.gacha_types.len());
        for gacha_type in &self.config.gacha_types {
            let records = self.store.get_gacha_records(player_id, &gacha_type.key)?;
            let gacha_type = self.localized_gacha_type(gacha_type);
            stats.push(self.calc.calc(&records, &gacha_type));
        }

        Ok(StatsResponse {
            success: true,
            player_id: player_id.to_string(),
            stats,
        })
    }

    pub async fn track_url(&self, url: impl AsRef<str>) -> Result<StatsResponse, AppError> {
        let fetch_result = self.fetch_and_save(url.as_ref()).await?;
        self.get_stats(fetch_result.payload.player_id)
    }

    pub async fn fetch_and_save(&self, target_url: &str) -> Result<FetchResult, AppError> {
        let target_url = target_url.trim().replace('\\', "");
        if target_url.is_empty() {
            return Err(AppError::MissingUrl);
        }
        let payload = self.tracker.parse_payload_from_url(&target_url)?;
        let fetch_result = self
            .tracker
            .fetch_all_records(payload, &self.config.gacha_types)
            .await?;
        if fetch_result.records.is_empty() {
            return Err(AppError::InvalidGachaUrl);
        }
        self.save_fetch_result(fetch_result.clone())?;
        Ok(fetch_result)
    }

    pub fn export_report(
        &self,
        player_id: &str,
        format: ReportFormat,
        lang: &str,
    ) -> Result<Vec<u8>, AppError> {
        let stats = self.get_stats(player_id)?;
        if stats.stats.is_empty() {
            return Err(AppError::NoValidRecords);
        }
        reporter::export(
            &self.config,
            &ReportData {
                player_id: stats.player_id,
                stats: stats.stats,
            },
            format,
            lang,
        )
    }

    pub fn backup(&self, output: impl AsRef<Path>) -> Result<(), AppError> {
        self.store.export_backup(output.as_ref())
    }

    pub fn merge_backup(&self, input: impl AsRef<Path>) -> Result<(), AppError> {
        self.store.merge_backup(input.as_ref())
    }

    pub fn load_fetch_result_file(&self, path: impl AsRef<Path>) -> Result<FetchResult, AppError> {
        let path = path.as_ref();
        let bytes = fs::read(path)?;
        if let Ok(fetch_result) = serde_json::from_slice::<FetchResult>(&bytes) {
            if !fetch_result.records.is_empty() {
                return Ok(fetch_result);
            }
        }
        let records =
            serde_json::from_slice::<BTreeMap<String, Vec<crate::types::Record>>>(&bytes)?;
        let player_id = path
            .file_stem()
            .and_then(|name| name.to_str())
            .unwrap_or("offline")
            .split_once('-')
            .map(|(id, _)| id)
            .unwrap_or_else(|| {
                path.file_stem()
                    .and_then(|name| name.to_str())
                    .unwrap_or("offline")
            })
            .to_string();
        Ok(FetchResult {
            payload: crate::types::Payload {
                player_id,
                ..Default::default()
            },
            records,
        })
    }

    fn localized_gacha_type(&self, gacha_type: &GachaType) -> GachaType {
        let mut item = gacha_type.clone();
        if let Ok(locale) = tracker::load_local_gacha_locale(&self.config.language) {
            if let Some(name) = locale.select_list.get(&item.key) {
                item.name = name.clone();
            }
        }
        item
    }
}
