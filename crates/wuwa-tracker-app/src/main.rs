mod api;
mod cli;
mod http;
mod logging;
mod service;
mod settings;
mod webui_assets;

use anyhow::Result;
use clap::{Parser, Subcommand};
use service::Service;
use std::{env, path::PathBuf};
use wuwa_tracker_core::Config;

const ENV_DB_PATH: &str = "WUWA_TRACKER_DB_PATH";
const ENV_LOG_PATH: &str = "WUWA_TRACKER_LOG_PATH";

#[derive(Debug, Parser)]
#[command(name = "wuwa-tracker")]
#[command(about = "Wuwa Tracker")]
struct Cli {
    #[arg(long = "dbpath", global = true, help = "Local JSON store path")]
    db_path: Option<PathBuf>,
    #[arg(long = "logpath", global = true, help = "Application log file path")]
    log_path: Option<PathBuf>,
    #[command(subcommand)]
    command: Option<Command>,
}

#[derive(Debug, Subcommand)]
enum Command {
    #[command(about = "Run the HTTP API server")]
    Serve(http::ServeArgs),
    #[command(about = "Print the application version")]
    Version,
    #[command(about = "Scan a game directory or log file for a tracking URL")]
    Scan(cli::ScanArgs),
    #[command(about = "Generate a report from a URL or saved JSON file")]
    Report(cli::ReportArgs),
    #[command(about = "Scan or fetch gacha records and generate a report")]
    Run(cli::RunArgs),
    #[command(about = "Periodically scan logs and run when the tracking URL changes")]
    Autorun(cli::AutorunArgs),
    #[command(about = "Manage saved CLI defaults")]
    Config(cli::ConfigArgs),
    #[command(about = "Export the local store to a backup JSON file")]
    Backup(cli::BackupArgs),
    #[command(about = "Merge a backup JSON file into the local store")]
    Merge(cli::MergeArgs),
    #[command(about = "Inspect local store data")]
    Db(cli::DbArgs),
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    if matches!(&cli.command, Some(Command::Version)) {
        println!("{}", env!("CARGO_PKG_VERSION"));
        return Ok(());
    }

    let config = build_config(&cli);
    if let Some(Command::Config(args)) = &cli.command {
        return cli::config(args.clone(), &config);
    }

    let console_level = match &cli.command {
        Some(Command::Serve(_)) => Some("info"),
        Some(_) => Some("error"),
        None => None,
    };
    logging::init(&config.log_path, console_level)?;
    let service = Service::new(config)?;

    match cli.command {
        Some(Command::Serve(args)) => http::serve(args, service).await,
        Some(Command::Version) => unreachable!("version command is handled before service setup"),
        Some(Command::Scan(args)) => cli::scan(args, service),
        Some(Command::Report(args)) => cli::report(args, service).await,
        Some(Command::Run(args)) => cli::run(args, service).await,
        Some(Command::Autorun(args)) => cli::autorun(args, service).await,
        Some(Command::Config(_)) => unreachable!("config command is handled before service setup"),
        Some(Command::Backup(args)) => cli::backup(args, service),
        Some(Command::Merge(args)) => cli::merge(args, service),
        Some(Command::Db(args)) => cli::db(args, service),
        None => run_gui(service),
    }
}

fn build_config(cli: &Cli) -> Config {
    let mut config = Config::default();
    if let Some(db_path) = cli.db_path.clone().or_else(|| get_env(ENV_DB_PATH)) {
        config.db_path = db_path;
    }
    if let Some(log_path) = cli.log_path.clone().or_else(|| get_env(ENV_LOG_PATH)) {
        config.log_path = log_path;
    }
    if let Some(interval_secs) = get_env_u64(settings::ENV_AUTORUN_INTERVAL_SECS) {
        config.autorun_interval_secs = interval_secs;
    }
    if let Some(interval_secs) = settings::load(&config.settings_path)
        .ok()
        .and_then(|settings| settings.interval_secs)
    {
        config.autorun_interval_secs = interval_secs;
    }
    config
}

// get_env를 Option의 메서드 체이닝으로 단순화
fn get_env(key: &str) -> Option<PathBuf> {
    env::var(key)
        .ok()
        .map(|v| v.trim().to_string())
        .filter(|v| !v.is_empty())
        .map(PathBuf::from)
}

fn get_env_u64(key: &str) -> Option<u64> {
    env::var(key).ok()?.trim().parse().ok()
}

fn run_gui(service: Service) -> Result<()> {
    tauri::Builder::default()
        .manage(service)
        .invoke_handler(tauri::generate_handler![
            api::get_config,
            api::list_players,
            api::get_stats,
            api::scan_url,
            api::track_url,
            api::upload_json,
            api::get_i18n,
            api::export_report,
            api::export_backup,
        ])
        .run(tauri::generate_context!())?;
    Ok(())
}
