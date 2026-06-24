use crate::service::Service;
use anyhow::{Context, Result};
use clap::{Args, Subcommand};
use serde::Serialize;
use std::{
    fs,
    io::Write,
    path::{Path, PathBuf},
    process::{Command, Stdio},
    str::FromStr,
    time::{SystemTime, UNIX_EPOCH},
};
use unicode_width::UnicodeWidthStr;
use wuwa_tracker_core::reporter::ReportFormat;
use wuwa_tracker_types::{FetchResult, FiveStarRecord, Stats, StatsResponse};

const DB_RECORDS_ID_WIDTH: usize = 4;
const DB_RECORDS_KEY_WIDTH: usize = 26;
const DB_RECORDS_NAME_WIDTH: usize = 28;

#[derive(Debug, Clone, Args)]
pub struct ScanArgs {
    #[arg(short, long, help = "Game root directory or log file path to scan")]
    pub path: PathBuf,
    #[arg(long, help = "Copy the scanned URL to the system clipboard")]
    pub clipboard: bool,
}

#[derive(Debug, Clone, Args)]
pub struct ReportArgs {
    #[arg(long, help = "Tracking URL to fetch gacha records from")]
    pub url: Option<String>,
    #[arg(
        short = 'f',
        long = "file",
        help = "FetchResult or legacy JSON file to import before reporting"
    )]
    pub file: Option<PathBuf>,
    #[arg(
        long,
        default_value = "html",
        help = "Report format: html, json, or csv"
    )]
    pub format: String,
    #[arg(
        short = 'o',
        long = "output",
        default_value = "report",
        help = "Output file path or basename"
    )]
    pub output: PathBuf,
    #[arg(long, default_value = "ko", help = "Report language code")]
    pub lang: String,
    #[arg(short = 'v', long, help = "Print progress messages")]
    pub verbose: bool,
}

#[derive(Debug, Clone, Args)]
pub struct RunArgs {
    #[arg(long, help = "Tracking URL to fetch gacha records from")]
    pub url: Option<String>,
    #[arg(short, long, help = "Game root directory or log file path to scan")]
    pub path: Option<PathBuf>,
    #[arg(
        long,
        default_value = "html",
        help = "Report format: html, json, or csv"
    )]
    pub format: String,
    #[arg(
        short = 'o',
        long = "output",
        default_value = "report",
        help = "Output file path or basename"
    )]
    pub output: PathBuf,
    #[arg(long, default_value = "ko", help = "Report language code")]
    pub lang: String,
    #[arg(short = 'v', long, help = "Print progress messages")]
    pub verbose: bool,
}

#[derive(Debug, Clone, Args)]
pub struct BackupArgs {
    #[arg(
        short = 'o',
        long = "output",
        default_value = "wuwa-tracker.backup.json",
        help = "Backup JSON output path"
    )]
    pub output: PathBuf,
}

#[derive(Debug, Clone, Args)]
pub struct MergeArgs {
    #[arg(short = 'f', long = "file", help = "Backup JSON file to merge")]
    pub file: PathBuf,
}

#[derive(Debug, Clone, Args)]
pub struct DbArgs {
    #[command(subcommand)]
    pub command: Option<DbCommand>,
}

#[derive(Debug, Clone, Subcommand)]
pub enum DbCommand {
    #[command(about = "List player IDs stored locally")]
    Players,
    #[command(about = "Show store status or summarized statistics for a player")]
    Stats {
        #[arg(help = "Player ID to inspect; omit to show store status")]
        player_id: Option<String>,
    },
    #[command(about = "Show per-banner record counts for a player")]
    Records {
        #[arg(help = "Player ID to inspect")]
        player_id: String,
    },
}

