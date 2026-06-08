package backup

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

// Run 은 backup 서브커맨드를 실행합니다.
// 현재 Badger repository 데이터를 단일 백업 파일로 출력합니다.
func run(cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	outFlag := fs.String("o", "wuwa-tracker.backup", "Output backup file path")
	fs.String("dbpath", cfg.DBPath, "Badger repository storage directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*outFlag) == "" {
		return fmt.Errorf("output file path is required. Use -o")
	}

	core, err := db.OpenBadger(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open badger core: %w", err)
	}
	repository, err := db.NewBadgerRepository(core)
	if err != nil {
		_ = core.Close()
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	defer func() {
		_ = repository.Close()
	}()

	f, err := os.Create(*outFlag)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err := repository.Backup(f); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Printf("Backup successfully created! File: %s\n", *outFlag)
	return nil
}
