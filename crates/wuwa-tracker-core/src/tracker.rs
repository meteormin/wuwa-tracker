use crate::error::AppError;
use std::{collections::BTreeMap, time::Duration};
use tracing::debug;
use url::Url;
use wuwa_tracker_types::{FetchResult, GachaResponse, GachaType, LocaleData, Payload, Record};

const HTTP_TIMEOUT: Duration = Duration::from_secs(5);

#[derive(Clone)]
/// 게임 리소스와 뽑기 기록 API에 접근하는 HTTP client입니다.
pub struct TrackerClient {
    client: reqwest::Client,
    resources_url: String,
    tracking_url: String,
}

impl TrackerClient {
    pub fn new(resources_url: String, tracking_url: String) -> Self {
        let client = reqwest::Client::builder()
            .timeout(HTTP_TIMEOUT)
            .build()
            .unwrap_or_else(|_| reqwest::Client::new());

        Self {
            client,
            resources_url,
            tracking_url,
        }
    }

    /// URL의 query 또는 fragment query에서 API 요청 payload를 추출합니다.
    ///
    /// # Errors
    ///
    /// URL을 파싱할 수 없거나 필수 식별자가 없으면 [`AppError`]를 반환합니다.
    pub fn parse_payload_from_url(&self, input: &str) -> Result<Payload, AppError> {
        let parsed = Url::parse(input.trim().replace('\\', "").as_str())?;
        let query = if let Some(fragment) = parsed.fragment() {
            fragment
                .split_once('?')
                .map(|(_, query)| query.to_string())
                .unwrap_or_else(|| parsed.query().unwrap_or_default().to_string())
        } else {
            parsed.query().unwrap_or_default().to_string()
        };

        let params: BTreeMap<String, String> = url::form_urlencoded::parse(query.as_bytes())
            .into_owned()
            .collect();
        let payload = Payload {
            player_id: params.get("player_id").cloned().unwrap_or_default(),
            server_id: params.get("svr_id").cloned().unwrap_or_default(),
            language_code: params.get("lang").cloned().unwrap_or_default(),
            record_id: params.get("record_id").cloned().unwrap_or_default(),
            card_pool_id: params.get("gacha_id").cloned().unwrap_or_default(),
            card_pool_type: 0,
        };

        if payload.player_id.is_empty()
            || payload.server_id.is_empty()
            || payload.record_id.is_empty()
        {
            return Err(AppError::InvalidGachaUrl);
        }
        Ok(payload)
    }

    /// 설정된 모든 배너의 기록을 조회합니다.
    ///
    /// 일부 배너 조회가 실패해도 하나 이상 성공하면 성공한 데이터만 반환합니다. 모든 배너가
    /// 실패한 경우 마지막 오류를 반환합니다.
    ///
    /// # Errors
    ///
    /// 모든 배너 조회가 실패하면 해당 API 또는 네트워크 오류를 반환합니다.
    pub async fn fetch_all_records(
        &self,
        mut payload: Payload,
        gacha_types: &[GachaType],
    ) -> Result<FetchResult, AppError> {
        let mut records = BTreeMap::new();
        let mut last_error = None;
        for gacha_type in gacha_types {
            payload.card_pool_type = gacha_type.id;
            match self.fetch_records(&payload).await {
                Ok(items) => {
                    debug!(
                        event = "tracker_pool_fetched",
                        pool_type = gacha_type.id,
                        records = items.len(),
                    );
                    records.insert(gacha_type.key.clone(), items);
                }
                Err(error) => {
                    debug!(
                        event = "tracker_pool_fetch_failed",
                        pool_type = gacha_type.id,
                        error = %error,
                    );
                    last_error = Some(error);
                }
            }
        }

        if records.is_empty() {
            return Err(last_error.unwrap_or(AppError::InvalidGachaUrl));
        }
        Ok(FetchResult { payload, records })
    }

    async fn fetch_records(&self, payload: &Payload) -> Result<Vec<Record>, AppError> {
        let endpoint = format!(
            "{}/gacha/record/query",
            self.tracking_url.trim_end_matches('/')
        );
        let response = self
            .client
            .post(endpoint)
            .json(payload)
            .send()
            .await?
            .error_for_status()?
            .json::<GachaResponse>()
            .await?;

        if response.code != 0 {
            return Err(AppError::TrackerRejected {
                code: response.code,
                message: response.message,
            });
        }
        Ok(response.data)
    }

    /// 지정한 언어의 게임 locale을 원격 리소스에서 가져옵니다.
    ///
    /// 빈 언어 코드는 한국어(`ko`)로 처리합니다.
    ///
    /// # Errors
    ///
    /// HTTP 요청, 상태 검사 또는 JSON 역직렬화에 실패하면 [`AppError`]를 반환합니다.
    pub async fn fetch_gacha_locale(&self, lang: &str) -> Result<LocaleData, AppError> {
        let lang = if lang.is_empty() { "ko" } else { lang };
        let endpoint = format!(
            "{}/aki/gacha/locales/{}.json",
            self.resources_url.trim_end_matches('/'),
            lang
        );
        let locale = self
            .client
            .get(endpoint)
            .send()
            .await?
            .error_for_status()?
            .json::<LocaleData>()
            .await?;
        debug!(event = "tracker_locale_fetched", lang);
        Ok(locale)
    }
}

/// URL의 query 또는 fragment query에서 언어 코드를 찾습니다.
pub fn extract_lang(input: &str) -> Option<String> {
    let parsed = Url::parse(input.trim()).ok()?;
    if let Some(value) = parsed
        .query_pairs()
        .find_map(|(key, value)| (key == "lang").then(|| value.into_owned()))
    {
        return Some(value);
    }
    let fragment = parsed.fragment()?;
    let (_, query) = fragment.split_once('?')?;
    url::form_urlencoded::parse(query.as_bytes())
        .find_map(|(key, value)| (key == "lang").then(|| value.into_owned()))
}

/// 내장된 게임 locale을 로드합니다.
///
/// 영어(`en`) 이외의 언어 코드는 한국어 locale로 처리합니다.
pub fn load_local_gacha_locale(lang: &str) -> Result<LocaleData, AppError> {
    let source = if lang == "en" {
        include_str!("../../../locales/en.json")
    } else {
        include_str!("../../../locales/ko.json")
    };
    Ok(serde_json::from_str(source)?)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tracker_rejected_error_includes_api_code_and_message() {
        let error = AppError::TrackerRejected {
            code: -1,
            message: "request failed".to_string(),
        };

        assert_eq!(
            error.to_string(),
            "tracker API rejected request: code=-1, message=request failed"
        );
    }
}
