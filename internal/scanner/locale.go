package scanner

import (
	"strings"
)

// normalizeLocale 은 원시 로케일 문자열을 Wuthering Waves 가챠 API에 맞는 BCP 47 형식의 소문자 언어 코드로 가공합니다.
func normalizeLocale(locale string) string {
	if locale == "" {
		return ""
	}

	// 소문자로 변환하여 대소문자 구분 없이 처리
	locale = strings.ToLower(locale)
	locale = strings.ReplaceAll(locale, "-", "_")
	parts := strings.Split(locale, "_")
	extracted := parts[0]

	// C, POSIX 등 로케일이 비정상적이거나 너무 짧은 경우 제외
	if extracted == "c" || extracted == "posix" || len(extracted) < 2 {
		return ""
	}

	// 중국어(zh) 로케일 분기 처리: 번체(hant, tw, hk, mo)와 간체(hans, cn, sg 등)
	if extracted == "zh" {
		isTraditional := false
		for _, term := range []string{"hant", "tw", "hk", "mo"} {
			if strings.Contains(locale, term) {
				isTraditional = true
				break
			}
		}
		if isTraditional {
			return "zh-hant"
		}
		return "zh-hans"
	}

	return extracted
}
