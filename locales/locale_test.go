package locales

import (
	"testing"
)

func TestLoadSelectList(t *testing.T) {
	// 한국어(ko) 로케일에서 selectList를 불러올 수 있는지 검증
	selectList, err := LoadSelectList("ko")
	if err != nil {
		t.Fatalf("Failed to load 'ko' selectList: %v", err)
	}

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
