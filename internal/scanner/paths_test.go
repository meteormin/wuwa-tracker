package scanner

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestFindURLInDirectory_OSCases 는 Windows, macOS, Linux 각 OS별 가상 게임 루트 폴더를 
// 입력값으로 받았을 때, 하위 로그 파일 경로를 올바르게 탐색하고 가챠 URL을 추출하는지 테스트합니다.
func TestFindURLInDirectory_OSCases(t *testing.T) {
	testCases := []struct {
		name         string
		gameRootName string // OS별 게임 루트 폴더 이름 가정 (예: Wuthering Waves Game, com.kurogame.wutheringwaves.global 등)
		logRelPath   string // 게임 루트로부터 실제 로그 파일까지의 상대 경로
	}{
		{
			name:         "Windows Case 1 (Standard Launcher)",
			gameRootName: "Wuthering Waves Game",
			logRelPath:   filepath.Join("Client", "Saved", "Logs", "Client.log"),
		},
		{
			name:         "Windows Case 2 (WebView Debug Log)",
			gameRootName: "Wuthering Waves Game",
			logRelPath:   filepath.Join("Client", "Binaries", "Win64", "ThirdParty", "KrPcSdk_Global", "KRSDKRes", "KRSDKWebView", "debug.log"),
		},
		{
			name:         "macOS Case (App Container)",
			gameRootName: "com.kurogame.wutheringwaves.global",
			logRelPath:   filepath.Join("Data", "Library", "Logs", "Client", "Client.log"),
		},
		{
			name:         "Linux Case (Client Subdir Fallback)",
			gameRootName: "wuthering-waves",
			logRelPath:   filepath.Join("Client", "Client.log"),
		},
		{
			name:         "Linux Case (Direct Fallback)",
			gameRootName: "wuthering-waves-direct",
			logRelPath:   "Client.log",
		},
	}

	mockURL := "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record=test_token_12345"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// OS별 게임 루트 폴더 경로 생성
			gameRoot := filepath.Join(tmpDir, tc.gameRootName)

			// 게임 루트 내에 실제 로그 파일이 위치할 디렉터리 경로 생성
			logFilePath := filepath.Join(gameRoot, tc.logRelPath)
			err := os.MkdirAll(filepath.Dir(logFilePath), 0o755)
			if err != nil {
				t.Fatalf("[%s] 디렉터리 생성 실패: %v", tc.name, err)
			}

			// 유효한 가챠 URL이 기록된 로그 파일 생성
			content := "some log header...\n[Info] WebView URL: " + mockURL + "\nsome log footer..."
			err = os.WriteFile(logFilePath, []byte(content), 0o644)
			if err != nil {
				t.Fatalf("[%s] 모크 로그 파일 생성 실패: %v", tc.name, err)
			}

			// 게임 루트 경로를 입력값으로 전달하여 스캔 실행
			url, err := FindURLInDirectory(gameRoot)
			if err != nil {
				t.Errorf("[%s] FindURLInDirectory(%q) 예상치 못한 에러: %v", tc.name, gameRoot, err)
			}
			if url != mockURL {
				t.Errorf("[%s] FindURLInDirectory(%q) 결과 = %q, 기대값 = %q", tc.name, gameRoot, url, mockURL)
			}
		})
	}
}

// TestFindURLInDirectory_NotFound 는 디렉터리 내에 가챠 URL이 기록된 로그가 없는 경우의 처리를 테스트합니다.
func TestFindURLInDirectory_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// 비어있는 로그 디렉터리 구조 생성
	err := os.MkdirAll(filepath.Join(tmpDir, "Client", "Saved", "Logs"), 0o755)
	if err != nil {
		t.Fatalf("디렉터리 생성 실패: %v", err)
	}

	url, err := FindURLInDirectory(tmpDir)

	if err == nil {
		t.Errorf("FindURLInDirectory() 에러가 발생해야 하나 nil 반환")
	}

	if !errors.Is(err, ErrURLNotFound) {
		t.Errorf("FindURLInDirectory() 에러 = %v, 기대값 = ErrURLNotFound", err)
	}

	if url != "" {
		t.Errorf("FindURLInDirectory() 결과 = %q, 기대값 = 빈 문자열", url)
	}
}

// TestFindURLInDirectory_DirectFile 은 디렉터리가 아닌 로그 파일 경로를 직접 입력받은 경우의 동작을 테스트합니다.
func TestFindURLInDirectory_DirectFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "specific.log")
	mockURL := "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record_direct"
	content := `[2026-05-19 12:00:00] [Debug] Request to: ` + mockURL
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("임시 로그 파일 작성 실패: %v", err)
	}

	url, err := FindURLInDirectory(logFile)
	if err != nil {
		t.Errorf("FindURLInDirectory() 에러 = %v, 기대값 = nil", err)
	}

	if url != mockURL {
		t.Errorf("FindURLInDirectory() 결과 = %q, 기대값 = %q", url, mockURL)
	}
}

// TestScanLogFile_ValidURL 은 로그 파일 스캔 성공 케이스를 테스트합니다.
func TestScanLogFile_ValidURL(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	mockURL := "https://aki-gm-resources.aki-game.com/aki/gacha/index.html#/record_valid"
	content := `
[2026-05-19 12:00:00] [Debug] Game initialized
[2026-05-19 12:01:00] [Info] Network request: ` + mockURL + `
[2026-05-19 12:02:00] [Debug] Another log line
`
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("임시 로그 파일 작성 실패: %v", err)
	}

	url, err := ScanLogFile(logFile)
	if err != nil {
		t.Fatalf("ScanLogFile() 에러 = %v", err)
	}

	if url != mockURL {
		t.Errorf("ScanLogFile() 결과 = %q, 기대값 = %q", url, mockURL)
	}
}

// TestScanLogFile_NoURL 은 가챠 URL이 매칭되지 않는 로그 파일의 스캔 실패 케이스를 테스트합니다.
func TestScanLogFile_NoURL(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "empty.log")
	content := `
[2026-05-19 12:00:00] [Info] Game started
[2026-05-19 12:01:00] [Debug] No network activity
`
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("임시 로그 파일 작성 실패: %v", err)
	}

	_, err := ScanLogFile(logFile)
	if !errors.Is(err, ErrURLNotFound) {
		t.Errorf("ScanLogFile() 에러 = %v, 기대값 = ErrURLNotFound", err)
	}
}
