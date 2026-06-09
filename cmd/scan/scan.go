package scan

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/meteormin/wuwa-tracker/cmd/cli"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
)

func Runner(cfg *config.Config) func(args []string) error {
	return func(args []string) error {
		return run(cfg, args)
	}
}

// Run 은 scan 서브커맨드를 실행합니다.
// 게임 로그 경로를 전달받아 가챠 기록 URL을 추출하고 표준 출력으로 반환합니다.
func run(cfg *config.Config, args []string) error {
	fs := cli.NewFlagSet("scan", "wuwa-tracker scan -path <game-root-or-log-path> [arguments]")
	pathFlag := fs.String("path", "", "Wuthering Waves Game root path to scan for logs")
	clipboardFlag := fs.Bool("clipboard", false, "Copy the URL to the clipboard")
	if handled, err := cli.Parse(fs, args); handled || err != nil {
		return err
	}

	path := strings.TrimSpace(*pathFlag)
	if path == "" {
		return fmt.Errorf("path parameter is required. Use -path")
	}

	fullLogPaths, err := scanner.ExpandLogPaths(path, cfg.ScanLogPaths)
	if err != nil {
		return fmt.Errorf("failed to scan path: %w", err)
	}

	foundURL, err := scanner.FindURLInDirectory(fullLogPaths, cfg.ResourcesURL)
	if err != nil {
		return fmt.Errorf("failed to scan URL: %w", err)
	}

	// 추출된 URL만 표준 출력으로 출력합니다.
	fmt.Println(foundURL)

	if *clipboardFlag {
		if err := copyToClipboard(foundURL); err != nil {
			return err
		}
		fmt.Println("URL copied to clipboard.")
	}

	return nil
}

// copyToClipboard - OS별 기본 네이티브 CLI 도구를 활용한 클립보드 복사
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		// macOS 내장 클립보드 복사 명령어 'pbcopy'
		cmd = exec.Command("pbcopy")

	case "windows": // Windows
		// Windows 내장 클립보드 복사 명령어 'clip'
		cmd = exec.Command("clip")

	case "linux": // Linux
		// Linux는 환경(X11 vs Wayland)에 따라 유동적임. 보통 xclip을 표준으로 사용
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy") // Wayland 환경용
		} else {
			return fmt.Errorf("required utilities (xclip or wl-copy) not found on this Linux system")
		}

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// 명령어의 표준 입력(Stdin)으로 복사할 텍스트 주입
	cmd.Stdin = bytes.NewBufferString(text)

	// 명령어 실행
	return cmd.Run()
}
