use anyhow::Result;
use axum::{
    extract::{Path, Query, Request, State},
    http::{header, StatusCode},
    middleware::{self, Next},
    response::{IntoResponse, Response},
    routing::{get, post},
    Json, Router,
};
use clap::Args;
use serde::Deserialize;
use std::{str::FromStr, time::Instant};
use tower_http::cors::CorsLayer;
use wuwa_tracker_core::{reporter::ReportFormat, translations, AppError, Service};
use wuwa_tracker_types::{
    ConfigResponse, ErrorResponse, FetchResult, PlayersResponse, ScanResponse, StatsResponse,
};

#[derive(Debug, Clone, Args)]
pub struct ServeArgs {
    #[arg(
        long,
        env = "WUWA_TRACKER_HOST",
        default_value = "127.0.0.1",
        help = "Host address to bind"
    )]
    pub host: String,
    #[arg(
        long,
        env = "WUWA_TRACKER_PORT",
        default_value = "3000",
        help = "TCP port to listen on"
    )]
    pub port: u16,
}

#[derive(Debug, Deserialize)]
struct ScanRequest {
    path: String,
}

#[derive(Debug, Deserialize)]
struct TrackRequest {
    url: String,
}

#[derive(Debug, Deserialize)]
struct ExportQuery {
    format: Option<String>,
    lang: Option<String>,
}

#[derive(Debug, Deserialize)]
struct I18nQuery {
    lang: Option<String>,
}

pub async fn serve(args: ServeArgs, service: Service) -> Result<()> {
    print_startup_info(&args);
    let app = Router::new()
        .route("/api/config", get(get_config))
        .route("/api/players", get(list_players))
        .route("/api/stats/{player_id}", get(get_stats))
        .route("/api/scan", post(scan_url))
        .route("/api/track", post(track_url))
        .route("/api/upload", post(upload_json))
        .route("/api/i18n", get(get_i18n))
        .route("/api/export/{player_id}", get(export_report))
        .route("/api/backup", get(export_backup))
        .layer(middleware::from_fn(access_log))
        .layer(CorsLayer::permissive())
        .with_state(service);

    let listener = tokio::net::TcpListener::bind((args.host.as_str(), args.port)).await?;
    axum::serve(listener, app).await?;
    Ok(())
}

fn print_startup_info(args: &ServeArgs) {
    let api_url = format!("http://{}:{}", args.host, args.port);
    println!("{} {}", env!("CARGO_PKG_NAME"), env!("CARGO_PKG_VERSION"));
    println!("Server: HTTP API");
    println!("Listening: {api_url}");
}

async fn access_log(request: Request, next: Next) -> Response {
    let started = Instant::now();
    let method = request.method().to_string();
    let path = request.uri().path().to_string();
    let user_agent = request
        .headers()
        .get(header::USER_AGENT)
        .and_then(|value| value.to_str().ok())
        .unwrap_or_default()
        .to_string();

    let response = next.run(request).await;
    let status = response.status();
    tracing::info!(
        event = "http_access",
        method = %method,
        path = %path,
        status = status.as_u16(),
        duration_ms = started.elapsed().as_millis() as u64,
        user_agent = %user_agent,
    );
    response
}

async fn get_config(State(service): State<Service>) -> Json<ConfigResponse> {
    Json(ConfigResponse {
        success: true,
        luck_score_thresholds: service.config().luck_score_thresholds.clone(),
    })
}

async fn list_players(State(service): State<Service>) -> Json<PlayersResponse> {
    Json(PlayersResponse {
        success: true,
        players: service.list_players(),
    })
}

async fn get_stats(
    Path(player_id): Path<String>,
    State(service): State<Service>,
) -> Result<Json<StatsResponse>, ApiError> {
    Ok(Json(service.get_stats(player_id)?))
}

async fn scan_url(
    State(service): State<Service>,
    Json(request): Json<ScanRequest>,
) -> Result<Json<ScanResponse>, ApiError> {
    Ok(Json(service.scan(request.path)?))
}

async fn track_url(
    State(service): State<Service>,
    Json(request): Json<TrackRequest>,
) -> Result<Json<StatsResponse>, ApiError> {
    Ok(Json(service.track_url(request.url).await?))
}

