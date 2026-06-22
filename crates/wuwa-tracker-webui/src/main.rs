mod api;
mod app;
mod i18n;
mod types;

fn main() {
    leptos::mount::mount_to_body(app::App);
}
