use crate::{error::AppError, merge::merge_records};
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, fs, path::PathBuf, sync::Mutex};
use tracing::debug;
use wuwa_tracker_types::Record;

#[derive(Debug)]
/// 플레이어별 기록을 하나의 JSON 파일에 보관하는 프로세스 내 저장소입니다.
///
/// 메모리 상태는 mutex로 보호하지만 여러 프로세스의 동시 쓰기는 지원하지 않습니다.
pub struct JsonStore {
    path: PathBuf,
    data: Mutex<StoreData>,
}

#[derive(Debug, Clone)]
/// 현재 저장소 파일과 메모리 데이터의 요약입니다.
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
    /// 기존 JSON 파일을 읽거나 파일이 없으면 빈 저장소를 생성합니다.
    ///
    /// # Errors
    ///
    /// 파일 읽기 또는 JSON 역직렬화에 실패하면 [`AppError`]를 반환합니다.
    pub fn new(path: PathBuf) -> Result<Self, AppError> {
        let data = if path.exists() {
            let bytes = fs::read(&path)?;
            debug!(event = "store_file_loaded", path = %path.display(), bytes = bytes.len());
            serde_json::from_slice(&bytes)?
        } else {
            debug!(event = "store_file_missing", path = %path.display());
            StoreData::default()
        };

        Ok(Self {
            path,
            data: Mutex::new(data),
        })
    }

    /// 기존 기록과 새 기록을 최신순으로 병합한 뒤 전체 저장소 파일을 갱신합니다.
    ///
    /// # Errors
    ///
    /// 저장소 파일을 직렬화하거나 쓰지 못하면 [`AppError`]를 반환합니다.
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

    /// 플레이어나 배너가 없으면 오류 대신 빈 목록을 반환합니다.
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
        data.players.keys().cloned().collect()
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

    /// 현재 저장소를 복원 가능한 JSON 바이트로 직렬화합니다.
    pub fn export_backup(&self) -> Result<Vec<u8>, AppError> {
        let data = self.data.lock().expect("store lock poisoned");
        Ok(serde_json::to_vec_pretty(&*data)?)
    }

    /// 백업 파일을 현재 데이터에 병합하고 전체 저장소 파일을 갱신합니다.
    ///
    /// # Errors
    ///
    /// 백업 읽기, JSON 역직렬화 또는 저장소 쓰기에 실패하면 [`AppError`]를 반환합니다.
    pub fn merge_backup(&self, path: &std::path::Path) -> Result<(), AppError> {
        let bytes = fs::read(path)?;
        debug!(event = "store_backup_loaded", path = %path.display(), bytes = bytes.len());
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
        debug!(event = "store_file_writing", path = %self.path.display(), bytes = bytes.len());
        fs::write(&self.path, bytes)?;
        Ok(())
    }
}
