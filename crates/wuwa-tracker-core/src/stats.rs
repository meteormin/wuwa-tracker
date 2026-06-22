use crate::config::Config;
use wuwa_tracker_types::{FiveStarRecord, GachaType, Record, Stats};

#[derive(Debug, Clone)]
pub struct StatsCalculator {
    standard_five_star_resources: Vec<i32>,
    astrite_per_pull: usize,
}

impl StatsCalculator {
    pub fn new(config: &Config) -> Self {
        Self {
            standard_five_star_resources: config.standard_five_star_resources.clone(),
            astrite_per_pull: config.astrite_per_pull,
        }
    }

    pub fn calc(&self, records: &[Record], gacha_type: &GachaType) -> Stats {
        let mut stats = Stats {
            gacha_type: gacha_type.id,
            gacha_name: gacha_type.name.clone(),
            total_pulls: records.len(),
            total_astrite: records.len() * self.astrite_per_pull,
            current_pity5: 0,
            current_pity4: 0,
            base_rate: gacha_type.base_rate,
            expected_pulls: gacha_type.expected_pulls,
            five_stars: Vec::new(),
            records: Vec::new(),
            avg_pulls: 0.0,
            actual_rate: 0.0,
            luck_score: 0.0,
            has_five_star: false,
        };

        let mut pity5 = 0;
        let mut pity4 = 0;

        for record in records.iter().rev() {
            pity5 += 1;
            pity4 += 1;
            stats.records.push(record.clone());

            match record.quality_level {
                5 => {
                    let is_pick_up = !gacha_type.has_off_banner_drop
                        || !self
                            .standard_five_star_resources
                            .contains(&record.resource_id);
                    stats.five_stars.push(FiveStarRecord {
                        name: record.name.clone(),
                        time: record.time.clone(),
                        pity: pity5,
                        is_pick_up,
                    });
                    pity5 = 0;
                }
                4 => pity4 = 0,
                _ => {}
            }
        }

        stats.current_pity5 = pity5;
        stats.current_pity4 = pity4;

        let five_star_count = stats.five_stars.len();
        if five_star_count > 0 {
            stats.has_five_star = true;
            let sum_pity: i32 = stats.five_stars.iter().map(|item| item.pity).sum();
            stats.avg_pulls = sum_pity as f64 / five_star_count as f64;
            if stats.total_pulls > 0 {
                stats.actual_rate = five_star_count as f64 / stats.total_pulls as f64 * 100.0;
            }

            let mut expected_total = 0;
            let mut actual_total = 0;
            let mut current_cycle_pulls = 0;

            for five_star in &stats.five_stars {
                current_cycle_pulls += five_star.pity;
                if !gacha_type.has_off_banner_drop || five_star.is_pick_up {
                    expected_total += gacha_type.expected_pulls;
                    actual_total += current_cycle_pulls;
                    current_cycle_pulls = 0;
                }
            }

            if current_cycle_pulls > 0 {
                expected_total += gacha_type.expected_pulls;
                actual_total += current_cycle_pulls;
            }

            if actual_total > 0 {
                stats.luck_score = expected_total as f64 / actual_total as f64 * 100.0;
            }
        }

        stats.five_stars.reverse();
        stats.records.reverse();
        stats
    }
}
