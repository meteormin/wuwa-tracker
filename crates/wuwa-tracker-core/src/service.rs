use crate::{
    config::Config,
    error::AppError,
    logger::AppLogger,
    reporter, scanner,
    stats::StatsCalculator,
    store::JsonStore,
    tracker::{self, TrackerClient},
    types::{
        FetchResult, GachaType, LocaleData, ReportData, ReportFormat, ScanResponse, StatsResponse,
    },
};
use serde_json::{json, Value};
use std::{
    collections::BTreeMap,
    fs,
    path::Path,
    sync::{Arc, RwLock},
};

#[derive(Clone)]
pub struct Service {
    config: Arc<Config>,
    store: Arc<JsonStore>,
    logger: Arc<AppLogger>,
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
        let logger = Arc::new(AppLogger::new(config.log_path.clone()));
        let tracker = TrackerClient::new(config.resources_url.clone(), config.tracking_url.clone());
        let service = Self {
            config: Arc::new(config),
            store,
            logger,
            calc,
            tracker,
            locale: Arc::new(RwLock::new(None)),
        };
        service.log_info(
            "service_initialized",
            &[
                (
                    "db_path",
                    json!(service.config.db_path.display().to_string()),
                ),
                (
                    "log_path",
                    json!(service.config.log_path.display().to_string()),
                ),
            ],
        );
        Ok(service)
    }

    pub fn config(&self) -> &Config {
        &self.config
    }

    pub fn log_event(&self, level: &str, event: &str, fields: &[(&str, Value)]) {
        let _ = self.logger.log(level, event, fields);
    }

    pub fn list_players(&self) -> Vec<String> {
        let players = self.store.list_players();
        self.log_info("players_listed", &[("players", json!(players.len()))]);
        players
    }

    pub fn store_stats(&self) -> Result<crate::store::StoreStats, AppError> {
        let result = self.store.stats();
        match &result {
            Ok(stats) => self.log_info(
                "store_stats_loaded",
                &[
                    ("players", json!(stats.players)),
                    ("banners", json!(stats.banners)),
                    ("records", json!(stats.records)),
                    ("size_bytes", json!(stats.size_bytes)),
                ],
            ),
            Err(error) => self.log_error("store_stats_failed", error),
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
            Ok(counts) => self.log_info(
                "banner_record_counts_loaded",
                &[
                    ("player_id", json!(player_id)),
                    ("banners", json!(counts.len())),
                ],
            ),
            Err(error) => self.log_error_with_fields(
                "banner_record_counts_failed",
                error,
                &[("player_id", json!(player_id))],
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
            Ok(_) => self.log_info(
                "scan_completed",
                &[("path", json!(path.display().to_string()))],
            ),
            Err(error) => self.log_error_with_fields(
                "scan_failed",
                error,
                &[("path", json!(path.display().to_string()))],
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
                self.log_info("locale_prepared", &[("lang", json!(lang))]);
            }
            Err(error) => self.log_error_with_fields(
                "locale_prepare_failed",
                &error,
                &[("lang", json!(lang))],
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
            Ok(_) => self.log_info(
                "upload_completed",
                &[
                    ("player_id", json!(player_id)),
                    ("records", json!(total_records)),
                ],
            ),
            Err(error) => self.log_error_with_fields(
                "upload_failed",
                error,
                &[
                    ("player_id", json!(player_id)),
                    ("records", json!(total_records)),
                ],
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
        self.log_info(
            "fetch_result_saved",
            &[
                ("player_id", json!(player_id)),
                ("records", json!(count_records(&fetch_result.records))),
            ],
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
        self.log_info(
            "stats_loaded",
            &[
                ("player_id", json!(player_id)),
                ("banners", json!(response.stats.len())),
            ],
        );
        Ok(response)
    }

    pub async fn track_url(&self, url: impl AsRef<str>) -> Result<StatsResponse, AppError> {
        let result = match self.fetch_and_save(url.as_ref()).await {
            Ok(fetch_result) => self.get_stats(fetch_result.payload.player_id),
            Err(error) => Err(error),
        };
        match &result {
            Ok(response) => self.log_info(
                "track_url_completed",
                &[("player_id", json!(response.player_id))],
            ),
            Err(error) => self.log_error("track_url_failed", error),
        }
        result
    }

    pub async fn fetch_and_save(&self, target_url: &str) -> Result<FetchResult, AppError> {
        self.log_info("fetch_started", &[]);
        let target_url = target_url.trim().replace('\\', "");
        if target_url.is_empty() {
            self.log_error("fetch_failed", &AppError::MissingUrl);
            return Err(AppError::MissingUrl);
        }
        let payload = match self.tracker.parse_payload_from_url(&target_url) {
            Ok(payload) => payload,
            Err(error) => {
                self.log_error("fetch_failed", &error);
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
                self.log_error("fetch_failed", &error);
                return Err(error);
            }
        };
        if fetch_result.records.is_empty() {
            self.log_error("fetch_failed", &AppError::InvalidGachaUrl);
            return Err(AppError::InvalidGachaUrl);
        }
        if let Err(error) = self.save_fetch_result(fetch_result.clone()) {
            self.log_error("fetch_failed", &error);
            return Err(error);
        }
        self.log_info(
            "fetch_completed",
            &[
                ("player_id", json!(fetch_result.payload.player_id)),
                ("records", json!(count_records(&fetch_result.records))),
            ],
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
            Ok(content) => self.log_info(
                "report_exported",
                &[
                    ("player_id", json!(player_id)),
                    ("format", json!(format.extension())),
                    ("bytes", json!(content.len())),
                ],
            ),
            Err(error) => self.log_error_with_fields(
                "report_export_failed",
                error,
                &[
                    ("player_id", json!(player_id)),
                    ("format", json!(format.extension())),
                ],
            ),
        }
        result
    }

    pub fn export_backup(&self) -> Result<Vec<u8>, AppError> {
        let result = self.store.export_backup();
        match &result {
            Ok(content) => self.log_info("backup_exported", &[("bytes", json!(content.len()))]),
            Err(error) => self.log_error("backup_export_failed", error),
        }
        result
    }

    pub fn merge_backup(&self, input: impl AsRef<Path>) -> Result<(), AppError> {
        let input = input.as_ref();
        let result = self.store.merge_backup(input);
        match &result {
            Ok(_) => self.log_info(
                "backup_merged",
                &[("path", json!(input.display().to_string()))],
            ),
            Err(error) => self.log_error_with_fields(
                "backup_merge_failed",
                error,
                &[("path", json!(input.display().to_string()))],
            ),
        }
        result
    }

    pub fn load_fetch_result_file(&self, path: impl AsRef<Path>) -> Result<FetchResult, AppError> {
        let path = path.as_ref();
        let bytes = fs::read(path)?;
        if let Ok(fetch_result) = serde_json::from_slice::<FetchResult>(&bytes) {
            if !fetch_result.records.is_empty() {
                self.log_info(
                    "fetch_result_file_loaded",
                    &[
                        ("path", json!(path.display().to_string())),
                        ("records", json!(count_records(&fetch_result.records))),
                    ],
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
        self.log_info(
            "legacy_fetch_result_file_loaded",
            &[
                ("path", json!(path.display().to_string())),
                ("records", json!(count_records(&fetch_result.records))),
            ],
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

    fn log_info(&self, event: &str, fields: &[(&str, Value)]) {
        self.log_event("info", event, fields);
    }

    fn log_error(&self, event: &str, error: &dyn std::error::Error) {
        self.log_error_with_fields(event, error, &[]);
    }

    fn log_error_with_fields(
        &self,
        event: &str,
        error: &dyn std::error::Error,
        fields: &[(&str, Value)],
    ) {
        let mut values = Vec::with_capacity(fields.len() + 1);
        values.push(("error", json!(error.to_string())));
        values.extend_from_slice(fields);
        self.log_event("error", event, &values);
    }
}

fn count_records(records: &BTreeMap<String, Vec<crate::types::Record>>) -> usize {
    records.values().map(Vec::len).sum()
}
