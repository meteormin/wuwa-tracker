use crate::error::AppError;
use serde_json::{Map, Value};
use std::{
    fs::{self, OpenOptions},
    io::Write,
    path::{Path, PathBuf},
    sync::Mutex,
    time::{SystemTime, UNIX_EPOCH},
};

const MAX_LOG_BYTES: u64 = 10 * 1024 * 1024;
const MAX_LOG_FILES: usize = 10;

#[derive(Debug)]
pub struct AppLogger {
    path: PathBuf,
    max_bytes: u64,
    max_files: usize,
    lock: Mutex<()>,
}

impl AppLogger {
    pub fn new(path: PathBuf) -> Self {
        Self::with_rotation(path, MAX_LOG_BYTES, MAX_LOG_FILES)
    }

    fn with_rotation(path: PathBuf, max_bytes: u64, max_files: usize) -> Self {
        Self {
            path,
            max_bytes,
            max_files,
            lock: Mutex::new(()),
        }
    }

    pub fn log(&self, level: &str, event: &str, fields: &[(&str, Value)]) -> Result<(), AppError> {
        let _guard = self.lock.lock().expect("logger lock poisoned");
        if let Some(parent) = self.path.parent() {
            fs::create_dir_all(parent)?;
        }

        let mut entry = Map::new();
        entry.insert("timestamp".to_string(), Value::String(timestamp()));
        entry.insert("level".to_string(), Value::String(level.to_string()));
        entry.insert("event".to_string(), Value::String(event.to_string()));
        for (key, value) in fields {
            entry.insert((*key).to_string(), value.clone());
        }

        let line = format!("{}\n", Value::Object(entry));
        self.rotate_if_needed(line.len() as u64)?;
        let mut file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(&self.path)?;
        file.write_all(line.as_bytes())?;
        Ok(())
    }

    fn rotate_if_needed(&self, incoming_bytes: u64) -> Result<(), AppError> {
        let current_bytes = fs::metadata(&self.path)
            .map(|metadata| metadata.len())
            .unwrap_or_default();
        if self.max_files == 0
            || current_bytes == 0
            || current_bytes + incoming_bytes <= self.max_bytes
        {
            return Ok(());
        }

        for index in (1..=self.max_files).rev() {
            let source = rotated_path(&self.path, index);
            let target = rotated_path(&self.path, index + 1);
            if source.exists() {
                if index == self.max_files {
                    fs::remove_file(source)?;
                } else {
                    fs::rename(source, target)?;
                }
            }
        }

        let first_rotated = rotated_path(&self.path, 1);
        if self.path.exists() {
            fs::rename(&self.path, first_rotated)?;
        }
        Ok(())
    }
}

fn timestamp() -> String {
    let seconds = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|value| value.as_secs())
        .unwrap_or_default();
    seconds.to_string()
}

fn rotated_path(path: &Path, index: usize) -> PathBuf {
    let mut value = path.as_os_str().to_os_string();
    value.push(format!(".{index}"));
    PathBuf::from(value)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rotated_path_appends_index_after_file_name() {
        assert_eq!(
            rotated_path(Path::new("wuwa-tracker.log"), 1),
            PathBuf::from("wuwa-tracker.log.1")
        );
    }

    #[test]
    fn log_rotates_when_size_limit_is_exceeded() {
        let unique = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|value| value.as_nanos())
            .unwrap_or_default();
        let root = std::env::temp_dir().join(format!("wuwa-logger-test-{unique}"));
        fs::create_dir_all(&root).unwrap();
        let path = root.join("app.log");
        let logger = AppLogger::with_rotation(path.clone(), 120, 2);

        logger
            .log(
                "info",
                "first",
                &[("message", Value::String("a".repeat(80)))],
            )
            .unwrap();
        logger
            .log(
                "info",
                "second",
                &[("message", Value::String("b".repeat(80)))],
            )
            .unwrap();

        assert!(path.exists());
        assert!(rotated_path(&path, 1).exists());
        fs::remove_dir_all(root).ok();
    }
}
