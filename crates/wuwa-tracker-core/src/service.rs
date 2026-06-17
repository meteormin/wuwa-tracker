use crate::{
    config::Config,
    error::AppError,
    reporter, scanner,
    stats::StatsCalculator,
    store::JsonStore,
    tracker::{self, TrackerClient},
    types::{
        FetchResult, GachaType, LocaleData, ReportData, ReportFormat, ScanResponse, StatsResponse,
    },
};
use std::{
    collections::BTreeMap,
    fs,
    path::Path,
    sync::{Arc, RwLock},
};
use tracing::{error, info};

#[derive(Clone)]
pub struct Service {
    config: Arc<Config>,
    store: Arc<JsonStore>,
    calc: StatsCalculator,
    tracker: TrackerClient,
    locale: Arc<RwLock<Option<LocaleData>>>,
}

#[derive(Debug, Clone)]
pub struct BannerRecordCount {
    pub id: i32,
    pub key: String,
    pub name: String,
    pub records: usize,
}

impl Service {
    pub fn new(config: Config) -> Result<Self, AppError> {
        let calc = StatsCalculator::new(&config);
        let store = Arc::new(JsonStore::new(config.db_path.clone())?);
        let tracker = TrackerClient::new(config.resources_url.clone(), config.tracking_url.clone());
        let service = Self {
            config: Arc::new(config),
            store,
            calc,
            tracker,
            locale: Arc::new(RwLock::new(None)),
        };
        info!(
            event = "service_initialized",
            db_path = %service.config.db_path.display(),
            log_path = %service.config.log_path.display(),
        );
        Ok(service)
    }

    pub fn config(&self) -> &Config {
        &self.config
    }

    pub fn list_players(&self) -> Vec<String> {
        let players = self.store.list_players();
        info!(event = "players_listed", players = players.len());
        players
    }

    pub fn store_stats(&self) -> Result<crate::store::StoreStats, AppError> {
        let result = self.store.stats();
        match &result {
            Ok(stats) => info!(
                event = "store_stats_loaded",
                players = stats.players,
                banners = stats.banners,
                records = stats.records,
                size_bytes = stats.size_bytes,
            ),
            Err(error) => error!(event = "store_stats_failed", error = %error),
        }
        result
    }

    pub fn banner_record_counts(
        &self,
        player_id: impl AsRef<str>,
    ) -> Result<Vec<BannerRecordCount>, AppError> {
        let player_id = player_id.as_ref().trim();
        if player_id.is_empty() {
            return Err(AppError::MissingPlayerId);
        }
        if !self.store.has_player(player_id) {
            return Err(AppError::PlayerNotFound);
        }

        let result: Result<Vec<BannerRecordCount>, AppError> = self
            .config
            .gacha_types
            .iter()
            .map(|gacha_type| {
                let records = self.store.get_gacha_records(player_id, &gacha_type.key)?;
                let gacha_type = self.localized_gacha_type(gacha_type);
                Ok(BannerRecordCount {
                    id: gacha_type.id,
                    key: gacha_type.key,
                    name: gacha_type.name,
                    records: records.len(),
                })
            })
            .collect();
        match &result {
            Ok(counts) => info!(
                event = "banner_record_counts_loaded",
                player_id = %player_id,
                banners = counts.len(),
            ),
            Err(error) => error!(
                event = "banner_record_counts_failed",
                error = %error,
                player_id = %player_id,
            ),
        }
        result
    }

    pub fn scan(&self, path: impl AsRef<Path>) -> Result<ScanResponse, AppError> {
        let path = path.as_ref();
        let result = scanner::scan_url(
            path,
            &self.config.scan_log_paths,
            &self.config.resources_url,
        )
        .map(|url| ScanResponse { success: true, url });
        match &result {
            Ok(_) => info!(
                event = "scan_completed",
                path = %path.display(),
            ),
            Err(error) => error!(
                event = "scan_failed",
                error = %error,
                path = %path.display(),
            ),
        }
        result
    }

    pub async fn prepare_locale(&self, lang: &str) {
        let lang = if lang.trim().is_empty() {
            self.config.language.as_str()
        } else {
            lang.trim()
        };
        let locale = self
            .tracker
            .fetch_gacha_locale(lang)
            .await
            .or_else(|_| tracker::load_local_gacha_locale(lang))
            .or_else(|_| tracker::load_local_gacha_locale("ko"));

        match locale {
            Ok(locale) => {
                *self.locale.write().expect("locale lock poisoned") = Some(locale);
                info!(event = "locale_prepared", lang = %lang);
            }
            Err(error) => error!(
                event = "locale_prepare_failed",
                error = %error,
                lang = %lang,
            ),
        }
    }