async fn upload_json(
    State(service): State<Service>,
    Json(fetch_result): Json<FetchResult>,
) -> Result<Json<StatsResponse>, ApiError> {
    Ok(Json(service.upload(fetch_result)?))
}

async fn get_i18n(
    Query(query): Query<I18nQuery>,
) -> Result<Json<translations::TranslationResponse>, ApiError> {
    Ok(Json(translations::load(
        query.lang.as_deref().unwrap_or("ko"),
    )?))
}

async fn export_report(
    Path(player_id): Path<String>,
    Query(query): Query<ExportQuery>,
    State(service): State<Service>,
) -> Result<Response, ApiError> {
    let format = ReportFormat::from_str(query.format.as_deref().unwrap_or("html"))?;
    let lang = query.lang.as_deref().unwrap_or("ko");
    let content = service.export_report(&player_id, format, lang)?;
    Ok(Response::builder()
        .header(header::CONTENT_TYPE, format.content_type())
        .header(
            header::CONTENT_DISPOSITION,
            format!(
                "attachment; filename=\"report_{}.{}\"",
                player_id,
                format.extension()
            ),
        )
        .body(content.into())
        .expect("valid response"))
}

async fn export_backup(State(service): State<Service>) -> Result<Response, ApiError> {
    let content = service.export_backup()?;
    Ok(Response::builder()
        .header(header::CONTENT_TYPE, "application/json; charset=utf-8")
        .header(
            header::CONTENT_DISPOSITION,
            "attachment; filename=\"wuwa-tracker.backup.json\"",
        )
        .body(content.into())
        .expect("valid response"))
}

struct ApiError(AppError);

impl From<AppError> for ApiError {
    fn from(value: AppError) -> Self {
        Self(value)
    }
}

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        let status = match self.0 {
            AppError::MissingPlayerId
            | AppError::EmptyUploadData
            | AppError::InvalidRequest
            | AppError::InvalidGachaUrl
            | AppError::MissingUrl
            | AppError::UnsupportedReportFormat(_)
            | AppError::TrackerRejected { .. } => StatusCode::BAD_REQUEST,
            AppError::ScanPathNotFound | AppError::LogFileNotFound | AppError::UrlNotFound => {
                StatusCode::NOT_FOUND
            }
            AppError::PlayerNotFound => StatusCode::NOT_FOUND,
            AppError::RemoteTrackingUnsupported => StatusCode::NOT_IMPLEMENTED,
            AppError::NoValidRecords => StatusCode::UNPROCESSABLE_ENTITY,
            AppError::Io(_)
            | AppError::Json(_)
            | AppError::Http(_)
            | AppError::Url(_)
            | AppError::Template(_) => StatusCode::INTERNAL_SERVER_ERROR,
        };
        let body = Json(ErrorResponse {
            success: false,
            error: self.0.to_string(),
            error_key: error_key(&self.0).to_string(),
        });
        (status, body).into_response()
    }
}

fn error_key(error: &AppError) -> &'static str {
    match error {
        AppError::InvalidRequest | AppError::Json(_) => "err.invalid_request_body",
        AppError::MissingUrl => "err.missing_url",
        AppError::MissingPlayerId => "err.missing_player_id",
        AppError::EmptyUploadData => "err.empty_upload_data",
        AppError::ScanPathNotFound => "err.scan_path_not_found",
        AppError::LogFileNotFound => "err.scan_log_file_not_found",
        AppError::UrlNotFound => "err.scan_url_not_found",
        AppError::InvalidGachaUrl | AppError::TrackerRejected { .. } | AppError::Url(_) => {
            "err.invalid_url_format"
        }
        AppError::UnsupportedReportFormat(_) => "err.unsupported_report_format",
        AppError::NoValidRecords | AppError::Template(_) => "err.report_generation_failed",
        AppError::PlayerNotFound => "err.database_query_failed",
        AppError::RemoteTrackingUnsupported | AppError::Http(_) | AppError::Io(_) => {
            "app.network_error"
        }
    }
}
