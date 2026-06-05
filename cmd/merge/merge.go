package merge

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
)

func Runner(cfg *config.Config) func(args []string) error {
	return func(args []string) error {
		return run(cfg, args)
	}
}

// Run 은 merge 서브커맨드를 실행합니다.
// BadgerDB 백업 파일을 현재 DB에 가챠 기록 단위로 병합합니다.
func run(cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	fileFlag := fs.String("f", "", "Path to a BadgerDB backup file")
	fs.String("dbpath", cfg.DBPath, "BadgerDB storage directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*fileFlag) == "" {
		return fmt.Errorf("backup file path is required. Use -f")
	}

	badgerDB, err := db.NewBadgerDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = badgerDB.Close()
	}()

	f, err := os.Open(*fileFlag)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	result, err := badgerDB.MergeFromBackup(f)
	if err != nil {
		return fmt.Errorf("failed to merge backup: %w", err)
	}

	fmt.Printf("Backup merged! players=%d banners=%d records=%d\n", result.Players, result.Banners, result.Records)
	return nil
}
