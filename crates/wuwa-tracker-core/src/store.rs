use crate::{error::AppError, merge::merge_records};
use redb::{Database, ReadableDatabase, ReadableTable, TableDefinition};
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, fs, path::PathBuf};
use tracing::debug;
use wuwa_tracker_types::Record;

const RECORDS: TableDefinition<&str, &[u8]> = TableDefinition::new("records");
const KEY_SEPARATOR: char = '\0';

#[derive(Debug)]
/// 플레이어와 배너별 기록을 binary 값으로 보관하는 내장 저장소입니다.
pub struct RedbStore {
    path: PathBuf,
    database: Database,
}

#[derive(Debug, Clone)]
/// 현재 저장소 파일과 데이터의 요약입니다.
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

impl RedbStore {
    /// 기존 데이터베이스를 열거나 파일이 없으면 빈 저장소를 생성합니다.
    pub fn new(path: PathBuf) -> Result<Self, AppError> {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)?;
        }
        let database = Database::create(&path).map_err(database_error)?;
        let write = database.begin_write().map_err(database_error)?;
        write.open_table(RECORDS).map_err(database_error)?;
        write.commit().map_err(database_error)?;
        debug!(event = "store_opened", path = %path.display());
        Ok(Self { path, database })
    }

    /// 기존 기록과 새 기록을 최신순으로 병합해 해당 배너만 갱신합니다.
    pub fn save_gacha_records(
        &self,
        player_id: &str,
        card_pool_type: &str,
        records: &[Record],
    ) -> Result<(), AppError> {
        let key = record_key(player_id, card_pool_type);
        let write = self.database.begin_write().map_err(database_error)?;
        {
            let mut table = write.open_table(RECORDS).map_err(database_error)?;
            let existing = read_records(&table, &key)?;
            let bytes = postcard::to_allocvec(&merge_records(&existing, records))
                .map_err(database_error)?;
            table
                .insert(key.as_str(), bytes.as_slice())
                .map_err(database_error)?;
        }
        write.commit().map_err(database_error)
    }

    /// 플레이어나 배너가 없으면 오류 대신 빈 목록을 반환합니다.
    pub fn get_gacha_records(
        &self,
        player_id: &str,
        card_pool_type: &str,
    ) -> Result<Vec<Record>, AppError> {
        let read = self.database.begin_read().map_err(database_error)?;
        let table = read.open_table(RECORDS).map_err(database_error)?;
        read_records(&table, &record_key(player_id, card_pool_type))
    }

    pub fn list_players(&self) -> Vec<String> {
        self.load_data()
            .map(|data| data.players.into_keys().collect())
            .unwrap_or_default()
    }

    pub fn has_player(&self, player_id: &str) -> bool {
        self.list_players().iter().any(|value| value == player_id)
    }

    pub fn stats(&self) -> Result<StoreStats, AppError> {
        let data = self.load_data()?;
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

    /// 현재 저장소를 복원 가능한 JSON 바이트로 직렬화합니다.
    pub fn export_backup(&self) -> Result<Vec<u8>, AppError> {
        Ok(serde_json::to_vec_pretty(&self.load_data()?)?)
    }

    /// JSON 백업을 현재 데이터에 병합하고 하나의 transaction으로 반영합니다.
    pub fn merge_backup(&self, path: &std::path::Path) -> Result<(), AppError> {
        let bytes = fs::read(path)?;
        debug!(event = "store_backup_loaded", path = %path.display(), bytes = bytes.len());
        let incoming: StoreData = serde_json::from_slice(&bytes)?;
        let write = self.database.begin_write().map_err(database_error)?;
        {
            let mut table = write.open_table(RECORDS).map_err(database_error)?;
            for (player_id, pools) in incoming.players {
                for (pool, records) in pools {
                    let key = record_key(&player_id, &pool);
                    let existing = read_records(&table, &key)?;
                    let bytes = postcard::to_allocvec(&merge_records(&existing, &records))
                        .map_err(database_error)?;
                    table
                        .insert(key.as_str(), bytes.as_slice())
                        .map_err(database_error)?;
                }
            }
        }
        write.commit().map_err(database_error)
    }

    fn load_data(&self) -> Result<StoreData, AppError> {
        let read = self.database.begin_read().map_err(database_error)?;
        let table = read.open_table(RECORDS).map_err(database_error)?;
        let mut data = StoreData::default();
        for entry in table.iter().map_err(database_error)? {
            let (key, value) = entry.map_err(database_error)?;
            let (player_id, pool) = split_record_key(key.value())?;
            let records = postcard::from_bytes(value.value()).map_err(database_error)?;
            data.players
                .entry(player_id.to_string())
                .or_default()
                .insert(pool.to_string(), records);
        }
        Ok(data)
    }
}

fn read_records(
    table: &impl ReadableTable<&'static str, &'static [u8]>,
    key: &str,
) -> Result<Vec<Record>, AppError> {
    match table.get(key).map_err(database_error)? {
        Some(value) => postcard::from_bytes(value.value()).map_err(database_error),
        None => Ok(Vec::new()),
    }
}

fn record_key(player_id: &str, card_pool_type: &str) -> String {
    format!("{player_id}{KEY_SEPARATOR}{card_pool_type}")
}

fn split_record_key(key: &str) -> Result<(&str, &str), AppError> {
    key.split_once(KEY_SEPARATOR)
        .ok_or_else(|| AppError::Database("invalid record key".to_string()))
}

fn database_error(error: impl std::fmt::Display) -> AppError {
    AppError::Database(error.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::{SystemTime, UNIX_EPOCH};

    #[test]
    fn stores_banner_records_and_exports_json_backup() {
        let unique = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let path = std::env::temp_dir().join(format!("wuwa-store-{unique}.redb"));
        let store = RedbStore::new(path.clone()).unwrap();
        let record = Record {
            card_pool_type: "1".to_string(),
            resource_id: 100,
            quality_level: 5,
            resource_type: "Resonator".to_string(),
            name: "Jiyan".to_string(),
            count: 1,
            time: "2026-01-01 00:00:00".to_string(),
        };

        store
            .save_gacha_records("player", "characterEvent", &[record])
            .unwrap();

        let records = store.get_gacha_records("player", "characterEvent").unwrap();
        assert_eq!(records.len(), 1);
        assert_eq!(records[0].resource_id, 100);
        assert_eq!(store.list_players(), vec!["player"]);
        let backup: serde_json::Value =
            serde_json::from_slice(&store.export_backup().unwrap()).unwrap();
        assert_eq!(
            backup["players"]["player"]["characterEvent"][0]["resourceId"],
            100
        );

        drop(store);
        fs::remove_file(path).unwrap();
    }
}
