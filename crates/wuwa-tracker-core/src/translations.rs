use crate::error::AppError;
use serde::Serialize;
use std::collections::BTreeMap;

const KO: &str = include_str!("../../../locales/ui/ko.json");
const EN: &str = include_str!("../../../locales/ui/en.json");

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TranslationResponse {
    pub success: bool,
    pub lang: String,
    pub translations: BTreeMap<String, String>,
}

pub fn load(lang: &str) -> Result<TranslationResponse, AppError> {
    let resolved_lang = if lang == "en" { "en" } else { "ko" };
    let source = if resolved_lang == "en" { EN } else { KO };
    let translations = serde_json::from_str(source)?;
    Ok(TranslationResponse {
        success: true,
        lang: resolved_lang.to_string(),
        translations,
    })
}
