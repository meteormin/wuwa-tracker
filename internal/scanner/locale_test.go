package scanner

import (
	"os"
	"testing"
)

func TestGetSystemLocale(t *testing.T) {
	// 1. 환경변수 우선순위 테스트 (LC_ALL > LC_MESSAGES > LANG) 및 정규화 결과 확인
	origLCAll := os.Getenv("LC_ALL")
	origLCMsg := os.Getenv("LC_MESSAGES")
	origLang := os.Getenv("LANG")

	defer func() {
		_ = os.Setenv("LC_ALL", origLCAll)
		_ = os.Setenv("LC_MESSAGES", origLCMsg)
		_ = os.Setenv("LANG", origLang)
	}()

	_ = os.Setenv("LC_ALL", "fr_FR.UTF-8")
	_ = os.Setenv("LC_MESSAGES", "en_US.UTF-8")
	_ = os.Setenv("LANG", "de_DE.UTF-8")

	locale := GetSystemLocale()
	if locale != "fr" {
		t.Errorf("Expected 'fr', got %q (LC_ALL should take precedence and be normalized)", locale)
	}

	_ = os.Setenv("LC_ALL", "")
	locale = GetSystemLocale()
	if locale != "en" {
		t.Errorf("Expected 'en', got %q (LC_MESSAGES should take precedence when LC_ALL is empty and be normalized)", locale)
	}

	_ = os.Setenv("LC_ALL", "")
	_ = os.Setenv("LC_MESSAGES", "")
	locale = GetSystemLocale()
	if locale != "de" {
		t.Errorf("Expected 'de', got %q (LANG should be used when LC_ALL/LC_MESSAGES are empty and be normalized)", locale)
	}

	// 2. 실제 시스템 로케일 조회 시 패닉이 나지 않고 정상 동작하는지 테스트
	_ = os.Setenv("LC_ALL", "")
	_ = os.Setenv("LC_MESSAGES", "")
	_ = os.Setenv("LANG", "")

	sysLocale := GetSystemLocale()
	t.Logf("System locale returned: %q", sysLocale)
}
