package scan

import (
	"flag"
	"fmt"

	"github.com/meteormin/wuwa-tracker/internal/scanner"
)

// Run 은 scan 서브커맨드를 실행합니다.
// 게임 로그 경로를 전달받아 가챠 기록 URL을 추출하고 표준 출력으로 반환합니다.
func Run(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	pathFlag := fs.String("path", "", "Wuthering Waves Game root path to scan for logs")
	verboseFlag := fs.Bool("v", false, "Enable verbose logging")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *pathFlag == "" {
		return fmt.Errorf("path parameter is required. Use -path")
	}

	if *verboseFlag {
		fmt.Printf("Scanning logs in: %s\n", *pathFlag)
	}

	foundURL, err := scanner.FindURLInDirectory(*pathFlag)
	if err != nil {
		return fmt.Errorf("failed to scan URL: %w", err)
	}

	// 추출된 URL만 표준 출력으로 출력합니다.
	fmt.Println(foundURL)
	return nil
}
