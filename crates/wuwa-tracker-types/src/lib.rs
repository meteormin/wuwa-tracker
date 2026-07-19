use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

pub const CHARACTER_BANNER_TYPES: [i32; 6] = [1, 3, 5, 6, 8, 10];

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
    #[serde(default)]
    pub character: String,
    #[serde(default)]
    pub weapon: String,
    #[serde(default)]
    pub item: String,
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

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct CharacterSummary {
    pub resource_id: i32,
    pub name: String,
    pub quality_level: i32,
    pub resource_type: String,
    pub copies: usize,
    pub spent_astrite: usize,
    pub banner_count: usize,
    pub banners: Vec<String>,
    pub last_time: String,
}

#[derive(Default)]
struct CharacterTotal {
    summary: CharacterSummary,
    banners: BTreeMap<i32, String>,
}

pub fn character_summaries(
    stats: &[Stats],
    character_resource_type: &str,
    astrite_per_pull: usize,
) -> Vec<CharacterSummary> {
    if character_resource_type.is_empty() {
        return Vec::new();
    }

    let mut totals: BTreeMap<i32, CharacterTotal> = BTreeMap::new();
    for stat in stats {
        if !CHARACTER_BANNER_TYPES.contains(&stat.gacha_type) {
            continue;
        }

        let mut pity = 0usize;
        for record in stat.records.iter().rev() {
            pity += 1;
            if record.quality_level != 5 {
                continue;
            }

            if record.resource_type == character_resource_type {
                let total = totals.entry(record.resource_id).or_default();
                total.summary.resource_id = record.resource_id;
                total.summary.name = record.name.clone();
                total.summary.quality_level = record.quality_level;
                total.summary.resource_type = record.resource_type.clone();
                total.summary.copies += 1;
                total.summary.spent_astrite += pity * astrite_per_pull;
                total.summary.last_time = record.time.clone();
                total.banners.insert(
                    stat.gacha_type,
                    if stat.gacha_name.is_empty() {
                        stat.gacha_type.to_string()
                    } else {
                        stat.gacha_name.clone()
                    },
                );
            }
            pity = 0;
        }
    }

    let mut summaries: Vec<_> = totals
        .into_values()
        .map(|mut total| {
            total.summary.banner_count = total.banners.len();
            total.summary.banners = total.banners.into_values().collect();
            total.summary
        })
        .collect();
    summaries.sort_by(|left, right| {
        right
            .quality_level
            .cmp(&left.quality_level)
            .then(right.copies.cmp(&left.copies))
            .then(right.last_time.cmp(&left.last_time))
            .then(left.name.cmp(&right.name))
    });
    summaries
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
    #[serde(default)]
    pub resource_types: ResourceTypes,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ResourceTypes {
    #[serde(default)]
    pub character: String,
    #[serde(default)]
    pub weapon: String,
    #[serde(default)]
    pub item: String,
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn character_summaries_count_copies_and_astrite() {
        let stats = vec![Stats {
            gacha_type: 1,
            gacha_name: "Featured Resonator Convene".to_string(),
            records: vec![
                record(100, 5, "Resonator", "Jiyan", "2026-01-04"),
                record(1, 3, "Weapon", "Sword", "2026-01-03"),
                record(2, 3, "Weapon", "Sword", "2026-01-02"),
                record(100, 5, "Resonator", "Jiyan", "2026-01-01"),
            ],
            ..Default::default()
        }];

        let summaries = character_summaries(&stats, "Resonator", 160);

        assert_eq!(summaries.len(), 1);
        assert_eq!(summaries[0].copies, 2);
        assert_eq!(summaries[0].spent_astrite, 640);
        assert_eq!(
            summaries[0].banners,
            vec!["Featured Resonator Convene".to_string()]
        );
    }

    #[test]
    fn character_summaries_ignore_weapons_and_weapon_banners() {
        let stats = vec![
            Stats {
                gacha_type: 1,
                records: vec![record(200, 5, "Weapon", "Sword", "2026-01-01")],
                ..Default::default()
            },
            Stats {
                gacha_type: 2,
                records: vec![record(100, 5, "Resonator", "Jiyan", "2026-01-01")],
                ..Default::default()
            },
        ];

        assert!(character_summaries(&stats, "Resonator", 160).is_empty());
    }

    fn record(
        resource_id: i32,
        quality_level: i32,
        resource_type: &str,
        name: &str,
        time: &str,
    ) -> Record {
        Record {
            resource_id,
            quality_level,
            resource_type: resource_type.to_string(),
            name: name.to_string(),
            time: time.to_string(),
            ..Default::default()
        }
    }
}