    pub fn upload(&self, fetch_result: FetchResult) -> Result<StatsResponse, AppError> {
        let player_id = fetch_result.payload.player_id.trim().to_string();
        let total_records = count_records(&fetch_result.records);
        let result = self
            .save_fetch_result(fetch_result)
            .and_then(|_| self.get_stats(&player_id));
        match &result {
            Ok(_) => info!(
                event = "upload_completed",
                player_id = %player_id,
                records = total_records,
            ),
            Err(error) => error!(
                event = "upload_failed",
                error = %error,
                player_id = %player_id,
                records = total_records,
            ),
        }
        result
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
        info!(
            event = "fetch_result_saved",
            player_id = %player_id,
            records = count_records(&fetch_result.records),
        );
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

        let response = StatsResponse {
            success: true,
            player_id: player_id.to_string(),
            stats,
        };
        info!(
            event = "stats_loaded",
            player_id = %player_id,
            banners = response.stats.len(),
        );
        Ok(response)
    }

    pub async fn track_url(&self, url: impl AsRef<str>) -> Result<StatsResponse, AppError> {
        let result = match self.fetch_and_save(url.as_ref()).await {
            Ok(fetch_result) => self.get_stats(fetch_result.payload.player_id),
            Err(error) => Err(error),
        };
        match &result {
            Ok(response) => info!(
                event = "track_url_completed",
                player_id = %response.player_id,
            ),
            Err(error) => error!(event = "track_url_failed", error = %error),
        }
        result
    }

    pub async fn fetch_and_save(&self, target_url: &str) -> Result<FetchResult, AppError> {
        info!(event = "fetch_started");
        let target_url = target_url.trim().replace('\\', "");
        if target_url.is_empty() {
            error!(event = "fetch_failed", error = %AppError::MissingUrl);
            return Err(AppError::MissingUrl);
        }
        let payload = match self.tracker.parse_payload_from_url(&target_url) {
            Ok(payload) => payload,
            Err(error) => {
                error!(event = "fetch_failed", error = %error);
                return Err(error);
            }
        };
        if !payload.language_code.trim().is_empty() {
            self.prepare_locale(&payload.language_code).await;
        }
        let fetch_result = match self
            .tracker
            .fetch_all_records(payload, &self.config.gacha_types)
            .await
        {
            Ok(fetch_result) => fetch_result,
            Err(error) => {
                error!(event = "fetch_failed", error = %error);
                return Err(error);
            }
        };
        if fetch_result.records.is_empty() {
            error!(event = "fetch_empty", error = %AppError::InvalidGachaUrl);
            return Err(AppError::InvalidGachaUrl);
        }
        if let Err(error) = self.save_fetch_result(fetch_result.clone()) {
            error!(event = "fetch_failed", error = %error);
            return Err(error);
        }
        info!(
            event = "fetch_completed",
            player_id = %fetch_result.payload.player_id,
            records = count_records(&fetch_result.records),
        );
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
        let result = reporter::export(
            &self.config,
            &ReportData {
                player_id: stats.player_id,
                stats: stats.stats,
            },
            format,
            lang,
        );
        match &result {
            Ok(content) => info!(
                event = "report_exported",
                player_id = %player_id,
                format = %format.extension(),
                bytes = content.len(),
            ),
            Err(error) => error!(
                event = "report_export_failed",
                error = %error,
                player_id = %player_id,
                format = %format.extension(),
            ),
        }
        result
    }

    pub fn export_backup(&self) -> Result<Vec<u8>, AppError> {
        let result = self.store.export_backup();
        match &result {
            Ok(content) => info!(event = "backup_exported", bytes = content.len()),
            Err(error) => error!(event = "backup_export_failed", error = %error),
        }
        result
    }

    pub fn merge_backup(&self, input: impl AsRef<Path>) -> Result<(), AppError> {
        let input = input.as_ref();
        let result = self.store.merge_backup(input);
        match &result {
            Ok(_) => info!(
                event = "backup_merged",
                path = %input.display(),
            ),
            Err(error) => error!(
                event = "backup_merge_failed",
                error = %error,
                path = %input.display(),
            ),
        }
        result
    }

    pub fn load_fetch_result_file(&self, path: impl AsRef<Path>) -> Result<FetchResult, AppError> {
        let path = path.as_ref();
        let bytes = fs::read(path)?;
        if let Ok(fetch_result) = serde_json::from_slice::<FetchResult>(&bytes) {
            if !fetch_result.records.is_empty() {
                info!(
                    event = "fetch_result_file_loaded",
                    path = %path.display(),
                    records = count_records(&fetch_result.records),
                );
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
        let fetch_result = FetchResult {
            payload: crate::types::Payload {
                player_id,
                ..Default::default()
            },
            records,
        };
        info!(
            event = "legacy_fetch_result_file_loaded",
            path = %path.display(),
            records = count_records(&fetch_result.records),
        );
        Ok(fetch_result)
    }

    fn localized_gacha_type(&self, gacha_type: &GachaType) -> GachaType {
        let mut item = gacha_type.clone();
        if let Some(name) = self
            .locale
            .read()
            .expect("locale lock poisoned")
            .as_ref()
            .and_then(|locale| locale.select_list.get(&item.key))
            .cloned()
        {
            item.name = name;
            return item;
        }
        if let Ok(locale) = tracker::load_local_gacha_locale(&self.config.language) {
            if let Some(name) = locale.select_list.get(&item.key).cloned() {
                item.name = name;
            }
        }
        item
    }
}

fn count_records(records: &BTreeMap<String, Vec<crate::types::Record>>) -> usize {
    records.values().map(Vec::len).sum()
}
