mod api;
mod cli;
mod http;

use anyhow::Result;
use clap::{Parser, Subcommand};
use wuwa_tracker_core::{Config, Service};

#[derive(Debug, Parser)]
#[command(name = "wuwa-tracker")]
#[command(about = "Wuwa Tracker v2")]
struct Cli {
    #[command(subcommand)]
    command: Option<Command>,
}

#[derive(Debug, Subcommand)]
enum Command {
    Serve(http::ServeArgs),
    Version,
    Scan(cli::ScanArgs),
    Report(cli::ReportArgs),
    Run(cli::RunArgs),
    Backup(cli::BackupArgs),
    Merge(cli::MergeArgs),
    Db(cli::DbArgs),
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    let service = Service::new(Config::default())?;

    match cli.command {
        Some(Command::Serve(args)) => http::serve(args, service).await,
        Some(Command::Version) => {
            println!(
                "{}",
                option_env!("WUWA_TRACKER_BUILD_TAG").unwrap_or(env!("CARGO_PKG_VERSION"))
            );
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
        ])
        .run(tauri::generate_context!())?;
    Ok(())
}
