use crate::error::AppError;
use regex::Regex;
use std::{
    cmp::Reverse,
    fs,
    path::{Path, PathBuf},
    time::SystemTime,
};

pub fn scan_url(
    root: &Path,
    candidates: &[PathBuf],
    resources_url: &str,
) -> Result<String, AppError> {
    let root = normalize_scan_path(root);
    if !root.exists() {
        return Err(AppError::ScanPathNotFound);
    }

    let url_regex = new_url_regex(resources_url)?;
    let paths = existing_log_files(&expand_paths(&root, candidates))?;
    for path in paths {
        let content = fs::read(path)?;
        if let Some(url) = find_last_url_bytes(&content, &url_regex) {
            return Ok(url);
        }
        let decoded = decode_obfuscated_log(&content);
        if let Some(url) = find_last_url_bytes(&decoded, &url_regex) {
            return Ok(url);
        }
    }

    Err(AppError::UrlNotFound)
}

fn normalize_scan_path(path: &Path) -> PathBuf {
    let value = path.to_string_lossy();
    PathBuf::from(value.trim().trim_matches(|ch| ch == '"' || ch == '\''))
}

fn expand_paths(root: &Path, candidates: &[PathBuf]) -> Vec<PathBuf> {
    if root.is_file() {
        return vec![root.to_path_buf()];
    }
    candidates
        .iter()
        .map(|candidate| root.join(candidate))
        .collect()
}

fn existing_log_files(paths: &[PathBuf]) -> Result<Vec<PathBuf>, AppError> {
    let mut files = Vec::new();
    let mut last_err = None;

    for path in paths {
        match fs::metadata(path) {
            Ok(metadata) if metadata.is_file() => {
                let modified = metadata.modified().unwrap_or(SystemTime::UNIX_EPOCH);
                files.push((path.clone(), modified));
            }
            Ok(_) => {}
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
            Err(error) => last_err = Some(error),
        }
    }

    if files.is_empty() {
        if let Some(error) = last_err {
            return Err(AppError::Io(error));
        }
        return Err(AppError::LogFileNotFound);
    }

    files.sort_by_key(|(_, modified)| Reverse(*modified));
    Ok(files.into_iter().map(|(path, _)| path).collect())
}

fn new_url_regex(resources_url: &str) -> Result<Regex, AppError> {
    let resources_url = resources_url.trim().trim_end_matches('/');
    if resources_url.is_empty() {
        return Err(AppError::InvalidGachaUrl);
    }
    let pattern = format!(
        r#"{}\/aki\/gacha\/index\.html#/record[^"\s]*"#,
        regex::escape(resources_url)
    );
    Regex::new(&pattern).map_err(|_| AppError::InvalidGachaUrl)
}

fn find_last_url_bytes(content: &[u8], url_regex: &Regex) -> Option<String> {
    let content = String::from_utf8_lossy(content);
    find_last_url(&content, url_regex)
}

fn find_last_url(content: &str, url_regex: &Regex) -> Option<String> {
    url_regex
        .find_iter(content)
        .last()
        .map(|match_| match_.as_str().replace('\\', ""))
}

fn decode_obfuscated_log(data: &[u8]) -> Vec<u8> {
    data.iter()
        .map(|b| {
            if (b & 0x0F) % 2 == 1 {
                b ^ 0xA5
            } else {
                b ^ 0xEF
            }
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        fs,
        time::{SystemTime, UNIX_EPOCH},
    };

    const URL: &str = "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?serverId=123&playerId=456&recordId=789";
    const RESOURCES_URL: &str = "https://aki-gm-resources-oversea.aki-game.net";

    #[test]
    fn find_last_url_returns_last_plain_url() {
        let content = format!("{URL}?first=1\nprefix {URL}");
        let url_regex = new_url_regex(RESOURCES_URL).unwrap();
        assert_eq!(find_last_url(&content, &url_regex).as_deref(), Some(URL));
    }

    #[test]
    fn scan_url_decodes_obfuscated_log() {
        let root = temp_scan_dir();
        let log_path = root.join("Client.log");
        let content = format!("LogHttp: Display: HTTP URL: {URL}");
        fs::write(&log_path, encode_client_log_for_test(content.as_bytes())).unwrap();

        let result = scan_url(&log_path, &[], RESOURCES_URL).unwrap();

        assert_eq!(result, URL);
        fs::remove_file(log_path).ok();
        fs::remove_dir(root).ok();
    }

    #[test]
    fn scan_url_normalizes_quoted_file_path() {
        let root = temp_scan_dir();
        let log_path = root.join("Client.log");
        fs::write(&log_path, format!("HTTP URL: {URL}")).unwrap();
        let quoted = PathBuf::from(format!("\"{}\"", log_path.display()));

        let result = scan_url(&quoted, &[], RESOURCES_URL).unwrap();

        assert_eq!(result, URL);
        fs::remove_file(log_path).ok();
        fs::remove_dir(root).ok();
    }

    #[test]
    fn scan_url_reads_newest_candidate_first() {
        let root = temp_scan_dir();
        let old_path = root.join("old.log");
        let new_path = root.join("new.log");
        let old_url = URL.replace("recordId=789", "recordId=old");
        fs::write(&old_path, format!("HTTP URL: {old_url}")).unwrap();
        std::thread::sleep(std::time::Duration::from_millis(10));
        fs::write(&new_path, format!("HTTP URL: {URL}")).unwrap();

        let result = scan_url(
            &root,
            &[PathBuf::from("old.log"), PathBuf::from("new.log")],
            RESOURCES_URL,
        )
        .unwrap();

        assert_eq!(result, URL);
        fs::remove_file(old_path).ok();
        fs::remove_file(new_path).ok();
        fs::remove_dir(root).ok();
    }

    fn encode_client_log_for_test(data: &[u8]) -> Vec<u8> {
        data.iter()
            .map(|b| {
                let candidate = b ^ 0xA5;
                if (candidate & 0x0F) % 2 == 1 {
                    candidate
                } else {
                    b ^ 0xEF
                }
            })
            .collect()
    }

    fn temp_scan_dir() -> PathBuf {
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let path = std::env::temp_dir().join(format!("wuwa-tracker-scan-test-{nanos}"));
        fs::create_dir_all(&path).unwrap();
        path
    }
}
