use crate::error::AppError;
use serde::Serialize;
use std::collections::BTreeMap;

const KO: &str = include_str!("../../../locales/ui/ko.json");
const EN: &str = include_str!("../../../locales/ui/en.json");

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
/// UI 번역과 실제 적용된 언어 코드를 포함합니다.
pub struct TranslationResponse {
    pub success: bool,
    pub lang: String,
    pub translations: BTreeMap<String, String>,
}

/// 내장된 UI 번역을 로드하며 영어(`en`) 이외의 언어 코드는 한국어로 처리합니다.
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
