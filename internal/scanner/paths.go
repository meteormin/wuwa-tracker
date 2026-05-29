package scanner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
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
	if err != nil {
		return "", normalizePathErr(err)
	}
	if !info.IsDir() {
		url, err := ScanLogFile(gameRoot)
		if err != nil {
			return "", normalizePathErr(err)
		}
		return url, nil
	}

	type logFileItem struct {
		path    string
		modTime time.Time
	}

	var files []logFileItem
	var lastPathErr error
	for _, relPath := range LogPaths {
		logFilePath := filepath.Join(gameRoot, relPath)
		info, err := os.Stat(logFilePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				lastPathErr = normalizePathErr(err)
			}
			continue
		}
		if info.IsDir() {
			continue
		}
		files = append(files, logFileItem{
			path:    logFilePath,
			modTime: info.ModTime(),
		})
	}

	if len(files) == 0 {
		if lastPathErr != nil {
			return "", lastPathErr
		}
		return "", ErrLogFileNotFound
	}

	// 파일 수정 시간 기준으로 내림차순 정렬 (가장 최근에 수정된 파일이 앞에 오도록 함)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	var lastErr error
	for _, file := range files {
		url, err := ScanLogFile(file.path)
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

func normalizePathErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrURLNotFound):
		return err
	case errors.Is(err, os.ErrNotExist):
		return ErrScanPathNotFound
	case errors.Is(err, os.ErrPermission):
		return ErrScanPathAccessDenied
	default:
		return fmt.Errorf("scan path error: %w", err)
	}
}
