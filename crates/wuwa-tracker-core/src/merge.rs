use crate::types::Record;
use std::collections::{BTreeMap, BTreeSet, HashMap};

pub fn merge_records(db_records: &[Record], new_records: &[Record]) -> Vec<Record> {
    if db_records.is_empty() {
        return new_records.to_vec();
    }
    if new_records.is_empty() {
        return db_records.to_vec();
    }

    for k in 0..new_records.len() {
        let suffix_len = new_records.len() - k;
        if suffix_len > db_records.len() {
            continue;
        }

        let matches = (0..suffix_len).all(|i| {
            let incoming = &new_records[k + i];
            let existing = &db_records[i];
            incoming.resource_id == existing.resource_id && incoming.time == existing.time
        });

        if matches {
            let mut merged = Vec::with_capacity(k + db_records.len());
            merged.extend_from_slice(&new_records[..k]);
            merged.extend_from_slice(db_records);
            return merged;
        }
    }

    let newest_db_time = &db_records[0].time;
    let oldest_new_time = &new_records[new_records.len() - 1].time;
    if oldest_new_time >= newest_db_time {
        let mut merged = Vec::with_capacity(new_records.len() + db_records.len());
        merged.extend_from_slice(new_records);
        merged.extend_from_slice(db_records);
        return merged;
    }

    let newest_new_time = &new_records[0].time;
    let oldest_db_time = &db_records[db_records.len() - 1].time;
    if newest_new_time <= oldest_db_time {
        let mut merged = Vec::with_capacity(new_records.len() + db_records.len());
        merged.extend_from_slice(db_records);
        merged.extend_from_slice(new_records);
        return merged;
    }

    union_merge(db_records, new_records)
}

fn union_merge(db_records: &[Record], new_records: &[Record]) -> Vec<Record> {
    let mut groups: BTreeMap<String, Vec<Record>> = BTreeMap::new();
    let mut all_times = BTreeSet::new();

    for record in db_records {
        groups
            .entry(record.time.clone())
            .or_default()
            .push(record.clone());
        all_times.insert(record.time.clone());
    }

    for record in new_records {
        if !all_times.contains(&record.time) {
            groups
                .entry(record.time.clone())
                .or_default()
                .push(record.clone());
            all_times.insert(record.time.clone());
        }
    }

    let new_times: BTreeSet<&str> = new_records
        .iter()
        .map(|record| record.time.as_str())
        .collect();
    for (time, items) in groups.iter_mut() {
        if !new_times.contains(time.as_str()) {
            continue;
        }

        let db_items: Vec<&Record> = db_records
            .iter()
            .filter(|record| record.time == *time)
            .collect();
        let incoming_items: Vec<&Record> = new_records
            .iter()
            .filter(|record| record.time == *time)
            .collect();
        if db_items.is_empty() || incoming_items.is_empty() {
            continue;
        }

        let mut db_freq = HashMap::new();
        let mut db_templates = HashMap::new();
        for record in db_items {
            *db_freq.entry(record.resource_id).or_insert(0usize) += 1;
            db_templates.insert(record.resource_id, record.clone());
        }

        let mut incoming_freq = HashMap::new();
        let mut incoming_templates = HashMap::new();
        for record in incoming_items {
            *incoming_freq.entry(record.resource_id).or_insert(0usize) += 1;
            incoming_templates.insert(record.resource_id, record.clone());
        }

        let resource_ids: BTreeSet<i32> = db_freq
            .keys()
            .chain(incoming_freq.keys())
            .copied()
            .collect();
        let mut merged_items = Vec::new();
        for resource_id in resource_ids {
            let count = db_freq
                .get(&resource_id)
                .copied()
                .unwrap_or_default()
                .max(incoming_freq.get(&resource_id).copied().unwrap_or_default());
            let template = incoming_templates
                .get(&resource_id)
                .or_else(|| db_templates.get(&resource_id))
                .expect("template exists");
            for _ in 0..count {
                merged_items.push((*template).clone());
            }
        }
        *items = merged_items;
    }

    let mut merged: Vec<Record> = groups.into_values().flatten().collect();
    merged.sort_by(|left, right| {
        right
            .time
            .cmp(&left.time)
            .then(right.quality_level.cmp(&left.quality_level))
            .then(left.resource_id.cmp(&right.resource_id))
    });
    merged
}
