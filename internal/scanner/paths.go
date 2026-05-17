package scanner

import (
	"errors"
	"os"
	"path/filepath"
)

// LogPaths 는 게임 루트 폴더를 기준으로 로그 파일이 존재하는 상대 경로들입니다.
var LogPaths = []string{
	// Windows standard
	filepath.Join("Client", "Saved", "Logs", "Client.log"),
	filepath.Join("Client", "Binaries", "Win64", "ThirdParty", "KrPcSdk_Global", "KRSDKRes", "KRSDKWebView", "debug.log"),
	// Mac App Container (com.kurogame.wutheringwaves.global)
	filepath.Join("Data", "Library", "Logs", "Client", "Client.log"),
	// Direct sub-directory fallback (e.g. if user path is already Logs dir)
	filepath.Join("Client", "Client.log"),
	"Client.log",
}

// FindURLInDirectory 는 주어진 경로 내에서 로그 파일들을 탐색하고 URL을 추출합니다.
// 경로가 이미 파일인 경우 해당 파일을 직접 스캔합니다.
func FindURLInDirectory(gameRoot string) (string, error) {
	// 입력받은 경로가 디렉터리가 아니라면(즉, 파일이라면) 직접 스캔
	info, err := os.Stat(gameRoot)
	if err == nil && !info.IsDir() {
		return ScanLogFile(gameRoot)
	}

	var lastErr error
	for _, relPath := range LogPaths {
		logFilePath := filepath.Join(gameRoot, relPath)
		if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
			continue
		}

		url, err := ScanLogFile(logFilePath)
		if err == nil {
			return url, nil
		}
		if !errors.Is(err, ErrURLNotFound) {
			lastErr = err
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	return "", ErrURLNotFound
}
