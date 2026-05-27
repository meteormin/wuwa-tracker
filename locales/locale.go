package locales

import (
	"embed"
	"encoding/json"
)

const fallbackLang = "ko"

//go:embed *.json ui/*.json
var localeFS embed.FS

func loadJSON[T any](path string) (T, error) {
	var target T

	raw, err := localeFS.ReadFile(path)
	if err != nil {
		return target, err
	}

	if err := json.Unmarshal(raw, &target); err != nil {
		return target, err
	}
	return target, nil
}

// LoadSelectList 는 지정된 언어 파일에서 가챠 배너 선택 리스트(selectList)를 불러옵니다.
func LoadSelectList(lang string) (map[string]string, error) {
	type localesMap map[string]any

	loadedMap, err := loadJSON[localesMap](lang + ".json")
	if err != nil {
		return nil, err
	}

	selectList := make(map[string]string)
	if rawSelectList, ok := loadedMap["selectList"].(localesMap); ok {
		for k, v := range rawSelectList {
			if strVal, ok := v.(string); ok {
				selectList[k] = strVal
			}
		}
	}
	return selectList, nil
}

// LoadUITranslations 는 지정된 언어의 UI 번역 리소스를 불러옵니다.
func LoadUITranslations(lang string) (map[string]string, error) {
	type uiLocalesMap map[string]string
	return loadJSON[uiLocalesMap]("ui/" + lang + ".json")
}

// LoadUITranslationsWithFallback 은 UI 번역 리소스를 요청 언어 -> ko 순서로 불러옵니다.
func LoadUITranslationsWithFallback(lang string) (string, map[string]string, error) {
	if lang == "" {
		lang = fallbackLang
	}

	translations, err := LoadUITranslations(lang)
	if err == nil {
		return lang, translations, nil
	}

	if lang == fallbackLang {
		return lang, nil, err
	}

	translations, fallbackErr := LoadUITranslations(fallbackLang)
	if fallbackErr != nil {
		return fallbackLang, nil, fallbackErr
	}
	return fallbackLang, translations, nil
}
