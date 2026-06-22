use crate::error::AppError;
use std::{collections::BTreeMap, time::Duration};
use url::Url;
use wuwa_tracker_types::{FetchResult, GachaResponse, GachaType, LocaleData, Payload, Record};

const HTTP_TIMEOUT: Duration = Duration::from_secs(5);

#[derive(Clone)]
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
                    records.insert(gacha_type.key.clone(), items);
                }
                Err(error) => {
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

    pub async fn fetch_gacha_locale(&self, lang: &str) -> Result<LocaleData, AppError> {
        let lang = if lang.is_empty() { "ko" } else { lang };
        let endpoint = format!(
            "{}/aki/gacha/locales/{}.json",
            self.resources_url.trim_end_matches('/'),
            lang
        );
        Ok(self
            .client
            .get(endpoint)
            .send()
            .await?
            .error_for_status()?
            .json::<LocaleData>()
            .await?)
    }
}

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

pub fn load_local_gacha_locale(lang: &str) -> Result<LocaleData, AppError> {
    let _ = lang;
    let source = include_str!("../../../locales/ko.json");
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
