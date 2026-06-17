use anyhow::Result;
use serde_json::{Map, Value};
use std::{fmt, path::Path};
use tracing::{
    field::{Field, Visit},
    Event, Level, Subscriber,
};
use tracing_subscriber::{
    filter::LevelFilter,
    fmt::format::FmtSpan,
    layer::{Context, SubscriberExt},
    registry::LookupSpan,
    util::SubscriberInitExt,
    Layer,
};
use wuwa_tracker_core::logger::AppLogger;

pub fn init(log_path: &Path, console: bool) -> Result<()> {
    let file_layer = FileLogLayer::new(log_path.to_path_buf());

    if console {
        tracing_subscriber::registry()
            .with(file_layer.with_filter(LevelFilter::INFO))
            .with(
                tracing_subscriber::fmt::layer()
                    .compact()
                    .with_target(false)
                    .with_span_events(FmtSpan::NONE)
                    .with_filter(LevelFilter::INFO),
            )
            .try_init()?;
    } else {
        tracing_subscriber::registry()
            .with(file_layer.with_filter(LevelFilter::INFO))
            .try_init()?;
    }

    Ok(())
}

struct FileLogLayer {
    logger: AppLogger,
}

impl FileLogLayer {
    fn new(path: std::path::PathBuf) -> Self {
        Self {
            logger: AppLogger::new(path),
        }
    }
}

impl<S> Layer<S> for FileLogLayer
where
    S: Subscriber + for<'a> LookupSpan<'a>,
{
    fn on_event(&self, event: &Event<'_>, _ctx: Context<'_, S>) {
        let metadata = event.metadata();
        let mut visitor = JsonVisitor::default();
        event.record(&mut visitor);

        let event_name = visitor
            .fields
            .remove("event")
            .and_then(|value| value.as_str().map(ToString::to_string))
            .unwrap_or_else(|| metadata.name().to_string());
        visitor.fields.insert(
            "target".to_string(),
            Value::String(metadata.target().to_string()),
        );

        let _ = self
            .logger
            .log_entry(level_name(metadata.level()), &event_name, visitor.fields);
    }
}

#[derive(Default)]
struct JsonVisitor {
    fields: Map<String, Value>,
}

impl Visit for JsonVisitor {
    fn record_bool(&mut self, field: &Field, value: bool) {
        self.fields
            .insert(field.name().to_string(), Value::Bool(value));
    }

    fn record_i64(&mut self, field: &Field, value: i64) {
        self.fields
            .insert(field.name().to_string(), Value::Number(value.into()));
    }

    fn record_u64(&mut self, field: &Field, value: u64) {
        self.fields
            .insert(field.name().to_string(), Value::Number(value.into()));
    }

    fn record_str(&mut self, field: &Field, value: &str) {
        self.fields
            .insert(field.name().to_string(), Value::String(value.to_string()));
    }

    fn record_error(&mut self, field: &Field, value: &(dyn std::error::Error + 'static)) {
        self.fields
            .insert(field.name().to_string(), Value::String(value.to_string()));
    }

    fn record_debug(&mut self, field: &Field, value: &dyn fmt::Debug) {
        self.fields.insert(
            field.name().to_string(),
            Value::String(format!("{value:?}")),
        );
    }
}

fn level_name(level: &Level) -> &'static str {
    match *level {
        Level::ERROR => "error",
        Level::WARN => "warn",
        Level::INFO => "info",
        Level::DEBUG => "debug",
        Level::TRACE => "trace",
    }
}
