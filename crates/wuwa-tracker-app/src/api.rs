use serde::Serialize;
use std::str::FromStr;
use tauri::State;
use wuwa_tracker_core::{
    translations,
    types::ReportFormat,
    types::{FetchResult, ScanResponse, StatsResponse},
    AppError, Service,
};

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ConfigResponse {
    success: bool,
    luck_score_thresholds: Vec<wuwa_tracker_core::types::LuckScoreThreshold>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PlayersResponse {
    success: bool,
    players: Vec<String>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ExportResponse {
    success: bool,
    filename: String,
    content_type: String,
    content: Vec<u8>,
}

#[tauri::command]
pub fn get_config(service: State<'_, Service>) -> ConfigResponse {
    ConfigResponse {
        success: true,
        luck_score_thresholds: service.config().luck_score_thresholds.clone(),
    }
}

#[tauri::command]
pub fn list_players(service: State<'_, Service>) -> PlayersResponse {
    PlayersResponse {
        success: true,
        players: service.list_players(),
    }
}

#[tauri::command]
pub fn get_stats(
    player_id: String,
    service: State<'_, Service>,
) -> Result<StatsResponse, AppError> {
    service.get_stats(player_id)
}

#[tauri::command]
pub fn scan_url(path: String, service: State<'_, Service>) -> Result<ScanResponse, AppError> {
    service.scan(path)
}

#[tauri::command]
pub async fn track_url(
    url: String,
    service: State<'_, Service>,
) -> Result<StatsResponse, AppError> {
    service.track_url(url).await
}

#[tauri::command]
pub fn upload_json(
    data: serde_json::Value,
    service: State<'_, Service>,
) -> Result<StatsResponse, AppError> {
    let fetch_result: FetchResult = serde_json::from_value(data)?;
    service.upload(fetch_result)
}

#[tauri::command]
pub fn get_i18n(lang: String) -> Result<translations::TranslationResponse, AppError> {
    translations::load(&lang)
}

#[tauri::command]
pub fn export_report(
    player_id: String,
    format: String,
    lang: String,
    service: State<'_, Service>,
) -> Result<ExportResponse, AppError> {
    let format = ReportFormat::from_str(&format)?;
    let content = service.export_report(&player_id, format, &lang)?;
    Ok(ExportResponse {
        success: true,
        filename: format!("report_{}.{}", player_id, format.extension()),
        content_type: format.content_type().to_string(),
        content,
    })
}

#[tauri::command]
pub fn export_backup(service: State<'_, Service>) -> Result<ExportResponse, AppError> {
    let content = service.export_backup()?;
    Ok(ExportResponse {
        success: true,
        filename: "wuwa-tracker.backup.json".to_string(),
        content_type: "application/json; charset=utf-8".to_string(),
        content,
    })
}
