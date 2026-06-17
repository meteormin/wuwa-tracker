mod api;
mod cli;
mod http;

use anyhow::Result;
use clap::{Parser, Subcommand};
use std::path::PathBuf;
use wuwa_tracker_core::{Config, Service};

#[derive(Debug, Parser)]
#[command(name = "wuwa-tracker")]
#[command(about = "Wuwa Tracker")]
struct Cli {
    #[arg(
        long = "dbpath",
        env = "WUWA_TRACKER_DB_PATH",
        global = true,
        help = "Local JSON store path"
    )]
    db_path: Option<PathBuf>,
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
    let mut config = Config::default();
    if let Some(db_path) = cli.db_path {
        config.db_path = db_path;
    }
    let service = Service::new(config)?;

    match cli.command {
        Some(Command::Serve(args)) => http::serve(args, service).await,
        Some(Command::Version) => {
            println!("{}", env!("CARGO_PKG_VERSION"));
            Ok(())
        }
        Some(Command::Scan(args)) => cli::scan(args, service),
        Some(Command::Report(args)) => cli::report(args, service).await,
        Some(Command::Run(args)) => cli::run(args, service).await,
        Some(Command::Backup(args)) => cli::backup(args, service),
        Some(Command::Merge(args)) => cli::merge(args, service),
        Some(Command::Db(args)) => cli::db(args, service),
        None => run_gui(service),
    }
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
