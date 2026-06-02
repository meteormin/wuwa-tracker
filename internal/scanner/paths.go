package scanner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FindURLInDirectoryWithPaths 는 전달받은 상대 로그 경로 목록을 사용하여 URL을 추출합니다.
func FindURLInDirectoryWithPaths(gameRoot string, logPaths []string) (string, error) {
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
	for _, relPath := range logPaths {
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
