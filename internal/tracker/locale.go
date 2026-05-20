package tracker

import (
	"log"

	"github.com/meteormin/wuwa-tracker/locales"
)

const fallbackLang = "ko"

// LoadGachaLocaleWithFallback 은 Client를 통해 원격 API로부터 가챠 로케일을 조회하고,
// 실패할 경우 로컬 내장 로케일 파일을 단계적(요청 언어 -> ko)으로 fallback하여 불러옵니다.
// Client 객체는 순수하게 외부 연동 책임만 수행하도록 유지하기 위해, 폴백 제어 로직을 별도 유틸리티 함수로 정의합니다.
func LoadGachaLocaleWithFallback(client *Client, urlstr string, lang string) map[string]string {
	if lang == "" {
		lang = "ko"
	}

	// 1. 원격에서 요청한 언어 로케일 조회 시도
	localeData, err := client.FetchGachaLocale(urlstr, lang)
	if err == nil {
		return localeData.SelectList
	}

	log.Printf("Warning: failed to fetch remote localized banner names for %q: %v. Trying local fallback.\n", lang, err)

	// 2. 로컬 내장에서 요청한 언어 로케일 조회 시도
	localSelectList, localErr := locales.LoadSelectList(lang)
	if localErr == nil {
		return localSelectList
	}
	log.Printf("Warning: failed to load local fallback for %q: %v.\n", lang, localErr)

	// 요청한 언어가 한국어("ko")가 아닌 경우, 한국어를 예비(fallback) 언어로 시도
	if lang != fallbackLang {
		log.Printf("Trying fallback to 'ko'.\n")

		// 3. 원격에서 한국어("ko") 로케일 조회 시도
		localeData, err = client.FetchGachaLocale(urlstr, fallbackLang)
		if err == nil {
			return localeData.SelectList
		}
		log.Printf("Warning: failed to fetch remote 'ko' banner names: %v. Trying local 'ko'.\n", err)

		// 4. 로컬 내장에서 한국어("ko") 로케일 조회 시도
		localSelectList, localErr = locales.LoadSelectList("ko")
		if localErr == nil {
			return localSelectList
		}
		log.Printf("Warning: failed to load local 'ko' fallback: %v\n", localErr)
	}

	return make(map[string]string)
}
