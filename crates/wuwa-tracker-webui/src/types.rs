use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct FiveStarRecord {
    pub name: String,
    pub time: String,
    pub pity: i32,
    pub is_pick_up: bool,
}

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Record {
    pub quality_level: i32,
    pub resource_type: String,
    pub name: String,
}

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Stats {
    pub gacha_type: i32,
    pub gacha_name: String,
    pub total_pulls: usize,
    pub total_astrite: usize,
    pub current_pity5: i32,
    pub base_rate: f64,
    pub expected_pulls: i32,
    pub avg_pulls: f64,
    pub actual_rate: f64,
    pub luck_score: f64,
    #[serde(default)]
    pub five_stars: Vec<FiveStarRecord>,
    #[serde(default)]
    pub records: Vec<Record>,
    pub has_five_star: bool,
}

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LuckScoreThreshold {
    pub min_score: f64,
    pub state: String,
}

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ConfigResponse {
    pub success: bool,
    #[serde(default)]
    pub luck_score_thresholds: Vec<LuckScoreThreshold>,
}

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PlayersResponse {
    pub success: bool,
    #[serde(default)]
    pub players: Vec<String>,
}

#[derive(Clone, Debug, Default, Deserialize)]
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

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ScanResponse {
    pub success: bool,
    #[serde(default)]
    pub url: String,
    pub error: Option<String>,
    pub error_key: Option<String>,
}

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ExportResponse {
    pub filename: String,
    pub content_type: String,
    pub content: Vec<u8>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PlayerArgs<'a> {
    pub player_id: &'a str,
}

#[derive(Serialize)]
pub struct PathArgs<'a> {
    pub path: &'a str,
}

#[derive(Serialize)]
pub struct UrlArgs<'a> {
    pub url: &'a str,
}

#[derive(Serialize)]
pub struct UploadArgs<'a> {
    pub data: &'a serde_json::Value,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ExportArgs<'a> {
    pub player_id: &'a str,
    pub format: &'a str,
    pub lang: &'a str,
}
