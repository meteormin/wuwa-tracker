package scanner

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
)

var (
	ErrInvalidURL = errors.New("invalid gacha url")
	// ErrURLNotFound 는 로그 파일 내에서 가챠 URL을 찾을 수 없을 때 반환됩니다.
	ErrURLNotFound = errors.New("gacha url not found in the log")

	ErrScanPathNotFound     = errors.New("scan path not found")
	ErrScanPathAccessDenied = errors.New("scan path access denied")
	ErrLogFileNotFound      = errors.New("log file not found")
)

// FindURL 은 제공된 io.Reader에서 라인 단위로 읽어 정규식과 일치하는 마지막 URL을 반환합니다.
// Client.log, debug.log 둘 다 매칭 가능합니다.
func FindURL(r io.Reader, targetURL ...string) (string, error) {
	scanner := bufio.NewScanner(r)

	// 최대 버퍼 크기 증가 (로그 라인이 매우 긴 경우 대비)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	urlRegex, err := newURLRegex(targetURL...)
	if err != nil {
		return "", err
	}

	var lastFound string
	for scanner.Scan() {
		line := scanner.Text()
		match := urlRegex.FindString(line)
		if match != "" {
			lastFound = match
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if lastFound == "" {
		return "", ErrURLNotFound
	}

	return lastFound, nil
}

// ScanLogFile 은 특정 파일 경로를 열어 가챠 URL을 스캔합니다.
func ScanLogFile(filePath string, targetURL ...string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	return FindURL(f, targetURL...)
}

func newURLRegex(targetURL ...string) (*regexp.Regexp, error) {
	if len(targetURL) > 0 && targetURL[0] != "" {
		return regexp.Compile(regexp.QuoteMeta(targetURL[0]) + `/aki/gacha/index\.html#/record[^"\s]*`)
	}
	return nil, ErrInvalidURL
}
