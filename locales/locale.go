package locales

import (
	"embed"
	"encoding/json"
)

//go:embed *.json
var localeFS embed.FS

func Load(lang string) (map[string]any, error) {
	raw, err := localeFS.ReadFile(lang + ".json")
	if err != nil {
		return nil, err
	}

	var localesMap map[string]any
	if err := json.Unmarshal(raw, &localesMap); err != nil {
		return nil, err
	}
	return localesMap, nil
}

// LoadSelectList 는 지정된 언어 파일에서 가챠 배너 선택 리스트(selectList)를 불러옵니다.
func LoadSelectList(lang string) (map[string]string, error) {
	localesMap, err := Load(lang)
	if err != nil {
		return nil, err
	}

	selectList := make(map[string]string)
	if rawSelectList, ok := localesMap["selectList"].(map[string]any); ok {
		for k, v := range rawSelectList {
			if strVal, ok := v.(string); ok {
				selectList[k] = strVal
			}
		}
	}
	return selectList, nil
}
