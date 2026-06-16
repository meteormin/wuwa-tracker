use crate::{error::AppError, merge::merge_records, types::Record};
use serde::{Deserialize, Serialize};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::PathBuf,
    sync::Mutex,
};

#[derive(Debug)]
pub struct JsonStore {
    path: PathBuf,
    data: Mutex<StoreData>,
}

#[derive(Debug, Clone)]
pub struct StoreStats {
    pub path: PathBuf,
    pub exists: bool,
    pub size_bytes: u64,
    pub players: usize,
    pub banners: usize,
    pub records: usize,
}

#[derive(Debug, Default, Serialize, Deserialize)]
pub struct StoreData {
    players: BTreeMap<String, BTreeMap<String, Vec<Record>>>,
}

impl JsonStore {
    pub fn new(path: PathBuf) -> Result<Self, AppError> {
        let data = if path.exists() {
            let bytes = fs::read(&path)?;
            serde_json::from_slice(&bytes)?
        } else {
            StoreData::default()
        };

        Ok(Self {
            path,
            data: Mutex::new(data),
        })
    }

    pub fn save_gacha_records(
        &self,
        player_id: &str,
        card_pool_type: &str,
        records: &[Record],
    ) -> Result<(), AppError> {
        let mut data = self.data.lock().expect("store lock poisoned");
        let player = data.players.entry(player_id.to_string()).or_default();
        let existing = player.entry(card_pool_type.to_string()).or_default();
        *existing = merge_records(existing, records);
        self.flush(&data)
    }

    pub fn get_gacha_records(
        &self,
        player_id: &str,
        card_pool_type: &str,
    ) -> Result<Vec<Record>, AppError> {
        let data = self.data.lock().expect("store lock poisoned");
        Ok(data
            .players
            .get(player_id)
            .and_then(|player| player.get(card_pool_type))
            .cloned()
            .unwrap_or_default())
    }

    pub fn list_players(&self) -> Vec<String> {
        let data = self.data.lock().expect("store lock poisoned");
        data.players
            .keys()
            .cloned()
            .collect::<BTreeSet<_>>()
            .into_iter()
            .collect()
    }

    pub fn has_player(&self, player_id: &str) -> bool {
        let data = self.data.lock().expect("store lock poisoned");
        data.players.contains_key(player_id)
    }

    pub fn stats(&self) -> Result<StoreStats, AppError> {
        let data = self.data.lock().expect("store lock poisoned");
        let metadata = fs::metadata(&self.path).ok();
        let banners = data.players.values().map(BTreeMap::len).sum();
        let records = data
            .players
            .values()
            .flat_map(BTreeMap::values)
            .map(Vec::len)
            .sum();

        Ok(StoreStats {
            path: self.path.clone(),
            exists: metadata.is_some(),
            size_bytes: metadata.map(|value| value.len()).unwrap_or_default(),
            players: data.players.len(),
            banners,
            records,
        })
    }

    pub fn export_backup(&self) -> Result<Vec<u8>, AppError> {
        let data = self.data.lock().expect("store lock poisoned");
        Ok(serde_json::to_vec_pretty(&*data)?)
    }

    pub fn merge_backup(&self, path: &std::path::Path) -> Result<(), AppError> {
        let bytes = fs::read(path)?;
        let incoming: StoreData = serde_json::from_slice(&bytes)?;
        let mut data = self.data.lock().expect("store lock poisoned");
        for (player_id, pools) in incoming.players {
            let player = data.players.entry(player_id).or_default();
            for (pool, records) in pools {
                let existing = player.entry(pool).or_default();
                *existing = merge_records(existing, &records);
            }
        }
        self.flush(&data)
    }

    fn flush(&self, data: &StoreData) -> Result<(), AppError> {
        if let Some(parent) = self.path.parent() {
            fs::create_dir_all(parent)?;
        }
        let bytes = serde_json::to_vec_pretty(data)?;
        fs::write(&self.path, bytes)?;
        Ok(())
    }
}
