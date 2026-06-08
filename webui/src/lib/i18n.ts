import { writable, derived } from "svelte/store";
import { apiHost } from "./api/config";

export type Locale = "ko" | "en";

const fallbackLocale: Locale = "ko";

const savedLocale = (
  typeof window !== "undefined" ? localStorage.getItem("locale") : null
) as Locale;

const initialLocale: Locale =
  savedLocale === "ko" || savedLocale === "en"
    ? savedLocale
    : typeof navigator !== "undefined" && navigator.language.startsWith("ko")
      ? "ko"
      : "en";

export const locale = writable<Locale>(initialLocale);
export const translations = writable<Record<string, string>>({});

if (typeof window !== "undefined") {
  locale.subscribe((value) => {
    localStorage.setItem("locale", value);
  });
}

export async function loadTranslations(targetLocale: Locale) {
  const res = await fetch(`${apiHost}/api/i18n?lang=${targetLocale}`);
  if (!res.ok) {
    throw new Error(`failed to load translations: ${res.status}`);
  }

  const data = await res.json();
  if (!data.success || !data.translations) {
    throw new Error("invalid translations response");
  }

  translations.set(data.translations);
  if (data.lang === "ko" || data.lang === "en") {
    locale.set(data.lang);
  }
}

export async function setLocale(targetLocale: Locale) {
  await loadTranslations(targetLocale);
}

export async function initI18n() {
  try {
    await loadTranslations(initialLocale);
  } catch (e) {
    if (initialLocale !== fallbackLocale) {
      await loadTranslations(fallbackLocale);
      return;
    }
    throw e;
  }
}

export const t = derived(translations, ($translations) => {
  return (key: string, replaceParams?: Record<string, string | number>) => {
    let text = $translations[key] || key;
    if (replaceParams) {
      Object.entries(replaceParams).forEach(([k, v]) => {
        text = text.replace(new RegExp(`{${k}}`, "g"), String(v));
      });
    }
    return text;
  };
});
