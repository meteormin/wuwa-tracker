//go:build !windows

package scanner

import (
	"bufio"
	"os"
	"strings"
)

// GetSystemLocale 은 유닉스/리눅스/맥 OS 환경에서 환경변수(LC_ALL, LC_MESSAGES, LANG) 및 설정 파일을 조회하여
// 정규화된 언어 코드를 반환합니다.
func GetSystemLocale() string {
	// 1. 환경변수 확인 (LC_ALL, LC_MESSAGES, LANG)
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if val := os.Getenv(env); val != "" {
			return normalizeLocale(val)
		}
	}

	// 2. Linux 전용 처리 (환경변수가 비어있고 설정 파일이 존재하는 경우 파일 직접 분석)
	for _, path := range []string{"/etc/locale.conf", "/etc/default/locale"} {
		if file, err := os.Open(path); err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "LANG=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						_ = file.Close()
						return normalizeLocale(strings.Trim(parts[1], `"'`))
					}
				}
			}
			_ = file.Close()
		}
	}

	return ""
}
