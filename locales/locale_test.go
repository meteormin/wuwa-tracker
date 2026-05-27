package locales

import (
	"testing"
)

func TestLoadLocaleData(t *testing.T) {
	// 한국어(ko) 로케일에서 LocaleData를 불러올 수 있는지 검증
	localeData, err := LoadLocaleData("ko")
	if err != nil {
		t.Fatalf("Failed to load 'ko' localeData: %v", err)
	}

	selectList := localeData.SelectList
	if len(selectList) == 0 {
		t.Error("Loaded selectList is empty")
	}

	// 주요 배너 키가 존재하는지 확인
	expectedKeys := []string{
		"characterEvent",
		"weaponEvent",
		"characterPermanent",
		"weaponPermanent",
		"beginner",
		"beginnerSelect",
		"characterNovice",
		"weaponNovice",
	}

	for _, key := range expectedKeys {
		val, ok := selectList[key]
		if !ok {
			t.Errorf("Expected key %q not found in selectList", key)
		}
		if val == "" {
			t.Errorf("Value for key %q is empty", key)
		}
	}
}

func TestLoadUITranslationsWithFallback(t *testing.T) {
	lang, translations, err := LoadUITranslationsWithFallback("en")
	if err != nil {
		t.Fatalf("Failed to load UI translations: %v", err)
	}
	if lang != "en" {
		t.Fatalf("Expected resolved lang 'en', got %q", lang)
	}
	if translations["report.title"] == "" {
		t.Fatal("Expected report.title translation")
	}

	lang, translations, err = LoadUITranslationsWithFallback("missing")
	if err != nil {
		t.Fatalf("Failed to fallback UI translations: %v", err)
	}
	if lang != "ko" {
		t.Fatalf("Expected fallback lang 'ko', got %q", lang)
	}
	if translations["report.title"] == "" {
		t.Fatal("Expected fallback report.title translation")
	}
}
