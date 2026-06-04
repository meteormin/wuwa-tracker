package scanner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExpandLogPaths 는 입력 경로가 파일이면 그대로, 디렉터리면 상대 로그 경로들을 결합해 전체 경로 목록을 만듭니다.
func ExpandLogPaths(path string, relativeLogPaths []string) ([]string, error) {
	path = normalizeScanPath(path)
	if path == "" {
		return nil, ErrScanPathNotFound
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, normalizePathErr(err)
	}
	if !info.IsDir() {
		return []string{path}, nil
	}

	logPaths := make([]string, 0, len(relativeLogPaths))
	for _, relPath := range relativeLogPaths {
		logPaths = append(logPaths, filepath.Join(path, relPath))
	}
	return logPaths, nil
}

func normalizeScanPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"'`)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

// FindURLInDirectory 는 전달받은 전체 로그 파일 경로 목록에서 URL을 추출합니다.
func FindURLInDirectory(logPaths []string, targetURL string) (string, error) {
	type logFileItem struct {
		path    string
		modTime time.Time
	}

	var files []logFileItem
	var lastPathErr error
	for _, logPath := range logPaths {
		info, err := os.Stat(logPath)
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
			path:    logPath,
			modTime: info.ModTime(),
		})
	}

	if len(files) == 0 {
		if lastPathErr != nil {
			return "", lastPathErr
		}
		return "", ErrLogFileNotFound
	}

	// 파일 수정 시간 기준으로 내림차순 정렬합니다.
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	var lastErr error
	for _, file := range files {
		url, err := ScanLogFile(file.path, targetURL)
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
