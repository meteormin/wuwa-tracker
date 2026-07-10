use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    wuwa_tracker::run_cli().await
}