pub fn scan(args: ScanArgs, service: Service) -> Result<()> {
    let response = service.scan(args.path)?;
    println!("{}", response.url);
    if args.clipboard {
        copy_to_clipboard(&response.url)?;
        println!("URL copied to clipboard.");
    }
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
        service.prepare_locale(&args.lang).await;
        let fetch_result = service.fetch_and_save(&url).await?;
        if args.verbose {
            save_fetch_result_log(&fetch_result)?;
        }
        service.get_stats(fetch_result.payload.player_id)?
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
    service.prepare_locale(&args.lang).await;
    let fetch_result = service.fetch_and_save(&url).await?;
    if args.verbose {
        save_fetch_result_log(&fetch_result)?;
    }
    let stats = service.get_stats(fetch_result.payload.player_id)?;
    write_report(&service, &stats, &args.format, &args.output, &args.lang)?;
    Ok(())
}

pub fn backup(args: BackupArgs, service: Service) -> Result<()> {
    if let Some(parent) = args.output.parent() {
        fs::create_dir_all(parent)?;
    }
    fs::write(&args.output, service.export_backup()?)?;
    println!("Backup created: {}", args.output.display());
    Ok(())
}

pub fn merge(args: MergeArgs, service: Service) -> Result<()> {
    service.merge_backup(&args.file)?;
    println!("Backup merged: {}", args.file.display());
    Ok(())
}

pub fn db(args: DbArgs, service: Service) -> Result<()> {
    match args.command.unwrap_or(DbCommand::Stats { player_id: None }) {
        DbCommand::Stats { player_id } => {
            if let Some(player_id) = player_id {
                let stats = service.get_stats(player_id)?;
                println!(
                    "{}",
                    serde_json::to_string_pretty(&StatsSummary::from(&stats))?
                );
            } else {
                let stats = service.store_stats()?;
                println!("DB Stats");
                println!("Path: {}", stats.path.display());
                println!("Exists: {}", stats.exists);
                println!(
                    "Size: {} ({} bytes)",
                    format_bytes(stats.size_bytes),
                    stats.size_bytes
                );
                println!("Players: {}", stats.players);
                println!("Banners: {}", stats.banners);
                println!("Records: {}", stats.records);
            }
        }
        DbCommand::Players => {
            for player in service.list_players() {
                println!("{player}");
            }
        }
        DbCommand::Records { player_id } => {
            let counts = service.banner_record_counts(player_id)?;
            print_db_records_row("ID", "Key", "Name", "Records");
            for count in counts {
                print_db_records_row(
                    &count.id.to_string(),
                    &count.key,
                    &count.name,
                    &count.records.to_string(),
                );
            }
        }
    }
    Ok(())
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct StatsSummary<'a> {
    success: bool,
    player_id: &'a str,
    stats: Vec<BannerStatsSummary<'a>>,
}

impl<'a> From<&'a StatsResponse> for StatsSummary<'a> {
    fn from(response: &'a StatsResponse) -> Self {
        Self {
            success: response.success,
            player_id: &response.player_id,
            stats: response
                .stats
                .iter()
                .map(BannerStatsSummary::from)
                .collect(),
        }
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct BannerStatsSummary<'a> {
    gacha_type: i32,
    gacha_name: &'a str,
    total_pulls: usize,
    total_astrite: usize,
    current_pity5: i32,
    current_pity4: i32,
    base_rate: f64,
    expected_pulls: i32,
    five_stars: &'a [FiveStarRecord],
    avg_pulls: f64,
    actual_rate: f64,
    luck_score: f64,
    has_five_star: bool,
}

impl<'a> From<&'a Stats> for BannerStatsSummary<'a> {
    fn from(stats: &'a Stats) -> Self {
        Self {
            gacha_type: stats.gacha_type,
            gacha_name: &stats.gacha_name,
            total_pulls: stats.total_pulls,
            total_astrite: stats.total_astrite,
            current_pity5: stats.current_pity5,
            current_pity4: stats.current_pity4,
            base_rate: stats.base_rate,
            expected_pulls: stats.expected_pulls,
            five_stars: &stats.five_stars,
            avg_pulls: stats.avg_pulls,
            actual_rate: stats.actual_rate,
            luck_score: stats.luck_score,
            has_five_star: stats.has_five_star,
        }
    }
}

