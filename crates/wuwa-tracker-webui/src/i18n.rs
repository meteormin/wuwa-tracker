use leptos::prelude::*;
use std::collections::BTreeMap;

const KO: &str = include_str!("../../../locales/ui/ko.json");
const EN: &str = include_str!("../../../locales/ui/en.json");

#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
pub enum Locale {
    Ko,
    En,
}

impl Locale {
    pub fn code(self) -> &'static str {
        match self {
            Self::Ko => "ko",
            Self::En => "en",
        }
    }
}

#[derive(Clone, Copy)]
pub struct I18n {
    locale: RwSignal<Locale>,
    translations: RwSignal<BTreeMap<String, String>>,
}

impl I18n {
    pub fn new() -> Self {
        let locale = initial_locale();
        Self {
            locale: RwSignal::new(locale),
            translations: RwSignal::new(translations(locale)),
        }
    }

    pub fn locale(self) -> Locale {
        self.locale.get()
    }

    pub fn set_locale(self, locale: Locale) {
        self.locale.set(locale);
        self.translations.set(translations(locale));
        if let Some(storage) = window().local_storage().ok().flatten() {
            let _ = storage.set_item("locale", locale.code());
        }
    }

    pub fn text(self, key: &str) -> String {
        self.translations
            .with(|items| items.get(key).cloned())
            .unwrap_or_else(|| key.to_string())
    }

    pub fn format(self, key: &str, params: &[(&str, String)]) -> String {
        let mut value = self.text(key);
        for (name, replacement) in params {
            value = value.replace(&format!("{{{name}}}"), replacement);
        }
        value
    }
}

fn initial_locale() -> Locale {
    if let Some(storage) = window().local_storage().ok().flatten() {
        match storage.get_item("locale").ok().flatten().as_deref() {
            Some("ko") => return Locale::Ko,
            Some("en") => return Locale::En,
            _ => {}
        }
    }

    let language = window().navigator().language().unwrap_or_default();
    if language.starts_with("ko") {
        Locale::Ko
    } else {
        Locale::En
    }
}

fn translations(locale: Locale) -> BTreeMap<String, String> {
    serde_json::from_str(match locale {
        Locale::Ko => KO,
        Locale::En => EN,
    })
    .expect("embedded translations must be valid")
}
