//! Tauri IPC와 HTTP API를 모두 지원하는 Leptos CSR frontend입니다.

mod api;
mod app;
mod i18n;
mod types;

fn main() {
    leptos::mount::mount_to_body(app::App);
}