fn print_db_records_row(id: &str, key: &str, name: &str, records: &str) {
    println!(
        "{} {} {} {}",
        pad_display(id, DB_RECORDS_ID_WIDTH),
        pad_display(key, DB_RECORDS_KEY_WIDTH),
        pad_display(name, DB_RECORDS_NAME_WIDTH),
        records
    );
}

fn pad_display(value: &str, width: usize) -> String {
    let display_width = UnicodeWidthStr::width(value);
    let padding = width.saturating_sub(display_width);
    format!("{value}{}", " ".repeat(padding))
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

fn save_fetch_result_log(fetch_result: &FetchResult) -> Result<()> {
    fs::create_dir_all("logs")?;
    let timestamp = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
    let path = PathBuf::from("logs").join(format!(
        "{}-{}.json",
        fetch_result.payload.player_id, timestamp
    ));
    fs::write(path, serde_json::to_vec_pretty(fetch_result)?)?;
    Ok(())
}

fn format_bytes(bytes: u64) -> String {
    const UNIT: f64 = 1024.0;
    if bytes < 1024 {
        return format!("{bytes} B");
    }

    let mut value = bytes as f64;
    for suffix in ["KB", "MB", "GB", "TB"] {
        value /= UNIT;
        if value < UNIT {
            return trim_float(value, suffix);
        }
    }
    trim_float(value / UNIT, "PB")
}

fn trim_float(value: f64, suffix: &str) -> String {
    let value = format!("{value:.2}");
    format!(
        "{} {suffix}",
        value.trim_end_matches('0').trim_end_matches('.')
    )
}

fn copy_to_clipboard(text: &str) -> Result<()> {
    #[cfg(target_os = "macos")]
    let mut command = Command::new("pbcopy");

    #[cfg(target_os = "windows")]
    let mut command = Command::new("clip");

    #[cfg(target_os = "linux")]
    let mut command = {
        if command_exists("xclip") {
            let mut command = Command::new("xclip");
            command.args(["-selection", "clipboard"]);
            command
        } else if command_exists("wl-copy") {
            Command::new("wl-copy")
        } else {
            anyhow::bail!("required utilities (xclip or wl-copy) not found");
        }
    };

    #[cfg(not(any(target_os = "macos", target_os = "windows", target_os = "linux")))]
    anyhow::bail!("unsupported clipboard platform");

    let mut child = command.stdin(Stdio::piped()).spawn()?;
    if let Some(stdin) = child.stdin.as_mut() {
        stdin.write_all(text.as_bytes())?;
    }
    let status = child.wait()?;
    if !status.success() {
        anyhow::bail!("clipboard command failed");
    }
    Ok(())
}

#[cfg(target_os = "linux")]
fn command_exists(name: &str) -> bool {
    Command::new("sh")
        .arg("-c")
        .arg(format!("command -v {name}"))
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status()
        .map(|status| status.success())
        .unwrap_or(false)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stats_summary_omits_raw_records() {
        let response = StatsResponse {
            success: true,
            player_id: "123456789".to_string(),
            error: None,
            error_key: None,
            stats: vec![Stats {
                gacha_type: 1,
                gacha_name: "Featured Resonator Convene".to_string(),
                total_pulls: 10,
                total_astrite: 1_600,
                current_pity5: 10,
                current_pity4: 0,
                base_rate: 0.008,
                expected_pulls: 80,
                five_stars: Vec::new(),
                records: Vec::new(),
                avg_pulls: 0.0,
                actual_rate: 0.0,
                luck_score: 0.0,
                has_five_star: false,
            }],
        };

        let value = serde_json::to_value(StatsSummary::from(&response)).unwrap();

        assert_eq!(value["playerId"], "123456789");
        assert_eq!(value["stats"][0]["totalPulls"], 10);
        assert!(value["stats"][0].get("records").is_none());
    }
}
