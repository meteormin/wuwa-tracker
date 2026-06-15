use anyhow::{Context, Result};
use clap::{Args, Subcommand};
use std::{
    fs,
    path::{Path, PathBuf},
    str::FromStr,
};
use wuwa_tracker_core::{
    types::{ReportFormat, StatsResponse},
    Service,
};

#[derive(Debug, Clone, Args)]
pub struct ScanArgs {
    #[arg(short, long)]
    pub path: PathBuf,
}

#[derive(Debug, Clone, Args)]
pub struct ReportArgs {
    #[arg(long)]
    pub url: Option<String>,
    #[arg(short = 'f', long = "file")]
    pub file: Option<PathBuf>,
    #[arg(long, default_value = "html")]
    pub format: String,
    #[arg(short = 'o', long = "output", default_value = "report")]
    pub output: PathBuf,
    #[arg(long, default_value = "ko")]
    pub lang: String,
    #[arg(short = 'v', long)]
    pub verbose: bool,
}

#[derive(Debug, Clone, Args)]
pub struct RunArgs {
    #[arg(long)]
    pub url: Option<String>,
    #[arg(short, long)]
    pub path: Option<PathBuf>,
    #[arg(long, default_value = "html")]
    pub format: String,
    #[arg(short = 'o', long = "output", default_value = "report")]
    pub output: PathBuf,
    #[arg(long, default_value = "ko")]
    pub lang: String,
    #[arg(short = 'v', long)]
    pub verbose: bool,
}

#[derive(Debug, Clone, Args)]
pub struct BackupArgs {
    #[arg(
        short = 'o',
        long = "output",
        default_value = "wuwa-tracker.backup.json"
    )]
    pub output: PathBuf,
}

#[derive(Debug, Clone, Args)]
pub struct MergeArgs {
    #[arg(short = 'f', long = "file")]
    pub file: PathBuf,
}

#[derive(Debug, Clone, Args)]
pub struct DbArgs {
    #[command(subcommand)]
    pub command: Option<DbCommand>,
}

#[derive(Debug, Clone, Subcommand)]
pub enum DbCommand {
    Players,
}

pub fn scan(args: ScanArgs, service: Service) -> Result<()> {
    let response = service.scan(args.path)?;
    println!("{}", response.url);
    Ok(())
}

pub async fn report(args: ReportArgs, service: Service) -> Result<()> {
    if args.url.is_some() == args.file.is_some() {
        anyhow::bail!("provide exactly one of --url or --file");
    }

    let stats = if let Some(file) = args.file {
        let fetch_result = service.load_fetch_result_file(&file)?;
        let player_id = fetch_result.payload.player_id.clone();
        service.upload(fetch_result)?;
        service.get_stats(player_id)?
    } else {
        let url = args.url.expect("checked above");
        if args.verbose {
            println!("Fetching gacha data. Please wait...");
        }
        service.track_url(url).await?
    };

    write_report(&service, &stats, &args.format, &args.output, &args.lang)?;
    Ok(())
}

pub async fn run(args: RunArgs, service: Service) -> Result<()> {
    let url = match args.url {
        Some(url) => url,
        None => {
            let path = args
                .path
                .as_ref()
                .context("provide --url or --path for run")?;
            service.scan(path)?.url
        }
    };
    let stats = service.track_url(url).await?;
    write_report(&service, &stats, &args.format, &args.output, &args.lang)?;
    Ok(())
}

pub fn backup(args: BackupArgs, service: Service) -> Result<()> {
    service.backup(&args.output)?;
    println!("Backup created: {}", args.output.display());
    Ok(())
}

pub fn merge(args: MergeArgs, service: Service) -> Result<()> {
    service.merge_backup(&args.file)?;
    println!("Backup merged: {}", args.file.display());
    Ok(())
}

pub fn db(args: DbArgs, service: Service) -> Result<()> {
    match args.command.unwrap_or(DbCommand::Players) {
        DbCommand::Players => {
            for player in service.list_players() {
                println!("{player}");
            }
        }
    }
    Ok(())
}

fn write_report(
    service: &Service,
    stats: &StatsResponse,
    format: &str,
    output: &Path,
    lang: &str,
) -> Result<()> {
    let format = ReportFormat::from_str(format)?;
    let mut output = output.to_path_buf();
    if output.extension().and_then(|value| value.to_str()) != Some(format.extension()) {
        output.set_extension(format.extension());
    }
    if let Some(parent) = output.parent() {
        if !parent.as_os_str().is_empty() {
            fs::create_dir_all(parent)?;
        }
    }
    let content = service.export_report(&stats.player_id, format, lang)?;
    fs::write(&output, content)?;
    println!("Report successfully generated! File: {}", output.display());
    Ok(())
}
