use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Payload {
    pub player_id: String,
    pub server_id: String,
    pub language_code: String,
    pub record_id: String,
    pub card_pool_id: String,
    pub card_pool_type: i32,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct FetchResult {
    pub payload: Payload,
    pub records: BTreeMap<String, Vec<Record>>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GachaResponse {
    pub code: i32,
    pub message: String,
    pub data: Vec<Record>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LocaleData {
    pub select_list: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Record {
    pub card_pool_type: String,
    pub resource_id: i32,
    pub quality_level: i32,
    pub resource_type: String,
    pub name: String,
    pub count: i32,
    pub time: String,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct FiveStarRecord {
    pub name: String,
    pub time: String,
    pub pity: i32,
    pub is_pick_up: bool,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GachaType {
    pub id: i32,
    pub key: String,
    pub has_off_banner_drop: bool,
    pub name: String,
    pub base_rate: f64,
    pub expected_pulls: i32,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Stats {
    pub gacha_type: i32,
    pub gacha_name: String,
    pub total_pulls: usize,
    pub total_astrite: usize,
    pub current_pity5: i32,
    pub current_pity4: i32,
    pub base_rate: f64,
    pub expected_pulls: i32,
    pub five_stars: Vec<FiveStarRecord>,
    pub records: Vec<Record>,
    pub avg_pulls: f64,
    pub actual_rate: f64,
    pub luck_score: f64,
    pub has_five_star: bool,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LuckScoreThreshold {
    pub min_score: f64,
    pub state: String,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StatsResponse {
    pub success: bool,
    #[serde(default)]
    pub player_id: String,
    #[serde(default)]
    pub stats: Vec<Stats>,
    pub error: Option<String>,
    pub error_key: Option<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ScanResponse {
    pub success: bool,
    #[serde(default)]
    pub url: String,
    pub error: Option<String>,
    pub error_key: Option<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ErrorResponse {
    pub success: bool,
    pub error: String,
    pub error_key: String,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ConfigResponse {
    pub success: bool,
    #[serde(default)]
    pub luck_score_thresholds: Vec<LuckScoreThreshold>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PlayersResponse {
    pub success: bool,
    #[serde(default)]
    pub players: Vec<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ExportResponse {
    pub success: bool,
    pub filename: String,
    pub content_type: String,
    pub content: Vec<u8>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ReportData {
    pub player_id: String,
    pub stats: Vec<Stats>,
}
