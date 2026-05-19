import { writable, derived } from "svelte/store";
import ko from "./locales/ko.json";
import en from "./locales/en.json";

// 지원하는 로케일 타입 정의
export type Locale = "ko" | "en";

// 로컬 스토리지에 저장되었거나 브라우저 설정에 맞는 언어 감지
const savedLocale = (typeof window !== "undefined" ? localStorage.getItem("locale") : null) as Locale;
const initialLocale: Locale = savedLocale === "ko" || savedLocale === "en" 
  ? savedLocale 
  : (typeof navigator !== "undefined" && navigator.language.startsWith("ko") ? "ko" : "en");

// 반응형 로케일 스토어 선언
export const locale = writable<Locale>(initialLocale);

// 로케일이 변경될 때마다 로컬 스토리지에 자동 보존
if (typeof window !== "undefined") {
  locale.subscribe((value) => {
    localStorage.setItem("locale", value);
  });
}

// 다국어 번역 리소스 매핑
export const translations: Record<Locale, Record<string, string>> = {
  ko,
  en,
};

// 반응형 다국어 번역 헬퍼 함수
export const t = derived(locale, ($locale) => {
  return (key: keyof typeof ko, replaceParams?: Record<string, string | number>) => {
    const translationSet = translations[$locale] || translations.ko;
    let text = translationSet[key] || translations.ko[key] || key;
    if (replaceParams) {
      Object.entries(replaceParams).forEach(([k, v]) => {
        text = text.replace(new RegExp(`{${k}}`, "g"), String(v));
      });
    }
    return text;
  };
});
