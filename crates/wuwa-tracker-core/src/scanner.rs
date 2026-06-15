use crate::error::AppError;
use std::{
    fs,
    path::{Path, PathBuf},
};

const RESOURCE_URL_MARKER: &str = "aki-gm-resources";

pub fn scan_url(root: &Path, candidates: &[PathBuf]) -> Result<String, AppError> {
    if !root.exists() {
        return Err(AppError::ScanPathNotFound);
    }

    let paths = expand_paths(root, candidates);
    for path in paths {
        if !path.is_file() {
            continue;
        }
        let content = fs::read_to_string(path)?;
        if let Some(url) = find_last_url(&content) {
            return Ok(url);
        }
    }

    Err(AppError::UrlNotFound)
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

fn find_last_url(content: &str) -> Option<String> {
    content
        .split_whitespace()
        .rev()
        .filter_map(clean_url)
        .find(|url| url.contains(RESOURCE_URL_MARKER))
}

fn clean_url(token: &str) -> Option<String> {
    let start = token.find("http")?;
    let value = token[start..]
        .trim_matches(|ch: char| matches!(ch, '"' | '\'' | ',' | ')' | ']' | '}'))
        .replace('\\', "");
    Some(value)
}
