import { writable, derived } from "svelte/store";
import { invoke } from "@tauri-apps/api/core";
import { apiHost, isTauriRuntime } from "./api/config";
import koTranslations from "../../../locales/ui/ko.json";
import enTranslations from "../../../locales/ui/en.json";

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
export const translations = writable<Record<string, string>>(
  initialLocale === "ko" ? koTranslations : enTranslations,
);

if (typeof window !== "undefined") {
  locale.subscribe((value) => {
    localStorage.setItem("locale", value);
  });
}

export async function loadTranslations(targetLocale: Locale) {
  if (isTauriRuntime()) {
    const data = await invoke<{
      success: boolean;
      lang: Locale;
      translations: Record<string, string>;
    }>("get_i18n", { lang: targetLocale });

    translations.set(data.translations);
    if (data.lang === "ko" || data.lang === "en") {
      locale.set(data.lang);
    }
    return;
  }

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
      try {
        await loadTranslations(fallbackLocale);
        return;
      } catch {
        translations.set(koTranslations);
        locale.set(fallbackLocale);
        return;
      }
    }
    translations.set(koTranslations);
    locale.set(fallbackLocale);
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
